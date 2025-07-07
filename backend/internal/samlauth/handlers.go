package samlauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"phoenixgrc/backend/internal/auth" // For JWT token generation
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	"strings" // For email normalization

	"github.com/crewjam/saml/samlsp"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getSAMLServiceProvider fetches IdP config and creates a samlsp.Middleware (which includes SP)
func getSAMLServiceProvider(c *gin.Context, idpID uuid.UUID) (*samlsp.Middleware, *models.IdentityProvider, error) {
	db := database.GetDB()
	var idpModel models.IdentityProvider

	if err := db.Where("id = ? AND provider_type = ? AND is_active = ?", idpID, models.IDPTypeSAML, true).First(&idpModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil, fmt.Errorf("active SAML identity provider with ID %s not found", idpID)
		}
		return nil, nil, fmt.Errorf("database error fetching SAML IdP: %w", err)
	}

	opts, err := GetSAMLServiceProviderOptions(&idpModel)
	if err != nil {
		return nil, &idpModel, fmt.Errorf("failed to get SAML SP options: %w", err)
	}

	// The samlsp.Middleware can handle /metadata and /acs routes.
	// It also provides RequireAccount to protect handlers.
	spMiddleware, err := samlsp.New(opts)
	if err != nil {
		return nil, &idpModel, fmt.Errorf("failed to create SAML SP middleware: %w", err)
	}

	// Store IdP model in context for ACS to use later for attribute mapping etc.
	// This is a bit of a workaround as samlsp.Middleware doesn't easily pass custom context through its own handlers.
	// An alternative is to have a map of idpID to idpModel globally or pass it differently.
	// For now, this illustrates the need. A cleaner way might be a custom SessionProvider for samlsp.
	// c.Set("samlIdPModel", &idpModel)


	return spMiddleware, &idpModel, nil
}

