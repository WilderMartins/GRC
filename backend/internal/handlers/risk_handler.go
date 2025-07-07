package handlers

import (
	"net/http"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RiskPayload defines the structure for creating or updating a risk.
// OwnerID will be taken from the authenticated user or explicitly provided if allowed.
type RiskPayload struct {
	Title       string                `json:"title" binding:"required,min=3,max=255"`
	Description string                `json:"description"`
	Category    models.RiskCategory   `json:"category" binding:"omitempty,oneof=tecnologico operacional legal"` // Add more as defined
	Impact      models.RiskImpact     `json:"impact" binding:"omitempty,oneof=Baixo Médio Alto Crítico"`
	Probability models.RiskProbability `json:"probability" binding:"omitempty,oneof=Baixo Médio Alto Crítico"`
	Status      models.RiskStatus     `json:"status" binding:"omitempty,oneof=aberto em_andamento mitigado aceito"`
	OwnerID     string                `json:"owner_id"` // UUID as string, can be optional if creator is default owner
}

// CreateRiskHandler handles the creation of a new risk.
func CreateRiskHandler(c *gin.Context) {
	var payload RiskPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	db := database.GetDB()
	orgID, exists := c.Get("organizationID")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Organization ID not found in token"})
		return
	}
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in token"})
		return
	}

	var ownerUUID uuid.UUID
	if payload.OwnerID != "" {
		parsedOwnerID, err := uuid.Parse(payload.OwnerID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid OwnerID format"})
			return
		}
		// Optional: Check if this owner belongs to the same organization if necessary
		ownerUUID = parsedOwnerID
	} else {
		ownerUUID = userID.(uuid.UUID) // Default to creator if OwnerID is not provided
	}

	risk := models.Risk{
		// ID will be set by BeforeCreate hook
		OrganizationID: orgID.(uuid.UUID),
		Title:          payload.Title,
		Description:    payload.Description,
		Category:       payload.Category,
		Impact:         payload.Impact,
		Probability:    payload.Probability,
		Status:         payload.Status, // Default status can be set in model or here
		OwnerID:        ownerUUID,
	}
	if risk.Status == "" { // Set default status if not provided
		risk.Status = models.StatusOpen
	}


	if err := db.Create(&risk).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create risk: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, risk)
}

// GetRiskHandler handles fetching a single risk by its ID.
func GetRiskHandler(c *gin.Context) {
	riskIDStr := c.Param("riskId")
	riskID, err := uuid.Parse(riskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid risk ID format"})
		return
	}

	orgID, exists := c.Get("organizationID")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Organization ID not found in token"})
		return
	}

	db := database.GetDB()
	var risk models.Risk
	// Ensure risk belongs to the organization from the token
	if err := db.Preload("Owner").Where("id = ? AND organization_id = ?", riskID, orgID).First(&risk).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Risk not found or not part of your organization"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch risk: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, risk)
}

// ListRisksHandler handles fetching all risks for the organization.
// TODO: Implement pagination and filtering.
func ListRisksHandler(c *gin.Context) {
	orgID, exists := c.Get("organizationID")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Organization ID not found in token"})
		return
	}

	db := database.GetDB()
	var risks []models.Risk
	// Ensure risks belong to the organization from the token
	if err := db.Preload("Owner").Where("organization_id = ?", orgID).Find(&risks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list risks: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, risks)
}

// UpdateRiskHandler handles updating an existing risk.
func UpdateRiskHandler(c *gin.Context) {
	riskIDStr := c.Param("riskId")
	riskID, err := uuid.Parse(riskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid risk ID format"})
		return
	}

	var payload RiskPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	orgID, exists := c.Get("organizationID")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Organization ID not found in token"})
		return
	}
	userID, tokenUserExists := c.Get("userID") // To check if updater is owner or admin if needed
	userRole, tokenRoleExists := c.Get("userRole")


	db := database.GetDB()
	var risk models.Risk
	// Ensure risk belongs to the organization from the token before updating
	if err := db.Where("id = ? AND organization_id = ?", riskID, orgID).First(&risk).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Risk not found or not part of your organization"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch risk for update: " + err.Error()})
		return
	}

	// Authorization: Optionally, check if the user is the owner or an admin
	if tokenUserExists && tokenRoleExists {
		if risk.OwnerID != userID.(uuid.UUID) && userRole.(models.UserRole) != models.RoleAdmin && userRole.(models.UserRole) != models.RoleManager {
			// c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to update this risk"})
			// return
			// For now, let's allow update if user is in the org. More granular permissions can be added.
		}
	}


	// Update fields
	risk.Title = payload.Title
	risk.Description = payload.Description
	if payload.Category != "" {
		risk.Category = payload.Category
	}
	if payload.Impact != "" {
		risk.Impact = payload.Impact
	}
	if payload.Probability != "" {
		risk.Probability = payload.Probability
	}
	if payload.Status != "" {
		risk.Status = payload.Status
	}
	if payload.OwnerID != "" {
		parsedOwnerID, err := uuid.Parse(payload.OwnerID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid OwnerID format for update"})
			return
		}
		// Optional: Check if this new owner belongs to the same organization
		risk.OwnerID = parsedOwnerID
	}


	if err := db.Save(&risk).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update risk: " + err.Error()})
		return
	}

	// Fetch the updated risk with preloads to return
	var updatedRisk models.Risk
	db.Preload("Owner").Where("id = ?", risk.ID).First(&updatedRisk)

	c.JSON(http.StatusOK, updatedRisk)
}

