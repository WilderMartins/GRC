package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image/png" // Para verificar o QR code
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"phoenixgrc/backend/internal/auth"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	appConfig "phoenixgrc/backend/pkg/config"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var mockMFADb *gorm.DB
var sqlMockMFA sqlmock.Sqlmock

func setupMFATestEnvironment(t *testing.T) {
	gin.SetMode(gin.TestMode)

	currentJwtKey := os.Getenv("JWT_SECRET_KEY")
	if currentJwtKey == "" {
		os.Setenv("JWT_SECRET_KEY", "testsecretfortests_mfa")
	}
	if err := auth.InitializeJWT(); err != nil {
		t.Fatalf("Failed to initialize JWT for tests: %v", err)
	}
	if currentJwtKey == "" {
		defer os.Unsetenv("JWT_SECRET_KEY")
	}

	// Configurar TOTP Issuer Name para testes
	originalIssuer := appConfig.Cfg.TOTPIssuerName
	appConfig.Cfg.TOTPIssuerName = "TestIssuer"
	t.Cleanup(func() {
		appConfig.Cfg.TOTPIssuerName = originalIssuer
	})


	var err error
	db, smock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("An error '%s' was not expected when opening a stub database connection", err)
	}
	sqlMockMFA = smock

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Silent,
			Colorful:      true,
		},
	)
	mockMFADb, err = gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{Logger: gormLogger})
	if err != nil {
		t.Fatalf("An error '%s' was not expected when opening gorm database: %v", err)
	}
	originalDB := database.DB
	database.SetDB(mockMFADb)
	t.Cleanup(func() {
		database.DB = originalDB
		db.Close()
	})
}


func TestSetupTOTPHandler_Success(t *testing.T) {
	setupMFATestEnvironment(t)

	router := gin.Default()
	userID := uuid.New()
	userEmail := "testsetup@example.com"

	// Mock auth middleware
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Set("userEmail", userEmail) // Embora o handler pegue do DB, pode ser útil
		c.Next()
	})
	router.POST("/setup-totp", SetupTOTPHandler)

	mockUser := models.User{
		ID:    userID,
		Email: userEmail,
		// TOTPSecret e IsTOTPEnabled serão definidos pelo handler
	}

	// Mock DB calls
	// 1. Fetch user
	rowsUser := sqlMockMFA.NewRows([]string{"id", "email", "totp_secret", "is_totp_enabled"}).
		AddRow(mockUser.ID, mockUser.Email, "", false)
	sqlMockMFA.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT 1`)).
		WithArgs(userID).
		WillReturnRows(rowsUser)

	// 2. Save user with new TOTPSecret
	sqlMockMFA.ExpectBegin()
	// A ordem das colunas no UPDATE pode variar, mas os valores são importantes.
	// O GORM pode fazer um SELECT antes do UPDATE ou usar RETURNING.
	// Simplificando aqui para esperar um UPDATE.
	// A secret é gerada, então não podemos mockar o valor exato facilmente sem mockar otp.Generate
	// Em vez disso, verificamos que o campo é atualizado.
	sqlMockMFA.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET "totp_secret"=$1,"is_totp_enabled"=$2,"updated_at"=$3 WHERE "id" = $4`)).
		WithArgs(sqlmock.AnyArg(), false, sqlmock.AnyArg(), userID). // totp_secret (AnyArg), is_totp_enabled=false, updated_at (AnyArg)
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMockMFA.ExpectCommit()

	req, _ := http.NewRequest(http.MethodPost, "/setup-totp", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response SetupTOTPResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.NotEmpty(t, response.Secret, "Secret should not be empty")
	assert.Equal(t, userEmail, response.Account)
	assert.Equal(t, appConfig.Cfg.TOTPIssuerName, response.Issuer) // Verifica se o issuer do config foi usado
	assert.False(t, response.BackupCodesGenerated, "Backup codes should not be generated in this basic setup")

	// Validate QR Code (basic check: is it a valid base64 PNG?)
	assert.True(t, strings.HasPrefix(response.QRCode, "data:image/png;base64,"), "QR Code should be a base64 encoded PNG")
	qrData, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(response.QRCode, "data:image/png;base64,"))
	assert.NoError(t, err, "Failed to decode QR code base64 data")
	_, err = png.Decode(bytes.NewReader(qrData))
	assert.NoError(t, err, "QR code data is not a valid PNG")

	// Validate content of otpauth URL from secret (optional, more involved)
	otpKey, err := otp.NewKeyFromURL(fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s", response.Issuer, response.Account, response.Secret, response.Issuer))
	assert.NoError(t, err, "Generated secret does not form a valid otpauth URL")
	assert.Equal(t, response.Secret, otpKey.Secret())


	assert.NoError(t, sqlMockMFA.ExpectationsWereMet(), "SQL mock expectations were not met")
}

