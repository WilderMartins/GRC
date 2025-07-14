package oauth2auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"phoenixgrc/backend/pkg/config"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	"regexp"
	"strings"
	"testing"
	"time"
	"errors" // Adicionado para gorm.ErrRecordNotFound e outros erros customizados

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// mockDB configura um mock do GORM DB para os testes.
func mockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Falha ao abrir mock de conexão sql: %v", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Falha ao inicializar gorm com mock db: %v", err)
	}
	return gormDB, mock
}

func TestGoogleLoginHandler_OrgSpecific_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Configurar mock do DB e o serviço de banco de dados
	gormDB, dbMock := mockDB(t)
	database.SetDB(gormDB) // Configura o DB global usado pelos handlers

	// Configurações da aplicação (mock)
	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:8080"
	defer func() { config.Cfg = originalAppConfig }() // Restaurar config original

	idpID := uuid.New()
	orgID := uuid.New()

	oauthConfigJSON := GoogleOAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Scopes:       []string{"email", "profile"},
	}
	configBytes, _ := json.Marshal(oauthConfigJSON)

	idp := models.IdentityProvider{
		BaseModel:      models.BaseModel{ID: idpID},
		OrganizationID: orgID,
		Name:           "Test Google IdP",
		ProviderType:   models.IDPTypeOAuth2Google,
		ConfigJSON:     string(configBytes),
		IsActive:       true,
	}

	rows := sqlmock.NewRows([]string{"id", "organization_id", "name", "provider_type", "config_json", "is_active", "created_at", "updated_at", "deleted_at"}).
		AddRow(idp.ID, idp.OrganizationID, idp.Name, idp.ProviderType, idp.ConfigJSON, idp.IsActive, time.Now(), time.Now(), nil)

	// Espera a query SQL para buscar o IdentityProvider
	expectedSQL := `SELECT \* FROM "identity_providers" WHERE id = \$1 AND provider_type = \$2 AND is_active = \$3 AND "identity_providers"."deleted_at" IS NULL ORDER BY "identity_providers"."id" LIMIT \$4`
	dbMock.ExpectQuery(regexp.QuoteMeta(expectedSQL)).
		WithArgs(idpID.String(), models.IDPTypeOAuth2Google, true, 1). // GORM usa string para UUID em args
		WillReturnRows(rows)

	// Configurar Gin router
	router := gin.Default()
	router.GET("/auth/oauth2/google/:idpId/login", GoogleLoginHandler)

	// Criar requisição HTTP
	req, _ := http.NewRequest(http.MethodGet, "/auth/oauth2/google/"+idpID.String()+"/login", nil)
	rr := httptest.NewRecorder()

	// Executar handler
	router.ServeHTTP(rr, req)

	// Asserções
	assert.Equal(t, http.StatusFound, rr.Code)

	// Verificar cookie de estado
	cookies := rr.Result().Cookies()
	var stateCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == googleOAuthStateCookie {
			stateCookie = c
			break
		}
	}
	assert.NotNil(t, stateCookie, "Cookie de estado OAuth do Google não encontrado")
	assert.True(t, stateCookie.HttpOnly)
	assert.NotEmpty(t, stateCookie.Value)
	assert.WithinDuration(t, time.Now().Add(10*time.Minute), stateCookie.Expires, 10*time.Second, "Validade do cookie") // ~10 min
	assert.Equal(t, "/", stateCookie.Path)
	// Secure e SameSite dependem do ambiente de teste, mas idealmente seriam Strict/Lax e Secure=true

	// Verificar URL de redirecionamento
	location, err := rr.Result().Location()
	assert.NoError(t, err)
	assert.Contains(t, location.String(), "https://accounts.google.com/o/oauth2/auth")
	assert.Contains(t, location.String(), "client_id="+oauthConfigJSON.ClientID)
	assert.Contains(t, location.String(), "redirect_uri="+url.QueryEscape(config.Cfg.AppRootURL+"/auth/oauth2/google/"+idpID.String()+"/callback"))
	assert.Contains(t, location.String(), "scope="+url.QueryEscape(strings.Join(oauthConfigJSON.Scopes, " ")))
	assert.Contains(t, location.String(), "response_type=code")
	assert.Contains(t, location.String(), "state="+stateCookie.Value) // O estado na URL deve ser o mesmo do cookie

	// Verificar se todas as expectativas do mock DB foram atendidas
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestGoogleLoginHandler_Global_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Não precisamos mockar o DB para este caso, pois não deve haver consulta ao IdP
	gormDB, _ := mockDB(t) // Mock básico para evitar nil pointer se algo inesperado acontecer
	database.SetDB(gormDB)

	// Configurações da aplicação (mock)
	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:8080"
	config.Cfg.GoogleClientID = "global-google-client-id"
	config.Cfg.GoogleClientSecret = "global-google-client-secret" // Precisa estar presente para a config ser válida
	defer func() { config.Cfg = originalAppConfig }()

	// Configurar Gin router
	router := gin.Default()
	router.GET("/auth/oauth2/google/:idpId/login", GoogleLoginHandler)

	// Criar requisição HTTP
	req, _ := http.NewRequest(http.MethodGet, "/auth/oauth2/google/"+GlobalIdPIdentifier+"/login", nil)
	rr := httptest.NewRecorder()

	// Executar handler
	router.ServeHTTP(rr, req)

	// Asserções
	assert.Equal(t, http.StatusFound, rr.Code)

	// Verificar cookie de estado
	cookies := rr.Result().Cookies()
	var stateCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == googleOAuthStateCookie {
			stateCookie = c
			break
		}
	}
	assert.NotNil(t, stateCookie, "Cookie de estado OAuth do Google não encontrado")
	assert.True(t, stateCookie.HttpOnly)
	assert.NotEmpty(t, stateCookie.Value)

	// Verificar URL de redirecionamento
	location, err := rr.Result().Location()
	assert.NoError(t, err)
	assert.Contains(t, location.String(), "https://accounts.google.com/o/oauth2/auth")
	assert.Contains(t, location.String(), "client_id="+config.Cfg.GoogleClientID)
	expectedRedirectURI := config.Cfg.AppRootURL + "/auth/oauth2/google/" + GlobalIdPIdentifier + "/callback"
	assert.Contains(t, location.String(), "redirect_uri="+url.QueryEscape(expectedRedirectURI))
	// Scopes padrão para global: openid, profile, email
	assert.Contains(t, location.String(), "scope="+url.QueryEscape("openid profile email"))
	assert.Contains(t, location.String(), "response_type=code")
	assert.Contains(t, location.String(), "state="+stateCookie.Value)
}

