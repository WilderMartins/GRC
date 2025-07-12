package handlers

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	phxlog "phoenixgrc/backend/pkg/log" // Importar o logger zap
	"go.uber.org/zap"                 // Importar zap
	"phoenixgrc/backend/internal/notifications"
	"phoenixgrc/backend/internal/riskutils"
	"phoenixgrc/backend/pkg/features"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RiskPayload defines the structure for creating or updating a risk.
type RiskPayload struct {
	Title       string                `json:"title" binding:"required,min=3,max=255"`
	Description string                `json:"description"`
	Category    models.RiskCategory   `json:"category" binding:"omitempty,oneof=tecnologico operacional legal"`
	Impact      models.RiskImpact     `json:"impact" binding:"omitempty,oneof=Baixo Médio Alto Crítico"`
	Probability models.RiskProbability `json:"probability" binding:"omitempty,oneof=Baixo Médio Alto Crítico"`
	Status      models.RiskStatus     `json:"status" binding:"omitempty,oneof=aberto em_andamento mitigado aceito"`
	OwnerID     string                `json:"owner_id"`
}

// CreateRiskHandler handles the creation of a new risk.
func CreateRiskHandler(c *gin.Context) {
	var payload RiskPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}
	db := database.GetDB()
	orgID, _ := c.Get("organizationID")
	userID, _ := c.Get("userID")
	var ownerUUID uuid.UUID
	if payload.OwnerID != "" {
		parsedOwnerID, err := uuid.Parse(payload.OwnerID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid OwnerID format"})
			return
		}
		ownerUUID = parsedOwnerID
	} else {
		ownerUUID = userID.(uuid.UUID)
	}
	risk := models.Risk{
		OrganizationID: orgID.(uuid.UUID),
		Title:          payload.Title,
		Description:    payload.Description,
		Category:       payload.Category,
		Impact:         payload.Impact,
		Probability:    payload.Probability,
		Status:         payload.Status,
		OwnerID:        ownerUUID,
	}
	if risk.Status == "" {
		risk.Status = models.StatusOpen
	}
	risk.RiskLevel = riskutils.CalculateRiskLevel(risk.Impact, risk.Probability)
	if err := db.Create(&risk).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create risk: " + err.Error()})
		return
	}
	go notifications.NotifyRiskEvent(risk.OrganizationID, risk, models.EventTypeRiskCreated)
	if risk.OwnerID != uuid.Nil {
		emailSubject := fmt.Sprintf("Novo Risco Criado: %s", risk.Title)
		emailBody := fmt.Sprintf("Um novo risco foi criado e atribuído a você ou à sua equipe:\n\nTítulo: %s\nDescrição: %s\nImpacto: %s\nProbabilidade: %s\n\nAcesse o Phoenix GRC para mais detalhes.",
			risk.Title, risk.Description, risk.Impact, risk.Probability)
		notifications.NotifyUserByEmail(risk.OwnerID, emailSubject, emailBody)
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
	orgID, _ := c.Get("organizationID")
	db := database.GetDB()
	var risk models.Risk
	if err := db.Preload("Owner").Where("id = ? AND organization_id = ?", riskID, orgID).First(&risk).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Risk not found or not part of your organization"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch risk: " + err.Error()})
		return
	}
	if features.IsEnabled("LOG_DETALHADO_RISCO") {
		phxlog.L.Debug("Detailed risk information requested (feature flag enabled)",
			zap.String("riskID", riskID.String()),
			zap.Any("risk", risk), // zap.Any pode ser verboso; considerar campos específicos
		)
	}
	c.JSON(http.StatusOK, risk)
}

