package handlers

import (
	"net/http"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	"strings" // Adicionado para strings.ToLower e strings.Join
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"encoding/json"
	"fmt"
	"log"
	// "mime/multipart" // Removido - não usado diretamente aqui, o Gin lida com isso
	"path/filepath"
	"phoenixgrc/backend/internal/filestorage"
	"io" // Necessário para file.Seek e http.DetectContentType
	// "net/http/httputil" // Removido - não usado

	"gorm.io/gorm"
	"gorm.io/gorm/clause" // Para Upsert
)

// --- Framework and Control Handlers ---

// ListFrameworksHandler lists all available audit frameworks.
func ListFrameworksHandler(c *gin.Context) {
	db := database.GetDB()
	var frameworks []models.AuditFramework
	if err := db.Find(&frameworks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list audit frameworks: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, frameworks)
}

// GetControlFamiliesForFrameworkHandler lists all unique control families for a specific framework.
func GetControlFamiliesForFrameworkHandler(c *gin.Context) {
	frameworkIDStr := c.Param("frameworkId")
	frameworkID, err := uuid.Parse(frameworkIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid framework ID format"})
		return
	}

	db := database.GetDB()
	var families []string
	// Usar Distinct para pegar apenas famílias únicas e não nulas/vazias
	if err := db.Model(&models.AuditControl{}).
		Where("framework_id = ? AND family <> ''", frameworkID).
		Distinct("family").
		Order("family asc").
		Pluck("family", &families).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list control families for framework: " + err.Error()})
		return
	}

	if families == nil {
		families = []string{} // Garantir array vazio em vez de nulo se não houver famílias
	}

	c.JSON(http.StatusOK, gin.H{"families": families})
}

// GetFrameworkControlsHandler lists all controls for a specific framework.
func GetFrameworkControlsHandler(c *gin.Context) {
	frameworkIDStr := c.Param("frameworkId")
	frameworkID, err := uuid.Parse(frameworkIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid framework ID format"})
		return
	}

	db := database.GetDB()
	var controls []models.AuditControl
	if err := db.Where("framework_id = ?", frameworkID).Order("control_id asc").Find(&controls).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list controls for framework: " + err.Error()})
		return
	}

	if len(controls) == 0 {
		// Check if framework itself exists to give a better error
		var framework models.AuditFramework
		if errFramework := db.First(&framework, "id = ?", frameworkID).Error; errFramework == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Framework not found"})
			return
		}
	}

	// Para cada controle, buscar a avaliação da organização do usuário (se existir)
	orgID, orgOk := c.Get("organizationID")
	if !orgOk {
		c.JSON(http.StatusForbidden, gin.H{"error": "Organization ID not found in token for fetching assessments"})
		return
	}
	organizationID := orgID.(uuid.UUID)

	type AuditControlWithAssessmentResponse struct {
		models.AuditControl
		Assessment *models.AuditAssessment `json:"assessment,omitempty"`
	}

	responseControls := make([]AuditControlWithAssessmentResponse, 0, len(controls))
	controlIDs := make([]uuid.UUID, len(controls))
	for i, ctrl := range controls {
		controlIDs[i] = ctrl.ID
	}

	var assessments []models.AuditAssessment
	if len(controlIDs) > 0 {
		db.Where("organization_id = ? AND audit_control_id IN (?)", organizationID, controlIDs).Find(&assessments)
	}

	assessmentMap := make(map[uuid.UUID]models.AuditAssessment)
	for _, assess := range assessments {
		assessmentMap[assess.AuditControlID] = assess
	}

	for _, ctrl := range controls {
		respCtrl := AuditControlWithAssessmentResponse{
			AuditControl: ctrl,
		}
		if assessment, found := assessmentMap[ctrl.ID]; found {
			respCtrl.Assessment = &assessment
		}
		responseControls = append(responseControls, respCtrl)
	}

	c.JSON(http.StatusOK, responseControls)
}

// --- Assessment Handlers ---

