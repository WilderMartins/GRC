package handlers

import (
	"net/http"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserDashboardSummaryResponse struct {
	AssignedRisksOpenCount             int64 `json:"assigned_risks_open_count"`
	AssignedVulnerabilitiesOpenCount   int64 `json:"assigned_vulnerabilities_open_count"` // Ou total da org
	PendingApprovalTasksCount        int64 `json:"pending_approval_tasks_count"`
}

// GetUserDashboardSummaryHandler retorna um resumo de dados para o dashboard do usuário autenticado.
func GetUserDashboardSummaryHandler(c *gin.Context) {
	userIDToken, userOk := c.Get("userID")
	orgIDToken, orgOk := c.Get("organizationID")

	if !userOk || !orgOk {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User or Organization ID not found in token"})
		return
	}

	userID := userIDToken.(uuid.UUID)
	orgID := orgIDToken.(uuid.UUID)
	db := database.GetDB()

	var summary UserDashboardSummaryResponse

	// 1. Contagem de Riscos Abertos Atribuídos
	err := db.Model(&models.Risk{}).
		Where("owner_id = ? AND organization_id = ? AND status NOT IN (?, ?)",
			userID, orgID, models.StatusMitigated, models.StatusAccepted).
		Count(&summary.AssignedRisksOpenCount).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count assigned open risks: " + err.Error()})
		return
	}

	// 2. Contagem de Vulnerabilidades Abertas da Organização
	// (Vulnerabilidades não têm OwnerID no modelo atual, então contamos as da organização que não estão corrigidas)
	err = db.Model(&models.Vulnerability{}).
		Where("organization_id = ? AND status <> ?",
			orgID, models.VStatusRemediated).
		Count(&summary.AssignedVulnerabilitiesOpenCount).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count open vulnerabilities: " + err.Error()})
		return
	}

	// 3. Contagem de Tarefas de Aprovação Pendentes para o Usuário
	err = db.Model(&models.ApprovalWorkflow{}).
		Joins("JOIN risks ON risks.id = approval_workflows.risk_id").
		Where("approval_workflows.approver_id = ? AND approval_workflows.status = ? AND risks.organization_id = ?",
			userID, models.ApprovalPending, orgID).
		Count(&summary.PendingApprovalTasksCount).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count pending approval tasks: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}
