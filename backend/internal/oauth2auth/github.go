package oauth2auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"phoenixgrc/backend/internal/auth"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	githubOAuth2 "golang.org/x/oauth2/github"
	"gorm.io/gorm"
)

const githubOAuthStateCookie = "phoenixgrc_github_oauth_state"

// GithubOAuthConfig defines fields for Github OAuth2 provider from IdentityProvider.ConfigJSON
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

func getGithubOAuthConfig(idpModel *models.IdentityProvider) (*oauth2.Config, *GithubOAuthConfig, error) {
	if appRootURL == "" { // appRootURL is initialized by InitializeOAuth2GlobalConfig
		return nil, nil, fmt.Errorf("OAuth2 global configuration (APP_ROOT_URL) not initialized")
	}

	var cfg GithubOAuthConfig
	if err := json.Unmarshal([]byte(idpModel.ConfigJSON), &cfg); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal Github OAuth2 config from JSON: %w", err)
	}

	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, &cfg, fmt.Errorf("client_id or client_secret missing in Github OAuth2 config")
	}

	dynamicRedirectURI := fmt.Sprintf("%s/auth/oauth2/github/%s/callback", appRootURL, idpModel.ID.String())

	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"read:user", "user:email"} // Default scopes
	}

	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  dynamicRedirectURI,
		Scopes:       scopes,
		Endpoint:     githubOAuth2.Endpoint,
	}, &cfg, nil
}

// GithubLoginHandler initiates the Github OAuth2 login flow.
func GithubLoginHandler(c *gin.Context) {
	idpIDStr := c.Param("idpId")
	idpID, err := uuid.Parse(idpIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid IdP ID format for Github OAuth2"})
		return
	}

	db := database.GetDB()
	var idpModel models.IdentityProvider
	err = db.Where("id = ? AND provider_type = ? AND is_active = ?", idpID, models.IDPTypeOAuth2Github, true).First(&idpModel).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Active Github OAuth2 provider configuration not found for ID: " + idpIDStr})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error fetching Github IdP config: " + err.Error()})
		return
	}

	oauthCfg, _, err := getGithubOAuthConfig(&idpModel)
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
		Path:     "/",
		Secure:   c.Request.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	redirectURL := oauthCfg.AuthCodeURL(state)
	c.Redirect(http.StatusFound, redirectURL)
}

// GithubCallbackHandler handles the callback from Github after user authorization.
func GithubCallbackHandler(c *gin.Context) {
	idpIDStr := c.Param("idpId")
	idpID, err := uuid.Parse(idpIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid IdP ID format in callback for Github OAuth2"})
		return
	}

	stateCookie, err := c.Cookie(githubOAuthStateCookie)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing OAuth state cookie"})
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{Name: githubOAuthStateCookie, Value: "", MaxAge: -1, Path: "/"})

	if c.Query("state") != stateCookie {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid OAuth state"})
		return
	}

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "OAuth authorization code not found or access denied"})
		return
	}

	db := database.GetDB()
	var idpModel models.IdentityProvider
	err = db.Where("id = ? AND provider_type = ? AND is_active = ?", idpID, models.IDPTypeOAuth2Github, true).First(&idpModel).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve IdP configuration during callback"})
		return
	}

	oauthCfg, _, err := getGithubOAuthConfig(&idpModel)
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

	// Fetch primary user info
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
	// If primary email is not set or not public, try fetching from /user/emails
	if email == "" {
		reqEmails, _ := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
		respEmails, errEmails := client.Do(reqEmails)
		if errEmails != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user emails from Github: " + errEmails.Error()})
			return
		}
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
			if email == "" && len(ghEmails) > 0 { // Fallback to first verified email if no primary
				for _, e := range ghEmails {
					if e.Verified {
						email = e.Email
						break
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
		fullName = strings.TrimSpace(ghUser.Login) // Use username if Name is not set
	}
	if fullName == "" {
		fullName = email // Fallback name
	}
	externalID := fmt.Sprintf("%d", ghUser.ID) // Github's unique ID for the user (is int64)


	// --- User Provisioning/Login ---
	var user models.User
	err = db.Where("email = ? AND organization_id = ?", email, idpModel.OrganizationID).First(&user).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error fetching user: " + err.Error()})
		return
	}

	if err == gorm.ErrRecordNotFound { // User does not exist, provision
		user = models.User{
			OrganizationID: idpModel.OrganizationID,
			Name:           fullName,
			Email:          email,
			PasswordHash:   "OAUTH2_USER_NO_PASSWORD",
			SSOProvider:    idpModel.Name, // Store the friendly name of the IdP config
			SocialLoginID:  externalID,    // Store Github's User ID
			Role:           models.RoleUser,
		}
		if createErr := db.Create(&user).Error; createErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new OAuth2 user: " + createErr.Error()})
			return
		}
	} else { // User exists, update
		user.SSOProvider = idpModel.Name
		user.SocialLoginID = externalID
		if user.Name == "" { user.Name = fullName } // Update name if it was empty
		if saveErr := db.Save(&user).Error; saveErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update OAuth2 user: " + saveErr.Error()})
			return
		}
	}

	jwtToken, jwtErr := auth.GenerateToken(&user, user.OrganizationID)
	if jwtErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate session token: " + jwtErr.Error()})
		return
	}

	frontendRedirectURL := os.Getenv("FRONTEND_OAUTH2_CALLBACK_URL")
	if frontendRedirectURL == "" {
		frontendRedirectURL = os.Getenv("APP_ROOT_URL")
		if frontendRedirectURL == "" {
			frontendRedirectURL = "/"
		}
	}
	targetURL := fmt.Sprintf("%s?token=%s&sso_success=true&provider=github", frontendRedirectURL, jwtToken)
	c.Redirect(http.StatusFound, targetURL)
}