// AssessmentPayload defines the structure for creating or updating an assessment.
type AssessmentPayload struct {
	AuditControlID string                    `json:"audit_control_id" binding:"required"` // UUID of the AuditControl
	Status         models.AuditControlStatus `json:"status" binding:"required,oneof=conforme nao_conforme parcialmente_conforme nao_aplicavel"` // Adicionado nao_aplicavel
	EvidenceURL    string                    `json:"evidence_url" binding:"omitempty,url"`
	Score          *int                      `json:"score" binding:"omitempty,min=0,max=100"`      // Pointer for optional score
	AssessmentDate string                    `json:"assessment_date" binding:"omitempty,datetime=2006-01-02"` // YYYY-MM-DD
	Comments       *string                   `json:"comments,omitempty"`                               // Comentários da avaliação principal

	// Campos C2M2
	C2M2MaturityLevel *int    `json:"c2m2_maturity_level,omitempty" binding:"omitempty,min=0,max=3"` // 0-3
	C2M2AssessmentDate *string `json:"c2m2_assessment_date,omitempty" binding:"omitempty,datetime=2006-01-02"` // YYYY-MM-DD
	C2M2Comments      *string `json:"c2m2_comments,omitempty"`
}

const (
	maxEvidenceFileSize = 10 << 20 // 10 MB
)

var allowedMimeTypes = map[string]bool{
	"image/jpeg":                                                       true,
	"image/png":                                                        true,
	"application/pdf":                                                  true,
	"application/msword":                                               true, // .doc
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true, // .docx
	"application/vnd.ms-excel":                                         true, // .xls
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":   true, // .xlsx
	"text/plain":                                                       true, // .txt
	// Adicionar outros tipos conforme necessário
}


