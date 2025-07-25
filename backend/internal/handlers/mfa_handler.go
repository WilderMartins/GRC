package handlers

import (
	"bytes"
	"crypto/rand" // Adicionado para gerar códigos de backup
	"encoding/base64"
	"encoding/json" // Adicionado para Marshal/Unmarshal de backup codes
	"fmt"
	"image/png"
	"net/http"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	"phoenixgrc/backend/internal/utils"       // Added for crypto utils
	appConfig "phoenixgrc/backend/pkg/config" // Alias para o pacote de configuração
	phxlog "phoenixgrc/backend/pkg/log"        // Importar o logger zap
	"go.uber.org/zap"                         // Importar zap

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt" // Added for password verification
)

type SetupTOTPResponse struct {
	Secret    string `json:"secret"`     // Base32 encoded secret
	QRCode    string `json:"qr_code"`    // Base64 encoded PNG image
	Account   string `json:"account"`    // User's email or identifier
	Issuer    string `json:"issuer"`     // Issuer name (from config)
	BackupCodesGenerated bool `json:"backup_codes_generated"` // Indica se novos códigos de backup foram gerados
}

// SetupTOTPHandler generates a new TOTP secret for the user and returns it along with a QR code.
// This endpoint is called when a user wants to start setting up TOTP.
// The TOTP is not yet enabled; it's only enabled after verification.
func SetupTOTPHandler(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}
	userUUID := userID.(uuid.UUID)

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, "id = ?", userUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Generate a new TOTP key.
	// Issuer from config, account name is user's email.
	issuer := appConfig.Cfg.TOTPIssuerName
	if issuer == "" {
		issuer = "PhoenixGRC" // Fallback if not in config
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: user.Email, // Use user's email as the account name in the OTP app
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate TOTP key: " + err.Error()})
		return
	}

	// Store the secret in the user's record.
	// IMPORTANT: In a real application, consider encrypting this secret at rest in the database.
	encryptedSecret, err := utils.Encrypt(key.Secret())
	if err != nil {
		phxlog.L.Error("Failed to encrypt TOTP secret", zap.String("userID", user.ID.String()), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to secure TOTP secret"})
		return
	}
	user.TOTPSecret = encryptedSecret
	// Reset IsTOTPEnabled to false because this is a new setup/re-setup.
	// It will be set to true only after successful verification.
	user.IsTOTPEnabled = false

	// Generate new backup codes when setting up a new TOTP secret.
	// For simplicity, this example does not implement backup codes yet.
	// This would involve generating a set of single-use codes, hashing them,
	// and storing the hashes. The plain codes are shown to the user once.
	// user.TOTPBackupCodes = "[]" // Placeholder for hashed backup codes as JSON array string

	// For now, we will skip backup code generation in this step.
	// It should be a separate, explicit action by the user after enabling TOTP.

	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save TOTP secret: " + err.Error()})
		return
	}

	// Generate QR code image.
	// var qrCodeImage bytes.Buffer // Removida pois pngBytes é usado diretamente
	// The URL for the QR code is otpauth://totp/ISSUER:ACCOUNT?secret=SECRET&issuer=ISSUER
	// The key.String() method provides this URL.

	// Generate PNG image of the QR code
	pngBytes, err := qrcode.Encode(key.String(), qrcode.Medium, 256) // Returns []byte, error
	if err != nil {
		// Fallback or alternative if direct Encode fails or to be more explicit with png encoding
		img, errImg := qrcode.New(key.String(), qrcode.Medium)
		if errImg != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create QR code object: " + errImg.Error()})
			return
		}
		var buf bytes.Buffer
		if errPng := png.Encode(&buf, img.Image(256)); errPng != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode QR code to PNG: " + errPng.Error()})
			return
		}
		pngBytes = buf.Bytes()
	}

	response := SetupTOTPResponse{
		Secret:    key.Secret(), // The Base32 encoded secret string
		QRCode:    fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(pngBytes)),
		Account:   user.Email,
		Issuer:    issuer,
		BackupCodesGenerated: false, // Set to true if backup codes were generated
	}

	c.JSON(http.StatusOK, response)
}

