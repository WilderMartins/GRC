package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/filestorage"
	"phoenixgrc/backend/internal/models"
	phxlog "phoenixgrc/backend/pkg/log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	C2M2AssessmentDate *string `json:"c2m2_assessment_date,omitempty" binding:"omitempty,datetime=2006-01-02"` // YYYY-MM-DD
	C2M2Comments      *string `json:"c2m2_comments,omitempty"`

	// Novo campo para receber as avaliações detalhadas das práticas C2M2
	C2M2PracticeEvaluations map[string]string `json:"c2m2_practice_evaluations,omitempty"` // map[practiceID] -> status
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
}

// CreateOrUpdateAssessmentHandler creates a new assessment or updates an existing one.
func CreateOrUpdateAssessmentHandler(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form: " + err.Error()})
		return
	}

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

	if payload.AssessmentDate != "" {
		if _, err := time.Parse("2006-01-02", payload.AssessmentDate); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assessment_date format, use YYYY-MM-DD: " + err.Error()})
			return
		}
	}
	assessmentEvidenceIdentifier := payload.EvidenceURL

	assessmentModel := models.AuditAssessment{
		OrganizationID: organizationID,
		AuditControlID: auditControlUUID,
		Status:         payload.Status,
		C2M2Comments:      payload.C2M2Comments,
	}

	if payload.AssessmentDate != "" {
		parsedDate, _ := time.Parse("2006-01-02", payload.AssessmentDate)
		assessmentModel.AssessmentDate = &parsedDate
	} else {
		now := time.Now()
		assessmentModel.AssessmentDate = &now
	}

	if payload.C2M2AssessmentDate != nil && *payload.C2M2AssessmentDate != "" {
		parsedDate, err := time.Parse("2006-01-02", *payload.C2M2AssessmentDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid c2m2_assessment_date format, use YYYY-MM-DD: " + err.Error()})
			return
		}
		assessmentModel.C2M2AssessmentDate = &parsedDate
	}

	if payload.Score != nil {
		assessmentModel.Score = payload.Score
	} else {
		var defaultScore int
		switch payload.Status {
		case models.ControlStatusConformant:
			defaultScore = 100
		case models.ControlStatusPartiallyConformant:
			defaultScore = 50
		case models.ControlStatusNonConformant, models.ControlStatusNotApplicable:
			defaultScore = 0
		}
		assessmentModel.Score = &defaultScore
	}

	file, header, errFile := c.Request.FormFile("evidence_file")
	if errFile == nil {
		defer file.Close()

		if header.Size > maxEvidenceFileSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("File size exceeds limit of %d MB", maxEvidenceFileSize/(1024*1024))})
			return
		}

		buffer := make([]byte, 512)
		_, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file for MIME type detection"})
			return
		}
		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset file pointer"})
			return
		}

		mimeType := http.DetectContentType(buffer)
		log.Printf("Detected MIME type for uploaded file '%s': %s", header.Filename, mimeType)

		if !allowedMimeTypes[mimeType] {
			ext := strings.ToLower(filepath.Ext(header.Filename))
			if (ext == ".docx" && mimeType == "application/zip" && allowedMimeTypes["application/vnd.openxmlformats-officedocument.wordprocessingml.document"]) ||
				(ext == ".xlsx" && mimeType == "application/zip" && allowedMimeTypes["application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"]) {
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

		newFileName := fmt.Sprintf("%s_%s", uuid.New().String(), filepath.Base(header.Filename))
		objectPath := fmt.Sprintf("%s/audit_evidences/%s/%s", organizationID.String(), auditControlUUID.String(), newFileName)

		uploadedFileObjectName, errUpload := filestorage.DefaultFileStorageProvider.UploadFile(c.Request.Context(), organizationID.String(), objectPath, file)
		if errUpload != nil {
			log.Printf("Failed to upload evidence file to GCS: %v", errUpload)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload evidence file: " + errUpload.Error()})
			return
		}
		log.Printf("Evidence file uploaded for org %s, control %s. ObjectName: %s", organizationID, auditControlUUID, uploadedFileObjectName)
		assessmentEvidenceIdentifier = uploadedFileObjectName
	} else if errFile != http.ErrMissingFile {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error processing evidence file: " + errFile.Error()})
		return
	}
	assessmentModel.EvidenceURL = assessmentEvidenceIdentifier

	db := database.GetDB()

	err = db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "organization_id"}, {Name: "audit_control_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"status", "evidence_url", "score", "assessment_date",
			"c2m2_assessment_date", "c2m2_comments",
			"updated_at",
		}),
	}).Create(&assessmentModel).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create or update assessment: " + err.Error()})
		return
	}

	var resultAssessment models.AuditAssessment
	if err := db.Where("organization_id = ? AND audit_control_id = ?", organizationID, auditControlUUID).First(&resultAssessment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve created/updated assessment: " + err.Error()})
        return
	}

	if payload.C2M2PracticeEvaluations != nil && len(payload.C2M2PracticeEvaluations) > 0 {
		var evalsToUpsert []models.C2M2PracticeEvaluation
		for practiceIDStr, status := range payload.C2M2PracticeEvaluations {
			practiceID, errParse := uuid.Parse(practiceIDStr)
			if errParse != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid practice ID format in c2m2_practice_evaluations: %s", practiceIDStr)})
				return
			}
			validStatuses := map[string]bool{
				string(models.PracticeStatusNotImplemented):      true,
				string(models.PracticeStatusPartiallyImplemented): true,
				string(models.PracticeStatusFullyImplemented):    true,
			}
			if !validStatuses[status] {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid status '%s' for practice ID %s", status, practiceIDStr)})
				return
			}

			eval := models.C2M2PracticeEvaluation{
				AuditAssessmentID: resultAssessment.ID,
				PracticeID:        practiceID,
				Status:            models.PracticeStatus(status),
			}
			evalsToUpsert = append(evalsToUpsert, eval)
		}

		if len(evalsToUpsert) > 0 {
			tx := db.Begin()
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "audit_assessment_id"}, {Name: "practice_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"status", "updated_at"}),
			}).Create(&evalsToUpsert).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save C2M2 practice evaluations: " + err.Error()})
				return
			}
			tx.Commit()
		}
	}

	if err := db.Preload("C2M2PracticeEvaluations").First(&resultAssessment, resultAssessment.ID).Error; err != nil {
		phxlog.L.Warn("Failed to re-fetch assessment with practice evaluations", zap.String("assessmentID", resultAssessment.ID.String()), zap.Error(err))
	}

	c.JSON(http.StatusOK, resultAssessment)
}