func TestSetupTOTPHandler_UserNotFound(t *testing.T) {
	setupMFATestEnvironment(t)
	router := gin.Default()
	userID := uuid.New() // Um ID de usuário que não existirá
	router.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() }) // Mock Auth Middleware
	router.POST("/setup-totp", SetupTOTPHandler)

	// Mock DB: Fetch user (não encontrado)
	sqlMockMFA.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT 1`)).
		WithArgs(userID).
		WillReturnError(gorm.ErrRecordNotFound)

	req, _ := http.NewRequest(http.MethodPost, "/setup-totp", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	var errorResponse map[string]string
	json.Unmarshal(rr.Body.Bytes(), &errorResponse)
	assert.Equal(t, "User not found", errorResponse["error"])
	assert.NoError(t, sqlMockMFA.ExpectationsWereMet())
}

func TestVerifyTOTPHandler_Success_Enable2FA(t *testing.T) {
	setupMFATestEnvironment(t)
	router := gin.Default()
	userID := uuid.New()

	router.Use(func(c *gin.Context) { // Mock Auth Middleware
		c.Set("userID", userID)
		c.Next()
	})
	router.POST("/verify-totp", VerifyTOTPHandler)

	plainTextSecret := "R3K44Z6L54SM4PZJ" // Outro segredo válido
	encryptedSecret, errEncrypt := utils.Encrypt(plainTextSecret)
	assert.NoError(t, errEncrypt)

	mockUser := models.User{
		ID:            userID,
		Email:         "verify@example.com",
		TOTPSecret:    encryptedSecret,
		IsTOTPEnabled: false, // Importante: testando o fluxo de habilitação
		IsActive:      true,
	}

	// Mock DB calls
	// 1. Fetch user
	rowsUser := sqlMockMFA.NewRows([]string{"id", "email", "totp_secret", "is_totp_enabled", "is_active"}).
		AddRow(mockUser.ID, mockUser.Email, mockUser.TOTPSecret, mockUser.IsTOTPEnabled, mockUser.IsActive)
	sqlMockMFA.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT 1`)).
		WithArgs(userID).
		WillReturnRows(rowsUser)

	// 2. Save user (to set IsTOTPEnabled = true)
	sqlMockMFA.ExpectBegin()
	sqlMockMFA.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET "is_totp_enabled"=$1,"updated_at"=$2 WHERE "id" = $3`)).
		WithArgs(true, sqlmock.AnyArg(), userID). // is_totp_enabled=true
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMockMFA.ExpectCommit()

	validToken, errToken := totp.GenerateCode(plainTextSecret, time.Now())
	assert.NoError(t, errToken)

	payload := VerifyTOTPPayload{Token: validToken}
	jsonPayload, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/verify-totp", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Response body: %s", rr.Body.String())
	var response map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "TOTP successfully verified and enabled.", response["message"])
	assert.NoError(t, sqlMockMFA.ExpectationsWereMet())
}

func TestVerifyTOTPHandler_InvalidToken(t *testing.T) {
	setupMFATestEnvironment(t)
	router := gin.Default()
	userID := uuid.New()
	router.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	router.POST("/verify-totp", VerifyTOTPHandler)

	plainTextSecret := "R3K44Z6L54SM4PZK"
	encryptedSecret, _ := utils.Encrypt(plainTextSecret)
	mockUser := models.User{ID: userID, Email: "badtoken@example.com", TOTPSecret: encryptedSecret, IsTOTPEnabled: true, IsActive: true}

	rowsUser := sqlMockMFA.NewRows([]string{"id", "email", "totp_secret", "is_totp_enabled", "is_active"}).
		AddRow(mockUser.ID, mockUser.Email, mockUser.TOTPSecret, mockUser.IsTOTPEnabled, mockUser.IsActive)
	sqlMockMFA.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1`)).WithArgs(userID).WillReturnRows(rowsUser)

	payload := VerifyTOTPPayload{Token: "000000"} // Token inválido
	jsonPayload, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/verify-totp", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var errorResponse map[string]string
	json.Unmarshal(rr.Body.Bytes(), &errorResponse)
	assert.Equal(t, "Invalid TOTP token", errorResponse["error"])
	assert.NoError(t, sqlMockMFA.ExpectationsWereMet())
}

