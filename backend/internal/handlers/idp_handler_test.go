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
	"gorm.io/gorm" // Adicionado
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
	setupMockDB(t) // Adicionado
	gin.SetMode(gin.TestMode)
	// Assuming testUserID has RoleAdmin for testOrgID
	router := getRouterWithOrgAdminContext(testUserID, testOrgID, models.RoleAdmin)
	router.POST("/orgs/:orgId/identity-providers", CreateIdentityProviderHandler) // Match route used in main.go

	// Valid ConfigJSON for SAML example
	validSamlConfig := json.RawMessage(`{"idp_entity_id":"http://test.idp","idp_sso_url":"http://test.idp/sso","idp_x509_cert":"-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----"}`)

	t.Run("Successful IdP creation", func(t *testing.T) {
		isPublic := true // Testar com IsPublic = true
		payload := IdentityProviderPayload{
			ProviderType: models.IDPTypeSAML,
			Name:         "Test SAML IdP",
			IsActive:     func(b bool) *bool { return &b }(true),
			IsPublic:     &isPublic, // Adicionar IsPublic
			ConfigJSON:   validSamlConfig,
		}
		body, _ := json.Marshal(payload)

		sqlMock.ExpectBegin()
		// Adicionar "is_public" à query e aos args. A ordem exata depende de como o GORM gera.
		// Assumindo que é adicionado após "is_active" e antes de "config_json" na struct.
		// A query GORM para Create geralmente inclui todos os campos não-zero da struct.
		// A ordem no VALUES ($1, $2, ...) deve corresponder à ordem das colunas na query INSERT.
		// GORM pode ordenar as colunas alfabeticamente ou pela ordem na struct.
		// Para ser seguro, é melhor verificar a query gerada ou usar um regex mais flexível.
		// Por agora, vou adicionar "is_public" e o argumento correspondente.
		// COLUNAS: "id","organization_id","provider_type","name","is_active","is_public","config_json","attribute_mapping_json","created_at","updated_at"
		// VALORES: $1, $2,             $3,              $4,    $5,          $6,         $7,            $8,                       $9,         $10
		// O mock atual tem 9 VALUES, então "is_public" seria o $6 e os outros deslocados.
		// "id","organization_id","provider_type","name","is_active","is_public","config_json","attribute_mapping_json","created_at","updated_at"
		// $1,  $2,               $3,             $4,    $5,          $6,         $7,            $8,                      $9,         $10
		sqlMock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "identity_providers" ("id","organization_id","provider_type","name","is_active","is_public","config_json","attribute_mapping_json","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING "id"`)).
			WithArgs(
				sqlmock.AnyArg(), // id
				testOrgID,        // organization_id
				payload.ProviderType,
				payload.Name,
				*payload.IsActive,
				*payload.IsPublic, // Novo argumento
				string(payload.ConfigJSON),
				string(payload.AttributeMappingJSON), // Pode ser nil se não estiver no payload, mas o modelo GORM pode tentar inserir como string vazia ou NULL.
				sqlmock.AnyArg(), // created_at
				sqlmock.AnyArg(), // updated_at
			).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New().String()))
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
		assert.Equal(t, *payload.IsPublic, createdIdP.IsPublic) // Verificar IsPublic

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
	setupMockDB(t) // Adicionado
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

var testIdpID = uuid.New() // For Get, Update, Delete tests

func TestGetIdentityProviderHandler(t *testing.T) {
	setupMockDB(t) // Adicionado
	gin.SetMode(gin.TestMode)
	router := getRouterWithOrgAdminContext(testUserID, testOrgID, models.RoleAdmin)
	router.GET("/orgs/:orgId/identity-providers/:idpId", GetIdentityProviderHandler)

	t.Run("Successful get IdP", func(t *testing.T) {
		mockIdP := models.IdentityProvider{
			ID:             testIdpID,
			OrganizationID: testOrgID,
			Name:           "Test IdP Details",
			ProviderType:   models.IDPTypeSAML,
			IsActive:       true,
			ConfigJSON:     `{"entity_id":"test"}`,
		}
		rows := sqlmock.NewRows([]string{"id", "organization_id", "name", "provider_type", "is_active", "config_json"}).
			AddRow(mockIdP.ID, mockIdP.OrganizationID, mockIdP.Name, mockIdP.ProviderType, mockIdP.IsActive, mockIdP.ConfigJSON)

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "identity_providers" WHERE id = $1 AND organization_id = $2 ORDER BY "identity_providers"."id" LIMIT $3`)).
			WithArgs(testIdpID, testOrgID, 1).
			WillReturnRows(rows)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/orgs/%s/identity-providers/%s", testOrgID.String(), testIdpID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var idP models.IdentityProvider
		err := json.Unmarshal(rr.Body.Bytes(), &idP)
		assert.NoError(t, err)
		assert.Equal(t, mockIdP.Name, idP.Name)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("IdP not found", func(t *testing.T) {
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "identity_providers" WHERE id = $1 AND organization_id = $2`)).
			WithArgs(testIdpID, testOrgID).
			WillReturnError(gorm.ErrRecordNotFound)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/orgs/%s/identity-providers/%s", testOrgID.String(), testIdpID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("GetIdentityProviderHandler - Invalid idpId format", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/orgs/%s/identity-providers/%s", testOrgID.String(), "not-a-uuid"), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Invalid identity provider ID format")
	})
}

