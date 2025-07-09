package handlers

import (
	"fmt" // Adicionado
	"net/http"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	"phoenixgrc/backend/internal/riskutils" // Import riskutils package
	"phoenixgrc/backend/pkg/features"       // Import features package

	"phoenixgrc/backend/internal/notifications" // Import notifications package
	"strings"                                   // Para CSV
	"encoding/csv"                              // Para CSV
	"io"                                        // Para CSV
	"log"                                     // Para logs de notificação

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause" // Needed for OnConflict
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

	// Calculate Risk Level
	risk.RiskLevel = riskutils.CalculateRiskLevel(risk.Impact, risk.Probability)

	if err := db.Create(&risk).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create risk: " + err.Error()})
		return
	}

	// Disparar notificação de criação de risco
	organizationID := orgID.(uuid.UUID) // Corrigido: assertion de tipo e atribuição
	go notifications.NotifyRiskEvent(organizationID, risk, models.EventTypeRiskCreated)
	// Simular email para o proprietário do risco
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

	if features.IsEnabled("LOG_DETALHADO_RISCO") {
		log.Printf("FEATURE_LOG_DETALHADO_RISCO: Detalhes do risco %s solicitados: %+v", riskID, risk)
	}

	c.JSON(http.StatusOK, risk)
}

