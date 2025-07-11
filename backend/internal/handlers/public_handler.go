package handlers

import (
	"fmt"
	"log"
	"net/http"
	"phoenixgrc/backend/pkg/config" // Para acessar config.Cfg
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
		// Em produção, APP_ROOT_URL deve ser obrigatório e configurado corretamente.
		log.Printf("AVISO: APP_ROOT_URL não está configurado. As URLs de login social podem usar um fallback inseguro: %s", appRootURL)
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
