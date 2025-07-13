package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	phxlog "phoenixgrc/backend/pkg/log"
	"strings" // Para manipular EventTypes
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// EventType defines the types of events that can trigger a webhook.
type EventType string

const (
	EventRiskCreated          EventType = "risk_created"
	EventVulnerabilityAssigned EventType = "vulnerability_assigned"
)

// WebhookPayloadData defines the structure of the data sent in the webhook.
type WebhookPayloadData struct {
	Event     EventType   `json:"event"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// TriggerWebhooks finds relevant webhooks and sends the payload.
func TriggerWebhooks(orgID uuid.UUID, eventType EventType, data interface{}) {
	log := phxlog.L.Named("TriggerWebhooks")
	db := database.GetDB()

	var webhooks []models.WebhookConfiguration
	// Find active webhooks for the organization that are subscribed to the event type.
	if err := db.Where("organization_id = ? AND is_active = ? AND event_types LIKE ?", orgID, true, "%"+string(eventType)+"%").Find(&webhooks).Error; err != nil {
		log.Error("Failed to retrieve webhooks for triggering", zap.Error(err), zap.String("organization_id", orgID.String()))
		return
	}

	if len(webhooks) == 0 {
		return // No webhooks to trigger for this event
	}

	payload := WebhookPayloadData{
		Event:     eventType,
		Timestamp: time.Now(),
		Data:      data,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Error("Failed to marshal webhook payload", zap.Error(err))
		return
	}

	for _, wh := range webhooks {
		go sendWebhook(wh, payloadBytes)
	}
}

func sendWebhook(webhook models.WebhookConfiguration, payload []byte) {
	log := phxlog.L.Named("sendWebhook").With(zap.String("webhook_id", webhook.ID.String()), zap.String("url", webhook.URL))

	req, err := http.NewRequest("POST", webhook.URL, bytes.NewBuffer(payload))
	if err != nil {
		log.Error("Failed to create webhook request", zap.Error(err))
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "PhoenixGRC-Webhook/1.0")

	if webhook.SecretToken != "" {
		mac := hmac.New(sha256.New, []byte(webhook.SecretToken))
		mac.Write(payload)
		signature := fmt.Sprintf("sha256=%x", mac.Sum(nil))
		req.Header.Set("X-Phoenix-Signature-256", signature)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Error("Failed to send webhook", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Info("Webhook sent successfully")
	} else {
		log.Warn("Webhook sent but received non-success status code", zap.Int("status_code", resp.StatusCode))
	}
}

// WebhookPayload defines the structure for creating or updating a WebhookConfiguration.
type WebhookPayload struct {
	Name        string   `json:"name" binding:"required,min=3,max=100"`
	URL         string   `json:"url" binding:"required,url,max=2048"`
	EventTypes  []string `json:"event_types" binding:"required,dive,oneof=risk_created risk_status_changed"` // `dive` valida cada item do slice
	IsActive    *bool    `json:"is_active"`    // Pointer to distinguish false from not provided
	SecretToken *string  `json:"secret_token"` // Opcional
}

// WebhookResponseItem é o DTO para respostas de webhook, incluindo EventTypes como slice.
type WebhookResponseItem struct {
	models.WebhookConfiguration
	EventTypesList []string `json:"event_types_list"`
}

// newWebhookResponseItem cria um WebhookResponseItem a partir de um WebhookConfiguration.
func newWebhookResponseItem(wh models.WebhookConfiguration) WebhookResponseItem {
	return WebhookResponseItem{
		WebhookConfiguration: wh,
		EventTypesList:       stringToEventTypes(wh.EventTypes),
	}
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
	if payload.SecretToken != nil { // Adicionar SecretToken se fornecido
		webhookConfig.SecretToken = *payload.SecretToken
	}

	db := database.GetDB()
	if err := db.Create(&webhookConfig).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create webhook configuration: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, newWebhookResponseItem(webhookConfig))
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
	// Para listar, apenas verificamos se o usuário pertence à organização.
	if !orgOk || tokenOrgID.(uuid.UUID) != targetOrgID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to this organization's webhooks"})
		return
	}

	page, pageSize := GetPaginationParams(c)
	db := database.GetDB()
	var webhooks []models.WebhookConfiguration
	var totalItems int64

	query := db.Model(&models.WebhookConfiguration{}).Where("organization_id = ?", targetOrgID)
	if err := query.Count(&totalItems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count webhook configurations: " + err.Error()})
		return
	}

	if err := query.Scopes(PaginateScope(page, pageSize)).Order("created_at desc").Find(&webhooks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list webhook configurations: " + err.Error()})
		return
	}

	totalPages := totalItems / int64(pageSize)
	if totalItems%int64(pageSize) != 0 {
		totalPages++
	}
    if totalItems == 0 { totalPages = 0 }
    if totalPages == 0 && totalItems > 0 { totalPages = 1 }

	// Para retornar EventTypes como slice no JSON de resposta:
	type WebhookResponseItem struct {
		models.WebhookConfiguration
		EventTypesList []string `json:"event_types_list"`
	}
	var responseItems []WebhookResponseItem
	for _, wh := range webhooks {
		responseItems = append(responseItems, WebhookResponseItem{
			WebhookConfiguration: wh,
			EventTypesList:       stringToEventTypes(wh.EventTypes),
		})
	}

	response := PaginatedResponse{
		Items:      responseItems,
		TotalItems: totalItems,
		TotalPages: totalPages,
		Page:       page,
		PageSize:   pageSize,
	}
	c.JSON(http.StatusOK, response)
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
	c.JSON(http.StatusOK, newWebhookResponseItem(webhook))
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
	if payload.SecretToken != nil { // Permitir atualização do SecretToken
		webhook.SecretToken = *payload.SecretToken
	}


	if err := db.Save(&webhook).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update webhook configuration: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, newWebhookResponseItem(webhook))
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

// SendTestWebhookHandler sends a test event to a specific webhook.
func SendTestWebhookHandler(c *gin.Context) {
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch webhook configuration"})
		return
	}

	testData := gin.H{
		"message": "This is a test event from Phoenix GRC.",
		"webhook_name": webhook.Name,
	}

	go TriggerWebhooks(targetOrgID, "test_event", testData)

	c.JSON(http.StatusOK, gin.H{"message": "Test event sent successfully"})
}