// CreateOrUpdateAssessmentHandler creates a new assessment or updates an existing one.
// An assessment is unique per (OrganizationID, AuditControlID).
func CreateOrUpdateAssessmentHandler(c *gin.Context) {
	// Multipart form processing
	// Max 10 MB files
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form: " + err.Error()})
		return
	}

	// Get JSON payload from "data" form field
	payloadString := c.Request.FormValue("data")
	if payloadString == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'data' field in multipart form"})
		return
	}

	var payload AssessmentPayload
	if err := json.Unmarshal([]byte(payloadString), &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON in 'data' field: " + err.Error()})
		return
	}

	// Re-validate the unmarshalled payload using Gin's validator (optional, but good practice)
	// This requires payload to be bound again, or use a custom validator.
	// For now, we assume basic JSON unmarshalling is enough if `ShouldBindJSON` was to be used.
	// A better way would be to bind the JSON part specifically if Gin supports it for multipart.
	// Or, manually trigger validation:
	// validate := validator.New()
	// if err := validate.Struct(payload); err != nil { ... }


	orgID, orgExists := c.Get("organizationID")
	if !orgExists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Organization ID not found in token"})
		return
	}
	organizationID := orgID.(uuid.UUID)

	auditControlUUID, err := uuid.Parse(payload.AuditControlID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid audit_control_id format"})
		return
	}

	var parsedAssessmentDate time.Time
	if payload.AssessmentDate != "" {
		parsedAssessmentDate, err = time.Parse("2006-01-02", payload.AssessmentDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assessment_date format, use YYYY-MM-DD: " + err.Error()})
			return
		}
	} else {
		parsedAssessmentDate = time.Now() // Default to now if not provided
	}
	assessmentModel.AssessmentDate = &parsedAssessmentDate // Usar ponteiro

	if payload.Comments != nil {
		assessmentModel.Comments = payload.Comments
	}

	// Processar campos C2M2
	if payload.C2M2MaturityLevel != nil {
		assessmentModel.C2M2MaturityLevel = payload.C2M2MaturityLevel
	}
	if payload.C2M2Comments != nil {
		assessmentModel.C2M2Comments = payload.C2M2Comments
	}
	if payload.C2M2AssessmentDate != nil && *payload.C2M2AssessmentDate != "" {
		parsedC2M2Date, errDate := time.Parse("2006-01-02", *payload.C2M2AssessmentDate)
		if errDate != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid c2m2_assessment_date format, use YYYY-MM-DD: " + errDate.Error()})
			return
		}
		assessmentModel.C2M2AssessmentDate = &parsedC2M2Date
	}


	// assessmentEvidenceIdentifier armazenará o objectName se um arquivo for carregado,
	// ou a URL externa fornecida no payload se nenhum arquivo for carregado.
	assessmentEvidenceIdentifier := payload.EvidenceURL

	assessmentModel := models.AuditAssessment{
		OrganizationID: organizationID,
		AuditControlID: auditControlUUID,
		Status:         payload.Status,
		// EvidenceURL será setado abaixo
		// AssessmentDate, Score, Comments, e campos C2M2 serão setados abaixo
	}

	if payload.AssessmentDate != "" {
		parsedAssessmentDate, err = time.Parse("2006-01-02", payload.AssessmentDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assessment_date format, use YYYY-MM-DD: " + err.Error()})
			return
		}
		assessmentModel.AssessmentDate = &parsedAssessmentDate
	} else {
		now := time.Now()
		assessmentModel.AssessmentDate = &now // Default to now if not provided
	}

	if payload.Score != nil {
		assessmentModel.Score = payload.Score
	} else {
		// Default score logic
		var defaultScore int
		switch payload.Status {
		case models.ControlStatusConformant:
			defaultScore = 100
		case models.ControlStatusPartiallyConformant:
			defaultScore = 50
		case models.ControlStatusNonConformant, models.ControlStatusNotApplicable: // Adicionado NotApplicable
			defaultScore = 0
		}
		assessmentModel.Score = &defaultScore
	}

	if payload.Comments != nil {
		assessmentModel.Comments = payload.Comments
	}

	// Processar campos C2M2
	if payload.C2M2MaturityLevel != nil {
		assessmentModel.C2M2MaturityLevel = payload.C2M2MaturityLevel
	}
	if payload.C2M2Comments != nil {
		assessmentModel.C2M2Comments = payload.C2M2Comments
	}
	if payload.C2M2AssessmentDate != nil && *payload.C2M2AssessmentDate != "" {
		parsedC2M2Date, errDate := time.Parse("2006-01-02", *payload.C2M2AssessmentDate)
		if errDate != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid c2m2_assessment_date format, use YYYY-MM-DD: " + errDate.Error()})
			return
		}
		assessmentModel.C2M2AssessmentDate = &parsedC2M2Date
	}


	// Handle file upload if "evidence_file" is provided
	file, header, errFile := c.Request.FormFile("evidence_file")
	if errFile == nil { // File was provided
		defer file.Close()

		// Validação de Tamanho do Arquivo
		if header.Size > maxEvidenceFileSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("File size exceeds limit of %d MB", maxEvidenceFileSize/(1024*1024))})
			return
		}

		// Validação de Tipo MIME
		// Ler os primeiros 512 bytes para detectar o tipo MIME
		buffer := make([]byte, 512)
		_, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file for MIME type detection"})
			return
		}
		// Resetar o ponteiro do arquivo para o início, para que o upload leia o arquivo completo
		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset file pointer"})
			return
		}

		mimeType := http.DetectContentType(buffer)
		log.Printf("Detected MIME type for uploaded file '%s': %s", header.Filename, mimeType)

		if !allowedMimeTypes[mimeType] {
			// Tentar verificar pela extensão se o DetectContentType falhar para alguns tipos Office
			// (DetectContentType pode retornar "application/zip" para .docx, .xlsx)
			ext := strings.ToLower(filepath.Ext(header.Filename))
			if (ext == ".docx" && mimeType == "application/zip" && allowedMimeTypes["application/vnd.openxmlformats-officedocument.wordprocessingml.document"]) ||
			   (ext == ".xlsx" && mimeType == "application/zip" && allowedMimeTypes["application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"]) {
				// Permitir se a extensão for conhecida e o MIME for application/zip (comum para formatos OOXML)
				log.Printf("Permitting ZIP file with known Office extension: %s", ext)
			} else {
				allowedTypesStr := []string{}
				for k := range allowedMimeTypes {
					allowedTypesStr = append(allowedTypesStr, k)
				}
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("File type '%s' (detected: '%s') is not allowed. Allowed types: %s", header.Filename, mimeType, strings.Join(allowedTypesStr, ", "))})
				return
			}
		}


		if filestorage.DefaultFileStorageProvider == nil {
			log.Println("Attempted file upload, but no file storage provider is configured.")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "File storage service is not configured."})
			return
		}

		// Construct a unique object name for GCS
		newFileName := fmt.Sprintf("%s_%s", uuid.New().String(), filepath.Base(header.Filename))
		objectPath := fmt.Sprintf("%s/audit_evidences/%s/%s", organizationID.String(), auditControlUUID.String(), newFileName)

		fileURL, errUpload := filestorage.DefaultFileStorageProvider.UploadFile(c.Request.Context(), organizationID.String(), objectPath, file)
		if errUpload != nil {
			log.Printf("Failed to upload evidence file to GCS: %v", errUpload)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload evidence file: " + errUpload.Error()})
			return
		}
		uploadedFileURL = fileURL // Override with the GCS URL
		// uploadedFileURL agora é objectName
		uploadedFileObjectName := fileURL
		log.Printf("Evidence file uploaded for org %s, control %s. ObjectName: %s", organizationID, auditControlUUID, uploadedFileObjectName)
		// O campo EvidenceURL no payload JSON (payload.EvidenceURL) é ignorado se um arquivo for enviado.
		// Armazenaremos o objectName no campo EvidenceURL do modelo.
		assessmentEvidenceIdentifier = uploadedFileObjectName
	} else if errFile != http.ErrMissingFile {
		// Some other error occurred with FormFile other than file simply not being there
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error processing evidence file: " + errFile.Error()})
		return
	}
	assessmentModel.EvidenceURL = assessmentEvidenceIdentifier // Definir após o processamento do arquivo


	db := database.GetDB()

	// Upsert logic: Update if (OrganizationID, AuditControlID) exists, else Create.
	// GORM's Clauses(clause.OnConflict...) is suitable here.
	// Adicionar novos campos C2M2 à lista de AssignmentColumns
	err = db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "organization_id"}, {Name: "audit_control_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"status", "evidence_url", "score", "assessment_date", "comments",
			"c2m2_maturity_level", "c2m2_assessment_date", "c2m2_comments",
			"updated_at",
		}),
	}).Create(&assessmentModel).Error // Usar assessmentModel

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create or update assessment: " + err.Error()})
		return
	}

	// Fetch the potentially created/updated record to ensure ID is present in response for new records.
	// If it was an update, `assessment.ID` might not be populated by the upsert without a specific returning clause.
	// A safer bet is to re-fetch.
	var resultAssessment models.AuditAssessment
	if err := db.Where("organization_id = ? AND audit_control_id = ?", organizationID, auditControlUUID).First(&resultAssessment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve created/updated assessment: " + err.Error()})
        return
	}

	c.JSON(http.StatusOK, resultAssessment) // OK for both create and update via upsert
}