// GetAssessmentForControlHandler gets the assessment for a specific control for the authenticated user's organization.
func GetAssessmentForControlHandler(c *gin.Context) {
	controlIDStr := c.Param("controlId")
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
		Preload("AuditControl").
		First(&assessment).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
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
	orgIDStr := c.Param("orgId")
	targetOrgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}

	tokenAuthOrgID, tokenAuthOrgExists := c.Get("organizationID")

	if !tokenAuthOrgExists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: Missing authentication token information"})
		return
	}

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

	var controls []models.AuditControl
	if err := db.Model(&models.AuditControl{}).Where("framework_id = ?", frameworkID).Pluck("id", &controls).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve controls for framework: " + err.Error()})
		return
	}
	if len(controls) == 0 {
		c.JSON(http.StatusOK, []models.AuditAssessment{})
		return
	}

	var controlIDs []uuid.UUID
	for _, ctrl := range controls {
		controlIDs = append(controlIDs, ctrl.ID)
	}

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
	if totalItems == 0 {
		totalPages = 0
	}
	if totalPages == 0 && totalItems > 0 {
		totalPages = 1
	}

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
	ComplianceScore             float64   `json:"compliance_score"`
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

	tokenAuthOrgID, tokenAuthOrgExists := c.Get("organizationID")

	if !tokenAuthOrgExists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: Missing authentication token information"})
		return
	}

	if tokenAuthOrgID.(uuid.UUID) != targetOrgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to the specified organization's compliance score"})
		return
	}

	db := database.GetDB()

	var framework models.AuditFramework
	if err := db.First(&framework, "id = ?", frameworkID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Framework not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch framework details: " + err.Error()})
		return
	}

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
		}
		c.JSON(http.StatusOK, resp)
		return
	}

	var controlIDs []uuid.UUID
	for _, ctrl := range controls {
		controlIDs = append(controlIDs, ctrl.ID)
	}

	var assessments []models.AuditAssessment
	if err := db.Where("organization_id = ? AND audit_control_id IN (?)", targetOrgID, controlIDs).Find(&assessments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list assessments for score calculation: " + err.Error()})
		return
	}

	evaluatedControls := 0
	conformantControls := 0
	partiallyConformantControls := 0
	nonConformantControls := 0
	var totalScoreSum int

	assessmentMap := make(map[uuid.UUID]models.AuditAssessment)
	for _, assess := range assessments {
		assessmentMap[assess.AuditControlID] = assess
	}

	for _, ctrl := range controls {
		if assessment, found := assessmentMap[ctrl.ID]; found {
			evaluatedControls++
			if assessment.Score != nil {
				totalScoreSum += *assessment.Score
			}
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
		complianceScore = float64(totalScoreSum) / float64(evaluatedControls)
	} else {
		complianceScore = 0.0
	}

	if evaluatedControls == 0 && totalScoreSum != 0 {
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

	if !strings.HasPrefix(assessment.EvidenceURL, "http://") && !strings.HasPrefix(assessment.EvidenceURL, "https://") {
		if filestorage.DefaultFileStorageProvider == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "File storage provider not configured, cannot delete evidence file."})
			return
		}
		err := filestorage.DefaultFileStorageProvider.DeleteFile(c.Request.Context(), assessment.EvidenceURL)
		if err != nil {
			log.Printf("Failed to delete evidence file '%s' from storage, but proceeding to clear DB field: %v", assessment.EvidenceURL, err)
		}
	} else {
		log.Printf("EvidenceURL for assessment %s is an external URL, not deleting from managed storage.", assessmentID)
	}

	assessment.EvidenceURL = ""

	if err := db.Save(&assessment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update assessment after deleting evidence: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Evidence deleted successfully from assessment."})
}

