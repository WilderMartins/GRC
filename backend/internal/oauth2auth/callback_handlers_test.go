package oauth2auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"phoenixgrc/backend/internal/auth"
	"phoenixgrc/backend/internal/config"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/oauth2"
	googleAPI "google.golang.org/api/oauth2/v2" // Para mock de UserInfo
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// mockTokenExchangeServer cria um servidor HTTP de teste para simular o endpoint de troca de token OAuth2.
func mockTokenExchangeServer(t *testing.T, expectedTokenResponse map[string]interface{}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		r.Body.Close()
		r.Body = io.NopCloser(bytes.NewBuffer(body)) // Repopulate Body

		// Verificações básicas do corpo do request, se necessário
		// Ex: r.ParseForm(); assert.Equal(t, "authorization_code", r.FormValue("grant_type"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedTokenResponse)
	}))
}

// mockUserInfoServer cria um servidor HTTP de teste para simular o endpoint de informações do usuário OAuth2.
func mockUserInfoServer(t *testing.T, expectedUserInfoResponse interface{}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verifica o token de autorização, se necessário
		// authHeader := r.Header.Get("Authorization")
		// assert.True(t, strings.HasPrefix(authHeader, "Bearer "), "Token de bearer ausente ou malformado")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedUserInfoResponse)
	}))
}

// Funções mockDB e assertErrorResponse podem ser movidas para um test_utils.go se usadas em múltiplos arquivos de teste.
// Por enquanto, vou duplicar/adaptar se necessário.
func mockGormDBForCallbackTests(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Falha ao abrir mock de conexão sql: %v", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{
		// Logger: logger.Default.LogMode(logger.Info), // Para debug
	})
	if err != nil {
		t.Fatalf("Falha ao inicializar gorm com mock db: %v", err)
	}
	return gormDB, mock
}


// Mock para auth.GenerateToken
var mockGenerateToken func(user *models.User, organizationID uuid.NullUUID) (string, error)

type authMocker struct{}

func (m *authMocker) GenerateToken(user *models.User, organizationID uuid.NullUUID) (string, error) {
	if mockGenerateToken != nil {
		return mockGenerateToken(user, organizationID)
	}
	return "mocked_jwt_token_default", nil // Default mock
}

// Mock para googleAPI.NewService e Userinfo.Get().Do()
var mockGoogleNewService func(ctx context.Context, opts ...option.ClientOption) (*googleAPI.Service, error)
var mockGoogleUserinfoGetDo func(call *googleAPI.UserinfoGetCall) (*googleAPI.Userinfoplus, error)

// Salvar originais para restaurar
var originalGoogleNewService func(ctx context.Context, opts ...option.ClientOption) (*googleAPI.Service, error)
var originalGoogleUserinfoGetDo func(call *googleAPI.UserinfoGetCall) (*googleAPI.Userinfoplus, error)

func setupGoogleAPIMocks(userInfoResponse *googleAPI.Userinfoplus, expectedError error) {
	originalGoogleNewService = googleAPINewService // Supondo que googleAPINewService é o nome real
	googleAPINewService = func(ctx context.Context, opts ...option.ClientOption) (*googleAPI.Service, error) {
		// Podemos verificar opts se necessário, e.g. se um client específico foi passado.
		// Por agora, apenas retornamos um mock do serviço.
		return &googleAPI.Service{
			Userinfo: googleAPI.NewUserinfoService(&googleAPI.Service{}), // Mock aninhado
		}, nil
	}

	originalGoogleUserinfoGetDo = googleAPIUserinfoGetDo // Supondo que googleAPIUserinfoGetDo é o nome real
	googleAPIUserinfoGetDo = func(call *googleAPI.UserinfoGetCall) (*googleAPI.Userinfoplus, error) {
		if expectedError != nil {
			return nil, expectedError
		}
		return userInfoResponse, nil
	}
}

func restoreGoogleAPIMocks() {
	googleAPINewService = originalGoogleNewService
	googleAPIUserinfoGetDo = originalGoogleUserinfoGetDo
}

// Para que isso funcione, precisaremos mudar as chamadas no handler para usar essas vars:
// Em vez de: oauth2Service, err := googleAPI.NewService(...) -> oauth2Service, err := googleAPINewService(...)
// Em vez de: userInfo, err := oauth2Service.Userinfo.Get().Do() -> userInfo, err := googleAPIUserinfoGetDo(oauth2Service.Userinfo.Get())
// Isso é invasivo. Uma alternativa é usar uma interface wrapper para o cliente Google.

// Abordagem mais simples por agora: assumir que o http client default é interceptável
// ou que a configuração do TokenURL no oauth2.Config também afeta as chamadas de API subsequentes
// se o mesmo http.Client for reutilizado. A biblioteca `golang.org/x/oauth2` usa o client
// do contexto ou um client default para fazer a chamada de token. O client retornado por
// `oauthCfg.Client(context.Background(), token)` é então usado para as chamadas de API.
// Se pudermos fazer esse client acertar nosso mockUserInfoServer, seria ideal.

// Vamos tentar uma abordagem diferente: mockar o http.DefaultClient temporariamente para o UserInfo.
// Isso é arriscado e pode afetar outros testes se não for bem isolado.

// mockRoundTripper é um http.RoundTripper que intercepta chamadas para uma URL específica.
type mockRoundTripper struct {
	originalTransport http.RoundTripper
	mockTargetURL     string
	mockServerURL     string // URL do nosso servidor de mock (userInfoServer)
}

func (mrt *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasPrefix(req.URL.String(), mrt.mockTargetURL) {
		// Redirecionar para o nosso mock server
		// Preservar path e query params se necessário, mas para UserInfo geralmente é simples
		newURL, _ := url.Parse(mrt.mockServerURL)
		req.URL.Scheme = newURL.Scheme
		req.URL.Host = newURL.Host
		req.Host = newURL.Host // Importante para o servidor de teste
		// req.URL.Path = ... se o path do mock server for diferente
	}
	return mrt.originalTransport.RoundTrip(req)
}