// ListRisksHandler handles fetching all risks for the organization with pagination.
func ListRisksHandler(c *gin.Context) {
	orgID, _ := c.Get("organizationID")
	organizationID := orgID.(uuid.UUID)
	page, pageSize := GetPaginationParams(c)
	db := database.GetDB()
	var risks []models.Risk
	var totalItems int64
	query := db.Model(&models.Risk{}).Where("organization_id = ?", organizationID)
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if impact := c.Query("impact"); impact != "" {
		query = query.Where("impact = ?", impact)
	}
	if probability := c.Query("probability"); probability != "" {
		query = query.Where("probability = ?", probability)
	}
	if category := c.Query("category"); category != "" {
		query = query.Where("category = ?", category)
	}
	if err := query.Count(&totalItems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count risks: " + err.Error()})
		return
	}
	if err := query.Scopes(PaginateScope(page, pageSize)).Preload("Owner").Order("created_at desc").Find(&risks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list risks: " + err.Error()})
		return
	}
	totalPages := totalItems / int64(pageSize)
	if totalItems%int64(pageSize) != 0 {
		totalPages++
	}
	if totalItems == 0 { totalPages = 0 }
	if totalPages == 0 && totalItems > 0 { totalPages = 1 }
	response := PaginatedResponse{
		Items:      risks,
		TotalItems: totalItems,
		TotalPages: totalPages,
		Page:       page,
		PageSize:   pageSize,
	}
	c.JSON(http.StatusOK, response)
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
	orgID, _ := c.Get("organizationID")
	userIDToken, _ := c.Get("userID")
	userRoleToken, _ := c.Get("userRole")
	db := database.GetDB()
	var risk models.Risk
	var originalStatus models.RiskStatus

	// Fetch the risk
	if err := db.Where("id = ? AND organization_id = ?", riskID, orgID).First(&risk).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Risk not found or not part of your organization"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch risk for update: " + err.Error()})
		return
	}

	// Authorization check
	currentUserRole := userRoleToken.(models.UserRole)
	currentUserID := userIDToken.(uuid.UUID)
	isOwner := risk.OwnerID == currentUserID
	isAdmin := currentUserRole == models.RoleAdmin
	isManager := currentUserRole == models.RoleManager

	if !isOwner && !isAdmin && !isManager {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to update this risk"})
		return
	}

	originalStatus = risk.Status
	risk.Title = payload.Title
	risk.Description = payload.Description
	if payload.Category != "" { risk.Category = payload.Category }
	if payload.Impact != "" { risk.Impact = payload.Impact }
	if payload.Probability != "" { risk.Probability = payload.Probability }
	if payload.Status != "" { risk.Status = payload.Status }

	// Handle OwnerID change authorization
	if payload.OwnerID != "" {
		parsedOwnerID, err := uuid.Parse(payload.OwnerID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid OwnerID format for update"})
			return
		}
		if risk.OwnerID != parsedOwnerID { // If owner is being changed
			if isAdmin || isManager { // Only admin/manager can change owner
				risk.OwnerID = parsedOwnerID
			} else {
				c.JSON(http.StatusForbidden, gin.H{"error": "Only Admins or Managers can change the risk owner."})
				return
			}
		}
	}

	if payload.Impact != "" || payload.Probability != "" {
		risk.RiskLevel = riskutils.CalculateRiskLevel(risk.Impact, risk.Probability)
	}

	if err := db.Save(&risk).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update risk: " + err.Error()})
		return
	}

	var updatedRisk models.Risk
	db.Preload("Owner").Where("id = ?", risk.ID).First(&updatedRisk) // Re-fetch to get preloaded owner if changed

	if updatedRisk.Status != originalStatus {
		go notifications.NotifyRiskEvent(updatedRisk.OrganizationID, updatedRisk, models.EventTypeRiskStatusChanged)
		if updatedRisk.OwnerID != uuid.Nil {
			emailSubject := fmt.Sprintf("Status do Risco '%s' Alterado para '%s'", updatedRisk.Title, updatedRisk.Status)
			emailBody := fmt.Sprintf("O status do risco '%s' foi alterado de '%s' para '%s'.\n\nAcesse o Phoenix GRC para mais detalhes.",
				updatedRisk.Title, originalStatus, updatedRisk.Status)
			notifications.NotifyUserByEmail(updatedRisk.OwnerID, emailSubject, emailBody)
		}
	}
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
	orgID, _ := c.Get("organizationID")
	userIDToken, _ := c.Get("userID")
	userRoleToken, _ := c.Get("userRole")
	db := database.GetDB()
	var risk models.Risk

	// Fetch the risk
	if err := db.Where("id = ? AND organization_id = ?", riskID, orgID).First(&risk).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Risk not found or not part of your organization"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch risk for deletion: " + err.Error()})
		return
	}

	// Authorization check
	currentUserRole := userRoleToken.(models.UserRole)
	currentUserID := userIDToken.(uuid.UUID)
	isOwner := risk.OwnerID == currentUserID
	isAdmin := currentUserRole == models.RoleAdmin
	isManager := currentUserRole == models.RoleManager

	if !isOwner && !isAdmin && !isManager {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to delete this risk"})
		return
	}

	// Proceed with deletion
	// Note: Consider implications of deleting a risk, e.g., related approval workflows or audit items.
	// GORM's default behavior for Delete might not cascade unless explicitly configured with constraints
	// or through GORM settings (e.g., Select(clause.Associations) for many2many, or manual cleanup).
	// For now, directly deleting the risk.
	if err := db.Delete(&risk).Error; err != nil { // Changed from db.Delete(&models.Risk{}, riskID) to db.Delete(&risk) to allow GORM hooks on the specific instance if any.
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete risk: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Risk deleted successfully"})
}

