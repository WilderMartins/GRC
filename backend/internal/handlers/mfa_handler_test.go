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

// TODO: TestSetupTOTPHandler_UserNotFound
// TODO: TestVerifyTOTPHandler_Success_Enable2FA
// TODO: TestVerifyTOTPHandler_AlreadyEnabled
// TODO: TestVerifyTOTPHandler_InvalidToken
// TODO: TestVerifyTOTPHandler_TOTPNotSetup
// TODO: TestDisableTOTPHandler_Success
// TODO: TestDisableTOTPHandler_InvalidPassword
// TODO: TestDisableTOTPHandler_TOTPNotEnabled

// Ensure newline at end of file