// ListRisksHandler handles fetching all risks for the organization with pagination.
func ListRisksHandler(c *gin.Context) {
	orgID, exists := c.Get("organizationID")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Organization ID not found in token"})
		return
	}
	organizationID := orgID.(uuid.UUID)

	page, pageSize := GetPaginationParams(c)

	db := database.GetDB()
	var risks []models.Risk
	var totalItems int64

	// Contar o total de itens para a organização antes de aplicar a paginação
	query := db.Model(&models.Risk{}).Where("organization_id = ?", organizationID)

	// Aplicar filtros
	if status := c.Query("status"); status != "" {
		// Validar se o status é um valor permitido para models.RiskStatus
		// Para simplificar, assumimos que o frontend envia valores válidos.
		// Uma validação mais robusta verificaria contra os valores do ENUM.
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
	// TODO: Adicionar filtro por `title` (usando LIKE) ou `owner_id` se necessário.


	if err := query.Count(&totalItems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count risks: " + err.Error()})
		return
	}

	// Aplicar escopo de paginação e buscar os itens
	if err := query.Scopes(PaginateScope(page, pageSize)).Preload("Owner").Order("created_at desc").Find(&risks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list risks: " + err.Error()})
		return
	}

	totalPages := totalItems / int64(pageSize)
	if totalItems%int64(pageSize) != 0 {
		totalPages++
	}
	if totalItems == 0 { // Evitar totalPages = 1 se não houver itens
		totalPages = 0
	}
	if totalPages == 0 && totalItems > 0 { // Caso onde totalItems < pageSize
		totalPages = 1
	}


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

	orgID, exists := c.Get("organizationID")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Organization ID not found in token"})
		return
	}
	userID, tokenUserExists := c.Get("userID") // To check if updater is owner or admin if needed
	userRole, tokenRoleExists := c.Get("userRole")


	db := database.GetDB()
	var risk models.Risk
	var originalStatus models.RiskStatus
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


	originalStatus = risk.Status // Store original status before update

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

	// Recalculate Risk Level if impact or probability changed
	if payload.Impact != "" || payload.Probability != "" {
		risk.RiskLevel = riskutils.CalculateRiskLevel(risk.Impact, risk.Probability)
	}

	if err := db.Save(&risk).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update risk: " + err.Error()})
		return
	}

	// Fetch the updated risk with preloads to return
	var updatedRisk models.Risk
	db.Preload("Owner").Where("id = ?", risk.ID).First(&updatedRisk) // Use risk.ID as updatedRisk is not populated yet

	// Check if status changed to trigger notification
	if updatedRisk.Status != originalStatus {
		go notifications.NotifyRiskEvent(updatedRisk.OrganizationID, updatedRisk, models.EventTypeRiskStatusChanged)
		// Simular email para o proprietário do risco sobre a mudança de status
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
	var approverUser models.User // Approver is the risk owner
	var requesterUser models.User

	db.First(&requesterUser, "id = ?", approvalWorkflow.RequesterID) // Fetch requester details for the email
	db.First(&approverUser, "id = ?", approvalWorkflow.ApproverID)   // Fetch approver details for the email

	// Notify the approver (risk owner) by email
	if approverUser.ID != uuid.Nil && approverUser.IsActive {
		emailSubject := fmt.Sprintf("Ação Requerida: Aprovação de Aceite para o Risco '%s'", risk.Title)
		emailBody := fmt.Sprintf(
			"Olá %s,\n\nO risco '%s' (Descrição: %s) foi submetido para sua aprovação de aceite por %s.\n\nPor favor, acesse o Phoenix GRC para revisar e tomar uma decisão.\n\nDetalhes do Risco:\nImpacto: %s\nProbabilidade: %s\nNível de Risco: %s",
			approverUser.Name,
			risk.Title,
			risk.Description,
			requesterUser.Name, // Assume requesterUser was fetched successfully
			risk.Impact,
			risk.Probability,
			risk.RiskLevel,
		)
		notifications.NotifyUserByEmail(approverUser.ID, emailSubject, emailBody)
		log.Printf("Notificação por email enviada para o aprovador %s sobre submissão de risco '%s'", approverUser.Email, risk.Title)
	} else {
		log.Printf("Aprovador (ID: %s) não encontrado ou inativo, notificação por email não enviada para submissão do risco '%s'.", approvalWorkflow.ApproverID, risk.Title)
	}

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

	// Se o risco foi aceito, seu status mudou para "aceito", então disparamos a notificação de mudança de status.
	if approvalWorkflow.Status == models.ApprovalApproved {
		var approvedRisk models.Risk
		// Precisamos buscar o risco novamente para ter todos os campos para a notificação
		if err := db.First(&approvedRisk, approvalWorkflow.RiskID).Error; err == nil {
			go notifications.NotifyRiskEvent(approvedRisk.OrganizationID, approvedRisk, models.EventTypeRiskStatusChanged)
			// Simular email para o proprietário do risco (que é o aprovador aqui) e para o requisitante
			if approvedRisk.OwnerID != uuid.Nil { // Notificar o Owner (Aprovador)
				emailSubjectOwner := fmt.Sprintf("Risco '%s' Aceito (Status: %s)", approvedRisk.Title, approvedRisk.Status)
				emailBodyOwner := fmt.Sprintf("O risco '%s' que você aprovou foi atualizado para o status '%s'.\n\nComentários da aprovação: %s\n\nAcesse o Phoenix GRC para mais detalhes.",
					approvedRisk.Title, approvedRisk.Status, approvalWorkflow.Comments)
				notifications.NotifyUserByEmail(approvedRisk.OwnerID, emailSubjectOwner, emailBodyOwner)
			}
			if approvalWorkflow.RequesterID != uuid.Nil && approvalWorkflow.RequesterID != approvedRisk.OwnerID { // Notificar o Requisitante, se diferente do aprovador
				var approverDetails models.User
				if errDb := db.First(&approverDetails, tokenUserID.(uuid.UUID)).Error; errDb == nil {
					emailSubjectRequester := fmt.Sprintf("Sua solicitação de aceite para o Risco '%s' foi Aprovada", approvedRisk.Title)
					emailBodyRequester := fmt.Sprintf("A solicitação de aceite para o risco '%s' foi aprovada por %s.\nO status do risco foi atualizado para '%s'.\n\nComentários: %s\n\nAcesse o Phoenix GRC para mais detalhes.",
						approvedRisk.Title, approverDetails.Name, approvedRisk.Status, approvalWorkflow.Comments)
					notifications.NotifyUserByEmail(approvalWorkflow.RequesterID, emailSubjectRequester, emailBodyRequester)
				} else {
					log.Printf("Erro ao buscar detalhes do aprovador %s para notificação: %v", tokenUserID.(uuid.UUID), errDb)
					// Fallback: Enviar notificação sem o nome do aprovador se a busca falhar
					emailSubjectRequester := fmt.Sprintf("Sua solicitação de aceite para o Risco '%s' foi Aprovada", approvedRisk.Title)
					emailBodyRequester := fmt.Sprintf("A solicitação de aceite para o risco '%s' foi aprovada.\nO status do risco foi atualizado para '%s'.\n\nComentários: %s\n\nAcesse o Phoenix GRC para mais detalhes.",
						approvedRisk.Title, approvedRisk.Status, approvalWorkflow.Comments)
					notifications.NotifyUserByEmail(approvalWorkflow.RequesterID, emailSubjectRequester, emailBodyRequester)
				}
			}
		} else {
			log.Printf("Erro ao buscar risco para notificação de status alterado após aprovação: %v", err)
		}
	} else if approvalWorkflow.Status == models.ApprovalRejected {
        // Notificar o Requisitante sobre a rejeição
        if approvalWorkflow.RequesterID != uuid.Nil {
            var rejectedRisk models.Risk
            db.First(&rejectedRisk, approvalWorkflow.RiskID) // Pegar título do risco
            emailSubjectRequester := fmt.Sprintf("Sua solicitação de aceite para o Risco '%s' foi Rejeitada", rejectedRisk.Title)
            emailBodyRequester := fmt.Sprintf("A solicitação de aceite para o risco '%s' foi rejeitada.\n\nComentários: %s\n\nAcesse o Phoenix GRC para mais detalhes e para discutir os próximos passos.",
                rejectedRisk.Title, approvalWorkflow.Comments)
            notifications.NotifyUserByEmail(approvalWorkflow.RequesterID, emailSubjectRequester, emailBodyRequester)
        }
    }

	c.JSON(http.StatusOK, approvalWorkflow)
}

