package oauth2auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	// "os" // log e fmt.Fprintf(os.Stderr,...) serão substituídos
	"phoenixgrc/backend/internal/auth"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	phxlog "phoenixgrc/backend/pkg/log" // Importar o logger zap
	"go.uber.org/zap"                 // Importar zap
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	githubOAuth2 "golang.org/x/oauth2/github"
	"gorm.io/gorm"
	appConfig "phoenixgrc/backend/pkg/config" // Para credenciais globais e AppRootURL
)

const githubOAuthStateCookie = "phoenixgrc_github_oauth_state"
// globalIdPIdentifier já deve estar definido em common.go ou google.go, ou definir aqui se necessário.
// const globalIdPIdentifier = "global" // Definido em google.go e usado aqui

// GithubOAuthConfig defines fields for Github OAuth2 provider from IdentityProvider.ConfigJSON
// ou das variáveis de ambiente globais.
type GithubOAuthConfig struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	Scopes       []string `json:"scopes"` // e.g., ["user:email", "read:user"]
}

// GithubUserResponse defines the structure for user info from Github API
type GithubUserResponse struct {
	ID    int64  `json:"id"`
	Login string `json:"login"` // Username
	Name  string `json:"name"`
	Email string `json:"email"` // May be null if not public and no user:email scope
}

// GithubUserEmailResponse for fetching private emails if primary is not set
type GithubUserEmailResponse struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func getGithubOAuthConfig(idpIDStr string, db *gorm.DB) (*oauth2.Config, *GithubOAuthConfig, *models.IdentityProvider, error) {
	currentAppRootURL := appConfig.Cfg.AppRootURL
	if currentAppRootURL == "" {
		return nil, nil, nil, fmt.Errorf("OAuth2 global configuration (APP_ROOT_URL) not initialized or empty")
	}

	var cfg GithubOAuthConfig
	var idpModelFromDB *models.IdentityProvider = nil

	dynamicRedirectURI := fmt.Sprintf("%s/auth/oauth2/github/%s/callback", currentAppRootURL, idpIDStr)

	if idpIDStr == GlobalIdPIdentifier {
		cfg.ClientID = appConfig.Cfg.GithubClientID
		cfg.ClientSecret = appConfig.Cfg.GithubClientSecret
		if len(cfg.Scopes) == 0 { // Default scopes for global
			cfg.Scopes = []string{"read:user", "user:email"}
		}
		if cfg.ClientID == "" || cfg.ClientSecret == "" {
			return nil, &cfg, nil, fmt.Errorf("global Github OAuth2 (GITHUB_CLIENT_ID/SECRET) not configured")
		}
	} else {
		idpID, err := uuid.Parse(idpIDStr)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("invalid IdP ID format for Github OAuth2: %s", idpIDStr)
		}
		var fetchedModel models.IdentityProvider
		err = db.Where("id = ? AND provider_type = ? AND is_active = ?", idpID, models.IDPTypeOAuth2Github, true).First(&fetchedModel).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, nil, nil, fmt.Errorf("active Github OAuth2 provider configuration not found for ID: %s", idpIDStr)
			}
			return nil, nil, nil, fmt.Errorf("database error fetching Github IdP config for ID %s: %w", idpIDStr, err)
		}
		idpModelFromDB = &fetchedModel

		if errUnmarshal := json.Unmarshal([]byte(idpModelFromDB.ConfigJSON), &cfg); errUnmarshal != nil {
			return nil, nil, idpModelFromDB, fmt.Errorf("failed to unmarshal Github OAuth2 config from JSON for IdP %s: %w", idpIDStr, errUnmarshal)
		}
		if cfg.ClientID == "" || cfg.ClientSecret == "" {
			return nil, &cfg, idpModelFromDB, fmt.Errorf("client_id or client_secret missing in Github OAuth2 config for IdP %s", idpIDStr)
		}
		if len(cfg.Scopes) == 0 {
			cfg.Scopes = []string{"read:user", "user:email"} // Default scopes
		}
	}

	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  dynamicRedirectURI,
		Scopes:       cfg.Scopes,
		Endpoint:     githubOAuth2.Endpoint,
	}, &cfg, idpModelFromDB, nil
}