// DeleteRiskHandler handles deleting a risk.
func DeleteRiskHandler(c *gin.Context) {
	riskIDStr := c.Param("riskId")
	riskID, err := uuid.Parse(riskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid risk ID format"})
		return
	}

	orgID, exists := c.Get("organizationID")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Organization ID not found in token"})
		return
	}
	// userID, tokenUserExists := c.Get("userID")
	// userRole, tokenRoleExists := c.Get("userRole")

	db := database.GetDB()

	// First, verify the risk exists and belongs to the organization.
	var risk models.Risk
	if err := db.Where("id = ? AND organization_id = ?", riskID, orgID).First(&risk).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Risk not found or not part of your organization"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch risk for deletion: " + err.Error()})
		return
	}

	// Authorization: Similar to Update, check if user is owner or admin/manager
	// if tokenUserExists && tokenRoleExists {
	// 	if risk.OwnerID != userID.(uuid.UUID) && userRole.(models.UserRole) != models.RoleAdmin && userRole.(models.UserRole) != models.RoleManager {
	// 		c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to delete this risk"})
	// 		return
	// 	}
	// }

	// GORM uses soft delete by default if the model has a gorm.DeletedAt field.
	// Our Risk model doesn't have it, so this will be a hard delete.
	// If soft delete is desired, add `DeletedAt gorm.DeletedAt \`gorm:"index"\` to models.Risk
	if err := db.Delete(&models.Risk{}, riskID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete risk: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Risk deleted successfully"})
}

// --- Approval Workflow Handlers ---

// SubmitRiskForAcceptanceHandler handles a manager/admin submitting a risk for acceptance.
func SubmitRiskForAcceptanceHandler(c *gin.Context) {
	riskIDStr := c.Param("riskId")
	riskID, err := uuid.Parse(riskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid risk ID format"})
		return
	}

	tokenOrgID, orgExists := c.Get("organizationID")
	if !orgExists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Organization ID not found in token"})
		return
	}
	tokenUserID, userExists := c.Get("userID")
	if !userExists {
		c.JSON(http.StatusForbidden, gin.H{"error": "User ID not found in token"})
		return
	}
	tokenUserRole, roleExists := c.Get("userRole")
	if !roleExists {
		c.JSON(http.StatusForbidden, gin.H{"error": "User role not found in token"})
		return
	}

	// Authorization: Only Admin or Manager can submit for acceptance
	if tokenUserRole.(models.UserRole) != models.RoleAdmin && tokenUserRole.(models.UserRole) != models.RoleManager {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins or managers can submit risks for acceptance"})
		return
	}

	db := database.GetDB()
	var risk models.Risk
	// Ensure risk exists and belongs to the organization
	if err := db.Where("id = ? AND organization_id = ?", riskID, tokenOrgID).First(&risk).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Risk not found or not part of your organization"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch risk: " + err.Error()})
		return
	}

	// Check if risk owner is set
	if risk.OwnerID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Risk must have an owner assigned before submitting for acceptance"})
		return
	}

    // Check if risk status allows submission (e.g., must be 'aberto' or similar)
    // For now, we allow submission regardless of current risk status, but this could be a future enhancement.

	// Check for existing pending workflow for this risk
	var existingWorkflow models.ApprovalWorkflow
	err = db.Where("risk_id = ? AND status = ?", riskID, models.ApprovalPending).First(&existingWorkflow).Error
	if err == nil { // Record found, means there's an existing pending workflow
		c.JSON(http.StatusConflict, gin.H{"error": "An approval workflow for this risk is already pending"})
		return
	}
	if err != nil && err != gorm.ErrRecordNotFound { // Other DB error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check for existing workflows: " + err.Error()})
		return
	}

	approvalWorkflow := models.ApprovalWorkflow{
		RiskID:      riskID,
		RequesterID: tokenUserID.(uuid.UUID),
		ApproverID:  risk.OwnerID, // Risk owner is the approver
		Status:      models.ApprovalPending,
	}

	if err := db.Create(&approvalWorkflow).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create approval workflow: " + err.Error()})
		return
	}

    // Placeholder for notification
	var requesterUser models.User
	var approverUser models.User
	db.First(&requesterUser, "id = ?", approvalWorkflow.RequesterID)
	db.First(&approverUser, "id = ?", approvalWorkflow.ApproverID)
	log.Printf("NOTIFICAÇÃO (Simulada): Risco '%s' (ID: %s) submetido para aprovação por '%s' para o aprovador '%s'. Workflow ID: %s",
		risk.Title, risk.ID.String(), requesterUser.Email, approverUser.Email, approvalWorkflow.ID.String())


	c.JSON(http.StatusCreated, approvalWorkflow)
}

