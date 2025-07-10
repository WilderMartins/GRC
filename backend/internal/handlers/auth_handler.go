package handlers

import (
	"net/http"
	"phoenixgrc/backend/internal/auth"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid" // Added for parsing UserID
	"github.com/pquerna/otp/totp" // Added for TOTP validation
	"golang.org/x/crypto/bcrypt"
	"encoding/json" // Added for backup codes
)

type LoginPayload struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token          string          `json:"token"`
	UserID         string          `json:"user_id"`
	Email          string          `json:"email"`
	Name           string          `json:"name"`
	Role           models.UserRole `json:"role"`
	OrganizationID string          `json:"organization_id"`
}

// LoginHandler lida com o login do usuário.
func LoginHandler(c *gin.Context) {
	var payload LoginPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", payload.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Verificar se o usuário está ativo
	if !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User account is inactive"})
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(payload.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	tokenString, err := auth.GenerateToken(&user, user.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token: " + err.Error()})
		return
	}

	// Check if 2FA/TOTP is enabled for the user
	if user.IsTOTPEnabled {
		// Do not issue the full JWT token yet.
		// Issue a temporary token or indicate that 2FA is required.
		// For simplicity, we'll return a specific response.
		// The frontend will then prompt for the TOTP code and make a new request.
		c.JSON(http.StatusOK, gin.H{ // Could be a different status code like 202 Accepted or a custom one if needed
			"2fa_required": true,
			"user_id":      user.ID.String(),
			"message":      "Password verified. Please provide TOTP token.",
		})
		return
	}

	// If 2FA is not enabled, proceed with normal login and token issuance
	c.JSON(http.StatusOK, LoginResponse{
		Token:          tokenString,
		UserID:         user.ID.String(),
		Email:          user.Email,
		Name:           user.Name,
		Role:           user.Role,
		OrganizationID: user.OrganizationID.String(),
	})
}

type LoginVerifyBackupCodePayload struct {
	UserID     string `json:"user_id" binding:"required"`
	BackupCode string `json:"backup_code" binding:"required"`
}

// LoginVerifyBackupCodeHandler handles the 2FA step using a backup code.
func LoginVerifyBackupCodeHandler(c *gin.Context) {
	var payload LoginVerifyBackupCodePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	userUUID, err := uuid.Parse(payload.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UserID format"})
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, "id = ?", userUUID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found or invalid state"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User account is inactive"})
		return
	}

	if !user.IsTOTPEnabled || user.TOTPBackupCodes == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "2FA / Backup codes not enabled or not generated for this user."})
		return
	}

	var storedHashedCodes []string
	if err := json.Unmarshal([]byte(user.TOTPBackupCodes), &storedHashedCodes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse stored backup codes."})
		return
	}

	validCodeFound := false
	updatedHashedCodes := []string{}

	for _, hashedCode := range storedHashedCodes {
		err := bcrypt.CompareHashAndPassword([]byte(hashedCode), []byte(payload.BackupCode))
		if err == nil { // Match found
			validCodeFound = true
			// This code is now used, do not add it to updatedHashedCodes
		} else {
			updatedHashedCodes = append(updatedHashedCodes, hashedCode) // Keep unused codes
		}
	}

	if !validCodeFound {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid backup code."})
		return
	}

	// Update user's backup codes (remove the used one)
	newBackupCodesJSON, err := json.Marshal(updatedHashedCodes)
	if err != nil {
		// Log this error, but proceed with login as code was valid.
		// User should be prompted to regenerate backup codes if this list becomes empty.
		log.Printf("Error marshalling updated backup codes for user %s: %v", user.ID, err)
	} else {
		user.TOTPBackupCodes = string(newBackupCodesJSON)
		if err := db.Save(&user).Error; err != nil {
			// Log this error, but proceed with login.
			log.Printf("Error saving updated backup codes for user %s: %v", user.ID, err)
		}
	}

	// Backup code is valid, issue JWT
	tokenString, err := auth.GenerateToken(&user, user.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token:          tokenString,
		UserID:         user.ID.String(),
		Email:          user.Email,
		Name:           user.Name,
		Role:           user.Role,
		OrganizationID: user.OrganizationID.String(),
	})
}

type LoginVerifyTOTPPayload struct {
	UserID string `json:"user_id" binding:"required"`
	Token  string `json:"token" binding:"required"`
}

// LoginVerifyTOTPHandler handles the second step of login for users with TOTP enabled.
// It verifies the TOTP token and, if valid, issues the full JWT.
func LoginVerifyTOTPHandler(c *gin.Context) {
	var payload LoginVerifyTOTPPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	userUUID, err := uuid.Parse(payload.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UserID format"})
		return
	}

	db := database.GetDB()
	var user models.User
	// It's crucial to fetch the user from DB again to ensure their current state.
	if err := db.First(&user, "id = ?", userUUID).Error; err != nil {
		// This could happen if user was deleted between password step and 2FA step, though unlikely.
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found or invalid state"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User account is inactive"})
		return
	}

	if !user.IsTOTPEnabled || user.TOTPSecret == "" {
		// This state should ideally not be reachable if LoginHandler directed here,
		// but good to double-check.
		c.JSON(http.StatusForbidden, gin.H{"error": "TOTP is not enabled for this user."})
		return
	}

	valid := totp.Validate(payload.Token, user.TOTPSecret)
	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid TOTP token"})
		return
	}

	// TOTP is valid, now issue the full JWT token.
	tokenString, err := auth.GenerateToken(&user, user.OrganizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token:          tokenString,
		UserID:         user.ID.String(),
		Email:          user.Email,
		Name:           user.Name,
		Role:           user.Role,
		OrganizationID: user.OrganizationID.String(),
	})
}