// Helper para executar o handler e retornar o recorder e o router
func executeLoginRequest(t *testing.T, idpID string, handler gin.HandlerFunc, providerName string) *httptest.ResponseRecorder {
	router := gin.Default()
	router.GET("/auth/oauth2/"+providerName+"/:idpId/login", handler)

	req, err := http.NewRequest(http.MethodGet, "/auth/oauth2/"+providerName+"/"+idpID+"/login", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// Helper para verificar a resposta de erro JSON
func assertErrorResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatusCode int, expectedErrorMessageSubstring string) {
	assert.Equal(t, expectedStatusCode, rr.Code)
	var jsonResponse map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &jsonResponse)
	assert.NoError(t, err, "Falha ao decodificar resposta JSON de erro")
	assert.Contains(t, jsonResponse["error"], expectedErrorMessageSubstring)
}

func TestGoogleLoginHandler_OrgSpecific_IdPNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, dbMock := mockDB(t)
	database.SetDB(gormDB)
	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:8080" // Necessário para construção inicial do redirect URI
	defer func() { config.Cfg = originalAppConfig }()

	idpID := uuid.New()
	expectedSQL := `SELECT \* FROM "identity_providers" WHERE id = \$1 AND provider_type = \$2 AND is_active = \$3 AND "identity_providers"."deleted_at" IS NULL ORDER BY "identity_providers"."id" LIMIT \$4`
	dbMock.ExpectQuery(regexp.QuoteMeta(expectedSQL)).
		WithArgs(idpID.String(), models.IDPTypeOAuth2Google, true, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	rr := executeLoginRequest(t, idpID.String(), GoogleLoginHandler, "google")
	assertErrorResponse(t, rr, http.StatusNotFound, "Active Google OAuth2 provider configuration not found")
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestGoogleLoginHandler_OrgSpecific_DBError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, dbMock := mockDB(t)
	database.SetDB(gormDB)
	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:8080"
	defer func() { config.Cfg = originalAppConfig }()

	idpID := uuid.New()
	expectedSQL := `SELECT \* FROM "identity_providers" WHERE id = \$1 AND provider_type = \$2 AND is_active = \$3 AND "identity_providers"."deleted_at" IS NULL ORDER BY "identity_providers"."id" LIMIT \$4`
	dbMock.ExpectQuery(regexp.QuoteMeta(expectedSQL)).
		WithArgs(idpID.String(), models.IDPTypeOAuth2Google, true, 1).
		WillReturnError(errors.New("database communication error"))

	rr := executeLoginRequest(t, idpID.String(), GoogleLoginHandler, "google")
	assertErrorResponse(t, rr, http.StatusInternalServerError, "Database error fetching Google IdP config")
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestGoogleLoginHandler_OrgSpecific_InvalidJSONConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, dbMock := mockDB(t)
	database.SetDB(gormDB)
	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:8080"
	defer func() { config.Cfg = originalAppConfig }()

	idpID := uuid.New()
	orgID := uuid.New()
	idp := models.IdentityProvider{
		BaseModel:      models.BaseModel{ID: idpID},
		OrganizationID: orgID,
		ProviderType:   models.IDPTypeOAuth2Google,
		ConfigJSON:     `{"client_id": "abc", "client_secret": "def", malformed...}`, // JSON inválido
		IsActive:       true,
	}
	rows := sqlmock.NewRows([]string{"id", "config_json", "provider_type", "is_active"}).AddRow(idp.ID, idp.ConfigJSON, idp.ProviderType, idp.IsActive)
	expectedSQL := `SELECT \* FROM "identity_providers" WHERE id = \$1 AND provider_type = \$2 AND is_active = \$3 AND "identity_providers"."deleted_at" IS NULL ORDER BY "identity_providers"."id" LIMIT \$4`
	dbMock.ExpectQuery(regexp.QuoteMeta(expectedSQL)).
		WithArgs(idpID.String(), models.IDPTypeOAuth2Google, true, 1).
		WillReturnRows(rows)

	rr := executeLoginRequest(t, idpID.String(), GoogleLoginHandler, "google")
	assertErrorResponse(t, rr, http.StatusInternalServerError, "Failed to unmarshal Google OAuth2 config from JSON")
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestGoogleLoginHandler_OrgSpecific_MissingClientIDInJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, dbMock := mockDB(t)
	database.SetDB(gormDB)
	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:8080"
	defer func() { config.Cfg = originalAppConfig }()

	idpID := uuid.New()
	orgID := uuid.New()
	oauthCfgMissing := GoogleOAuthConfig{ClientSecret: "secret"} // Sem ClientID
	configBytes, _ := json.Marshal(oauthCfgMissing)
	idp := models.IdentityProvider{
		BaseModel:      models.BaseModel{ID: idpID},
		OrganizationID: orgID,
		ProviderType:   models.IDPTypeOAuth2Google,
		ConfigJSON:     string(configBytes),
		IsActive:       true,
	}
	rows := sqlmock.NewRows([]string{"id", "config_json", "provider_type", "is_active"}).AddRow(idp.ID, idp.ConfigJSON, idp.ProviderType, idp.IsActive)
	expectedSQL := `SELECT \* FROM "identity_providers" WHERE id = \$1 AND provider_type = \$2 AND is_active = \$3 AND "identity_providers"."deleted_at" IS NULL ORDER BY "identity_providers"."id" LIMIT \$4`
	dbMock.ExpectQuery(regexp.QuoteMeta(expectedSQL)).
		WithArgs(idpID.String(), models.IDPTypeOAuth2Google, true, 1).
		WillReturnRows(rows)

	rr := executeLoginRequest(t, idpID.String(), GoogleLoginHandler, "google")
	assertErrorResponse(t, rr, http.StatusInternalServerError, "client_id or client_secret missing")
	assert.NoError(t, dbMock.ExpectationsWereMet())
}