// --- Approval Workflow Handlers ---
func SubmitRiskForAcceptanceHandler(c *gin.Context) {
	riskIDStr := c.Param("riskId")
	riskID, err := uuid.Parse(riskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid risk ID format"})
		return
	}
	tokenOrgID, _ := c.Get("organizationID")
	tokenUserID, _ := c.Get("userID")
	tokenUserRole, _ := c.Get("userRole")

	if tokenUserRole.(models.UserRole) != models.RoleAdmin && tokenUserRole.(models.UserRole) != models.RoleManager {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins or managers can submit risks for acceptance"})
		return
	}
	db := database.GetDB()
	var risk models.Risk
	if err := db.Where("id = ? AND organization_id = ?", riskID, tokenOrgID).First(&risk).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Risk not found or not part of your organization"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch risk: " + err.Error()})
		return
	}
	if risk.OwnerID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Risk must have an owner assigned before submitting for acceptance"})
		return
	}
	var existingWorkflow models.ApprovalWorkflow
	err = db.Where("risk_id = ? AND status = ?", riskID, models.ApprovalPending).First(&existingWorkflow).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "An approval workflow for this risk is already pending"})
		return
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check for existing workflows: " + err.Error()})
		return
	}
	approvalWorkflow := models.ApprovalWorkflow{
		RiskID:      riskID,
		RequesterID: tokenUserID.(uuid.UUID),
		ApproverID:  risk.OwnerID,
		Status:      models.ApprovalPending,
	}
	if err := db.Create(&approvalWorkflow).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create approval workflow: " + err.Error()})
		return
	}
	var requesterUser models.User
	var approverUser models.User
	db.First(&requesterUser, "id = ?", approvalWorkflow.RequesterID)
	db.First(&approverUser, "id = ?", approvalWorkflow.ApproverID)
	if approverUser.ID != uuid.Nil && approverUser.IsActive {
		emailSubject := fmt.Sprintf("Ação Requerida: Aprovação de Aceite para o Risco '%s'", risk.Title)
		emailBody := fmt.Sprintf(
			"Olá %s,\n\nO risco '%s' (Descrição: %s) foi submetido para sua aprovação de aceite por %s.\n\nPor favor, acesse o Phoenix GRC para revisar e tomar uma decisão.\n\nDetalhes do Risco:\nImpacto: %s\nProbabilidade: %s\nNível de Risco: %s",
			approverUser.Name, risk.Title, risk.Description, requesterUser.Name,
			risk.Impact, risk.Probability, risk.RiskLevel,
		)
		notifications.NotifyUserByEmail(approverUser.ID, emailSubject, emailBody)
		phxlog.L.Info("Risk submission approval notification sent",
			zap.String("approverEmail", approverUser.Email),
			zap.String("riskTitle", risk.Title),
			zap.String("riskID", risk.ID.String()))
	} else {
		phxlog.L.Warn("Approver not found or inactive for risk submission notification",
			zap.String("approverID", approvalWorkflow.ApproverID.String()),
			zap.String("riskTitle", risk.Title),
			zap.String("riskID", risk.ID.String()))
	}
	c.JSON(http.StatusCreated, approvalWorkflow)
}