// GetAssessmentForControlHandler gets the assessment for a specific control for the authenticated user's organization.
func GetAssessmentForControlHandler(c *gin.Context) {
	controlIDStr := c.Param("controlId") // This is AuditControl.ID (the UUID)
	controlUUID, err := uuid.Parse(controlIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid control UUID format"})
		return
	}

	orgID, orgExists := c.Get("organizationID")
	if !orgExists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Organization ID not found in token"})
		return
	}
	organizationID := orgID.(uuid.UUID)

	db := database.GetDB()
	var assessment models.AuditAssessment
	err = db.Where("organization_id = ? AND audit_control_id = ?", organizationID, controlUUID).
		Preload("AuditControl"). // Optionally preload control details
		First(&assessment).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// It's not an error for an assessment to not exist yet. Return empty or specific status.
			// For now, let's return 404 with a clear message.
			c.JSON(http.StatusNotFound, gin.H{"message": "No assessment found for this control in your organization."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch assessment: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, assessment)
}


// ListOrgAssessmentsByFrameworkHandler lists all assessments for a given organization and framework.
func ListOrgAssessmentsByFrameworkHandler(c *gin.Context) {
	orgIDStr := c.Param("orgId") // Could also get this from token if API is /api/v1/audit/frameworks/...
	targetOrgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}

	// Security check: Ensure the authenticated user can access this organization's assessments.
	tokenAuthOrgID, tokenAuthOrgExists := c.Get("organizationID")
	// userAuthRole, roleAuthExists := c.Get("userRole") // userRole não é usado para esta verificação específica

	if !tokenAuthOrgExists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: Missing authentication token information"})
		return
	}

	// User must belong to the organization they are trying to access.
	// A future superadmin role would bypass this.
	if tokenAuthOrgID.(uuid.UUID) != targetOrgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to the specified organization's assessments"})
		return
	}

	frameworkIDStr := c.Param("frameworkId")
	frameworkID, err := uuid.Parse(frameworkIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid framework ID format"})
		return
	}

	db := database.GetDB()
	var assessments []models.AuditAssessment

	// Find all AuditControlIDs for the given frameworkID
	var controls []models.AuditControl
	if err := db.Model(&models.AuditControl{}).Where("framework_id = ?", frameworkID).Pluck("id", &controls).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve controls for framework: " + err.Error()})
		return
	}
	if len(controls) == 0 {
		c.JSON(http.StatusOK, []models.AuditAssessment{}) // No controls, so no assessments
		return
	}

	var controlIDs []uuid.UUID
	for _, ctrl := range controls {
		controlIDs = append(controlIDs, ctrl.ID)
	}

	// Find all assessments for these controls within the target organization
	// Preload AuditControl to get details like ControlID (e.g., AC-1) and Description
	page, pageSize := GetPaginationParams(c)
	var totalItems int64

	query := db.Model(&models.AuditAssessment{}).
		Where("organization_id = ? AND audit_control_id IN (?)", targetOrgID, controlIDs)

	if err := query.Count(&totalItems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count assessments for framework: " + err.Error()})
		return
	}

	if err := query.Scopes(PaginateScope(page, pageSize)).Preload("AuditControl").Order("assessment_date desc").Find(&assessments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list assessments for framework: " + err.Error()})
		return
	}

	totalPages := totalItems / int64(pageSize)
	if totalItems%int64(pageSize) != 0 {
		totalPages++
	}
    if totalItems == 0 { totalPages = 0 }
    if totalPages == 0 && totalItems > 0 { totalPages = 1 }

	response := PaginatedResponse{
		Items:      assessments,
		TotalItems: totalItems,
		TotalPages: totalPages,
		Page:       page,
		PageSize:   pageSize,
	}
	c.JSON(http.StatusOK, response)
}