func TestGoogleCallbackHandler_OrgSpecific_NewUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, dbMock := mockGormDBForCallbackTests(t)
	database.SetDB(gormDB)

	// 1. Configurar mock do servidor OAuth2
	expectedToken := map[string]interface{}{
		"access_token": "test_access_token",
		"token_type":   "Bearer",
		"expires_in":   3600,
	}
	tokenServer := mockTokenExchangeServer(t, expectedToken)
	defer tokenServer.Close()

	expectedUserInfo := googleAPI.Userinfoplus{
		Email: "new.user@example.com",
		Name:  "New User",
		Id:    "google-user-id-123",
	}
	userInfoServer := mockUserInfoServer(t, expectedUserInfo)
	defer userInfoServer.Close()

	// Mock do transporte HTTP para redirecionar chamadas de UserInfo
	originalTransport := http.DefaultClient.Transport
	if originalTransport == nil {
		originalTransport = http.DefaultTransport
	}
	http.DefaultClient.Transport = &mockRoundTripper{
		originalTransport: originalTransport,
		mockTargetURL:     "https://www.googleapis.com/oauth2/v2/userinfo", // URL real do Google UserInfo
		mockServerURL:     userInfoServer.URL,                               // Nosso mock server
	}
	defer func() { http.DefaultClient.Transport = originalTransport }() // Restaurar

	// Sobrescrever TokenURL do Google para apontar para nosso mock tokenServer
	originalGoogleEndpoint := oauth2.Google.Endpoint // Usar oauth2.Google diretamente
	oauth2.Google.Endpoint.TokenURL = tokenServer.URL
	defer func() { oauth2.Google.Endpoint.TokenURL = originalGoogleEndpoint.TokenURL }()


	// Configurações da aplicação
	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:3000" // Frontend para redirect
	config.Cfg.JWTSecret = "test-jwt-secret-for-token-generation-32chars"
	config.Cfg.JWTTokenLifespan = 1 * time.Hour
	defer func() { config.Cfg = originalAppConfig }()

	idpID := uuid.New()
	orgID := uuid.New()
	idpName := "Test Google IdP Org"

	oauthIdPConfig := GoogleOAuthConfig{
		ClientID:     "org-google-client-id",
		ClientSecret: "org-google-client-secret",
	}
	idpConfigBytes, _ := json.Marshal(oauthIdPConfig)
	idp := models.IdentityProvider{
		BaseModel:      models.BaseModel{ID: idpID},
		OrganizationID: orgID,
		Name:           idpName,
		ProviderType:   models.IDPTypeOAuth2Google,
		ConfigJSON:     string(idpConfigBytes),
		IsActive:       true,
	}

	// Mock para auth.GenerateToken
	originalGenerateToken := auth.GenerateToken // Salvar a função original
	auth.GenerateToken = func(user *models.User, organizationID uuid.NullUUID) (string, error) {
		assert.Equal(t, expectedUserInfo.Email, user.Email)
		assert.Equal(t, orgID, organizationID.UUID) // Deve ser o orgID do IdP
		return "generated_jwt_token_for_new_user", nil
	}
	defer func() { auth.GenerateToken = originalGenerateToken }() // Restaurar

	// 2. Mocks do DB
	// Para getGoogleOAuthConfig no callback
	dbMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "identity_providers" WHERE id = $1 AND provider_type = $2 AND is_active = $3`)).
		WithArgs(idpID.String(), models.IDPTypeOAuth2Google, true).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "name", "provider_type", "config_json", "is_active"}).
			AddRow(idp.ID, idp.OrganizationID, idp.Name, idp.ProviderType, idp.ConfigJSON, idp.IsActive))

	// Para busca de usuário (não encontrado)
	dbMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE (email = $1 AND organization_id = $2) OR (social_login_id = $3 AND organization_id = $4)`)).
		WithArgs(expectedUserInfo.Email, orgID, expectedUserInfo.Id, orgID).
		WillReturnError(gorm.ErrRecordNotFound)

	// Para criação do usuário
	dbMock.ExpectBegin()
	// A ordem e a quantidade exata de colunas depende da struct models.User e como o GORM a mapeia.
	// Vamos focar nos campos principais. O GORM pode omitir colunas com valor zero/nulo dependendo da config.
	// É importante que o mock corresponda à query real gerada pelo GORM.
	// Se a query for complexa, pode ser necessário logá-la durante um teste real para ajustar o mock.
	// Por agora, vamos assumir uma ordem comum e que os campos não nulos são incluídos.
	// O GORM também pode usar `DEFAULT` para timestamps, então eles podem não estar na query INSERT.
	// Se `id` é gerado pelo DB, ele não estaria no INSERT, mas pode estar no RETURNING.
	// O sqlmock não tem um bom suporte para RETURNING em ExpectQuery, é melhor com ExpectExec.
	// Se GORM usa Exec para INSERT sem RETURNING e depois um SELECT, o mock muda.
	// Assumindo que GORM usa Exec para INSERT e não esperamos RETURNING aqui.
	dbMock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "users" ("organization_id","name","email","password_hash","sso_provider","social_login_id","role","is_active","id","created_at","updated_at","deleted_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`)).
		WithArgs(
			idp.OrganizationID,        // organization_id
			expectedUserInfo.Name,     // name
			expectedUserInfo.Email,    // email
			"OAUTH2_USER_NO_PASSWORD", // password_hash
			idp.Name,                  // sso_provider
			expectedUserInfo.Id,       // social_login_id
			models.RoleUser,           // role
			true,                      // is_active
			sqlmock.AnyArg(),          // id (gerado pelo GORM/DB)
			sqlmock.AnyArg(),          // created_at
			sqlmock.AnyArg(),          // updated_at
			nil,                       // deleted_at (deve ser NULL)
		).
		WillReturnResult(sqlmock.NewResult(1, 1)) // 1 linha afetada
	dbMock.ExpectCommit()


	// 3. Configurar e executar o handler
	router := gin.Default()
	router.GET("/auth/oauth2/google/:idpId/callback", GoogleCallbackHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/auth/oauth2/google/"+idpID.String()+"/callback?code=testauthcode&state=teststate", nil)
	req.AddCookie(&http.Cookie{Name: googleOAuthStateCookie, Value: "teststate", HttpOnly: true})

	router.ServeHTTP(w, req)

	// 4. Asserções
	assert.Equal(t, http.StatusFound, w.Code)

	// Verificar redirect URL
	redirectURL, err := w.Result().Location()
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(redirectURL.String(), config.Cfg.AppRootURL+"/oauth2/callback"))
	q := redirectURL.Query()
	assert.Equal(t, "generated_jwt_token_for_new_user", q.Get("token"))
	assert.Equal(t, "true", q.Get("sso_success"))
	assert.Equal(t, "google", q.Get("provider"))

	// Verificar se o cookie de estado foi limpo
	foundStateCookie := false
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == googleOAuthStateCookie {
			assert.LessOrEqual(t, cookie.MaxAge, 0, "Cookie de estado não foi limpo")
			foundStateCookie = true
			break
		}
	}
	// Se o cookie foi explicitamente setado com MaxAge < 0, ele pode não estar na lista de cookies do Result().
	// A verificação de que ele é setado com MaxAge -1 no handler é importante.
	// Aqui podemos apenas verificar que ele não está mais com valor.

	assert.NoError(t, dbMock.ExpectationsWereMet(), "Expectativas do DB não foram atendidas")

	// TODO: Verificar se o usuário foi criado no DB com os dados corretos.
	// Isso requer capturar os argumentos do dbMock.ExpectQuery para o INSERT.
	// Por exemplo, usando sqlmock.AnyArg() ou combinadores de argumentos.
}


func TestGoogleCallbackHandler_InvalidState(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.Default()
	router.GET("/auth/oauth2/google/:idpId/callback", GoogleCallbackHandler)

	w := httptest.NewRecorder()
	// idpId pode ser qualquer UUID válido, não será usado se o estado falhar primeiro
	idpID := uuid.New().String()
	req, _ := http.NewRequest(http.MethodGet, "/auth/oauth2/google/"+idpID+"/callback?code=anycode&state=querystate", nil)
	req.AddCookie(&http.Cookie{Name: googleOAuthStateCookie, Value: "cookiestate", HttpOnly: true})

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var jsonResponse map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &jsonResponse)
	assert.NoError(t, err)
	assert.Contains(t, jsonResponse["error"], "Invalid OAuth state")

	// Verificar se o cookie de estado foi limpo
	cookieFoundAndCleared := false
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == googleOAuthStateCookie {
			assert.LessOrEqual(t, cookie.MaxAge, 0, "Cookie de estado não foi limpo após estado inválido")
			cookieFoundAndCleared = true // Mesmo que MaxAge seja < 0, o cookie ainda pode estar presente no header
			break
		}
	}
	// Se o cookie é setado para expirar, ele ainda estará no header Set-Cookie.
	// A asserção importante é que MaxAge <= 0.
	// Se o cookie não estiver presente no Set-Cookie (porque foi setado com MaxAge < 0 e alguns servidores/clientes o omitem),
	// isso também é aceitável. O importante é que ele não seja mais válido.
	// Para este teste, a presença com MaxAge <= 0 é uma boa verificação.
	// Se o handler não setar o cookie de forma alguma no erro, a flag cookieFoundAndCleared será false.
	// O comportamento atual do handler é setar o cookie com MaxAge = -1.
	assert.True(t, cookieFoundAndCleared, "Set-Cookie para limpar o cookie de estado não foi encontrado")
}