type DecisionPayload struct {
	Decision models.ApprovalStatus `json:"decision" binding:"required,oneof=aprovado rejeitado"`
	Comments string                `json:"comments"`
}

func ApproveOrRejectRiskAcceptanceHandler(c *gin.Context) {
	riskIDStr := c.Param("riskId")
	riskID, err := uuid.Parse(riskIDStr)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid risk ID format"}); return }
	approvalIDStr := c.Param("approvalId")
	approvalID, err := uuid.Parse(approvalIDStr)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid approval workflow ID format"}); return }
	var payload DecisionPayload
	if err := c.ShouldBindJSON(&payload); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()}); return }
	tokenUserID, _ := c.Get("userID")
	tokenOrgID, _ := c.Get("organizationID")
	db := database.GetDB()
	var approvalWorkflow models.ApprovalWorkflow
	err = db.Joins("Risk").Where(`"approval_workflows"."id" = ? AND "approval_workflows"."risk_id" = ? AND "Risk"."organization_id" = ?`,
        approvalID, riskID, tokenOrgID).First(&approvalWorkflow).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound { c.JSON(http.StatusNotFound, gin.H{"error": "Approval workflow not found..."}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch approval workflow..."}); return
	}
	if approvalWorkflow.ApproverID != tokenUserID.(uuid.UUID) { c.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized..."}); return }
	if approvalWorkflow.Status != models.ApprovalPending { c.JSON(http.StatusConflict, gin.H{"error": "This approval workflow has already been decided: " + approvalWorkflow.Status}); return }
	tx := db.Begin()
	if tx.Error != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start database transaction"}); return }
	approvalWorkflow.Status = payload.Decision
	approvalWorkflow.Comments = payload.Comments
	if err := tx.Save(&approvalWorkflow).Error; err != nil { tx.Rollback(); c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update approval workflow..."}); return }
	if payload.Decision == models.ApprovalApproved {
		var riskToUpdate models.Risk
		if err := tx.Where("id = ?", approvalWorkflow.RiskID).First(&riskToUpdate).Error; err != nil { tx.Rollback(); c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch risk for status update..."}); return }
		riskToUpdate.Status = models.StatusAccepted
		if err := tx.Save(&riskToUpdate).Error; err != nil { tx.Rollback(); c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update risk status..."}); return }
	}
	if err := tx.Commit().Error; err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"}); return }
	if approvalWorkflow.Status == models.ApprovalApproved {
		var approvedRisk models.Risk
		if err := db.First(&approvedRisk, approvalWorkflow.RiskID).Error; err == nil {
			go notifications.NotifyRiskEvent(approvedRisk.OrganizationID, approvedRisk, models.EventTypeRiskStatusChanged)
			if approvedRisk.OwnerID != uuid.Nil {
				emailSubjectOwner := fmt.Sprintf("Risco '%s' Aceito (Status: %s)", approvedRisk.Title, approvedRisk.Status)
				emailBodyOwner := fmt.Sprintf("O risco '%s' que você aprovou foi atualizado para o status '%s'.\n\nComentários da aprovação: %s\n\nAcesse o Phoenix GRC para mais detalhes.",
					approvedRisk.Title, approvedRisk.Status, approvalWorkflow.Comments)
				notifications.NotifyUserByEmail(approvedRisk.OwnerID, emailSubjectOwner, emailBodyOwner)
			}
			if approvalWorkflow.RequesterID != uuid.Nil && approvalWorkflow.RequesterID != approvedRisk.OwnerID {
				var approverDetails models.User
				if errDb := db.First(&approverDetails, tokenUserID.(uuid.UUID)).Error; errDb == nil {
					emailSubjectRequester := fmt.Sprintf("Sua solicitação de aceite para o Risco '%s' foi Aprovada", approvedRisk.Title)
					emailBodyRequester := fmt.Sprintf("A solicitação de aceite para o risco '%s' foi aprovada por %s.\nO status do risco foi atualizado para '%s'.\n\nComentários: %s\n\nAcesse o Phoenix GRC para mais detalhes.",
						approvedRisk.Title, approverDetails.Name, approvedRisk.Status, approvalWorkflow.Comments)
					notifications.NotifyUserByEmail(approvalWorkflow.RequesterID, emailSubjectRequester, emailBodyRequester)
				} else {
					phxlog.L.Error("Failed to fetch approver details for notification",
						zap.String("approverID", tokenUserID.(uuid.UUID).String()),
						zap.Error(errDb))
					// Fallback notification without approver name
					emailSubjectRequester := fmt.Sprintf("Sua solicitação de aceite para o Risco '%s' foi Aprovada", approvedRisk.Title)
					emailBodyRequester := fmt.Sprintf("A solicitação de aceite para o risco '%s' foi aprovada.\nO status do risco foi atualizado para '%s'.\n\nComentários: %s\n\nAcesse o Phoenix GRC para mais detalhes.",
						approvedRisk.Title, approvedRisk.Status, approvalWorkflow.Comments)
					notifications.NotifyUserByEmail(approvalWorkflow.RequesterID, emailSubjectRequester, emailBodyRequester)
				}
			}
		}
	} else if approvalWorkflow.Status == models.ApprovalRejected {
        if approvalWorkflow.RequesterID != uuid.Nil {
            var rejectedRisk models.Risk
            db.First(&rejectedRisk, approvalWorkflow.RiskID)
            emailSubjectRequester := fmt.Sprintf("Sua solicitação de aceite para o Risco '%s' foi Rejeitada", rejectedRisk.Title)
            emailBodyRequester := fmt.Sprintf("A solicitação de aceite para o risco '%s' foi rejeitada.\n\nComentários: %s\n\nAcesse o Phoenix GRC para mais detalhes e para discutir os próximos passos.",
                rejectedRisk.Title, approvalWorkflow.Comments)
            notifications.NotifyUserByEmail(approvalWorkflow.RequesterID, emailSubjectRequester, emailBodyRequester)
        }
    }
	c.JSON(http.StatusOK, approvalWorkflow)
}

func GetRiskApprovalHistoryHandler(c *gin.Context) {
	riskIDStr := c.Param("riskId")
	riskID, err := uuid.Parse(riskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid risk ID format"})
		return
	}
	tokenOrgID, _ := c.Get("organizationID")
	db := database.GetDB()
	var risk models.Risk
	if err := db.Where("id = ? AND organization_id = ?", riskID, tokenOrgID).First(&risk).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Risk not found or not part of your organization"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify risk: " + err.Error()})
		return
    }
	var approvalHistory []models.ApprovalWorkflow
	var totalItems int64
	page, pageSize := GetPaginationParams(c)
	query := db.Model(&models.ApprovalWorkflow{}).Where("risk_id = ?", riskID)
	if err := query.Count(&totalItems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count approval history: " + err.Error()})
		return
	}
	err = query.Scopes(PaginateScope(page, pageSize)).
		Preload("Requester").Preload("Approver").
		Order("created_at desc").
		Find(&approvalHistory).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch approval history: " + err.Error()})
		return
	}
	totalPages := totalItems / int64(pageSize)
	if totalItems%int64(pageSize) != 0 { totalPages++ }
    if totalItems == 0 { totalPages = 0 }
    if totalPages == 0 && totalItems > 0 { totalPages = 1 }
	response := PaginatedResponse{
		Items:      approvalHistory, TotalItems: totalItems, TotalPages: totalPages, Page: page, PageSize: pageSize,
	}
	c.JSON(http.StatusOK, response)
}