func TestUpdateIdentityProviderHandler(t *testing.T) {
	setupMockDB(t) // Adicionado
	gin.SetMode(gin.TestMode)
	router := getRouterWithOrgAdminContext(testUserID, testOrgID, models.RoleAdmin)
	router.PUT("/orgs/:orgId/identity-providers/:idpId", UpdateIdentityProviderHandler)

    // Valid ConfigJSON for SAML example for updates
	validSamlConfigUpdate := json.RawMessage(`{"idp_entity_id":"http://updated.idp","idp_sso_url":"http://updated.idp/sso"}`)

	t.Run("Successful IdP update", func(t *testing.T) {
		isActive := false
		isPublicUpdate := false // Testar atualização de IsPublic
		payload := IdentityProviderPayload{
			ProviderType: models.IDPTypeSAML,
			Name:         "Updated Test SAML IdP",
			IsActive:     &isActive,
			IsPublic:     &isPublicUpdate, // Adicionar IsPublic ao payload
			ConfigJSON:   validSamlConfigUpdate,
		}
		body, _ := json.Marshal(payload)

		originalIdP := models.IdentityProvider{ID: testIdpID, OrganizationID: testOrgID, Name: "Original Name", ProviderType: models.IDPTypeSAML, IsActive: true, IsPublic: true, ConfigJSON: "{}"}
		rows := sqlmock.NewRows([]string{"id", "organization_id", "name", "provider_type", "is_active", "is_public", "config_json"}).
			AddRow(originalIdP.ID, originalIdP.OrganizationID, originalIdP.Name, originalIdP.ProviderType, originalIdP.IsActive, originalIdP.IsPublic, originalIdP.ConfigJSON)

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "identity_providers" WHERE id = $1 AND organization_id = $2 ORDER BY "identity_providers"."id" LIMIT $3`)).
			WithArgs(testIdpID, testOrgID, 1).
			WillReturnRows(rows)

		sqlMock.ExpectBegin()
		// A query de UPDATE do GORM geralmente só inclui os campos que foram alterados ou todos os campos não-zero.
		// E a ordem dos SETs pode ser alfabética ou pela ordem da struct.
		// Para WithArgs, precisamos listar os valores na ordem que GORM os coloca na query.
		// Ex: "attribute_mapping_json","config_json","is_active","is_public","name","provider_type","updated_at","id","organization_id"
		// Vamos mockar os argumentos para os campos que estamos mudando: Name, IsActive, IsPublic, ConfigJSON, AttributeMappingJSON (se mudar), UpdatedAt, e o ID para o WHERE.
		// O GORM pode gerar a query de forma diferente. Se este mock falhar, precisaremos ver a query real.
		// A ordem no WithArgs para SET deve ser: ProviderType, Name, IsActive, IsPublic, ConfigJSON, AttributeMappingJSON (se houver), UpdatedAt (AnyArg), e para WHERE: ID, OrganizationID
		// O handler atualiza: ProviderType, Name, IsActive (se no payload), IsPublic (se no payload), ConfigJSON, AttributeMappingJSON
		// O mock atual `WithArgs(sqlmock.AnyArg(), payload.IsActive, payload.Name, testOrgID, payload.ProviderType, sqlmock.AnyArg(), testIdpID)` está incorreto na ordem e campos.
		// Corrigindo para uma ordem mais provável (alfabética dos campos atualizados) e incluindo IsPublic:
		// A query seria algo como: UPDATE "identity_providers" SET "config_json"=$1, "is_active"=$2, "is_public"=$3, "name"=$4, "provider_type"=$5, "updated_at"=$6 WHERE "id" = $7 AND "organization_id" = $8
		// (AttributeMappingJSON não está no payload deste teste)
		sqlMock.ExpectExec(regexp.QuoteMeta(`UPDATE "identity_providers" SET`)).
			WithArgs(
				string(payload.ConfigJSON), // config_json
				*payload.IsActive,          // is_active
				*payload.IsPublic,          // is_public
				payload.Name,               // name
				payload.ProviderType,       // provider_type
				sqlmock.AnyArg(),           // updated_at
				testIdpID,                  // WHERE id
				// testOrgID                // GORM pode não incluir organization_id no WHERE de um Save em um modelo já carregado
			).WillReturnResult(sqlmock.NewResult(0, 1))
		sqlMock.ExpectCommit()

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/orgs/%s/identity-providers/%s", testOrgID.String(), testIdpID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "Response: %s", rr.Body.String())
		var updatedIdP models.IdentityProvider
		err := json.Unmarshal(rr.Body.Bytes(), &updatedIdP)
		assert.NoError(t, err)
		assert.Equal(t, payload.Name, updatedIdP.Name)
		assert.False(t, updatedIdP.IsActive) // Checa se IsActive foi para false
		assert.Equal(t, *payload.IsPublic, updatedIdP.IsPublic) // Checa se IsPublic foi para false
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("UpdateIdentityProviderHandler - IdP not found", func(t *testing.T) {
		isActive := true
		localPayload := IdentityProviderPayload{Name: "Update NonExistent", ProviderType: models.IDPTypeSAML, IsActive: &isActive, ConfigJSON: json.RawMessage(`{}`)}
		localBody, _ := json.Marshal(localPayload)
		nonExistentIdpID := uuid.New()

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "identity_providers" WHERE id = $1 AND organization_id = $2`)).
			WithArgs(nonExistentIdpID, testOrgID).
			WillReturnError(gorm.ErrRecordNotFound)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/orgs/%s/identity-providers/%s", testOrgID.String(), nonExistentIdpID.String()), bytes.NewBuffer(localBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("UpdateIdentityProviderHandler - Invalid ConfigJSON", func(t *testing.T) {
		// isActive := true // Não usado diretamente para este teste de payload malformado
		// Payload com JSON inválido em ConfigJSON
		// payload := IdentityProviderPayload{Name: "Update Invalid JSON", ProviderType: models.IDPTypeSAML, IsActive: &isActive, ConfigJSON: json.RawMessage(`{"key": "value`)} // JSON incompleto
		// body, _ := json.Marshal(payload) // Removido - requestBodyJSON é usado diretamente
                                        // O teste deve ser sobre o backend recebendo um JSON string malformado.
                                        // Então, o payload JSON string enviado na requisição é o que importa.

        // Corrigindo: Montar o JSON da requisição manualmente para simular um JSON string malformado
        requestBodyJSON := `{"name": "Update Invalid JSON", "provider_type": "saml", "is_active": true, "config_json": "{malformed}"}`


		// Mock para buscar o IdP original (o handler busca antes de tentar atualizar)
		originalIdP := models.IdentityProvider{ID: testIdpID, OrganizationID: testOrgID}
		rows := sqlmock.NewRows([]string{"id"}).AddRow(originalIdP.ID)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "identity_providers" WHERE id = $1 AND organization_id = $2`)).
			WithArgs(testIdpID, testOrgID).
			WillReturnRows(rows)
        // Não esperamos Begin/Commit/Exec se a validação do payload do JSON falhar no handler

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/orgs/%s/identity-providers/%s", testOrgID.String(), testIdpID.String()), bytes.NewBuffer([]byte(requestBodyJSON)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Invalid ConfigJSON format")
		assert.NoError(t, sqlMock.ExpectationsWereMet()) // Verifica se o SELECT foi chamado, mas não o UPDATE
	})

    t.Run("UpdateIdentityProviderHandler - Invalid provider_type in payload", func(t *testing.T) {
		requestBodyJSON := `{"name": "Update Invalid Type", "provider_type": "invalid_type", "is_active": true, "config_json": "{}"}`
		requestBuffer := bytes.NewBuffer([]byte(requestBodyJSON))

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/orgs/%s/identity-providers/%s", testOrgID.String(), testIdpID.String()), requestBuffer) // Corrigido para usar requestBuffer
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Invalid request payload")
        // Nenhuma interação com o banco de dados esperada se o binding do payload principal falhar.
	})
}

func TestDeleteIdentityProviderHandler(t *testing.T) {
	setupMockDB(t) // Adicionado
	gin.SetMode(gin.TestMode)
	router := getRouterWithOrgAdminContext(testUserID, testOrgID, models.RoleAdmin)
	router.DELETE("/orgs/:orgId/identity-providers/:idpId", DeleteIdentityProviderHandler)

	t.Run("Successful IdP deletion", func(t *testing.T) {
		// Mock para buscar o IdP antes de deletar
		rows := sqlmock.NewRows([]string{"id"}).AddRow(testIdpID)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "identity_providers" WHERE id = $1 AND organization_id = $2 ORDER BY "identity_providers"."id" LIMIT $3`)).
			WithArgs(testIdpID, testOrgID, 1).
			WillReturnRows(rows)

		sqlMock.ExpectBegin()
		sqlMock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "identity_providers" WHERE id = $1 AND organization_id = $2`)).
			WithArgs(testIdpID, testOrgID).
			WillReturnResult(sqlmock.NewResult(0,1))
		sqlMock.ExpectCommit()

		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/orgs/%s/identity-providers/%s", testOrgID.String(), testIdpID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "Identity provider deleted successfully")
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("DeleteIdentityProviderHandler - IdP not found", func(t *testing.T) {
		nonExistentIdpID := uuid.New()
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "identity_providers" WHERE id = $1 AND organization_id = $2`)).
			WithArgs(nonExistentIdpID, testOrgID).
			WillReturnError(gorm.ErrRecordNotFound)

		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/orgs/%s/identity-providers/%s", testOrgID.String(), nonExistentIdpID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}