func TestGoogleCallbackHandler_EmailNotProvided(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, dbMock := mockGormDBForCallbackTests(t)
	database.SetDB(gormDB)

	expectedToken := map[string]interface{}{"access_token": "valid_token", "token_type": "Bearer"}
	tokenServer := mockTokenExchangeServer(t, expectedToken)
	defer tokenServer.Close()

	// UserInfo sem email
	expectedUserInfo := googleAPI.Userinfoplus{
		Name: "No Email User",
		Id:   "google-user-no-email",
		// Email: "", // Omitido ou vazio
	}
	userInfoServer := mockUserInfoServer(t, expectedUserInfo)
	defer userInfoServer.Close()

	originalTransport := http.DefaultClient.Transport
	if originalTransport == nil {originalTransport = http.DefaultTransport}
	http.DefaultClient.Transport = &mockRoundTripper{
		originalTransport: originalTransport,
		mockTargetURL:     "https://www.googleapis.com/oauth2/v2/userinfo",
		mockServerURL:     userInfoServer.URL,
	}
	defer func() { http.DefaultClient.Transport = originalTransport }()

	originalGoogleEndpoint := oauth2.Google.Endpoint
	oauth2.Google.Endpoint.TokenURL = tokenServer.URL
	defer func() { oauth2.Google.Endpoint.TokenURL = originalGoogleEndpoint.TokenURL }()

	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:3000"
	defer func() { config.Cfg = originalAppConfig }()

	idpID := uuid.New()
	idp := models.IdentityProvider{
		BaseModel:    models.BaseModel{ID: idpID},
		ProviderType: models.IDPTypeOAuth2Google,
		ConfigJSON:   `{"client_id":"id","client_secret":"secret"}`,
		IsActive:     true,
	}
	dbMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "identity_providers"`)).
		WithArgs(idpID.String(), models.IDPTypeOAuth2Google, true).
		WillReturnRows(sqlmock.NewRows([]string{"id", "provider_type", "config_json", "is_active"}).
			AddRow(idp.ID, idp.ProviderType, idp.ConfigJSON, idp.IsActive))

	router := gin.Default()
	router.GET("/auth/oauth2/google/:idpId/callback", GoogleCallbackHandler)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/auth/oauth2/google/"+idpID.String()+"/callback?code=validcode&state=teststate", nil)
	req.AddCookie(&http.Cookie{Name: googleOAuthStateCookie, Value: "teststate"})

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var jsonResponse map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &jsonResponse)
	assert.NoError(t, err)
	assert.Contains(t, jsonResponse["error"], "Email not provided by Google")
	assert.NoError(t, dbMock.ExpectationsWereMet())

	cookieFoundAndCleared := false
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == googleOAuthStateCookie {
			assert.LessOrEqual(t, cookie.MaxAge, 0)
			cookieFoundAndCleared = true
			break
		}
	}
	assert.True(t, cookieFoundAndCleared, "Set-Cookie para limpar o cookie de estado não foi encontrado")
}

func TestGoogleCallbackHandler_UserInfoFetchFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, dbMock := mockGormDBForCallbackTests(t)
	database.SetDB(gormDB)

	expectedToken := map[string]interface{}{"access_token": "valid_token", "token_type": "Bearer"}
	tokenServer := mockTokenExchangeServer(t, expectedToken)
	defer tokenServer.Close()

	// UserInfo server retorna erro
	userInfoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "provider internal error"})
	}))
	defer userInfoServer.Close()

	originalTransport := http.DefaultClient.Transport
	if originalTransport == nil {originalTransport = http.DefaultTransport}
	http.DefaultClient.Transport = &mockRoundTripper{
		originalTransport: originalTransport,
		mockTargetURL:     "https://www.googleapis.com/oauth2/v2/userinfo",
		mockServerURL:     userInfoServer.URL,
	}
	defer func() { http.DefaultClient.Transport = originalTransport }()

	originalGoogleEndpoint := oauth2.Google.Endpoint
	oauth2.Google.Endpoint.TokenURL = tokenServer.URL
	defer func() { oauth2.Google.Endpoint.TokenURL = originalGoogleEndpoint.TokenURL }()

	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:3000"
	defer func() { config.Cfg = originalAppConfig }()

	idpID := uuid.New()
	idp := models.IdentityProvider{
		BaseModel:    models.BaseModel{ID: idpID},
		ProviderType: models.IDPTypeOAuth2Google,
		ConfigJSON:   `{"client_id":"id","client_secret":"secret"}`,
		IsActive:     true,
	}
	dbMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "identity_providers"`)).
		WithArgs(idpID.String(), models.IDPTypeOAuth2Google, true).
		WillReturnRows(sqlmock.NewRows([]string{"id", "provider_type", "config_json", "is_active"}).
			AddRow(idp.ID, idp.ProviderType, idp.ConfigJSON, idp.IsActive))

	router := gin.Default()
	router.GET("/auth/oauth2/google/:idpId/callback", GoogleCallbackHandler)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/auth/oauth2/google/"+idpID.String()+"/callback?code=validcode&state=teststate", nil)
	req.AddCookie(&http.Cookie{Name: googleOAuthStateCookie, Value: "teststate"})

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var jsonResponse map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &jsonResponse)
	assert.NoError(t, err)
	assert.Contains(t, jsonResponse["error"], "Failed to get user info from Google")
	assert.NoError(t, dbMock.ExpectationsWereMet())

	cookieFoundAndCleared := false
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == googleOAuthStateCookie {
			assert.LessOrEqual(t, cookie.MaxAge, 0)
			cookieFoundAndCleared = true
			break
		}
	}
	assert.True(t, cookieFoundAndCleared, "Set-Cookie para limpar o cookie de estado não foi encontrado")
}

