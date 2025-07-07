package handlers

import (
	"net/http"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	"strings" // Para manipular EventTypes

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WebhookPayload defines the structure for creating or updating a WebhookConfiguration.
type WebhookPayload struct {
	Name       string   `json:"name" binding:"required,min=3,max=100"`
	URL        string   `json:"url" binding:"required,url,max=2048"`
	EventTypes []string `json:"event_types" binding:"required,dive,oneof=risk_created risk_status_changed"` // `dive` valida cada item do slice
	IsActive   *bool    `json:"is_active"` // Pointer to distinguish false from not provided
}

// Helper para serializar/desserializar EventTypes
func eventTypesToString(eventTypes []string) string {
	return strings.Join(eventTypes, ",")
}

func stringToEventTypes(eventTypesStr string) []string {
	if eventTypesStr == "" {
		return []string{}
	}
	return strings.Split(eventTypesStr, ",")
}

// CreateWebhookHandler handles adding a new webhook configuration for an organization.
func CreateWebhookHandler(c *gin.Context) {
	orgIDStr := c.Param("orgId")
	targetOrgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}

	// checkOrgAdmin é definido em idp_handler.go, idealmente seria movido para um pacote helper de auth/permissions
	// Por enquanto, vamos assumir que uma função similar está disponível ou reimplementar uma verificação básica.
	tokenOrgID, orgOk := c.Get("organizationID")
	tokenUserRole, roleOk := c.Get("userRole")
	if !orgOk || !roleOk || tokenOrgID.(uuid.UUID) != targetOrgID ||
		(tokenUserRole.(models.UserRole) != models.RoleAdmin && tokenUserRole.(models.UserRole) != models.RoleManager) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied or insufficient privileges"})
		return
	}


	var payload WebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	isActive := true
	if payload.IsActive != nil {
		isActive = *payload.IsActive
	}

	webhookConfig := models.WebhookConfiguration{
		OrganizationID: targetOrgID,
		Name:           payload.Name,
		URL:            payload.URL,
		EventTypes:     eventTypesToString(payload.EventTypes),
		IsActive:       isActive,
	}

	db := database.GetDB()
	if err := db.Create(&webhookConfig).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create webhook configuration: " + err.Error()})
		return
	}
    // Retornar o objeto criado, mas com EventTypes como slice
    response := webhookConfig
    // response.EventTypesSlice = stringToEventTypes(webhookConfig.EventTypes) // Se tivéssemos um campo extra para isso no DTO de resposta

	c.JSON(http.StatusCreated, response) // Ou um DTO de resposta que tenha EventTypes como slice
}

// ListWebhooksHandler lists all webhook configurations for an organization.
func ListWebhooksHandler(c *gin.Context) {
	orgIDStr := c.Param("orgId")
	targetOrgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}

    tokenOrgID, orgOk := c.Get("organizationID")
	tokenUserRole, roleOk := c.Get("userRole") // Pode não precisar de role admin para listar, só ser da org
	if !orgOk || !roleOk || tokenOrgID.(uuid.UUID) != targetOrgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}


	db := database.GetDB()
	var webhooks []models.WebhookConfiguration
	if err := db.Where("organization_id = ?", targetOrgID).Find(&webhooks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list webhook configurations: " + err.Error()})
		return
	}
    // Se quiser retornar EventTypes como slice no JSON de resposta:
    // type WebhookResponseDTO struct { ... EventTypes []string ... }
    // var responseDTOs []WebhookResponseDTO
    // for _, wh := range webhooks { ... converter e adicionar a responseDTOs ... }
	c.JSON(http.StatusOK, webhooks)
}

// GetWebhookHandler gets a specific webhook configuration.
func GetWebhookHandler(c *gin.Context) {
	orgIDStr := c.Param("orgId")
	targetOrgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}
	webhookIDStr := c.Param("webhookId")
	webhookID, err := uuid.Parse(webhookIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook ID format"})
		return
	}

    tokenOrgID, orgOk := c.Get("organizationID")
	if !orgOk || tokenOrgID.(uuid.UUID) != targetOrgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	db := database.GetDB()
	var webhook models.WebhookConfiguration
	if err := db.Where("id = ? AND organization_id = ?", webhookID, targetOrgID).First(&webhook).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Webhook configuration not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch webhook configuration: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, webhook)
}

// UpdateWebhookHandler updates an existing webhook configuration.
func UpdateWebhookHandler(c *gin.Context) {
	orgIDStr := c.Param("orgId")
	targetOrgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}
	webhookIDStr := c.Param("webhookId")
	webhookID, err := uuid.Parse(webhookIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook ID format"})
		return
	}

    tokenOrgID, orgOk := c.Get("organizationID")
	tokenUserRole, roleOk := c.Get("userRole")
	if !orgOk || !roleOk || tokenOrgID.(uuid.UUID) != targetOrgID ||
		(tokenUserRole.(models.UserRole) != models.RoleAdmin && tokenUserRole.(models.UserRole) != models.RoleManager) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied or insufficient privileges"})
		return
	}

	var payload WebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	db := database.GetDB()
	var webhook models.WebhookConfiguration
	if err := db.Where("id = ? AND organization_id = ?", webhookID, targetOrgID).First(&webhook).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Webhook configuration not found for update"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch webhook for update: " + err.Error()})
		return
	}

	webhook.Name = payload.Name
	webhook.URL = payload.URL
	webhook.EventTypes = eventTypesToString(payload.EventTypes)
	if payload.IsActive != nil {
		webhook.IsActive = *payload.IsActive
	}

	if err := db.Save(&webhook).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update webhook configuration: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, webhook)
}

// DeleteWebhookHandler deletes a webhook configuration.
func DeleteWebhookHandler(c *gin.Context) {
	orgIDStr := c.Param("orgId")
	targetOrgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
		return
	}
	webhookIDStr := c.Param("webhookId")
	webhookID, err := uuid.Parse(webhookIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook ID format"})
		return
	}

    tokenOrgID, orgOk := c.Get("organizationID")
	tokenUserRole, roleOk := c.Get("userRole")
	if !orgOk || !roleOk || tokenOrgID.(uuid.UUID) != targetOrgID ||
		(tokenUserRole.(models.UserRole) != models.RoleAdmin && tokenUserRole.(models.UserRole) != models.RoleManager) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied or insufficient privileges"})
		return
	}

	db := database.GetDB()
	// Verify it exists before deleting
	var webhook models.WebhookConfiguration
	if err := db.Where("id = ? AND organization_id = ?", webhookID, targetOrgID).First(&webhook).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Webhook configuration not found for deletion"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch webhook for deletion: " + err.Error()})
		return
    }

	if err := db.Delete(&models.WebhookConfiguration{}, "id = ? AND organization_id = ?", webhookID, targetOrgID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete webhook configuration: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Webhook configuration deleted successfully"})
}