type VerifyTOTPPayload struct {
	Token string `json:"token" binding:"required"`
}

// VerifyTOTPHandler verifies a TOTP token provided by the user during setup or login.
// If called during setup, it enables TOTP for the user.
// If called during a 2FA login step, it would complete the login (logic to be added to login flow).
func VerifyTOTPHandler(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}
	userUUID := userID.(uuid.UUID)

	var payload VerifyTOTPPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, "id = ?", userUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.TOTPSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TOTP not set up for this user. Please set up TOTP first."})
		return
	}

	decryptedSecret, err := utils.Decrypt(user.TOTPSecret)
	if err != nil {
		phxlog.L.Error("Failed to decrypt TOTP secret", zap.String("userID", user.ID.String()), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process TOTP secret"})
		return
	}

	valid := totp.Validate(payload.Token, decryptedSecret)
	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid TOTP token"})
		return
	}

	// If this is the first time verifying (i.e., enabling TOTP)
	if !user.IsTOTPEnabled {
		user.IsTOTPEnabled = true
		if err := db.Save(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enable TOTP: " + err.Error()})
			return
		}
		// TODO: Consider generating backup codes here and returning them.
		// For now, just confirm enablement.
		c.JSON(http.StatusOK, gin.H{"message": "TOTP successfully verified and enabled."})
		return
	}

	// If TOTP was already enabled, this endpoint might be used as a part of a 2FA login flow,
	// or just as a way to re-verify. For now, just a success message.
	// The actual login flow modification is a separate step.
	c.JSON(http.StatusOK, gin.H{"message": "TOTP token verified successfully."})
}

type DisableTOTPPayload struct {
	Password string `json:"password" binding:"required"`
}

// DisableTOTPHandler allows a user to disable TOTP for their account.
// Requires current password for verification.
func DisableTOTPHandler(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}
	userUUID := userID.(uuid.UUID)

	var payload DisableTOTPPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, "id = ?", userUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Verify current password
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(payload.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	if !user.IsTOTPEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TOTP is not currently enabled for this account."})
		return
	}

	user.IsTOTPEnabled = false
	user.TOTPSecret = "" // Clear the secret
	user.TOTPBackupCodes = "" // Clear backup codes as well

	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disable TOTP: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "TOTP has been successfully disabled."})
}

const numBackupCodes = 10
const backupCodeLength = 10 // Length of each backup code

// generateRandomString generates a random alphanumeric string of a given length.
// Used for backup codes.
func generateRandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}
	return string(bytes), nil
}

// GenerateBackupCodesResponse defines the response for generating backup codes.
type GenerateBackupCodesResponse struct {
	BackupCodes []string `json:"backup_codes"`
}

// GenerateBackupCodesHandler generates new backup codes for the user.
// This invalidates any previously generated backup codes.
func GenerateBackupCodesHandler(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}
	userUUID := userID.(uuid.UUID)

	db := database.GetDB()
	var user models.User
	if err := db.First(&user, "id = ?", userUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if !user.IsTOTPEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TOTP must be enabled to generate backup codes."})
		return
	}

	var plainTextCodes []string
	var hashedCodes []string

	for i := 0; i < numBackupCodes; i++ {
		code, err := generateRandomString(backupCodeLength)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate backup code string"})
			return
		}
		plainTextCodes = append(plainTextCodes, code)

		hashedCode, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash backup code"})
			return
		}
		hashedCodes = append(hashedCodes, string(hashedCode))
	}

	backupCodesJSON, err := json.Marshal(hashedCodes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal hashed backup codes"})
		return
	}
	user.TOTPBackupCodes = string(backupCodesJSON)

	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save backup codes: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, GenerateBackupCodesResponse{BackupCodes: plainTextCodes})
}


// Ensure newline at end of file
