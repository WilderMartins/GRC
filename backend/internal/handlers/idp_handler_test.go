package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"phoenixgrc/backend/internal/models"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// testOrgAdminID e testUserAdminID são definidos em main_test_handler.go (ou deveriam ser)
// Para este teste, vamos redefinir ou garantir que sejam acessíveis.
// Usaremos os mesmos testOrgID e testUserID de risk_handler_test.go, assumindo que main_test_handler.go os define.
// Se não, precisaremos de uma forma de inicializar estes valores aqui também.
// Por agora, vamos assumir que `testOrgID` e `testUserID` (com role admin) estão disponíveis.

func getRouterWithOrgAdminContext(userID uuid.UUID, orgID uuid.UUID, userRole models.UserRole) *gin.Engine {
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Set("organizationID", orgID)
		c.Set("userRole", userRole) // Ensure role is set for checkOrgAdmin
		c.Next()
	})
	return r
}


func TestCreateIdentityProviderHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Assuming testUserID has RoleAdmin for testOrgID
	router := getRouterWithOrgAdminContext(testUserID, testOrgID, models.RoleAdmin)
	router.POST("/orgs/:orgId/identity-providers", CreateIdentityProviderHandler) // Match route used in main.go

	// Valid ConfigJSON for SAML example
	validSamlConfig := json.RawMessage(`{"idp_entity_id":"http://test.idp","idp_sso_url":"http://test.idp/sso","idp_x509_cert":"-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----"}`)

	t.Run("Successful IdP creation", func(t *testing.T) {
		payload := IdentityProviderPayload{
			ProviderType: models.IDPTypeSAML,
			Name:         "Test SAML IdP",
			IsActive:     func(b bool) *bool { return &b }(true),
			ConfigJSON:   validSamlConfig,
		}
		body, _ := json.Marshal(payload)

		sqlMock.ExpectBegin()
		// Regex for INSERT INTO "identity_providers"
		// Columns: id, organization_id, provider_type, name, is_active, config_json, attribute_mapping_json, created_at, updated_at
		sqlMock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "identity_providers" ("id","organization_id","provider_type","name","is_active","config_json","attribute_mapping_json","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), testOrgID, payload.ProviderType, payload.Name, *payload.IsActive, string(payload.ConfigJSON), string(payload.AttributeMappingJSON), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New().String()))
		sqlMock.ExpectCommit()

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/orgs/%s/identity-providers", testOrgID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code, "Response code should be 201 Created: %s", rr.Body.String())

		var createdIdP models.IdentityProvider
		err := json.Unmarshal(rr.Body.Bytes(), &createdIdP)
		assert.NoError(t, err)
		assert.Equal(t, payload.Name, createdIdP.Name)
		assert.Equal(t, testOrgID, createdIdP.OrganizationID)
		assert.True(t, createdIdP.IsActive)

		assert.NoError(t, sqlMock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

	t.Run("Unauthorized - user not admin of target org", func(t *testing.T) {
		otherOrgID := uuid.New() // User from testOrgID trying to access otherOrgID
		routerOtherOrg := getRouterWithOrgAdminContext(testUserID, testOrgID, models.RoleUser) // User is 'user', not admin
		routerOtherOrg.POST("/orgs/:orgId/identity-providers", CreateIdentityProviderHandler)


		payload := IdentityProviderPayload{ /* ... */ Name: "Test", ProviderType: models.IDPTypeSAML, ConfigJSON: validSamlConfig}
		body, _ := json.Marshal(payload)

		// Trying to create for `otherOrgID` while authenticated as user of `testOrgID`
		// Or, user is part of `targetOrgID` but not an Admin/Manager.
		// Let's test the latter first, as checkOrgAdmin has that logic.

		// Scenario 1: User is not an admin/manager of the targetOrgID (even if it's their own org)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/orgs/%s/identity-providers", testOrgID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		// Use a router where the context user has 'RoleUser' for 'testOrgID'
		nonAdminRouter := getRouterWithOrgAdminContext(testUserID, testOrgID, models.RoleUser)
		nonAdminRouter.POST("/orgs/:orgId/identity-providers", CreateIdentityProviderHandler)
		nonAdminRouter.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusForbidden, rr.Code, "Should be forbidden if user role is not Admin/Manager")


		// Scenario 2: User from org A trying to create for org B (checkOrgAdmin also handles this)
		reqOrgMismatch, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/orgs/%s/identity-providers", otherOrgID.String()), bytes.NewBuffer(body))
		reqOrgMismatch.Header.Set("Content-Type", "application/json")
		rrOrgMismatch := httptest.NewRecorder()
		// Use the original router where user is admin of testOrgID, but path is for otherOrgID
		router.ServeHTTP(rrOrgMismatch, reqOrgMismatch)
		assert.Equal(t, http.StatusForbidden, rrOrgMismatch.Code, "Should be forbidden if orgId in path doesn't match token orgId")

		// No DB interaction expected for unauthorized cases if checks are done before DB calls.
	})

	t.Run("Invalid payload - missing name", func(t *testing.T) {
		payload := IdentityProviderPayload{ProviderType: models.IDPTypeSAML, ConfigJSON: validSamlConfig} // Missing Name
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/orgs/%s/identity-providers", testOrgID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestListIdentityProvidersHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := getRouterWithOrgAdminContext(testUserID, testOrgID, models.RoleAdmin)
	router.GET("/orgs/:orgId/identity-providers", ListIdentityProvidersHandler)

	t.Run("Successful list IdPs", func(t *testing.T) {
		mockIdps := []models.IdentityProvider{
			{ID: uuid.New(), OrganizationID: testOrgID, Name: "IdP 1", ProviderType: models.IDPTypeSAML, IsActive: true, ConfigJSON: "{}", CreatedAt: time.Now()},
			{ID: uuid.New(), OrganizationID: testOrgID, Name: "IdP 2", ProviderType: models.IDPTypeOAuth2Google, IsActive: false, ConfigJSON: "{}", CreatedAt: time.Now()},
		}

		rows := sqlmock.NewRows([]string{"id", "organization_id", "name", "provider_type", "is_active", "config_json", "created_at"}).
			AddRow(mockIdps[0].ID, mockIdps[0].OrganizationID, mockIdps[0].Name, mockIdps[0].ProviderType, mockIdps[0].IsActive, mockIdps[0].ConfigJSON, mockIdps[0].CreatedAt).
			AddRow(mockIdps[1].ID, mockIdps[1].OrganizationID, mockIdps[1].Name, mockIdps[1].ProviderType, mockIdps[1].IsActive, mockIdps[1].ConfigJSON, mockIdps[1].CreatedAt)

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "identity_providers" WHERE organization_id = $1`)).
			WithArgs(testOrgID).
			WillReturnRows(rows)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/orgs/%s/identity-providers", testOrgID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "Response code should be 200 OK: %s", rr.Body.String())
		var idps []models.IdentityProvider
		err := json.Unmarshal(rr.Body.Bytes(), &idps)
		assert.NoError(t, err)
		assert.Len(t, idps, 2)
		assert.Equal(t, mockIdps[0].Name, idps[0].Name)

		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}


// TODO: Add tests for GetIdentityProviderHandler, UpdateIdentityProviderHandler, DeleteIdentityProviderHandler
// These will follow similar patterns of setting up router, context, payload (if any),
// mocking DB interactions, making request, and asserting response & DB expectations.
// Remember to test authorization cases (e.g., user not admin, org mismatch) for each.
// For Update, test partial updates and full updates.
// For Delete, test successful deletion and "not found" cases.