// --- Risk Stakeholder Handlers ---

type AddStakeholderPayload struct {
	UserID string `json:"user_id" binding:"required"`
}

// AddRiskStakeholderHandler adds a user as a stakeholder to a risk.
func AddRiskStakeholderHandler(c *gin.Context) {
	riskIDStr := c.Param("riskId")
	riskID, err := uuid.Parse(riskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid risk ID format"})
		return
	}

	var payload AddStakeholderPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	stakeholderUserID, err := uuid.Parse(payload.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UserID format for stakeholder"})
		return
	}

	orgID, exists := c.Get("organizationID")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Organization ID not found in token"})
		return
	}
	organizationID := orgID.(uuid.UUID)

	db := database.GetDB()

	// 1. Verify the risk exists and belongs to the organization
	var risk models.Risk
	if err := db.Where("id = ? AND organization_id = ?", riskID, organizationID).First(&risk).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Risk not found or not part of your organization"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch risk: " + err.Error()})
		return
	}

	// 2. Verify the user to be added as stakeholder exists and belongs to the same organization
	var stakeholderUser models.User
	if err := db.Where("id = ? AND organization_id = ?", stakeholderUserID, organizationID).First(&stakeholderUser).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User to be added as stakeholder not found or not part of your organization"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user: " + err.Error()})
		return
	}

	// 3. Create the RiskStakeholder association
	riskStakeholder := models.RiskStakeholder{
		RiskID: riskID,
		UserID: stakeholderUserID,
	}

	// Use OnConflict to avoid duplicate errors if the association already exists
	// GORM's Create will return an error if the primary key (RiskID, UserID) already exists,
	// unless OnConflict is used.
	result := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&riskStakeholder)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add stakeholder to risk: " + result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
         // This means the stakeholder association already existed.
         // Return 200 OK or a specific message instead of 201 Created.
        c.JSON(http.StatusOK, gin.H{"message": "Stakeholder association already exists or was successfully processed."})
        return
    }

	c.JSON(http.StatusCreated, gin.H{"message": "Stakeholder added successfully"})
}

// RemoveRiskStakeholderHandler removes a user as a stakeholder from a risk.
func RemoveRiskStakeholderHandler(c *gin.Context) {
	riskIDStr := c.Param("riskId")
	riskID, err := uuid.Parse(riskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid risk ID format"})
		return
	}

	stakeholderUserIDStr := c.Param("userId") // Matches the route param {userId}
	stakeholderUserID, err := uuid.Parse(stakeholderUserIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid stakeholder UserID format"})
		return
	}

	orgID, exists := c.Get("organizationID")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Organization ID not found in token"})
		return
	}
	organizationID := orgID.(uuid.UUID) // For verification

	db := database.GetDB()

	// Optional: Verify the risk exists and belongs to the organization to ensure context
	var risk models.Risk
	if err := db.Where("id = ? AND organization_id = ?", riskID, organizationID).First(&risk).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// If the risk doesn't exist in the org, the stakeholder link shouldn't either or is irrelevant.
			// Depending on strictness, could return 404 here.
			// For now, we'll allow the delete attempt on the join table directly.
		}
		// Log other errors but don't necessarily block if the main goal is to delete the join entry.
		log.Printf("Warning/Error fetching risk context for stakeholder removal: %v", err)
	}


	// Delete the RiskStakeholder association
	// The where clause must ensure that the risk in question belongs to the user's organization.
	// This is implicitly handled if we only allow deletion based on risk_id that the user can access.
	// However, an explicit check on the risk's organization_id before deleting the stakeholder is safer.
	// If risk object is fetched as above, we already have this check.
	// If not, we'd need a subquery or join to verify organization_id of the risk.
	// Given we fetched `risk` above (and it belongs to `organizationID`),
	// we can be sure that `riskID` is valid for this organization.

	result := db.Where("risk_id = ? AND user_id = ?", riskID, stakeholderUserID).Delete(&models.RiskStakeholder{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove stakeholder: " + result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Stakeholder association not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Stakeholder removed successfully"})
}