// DecisionPayload for approving or rejecting risk acceptance
type DecisionPayload struct {
	Decision models.ApprovalStatus `json:"decision" binding:"required,oneof=aprovado rejeitado"`
	Comments string                `json:"comments"`
}

// ApproveOrRejectRiskAcceptanceHandler handles the decision (approve/reject) by the risk owner.
func ApproveOrRejectRiskAcceptanceHandler(c *gin.Context) {
	riskIDStr := c.Param("riskId") // Not strictly needed if approvalId is globally unique, but good for context
	riskID, err := uuid.Parse(riskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid risk ID format"})
		return
	}

	approvalIDStr := c.Param("approvalId")
	approvalID, err := uuid.Parse(approvalIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid approval workflow ID format"})
		return
	}

	var payload DecisionPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	tokenUserID, userExists := c.Get("userID")
	if !userExists {
		c.JSON(http.StatusForbidden, gin.H{"error": "User ID not found in token"})
		return
	}
	tokenOrgID, orgExists := c.Get("organizationID")
    if !orgExists {
        c.JSON(http.StatusForbidden, gin.H{"error": "Organization ID not found in token"})
        return
    }


	db := database.GetDB()
	var approvalWorkflow models.ApprovalWorkflow
	// Fetch the workflow, ensuring it belongs to the risk and is pending
	err = db.Joins("Risk").Where(`"approval_workflows"."id" = ? AND "approval_workflows"."risk_id" = ? AND "Risk"."organization_id" = ?`,
        approvalID, riskID, tokenOrgID).First(&approvalWorkflow).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Approval workflow not found, or does not belong to the specified risk/organization"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch approval workflow: " + err.Error()})
		return
	}

    // Authorization: Only the designated approver can decide
	if approvalWorkflow.ApproverID != tokenUserID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to decide on this approval workflow"})
		return
	}

	if approvalWorkflow.Status != models.ApprovalPending {
		c.JSON(http.StatusConflict, gin.H{"error": "This approval workflow has already been decided: " + approvalWorkflow.Status})
		return
	}

	// Start a transaction
	tx := db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start database transaction"})
		return
	}

	// Update workflow
	approvalWorkflow.Status = payload.Decision
	approvalWorkflow.Comments = payload.Comments
	// UpdatedAt will be set automatically by GORM
	if err := tx.Save(&approvalWorkflow).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update approval workflow: " + err.Error()})
		return
	}

	// If approved, update the risk status to "aceito"
	if payload.Decision == models.ApprovalApproved {
		var riskToUpdate models.Risk
		if err := tx.Where("id = ?", approvalWorkflow.RiskID).First(&riskToUpdate).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch risk for status update: " + err.Error()})
			return
		}
		riskToUpdate.Status = models.StatusAccepted
		if err := tx.Save(&riskToUpdate).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update risk status: " + err.Error()})
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

    // Placeholder for notification
    log.Printf("NOTIFICAÇÃO (Simulada): Workflow de aprovação ID %s para Risco ID %s foi %s pelo aprovador ID %s.",
        approvalWorkflow.ID.String(), approvalWorkflow.RiskID.String(), approvalWorkflow.Status, approvalWorkflow.ApproverID.String())


	c.JSON(http.StatusOK, approvalWorkflow)
}


// GetRiskApprovalHistoryHandler lists all approval workflows for a specific risk.
func GetRiskApprovalHistoryHandler(c *gin.Context) {
	riskIDStr := c.Param("riskId")
	riskID, err := uuid.Parse(riskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid risk ID format"})
		return
	}

	tokenOrgID, orgExists := c.Get("organizationID")
	if !orgExists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Organization ID not found in token"})
		return
	}
    // Any user in the org can view history? Or only involved parties/admins? For now, any user in org.

	db := database.GetDB()
	var risk models.Risk // To verify risk belongs to org
	if err := db.Where("id = ? AND organization_id = ?", riskID, tokenOrgID).First(&risk).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Risk not found or not part of your organization"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify risk: " + err.Error()})
		return
    }


	var approvalHistory []models.ApprovalWorkflow
	err = db.Where("risk_id = ?", riskID).
		Preload("Requester").Preload("Approver"). // Preload user details
		Order("created_at desc").
		Find(&approvalHistory).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch approval history: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, approvalHistory)
}
