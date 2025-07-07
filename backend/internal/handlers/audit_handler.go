package handlers

import (
	"net/http"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

// CreateOrUpdateAssessmentHandler creates a new assessment or updates an existing one.
// An assessment is unique per (OrganizationID, AuditControlID).
func CreateOrUpdateAssessmentHandler(c *gin.Context) {
	var payload AssessmentPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
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


	assessment := models.AuditAssessment{
		OrganizationID: organizationID,
		AuditControlID: auditControlUUID,
		Status:         payload.Status,
		EvidenceURL:    payload.EvidenceURL,
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
	if err := db.Preload("AuditControl").
		Where("organization_id = ? AND audit_control_id IN (?)", targetOrgID, controlIDs).
		Find(&assessments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list assessments for framework: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, assessments)
}