func TestGoogleCallbackHandler_TokenExchangeFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, dbMock := mockGormDBForCallbackTests(t)
	database.SetDB(gormDB)

	// Mock do servidor de token para retornar erro
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest) // Simular erro do provedor OAuth
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
	}))
	defer tokenServer.Close()

	originalGoogleEndpoint := oauth2.Google.Endpoint
	oauth2.Google.Endpoint.TokenURL = tokenServer.URL
	defer func() { oauth2.Google.Endpoint.TokenURL = originalGoogleEndpoint.TokenURL }()

	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:3000"
	defer func() { config.Cfg = originalAppConfig }()

	idpID := uuid.New()
	orgID := uuid.New()
	oauthIdPConfig := GoogleOAuthConfig{ClientID: "id", ClientSecret: "secret"}
	idpConfigBytes, _ := json.Marshal(oauthIdPConfig)
	idp := models.IdentityProvider{
		BaseModel:      models.BaseModel{ID: idpID},
		OrganizationID: orgID,
		ProviderType:   models.IDPTypeOAuth2Google,
		ConfigJSON:     string(idpConfigBytes),
		IsActive:       true,
	}
	dbMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "identity_providers"`)).
		WithArgs(idpID.String(), models.IDPTypeOAuth2Google, true).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "provider_type", "config_json", "is_active"}).
			AddRow(idp.ID, idp.OrganizationID, idp.ProviderType, idp.ConfigJSON, idp.IsActive))

	router := gin.Default()
	router.GET("/auth/oauth2/google/:idpId/callback", GoogleCallbackHandler)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/auth/oauth2/google/"+idpID.String()+"/callback?code=validcode&state=teststate", nil)
	req.AddCookie(&http.Cookie{Name: googleOAuthStateCookie, Value: "teststate"})

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var jsonResponse map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &jsonResponse)
	assert.NoError(t, err)
	assert.Contains(t, jsonResponse["error"], "Failed to exchange OAuth code for token")
	assert.NoError(t, dbMock.ExpectationsWereMet())

	// Verificar limpeza do cookie
	cookieFoundAndCleared := false
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == googleOAuthStateCookie {
			assert.LessOrEqual(t, cookie.MaxAge, 0)
			cookieFoundAndCleared = true
			break
		}
	}
	assert.True(t, cookieFoundAndCleared, "Set-Cookie para limpar o cookie de estado não foi encontrado")
}

func TestGoogleCallbackHandler_MissingAuthCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.Default()
	router.GET("/auth/oauth2/google/:idpId/callback", GoogleCallbackHandler)

	w := httptest.NewRecorder()
	idpID := uuid.New().String()
	// O state na query corresponde ao cookie, mas o 'code' está ausente
	req, _ := http.NewRequest(http.MethodGet, "/auth/oauth2/google/"+idpID+"/callback?state=teststate", nil)
	req.AddCookie(&http.Cookie{Name: googleOAuthStateCookie, Value: "teststate", HttpOnly: true})

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var jsonResponse map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &jsonResponse)
	assert.NoError(t, err)
	assert.Contains(t, jsonResponse["error"], "OAuth authorization code not found or access denied")

	// Verificar se o cookie de estado foi limpo
	cookieFoundAndCleared := false
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == googleOAuthStateCookie {
			assert.LessOrEqual(t, cookie.MaxAge, 0)
			cookieFoundAndCleared = true
			break
		}
	}
	assert.True(t, cookieFoundAndCleared, "Set-Cookie para limpar o cookie de estado não foi encontrado")
}

func TestGoogleCallbackHandler_OrgSpecific_ExistingUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, dbMock := mockGormDBForCallbackTests(t)
	database.SetDB(gormDB)

	expectedToken := map[string]interface{}{"access_token": "test_access_token_exist", "token_type": "Bearer"}
	tokenServer := mockTokenExchangeServer(t, expectedToken)
	defer tokenServer.Close()

	expectedUserInfo := googleAPI.Userinfoplus{
		Email: "existing.user@example.com",
		Name:  "Existing User Updated Name", // Nome pode ser diferente do que está no DB
		Id:    "google-user-id-existing",
	}
	userInfoServer := mockUserInfoServer(t, expectedUserInfo)
	defer userInfoServer.Close()

	originalTransport := http.DefaultClient.Transport
	if originalTransport == nil {originalTransport = http.DefaultTransport}
	http.DefaultClient.Transport = &mockRoundTripper{
		originalTransport: originalTransport,
		mockTargetURL:     "https://www.googleapis.com/oauth2/v2/userinfo",
		mockServerURL:     userInfoServer.URL,
	}
	defer func() { http.DefaultClient.Transport = originalTransport }()

	originalGoogleEndpoint := oauth2.Google.Endpoint
	oauth2.Google.Endpoint.TokenURL = tokenServer.URL
	defer func() { oauth2.Google.Endpoint.TokenURL = originalGoogleEndpoint.TokenURL }()

	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:3000"
	config.Cfg.JWTSecret = "test-jwt-secret-for-token-generation-32chars"
	config.Cfg.JWTTokenLifespan = 1 * time.Hour
	defer func() { config.Cfg = originalAppConfig }()

	idpID := uuid.New()
	orgID := uuid.New()
	idpName := "Test Google IdP Org Existing"
	oauthIdPConfig := GoogleOAuthConfig{ClientID: "org-google-client-id-exist", ClientSecret: "org-google-client-secret-exist"}
	idpConfigBytes, _ := json.Marshal(oauthIdPConfig)
	idp := models.IdentityProvider{
		BaseModel:      models.BaseModel{ID: idpID}, OrganizationID: orgID, Name: idpName,
		ProviderType: models.IDPTypeOAuth2Google, ConfigJSON: string(idpConfigBytes), IsActive: true,
	}

	existingUserID := uuid.New()
	existingUser := models.User{
		BaseModel:      models.BaseModel{ID: existingUserID},
		OrganizationID: uuid.NullUUID{UUID: orgID, Valid: true},
		Email:          expectedUserInfo.Email,
		Name:           "Existing User Original Name", // Nome original
		Role:           models.RoleAdmin,            // Role diferente para verificar se é preservado/ignorado
		IsActive:       true,
		// SocialLoginID e SSOProvider podem estar vazios ou diferentes antes da atualização
	}

	originalGenerateToken := auth.GenerateToken
	auth.GenerateToken = func(user *models.User, organizationID uuid.NullUUID) (string, error) {
		assert.Equal(t, existingUserID, user.ID)
		assert.Equal(t, orgID, organizationID.UUID)
		return "generated_jwt_token_for_existing_user", nil
	}
	defer func() { auth.GenerateToken = originalGenerateToken }()

	// Mocks do DB
	dbMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "identity_providers"`)).
		WithArgs(idpID.String(), models.IDPTypeOAuth2Google, true).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "name", "provider_type", "config_json", "is_active"}).
			AddRow(idp.ID, idp.OrganizationID, idp.Name, idp.ProviderType, idp.ConfigJSON, idp.IsActive))

	// Busca de usuário (encontrado)
	userRows := sqlmock.NewRows([]string{"id", "organization_id", "name", "email", "role", "is_active", "sso_provider", "social_login_id"}).
		AddRow(existingUser.ID, existingUser.OrganizationID, existingUser.Name, existingUser.Email, existingUser.Role, existingUser.IsActive, existingUser.SSOProvider, existingUser.SocialLoginID)
	dbMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE (email = $1 AND organization_id = $2) OR (social_login_id = $3 AND organization_id = $4)`)).
		WithArgs(expectedUserInfo.Email, orgID, expectedUserInfo.Id, orgID).
		WillReturnRows(userRows)

	// Atualização do usuário
	dbMock.ExpectBegin()
	// Verificar os campos que são atualizados: SSOProvider, SocialLoginID, Name (se vazio ou email), IsActive
	// A query de update do GORM pode ser complexa (ex: UPDATE ... SET ... WHERE id = ... AND version = ...)
	// Usar sqlmock.AnyArg() para campos como updated_at.
	// O importante é que `sso_provider`, `social_login_id`, `name` (se aplicável) e `is_active` são setados.
	// O GORM pode apenas atualizar os campos alterados.
	dbMock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET`)).
		// WithArgs precisa corresponder aos campos na ordem da query SET e depois os da WHERE.
		// Exemplo (pode precisar de ajuste fino):
		// .WithArgs(idp.Name, expectedUserInfo.Id, expectedUserInfo.Name, true, existingUser.ID)
		// O nome do usuário é atualizado se estava vazio ou era igual ao email. No nosso caso, o nome existente não está vazio.
		// A lógica do handler é: if user.Name == "" || user.Name == user.Email { user.Name = fullName }
		// Se existingUser.Name = "Existing User Original Name", ele não será atualizado para expectedUserInfo.Name
		// A menos que fullName (expectedUserInfo.Name) seja diferente e o nome original fosse placeholder.
		// Para este teste, vamos assumir que o nome não é atualizado se já existir um nome não-placeholder.
		// Apenas SSOProvider, SocialLoginID e IsActive serão atualizados com certeza.
		WithArgs(idp.Name, expectedUserInfo.Id, true, sqlmock.AnyArg(), existingUser.ID). // sso_provider, social_login_id, is_active, updated_at, WHERE id
		WillReturnResult(sqlmock.NewResult(1, 1))
	dbMock.ExpectCommit()

	router := gin.Default()
	router.GET("/auth/oauth2/google/:idpId/callback", GoogleCallbackHandler)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/auth/oauth2/google/"+idpID.String()+"/callback?code=validcode&state=teststate", nil)
	req.AddCookie(&http.Cookie{Name: googleOAuthStateCookie, Value: "teststate"})
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	redirectURL, err := w.Result().Location()
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(redirectURL.String(), config.Cfg.AppRootURL+"/oauth2/callback"))
	q := redirectURL.Query()
	assert.Equal(t, "generated_jwt_token_for_existing_user", q.Get("token"))

	assert.NoError(t, dbMock.ExpectationsWereMet(), "Expectativas do DB não foram atendidas")
}

