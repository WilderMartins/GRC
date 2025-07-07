package handlers

import (
	"net/http"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	"time"
	"net/http/httputil" // Para detectar MIME type (embora http.DetectContentType seja mais simples)


	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"path/filepath"
	"phoenixgrc/backend/internal/filestorage"

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
	c.JSON(http.StatusOK, controls)
}

// --- Assessment Handlers ---

// AssessmentPayload defines the structure for creating or updating an assessment.
type AssessmentPayload struct {
	AuditControlID string                    `json:"audit_control_id" binding:"required"` // UUID of the AuditControl
	Status         models.AuditControlStatus `json:"status" binding:"required,oneof=conforme nao_conforme parcialmente_conforme"`
	EvidenceURL    string                    `json:"evidence_url" binding:"omitempty,url"`
	Score          *int                      `json:"score" binding:"omitempty,min=0,max=100"` // Pointer for optional score
	AssessmentDate string                    `json:"assessment_date" binding:"omitempty,datetime=2006-01-02"` // YYYY-MM-DD
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

	uploadedFileURL := payload.EvidenceURL // Use URL from payload if no file is uploaded or if it's preferred

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
		log.Printf("Evidence file uploaded for org %s, control %s: %s", organizationID, auditControlUUID, uploadedFileURL)

	} else if errFile != http.ErrMissingFile {
		// Some other error occurred with FormFile other than file simply not being there
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error processing evidence file: " + errFile.Error()})
		return
	}


	assessment := models.AuditAssessment{
		OrganizationID: organizationID,
		AuditControlID: auditControlUUID,
		Status:         payload.Status,
		EvidenceURL:    uploadedFileURL, // Use the URL from upload or from payload
		AssessmentDate: parsedAssessmentDate,
	}
	if payload.Score != nil {
		assessment.Score = *payload.Score
	} else {
		// Default score logic if needed, or leave it as 0 if not provided
		// For example, based on status:
		switch payload.Status {
		case models.ControlStatusConformant:
			assessment.Score = 100
		case models.ControlStatusPartiallyConformant:
			assessment.Score = 50
		case models.ControlStatusNonConformant:
			assessment.Score = 0
		}
	}


	db := database.GetDB()

	// Upsert logic: Update if (OrganizationID, AuditControlID) exists, else Create.
	// GORM's Clauses(clause.OnConflict...) is suitable here.
	err = db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "organization_id"}, {Name: "audit_control_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "evidence_url", "score", "assessment_date", "updated_at"}),
	}).Create(&assessment).Error
	// Note: The .Create(&assessment) will set the ID if it's a new record,
	// or update the existing record and GORM might reload the assessment struct with its ID.

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
	// For now, assume user's token orgID must match path orgId if they are not a superadmin.
	tokenOrgID, tokenOrgExists := c.Get("organizationID")
	userRole, roleExists := c.Get("userRole")

	if !tokenOrgExists || !roleExists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: Missing token information"})
		return
	}
	// Basic check: user must belong to the org, or be a superadmin (not implemented yet)
	if tokenOrgID.(uuid.UUID) != targetOrgID && userRole.(models.UserRole) != models.RoleAdmin { // Simplistic admin check
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to this organization's assessments"})
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