// MetadataHandler serves the SAML SP metadata.
func MetadataHandler(c *gin.Context) {
	idpIDStr := c.Param("idpId")
	idpID, err := uuid.Parse(idpIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid IdP ID format"})
		return
	}

	sp, _, err := getSAMLServiceProvider(c, idpID)
	if err != nil {
		// Log the detailed error for admin, show generic error to user.
		fmt.Printf("Error getting SAML SP for metadata (IdP ID: %s): %v\n", idpIDStr, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to configure SAML service provider."})
		return
	}
	sp.ServeMetadata(c.Writer, c.Request)
}

// ACSHandler (Assertion Consumer Service) handles the SAML response from the IdP.
// This is where the user is actually logged in or provisioned.
func ACSHandler(c *gin.Context) {
	idpIDStr := c.Param("idpId")
	idpID, err := uuid.Parse(idpIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid IdP ID format"})
		return
	}

	sp, idpModel, err := getSAMLServiceProvider(c, idpID)
	if err != nil {
		fmt.Printf("Error getting SAML SP for ACS (IdP ID: %s): %v\n", idpIDStr, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to configure SAML service provider for ACS."})
		return
	}

	// Parse the SAML response
	// The samlsp.Middleware's ServeHTTP method normally handles this.
	// To use it, we'd typically wrap our final handler with `sp.RequireAccount`.
	// Since we need to do custom user provisioning here, we might need to call parts of `samlsp` more directly
	// or use the session it creates. `samlsp.SessionFromContext` can get the assertion.

	// Let's try using the middleware's session to get assertion attributes.
	// The `samlsp.Middleware` itself doesn't directly expose a handler for ACS that we can just call.
	// It's designed to BE the handler or wrap others.
	// We are making this handler the direct target of the ACS URL.

	// The `samlsp.Middleware` has a `ServeACSRequest` method, but it's not public.
	// The typical way is `sp.RequireAccount(http.HandlerFunc(ourActualAppHandler))`
	// `ourActualAppHandler` would then find the session via `samlsp.SessionFromContext(r.Context())`

	// For Gin, we might need a custom adapter or to call ParseRequest directly.
	// Let's simulate what `RequireAccount` does to get the session.
	// This is becoming complex because we are not using the middleware as intended for a "final" handler.

	// Simpler approach for now: Assume samlsp.Middleware's default cookie session provider
	// is used. We need to handle the assertion manually.
	// This is a deviation from using `samlsp.Middleware` as a black box.

	assertion, err := sp.Parse nahezu(c.Request) // This is a simplified call; error handling and details matter
	if err != nil {
		// Check if it's ErrNoSAMLResponse, which means we should redirect to IdP
		if err == samlsp.ErrNoSAMLResponse {
			// This handler is the ACS, it should not redirect to IdP to initiate login.
			// That should be a separate /auth/saml/{idpId}/login endpoint.
			// If we get here, it means the IdP POSTed to ACS but something was wrong with the request itself.
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid SAML request to ACS: " + err.Error()})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to parse SAML assertion: " + err.Error()})
		return
	}

	// --- User Provisioning/Login ---
	// Extract attributes from assertion.Attributes
	// Common attributes: email, givenName, sn (surname), uid
	// The names of these attributes depend on the IdP configuration.
	// Use idpModel.AttributeMappingJSON to map them.

	var samlAttrs struct {
		Email     []string `json:"email"` // SAML attributes can be multi-valued
		FirstName []string `json:"firstName"`
		LastName  []string `json:"lastName"`
		UserID    []string `json:"uid"` // Or some other unique ID from IdP
	}

	// Example of how to get attributes (actual names depend on IdP)
	// This is a placeholder; actual attribute names will vary.
	// A more robust solution would use the AttributeMappingJSON from idpModel.
	// For now, we'll assume some common names.

	email := getFirstString(assertion.Attributes.Get("urn:oid:0.9.2342.19200300.100.1.3"), assertion.Attributes.Get("email"), assertion.Attributes.Get("mail"))
	firstName := getFirstString(assertion.Attributes.Get("urn:oid:2.5.4.42"), assertion.Attributes.Get("givenName"))
	lastName := getFirstString(assertion.Attributes.Get("urn:oid:2.5.4.4"), assertion.Attributes.Get("sn"))
	// externalID := getFirstString(assertion.Attributes.Get("urn:oid:0.9.2342.19200300.100.1.1"), assertion.Attributes.Get("uid"), assertion.NameID.Value)
	// Using Subject NameID as the primary external identifier if specific UID attribute isn't found/mapped.
	externalID := assertion.Subject.NameID.Value


	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email not found in SAML assertion"})
		return
	}
	email = strings.ToLower(strings.TrimSpace(email))

	if firstName == "" && lastName == "" { // Try to construct name from email if others are missing
		parts := strings.Split(email, "@")
		firstName = parts[0]
	}
	fullName := strings.TrimSpace(firstName + " " + lastName)


	db := database.GetDB()
	var user models.User
	err = db.Where("email = ? AND organization_id = ?", email, idpModel.OrganizationID).First(&user).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error fetching user: " + err.Error()})
		return
	}

	if err == gorm.ErrRecordNotFound { // User does not exist, provision a new one
		user = models.User{
			OrganizationID: idpModel.OrganizationID,
			Name:           fullName,
			Email:          email,
			PasswordHash:   "SAML_USER_NO_PASSWORD", // Users authenticated via SAML don't use local passwords
			SSOProvider:    idpModel.Name,           // Store the name of the IdP
			SocialLoginID:  externalID,              // Store the SAML NameID or a persistent UID from IdP
			Role:           models.RoleUser,         // Default role for new SSO users, can be configured
		}
		if createErr := db.Create(&user).Error; createErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new SAML user: " + createErr.Error()})
			return
		}
	} else { // User exists, update details if necessary (e.g., SSOProvider, SocialLoginID)
		user.SSOProvider = idpModel.Name
		user.SocialLoginID = externalID
		if user.Name == "" || user.Name == " " { // Update name if it was blank
			user.Name = fullName
		}
		if saveErr := db.Save(&user).Error; saveErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update SAML user: " + saveErr.Error()})
			return
		}
	}

	// Generate Phoenix GRC JWT token for the user
	jwtToken, jwtErr := auth.GenerateToken(&user, user.OrganizationID)
	if jwtErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate session token: " + jwtErr.Error()})
		return
	}

	// Redirect user to frontend with the token
	// The frontend URL should be configurable.
	// For now, assume a query parameter. A more secure way might be a short-lived session cookie
	// that the frontend then exchanges for the JWT.
	frontendRedirectURL := os.Getenv("FRONTEND_SAML_CALLBACK_URL")
	if frontendRedirectURL == "" {
		frontendRedirectURL = os.Getenv("APP_ROOT_URL") // Fallback to app root
		if frontendRedirectURL == "" {
			frontendRedirectURL = "/" // Absolute fallback
		}
	}

	// It's common to redirect and pass the token as a query parameter or fragment.
	// However, this can expose the token in browser history/logs.
	// A more secure pattern is often:
	// 1. ACS sets a secure, HttpOnly cookie with the JWT.
	// 2. ACS redirects to a frontend page.
	// 3. Frontend page makes an XHR to a `/api/me`-like endpoint which relies on the cookie for auth.
	//    This endpoint can then return user info and, if needed for JS state, the JWT payload (not the token itself).
	// Or, if frontend and backend are same-site, the cookie works directly for API calls.
	// For now, simple query param redirect:
	targetURL := fmt.Sprintf("%s?token=%s&sso_success=true", frontendRedirectURL, jwtToken)
	c.Redirect(http.StatusFound, targetURL)
}