func TestGoogleLoginHandler_Global_NotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, _ := mockDB(t)
	database.SetDB(gormDB)
	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:8080"
	config.Cfg.GoogleClientID = "" // ClientID global não configurado
	config.Cfg.GoogleClientSecret = "secret"
	defer func() { config.Cfg = originalAppConfig }()

	rr := executeLoginRequest(t, GlobalIdPIdentifier, GoogleLoginHandler, "google")
	assertErrorResponse(t, rr, http.StatusInternalServerError, "global Google OAuth2 (GOOGLE_CLIENT_ID/SECRET) not configured")
}

func TestGoogleLoginHandler_AppRootURL_NotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, _ := mockDB(t) // Mock DB para evitar nil panic, embora não deva ser usado
	database.SetDB(gormDB)

	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "" // APP_ROOT_URL não configurado
	// Para IdP global
	config.Cfg.GoogleClientID = "global-id"
	config.Cfg.GoogleClientSecret = "global-secret"
	defer func() { config.Cfg = originalAppConfig }()


	rrGlobal := executeLoginRequest(t, GlobalIdPIdentifier, GoogleLoginHandler, "google")
	assertErrorResponse(t, rrGlobal, http.StatusInternalServerError, "APP_ROOT_URL) not initialized or empty")

	// Para IdP específico (simula que o DB retornaria algo, mas AppRootURL falha primeiro)
	// Não precisamos de mock do DB complexo aqui, pois a falha de AppRootURL é prioritária
	// e acontece antes da consulta ao DB na função getGoogleOAuthConfig.
	// No entanto, para ser mais preciso, a função getGoogleOAuthConfig é chamada.
	// Para simplificar, vamos assumir que a checagem de AppRootURL ocorre antes de qualquer lógica de DB.
	// Se a lógica fosse que o DB é consultado primeiro, este teste precisaria de mock de DB.
	// A implementação atual em getGoogleOAuthConfig verifica AppRootURL no início.
	idpID := uuid.New()
	rrOrg := executeLoginRequest(t, idpID.String(), GoogleLoginHandler, "google")
	assertErrorResponse(t, rrOrg, http.StatusInternalServerError, "APP_ROOT_URL) not initialized or empty")
}