// ListRiskStakeholdersHandler lists all stakeholders for a specific risk.
// Returns a list of User objects (or a subset of their fields).
func ListRiskStakeholdersHandler(c *gin.Context) {
	riskIDStr := c.Param("riskId")
	riskID, err := uuid.Parse(riskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid risk ID format"})
		return
	}

	orgID, exists := c.Get("organizationID")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Organization ID not found in token"})
		return
	}
	organizationID := orgID.(uuid.UUID)

	db := database.GetDB()

	// 1. Verify the risk exists and belongs to the organization
	var risk models.Risk
	if err := db.Where("id = ? AND organization_id = ?", riskID, organizationID).First(&risk).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Risk not found or not part of your organization"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch risk: " + err.Error()})
		return
	}

	var stakeholders []models.User
	// Query users who are listed in RiskStakeholder for the given riskID
	// and also belong to the same organization.
	err = db.Joins("JOIN risk_stakeholders rs ON rs.user_id = users.id").
		Where("rs.risk_id = ? AND users.organization_id = ?", riskID, organizationID).
		Select("users.id, users.name, users.email, users.role"). // Select specific fields
		Find(&stakeholders).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list stakeholders: " + err.Error()})
		return
	}

	// If we want to return the full User model but without PasswordHash,
    // we would need to map to a different struct or set PasswordHash to empty.
    // For now, Select specific fields is safer. If the full User model (minus password) is needed,
    // the response struct should be adjusted.

	c.JSON(http.StatusOK, stakeholders) // stakeholders will only contain the selected fields
}

// --- Bulk Upload Handler ---

// BulkUploadErrorDetail provides details about an error in a specific row during bulk upload.
type BulkUploadErrorDetail struct {
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
	var totalItems int64

	page, pageSize := GetPaginationParams(c)

	query := db.Model(&models.ApprovalWorkflow{}).Where("risk_id = ?", riskID)

	if err := query.Count(&totalItems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count approval history: " + err.Error()})
		return
	}

	err = query.Scopes(PaginateScope(page, pageSize)).
		Preload("Requester").Preload("Approver"). // Preload user details
		Order("created_at desc").
		Find(&approvalHistory).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch approval history: " + err.Error()})
		return
	}

	totalPages := totalItems / int64(pageSize)
	if totalItems%int64(pageSize) != 0 {
		totalPages++
	}
    if totalItems == 0 { totalPages = 0 }
    if totalPages == 0 && totalItems > 0 { totalPages = 1 }

	response := PaginatedResponse{
		Items:      approvalHistory,
		TotalItems: totalItems,
		TotalPages: totalPages,
		Page:       page,
		PageSize:   pageSize,
	}
	c.JSON(http.StatusOK, response)
}

// --- Risk Stakeholder Handlers ---

type AddStakeholderPayload struct {
	UserID string `json:"user_id" binding:"required"`
}