// --- Risk Stakeholder Handlers ---
type UserStakeholderResponse struct { // DTO definido aqui
	ID    uuid.UUID       `json:"id"`
	Name  string          `json:"name"`
	Email string          `json:"email"`
	Role  models.UserRole `json:"role"`
}

type AddStakeholderPayload struct {
	UserID string `json:"user_id" binding:"required"`
}

func AddRiskStakeholderHandler(c *gin.Context) {
	riskIDStr := c.Param("riskId")
	riskID, err := uuid.Parse(riskIDStr)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid risk ID format"}); return }
	var payload AddStakeholderPayload
	if err := c.ShouldBindJSON(&payload); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()}); return }
	stakeholderUserID, err := uuid.Parse(payload.UserID)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UserID format for stakeholder"}); return }
	orgID, _ := c.Get("organizationID")
	organizationID := orgID.(uuid.UUID)
	db := database.GetDB()
	var risk models.Risk
	if err := db.Where("id = ? AND organization_id = ?", riskID, organizationID).First(&risk).Error; err != nil {
		if err == gorm.ErrRecordNotFound { c.JSON(http.StatusNotFound, gin.H{"error": "Risk not found or not part of your organization"}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch risk: " + err.Error()}); return
	}
	var stakeholderUser models.User
	if err := db.Where("id = ? AND organization_id = ?", stakeholderUserID, organizationID).First(&stakeholderUser).Error; err != nil {
		if err == gorm.ErrRecordNotFound { c.JSON(http.StatusNotFound, gin.H{"error": "User to be added as stakeholder not found or not part of your organization"}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user: " + err.Error()}); return
	}
	riskStakeholder := models.RiskStakeholder{RiskID: riskID, UserID: stakeholderUserID}
	result := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&riskStakeholder)
	if result.Error != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add stakeholder to risk: " + result.Error.Error()}); return }
	if result.RowsAffected == 0 { c.JSON(http.StatusOK, gin.H{"message": "Stakeholder association already exists."}); return }
	c.JSON(http.StatusCreated, gin.H{"message": "Stakeholder added successfully"})
}

func RemoveRiskStakeholderHandler(c *gin.Context) {
	riskIDStr := c.Param("riskId")
	riskID, err := uuid.Parse(riskIDStr)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid risk ID format"}); return }
	stakeholderUserIDStr := c.Param("userId")
	stakeholderUserID, err := uuid.Parse(stakeholderUserIDStr)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid stakeholder UserID format"}); return }
	orgID, _ := c.Get("organizationID")
	organizationID := orgID.(uuid.UUID)
	db := database.GetDB()
	var risk models.Risk
	if err := db.Where("id = ? AND organization_id = ?", riskID, organizationID).First(&risk).Error; err != nil {
		if err == gorm.ErrRecordNotFound { c.JSON(http.StatusNotFound, gin.H{"error": "Risk not found or not part of your organization, cannot remove stakeholder."}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify risk before stakeholder removal: " + err.Error()}); return
	}
	result := db.Where("risk_id = ? AND user_id = ?", riskID, stakeholderUserID).Delete(&models.RiskStakeholder{})
	if result.Error != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove stakeholder: " + result.Error.Error()}); return }
	if result.RowsAffected == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "Stakeholder association not found"}); return }
	c.JSON(http.StatusOK, gin.H{"message": "Stakeholder removed successfully"})
}

func ListRiskStakeholdersHandler(c *gin.Context) {
	riskIDStr := c.Param("riskId")
	riskID, err := uuid.Parse(riskIDStr)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid risk ID format"}); return }
	orgID, _ := c.Get("organizationID")
	organizationID := orgID.(uuid.UUID)
	db := database.GetDB()
	var risk models.Risk
	if err := db.Where("id = ? AND organization_id = ?", riskID, organizationID).First(&risk).Error; err != nil {
		if err == gorm.ErrRecordNotFound { c.JSON(http.StatusNotFound, gin.H{"error": "Risk not found or not part of your organization"}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch risk: " + err.Error()}); return
	}
	var users []models.User
	err = db.Table("users").
		Select("users.id, users.name, users.email, users.role, users.organization_id, users.is_active, users.created_at, users.updated_at").
		Joins("JOIN risk_stakeholders rs ON rs.user_id = users.id").
		Where("rs.risk_id = ? AND users.organization_id = ?", riskID, organizationID).
		Find(&users).Error
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list stakeholders: " + err.Error()}); return }

	stakeholderResponses := make([]UserStakeholderResponse, len(users))
	for i, user := range users {
		stakeholderResponses[i] = UserStakeholderResponse{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Role:  user.Role,
		}
	}
	if stakeholderResponses == nil { stakeholderResponses = []UserStakeholderResponse{} }
	c.JSON(http.StatusOK, stakeholderResponses)
}

// --- Bulk Upload Handler ---
type BulkUploadErrorDetail struct {
	LineNumber int      `json:"line_number"`
	Errors     []string `json:"errors"`
}
type BulkUploadRisksResponse struct {
	SuccessfullyImported int                     `json:"successfully_imported"`
	FailedRows           []BulkUploadErrorDetail `json:"failed_rows,omitempty"`
	GeneralError         string                  `json:"general_error,omitempty"`
}
func normalizeHeader(header string) string {
	return strings.ToLower(strings.TrimSpace(header))
}
func isValidEnumValue(value string, validValues map[string]string) string {
	normalizedValue := strings.ToLower(strings.TrimSpace(value))
	return validValues[normalizedValue]
}
func BulkUploadRisksCSVHandler(c *gin.Context) {
	tokenOrgID, _ := c.Get("organizationID")
	organizationID := tokenOrgID.(uuid.UUID)
	tokenUserID, _ := c.Get("userID")
	ownerID := tokenUserID.(uuid.UUID)
	file, err := c.FormFile("file")
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "CSV file not provided..."}); return }
	src, err := file.Open(); if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file..."}); return }
	defer src.Close()
	reader := csv.NewReader(src)
	headers, err := reader.Read()
	if err == io.EOF { c.JSON(http.StatusBadRequest, gin.H{"error": "CSV file is empty"}); return }
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read CSV headers..."}); return }
	headerMap := make(map[string]int)
	for i, h := range headers { headerMap[normalizeHeader(h)] = i }
	requiredHeaders := []string{"title", "impact", "probability"}
	for _, reqHeader := range requiredHeaders {
		if _, ok := headerMap[reqHeader]; !ok { c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Missing required CSV header: %s", reqHeader)}); return }
	}
	validImpacts := map[string]string{"baixo": string(models.ImpactLow), "médio": string(models.ImpactMedium), "medio": string(models.ImpactMedium), "alto": string(models.ImpactHigh), "crítico": string(models.ImpactCritical), "critico": string(models.ImpactCritical)}
	validProbabilities := map[string]string{"baixo": string(models.ProbabilityLow), "médio": string(models.ProbabilityMedium), "medio": string(models.ProbabilityMedium), "alto": string(models.ProbabilityHigh), "crítico": string(models.ProbabilityCritical), "critico": string(models.ProbabilityCritical)}
	validCategories := map[string]string{"tecnologico": string(models.CategoryTechnological), "operacional": string(models.CategoryOperational), "legal": string(models.CategoryLegal)}
	defaultCategory := models.CategoryTechnological
	var risksToCreate []models.Risk
	var failedRows []BulkUploadErrorDetail
	lineNumber := 1
	for {
		lineNumber++
		record, err := reader.Read()
		if err == io.EOF { break }
		if err != nil { failedRows = append(failedRows, BulkUploadErrorDetail{LineNumber: lineNumber, Errors: []string{"Failed to parse CSV row: " + err.Error()}}); continue }
		var rowErrors []string
		var risk models.Risk
		titleIdx, _ := headerMap["title"]
		title := strings.TrimSpace(record[titleIdx])
		if title == "" { rowErrors = append(rowErrors, "title is required") } else if len(title) < 3 || len(title) > 255 { rowErrors = append(rowErrors, "title must be between 3 and 255 characters") }
		risk.Title = title
		if descIdx, ok := headerMap["description"]; ok { risk.Description = strings.TrimSpace(record[descIdx]) }
		risk.Category = defaultCategory
		if catIdx, ok := headerMap["category"]; ok {
			catValue := strings.TrimSpace(record[catIdx])
			if catValue != "" {
				if canonicalCat := isValidEnumValue(catValue, validCategories); canonicalCat != "" { risk.Category = models.RiskCategory(canonicalCat)
				} else { rowErrors = append(rowErrors, fmt.Sprintf("invalid category: '%s'. Valid are: tecnologico, operacional, legal. Using default '%s'.", catValue, defaultCategory)) }
			}
		}
		impactIdx, _ := headerMap["impact"]
		impactValue := strings.TrimSpace(record[impactIdx])
		if impactValue == "" { rowErrors = append(rowErrors, "impact is required")
		} else if canonicalImpact := isValidEnumValue(impactValue, validImpacts); canonicalImpact != "" { risk.Impact = models.RiskImpact(canonicalImpact)
		} else { rowErrors = append(rowErrors, fmt.Sprintf("invalid impact value: '%s'. Valid are: Baixo, Médio, Alto, Crítico.", impactValue)) }
		probIdx, _ := headerMap["probability"]
		probValue := strings.TrimSpace(record[probIdx])
		if probValue == "" { rowErrors = append(rowErrors, "probability is required")
		} else if canonicalProb := isValidEnumValue(probValue, validProbabilities); canonicalProb != "" { risk.Probability = models.RiskProbability(canonicalProb)
		} else { rowErrors = append(rowErrors, fmt.Sprintf("invalid probability value: '%s'. Valid are: Baixo, Médio, Alto, Crítico.", probValue)) }
		if len(rowErrors) > 0 { failedRows = append(failedRows, BulkUploadErrorDetail{LineNumber: lineNumber, Errors: rowErrors}); continue }
		risk.OrganizationID = organizationID
		risk.OwnerID = ownerID
		risk.Status = models.StatusOpen
		risk.RiskLevel = riskutils.CalculateRiskLevel(risk.Impact, risk.Probability) // Calculate risk level for bulk uploaded risks
		risksToCreate = append(risksToCreate, risk)
	}
	if len(risksToCreate) > 0 {
		db := database.GetDB()
		tx := db.Begin()
		if tx.Error != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start DB transaction..."}); return }
		if err := tx.Create(&risksToCreate).Error; err != nil {
			tx.Rollback()
			response := BulkUploadRisksResponse{ SuccessfullyImported: 0, FailedRows: failedRows, GeneralError: "Database error during bulk insert: " + err.Error()}
			if len(failedRows) == 0 && len(risksToCreate) > 0 {
				for i := 0; i < len(risksToCreate); i++ { failedRows = append(failedRows, BulkUploadErrorDetail{LineNumber: i + 2, Errors: []string{"Failed to save to database during batch operation."}}) }
			}
			c.JSON(http.StatusInternalServerError, response)
			return
		}
		if err := tx.Commit().Error; err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error committing bulk insert..."}); return }
	}
	response := BulkUploadRisksResponse{SuccessfullyImported: len(risksToCreate), FailedRows: failedRows}
	if len(failedRows) > 0 && len(risksToCreate) > 0 { c.JSON(http.StatusMultiStatus, response)
	} else if len(failedRows) > 0 && len(risksToCreate) == 0 { c.JSON(http.StatusBadRequest, response)
	} else { c.JSON(http.StatusOK, response) }
}
```