// ComplianceScoreResponse defines the structure for the compliance score endpoint.
type ComplianceScoreResponse struct {
	FrameworkID                 uuid.UUID `json:"framework_id"`
	FrameworkName               string    `json:"framework_name"`
	OrganizationID            uuid.UUID `json:"organization_id"`
	ComplianceScore             float64   `json:"compliance_score"` // Percentage
	TotalControls               int       `json:"total_controls"`
	EvaluatedControls           int       `json:"evaluated_controls"`
	ConformantControls          int       `json:"conformant_controls"`
	PartiallyConformantControls int       `json:"partially_conformant_controls"`
	NonConformantControls       int       `json:"non_conformant_controls"`
}

// GetComplianceScoreHandler calculates and returns the compliance score for a framework within an organization.
func GetComplianceScoreHandler(c *gin.Context) {
	orgIDStr := c.Param("orgId")
	targetOrgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}

	frameworkIDStr := c.Param("frameworkId")
	frameworkID, err := uuid.Parse(frameworkIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid framework ID format"})
		return
	}

	// Security check
	tokenAuthOrgID, tokenAuthOrgExists := c.Get("organizationID")
	// userAuthRole, roleAuthExists := c.Get("userRole") // userRole não é usado para esta verificação específica

	if !tokenAuthOrgExists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: Missing authentication token information"})
		return
	}

	// User must belong to the organization they are trying to access.
	// A future superadmin role would bypass this.
	if tokenAuthOrgID.(uuid.UUID) != targetOrgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to the specified organization's compliance score"})
		return
	}

	db := database.GetDB()

	// 1. Get Framework Details
	var framework models.AuditFramework
	if err := db.First(&framework, "id = ?", frameworkID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Framework not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch framework details: " + err.Error()})
		return
	}

	// 2. Get all controls for the framework
	var controls []models.AuditControl
	if err := db.Where("framework_id = ?", frameworkID).Find(&controls).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve controls for framework: " + err.Error()})
		return
	}

	totalControls := len(controls)
	if totalControls == 0 {
		resp := ComplianceScoreResponse{
			FrameworkID:    frameworkID,
			FrameworkName:  framework.Name,
			OrganizationID: targetOrgID,
			// All other counts will be 0, score 0.0 or NaN (handle NaN to be 0)
		}
		c.JSON(http.StatusOK, resp)
		return
	}

	var controlIDs []uuid.UUID
	for _, ctrl := range controls {
		controlIDs = append(controlIDs, ctrl.ID)
	}

	// 3. Get all assessments for these controls within the target organization
	var assessments []models.AuditAssessment
	if err := db.Where("organization_id = ? AND audit_control_id IN (?)", targetOrgID, controlIDs).Find(&assessments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list assessments for score calculation: " + err.Error()})
		return
	}

	// 4. Calculate score and counts
	evaluatedControls := 0
	conformantControls := 0
	partiallyConformantControls := 0
	nonConformantControls := 0
	totalScoreSum := 0

	assessmentMap := make(map[uuid.UUID]models.AuditAssessment)
	for _, assess := range assessments {
		assessmentMap[assess.AuditControlID] = assess
	}

	for _, ctrl := range controls {
		if assessment, found := assessmentMap[ctrl.ID]; found {
			evaluatedControls++
			totalScoreSum += assessment.Score // Assumes Score is 0-100
			switch assessment.Status {
			case models.ControlStatusConformant:
				conformantControls++
			case models.ControlStatusPartiallyConformant:
				partiallyConformantControls++
			case models.ControlStatusNonConformant:
				nonConformantControls++
			}
		}
	}

	var complianceScore float64
	if evaluatedControls > 0 {
		// Simple average of scores of *evaluated* controls.
		// Another interpretation could be average score across *all* controls, treating unevaluated as 0.
		// For now, average of evaluated.
		complianceScore = float64(totalScoreSum) / float64(evaluatedControls)
	} else {
		complianceScore = 0.0 // Or handle as NaN/null if preferred by frontend
	}

	// Ensure score is not NaN if evaluatedControls is 0, though already handled
	if evaluatedControls == 0 && totalScoreSum != 0 { // Should not happen if logic is correct
		complianceScore = 0.0
	}


	resp := ComplianceScoreResponse{
		FrameworkID:                 frameworkID,
		FrameworkName:               framework.Name,
		OrganizationID:            targetOrgID,
		ComplianceScore:             complianceScore,
		TotalControls:               totalControls,
		EvaluatedControls:           evaluatedControls,
		ConformantControls:          conformantControls,
		PartiallyConformantControls: partiallyConformantControls,
		NonConformantControls:       nonConformantControls,
	}

	c.JSON(http.StatusOK, resp)
}