// AddRiskStakeholderHandler adds a user as a stakeholder to a risk.
func AddRiskStakeholderHandler(c *gin.Context) {
	riskIDStr := c.Param("riskId")
	riskID, err := uuid.Parse(riskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid risk ID format"})
		return
	}

	var payload AddStakeholderPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	stakeholderUserID, err := uuid.Parse(payload.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UserID format for stakeholder"})
		return
	}

	orgID, exists := c.Get("organizationID")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Organization ID not found in token"})
		return
	}
	organizationID := orgID.(uuid.UUID)

	db := database.GetDB()

	// 1. Verify the risk exists and belongs to the organization
	var risk models.Risk
	if err := db.Where("id = ? AND organization_id = ?", riskID, organizationID).First(&risk).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Risk not found or not part of your organization"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch risk: " + err.Error()})
		return
	}

	// 2. Verify the user to be added as stakeholder exists and belongs to the same organization
	var stakeholderUser models.User
	if err := db.Where("id = ? AND organization_id = ?", stakeholderUserID, organizationID).First(&stakeholderUser).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User to be added as stakeholder not found or not part of your organization"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user: " + err.Error()})
		return
	}

	// 3. Create the RiskStakeholder association
	riskStakeholder := models.RiskStakeholder{
		RiskID: riskID,
		UserID: stakeholderUserID,
	}

	// Use OnConflict to avoid duplicate errors if the association already exists
	// GORM's Create will return an error if the primary key (RiskID, UserID) already exists,
	// unless OnConflict is used.
	result := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&riskStakeholder)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add stakeholder to risk: " + result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
         // This means the stakeholder association already existed.
         // Return 200 OK or a specific message instead of 201 Created.
        c.JSON(http.StatusOK, gin.H{"message": "Stakeholder association already exists."}) // Specific message
        return
    }

	c.JSON(http.StatusCreated, gin.H{"message": "Stakeholder added successfully"})
}

// RemoveRiskStakeholderHandler removes a user as a stakeholder from a risk.
func RemoveRiskStakeholderHandler(c *gin.Context) {
	riskIDStr := c.Param("riskId")
	riskID, err := uuid.Parse(riskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid risk ID format"})
		return
	}

	stakeholderUserIDStr := c.Param("userId") // Matches the route param {userId}
	stakeholderUserID, err := uuid.Parse(stakeholderUserIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid stakeholder UserID format"})
		return
	}

	orgID, exists := c.Get("organizationID")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Organization ID not found in token"})
		return
	}
	organizationID := orgID.(uuid.UUID)

	db := database.GetDB()

	// Verify the risk (and thus its organization_id) before allowing stakeholder removal.
	// This ensures that the operation is scoped to the user's organization.
	var risk models.Risk
	if err := db.Where("id = ? AND organization_id = ?", riskID, organizationID).First(&risk).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Risk not found or not part of your organization, cannot remove stakeholder."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify risk before stakeholder removal: " + err.Error()})
		return
	}

	result := db.Where("risk_id = ? AND user_id = ?", riskID, stakeholderUserID).Delete(&models.RiskStakeholder{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove stakeholder: " + result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Stakeholder association not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Stakeholder removed successfully"})
}

// ListRiskStakeholdersHandler lists all stakeholders for a specific risk.
// Returns a list of User objects (or a subset of their fields).
func ListRiskStakeholdersHandler(c *gin.Context) {
	riskIDStr := c.Param("riskId")
	riskID, err := uuid.Parse(riskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid risk ID format"})
		return
	}

	orgID, exists := c.Get("organizationID")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Organization ID not found in token"})
		return
	}
	organizationID := orgID.(uuid.UUID)

	db := database.GetDB()

	// 1. Verify the risk exists and belongs to the organization
	var risk models.Risk
	if err := db.Where("id = ? AND organization_id = ?", riskID, organizationID).First(&risk).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Risk not found or not part of your organization"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch risk: " + err.Error()})
		return
	}

	var stakeholders []models.UserResponse // Use UserResponse to omit PasswordHash

	err = db.Table("users").
		Select("users.id, users.name, users.email, users.role, users.organization_id, users.is_active, users.created_at, users.updated_at").
		Joins("JOIN risk_stakeholders rs ON rs.user_id = users.id").
		Where("rs.risk_id = ? AND users.organization_id = ?", riskID, organizationID).
		Find(&stakeholders).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list stakeholders: " + err.Error()})
		return
	}

	if stakeholders == nil { // Ensure we return an empty list instead of null if no stakeholders
        stakeholders = []models.UserResponse{}
    }

	c.JSON(http.StatusOK, stakeholders)
}


// --- Bulk Upload Handler ---

// BulkUploadErrorDetail provides details about an error in a specific row during bulk upload.
type BulkUploadErrorDetail struct {
	LineNumber int      `json:"line_number"`
	Errors     []string `json:"errors"`
}