// --- C2M2 Maturity Summary Handlers ---

type C2M2MaturityDistribution struct {
	MIL0 int `json:"mil0"`
	MIL1 int `json:"mil1"`
	MIL2 int `json:"mil2"`
	MIL3 int `json:"mil3"`
}

type C2M2NISTComponentSummary struct {
	NISTComponentType string                     `json:"nist_component_type"`
	NISTComponentName string                     `json:"nist_component_name"`
	AchievedMIL       int                        `json:"achieved_mil"`
	EvaluatedControls int                        `json:"evaluated_controls"`
	TotalControls     int                        `json:"total_controls"`
	MILDistribution   C2M2MaturityDistribution `json:"mil_distribution"`
}

type C2M2MaturityFrameworkSummaryResponse struct {
	FrameworkID     uuid.UUID                    `json:"framework_id"`
	FrameworkName   string                       `json:"framework_name"`
	OrganizationID  uuid.UUID                    `json:"organization_id"`
	SummaryByFunction []C2M2NISTComponentSummary `json:"summary_by_function"`
}

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
		c.JSON(http.StatusOK, C2M2MaturityFrameworkSummaryResponse{
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

	// TODO: A lógica de cálculo de maturidade foi alterada.
	// O campo `c2m2_maturity_level` foi removido de `AuditAssessment`.
	// A nova lógica deve carregar as `C2M2PracticeEvaluation` para cada assessment
	// e calcular a maturidade com base no status dessas práticas.
	// Por enquanto, este handler retornará uma estrutura vazia para permitir a compilação.

	response := C2M2MaturityFrameworkSummaryResponse{
		FrameworkID:     frameworkID,
		FrameworkName:   framework.Name,
		OrganizationID:  targetOrgID,
		SummaryByFunction: []C2M2NISTComponentSummary{}, // Retornar vazio
	}
	c.JSON(http.StatusOK, response)
	return // Retornar aqui para pular a lógica antiga e quebrada.

	// A lógica antiga foi removida.

	summaryByFunction := make(map[string][]int)
	controlsInFunction := make(map[string]int)
	evaluatedInFunction := make(map[string]int)
	milDistInFunction := make(map[string]*C2M2MaturityDistribution)

	nistFunctions := []string{"Identify", "Protect", "Detect", "Respond", "Recover", "Govern"}
	funcMap := make(map[string]bool)
	for _, fn := range nistFunctions {
		funcMap[fn] = true
		milDistInFunction[fn] = &C2M2MaturityDistribution{}
	}

	for _, ctrl := range controls {
		var nistFunction string
		familyNameParts := strings.SplitN(ctrl.Family, " (", 2)
		if len(familyNameParts) > 0 {
			potentialFunction := strings.TrimSpace(familyNameParts[0])
			if funcMap[potentialFunction] {
				nistFunction = potentialFunction
			} else {
				nistFunction = potentialFunction
			}
		}
		if nistFunction == "" {
			continue
		}

		controlsInFunction[nistFunction]++
	}

	var resultSummaries []C2M2NISTComponentSummary
	for _, nistFunction := range nistFunctions {
		if _, exists := controlsInFunction[nistFunction]; !exists && !funcMap[nistFunction] {
			continue
		}

		achievedMIL := 0
		mils := summaryByFunction[nistFunction]
		if len(mils) > 0 {
			counts := make(map[int]int)
			maxCount := 0
			for _, mil := range mils {
				counts[mil]++
				if counts[mil] > maxCount {
					maxCount = counts[mil]
					achievedMIL = mil
				} else if counts[mil] == maxCount && mil > achievedMIL {
					achievedMIL = mil
				}
			}
		}

		dist := milDistInFunction[nistFunction]
		if dist == nil {
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

	finalResponse := C2M2MaturityFrameworkSummaryResponse{
		FrameworkID:     frameworkID,
		FrameworkName:   framework.Name,
		OrganizationID:  targetOrgID,
		SummaryByFunction: resultSummaries,
	}

	c.JSON(http.StatusOK, finalResponse)
}
