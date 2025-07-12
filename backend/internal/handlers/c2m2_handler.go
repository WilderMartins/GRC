package handlers

import (
	"net/http"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ListC2M2DomainsHandler lista todos os domínios C2M2.
func ListC2M2DomainsHandler(c *gin.Context) {
	db := database.GetDB()
	var domains []models.C2M2Domain
	if err := db.Order("code asc").Find(&domains).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list C2M2 domains: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, domains)
}

// ListC2M2PracticesByDomainHandler lista todas as práticas C2M2 para um domínio específico.
func ListC2M2PracticesByDomainHandler(c *gin.Context) {
	domainIDStr := c.Param("domainId")
	domainID, err := uuid.Parse(domainIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid domain ID format"})
		return
	}

	db := database.GetDB()
	var practices []models.C2M2Practice
	// Verificar se o domínio existe primeiro
	var domain models.C2M2Domain
	if err := db.First(&domain, "id = ?", domainID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "C2M2 domain not found"})
		return
	}

	if err := db.Where("domain_id = ?", domainID).Order("code asc").Find(&practices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list C2M2 practices for domain: " + err.Error()})
		return
	}

	if practices == nil {
		practices = []models.C2M2Practice{} // Retornar array vazio em vez de nulo
	}
	c.JSON(http.StatusOK, practices)
}
