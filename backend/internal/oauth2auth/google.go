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

func getGoogleOAuthConfig(idpIDStr string, db *gorm.DB) (*oauth2.Config, *GoogleOAuthConfig, *models.IdentityProvider, error) {
	currentAppRootURL := appConfig.Cfg.AppRootURL
	if currentAppRootURL == "" {
		// Fallback for older initialization if appConfig is not yet fully propagated
		// This might happen if InitializeOAuth2GlobalConfig was called before appConfig was ready.
		// Ideally, appConfig.Cfg.AppRootURL should be the single source of truth.
		if appRootURL != "" {
			currentAppRootURL = appRootURL
		} else {
			return nil, nil, nil, fmt.Errorf("OAuth2 global configuration (APP_ROOT_URL) not initialized or empty")
		}
	}


	var cfg GoogleOAuthConfig
	var idpModelFromDB *models.IdentityProvider = nil

	dynamicRedirectURI := fmt.Sprintf("%s/auth/oauth2/google/%s/callback", currentAppRootURL, idpIDStr)

	if idpIDStr == GlobalIdPIdentifier {
		cfg.ClientID = appConfig.Cfg.GoogleClientID
		cfg.ClientSecret = appConfig.Cfg.GoogleClientSecret
		if len(cfg.Scopes) == 0 { // Default scopes for global
			cfg.Scopes = []string{googleAPI.UserinfoEmailScope, googleAPI.UserinfoProfileScope}
		}
		if cfg.ClientID == "" || cfg.ClientSecret == "" {
			return nil, &cfg, nil, fmt.Errorf("global Google OAuth2 (GOOGLE_CLIENT_ID/SECRET) not configured")
		}
	} else {
		idpID, err := uuid.Parse(idpIDStr)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("invalid IdP ID format for Google OAuth2: %s", idpIDStr)
		}
		var fetchedModel models.IdentityProvider
		err = db.Where("id = ? AND provider_type = ? AND is_active = ?", idpID, models.IDPTypeOAuth2Google, true).First(&fetchedModel).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, nil, nil, fmt.Errorf("active Google OAuth2 provider configuration not found for ID: %s", idpIDStr)
			}
			return nil, nil, nil, fmt.Errorf("database error fetching Google IdP config for ID %s: %w", idpIDStr, err)
		}
		idpModelFromDB = &fetchedModel

		if errUnmarshal := json.Unmarshal([]byte(idpModelFromDB.ConfigJSON), &cfg); errUnmarshal != nil {
			return nil, nil, idpModelFromDB, fmt.Errorf("failed to unmarshal Google OAuth2 config from JSON for IdP %s: %w", idpIDStr, errUnmarshal)
		}
		if cfg.ClientID == "" || cfg.ClientSecret == "" {
			return nil, &cfg, idpModelFromDB, fmt.Errorf("client_id or client_secret missing in Google OAuth2 config for IdP %s", idpIDStr)
		}
		if len(cfg.Scopes) == 0 { // Default scopes
			cfg.Scopes = []string{googleAPI.UserinfoEmailScope, googleAPI.UserinfoProfileScope}
		}
	}

	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  dynamicRedirectURI,
		Scopes:       cfg.Scopes,
		Endpoint:     googleOAuth2.Endpoint,
	}, &cfg, idpModelFromDB, nil
}