// DeleteAssessmentEvidenceHandler remove a evidência de uma avaliação específica.
func DeleteAssessmentEvidenceHandler(c *gin.Context) {
	assessmentIDStr := c.Param("assessmentId")
	assessmentID, err := uuid.Parse(assessmentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assessment ID format"})
		return
	}

	orgIDToken, orgOk := c.Get("organizationID")
	if !orgOk {
		c.JSON(http.StatusForbidden, gin.H{"error": "Organization ID not found in token"})
		return
	}
	organizationID := orgIDToken.(uuid.UUID)

	// TODO: Adicionar verificação de role se necessário (ex: apenas admin/manager ou quem criou/atualizou a avaliação)
	// Por enquanto, qualquer um da organização pode remover a evidência de uma avaliação da sua organização.

	db := database.GetDB()
	var assessment models.AuditAssessment
	if err := db.Where("id = ? AND organization_id = ?", assessmentID, organizationID).First(&assessment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assessment not found or not part of your organization"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch assessment: " + err.Error()})
		return
	}

	if assessment.EvidenceURL == "" {
		c.JSON(http.StatusOK, gin.H{"message": "No evidence to delete for this assessment."})
		return
	}

	// Assume que EvidenceURL armazena o objectName se for um arquivo gerenciado,
	// ou uma URL externa se não for. Só tentamos deletar se não for uma URL HTTP(S).
	if !strings.HasPrefix(assessment.EvidenceURL, "http://") && !strings.HasPrefix(assessment.EvidenceURL, "https://") {
		if filestorage.DefaultFileStorageProvider == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "File storage provider not configured, cannot delete evidence file."})
			return
		}
		err := filestorage.DefaultFileStorageProvider.DeleteFile(c.Request.Context(), assessment.EvidenceURL)
		if err != nil {
			// Log o erro, mas continue para limpar o campo no DB.
			// O usuário pode ter deletado o arquivo diretamente no bucket.
			log.Printf("Failed to delete evidence file '%s' from storage, but proceeding to clear DB field: %v", assessment.EvidenceURL, err)
		}
	} else {
		log.Printf("EvidenceURL for assessment %s is an external URL, not deleting from managed storage.", assessmentID)
	}

	// Limpar o campo EvidenceURL e, opcionalmente, reajustar status/score
	assessment.EvidenceURL = ""
	// Poderia-se também resetar o score ou status se a remoção da evidência invalidar a avaliação.
	// Ex: assessment.Score = 0; assessment.Status = models.ControlStatusNonConformant (ou um novo status "evidence_removed")
	// Por ora, apenas remove a URL.

	if err := db.Save(&assessment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update assessment after deleting evidence: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Evidence deleted successfully from assessment."})
}

// --- C2M2 Maturity Summary Handlers ---

// C2M2MaturityDistribution representa a contagem de controles por nível MIL.
type C2M2MaturityDistribution struct {
	MIL0 int `json:"mil0"`
	MIL1 int `json:"mil1"`
	MIL2 int `json:"mil2"`
	MIL3 int `json:"mil3"`
}

// C2M2NISTComponentSummary resume a maturidade C2M2 para um componente NIST (Função ou Categoria).
type C2M2NISTComponentSummary struct {
	NISTComponentType string                     `json:"nist_component_type"` // "Function" ou "Category"
	NISTComponentName string                     `json:"nist_component_name"` // Nome da Função ou Categoria (ex: "Identify", "ID.AM")
	AchievedMIL       int                        `json:"achieved_mil"`        // Nível de Maturidade (0-3) agregado para este componente
	EvaluatedControls int                        `json:"evaluated_controls"`  // Número de controles NIST avaliados sob C2M2 neste componente
	TotalControls     int                        `json:"total_controls"`      // Número total de controles NIST neste componente
	MILDistribution   C2M2MaturityDistribution `json:"mil_distribution"`    // Distribuição dos MILs dos controles avaliados
}

