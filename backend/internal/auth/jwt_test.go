package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"phoenixgrc/backend/internal/models"
	"testing"
	"time"
	"errors" // Adicionado para errors.Is

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Setup: Initialize JWT with a test key before running tests
	os.Setenv("JWT_SECRET_KEY", "testsecretkeyforjwtauthentication")
	os.Setenv("JWT_TOKEN_LIFESPAN_HOURS", "1")
	if err := InitializeJWT(); err != nil {
		panic("Failed to initialize JWT for testing: " + err.Error())
	}
	// Run tests
	exitVal := m.Run()
	// Teardown: Clean up environment variables if necessary
	os.Unsetenv("JWT_SECRET_KEY")
	os.Unsetenv("JWT_TOKEN_LIFESPAN_HOURS")
	os.Exit(exitVal)
}

func TestGenerateToken(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	user := &models.User{
		ID:             userID,
		Email:          "test@example.com",
		Role:           models.RoleUser,
		OrganizationID: uuid.NullUUID{UUID: orgID, Valid: true},
	}

	tokenString, err := GenerateToken(user, user.OrganizationID.UUID)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	// Validate the token structure (optional, more thorough)
	claims, err := ValidateToken(tokenString)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, user.Email, claims.Email)
	assert.Equal(t, user.Role, claims.Role)
	assert.Equal(t, user.OrganizationID.UUID, claims.OrganizationID)
	assert.Equal(t, "phoenix-grc", claims.Issuer)
	assert.WithinDuration(t, time.Now().Add(1*time.Hour), claims.ExpiresAt.Time, 5*time.Second) // Allow 5s clock skew
}

func TestValidateToken_Valid(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	user := &models.User{

		ID:             userID,
		Email:          "valid@example.com",
		Role:           models.RoleAdmin,
		OrganizationID: uuid.NullUUID{UUID: orgID, Valid: true},
	}
	tokenString, _ := GenerateToken(user, user.OrganizationID.UUID)

	claims, err := ValidateToken(tokenString)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	// Generate a token with the correct key
	userID := uuid.New()
	orgID := uuid.New()
	user := &models.User{ID: userID, Email: "test@example.com", Role: models.RoleUser, OrganizationID: uuid.NullUUID{UUID: orgID, Valid: true}}
	tokenString, _ := GenerateToken(user, user.OrganizationID)

	// Tamper with the token or try to validate with a different key (simulated by re-initializing with wrong key)
	// For simplicity, we'll just check against a known invalid token structure.
	// A more direct way is to parse without validation and then try to validate parts.
	// Or, create a token signed with a *different* key.

	// Let's try validating a structurally valid but wrongly signed token by changing the key temporarily
	originalKey := jwtKey
	jwtKey = []byte("wrongsecretkey") // Use a different key for this test case

	_, err := ValidateToken(tokenString) // This token was signed with 'testsecretkey...'
	assert.Error(t, err)
	// The error should be something like "signature is invalid" or related to crypto verification
	// This depends on the jwt library's specific error messages.
	// For jwt/v5, it's often jwt.ErrSignatureInvalid
	assert.Contains(t, err.Error(), "signature is invalid", "Error message should indicate invalid signature")


	jwtKey = originalKey // Restore the correct key
}


func TestValidateToken_Expired(t *testing.T) {
	os.Setenv("JWT_TOKEN_LIFESPAN_HOURS", "-1") // Set lifespan to negative 1 hour
	InitializeJWT() // Re-initialize to pick up new lifespan

	userID := uuid.New()
	orgID := uuid.New()
	user := &models.User{ID: userID, Email: "expired@example.com", Role: models.RoleUser, OrganizationID: uuid.NullUUID{UUID: orgID, Valid: true}}

	tokenString, err := GenerateToken(user, user.OrganizationID.UUID)
	assert.NoError(t, err) // Token generation itself should be fine

	// Wait a tiny moment to ensure it's definitely past the "expiry" if clock skew is an issue
	// time.Sleep(50 * time.Millisecond)

	_, err = ValidateToken(tokenString)
	assert.Error(t, err)
	// Para jwt/v5, usamos errors.Is para verificar erros de token expirado
	if !errors.Is(err, jwt.ErrTokenExpired) {
		// O erro não é jwt.ErrTokenExpired como esperado.
		// Logar o erro real para diagnóstico.
		t.Errorf("Expected error to be or wrap jwt.ErrTokenExpired, but got %T: %v", err, err)
		// Forçar a falha do teste se não for o erro esperado.
		assert.True(t, false, "Error was not jwt.ErrTokenExpired")
	} else {
		// Se errors.Is(err, jwt.ErrTokenExpired) for verdadeiro, o teste está correto.
		assert.True(t, true, "Correctly identified jwt.ErrTokenExpired")
	}

	os.Setenv("JWT_TOKEN_LIFESPAN_HOURS", "1") // Reset lifespan for other tests
	InitializeJWT()
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup a test router with the middleware and a test handler
	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/testauth", func(c *gin.Context) {
		userID, exists := c.Get("userID")
		assert.True(t, exists)
		assert.NotNil(t, userID)
		c.Status(http.StatusOK)
	})

	// Case 1: No Authorization Header
	reqNoAuth, _ := http.NewRequest(http.MethodGet, "/testauth", nil)
	rrNoAuth := httptest.NewRecorder()
	router.ServeHTTP(rrNoAuth, reqNoAuth)
	assert.Equal(t, http.StatusUnauthorized, rrNoAuth.Code)
	assert.Contains(t, rrNoAuth.Body.String(), "Authorization header required")

	// Case 2: Malformed Authorization Header
	reqMalformed, _ := http.NewRequest(http.MethodGet, "/testauth", nil)
	reqMalformed.Header.Set("Authorization", "Bearer") // Missing token part
	rrMalformed := httptest.NewRecorder()
	router.ServeHTTP(rrMalformed, reqMalformed)
	assert.Equal(t, http.StatusUnauthorized, rrMalformed.Code)
	assert.Contains(t, rrMalformed.Body.String(), "Authorization header format must be Bearer {token}")

	// Case 3: Invalid Token (e.g., tampered or wrongly signed)
	reqInvalidToken, _ := http.NewRequest(http.MethodGet, "/testauth", nil)
	reqInvalidToken.Header.Set("Authorization", "Bearer aninvalidtokenstring")
	rrInvalidToken := httptest.NewRecorder()
	router.ServeHTTP(rrInvalidToken, reqInvalidToken)
	assert.Equal(t, http.StatusUnauthorized, rrInvalidToken.Code)
	assert.Contains(t, rrInvalidToken.Body.String(), "Invalid token")

	// Case 4: Valid Token
	userID := uuid.New()
	orgID := uuid.New()
	user := &models.User{ID: userID, Email: "authmiddleware@example.com", Role: models.RoleManager, OrganizationID: uuid.NullUUID{UUID: orgID, Valid: true}}
	validToken, _ := GenerateToken(user, user.OrganizationID)

	reqValid, _ := http.NewRequest(http.MethodGet, "/testauth", nil)
	reqValid.Header.Set("Authorization", "Bearer "+validToken)
	rrValid := httptest.NewRecorder()
	router.ServeHTTP(rrValid, reqValid)
	assert.Equal(t, http.StatusOK, rrValid.Code)
}