// GithubLoginHandler initiates the Github OAuth2 login flow.
func GithubLoginHandler(c *gin.Context) {
	idpIDStr := c.Param("idpId") // Pode ser UUID ou "global"
	db := database.GetDB()

	oauthCfg, _, _, err := getGithubOAuthConfig(idpIDStr, db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to configure Github OAuth2: " + err.Error()})
		return
	}

	b := make([]byte, 16)
	_, _ = rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     githubOAuthStateCookie,
		Value:    state,
		Expires:  time.Now().Add(10 * time.Minute),
		HttpOnly: true,
		Path:     "/", // Consider path "/auth/oauth2/github/"+idpIDStr for specificity
		Secure:   c.Request.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	redirectURL := oauthCfg.AuthCodeURL(state)
	c.Redirect(http.StatusFound, redirectURL)
}

// GithubCallbackHandler handles the callback from Github after user authorization.
func GithubCallbackHandler(c *gin.Context) {
	idpIDStr := c.Param("idpId") // Pode ser UUID ou "global"

	stateCookie, err := c.Cookie(githubOAuthStateCookie)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing OAuth state cookie"})
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{Name: githubOAuthStateCookie, Value: "", MaxAge: -1, Path: "/"})

	if c.Query("state") != stateCookie.Value { // Compare with stateCookie.Value
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid OAuth state"})
		return
	}

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "OAuth authorization code not found or access denied"})
		return
	}

	db := database.GetDB()
	oauthCfg, _, idpModelFromDB, err := getGithubOAuthConfig(idpIDStr, db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to re-configure Github OAuth2 for token exchange: " + err.Error()})
		return
	}

	token, err := oauthCfg.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange OAuth code for token: " + err.Error()})
		return
	}
	if !token.Valid() {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth token received is invalid or expired"})
		return
	}

	// Get user info from Github
	client := oauthCfg.Client(context.Background(), token)
	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info from Github: " + err.Error()})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var ghUser GithubUserResponse
	if err := json.Unmarshal(body, &ghUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse Github user info: " + err.Error()})
		return
	}

	email := ghUser.Email
	if email == "" {
		reqEmails, _ := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
		respEmails, errEmails := client.Do(reqEmails)
		if errEmails != nil {
			phxlog.L.Warn("Failed to get user emails from Github during OAuth callback",
				zap.String("idpIDStr", idpIDStr), // Adicionar contexto
				zap.Error(errEmails))
			// Continuar, pois o email primário pode ter sido encontrado ou pode ser um erro não fatal.
		} else {
			defer respEmails.Body.Close()
			bodyEmails, _ := io.ReadAll(respEmails.Body)
			var ghEmails []GithubUserEmailResponse
			if errJson := json.Unmarshal(bodyEmails, &ghEmails); errJson == nil {
				for _, e := range ghEmails {
					if e.Primary && e.Verified {
						email = e.Email
						break
					}
				}
				if email == "" { // Fallback to first verified email if no primary
					for _, e := range ghEmails {
						if e.Verified {
							email = e.Email
							break
						}
					}
				}
			}
		}
	}

	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email not provided or accessible from Github. Ensure 'user:email' scope is granted and a verified public email exists."})
		return
	}
	email = strings.ToLower(strings.TrimSpace(email))
	fullName := strings.TrimSpace(ghUser.Name)
	if fullName == "" {
		fullName = strings.TrimSpace(ghUser.Login)
	}
	if fullName == "" {
		fullName = email
	}
	githubUserIDStr := fmt.Sprintf("%d", ghUser.ID)

	// --- User Provisioning/Login ---
	var user models.User
	var userOrgID uuid.NullUUID
	ssoProviderName := "github" // Default for global

	if idpIDStr == GlobalIdPIdentifier {
		ssoProviderName = "global_github"
		// Try to find user by email OR by social login ID if previously linked with global_github
		err = db.Where("email = ? OR (social_login_id = ? AND sso_provider = ?)", email, githubUserIDStr, ssoProviderName).First(&user).Error

		if err != nil && err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error fetching user for global Github login: " + err.Error()})
			return
		}

		if err == gorm.ErrRecordNotFound {
			if !appConfig.Cfg.AllowGlobalSSOUserCreation {
				c.JSON(http.StatusForbidden, gin.H{"error": "New user registration via global Github SSO is disabled."})
				return
			}
			orgIDForNewUser := uuid.NullUUID{}
			if appConfig.Cfg.DefaultOrganizationIDForGlobalSSO != "" {
				parsedOrgID, errParseOrg := uuid.Parse(appConfig.Cfg.DefaultOrganizationIDForGlobalSSO)
				if errParseOrg == nil {
					orgIDForNewUser = uuid.NullUUID{UUID: parsedOrgID, Valid: true}
				} else {
					phxlog.L.Warn("DEFAULT_ORGANIZATION_ID_FOR_GLOBAL_SSO is not a valid UUID. User will be created without an organization.",
						zap.String("configuredDefaultOrgID", appConfig.Cfg.DefaultOrganizationIDForGlobalSSO),
						zap.Error(errParseOrg))
				}
			}
			user = models.User{
				OrganizationID: orgIDForNewUser,
				Name:           fullName,
				Email:          email,
				PasswordHash:   "OAUTH2_USER_NO_PASSWORD",
				SSOProvider:    ssoProviderName,
				SocialLoginID:  githubUserIDStr,
				Role:           models.RoleUser,
				IsActive:       true,
			}
			if createErr := db.Create(&user).Error; createErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new global Github SSO user: " + createErr.Error()})
				return
			}
		} else { // User exists
			user.SSOProvider = ssoProviderName // Ensure it's marked as global_github
			user.SocialLoginID = githubUserIDStr
			if user.Name == "" || user.Name == user.Email { user.Name = fullName }
			user.IsActive = true
			if saveErr := db.Save(&user).Error; saveErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update global Github SSO user: " + saveErr.Error()})
				return
			}
		}
		userOrgID = user.OrganizationID // This will be nil if the user is not yet associated with an org

	} else { // Organization-specific IdP flow
		if idpModelFromDB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "IdP configuration missing for organization-specific Github login."})
			return
		}
		ssoProviderName = idpModelFromDB.Name // Use the actual name from the IdP config

		// Try to find user by email within the organization OR by social login ID if previously linked to this org's IdP
		err = db.Where("(email = ? AND organization_id = ?) OR (social_login_id = ? AND organization_id = ? AND sso_provider = ?)",
			email, idpModelFromDB.OrganizationID, githubUserIDStr, idpModelFromDB.OrganizationID, ssoProviderName).First(&user).Error

		if err != nil && err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error fetching user for org Github login: " + err.Error()})
			return
		}

		if err == gorm.ErrRecordNotFound { // User does not exist in this org with this IdP, provision
			user = models.User{
				OrganizationID: idpModelFromDB.OrganizationID,
				Name:           fullName,
				Email:          email,
				PasswordHash:   "OAUTH2_USER_NO_PASSWORD",
				SSOProvider:    ssoProviderName,
				SocialLoginID:  githubUserIDStr,
				Role:           models.RoleUser, // Default role
				IsActive:       true,
			}
			if createErr := db.Create(&user).Error; createErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new org Github SSO user: " + createErr.Error()})
				return
			}
		} else { // User exists, update
			user.SSOProvider = ssoProviderName
			user.SocialLoginID = githubUserIDStr
			if user.Name == "" || user.Name == user.Email { user.Name = fullName }
			user.IsActive = true
			// Ensure the user is associated with this IdP's organization
			if !user.OrganizationID.Valid || user.OrganizationID.UUID != idpModelFromDB.OrganizationID.UUID {
				user.OrganizationID = idpModelFromDB.OrganizationID
			}
			if saveErr := db.Save(&user).Error; saveErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update org Github SSO user: " + saveErr.Error()})
				return
			}
		}
		userOrgID = user.OrganizationID // Should be valid and match idpModelFromDB.OrganizationID
	}

	// Generate Phoenix GRC JWT token
	jwtToken, jwtErr := auth.GenerateToken(&user, userOrgID)
	if jwtErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate session token: " + jwtErr.Error()})
		return
	}

	frontendRedirectURL := os.Getenv("FRONTEND_OAUTH2_CALLBACK_URL")
	if frontendRedirectURL == "" {
		frontendRedirectURL = appConfig.Cfg.AppRootURL // Use configured AppRootURL
		if frontendRedirectURL == "" {
			frontendRedirectURL = "/" // Fallback
		}
	}
	// Ensure no double slashes if frontendRedirectURL ends with / and APP_ROOT_URL also has it
	targetURL := fmt.Sprintf("%s/oauth2/callback?token=%s&sso_success=true&provider=%s", strings.TrimSuffix(frontendRedirectURL, "/"), jwtToken, strings.ReplaceAll(ssoProviderName, "global_", ""))
	c.Redirect(http.StatusFound, targetURL)
}
