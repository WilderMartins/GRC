package handlers

import (
	"fmt"
	"net/http"
	"phoenixgrc/backend/pkg/config" // Para acessar config.Cfg
	phxlog "phoenixgrc/backend/pkg/log"  // Importar o logger zap
	"go.uber.org/zap"                 // Importar zap
	"strings"

	"github.com/gin-gonic/gin"
)

// GlobalIdPResponse define a estrutura para provedores de identidade sociais globais.
type GlobalIdPResponse struct {
	Key      string `json:"key"`       // ex: "google", "github" (para uso no frontend)
	Type     string `json:"type"`      // ex: "oauth2_google", "oauth2_github"
	Name     string `json:"name"`      // ex: "Login com Google", "Login com GitHub"
	LoginURL string `json:"login_url"` // URL para iniciar o fluxo de login OAuth2
}

// ListGlobalSocialIdentityProvidersHandler retorna uma lista de provedores de identidade sociais
// configurados globalmente através de variáveis de ambiente.
func ListGlobalSocialIdentityProvidersHandler(c *gin.Context) {
	var providers []GlobalIdPResponse

	appRootURL := config.Cfg.AppRootURL
	if appRootURL == "" {
		appRootURL = "http://localhost:8080" // Fallback para desenvolvimento se não configurado via .env
		phxlog.L.Warn("APP_ROOT_URL is not configured. Social login URLs may use an insecure fallback.",
			zap.String("fallback_url", appRootURL))
	}
	// Remover quaisquer barras extras do final para evitar // no path
	appRootURL = strings.TrimSuffix(appRootURL, "/")

	// Verificar Google
	// As variáveis config.Cfg.GoogleClientID e config.Cfg.GoogleClientSecret precisam ser adicionadas à struct AppConfig
	// e carregadas a partir de variáveis de ambiente (ex: GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET).
	if config.Cfg.GoogleClientID != "" && config.Cfg.GoogleClientSecret != "" {
		providers = append(providers, GlobalIdPResponse{
			Key:      "google",
			Type:     "oauth2_google",
			Name:     "Login com Google",
			LoginURL: fmt.Sprintf("%s/auth/oauth2/google/global/login", appRootURL),
		})
	}

	// Verificar GitHub
	// As variáveis config.Cfg.GithubClientID e config.Cfg.GithubClientSecret precisam ser adicionadas.
	if config.Cfg.GithubClientID != "" && config.Cfg.GithubClientSecret != "" {
		providers = append(providers, GlobalIdPResponse{
			Key:      "github",
			Type:     "oauth2_github",
			Name:     "Login com GitHub",
			LoginURL: fmt.Sprintf("%s/auth/oauth2/github/global/login", appRootURL),
		})
	}

	c.JSON(http.StatusOK, providers)
}

// SetupStatusResponse define a resposta para o endpoint de status do setup.
type SetupStatusResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// GetSetupStatusHandler verifica e retorna o estado atual da configuração da aplicação.
func GetSetupStatusHandler(c *gin.Context) {
	db := database.GetDB()

	// 1. Verificar conexão com o DB
	sqlDB, err := db.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, SetupStatusResponse{
			Status:  "database_not_configured",
			Message: "A instância do banco de dados não está disponível.",
		})
		return
	}
	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, SetupStatusResponse{
			Status:  "database_not_connected",
			Message: "Não foi possível conectar ao banco de dados. Verifique as credenciais e a conectividade.",
		})
		return
	}

	// 2. Verificar se as migrações foram executadas
	// Uma forma simples é verificar se a tabela 'users' existe.
	if !db.Migrator().HasTable(&models.User{}) {
		c.JSON(http.StatusOK, SetupStatusResponse{
			Status:  "migrations_not_run",
			Message: "Conexão com o banco de dados OK, mas as tabelas da aplicação não foram criadas. Execute o setup.",
		})
		return
	}

	// 3. Verificar se a primeira organização e o admin foram criados
	var orgCount int64
	db.Model(&models.Organization{}).Count(&orgCount)
	if orgCount == 0 {
		c.JSON(http.StatusOK, SetupStatusResponse{
			Status:  "setup_pending_org",
			Message: "Migrações concluídas, mas a primeira organização e o usuário administrador precisam ser criados.",
		})
		return
	}

	var adminCount int64
	db.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&adminCount)
	if adminCount == 0 {
		c.JSON(http.StatusOK, SetupStatusResponse{
			Status:  "setup_pending_admin",
			Message: "Organização criada, mas o usuário administrador não foi encontrado. Complete o setup.",
		})
		return
	}

	// Se tudo estiver OK
	c.JSON(http.StatusOK, SetupStatusResponse{
		Status:  "setup_complete",
		Message: "A aplicação está configurada e pronta para uso.",
	})
}
