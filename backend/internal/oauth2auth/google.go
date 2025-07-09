package oauth2auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"phoenixgrc/backend/internal/auth" // For JWT token generation
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	googleOAuth2 "golang.org/x/oauth2/google"
	googleAPI "google.golang.org/api/oauth2/v2" // To get user info
	"google.golang.org/api/option"
	"gorm.io/gorm"
)

const googleOAuthStateCookie = "phoenixgrc_google_oauth_state"
var appRootURL string // To be initialized

// GoogleOAuthConfig defines fields for Google OAuth2 provider from IdentityProvider.ConfigJSON
type GoogleOAuthConfig struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURI  string   `json:"redirect_uri"` // Will be constructed: appRootURL + /auth/oauth2/{idpId}/callback
	Scopes       []string `json:"scopes"`       // e.g., ["email", "profile"]
}

// InitializeOAuth2GlobalConfig loads global OAuth2 settings like APP_ROOT_URL
func InitializeOAuth2GlobalConfig() error {
	appRootURL = os.Getenv("APP_ROOT_URL")
	if appRootURL == "" {
		return fmt.Errorf("APP_ROOT_URL environment variable not set (required for OAuth2 Redirect URIs)")
	}
	return nil
}


func getGoogleOAuthConfig(idpModel *models.IdentityProvider) (*oauth2.Config, *GoogleOAuthConfig, error) {
	if appRootURL == "" {
		return nil, nil, fmt.Errorf("OAuth2 global configuration (APP_ROOT_URL) not initialized")
	}

	var cfg GoogleOAuthConfig
	if err := json.Unmarshal([]byte(idpModel.ConfigJSON), &cfg); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal Google OAuth2 config from JSON: %w", err)
	}

	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, &cfg, fmt.Errorf("client_id or client_secret missing in Google OAuth2 config")
	}

	// Construct the RedirectURI dynamically based on the IdP ID
	// Example: http://localhost:8080/auth/oauth2/google/{idp_uuid}/callback
	// The {idp_uuid} part needs to be handled carefully if the IdP is identified by a path param.
	// For now, let's assume a generic callback path per provider type if idpId is not in the path,
	// or make it part of the state.
	// For this implementation, the idpId IS in the path.
	dynamicRedirectURI := fmt.Sprintf("%s/auth/oauth2/google/%s/callback", appRootURL, idpModel.ID.String())


	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{googleAPI.UserinfoEmailScope, googleAPI.UserinfoProfileScope} // Default scopes
	}

	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  dynamicRedirectURI, // Use the dynamically constructed one
		Scopes:       scopes,
		Endpoint:     googleOAuth2.Endpoint,
	}, &cfg, nil
}

// GoogleLoginHandler initiates the Google OAuth2 login flow.
func GoogleLoginHandler(c *gin.Context) {
	idpIDStr := c.Param("idpId") // Assuming idpId identifies the specific Google configuration
	idpID, err := uuid.Parse(idpIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid IdP ID format for Google OAuth2"})
		return
	}

	db := database.GetDB()
	var idpModel models.IdentityProvider
	err = db.Where("id = ? AND provider_type = ? AND is_active = ?", idpID, models.IDPTypeOAuth2Google, true).First(&idpModel).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Active Google OAuth2 provider configuration not found for ID: " + idpIDStr})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error fetching Google IdP config: " + err.Error()})
		return
	}

	oauthCfg, _, err := getGoogleOAuthConfig(&idpModel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to configure Google OAuth2: " + err.Error()})
		return
	}

	// Generate random state string for CSRF protection
	b := make([]byte, 16)
	_, err = rand.Read(b)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OAuth state: " + err.Error()})
		return
	}
	state := base64.URLEncoding.EncodeToString(b)

	// Store state in a short-lived cookie
	// The cookie's path should be specific enough if multiple OAuth providers are on the same domain.
	// For now, using a generic path.
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     googleOAuthStateCookie,
		Value:    state,
		Expires:  time.Now().Add(10 * time.Minute),
		HttpOnly: true,
		Path:     "/", // Be more specific if needed, e.g., "/auth/oauth2/google/" + idpIDStr
		Secure:   c.Request.TLS != nil, // True if HTTPS
		SameSite: http.SameSiteLaxMode,
	})

	// Include idpId in the state so we can retrieve the correct config in the callback
	// This is important if the callback URL is generic for all Google IdPs.
	// However, our callback URL /auth/oauth2/google/{idpId}/callback already has the idpId.
	// So, the state is purely for CSRF.

	redirectURL := oauthCfg.AuthCodeURL(state)
	c.Redirect(http.StatusFound, redirectURL)
}

