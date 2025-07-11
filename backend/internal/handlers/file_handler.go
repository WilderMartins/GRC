package handlers

import (
	"net/http"
	"phoenixgrc/backend/internal/filestorage"
	"strconv"

	"github.com/gin-gonic/gin"
)

const defaultSignedURLDurationMinutes = 15 // Duração padrão da URL assinada

// GetSignedURLForObjectHandler gera uma URL assinada para um objeto de arquivo.
// Query param: ?objectKey=your/object/key.txt
// Query param: ?durationMinutes=30 (opcional, default 15)
func GetSignedURLForObjectHandler(c *gin.Context) {
	objectKey := c.Query("objectKey")
	if objectKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "objectKey query parameter is required"})
		return
	}

	durationStr := c.Query("durationMinutes")
	durationMinutes := defaultSignedURLDurationMinutes
	if durationStr != "" {
		if val, err := strconv.Atoi(durationStr); err == nil && val > 0 && val <= 60*7 { // Max 7 dias (AWS S3 V4 max)
			durationMinutes = val
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid durationMinutes, must be a positive integer (max 10080 for 7 days)."})
			return
		}
	}

	if filestorage.DefaultFileStorageProvider == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "File storage provider not configured"})
		return
	}

	signedURL, err := filestorage.DefaultFileStorageProvider.GetSignedURL(c.Request.Context(), objectKey, durationMinutes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate signed URL: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"signed_url": signedURL})
}
