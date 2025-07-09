package handlers

import (
	"net/http"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	// "strconv" // Removido - não usado
	"time"    // Adicionado

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserResponse DTO para evitar expor PasswordHash, etc.
type UserResponse struct {
	ID             uuid.UUID      `json:"id"`
	OrganizationID uuid.UUID      `json:"organization_id"`
	Name           string         `json:"name"`
	Email          string         `json:"email"`
	Role           models.UserRole `json:"role"`
	IsActive       bool           `json:"is_active"`
	SSOProvider    string         `json:"sso_provider,omitempty"`
	SocialLoginID  string         `json:"social_login_id,omitempty"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
}

func newUserResponse(user models.User) UserResponse {
	return UserResponse{
		ID:             user.ID,
		OrganizationID: user.OrganizationID,
		Name:           user.Name,
		Email:          user.Email,
		Role:           user.Role,
		IsActive:       user.IsActive,
		SSOProvider:    user.SSOProvider,
		SocialLoginID:  user.SocialLoginID,
		CreatedAt:      user.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      user.UpdatedAt.Format(time.RFC3339),
	}
}

func newListUserResponse(users []models.User) []UserResponse {
	responses := make([]UserResponse, len(users))
	for i, user := range users {
		responses[i] = newUserResponse(user)
	}
	return responses
}

// checkOrgAdminOrManager verifica se o usuário autenticado é admin ou manager da organização alvo.
// Esta é uma função helper que pode ser movida para um pacote de utils/auth no futuro.
func checkOrgAdminOrManager(c *gin.Context, targetOrgID uuid.UUID) bool {
	tokenOrgID, orgOk := c.Get("organizationID")
	tokenUserRole, roleOk := c.Get("userRole")

	if !orgOk || !roleOk {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Acesso negado: Informações do token ausentes"})
		return false
	}
	if tokenOrgID.(uuid.UUID) != targetOrgID {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Acesso negado: Você não pertence a esta organização"})
		return false
	}

	role := tokenUserRole.(models.UserRole)
	if role != models.RoleAdmin && role != models.RoleManager {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Acesso negado: Privilégios insuficientes (requer Admin ou Manager da organização)"})
		return false
	}
	return true
}


// ListOrganizationUsersHandler lista usuários de uma organização com paginação.
func ListOrganizationUsersHandler(c *gin.Context) {
	orgIDStr := c.Param("orgId")
	targetOrgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de ID da organização inválido"})
		return
	}

	if !checkOrgAdminOrManager(c, targetOrgID) {
		return // Erro já enviado por checkOrgAdminOrManager
	}

	page, pageSize := GetPaginationParams(c) // Helper de common.go

	db := database.GetDB()
	var users []models.User
	var totalItems int64

	query := db.Model(&models.User{}).Where("organization_id = ?", targetOrgID)

	if err := query.Count(&totalItems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao contar usuários da organização: " + err.Error()})
		return
	}

	if err := query.Scopes(PaginateScope(page, pageSize)).Order("name asc").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao listar usuários da organização: " + err.Error()})
		return
	}

	totalPages := totalItems / int64(pageSize)
	if totalItems%int64(pageSize) != 0 {
		totalPages++
	}
    if totalItems == 0 { totalPages = 0 }
    if totalPages == 0 && totalItems > 0 { totalPages = 1 }

	response := PaginatedResponse{
		Items:      newListUserResponse(users),
		TotalItems: totalItems,
		TotalPages: totalPages,
		Page:       page,
		PageSize:   pageSize,
	}
	c.JSON(http.StatusOK, response)
}

// GetOrganizationUserHandler obtém detalhes de um usuário específico da organização.
func GetOrganizationUserHandler(c *gin.Context) {
	orgIDStr := c.Param("orgId")
	targetOrgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de ID da organização inválido"})
		return
	}
	userIDStr := c.Param("userId")
	targetUserID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de ID do usuário inválido"})
		return
	}

	if !checkOrgAdminOrManager(c, targetOrgID) {
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.Where("id = ? AND organization_id = ?", targetUserID, targetOrgID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Usuário não encontrado nesta organização"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao buscar usuário: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, newUserResponse(user))
}


// UpdateUserRolePayload define o payload para atualizar a role de um usuário.
type UpdateUserRolePayload struct {
	Role models.UserRole `json:"role" binding:"required,oneof=admin manager user"`
}

// UpdateOrganizationUserRoleHandler atualiza a role de um usuário na organização.
func UpdateOrganizationUserRoleHandler(c *gin.Context) {
	orgIDStr := c.Param("orgId")
	targetOrgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de ID da organização inválido"})
		return
	}
	userIDStr := c.Param("userId")
	targetUserID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de ID do usuário inválido"})
		return
	}

	actingUserID, _ := c.Get("userID") // ID do usuário que está fazendo a ação

	if !checkOrgAdminOrManager(c, targetOrgID) {
		return
	}

	var payload UpdateUserRolePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido: " + err.Error()})
		return
	}

	db := database.GetDB()
	var userToUpdate models.User
	if err := db.Where("id = ? AND organization_id = ?", targetUserID, targetOrgID).First(&userToUpdate).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Usuário não encontrado para atualizar role"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao buscar usuário para atualizar role: " + err.Error()})
		return
	}

	// Lógica de prevenção de bloqueio: não permitir que o último admin/manager se rebaixe
	if userToUpdate.ID == actingUserID.(uuid.UUID) && (userToUpdate.Role == models.RoleAdmin || userToUpdate.Role == models.RoleManager) && payload.Role == models.RoleUser {
		var adminOrManagerCount int64
		db.Model(&models.User{}).Where("organization_id = ? AND (role = ? OR role = ?)", targetOrgID, models.RoleAdmin, models.RoleManager).Count(&adminOrManagerCount)
		if adminOrManagerCount <= 1 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Não é possível rebaixar o último administrador/gerente da organização."})
			return
		}
	}
    // Um admin não pode rebaixar outro admin se ele não for o único, mas um admin não pode rebaixar a si mesmo para user se for o único admin/manager.
    // A regra acima cobre o auto-rebaixamento. Para rebaixar outros, a role do ator já é verificada.

	userToUpdate.Role = payload.Role
	if err := db.Save(&userToUpdate).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao atualizar role do usuário: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, newUserResponse(userToUpdate))
}

// UpdateUserStatusPayload define o payload para ativar/desativar um usuário.
type UpdateUserStatusPayload struct {
	IsActive *bool `json:"is_active" binding:"required"` // Usar ponteiro para distinguir false de não fornecido
}

// UpdateOrganizationUserStatusHandler ativa ou desativa um usuário na organização.
func UpdateOrganizationUserStatusHandler(c *gin.Context) {
	orgIDStr := c.Param("orgId")
	targetOrgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de ID da organização inválido"})
		return
	}
	userIDStr := c.Param("userId")
	targetUserID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de ID do usuário inválido"})
		return
	}

	actingUserID, _ := c.Get("userID")

	if !checkOrgAdminOrManager(c, targetOrgID) {
		return
	}

	var payload UpdateUserStatusPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload inválido: " + err.Error()})
		return
	}

	db := database.GetDB()
	var userToUpdate models.User
	if err := db.Where("id = ? AND organization_id = ?", targetUserID, targetOrgID).First(&userToUpdate).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Usuário não encontrado para atualizar status"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao buscar usuário para atualizar status: " + err.Error()})
		return
	}

	// Lógica de prevenção de bloqueio: não permitir desativar o último admin/manager ativo
	if userToUpdate.ID == actingUserID.(uuid.UUID) && !(*payload.IsActive) && (userToUpdate.Role == models.RoleAdmin || userToUpdate.Role == models.RoleManager) {
		var activeAdminOrManagerCount int64
		db.Model(&models.User{}).Where("organization_id = ? AND is_active = ? AND (role = ? OR role = ?)",
            targetOrgID, true, models.RoleAdmin, models.RoleManager).Count(&activeAdminOrManagerCount)
		if activeAdminOrManagerCount <= 1 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Não é possível desativar o último administrador/gerente ativo da organização."})
			return
		}
	}


	userToUpdate.IsActive = *payload.IsActive
	if err := db.Save(&userToUpdate).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao atualizar status do usuário: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, newUserResponse(userToUpdate))
}