func TestVerifyTOTPHandler_TOTPNotSetup(t *testing.T) {
	setupMFATestEnvironment(t)
	router := gin.Default()
	userID := uuid.New()
	router.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	router.POST("/verify-totp", VerifyTOTPHandler)

	mockUser := models.User{ID: userID, Email: "notsetup@example.com", TOTPSecret: "", IsTOTPEnabled: false, IsActive: true} // TOTPSecret está vazio
	rowsUser := sqlMockMFA.NewRows([]string{"id", "email", "totp_secret", "is_totp_enabled", "is_active"}).
		AddRow(mockUser.ID, mockUser.Email, mockUser.TOTPSecret, mockUser.IsTOTPEnabled, mockUser.IsActive)
	sqlMockMFA.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1`)).WithArgs(userID).WillReturnRows(rowsUser)

	payload := VerifyTOTPPayload{Token: "123456"}
	jsonPayload, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/verify-totp", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errorResponse map[string]string
	json.Unmarshal(rr.Body.Bytes(), &errorResponse)
	assert.Equal(t, "TOTP not set up for this user. Please set up TOTP first.", errorResponse["error"])
	assert.NoError(t, sqlMockMFA.ExpectationsWereMet())
}

func TestVerifyTOTPHandler_AlreadyEnabled(t *testing.T) {
	setupMFATestEnvironment(t)
	router := gin.Default()
	userID := uuid.New()
	router.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	router.POST("/verify-totp", VerifyTOTPHandler)

	plainTextSecret := "R3K44Z6L54SM4PXL"
	encryptedSecret, _ := utils.Encrypt(plainTextSecret)
	mockUser := models.User{
		ID:            userID,
		Email:         "alreadyenabled@example.com",
		TOTPSecret:    encryptedSecret,
		IsTOTPEnabled: true, // TOTP já está habilitado
		IsActive:      true,
	}

	rowsUser := sqlMockMFA.NewRows([]string{"id", "email", "totp_secret", "is_totp_enabled", "is_active"}).
		AddRow(mockUser.ID, mockUser.Email, mockUser.TOTPSecret, mockUser.IsTOTPEnabled, mockUser.IsActive)
	sqlMockMFA.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1`)).WithArgs(userID).WillReturnRows(rowsUser)
	// Nenhuma chamada de DB Save é esperada aqui, pois IsTOTPEnabled já é true

	validToken, _ := totp.GenerateCode(plainTextSecret, time.Now())
	payload := VerifyTOTPPayload{Token: validToken}
	jsonPayload, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, "/verify-totp", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var response map[string]string
	json.Unmarshal(rr.Body.Bytes(), &response)
	assert.Equal(t, "TOTP token verified successfully.", response["message"]) // Mensagem para quando já está habilitado
	assert.NoError(t, sqlMockMFA.ExpectationsWereMet())
}

func TestDisableTOTPHandler_Success(t *testing.T) {
	setupMFATestEnvironment(t)
	router := gin.Default()
	userID := uuid.New()
	userPassword := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(userPassword), bcrypt.DefaultCost)

	router.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	router.POST("/disable-totp", DisableTOTPHandler)

	encryptedSecret, _ := utils.Encrypt("SOMEBIGSECRET")
	mockUser := models.User{
		ID:            userID,
		Email:         "disable@example.com",
		PasswordHash:  string(hashedPassword),
		TOTPSecret:    encryptedSecret,
		IsTOTPEnabled: true,
		IsActive:      true,
	}
	rowsUser := sqlMockMFA.NewRows([]string{"id", "email", "password_hash", "totp_secret", "is_totp_enabled", "is_active"}).
		AddRow(mockUser.ID, mockUser.Email, mockUser.PasswordHash, mockUser.TOTPSecret, mockUser.IsTOTPEnabled, mockUser.IsActive)
	sqlMockMFA.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT 1`)).
		WithArgs(userID).
		WillReturnRows(rowsUser)

	sqlMockMFA.ExpectBegin()
	sqlMockMFA.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET "is_totp_enabled"=$1,"totp_secret"=$2,"totp_backup_codes"=$3,"updated_at"=$4 WHERE "id" = $5`)).
		WithArgs(false, "", "", sqlmock.AnyArg(), userID). // is_totp_enabled=false, secret="", backup_codes=""
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMockMFA.ExpectCommit()

	payload := DisableTOTPPayload{Password: userPassword}
	jsonPayload, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/disable-totp", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Response: %s", rr.Body.String())
	var response map[string]string
	json.Unmarshal(rr.Body.Bytes(), &response)
	assert.Equal(t, "TOTP has been successfully disabled.", response["message"])
	assert.NoError(t, sqlMockMFA.ExpectationsWereMet())
}