// --- Testes para GithubLoginHandler ---

func TestGithubLoginHandler_OrgSpecific_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, dbMock := mockDB(t)
	database.SetDB(gormDB)
	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:8080"
	defer func() { config.Cfg = originalAppConfig }()

	idpID := uuid.New()
	orgID := uuid.New()
	oauthConfigJSON := GithubOAuthConfig{ // Usar GithubOAuthConfig
		ClientID:     "test-github-client-id",
		ClientSecret: "test-github-client-secret",
		Scopes:       []string{"read:user", "user:email"},
	}
	configBytes, _ := json.Marshal(oauthConfigJSON)
	idp := models.IdentityProvider{
		BaseModel:      models.BaseModel{ID: idpID},
		OrganizationID: orgID,
		Name:           "Test Github IdP",
		ProviderType:   models.IDPTypeOAuth2Github, // Mudar para Github
		ConfigJSON:     string(configBytes),
		IsActive:       true,
	}
	rows := sqlmock.NewRows([]string{"id", "organization_id", "name", "provider_type", "config_json", "is_active"}).
		AddRow(idp.ID, idp.OrganizationID, idp.Name, idp.ProviderType, idp.ConfigJSON, idp.IsActive)
	expectedSQL := `SELECT \* FROM "identity_providers" WHERE id = \$1 AND provider_type = \$2 AND is_active = \$3 AND "identity_providers"."deleted_at" IS NULL ORDER BY "identity_providers"."id" LIMIT \$4`
	dbMock.ExpectQuery(regexp.QuoteMeta(expectedSQL)).
		WithArgs(idpID.String(), models.IDPTypeOAuth2Github, true, 1). // Mudar para Github
		WillReturnRows(rows)

	rr := executeLoginRequest(t, idpID.String(), GithubLoginHandler, "github") // Usar GithubLoginHandler

	assert.Equal(t, http.StatusFound, rr.Code)
	cookies := rr.Result().Cookies()
	var stateCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == githubOAuthStateCookie { // Usar githubOAuthStateCookie
			stateCookie = c
			break
		}
	}
	assert.NotNil(t, stateCookie, "Cookie de estado OAuth do Github não encontrado")
	assert.True(t, stateCookie.HttpOnly)

	location, err := rr.Result().Location()
	assert.NoError(t, err)
	assert.Contains(t, location.String(), "https://github.com/login/oauth/authorize") // Endpoint do Github
	assert.Contains(t, location.String(), "client_id="+oauthConfigJSON.ClientID)
	assert.Contains(t, location.String(), "redirect_uri="+url.QueryEscape(config.Cfg.AppRootURL+"/auth/oauth2/github/"+idpID.String()+"/callback"))
	assert.Contains(t, location.String(), "scope="+url.QueryEscape(strings.Join(oauthConfigJSON.Scopes, " ")))
	assert.Contains(t, location.String(), "state="+stateCookie.Value)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestGithubLoginHandler_Global_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, _ := mockDB(t)
	database.SetDB(gormDB)
	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:8080"
	config.Cfg.GithubClientID = "global-github-client-id"     // Usar GithubClientID
	config.Cfg.GithubClientSecret = "global-github-client-secret" // Usar GithubClientSecret
	defer func() { config.Cfg = originalAppConfig }()

	rr := executeLoginRequest(t, GlobalIdPIdentifier, GithubLoginHandler, "github") // Usar GithubLoginHandler

	assert.Equal(t, http.StatusFound, rr.Code)
	cookies := rr.Result().Cookies()
	var stateCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == githubOAuthStateCookie { // Usar githubOAuthStateCookie
			stateCookie = c
			break
		}
	}
	assert.NotNil(t, stateCookie, "Cookie de estado OAuth do Github não encontrado")

	location, err := rr.Result().Location()
	assert.NoError(t, err)
	assert.Contains(t, location.String(), "https://github.com/login/oauth/authorize") // Endpoint do Github
	assert.Contains(t, location.String(), "client_id="+config.Cfg.GithubClientID)
	expectedRedirectURI := config.Cfg.AppRootURL + "/auth/oauth2/github/" + GlobalIdPIdentifier + "/callback"
	assert.Contains(t, location.String(), "redirect_uri="+url.QueryEscape(expectedRedirectURI))
	// Scopes padrão para github global: read:user, user:email
	assert.Contains(t, location.String(), "scope="+url.QueryEscape("read:user user:email"))
	assert.Contains(t, location.String(), "state="+stateCookie.Value)
}

