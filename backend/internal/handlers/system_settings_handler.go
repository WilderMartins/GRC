package handlers

import (
	"net/http"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	phxlog "phoenixgrc/backend/pkg/log"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SystemSettingResponse é a estrutura para retornar configurações ao frontend.
// Importante: este DTO nunca expõe o valor criptografado.
type SystemSettingResponse struct {
	Key         string `json:"key"`
	Value       string `json:"value"` // O valor descriptografado
	Description string `json:"description"`
	IsEncrypted bool   `json:"is_encrypted"`
}

// ListSystemSettingsHandler lista todas as configurações do sistema que podem ser expostas na UI.
func ListSystemSettingsHandler(c *gin.Context) {
	log := phxlog.L.Named("ListSystemSettingsHandler")
	db := database.GetDB()

	var settings []models.SystemSetting
	if err := db.Where("exposed_to_ui = ?", true).Find(&settings).Error; err != nil {
		log.Error("Failed to retrieve system settings", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve system settings"})
		return
	}

	response := make([]SystemSettingResponse, len(settings))
	for i, s := range settings {
		decryptedValue, err := s.GetDecryptedValue()
		if err != nil {
			log.Error("Failed to decrypt setting value", zap.String("key", s.Key), zap.Error(err))
			// Não retorne um erro, apenas retorne o valor como uma string vazia ou um placeholder
			decryptedValue = "******" // Ou ""
		}
		response[i] = SystemSettingResponse{
			Key:         s.Key,
			Value:       decryptedValue,
			Description: s.Description,
			IsEncrypted: s.IsEncrypted,
		}
	}

	c.JSON(http.StatusOK, response)
}

// UpdateSystemSettingsPayload define a estrutura para a atualização em massa de configurações.
type UpdateSystemSettingsPayload struct {
	Settings []struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value"` // O valor pode ser vazio
	} `json:"settings" binding:"required,dive"`
}

// UpdateSystemSettingsHandler atualiza uma ou mais configurações do sistema.
func UpdateSystemSettingsHandler(c *gin.Context) {
	log := phxlog.L.Named("UpdateSystemSettingsHandler")
	db := database.GetDB()

	var payload UpdateSystemSettingsPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Inicia uma transação para garantir que todas as atualizações sejam bem-sucedidas ou nenhuma.
	tx := db.Begin()
	if tx.Error != nil {
		log.Error("Failed to begin transaction", zap.Error(tx.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start database transaction"})
		return
	}

	for _, settingUpdate := range payload.Settings {
		var setting models.SystemSetting
		// Encontra a configuração pela chave e verifica se ela pode ser alterada pela UI.
		if err := tx.Where("key = ? AND exposed_to_ui = ?", settingUpdate.Key, true).First(&setting).Error; err != nil {
			tx.Rollback()
			log.Warn("Attempted to update non-existent or non-UI-exposed setting", zap.String("key", settingUpdate.Key))
			c.JSON(http.StatusNotFound, gin.H{"error": "Setting not found or not updatable: " + settingUpdate.Key})
			return
		}

		// Atualiza o valor. O hook BeforeSave cuidará da criptografia.
		setting.Value = settingUpdate.Value
		if err := tx.Save(&setting).Error; err != nil {
			tx.Rollback()
			log.Error("Failed to save setting", zap.String("key", setting.Key), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update setting: " + setting.Key})
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		log.Error("Failed to commit transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit updates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "System settings updated successfully"})
}