func TestDisableTOTPHandler_InvalidPassword(t *testing.T) {
	setupMFATestEnvironment(t)
	router := gin.Default()
	userID := uuid.New()
	router.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	router.POST("/disable-totp", DisableTOTPHandler)

	userPassword := "realpassword"
	wrongPassword := "wrongpassword"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(userPassword), bcrypt.DefaultCost)
	encryptedSecret, _ := utils.Encrypt("SOMEBIGSECRET")
	mockUser := models.User{ID: userID, PasswordHash: string(hashedPassword), TOTPSecret: encryptedSecret, IsTOTPEnabled: true, IsActive: true}

	rowsUser := sqlMockMFA.NewRows([]string{"id", "password_hash", "totp_secret", "is_totp_enabled", "is_active"}).
		AddRow(mockUser.ID, mockUser.PasswordHash, mockUser.TOTPSecret, mockUser.IsTOTPEnabled, mockUser.IsActive)
	sqlMockMFA.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1`)).WithArgs(userID).WillReturnRows(rowsUser)

	payload := DisableTOTPPayload{Password: wrongPassword}
	jsonPayload, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/disable-totp", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	var errorResponse map[string]string
	json.Unmarshal(rr.Body.Bytes(), &errorResponse)
	assert.Equal(t, "Invalid password", errorResponse["error"])
	assert.NoError(t, sqlMockMFA.ExpectationsWereMet())
}

func TestDisableTOTPHandler_TOTPNotEnabled(t *testing.T) {
	setupMFATestEnvironment(t)
	router := gin.Default()
	userID := uuid.New()
	userPassword := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(userPassword), bcrypt.DefaultCost)
	router.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	router.POST("/disable-totp", DisableTOTPHandler)

	mockUser := models.User{ID: userID, PasswordHash: string(hashedPassword), IsTOTPEnabled: false, IsActive: true} // TOTP já desabilitado

	rowsUser := sqlMockMFA.NewRows([]string{"id", "password_hash", "is_totp_enabled", "is_active"}).
		AddRow(mockUser.ID, mockUser.PasswordHash, mockUser.IsTOTPEnabled, mockUser.IsActive)
	sqlMockMFA.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1`)).WithArgs(userID).WillReturnRows(rowsUser)

	payload := DisableTOTPPayload{Password: userPassword}
	jsonPayload, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, "/disable-totp", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errorResponse map[string]string
	json.Unmarshal(rr.Body.Bytes(), &errorResponse)
	assert.Equal(t, "TOTP is not currently enabled for this account.", errorResponse["error"])
	assert.NoError(t, sqlMockMFA.ExpectationsWereMet())
}

func TestGenerateBackupCodesHandler_Success(t *testing.T) {
	setupMFATestEnvironment(t)
	router := gin.Default()
	userID := uuid.New()
	router.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	router.POST("/generate-backup-codes", GenerateBackupCodesHandler)

	mockUser := models.User{ID: userID, IsTOTPEnabled: true, IsActive: true} // TOTP precisa estar habilitado
	rowsUser := sqlMockMFA.NewRows([]string{"id", "is_totp_enabled", "is_active"}).
		AddRow(mockUser.ID, mockUser.IsTOTPEnabled, mockUser.IsActive)
	sqlMockMFA.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT 1`)).
		WithArgs(userID).
		WillReturnRows(rowsUser)

	sqlMockMFA.ExpectBegin()
	sqlMockMFA.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET "totp_backup_codes"=$1,"updated_at"=$2 WHERE "id" = $3`)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), userID). // Verifica se totp_backup_codes é atualizado
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMockMFA.ExpectCommit()

	req, _ := http.NewRequest(http.MethodPost, "/generate-backup-codes", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var response GenerateBackupCodesResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response.BackupCodes, numBackupCodes, "Deveria gerar o número correto de códigos de backup")
	for _, code := range response.BackupCodes {
		assert.Len(t, code, backupCodeLength, "Código de backup com tamanho incorreto")
	}
	assert.NoError(t, sqlMockMFA.ExpectationsWereMet())
}

func TestGenerateBackupCodesHandler_TOTPNotEnabled(t *testing.T) {
	setupMFATestEnvironment(t)
	router := gin.Default()
	userID := uuid.New()
	router.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	router.POST("/generate-backup-codes", GenerateBackupCodesHandler)

	mockUser := models.User{ID: userID, IsTOTPEnabled: false, IsActive: true} // TOTP desabilitado
	rowsUser := sqlMockMFA.NewRows([]string{"id", "is_totp_enabled", "is_active"}).
		AddRow(mockUser.ID, mockUser.IsTOTPEnabled, mockUser.IsActive)
	sqlMockMFA.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1`)).WithArgs(userID).WillReturnRows(rowsUser)

	req, _ := http.NewRequest(http.MethodPost, "/generate-backup-codes", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	var errorResponse map[string]string
	json.Unmarshal(rr.Body.Bytes(), &errorResponse)
	assert.Equal(t, "TOTP must be enabled to generate backup codes.", errorResponse["error"])
	assert.NoError(t, sqlMockMFA.ExpectationsWereMet())
}

// Ensure newline at end of file