// C2M2MaturityFrameworkSummaryResponse é a resposta para o sumário de maturidade C2M2 de um framework.
type C2M2MaturityFrameworkSummaryResponse struct {
	FrameworkID     uuid.UUID                    `json:"framework_id"`
	FrameworkName   string                       `json:"framework_name"`
	OrganizationID  uuid.UUID                    `json:"organization_id"`
	SummaryByFunction []C2M2NISTComponentSummary `json:"summary_by_function"`
	// SummaryByCategory []C2M2NISTComponentSummary `json:"summary_by_category,omitempty"` // Opcional, pode ser muito granular
}

// GetC2M2MaturitySummaryHandler calcula e retorna o sumário de maturidade C2M2
// para um framework dentro de uma organização, agregado por Função NIST.
func GetC2M2MaturitySummaryHandler(c *gin.Context) {
	orgIDStr := c.Param("orgId")
	targetOrgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}

	frameworkIDStr := c.Param("frameworkId")
	frameworkID, err := uuid.Parse(frameworkIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid framework ID format"})
		return
	}

	// Security check (usuário pertence à organização)
	tokenAuthOrgID, tokenAuthOrgExists := c.Get("organizationID")
	if !tokenAuthOrgExists || tokenAuthOrgID.(uuid.UUID) != targetOrgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to the specified organization's maturity summary"})
		return
	}

	db := database.GetDB()

	var framework models.AuditFramework
	if err := db.First(&framework, "id = ?", frameworkID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Framework not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch framework: " + err.Error()})
		return
	}

	var controls []models.AuditControl
	if err := db.Where("framework_id = ?", frameworkID).Order("control_id asc").Find(&controls).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list controls for framework: " + err.Error()})
		return
	}

	if len(controls) == 0 {
		c.JSON(http.StatusOK, C2M2MaturityFrameworkSummaryResponse{ // Retornar resposta vazia bem formada
			FrameworkID:    frameworkID,
			FrameworkName:  framework.Name,
			OrganizationID: targetOrgID,
			SummaryByFunction: []C2M2NISTComponentSummary{},
		})
		return
	}

	controlIDs := make([]uuid.UUID, len(controls))
	for i, ctrl := range controls {
		controlIDs[i] = ctrl.ID
	}

	var assessments []models.AuditAssessment
	if err := db.Where("organization_id = ? AND audit_control_id IN (?) AND c2m2_maturity_level IS NOT NULL", targetOrgID, controlIDs).
		Find(&assessments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch C2M2 assessments: " + err.Error()})
		return
	}

	// Mapear assessments por AuditControlID para fácil lookup
	assessmentMap := make(map[uuid.UUID]models.AuditAssessment)
	for _, assess := range assessments {
		assessmentMap[assess.AuditControlID] = assess
	}

	// Agrupar por Função NIST (extraída da Família do Controle, ex: "Identify (ID)")
	// Exemplo de Família: "Identify (ID.AM)", "Protect (PR.IP)"
	// A Função é a primeira parte antes do " (".
	summaryByFunction := make(map[string][]int)      // funcName -> lista de MILs
	controlsInFunction := make(map[string]int)       // funcName -> contagem total de controles
	evaluatedInFunction := make(map[string]int)      // funcName -> contagem de controles avaliados com C2M2
	milDistInFunction := make(map[string]*C2M2MaturityDistribution) // funcName -> distribuição

	nistFunctions := []string{"Identify", "Protect", "Detect", "Respond", "Recover", "Govern"} // Funções NIST CSF 2.0
	funcMap := make(map[string]bool)
	for _, fn := range nistFunctions {
		funcMap[fn] = true
		milDistInFunction[fn] = &C2M2MaturityDistribution{} // Inicializar
	}


	for _, ctrl := range controls {
		var nistFunction string
		// Tentar extrair a função da família. Ex: "Identify (ID.AM)" -> "Identify"
		// Ou "Access Control (AC)" -> "Access Control" (se não for estritamente NIST CSF)
		familyNameParts := strings.SplitN(ctrl.Family, " (", 2)
		if len(familyNameParts) > 0 {
			potentialFunction := strings.TrimSpace(familyNameParts[0])
			if funcMap[potentialFunction] { // Validar se é uma das funções NIST CSF conhecidas
				nistFunction = potentialFunction
			} else {
				// Se não for uma função NIST conhecida, podemos agrupar por família completa
				// ou ter uma categoria "Outros". Por simplicidade, vamos usar a família.
				// Para C2M2, focamos nas funções NIST. Se a família não mapear, pode ser um controle
				// que não se encaixa diretamente na agregação por função NIST CSF.
				// Ou, o campo `ctrl.Family` pode precisar ser padronizado para apenas a Função.
				// Por ora, se não for uma função conhecida, podemos pular na agregação por função.
				// No entanto, todos os controles do seeder NIST CSF devem ter famílias que mapeiam.
				// Ex: Família "Asset Management (ID.AM)" -> Função "Identify"
				// O seeder NIST CSF 2.0 usa a forma "Função (XX)" para família.
				// Ex: "Govern (GV.GV)", "Identify (ID.AM)"
				nistFunction = potentialFunction // Usar a parte antes do parêntese como função
				if !funcMap[nistFunction] { // Se ainda não for uma função conhecida, logar e pular
					// log.Printf("Controle '%s' com família '%s' não mapeia para uma Função NIST CSF conhecida.", ctrl.ControlID, ctrl.Family)
					// continue
					// Para o propósito do C2M2 e NIST CSF, vamos assumir que a família contém a função.
					// Se a família for "Risk Management Strategy (RS.MA)", a função é "Govern" (GV) no CSF 2.0
					// O seeder atual usa "Govern (GV.RM)" para RS.MA. Então a extração "Govern" está correta.
				}
			}
		}
		if nistFunction == "" { // Fallback ou se a lógica de extração falhar
			// log.Printf("Não foi possível determinar a Função NIST para o controle %s (Família: %s)", ctrl.ControlID, ctrl.Family)
			continue // Pular controles sem função clara para este sumário
		}


		controlsInFunction[nistFunction]++
		if assessment, found := assessmentMap[ctrl.ID]; found && assessment.C2M2MaturityLevel != nil {
			summaryByFunction[nistFunction] = append(summaryByFunction[nistFunction], *assessment.C2M2MaturityLevel)
			evaluatedInFunction[nistFunction]++
			if dist, ok := milDistInFunction[nistFunction]; ok {
				switch *assessment.C2M2MaturityLevel {
				case 0: dist.MIL0++
				case 1: dist.MIL1++
				case 2: dist.MIL2++
				case 3: dist.MIL3++
				}
			}
		}
	}

	var resultSummaries []C2M2NISTComponentSummary
	for _, nistFunction := range nistFunctions { // Iterar na ordem definida para consistência
		if _, exists := controlsInFunction[nistFunction]; !exists && !funcMap[nistFunction]{ // Pular se não for uma função conhecida ou não tiver controles
			continue
		}

		mils := summaryByFunction[nistFunction]
		achievedMIL := 0 // Default para MIL0
		if len(mils) > 0 {
			// Lógica de agregação de MIL (Simplificação Inicial: Moda)
			counts := make(map[int]int)
			maxCount := 0
			for _, mil := range mils {
				counts[mil]++
				if counts[mil] > maxCount {
					maxCount = counts[mil]
					achievedMIL = mil
				} else if counts[mil] == maxCount && mil > achievedMIL { // Desempate: pegar o MIL maior
					achievedMIL = mil
				}
			}
		}

		dist := milDistInFunction[nistFunction]
		if dist == nil { // Caso não haja controles avaliados para esta função
			dist = &C2M2MaturityDistribution{}
		}


		resultSummaries = append(resultSummaries, C2M2NISTComponentSummary{
			NISTComponentType: "Function",
			NISTComponentName: nistFunction,
			AchievedMIL:       achievedMIL,
			EvaluatedControls: evaluatedInFunction[nistFunction],
			TotalControls:     controlsInFunction[nistFunction],
			MILDistribution:   *dist,
		})
	}

	// Ordenar resultSummaries pela ordem de nistFunctions para consistência
	// (já está implícito pela iteração em nistFunctions, mas uma ordenação explícita seria mais robusta se a fonte de funções mudasse)


	response := C2M2MaturityFrameworkSummaryResponse{
		FrameworkID:     frameworkID,
		FrameworkName:   framework.Name,
		OrganizationID:  targetOrgID,
		SummaryByFunction: resultSummaries,
	}

	c.JSON(http.StatusOK, response)
}