func TestGithubLoginHandler_OrgSpecific_IdPNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, dbMock := mockDB(t)
	database.SetDB(gormDB)
	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:8080"
	defer func() { config.Cfg = originalAppConfig }()

	idpID := uuid.New()
	expectedSQL := `SELECT \* FROM "identity_providers" WHERE id = \$1 AND provider_type = \$2 AND is_active = \$3 AND "identity_providers"."deleted_at" IS NULL ORDER BY "identity_providers"."id" LIMIT \$4`
	dbMock.ExpectQuery(regexp.QuoteMeta(expectedSQL)).
		WithArgs(idpID.String(), models.IDPTypeOAuth2Github, true, 1). // Github
		WillReturnError(gorm.ErrRecordNotFound)

	rr := executeLoginRequest(t, idpID.String(), GithubLoginHandler, "github")
	assertErrorResponse(t, rr, http.StatusNotFound, "Active Github OAuth2 provider configuration not found") // Github
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestGithubLoginHandler_OrgSpecific_DBError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, dbMock := mockDB(t)
	database.SetDB(gormDB)
	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:8080"
	defer func() { config.Cfg = originalAppConfig }()

	idpID := uuid.New()
	expectedSQL := `SELECT \* FROM "identity_providers" WHERE id = \$1 AND provider_type = \$2 AND is_active = \$3 AND "identity_providers"."deleted_at" IS NULL ORDER BY "identity_providers"."id" LIMIT \$4`
	dbMock.ExpectQuery(regexp.QuoteMeta(expectedSQL)).
		WithArgs(idpID.String(), models.IDPTypeOAuth2Github, true, 1). // Github
		WillReturnError(errors.New("github db error"))

	rr := executeLoginRequest(t, idpID.String(), GithubLoginHandler, "github")
	assertErrorResponse(t, rr, http.StatusInternalServerError, "Database error fetching Github IdP config") // Github
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestGithubLoginHandler_OrgSpecific_InvalidJSONConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, dbMock := mockDB(t)
	database.SetDB(gormDB)
	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:8080"
	defer func() { config.Cfg = originalAppConfig }()

	idpID := uuid.New()
	idp := models.IdentityProvider{
		BaseModel:    models.BaseModel{ID: idpID},
		ProviderType: models.IDPTypeOAuth2Github, // Github
		ConfigJSON:   `{invalid}`,
		IsActive:     true,
	}
	rows := sqlmock.NewRows([]string{"id", "config_json", "provider_type", "is_active"}).AddRow(idp.ID, idp.ConfigJSON, idp.ProviderType, idp.IsActive)
	expectedSQL := `SELECT \* FROM "identity_providers" WHERE id = \$1 AND provider_type = \$2 AND is_active = \$3 AND "identity_providers"."deleted_at" IS NULL ORDER BY "identity_providers"."id" LIMIT \$4`
	dbMock.ExpectQuery(regexp.QuoteMeta(expectedSQL)).
		WithArgs(idpID.String(), models.IDPTypeOAuth2Github, true, 1). // Github
		WillReturnRows(rows)

	rr := executeLoginRequest(t, idpID.String(), GithubLoginHandler, "github")
	assertErrorResponse(t, rr, http.StatusInternalServerError, "Failed to unmarshal Github OAuth2 config from JSON") // Github
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestGithubLoginHandler_OrgSpecific_MissingClientIDInJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, dbMock := mockDB(t)
	database.SetDB(gormDB)
	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:8080"
	defer func() { config.Cfg = originalAppConfig }()

	idpID := uuid.New()
	oauthCfgMissing := GithubOAuthConfig{ClientSecret: "secret"} // Sem ClientID
	configBytes, _ := json.Marshal(oauthCfgMissing)
	idp := models.IdentityProvider{
		BaseModel:    models.BaseModel{ID: idpID},
		ProviderType: models.IDPTypeOAuth2Github, // Github
		ConfigJSON:   string(configBytes),
		IsActive:     true,
	}
	rows := sqlmock.NewRows([]string{"id", "config_json", "provider_type", "is_active"}).AddRow(idp.ID, idp.ConfigJSON, idp.ProviderType, idp.IsActive)
	expectedSQL := `SELECT \* FROM "identity_providers" WHERE id = \$1 AND provider_type = \$2 AND is_active = \$3 AND "identity_providers"."deleted_at" IS NULL ORDER BY "identity_providers"."id" LIMIT \$4`
	dbMock.ExpectQuery(regexp.QuoteMeta(expectedSQL)).
		WithArgs(idpID.String(), models.IDPTypeOAuth2Github, true, 1). // Github
		WillReturnRows(rows)

	rr := executeLoginRequest(t, idpID.String(), GithubLoginHandler, "github")
	assertErrorResponse(t, rr, http.StatusInternalServerError, "client_id or client_secret missing") // Github
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestGithubLoginHandler_Global_NotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, _ := mockDB(t)
	database.SetDB(gormDB)
	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:8080"
	config.Cfg.GithubClientID = "" // GithubClientID global não configurado
	config.Cfg.GithubClientSecret = "secret"
	defer func() { config.Cfg = originalAppConfig }()

	rr := executeLoginRequest(t, GlobalIdPIdentifier, GithubLoginHandler, "github")
	assertErrorResponse(t, rr, http.StatusInternalServerError, "global Github OAuth2 (GITHUB_CLIENT_ID/SECRET) not configured") // Github
}

// Teste para AppRootURL não configurado já é coberto pelo TestGoogleLoginHandler_AppRootURL_NotConfigured
// pois a lógica de falha em get<Provider>OAuthConfig é a mesma para ambos.
// Se quisermos um teste explícito para Github:
func TestGithubLoginHandler_AppRootURL_NotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, _ := mockDB(t)
	database.SetDB(gormDB)

	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "" // APP_ROOT_URL não configurado
	config.Cfg.GithubClientID = "global-id" // Configurar para o caso global
	config.Cfg.GithubClientSecret = "global-secret"
	defer func() { config.Cfg = originalAppConfig }()

	rrGlobal := executeLoginRequest(t, GlobalIdPIdentifier, GithubLoginHandler, "github")
	assertErrorResponse(t, rrGlobal, http.StatusInternalServerError, "APP_ROOT_URL) not initialized or empty")

	idpID := uuid.New()
	rrOrg := executeLoginRequest(t, idpID.String(), GithubLoginHandler, "github")
	assertErrorResponse(t, rrOrg, http.StatusInternalServerError, "APP_ROOT_URL) not initialized or empty")
}
