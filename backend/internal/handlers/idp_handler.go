package handlers

import (
	"encoding/json"
	"net/http"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IdentityProviderPayload defines the structure for creating or updating an Identity Provider.
type IdentityProviderPayload struct {
	ProviderType         models.IdentityProviderType `json:"provider_type" binding:"required,oneof=saml oauth2_google oauth2_github"`
	Name                 string                    `json:"name" binding:"required,min=3,max=100"`
	IsActive             *bool                     `json:"is_active"` // Pointer to distinguish between false and not provided
	ConfigJSON           json.RawMessage           `json:"config_json" binding:"required"` // Keep as RawMessage for flexibility
	AttributeMappingJSON json.RawMessage           `json:"attribute_mapping_json"`       // Optional
}

// Helper to check if the authenticated user is an admin of the target organization
func checkOrgAdmin(c *gin.Context, targetOrgID uuid.UUID) bool {
	tokenOrgID, orgExists := c.Get("organizationID")
	tokenUserRole, roleExists := c.Get("userRole")

	if !orgExists || !roleExists {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied: Missing token information"})
		return false
	}
	if tokenOrgID.(uuid.UUID) != targetOrgID {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied: You do not belong to this organization"})
		return false
	}
	// Assuming RoleAdmin allows managing IdPs for their own organization.
	// More granular permissions could be added later.
	if tokenUserRole.(models.UserRole) != models.RoleAdmin && tokenUserRole.(models.UserRole) != models.RoleManager {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied: Insufficient privileges"})
		return false
	}
	return true
}


// CreateIdentityProviderHandler handles adding a new identity provider for an organization.
func CreateIdentityProviderHandler(c *gin.Context) {
	orgIDStr := c.Param("orgId")
	targetOrgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}

	if !checkOrgAdmin(c, targetOrgID) {
		return // Error response already sent by checkOrgAdmin
	}

	var payload IdentityProviderPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Validate ConfigJSON (basic validation, more specific validation might be needed per provider_type)
	var tempConfig interface{}
	if err := json.Unmarshal(payload.ConfigJSON, &tempConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ConfigJSON format: " + err.Error()})
		return
	}
	if payload.AttributeMappingJSON != nil {
		var tempMapping interface{}
		if err := json.Unmarshal(payload.AttributeMappingJSON, &tempMapping); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid AttributeMappingJSON format: " + err.Error()})
			return
		}
	}

	isActive := true // Default to true if not provided
	if payload.IsActive != nil {
		isActive = *payload.IsActive
	}


	idp := models.IdentityProvider{
		OrganizationID:       targetOrgID,
		ProviderType:         payload.ProviderType,
		Name:                 payload.Name,
		IsActive:             isActive,
		ConfigJSON:           string(payload.ConfigJSON),
		AttributeMappingJSON: string(payload.AttributeMappingJSON),
	}

	db := database.GetDB()
	if err := db.Create(&idp).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create identity provider: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, idp)
}

// ListIdentityProvidersHandler lists all identity providers for an organization.
func ListIdentityProvidersHandler(c *gin.Context) {
	orgIDStr := c.Param("orgId")
	targetOrgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}

	// For listing, user might just need to be part of the org, not necessarily admin.
	// The checkOrgAdmin currently requires Admin or Manager.
	// Let's use a simpler check for listing: just ensure token's orgID matches path orgID.
	tokenOrgID, orgOk := c.Get("organizationID")
	if !orgOk || tokenOrgID.(uuid.UUID) != targetOrgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to this organization's identity providers"})
		return
	}

	page, pageSize := GetPaginationParams(c)
	db := database.GetDB()
	var idps []models.IdentityProvider
	var totalItems int64

	query := db.Model(&models.IdentityProvider{}).Where("organization_id = ?", targetOrgID)
	if err := query.Count(&totalItems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count identity providers: " + err.Error()})
		return
	}

	if err := query.Scopes(PaginateScope(page, pageSize)).Order("created_at desc").Find(&idps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list identity providers: " + err.Error()})
		return
	}

	totalPages := totalItems / int64(pageSize)
	if totalItems%int64(pageSize) != 0 {
		totalPages++
	}
    if totalItems == 0 { totalPages = 0 }
    if totalPages == 0 && totalItems > 0 { totalPages = 1 }

	response := PaginatedResponse{
		Items:      idps,
		TotalItems: totalItems,
		TotalPages: totalPages,
		Page:       page,
		PageSize:   pageSize,
	}
	c.JSON(http.StatusOK, response)
}

// GetIdentityProviderHandler gets a specific identity provider.
func GetIdentityProviderHandler(c *gin.Context) {
	orgIDStr := c.Param("orgId")
	targetOrgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}
	idpIDStr := c.Param("idpId")
	idpID, err := uuid.Parse(idpIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid identity provider ID format"})
		return
	}

	if !checkOrgAdmin(c, targetOrgID) {
		return
	}

	db := database.GetDB()
	var idp models.IdentityProvider
	if err := db.Where("id = ? AND organization_id = ?", idpID, targetOrgID).First(&idp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Identity provider not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch identity provider: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, idp)
}

// UpdateIdentityProviderHandler updates an existing identity provider.
func UpdateIdentityProviderHandler(c *gin.Context) {
	orgIDStr := c.Param("orgId")
	targetOrgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}
	idpIDStr := c.Param("idpId")
	idpID, err := uuid.Parse(idpIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid identity provider ID format"})
		return
	}

	if !checkOrgAdmin(c, targetOrgID) {
		return
	}

	var payload IdentityProviderPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Validate ConfigJSON
	var tempConfig interface{}
	if err := json.Unmarshal(payload.ConfigJSON, &tempConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ConfigJSON format: " + err.Error()})
		return
	}
	if payload.AttributeMappingJSON != nil {
		var tempMapping interface{}
		if err := json.Unmarshal(payload.AttributeMappingJSON, &tempMapping); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid AttributeMappingJSON format: " + err.Error()})
			return
		}
	}

	db := database.GetDB()
	var idp models.IdentityProvider
	if err := db.Where("id = ? AND organization_id = ?", idpID, targetOrgID).First(&idp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Identity provider not found for update"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch identity provider for update: " + err.Error()})
		return
	}

	idp.ProviderType = payload.ProviderType
	idp.Name = payload.Name
	if payload.IsActive != nil {
		idp.IsActive = *payload.IsActive
	}
	idp.ConfigJSON = string(payload.ConfigJSON)
	idp.AttributeMappingJSON = string(payload.AttributeMappingJSON)


	if err := db.Save(&idp).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update identity provider: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, idp)
}

// DeleteIdentityProviderHandler deletes an identity provider.
func DeleteIdentityProviderHandler(c *gin.Context) {
	orgIDStr := c.Param("orgId")
	targetOrgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}
	idpIDStr := c.Param("idpId")
	idpID, err := uuid.Parse(idpIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid identity provider ID format"})
		return
	}

	if !checkOrgAdmin(c, targetOrgID) {
		return
	}

	db := database.GetDB()
	// Verify it exists before deleting
	var idp models.IdentityProvider
	if err := db.Where("id = ? AND organization_id = ?", idpID, targetOrgID).First(&idp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Identity provider not found for deletion"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch identity provider for deletion: " + err.Error()})
		return
	}


	if err := db.Delete(&models.IdentityProvider{}, "id = ? AND organization_id = ?", idpID, targetOrgID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete identity provider: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Identity provider deleted successfully"})
}