// Helper to get first non-empty string from a list of SAML attributes (which can be multi-valued)
func getFirstString(values ...saml.AttributeValue) string {
	for _, v := range values {
		if v != nil && len(v.Values) > 0 && v.Values[0].Value != "" {
			return v.Values[0].Value
		}
	}
	return ""
}


// SAMLLoginHandler initiates the SAML login flow by redirecting to the IdP.
func SAMLLoginHandler(c *gin.Context) {
	idpIDStr := c.Param("idpId")
	idpID, err := uuid.Parse(idpIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid IdP ID format"})
		return
	}

	sp, _, err := getSAMLServiceProvider(c, idpID)
	if err != nil {
		fmt.Printf("Error getting SAML SP for Login (IdP ID: %s): %v\n", idpIDStr, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to configure SAML service provider for login."})
		return
	}

	// To initiate login, we need to generate an AuthnRequest and redirect the user.
	// The samlsp.Middleware's `HandleStartAuthFlow` method does this.
	// It's typically bound to `sp.BindingLocation(saml.HTTPRedirectBinding)`.
	// We can call it directly here.

	// The `Track` function is used to store information about the request
    // so that it can be recovered when the IdP redirects back to the ACS endpoint.
    // We can use the `RelayState` for this, which will be echoed back by the IdP.
    // For example, to redirect the user back to a specific page after login.
    // For now, we'll use a simple relay state or leave it empty.
    // relayState := c.Query("redirect_to") // Get a redirect_to query param if present

	// The `samlsp.ServeHTTP` or more specifically `samlsp.HandleStartAuthFlow`
	// should be called. This will build the AuthnRequest and redirect.
	// `samlsp.Middleware` itself is an `http.Handler`.
	// We need to ensure the request context for `ServeHTTP` is correctly set up
	// if it relies on values that `samlsp` puts there (like `samlsp.SessionTracker`).

	// The `samlsp.DefaultSessionProvider.CreateSession` is called by `ServeACSRequest`.
	// For `HandleStartAuthFlow`, it might use `TrackID` if you use `TrackRequests`.
	// Let's try to invoke the part that generates AuthnRequest.

	// A simpler way might be to get the redirect URL and redirect manually.
	// `sp.ServiceProvider.MakeAuthenticationRequest` creates the request.
	authnRequest, err := sp.ServiceProvider.MakeAuthenticationRequest(sp.ServiceProvider.GetSSOBindingLocation(saml.HTTPRedirectBinding))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create SAML AuthnRequest: " + err.Error()})
		return
	}

	// For HTTP-Redirect binding, the AuthnRequest is encoded and put into URL parameters.
	redirectURL, err := authnRequest.Redirect(saml.HTTPRedirectBinding, sp.ServiceProvider.Key, sp.ServiceProvider.Certificate, sp.ServiceProvider.Clock)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get SAML redirect URL: " + err.Error()})
		return
	}

	c.Redirect(http.StatusFound, redirectURL.String())
}

// TODO: Implement SAML SLO (Single Log-Out) Handler if needed.
// func SLOMetadataHandler(c *gin.Context) { ... }
// func SLOHandler(c *gin.Context) { ... }