// GoogleCallbackHandler handles the callback from Google after user authorization.
func GoogleCallbackHandler(c *gin.Context) {
	idpIDStr := c.Param("idpId")
	idpID, err := uuid.Parse(idpIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid IdP ID format in callback for Google OAuth2"})
		return
	}

	// Verify state cookie
	stateCookie, err := c.Cookie(googleOAuthStateCookie)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing OAuth state cookie"})
		return
	}
	// Clear the state cookie once used
	http.SetCookie(c.Writer, &http.Cookie{Name: googleOAuthStateCookie, Value: "", MaxAge: -1, Path: "/"})

	if c.Query("state") != stateCookie {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid OAuth state"})
		return
	}

	code := c.Query("code")
	if code == "" {
		// Handle error from Google (e.g., user denied access)
		// errorReason := c.Query("error")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "OAuth authorization code not found or access denied"})
		return
	}

	db := database.GetDB()
	var idpModel models.IdentityProvider
	err = db.Where("id = ? AND provider_type = ? AND is_active = ?", idpID, models.IDPTypeOAuth2Google, true).First(&idpModel).Error
	if err != nil {
		// Log this error server-side
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve IdP configuration during callback"})
		return
	}

	oauthCfg, _, err := getGoogleOAuthConfig(&idpModel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to re-configure Google OAuth2 for token exchange: " + err.Error()})
		return
	}

	// Exchange authorization code for a token
	token, err := oauthCfg.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange OAuth code for token: " + err.Error()})
		return
	}
	if !token.Valid() {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth token received is invalid or expired"})
		return
	}

	// Get user info from Google
	client := oauthCfg.Client(context.Background(), token)
	oauth2Service, err := googleAPI.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Google API service client: " + err.Error()})
		return
	}
	userInfo, err := oauth2Service.Userinfo.Get().Do()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info from Google: " + err.Error()})
		return
	}

	if userInfo.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email not provided by Google"})
		return
	}
	email := strings.ToLower(strings.TrimSpace(userInfo.Email))
	fullName := strings.TrimSpace(userInfo.Name)
	if fullName == "" {
		fullName = email // Fallback name
	}
	externalID := userInfo.Id // Google's unique ID for the user


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
			SSOProvider:    idpModel.Name,
			SocialLoginID:  externalID, // Store Google's User ID
			Role:           models.RoleUser,
		}
		if createErr := db.Create(&user).Error; createErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new OAuth2 user: " + createErr.Error()})
			return
		}
	} else { // User exists, update
		user.SSOProvider = idpModel.Name
		user.SocialLoginID = externalID
		if user.Name == "" { user.Name = fullName }
		if saveErr := db.Save(&user).Error; saveErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update OAuth2 user: " + saveErr.Error()})
			return
		}
	}

	// Generate Phoenix GRC JWT token
	jwtToken, jwtErr := auth.GenerateToken(&user, user.OrganizationID)
	if jwtErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate session token: " + jwtErr.Error()})
		return
	}

	// Redirect to frontend (similar to SAML)
	frontendRedirectURL := os.Getenv("FRONTEND_OAUTH2_CALLBACK_URL")
	if frontendRedirectURL == "" {
		frontendRedirectURL = os.Getenv("APP_ROOT_URL")
		if frontendRedirectURL == "" {
			frontendRedirectURL = "/"
		}
	}
	targetURL := fmt.Sprintf("%s?token=%s&sso_success=true&provider=google", frontendRedirectURL, jwtToken)
	c.Redirect(http.StatusFound, targetURL)
}