// GoogleLoginHandler initiates the Google OAuth2 login flow.
func GoogleLoginHandler(c *gin.Context) {
	idpIDStr := c.Param("idpId") // Pode ser UUID ou "global"
	db := database.GetDB()

	oauthCfg, _, _, err := getGoogleOAuthConfig(idpIDStr, db)
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
	idpIDStr := c.Param("idpId") // Pode ser UUID ou "global"

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

	db := database.GetDB() // db instance
	oauthCfg, _, idpModelFromDB, err := getGoogleOAuthConfig(idpIDStr, db) // Passar db
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
	var userOrgID uuid.NullUUID // Used for token generation; may be null for global IdP users not yet in an org
	ssoProviderName := "google" // Default for global

	if idpIDStr == GlobalIdPIdentifier {
		// Global IdP flow
		ssoProviderName = "global_google"
		err = db.Where("email = ? AND (sso_provider = ? OR social_login_id = ?)", email, ssoProviderName, externalID).First(&user).Error

		if err != nil && err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error fetching user for global Google login: " + err.Error()})
			return
		}

		if err == gorm.ErrRecordNotFound { // User does not exist
			if !appConfig.Cfg.AllowGlobalSSOUserCreation {
				c.JSON(http.StatusForbidden, gin.H{"error": "New user registration via global Google SSO is disabled. Please use an organization-specific login or contact support."})
				return
			}
			// Create a new user for global SSO
			orgIDForNewUser := uuid.NullUUID{}
			if appConfig.Cfg.DefaultOrganizationIDForGlobalSSO != "" {
				parsedOrgID, errParseOrg := uuid.Parse(appConfig.Cfg.DefaultOrganizationIDForGlobalSSO)
				if errParseOrg == nil {
					orgIDForNewUser = uuid.NullUUID{UUID: parsedOrgID, Valid: true}
				} else {
					log.Printf("Aviso: DEFAULT_ORGANIZATION_ID_FOR_GLOBAL_SSO ('%s') não é um UUID válido. Usuário será criado sem organização.", appConfig.Cfg.DefaultOrganizationIDForGlobalSSO)
				}
			}

			user = models.User{
				OrganizationID: orgIDForNewUser,
				Name:           fullName,
				Email:          email,
				PasswordHash:   "OAUTH2_USER_NO_PASSWORD",
				SSOProvider:    ssoProviderName,
				SocialLoginID:  externalID,
				Role:           models.RoleUser, // Default role for new global users
				IsActive:       true,            // Activate immediately
			}
			if createErr := db.Create(&user).Error; createErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new global Google SSO user: " + createErr.Error()})
				return
			}
		} else { // User exists, update
			user.SSOProvider = ssoProviderName
			user.SocialLoginID = externalID
			if user.Name == "" || user.Name == user.Email { user.Name = fullName } // Update name if it was a placeholder
			user.IsActive = true // Ensure user is active
			if saveErr := db.Save(&user).Error; saveErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update global Google SSO user: " + saveErr.Error()})
				return
			}
		}
		// For global users, OrganizationID in token will be based on user.OrganizationID (which might be null)
		userOrgID = user.OrganizationID

	} else { // Organization-specific IdP flow
		if idpModelFromDB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "IdP configuration missing for organization-specific Google login."})
			return
		}
		ssoProviderName = idpModelFromDB.Name // Use the actual name from the IdP config

		// Try to find user by email within the organization OR by social login ID if previously linked
		err = db.Where("(email = ? AND organization_id = ?) OR (social_login_id = ? AND organization_id = ?)",
			email, idpModelFromDB.OrganizationID, externalID, idpModelFromDB.OrganizationID).First(&user).Error

		if err != nil && err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error fetching user for org Google login: " + err.Error()})
			return
		}

		if err == gorm.ErrRecordNotFound { // User does not exist in this org, provision
			user = models.User{
				OrganizationID: idpModelFromDB.OrganizationID,
				Name:           fullName,
				Email:          email,
				PasswordHash:   "OAUTH2_USER_NO_PASSWORD",
				SSOProvider:    ssoProviderName,
				SocialLoginID:  externalID,
				Role:           models.RoleUser, // Default role, could be customized based on IdP config later
				IsActive:       true,
			}
			if createErr := db.Create(&user).Error; createErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new org Google SSO user: " + createErr.Error()})
				return
			}
		} else { // User exists, update
			user.SSOProvider = ssoProviderName
			user.SocialLoginID = externalID
			if user.Name == "" || user.Name == user.Email { user.Name = fullName }
			user.IsActive = true
			// Ensure the user is associated with this IdP's organization if they somehow existed without it but matched email
			if !user.OrganizationID.Valid || user.OrganizationID.UUID != idpModelFromDB.OrganizationID.UUID {
				user.OrganizationID = idpModelFromDB.OrganizationID
			}
			if saveErr := db.Save(&user).Error; saveErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update org Google SSO user: " + saveErr.Error()})
				return
			}
		}
		userOrgID = user.OrganizationID // Should be valid and match idpModelFromDB.OrganizationID
	}

	// Generate Phoenix GRC JWT token
	// Pass userOrgID which might be null for global users not yet part of an org.
	// The GenerateToken function should handle a potentially nil OrganizationID for the token claims.
	jwtToken, jwtErr := auth.GenerateToken(&user, userOrgID)
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