func TestGoogleCallbackHandler_Global_NewUser_WithDefaultOrg(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, dbMock := mockGormDBForCallbackTests(t)
	database.SetDB(gormDB)

	expectedToken := map[string]interface{}{"access_token": "global_new_user_token_default_org", "token_type": "Bearer"}
	tokenServer := mockTokenExchangeServer(t, expectedToken)
	defer tokenServer.Close()

	userInfo := googleAPI.Userinfoplus{Email: "global.new@example.com", Name: "Global New Default Org", Id: "google-global-new-default-org"}
	userInfoServer := mockUserInfoServer(t, userInfo)
	defer userInfoServer.Close()

	originalTransport := http.DefaultClient.Transport
	if originalTransport == nil {originalTransport = http.DefaultTransport}
	http.DefaultClient.Transport = &mockRoundTripper{originalTransport, "https://www.googleapis.com/oauth2/v2/userinfo", userInfoServer.URL}
	defer func() { http.DefaultClient.Transport = originalTransport }()

	originalGoogleEndpoint := oauth2.Google.Endpoint
	oauth2.Google.Endpoint.TokenURL = tokenServer.URL
	defer func() { oauth2.Google.Endpoint.TokenURL = originalGoogleEndpoint.TokenURL }()

	defaultOrgID := uuid.New()
	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:3000"
	config.Cfg.GoogleClientID = "global-client-id-for-new-user-default-org"
	config.Cfg.GoogleClientSecret = "global-client-secret-for-new-user-default-org"
	config.Cfg.AllowGlobalSSOUserCreation = true
	config.Cfg.DefaultOrganizationIDForGlobalSSO = defaultOrgID.String()
	config.Cfg.JWTSecret = "test-jwt-secret"
	config.Cfg.JWTTokenLifespan = 1 * time.Hour
	defer func() { config.Cfg = originalAppConfig }()


	originalGenerateToken := auth.GenerateToken
	auth.GenerateToken = func(user *models.User, organizationID uuid.NullUUID) (string, error) {
		assert.Equal(t, userInfo.Email, user.Email)
		assert.True(t, organizationID.Valid)
		assert.Equal(t, defaultOrgID, organizationID.UUID) // Verifica se o OrgID padrão foi usado
		return "jwt_global_new_user_default_org", nil
	}
	defer func() { auth.GenerateToken = originalGenerateToken }()

	// Mocks do DB:
	// getGoogleOAuthConfig para global não consulta o DB por IdP.
	// Busca de usuário (não encontrado)
	dbMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email = $1 AND (sso_provider = $2 OR social_login_id = $3)`)).
		WithArgs(userInfo.Email, "global_google", userInfo.Id).
		WillReturnError(gorm.ErrRecordNotFound)

	// Criação do usuário com OrgID padrão
	dbMock.ExpectBegin()
	dbMock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "users"`)).
		WithArgs(
			uuid.NullUUID{UUID: defaultOrgID, Valid: true}, // organization_id
			userInfo.Name,
			userInfo.Email,
			"OAUTH2_USER_NO_PASSWORD",
			"global_google", // sso_provider
			userInfo.Id,     // social_login_id
			models.RoleUser,
			true,
			sqlmock.AnyArg(), // id
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
			nil,              // deleted_at
		).WillReturnResult(sqlmock.NewResult(1, 1))
	dbMock.ExpectCommit()

	router := gin.Default()
	router.GET("/auth/oauth2/google/:idpId/callback", GoogleCallbackHandler)
	w := httptest.NewRecorder()
	// Usar GlobalIdPIdentifier
	req, _ := http.NewRequest(http.MethodGet, "/auth/oauth2/google/"+GlobalIdPIdentifier+"/callback?code=globalcode&state=globalstate", nil)
	req.AddCookie(&http.Cookie{Name: googleOAuthStateCookie, Value: "globalstate"})
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	redirectURL, err := w.Result().Location()
	assert.NoError(t, err)
	q := redirectURL.Query()
	assert.Equal(t, "jwt_global_new_user_default_org", q.Get("token"))
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestGoogleCallbackHandler_Global_NewUser_NoDefaultOrg(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, dbMock := mockGormDBForCallbackTests(t)
	database.SetDB(gormDB)

	expectedToken := map[string]interface{}{"access_token": "global_new_user_token_no_default_org", "token_type": "Bearer"}
	tokenServer := mockTokenExchangeServer(t, expectedToken)
	defer tokenServer.Close()

	userInfo := googleAPI.Userinfoplus{Email: "global.new.no@example.com", Name: "Global New No Default Org", Id: "google-global-new-no-default-org"}
	userInfoServer := mockUserInfoServer(t, userInfo)
	defer userInfoServer.Close()

	originalTransport := http.DefaultClient.Transport
	if originalTransport == nil {originalTransport = http.DefaultTransport}
	http.DefaultClient.Transport = &mockRoundTripper{originalTransport, "https://www.googleapis.com/oauth2/v2/userinfo", userInfoServer.URL}
	defer func() { http.DefaultClient.Transport = originalTransport }()

	originalGoogleEndpoint := oauth2.Google.Endpoint
	oauth2.Google.Endpoint.TokenURL = tokenServer.URL
	defer func() { oauth2.Google.Endpoint.TokenURL = originalGoogleEndpoint.TokenURL }()

	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:3000"
	config.Cfg.GoogleClientID = "global-client-id-no-default"
	config.Cfg.GoogleClientSecret = "global-client-secret-no-default"
	config.Cfg.AllowGlobalSSOUserCreation = true
	config.Cfg.DefaultOrganizationIDForGlobalSSO = "" // Sem Org Padrão
	config.Cfg.JWTSecret = "test-jwt-secret"
	config.Cfg.JWTTokenLifespan = 1 * time.Hour
	defer func() { config.Cfg = originalAppConfig }()

	originalGenerateToken := auth.GenerateToken
	auth.GenerateToken = func(user *models.User, organizationID uuid.NullUUID) (string, error) {
		assert.Equal(t, userInfo.Email, user.Email)
		assert.False(t, organizationID.Valid) // Verifica se o OrgID é nulo/inválido
		return "jwt_global_new_user_no_default_org", nil
	}
	defer func() { auth.GenerateToken = originalGenerateToken }()

	dbMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email = $1 AND (sso_provider = $2 OR social_login_id = $3)`)).
		WithArgs(userInfo.Email, "global_google", userInfo.Id).
		WillReturnError(gorm.ErrRecordNotFound)

	dbMock.ExpectBegin()
	dbMock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "users"`)).
		WithArgs(
			nil, // organization_id (deve ser NULL)
			userInfo.Name, userInfo.Email, "OAUTH2_USER_NO_PASSWORD", "global_google", userInfo.Id,
			models.RoleUser, true, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), nil,
		).WillReturnResult(sqlmock.NewResult(1, 1))
	dbMock.ExpectCommit()

	router := gin.Default()
	router.GET("/auth/oauth2/google/:idpId/callback", GoogleCallbackHandler)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/auth/oauth2/google/"+GlobalIdPIdentifier+"/callback?code=globalnocode&state=globalnostate", nil)
	req.AddCookie(&http.Cookie{Name: googleOAuthStateCookie, Value: "globalnostate"})
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	redirectURL, err := w.Result().Location()
	assert.NoError(t, err)
	q := redirectURL.Query()
	assert.Equal(t, "jwt_global_new_user_no_default_org", q.Get("token"))
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestGoogleCallbackHandler_Global_NewUser_CreationDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, dbMock := mockGormDBForCallbackTests(t)
	database.SetDB(gormDB)

	expectedToken := map[string]interface{}{"access_token": "token_creation_disabled", "token_type": "Bearer"}
	tokenServer := mockTokenExchangeServer(t, expectedToken)
	defer tokenServer.Close()

	userInfo := googleAPI.Userinfoplus{Email: "global.new.disabled@example.com", Name: "Global New Disabled", Id: "google-global-new-disabled"}
	userInfoServer := mockUserInfoServer(t, userInfo)
	defer userInfoServer.Close()

	originalTransport := http.DefaultClient.Transport
	if originalTransport == nil {originalTransport = http.DefaultTransport}
	http.DefaultClient.Transport = &mockRoundTripper{originalTransport, "https://www.googleapis.com/oauth2/v2/userinfo", userInfoServer.URL}
	defer func() { http.DefaultClient.Transport = originalTransport }()

	originalGoogleEndpoint := oauth2.Google.Endpoint
	oauth2.Google.Endpoint.TokenURL = tokenServer.URL
	defer func() { oauth2.Google.Endpoint.TokenURL = originalGoogleEndpoint.TokenURL }()

	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:3000"
	config.Cfg.GoogleClientID = "global-client-id-disabled"
	config.Cfg.GoogleClientSecret = "global-client-secret-disabled"
	config.Cfg.AllowGlobalSSOUserCreation = false // Criação desabilitada
	defer func() { config.Cfg = originalAppConfig }()

	// auth.GenerateToken não deve ser chamado
	originalGenerateToken := auth.GenerateToken
	auth.GenerateToken = func(user *models.User, organizationID uuid.NullUUID) (string, error) {
		t.Error("auth.GenerateToken não deveria ser chamado quando a criação de usuário global está desabilitada para novo usuário")
		return "", errors.New("não deveria ser chamado")
	}
	defer func() { auth.GenerateToken = originalGenerateToken }()

	dbMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email = $1 AND (sso_provider = $2 OR social_login_id = $3)`)).
		WithArgs(userInfo.Email, "global_google", userInfo.Id).
		WillReturnError(gorm.ErrRecordNotFound)

	// Nenhuma chamada de INSERT deve ser esperada

	router := gin.Default()
	router.GET("/auth/oauth2/google/:idpId/callback", GoogleCallbackHandler)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/auth/oauth2/google/"+GlobalIdPIdentifier+"/callback?code=disabled_code&state=disabled_state", nil)
	req.AddCookie(&http.Cookie{Name: googleOAuthStateCookie, Value: "disabled_state"})
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	var jsonResponse map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &jsonResponse)
	assert.NoError(t, err)
	assert.Contains(t, jsonResponse["error"], "New user registration via global Google SSO is disabled")

	// Verificar limpeza do cookie
	cookieFoundAndCleared := false
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == googleOAuthStateCookie {
			assert.LessOrEqual(t, cookie.MaxAge, 0)
			cookieFoundAndCleared = true
			break
		}
	}
	assert.True(t, cookieFoundAndCleared, "Set-Cookie para limpar o cookie de estado não foi encontrado")
	assert.NoError(t, dbMock.ExpectationsWereMet()) // Garante que o INSERT não foi chamado
}

func TestGoogleCallbackHandler_Global_ExistingUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, dbMock := mockGormDBForCallbackTests(t)
	database.SetDB(gormDB)

	tokenServer := mockTokenExchangeServer(t, map[string]interface{}{"access_token": "global_exist_token", "token_type": "Bearer"})
	defer tokenServer.Close()

	userInfo := googleAPI.Userinfoplus{Email: "global.existing@example.com", Name: "Global Existing Updated Name", Id: "google-global-existing-id"}
	userInfoServer := mockUserInfoServer(t, userInfo)
	defer userInfoServer.Close()

	originalTransport := http.DefaultClient.Transport; if originalTransport == nil {originalTransport = http.DefaultTransport}
	http.DefaultClient.Transport = &mockRoundTripper{originalTransport, "https://www.googleapis.com/oauth2/v2/userinfo", userInfoServer.URL}
	defer func() { http.DefaultClient.Transport = originalTransport }()

	originalGoogleEndpoint := oauth2.Google.Endpoint; oauth2.Google.Endpoint.TokenURL = tokenServer.URL
	defer func() { oauth2.Google.Endpoint.TokenURL = originalGoogleEndpoint.TokenURL }()

	// Usuário existente pode ou não ter uma OrgID. Vamos testar com OrgID nulo.
	existingUserOrgID := uuid.NullUUID{Valid: false}
	existingUserID := uuid.New()
	existingUser := models.User{
		BaseModel:      models.BaseModel{ID: existingUserID},
		OrganizationID: existingUserOrgID,
		Email:          userInfo.Email,
		Name:           "Global Existing Original Name",
		Role:           models.RoleUser,
		IsActive:       false, // Para verificar se é ativado
		SSOProvider:    "some_other_provider", // Para verificar se é atualizado
		SocialLoginID:  "some_other_id",
	}


	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:3000"
	config.Cfg.GoogleClientID = "global-client-exist"
	config.Cfg.GoogleClientSecret = "global-secret-exist"
	config.Cfg.AllowGlobalSSOUserCreation = false // Não deve impedir login de usuário existente
	config.Cfg.JWTSecret = "test-jwt-secret"
	config.Cfg.JWTTokenLifespan = 1 * time.Hour
	defer func() { config.Cfg = originalAppConfig }()

	originalGenerateToken := auth.GenerateToken
	auth.GenerateToken = func(user *models.User, organizationID uuid.NullUUID) (string, error) {
		assert.Equal(t, existingUserID, user.ID)
		assert.Equal(t, existingUserOrgID.Valid, organizationID.Valid) // Preservar OrgID original (nulo neste caso)
		if existingUserOrgID.Valid {
			assert.Equal(t, existingUserOrgID.UUID, organizationID.UUID)
		}
		assert.True(t, user.IsActive)
		assert.Equal(t, "global_google", user.SSOProvider)
		assert.Equal(t, userInfo.Id, user.SocialLoginID)
		return "jwt_global_existing_user", nil
	}
	defer func() { auth.GenerateToken = originalGenerateToken }()

	// Mock DB: Busca de usuário (encontrado)
	userRows := sqlmock.NewRows([]string{"id", "organization_id", "name", "email", "role", "is_active", "sso_provider", "social_login_id"}).
		AddRow(existingUser.ID, existingUser.OrganizationID, existingUser.Name, existingUser.Email, existingUser.Role, existingUser.IsActive, existingUser.SSOProvider, existingUser.SocialLoginID)
	dbMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email = $1 AND (sso_provider = $2 OR social_login_id = $3)`)).
		WithArgs(userInfo.Email, "global_google", userInfo.Id).
		WillReturnRows(userRows)

	// Mock DB: Atualização do usuário
	dbMock.ExpectBegin()
	// Name não deve ser atualizado porque existingUser.Name não é "" nem igual ao email.
	dbMock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET`)).
		WithArgs("global_google", userInfo.Id, true, sqlmock.AnyArg(), existingUser.ID). // sso_provider, social_login_id, is_active, updated_at, WHERE id
		WillReturnResult(sqlmock.NewResult(1, 1))
	dbMock.ExpectCommit()


	router := gin.Default()
	router.GET("/auth/oauth2/google/:idpId/callback", GoogleCallbackHandler)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/auth/oauth2/google/"+GlobalIdPIdentifier+"/callback?code=globalexistcode&state=globalexiststate", nil)
	req.AddCookie(&http.Cookie{Name: googleOAuthStateCookie, Value: "globalexiststate"})
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	redirectURL, err := w.Result().Location()
	assert.NoError(t, err)
	q := redirectURL.Query()
	assert.Equal(t, "jwt_global_existing_user", q.Get("token"))
	assert.NoError(t, dbMock.ExpectationsWereMet())
}


// Adicionar mais testes para GoogleCallbackHandler (usuário existente, IdP global, erros, etc.)

// --- Testes para GithubCallbackHandler ---

// mockGithubUserInfoServer simula o endpoint de user info do Github.
// Pode precisar de lógica para lidar com /user e /user/emails separadamente se necessário.
func mockGithubUserInfoServer(t *testing.T, primaryUserInfo GithubUserResponse, emailInfo []GithubUserEmailResponse, primaryPathOnly bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/user/emails") && !primaryPathOnly {
			json.NewEncoder(w).Encode(emailInfo)
		} else if strings.HasSuffix(r.URL.Path, "/user") || primaryPathOnly { // Assume /user se não for /user/emails
			json.NewEncoder(w).Encode(primaryUserInfo)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}


func TestGithubCallbackHandler_OrgSpecific_NewUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, dbMock := mockGormDBForCallbackTests(t)
	database.SetDB(gormDB)

	expectedToken := map[string]interface{}{"access_token": "github_token_org_new", "token_type": "bearer"}
	tokenServer := mockTokenExchangeServer(t, expectedToken) // Reutilizável
	defer tokenServer.Close()

	// UserInfo do Github - simplificado, email primário já vem
	githubUser := GithubUserResponse{
		Email: "new.gh.user@example.com",
		Name:  "New Github User",
		ID:    12345,
		Login: "newghuser",
	}
	// Para este teste, assumimos que o email vem na primeira chamada /user
	userInfoServer := mockGithubUserInfoServer(t, githubUser, nil, true)
	defer userInfoServer.Close()


	originalTransport := http.DefaultClient.Transport
	if originalTransport == nil {originalTransport = http.DefaultTransport}
	// Mock do transporte HTTP para redirecionar chamadas de UserInfo do Github
	// A URL real do Github API é https://api.github.com/user
	http.DefaultClient.Transport = &mockRoundTripper{
		originalTransport: originalTransport,
		mockTargetURL:     "https://api.github.com/user", // URL base da API do Github para /user
		mockServerURL:     userInfoServer.URL,            // Nosso mock server (que só responde para /user neste config)
	}
	defer func() { http.DefaultClient.Transport = originalTransport }()


	// Sobrescrever TokenURL do Github para apontar para nosso mock tokenServer
	originalGithubEndpoint := oauth2.GitHub.Endpoint
	oauth2.GitHub.Endpoint.TokenURL = tokenServer.URL
	defer func() { oauth2.GitHub.Endpoint.TokenURL = originalGithubEndpoint.TokenURL }()


	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:3000"
	config.Cfg.JWTSecret = "test-jwt-secret-github"
	config.Cfg.JWTTokenLifespan = 1 * time.Hour
	defer func() { config.Cfg = originalAppConfig }()

	idpID := uuid.New()
	orgID := uuid.New()
	idpName := "Test Github IdP Org"
	oauthIdPConfig := GithubOAuthConfig{ClientID: "org-github-client-id", ClientSecret: "org-github-client-secret"}
	idpConfigBytes, _ := json.Marshal(oauthIdPConfig)
	idp := models.IdentityProvider{
		BaseModel:      models.BaseModel{ID: idpID}, OrganizationID: orgID, Name: idpName,
		ProviderType: models.IDPTypeOAuth2Github, ConfigJSON: string(idpConfigBytes), IsActive: true,
	}

	originalGenerateToken := auth.GenerateToken
	auth.GenerateToken = func(user *models.User, organizationID uuid.NullUUID) (string, error) {
		assert.Equal(t, githubUser.Email, user.Email)
		assert.Equal(t, orgID, organizationID.UUID)
		assert.Equal(t, idpName, user.SSOProvider) // Nome do IdP da Org
		assert.Equal(t, fmt.Sprintf("%d", githubUser.ID), user.SocialLoginID)
		return "jwt_github_org_new_user", nil
	}
	defer func() { auth.GenerateToken = originalGenerateToken }()

	// Mocks do DB
	dbMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "identity_providers" WHERE id = $1 AND provider_type = $2 AND is_active = $3`)).
		WithArgs(idpID.String(), models.IDPTypeOAuth2Github, true).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "name", "provider_type", "config_json", "is_active"}).
			AddRow(idp.ID, idp.OrganizationID, idp.Name, idp.ProviderType, idp.ConfigJSON, idp.IsActive))

	dbMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE (email = $1 AND organization_id = $2) OR (social_login_id = $3 AND organization_id = $4 AND sso_provider = $5)`)).
		WithArgs(githubUser.Email, orgID, fmt.Sprintf("%d", githubUser.ID), orgID, idpName).
		WillReturnError(gorm.ErrRecordNotFound)

	dbMock.ExpectBegin()
	dbMock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "users"`)).
		WithArgs(
			idp.OrganizationID, githubUser.Name, githubUser.Email, "OAUTH2_USER_NO_PASSWORD",
			idp.Name, fmt.Sprintf("%d", githubUser.ID), models.RoleUser, true,
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), nil,
		).WillReturnResult(sqlmock.NewResult(1, 1))
	dbMock.ExpectCommit()

	router := gin.Default()
	router.GET("/auth/oauth2/github/:idpId/callback", GithubCallbackHandler) // Usar GithubCallbackHandler
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/auth/oauth2/github/"+idpID.String()+"/callback?code=gh_org_code&state=gh_org_state", nil)
	req.AddCookie(&http.Cookie{Name: githubOAuthStateCookie, Value: "gh_org_state"}) // Usar githubOAuthStateCookie
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	redirectURL, err := w.Result().Location()
	assert.NoError(t, err)
	q := redirectURL.Query()
	assert.Equal(t, "jwt_github_org_new_user", q.Get("token"))
	assert.Equal(t, "github", q.Get("provider")) // Verificar provider no redirect
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestGithubCallbackHandler_Global_NewUser_WithDefaultOrg(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, dbMock := mockGormDBForCallbackTests(t)
	database.SetDB(gormDB)

	expectedToken := map[string]interface{}{"access_token": "gh_global_new_def_org_token", "token_type": "Bearer"}
	tokenServer := mockTokenExchangeServer(t, expectedToken)
	defer tokenServer.Close()

	githubUser := GithubUserResponse{Email: "gh.global.new.def@example.com", Name: "GH Global New Default Org", ID: 56789, Login: "ghglobalnewdef"}
	userInfoServer := mockGithubUserInfoServer(t, githubUser, nil, true) // Email na primeira chamada
	defer userInfoServer.Close()

	originalTransport := http.DefaultClient.Transport; if originalTransport == nil {originalTransport = http.DefaultTransport}
	http.DefaultClient.Transport = &mockRoundTripper{originalTransport, "https://api.github.com/user", userInfoServer.URL}
	defer func() { http.DefaultClient.Transport = originalTransport }()

	originalGithubEndpoint := oauth2.GitHub.Endpoint; oauth2.GitHub.Endpoint.TokenURL = tokenServer.URL
	defer func() { oauth2.GitHub.Endpoint.TokenURL = originalGithubEndpoint.TokenURL }()

	defaultOrgID := uuid.New()
	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:3000"
	config.Cfg.GithubClientID = "global-gh-client-id-new-def"
	config.Cfg.GithubClientSecret = "global-gh-client-secret-new-def"
	config.Cfg.AllowGlobalSSOUserCreation = true
	config.Cfg.DefaultOrganizationIDForGlobalSSO = defaultOrgID.String()
	config.Cfg.JWTSecret = "test-jwt-secret-gh"
	config.Cfg.JWTTokenLifespan = 1 * time.Hour
	defer func() { config.Cfg = originalAppConfig }()

	originalGenerateToken := auth.GenerateToken
	auth.GenerateToken = func(user *models.User, organizationID uuid.NullUUID) (string, error) {
		assert.Equal(t, githubUser.Email, user.Email)
		assert.True(t, organizationID.Valid)
		assert.Equal(t, defaultOrgID, organizationID.UUID)
		assert.Equal(t, "global_github", user.SSOProvider)
		return "jwt_gh_global_new_default_org", nil
	}
	defer func() { auth.GenerateToken = originalGenerateToken }()

	// Mock DB: Busca de usuário global (não encontrado)
	// A query no handler é: email = ? OR (social_login_id = ? AND sso_provider = ?)
	dbMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE email = $1 OR (social_login_id = $2 AND sso_provider = $3)`)).
		WithArgs(githubUser.Email, fmt.Sprintf("%d", githubUser.ID), "global_github").
		WillReturnError(gorm.ErrRecordNotFound)

	// Mock DB: Criação do usuário com OrgID padrão
	dbMock.ExpectBegin()
	dbMock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "users"`)).
		WithArgs(
			uuid.NullUUID{UUID: defaultOrgID, Valid: true}, // organization_id
			githubUser.Name, githubUser.Email, "OAUTH2_USER_NO_PASSWORD",
			"global_github", fmt.Sprintf("%d", githubUser.ID), // sso_provider, social_login_id
			models.RoleUser, true,
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), nil,
		).WillReturnResult(sqlmock.NewResult(1, 1))
	dbMock.ExpectCommit()

	router := gin.Default()
	router.GET("/auth/oauth2/github/:idpId/callback", GithubCallbackHandler)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/auth/oauth2/github/"+GlobalIdPIdentifier+"/callback?code=ghglobalcode_def&state=ghglobalstate_def", nil)
	req.AddCookie(&http.Cookie{Name: githubOAuthStateCookie, Value: "ghglobalstate_def"})
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	redirectURL, err := w.Result().Location()
	assert.NoError(t, err)
	q := redirectURL.Query()
	assert.Equal(t, "jwt_gh_global_new_default_org", q.Get("token"))
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestGithubCallbackHandler_TokenExchangeFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gormDB, dbMock := mockGormDBForCallbackTests(t)
	database.SetDB(gormDB)

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "bad_verification_code"})
	}))
	defer tokenServer.Close()

	originalGithubEndpoint := oauth2.GitHub.Endpoint
	oauth2.GitHub.Endpoint.TokenURL = tokenServer.URL
	defer func() { oauth2.GitHub.Endpoint.TokenURL = originalGithubEndpoint.TokenURL }()

	originalAppConfig := config.Cfg
	config.Cfg.AppRootURL = "http://localhost:3000"
	// Para IdP específico, para garantir que getGithubOAuthConfig seja chamado
	config.Cfg.GithubClientID = ""
	config.Cfg.GithubClientSecret = ""
	defer func() { config.Cfg = originalAppConfig }()


	idpID := uuid.New()
	idp := models.IdentityProvider{
		BaseModel:    models.BaseModel{ID: idpID},
		ProviderType: models.IDPTypeOAuth2Github,
		ConfigJSON:   `{"client_id":"gh-id","client_secret":"gh-secret"}`,
		IsActive:     true,
	}
	dbMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "identity_providers"`)).
		WithArgs(idpID.String(), models.IDPTypeOAuth2Github, true).
		WillReturnRows(sqlmock.NewRows([]string{"id", "provider_type", "config_json", "is_active"}).
			AddRow(idp.ID, idp.ProviderType, idp.ConfigJSON, idp.IsActive))

	router := gin.Default()
	router.GET("/auth/oauth2/github/:idpId/callback", GithubCallbackHandler)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/auth/oauth2/github/"+idpID.String()+"/callback?code=validcode&state=teststate_gh_fail", nil)
	req.AddCookie(&http.Cookie{Name: githubOAuthStateCookie, Value: "teststate_gh_fail"})
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var jsonResponse map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &jsonResponse)
	assert.NoError(t, err)
	assert.Contains(t, jsonResponse["error"], "Failed to exchange OAuth code for token")
	assert.NoError(t, dbMock.ExpectationsWereMet())

	cookieFoundAndCleared := false
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == githubOAuthStateCookie {
			assert.LessOrEqual(t, cookie.MaxAge, 0)
			cookieFoundAndCleared = true
			break
		}
	}
	assert.True(t, cookieFoundAndCleared, "Set-Cookie para limpar o cookie de estado não foi encontrado")
}
