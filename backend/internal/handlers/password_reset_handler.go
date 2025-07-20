package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	"phoenixgrc/backend/internal/notifications"
	phxlog "phoenixgrc/backend/pkg/log"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type ForgotPasswordPayload struct {
	Email string `json:"email" binding:"required,email"`
}

// ForgotPasswordHandler inicia o processo de reset de senha.
func ForgotPasswordHandler(c *gin.Context) {
	log := phxlog.L.Named("ForgotPasswordHandler")
	var payload ForgotPasswordPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.Where("email = ?", payload.Email).First(&user).Error; err != nil {
		// Não revele se o e-mail existe ou não.
		log.Info("Password reset requested for non-existent email", zap.String("email", payload.Email))
		c.JSON(http.StatusOK, gin.H{"message": "If an account with that email exists, a password reset link has been sent."})
		return
	}

	// Gerar um token seguro
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		log.Error("Failed to generate password reset token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	token := hex.EncodeToString(tokenBytes)

	resetToken := models.PasswordResetToken{
		Token:     token,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(1 * time.Hour), // Token válido por 1 hora
	}

	if err := db.Create(&resetToken).Error; err != nil {
		log.Error("Failed to save password reset token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save token"})
		return
	}

	// Enviar e-mail de reset de senha
	frontendBaseURL, _ := models.GetSystemSetting(db, "FRONTEND_BASE_URL")
	if frontendBaseURL == "" {
		frontendBaseURL = "http://localhost" // Fallback
	}
	resetLink := fmt.Sprintf("%s/auth/reset-password?token=%s", frontendBaseURL, token)

	bodyHTML := fmt.Sprintf(`
        <h2>Password Reset Request</h2>
        <p>You requested a password reset. Click the link below to reset your password:</p>
        <p><a href="%s">Reset Password</a></p>
        <p>This link is valid for 1 hour. If you did not request this, please ignore this email.</p>
    `, resetLink)

	// O corpo do e-mail para SES pode ser o mesmo para HTML e Texto por simplicidade,
	// ou você pode criar versões diferentes. Usando bodyHTML para ambos.
	if err := notifications.DefaultEmailNotifier.Send(c.Request.Context(), user.Email, "Password Reset Request", bodyHTML); err != nil {
		log.Error("Failed to send password reset email", zap.Error(err))
		// Não retorne o erro ao usuário por segurança.
	}

	c.JSON(http.StatusOK, gin.H{"message": "If an account with that email exists, a password reset link has been sent."})
}

type ResetPasswordPayload struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

// ResetPasswordHandler finaliza o processo de reset de senha.
func ResetPasswordHandler(c *gin.Context) {
	log := phxlog.L.Named("ResetPasswordHandler")
	var payload ResetPasswordPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	db := database.GetDB()
	var resetToken models.PasswordResetToken
	if err := db.Where("token = ?", payload.Token).Preload("User").First(&resetToken).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired token"})
		return
	}

	if time.Now().After(resetToken.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired token"})
		// Opcional: deletar o token expirado
		db.Delete(&resetToken)
		return
	}

	// Atualizar a senha do usuário
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("Failed to hash new password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process new password"})
		return
	}

	user := resetToken.User
	user.PasswordHash = string(hashedPassword)
	if err := db.Save(&user).Error; err != nil {
		log.Error("Failed to update password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	// Invalidar o token após o uso
	db.Delete(&resetToken)

	c.JSON(http.StatusOK, gin.H{"message": "Password has been reset successfully."})
}