// BulkUploadRisksResponse defines the response structure for bulk risk upload.
type BulkUploadRisksResponse struct {
	SuccessfullyImported int                     `json:"successfully_imported"`
	FailedRows           []BulkUploadErrorDetail `json:"failed_rows,omitempty"`
	GeneralError         string                  `json:"general_error,omitempty"`
}

// normalizeHeader trims space and converts to lower case for case-insensitive matching.
func normalizeHeader(header string) string {
	return strings.ToLower(strings.TrimSpace(header))
}

// isValidEnumValue checks if a value is part of a predefined list of valid enum values (case-insensitive).
// Returns the canonical value if valid, or empty string if not.
func isValidEnumValue(value string, validValues map[string]string) string {
	normalizedValue := strings.ToLower(strings.TrimSpace(value))
	return validValues[normalizedValue]
}

// BulkUploadRisksCSVHandler handles the bulk upload of risks via CSV file.
func BulkUploadRisksCSVHandler(c *gin.Context) {
	tokenOrgID, orgExists := c.Get("organizationID")
	if !orgExists {
		c.JSON(http.StatusForbidden, gin.H{"error": "Organization ID not found in token"})
		return
	}
	organizationID := tokenOrgID.(uuid.UUID)

	tokenUserID, userExists := c.Get("userID")
	if !userExists {
		c.JSON(http.StatusForbidden, gin.H{"error": "User ID not found in token"})
		return
	}
	ownerID := tokenUserID.(uuid.UUID)

	file, err := c.FormFile("file") // "file" is the name of the form field
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV file not provided or invalid form field name: " + err.Error()})
		return
	}

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open uploaded file: " + err.Error()})
		return
	}
	defer src.Close()

	reader := csv.NewReader(src)
	headers, err := reader.Read() // Read header row
	if err == io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV file is empty"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read CSV headers: " + err.Error()})
		return
	}

	// Normalize headers and map them to indices
	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[normalizeHeader(h)] = i
	}

	// Validate required headers
	requiredHeaders := []string{"title", "impact", "probability"}
	for _, reqHeader := range requiredHeaders {
		if _, ok := headerMap[reqHeader]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Missing required CSV header: %s", reqHeader)})
			return
		}
	}

	// Define valid enum values for case-insensitive matching and getting canonical form
	// Store canonical values (as defined in models)
	validImpacts := map[string]string{
		"baixo":   string(models.ImpactLow), "médio": string(models.ImpactMedium), "medio": string(models.ImpactMedium), // Allow "medio"
		"alto": string(models.ImpactHigh), "crítico": string(models.ImpactCritical), "critico": string(models.ImpactCritical), // Allow "critico"
	}
	validProbabilities := map[string]string{
		"baixo":   string(models.ProbabilityLow), "médio": string(models.ProbabilityMedium), "medio": string(models.ProbabilityMedium),
		"alto": string(models.ProbabilityHigh), "crítico": string(models.ProbabilityCritical), "critico": string(models.ProbabilityCritical),
	}
	validCategories := map[string]string{
		"tecnologico": string(models.CategoryTechnological), "operacional": string(models.CategoryOperational),
		"legal": string(models.CategoryLegal),
	}
	defaultCategory := models.CategoryTechnological // Default if not provided or invalid

	var risksToCreate []models.Risk
	var failedRows []BulkUploadErrorDetail
	lineNumber := 1 // Header is line 1, data starts at line 2

	for {
		lineNumber++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			failedRows = append(failedRows, BulkUploadErrorDetail{LineNumber: lineNumber, Errors: []string{"Failed to parse CSV row: " + err.Error()}})
			continue
		}

		var rowErrors []string
		var risk models.Risk

		// Title
		titleIdx, titleOk := headerMap["title"]
		if !titleOk { /* Should have been caught by header check, but good practice */ }
		title := strings.TrimSpace(record[titleIdx])
		if title == "" {
			rowErrors = append(rowErrors, "title is required")
		} else if len(title) < 3 || len(title) > 255 {
			rowErrors = append(rowErrors, "title must be between 3 and 255 characters")
		}
		risk.Title = title

		// Description (optional)
		if descIdx, ok := headerMap["description"]; ok {
			risk.Description = strings.TrimSpace(record[descIdx])
		}

		// Category (optional, with default)
		risk.Category = defaultCategory
		if catIdx, ok := headerMap["category"]; ok {
			catValue := strings.TrimSpace(record[catIdx])
			if catValue != "" {
				if canonicalCat := isValidEnumValue(catValue, validCategories); canonicalCat != "" {
					risk.Category = models.RiskCategory(canonicalCat)
				} else {
					rowErrors = append(rowErrors, fmt.Sprintf("invalid category: '%s'. Valid are: tecnologico, operacional, legal. Using default '%s'.", catValue, defaultCategory))
                    // Still using default, not strictly an error that stops import of row if default is acceptable.
                    // If strict, this should be a hard error. For now, it's a soft warning and uses default.
				}
			}
		}

		// Impact (required)
		impactIdx, impactOk := headerMap["impact"]
		if !impactOk { /* Should have been caught */ }
		impactValue := strings.TrimSpace(record[impactIdx])
		if impactValue == "" {
			rowErrors = append(rowErrors, "impact is required")
		} else if canonicalImpact := isValidEnumValue(impactValue, validImpacts); canonicalImpact != "" {
			risk.Impact = models.RiskImpact(canonicalImpact)
		} else {
			rowErrors = append(rowErrors, fmt.Sprintf("invalid impact value: '%s'. Valid are: Baixo, Médio, Alto, Crítico.", impactValue))
		}

		// Probability (required)
		probIdx, probOk := headerMap["probability"]
		if !probOk { /* Should have been caught */ }
		probValue := strings.TrimSpace(record[probIdx])
		if probValue == "" {
			rowErrors = append(rowErrors, "probability is required")
		} else if canonicalProb := isValidEnumValue(probValue, validProbabilities); canonicalProb != "" {
			risk.Probability = models.RiskProbability(canonicalProb)
		} else {
			rowErrors = append(rowErrors, fmt.Sprintf("invalid probability value: '%s'. Valid are: Baixo, Médio, Alto, Crítico.", probValue))
		}


		if len(rowErrors) > 0 {
			failedRows = append(failedRows, BulkUploadErrorDetail{LineNumber: lineNumber, Errors: rowErrors})
			continue // Skip this row
		}

		// Set auto-generated fields
		risk.OrganizationID = organizationID
		risk.OwnerID = ownerID
		risk.Status = models.StatusOpen // Default status

		risksToCreate = append(risksToCreate, risk)
	}

	if len(risksToCreate) > 0 {
		db := database.GetDB()
		// GORM's CreateInBatches might be useful for very large CSVs, but simple Create works for moderate sizes.
		// Using a transaction for atomicity.
		tx := db.Begin()
		if tx.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start database transaction for bulk import"})
			return
		}
		if err := tx.Create(&risksToCreate).Error; err != nil {
			tx.Rollback()
			// Add a general error if DB create fails for the batch
			response := BulkUploadRisksResponse{
				SuccessfullyImported: 0,
				FailedRows:           failedRows, // Include parsing/validation errors found so far
				GeneralError:         "Database error during bulk insert: " + err.Error(),
			}
			// It's possible some rows were valid but the batch insert failed.
			// Add all initially valid rows to failedRows with a generic DB error message.
			if len(failedRows) == 0 && len(risksToCreate) > 0 { // If all rows were valid but batch failed
				for i := 0; i < len(risksToCreate); i++ { // Line numbers would need more careful tracking for this case
					failedRows = append(failedRows, BulkUploadErrorDetail{LineNumber: i + 2, Errors: []string{"Failed to save to database during batch operation."}})
				}
			}
			c.JSON(http.StatusInternalServerError, response)
			return
		}
		if err := tx.Commit().Error; err != nil {
            // Rollback should have happened automatically on commit error with some DBs, but explicit is safer.
            // tx.Rollback() // Not strictly needed if commit fails, GORM might handle.
			response := BulkUploadRisksResponse{
				SuccessfullyImported: 0,
				FailedRows:           failedRows,
				GeneralError:         "Database error committing bulk insert: " + err.Error(),
			}
			c.JSON(http.StatusInternalServerError, response)
			return
		}
	}

	response := BulkUploadRisksResponse{
		SuccessfullyImported: len(risksToCreate),
		FailedRows:           failedRows,
	}

	if len(failedRows) > 0 && len(risksToCreate) > 0 {
		c.JSON(http.StatusMultiStatus, response) // Some succeeded, some failed
	} else if len(failedRows) > 0 && len(risksToCreate) == 0 {
		c.JSON(http.StatusBadRequest, response) // All rows failed validation before DB
	} else {
		c.JSON(http.StatusOK, response) // All rows (if any) imported successfully
	}
}
