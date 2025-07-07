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
	Impact      models.RiskImpact     `json:"impact" binding:"omitempty,oneof=baixo medio alto critico"`
	Probability models.RiskProbability `json:"probability" binding:"omitempty,oneof=baixa media alta critica"`
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
