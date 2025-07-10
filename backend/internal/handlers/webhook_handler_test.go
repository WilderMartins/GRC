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
	"gorm.io/gorm"
)

// testOrgAdminID, testUserID (admin/manager de testOrgID)
// testUserNonAdminID (usuário comum de testOrgID)
// testOrgID
// São assumidos como definidos em main_test_handler.go ou similar
var testWebhookID = uuid.New()

func TestCreateWebhookHandler(t *testing.T) {
	setupMockDB(t) // Adicionado
	gin.SetMode(gin.TestMode)
	router := getRouterWithOrgAdminContext(testUserID, testOrgID, models.RoleAdmin)
	router.POST("/orgs/:orgId/webhooks", CreateWebhookHandler)

	t.Run("Successful webhook creation", func(t *testing.T) {
		isActive := true
		payload := WebhookPayload{
			Name:       "Test Webhook",
			URL:        "https://example.com/hook",
			EventTypes: []string{string(models.EventTypeRiskCreated), string(models.EventTypeRiskStatusChanged)},
			IsActive:   &isActive,
		}
		body, _ := json.Marshal(payload)

		sqlMock.ExpectBegin()
		sqlMock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "webhook_configurations" ("id","organization_id","name","url","event_types","is_active","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), testOrgID, payload.Name, payload.URL, "risk_created,risk_status_changed", *payload.IsActive, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New().String()))
		sqlMock.ExpectCommit()

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/orgs/%s/webhooks", testOrgID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code, "Response: %s", rr.Body.String())
		var createdWebhook models.WebhookConfiguration
		err := json.Unmarshal(rr.Body.Bytes(), &createdWebhook)
		assert.NoError(t, err)
		assert.Equal(t, payload.Name, createdWebhook.Name)
		assert.Equal(t, payload.URL, createdWebhook.URL)
		assert.True(t, createdWebhook.IsActive)
		assert.Equal(t, "risk_created,risk_status_changed", createdWebhook.EventTypes)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Invalid payload - missing URL", func(t *testing.T) {
		isActive := true
		payload := WebhookPayload{Name: "No URL Webhook", EventTypes: []string{"risk_created"}, IsActive: &isActive}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/orgs/%s/webhooks", testOrgID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

    t.Run("Invalid payload - invalid event type", func(t *testing.T) {
		isActive := true
		payload := WebhookPayload{Name: "Invalid Event", URL: "https://example.com/hook", EventTypes: []string{"INVALID_EVENT_TYPE"}, IsActive: &isActive}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/orgs/%s/webhooks", testOrgID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
        assert.Contains(t, rr.Body.String(), "EventTypes[0]' Error:Field validation for 'EventTypes[0]' failed on the 'oneof' tag")
	})
}

func TestListWebhooksHandler(t *testing.T) {
	setupMockDB(t) // Adicionado
	gin.SetMode(gin.TestMode)
    // Listar pode ser permitido para qualquer usuário da organização
	router := getRouterWithOrgAdminContext(testUserID, testOrgID, models.RoleUser)
	router.GET("/orgs/:orgId/webhooks", ListWebhooksHandler)

	t.Run("Successful list webhooks", func(t *testing.T) {
		mockWebhooks := []models.WebhookConfiguration{
			{ID: uuid.New(), OrganizationID: testOrgID, Name: "WH1", URL: "http://h.com/1", EventTypes: "risk_created", IsActive: true, CreatedAt: time.Now()},
			{ID: uuid.New(), OrganizationID: testOrgID, Name: "WH2", URL: "http://h.com/2", EventTypes: "risk_status_changed", IsActive: false, CreatedAt: time.Now()},
		}
		rows := sqlmock.NewRows([]string{"id", "organization_id", "name", "url", "event_types", "is_active", "created_at"}).
			AddRow(mockWebhooks[0].ID, mockWebhooks[0].OrganizationID, mockWebhooks[0].Name, mockWebhooks[0].URL, mockWebhooks[0].EventTypes, mockWebhooks[0].IsActive, mockWebhooks[0].CreatedAt).
			AddRow(mockWebhooks[1].ID, mockWebhooks[1].OrganizationID, mockWebhooks[1].Name, mockWebhooks[1].URL, mockWebhooks[1].EventTypes, mockWebhooks[1].IsActive, mockWebhooks[1].CreatedAt)

		// Mock para Count (se ListWebhooksHandler implementar paginação no futuro)
        // sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "webhook_configurations" WHERE organization_id = $1`)).
		// 	WithArgs(testOrgID).
		// 	WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(len(mockWebhooks)))

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "webhook_configurations" WHERE organization_id = $1`)). // Adicionar LIMIT OFFSET se paginado
			WithArgs(testOrgID).
			WillReturnRows(rows)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/orgs/%s/webhooks", testOrgID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		// A resposta agora é uma PaginatedResponse com Items sendo []WebhookResponseItem
		var resp PaginatedResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		// assert.Equal(t, int64(len(mockWebhooks)), resp.TotalItems) // Se paginado
		assert.Len(t, resp.Items, len(mockWebhooks))
		// Para verificar o conteúdo de resp.Items, precisa fazer type assertion para []map[string]interface{} e depois para WebhookResponseItem

		// Exemplo de verificação mais profunda se necessário:
        // itemsAsMaps := resp.Items.([]interface{})
        // firstItemMap := itemsAsMaps[0].(map[string]interface{})
        // assert.Equal(t, mockWebhooks[0].Name, firstItemMap["name"])
        // assert.Contains(t, firstItemMap["event_types_list"], "risk_created")

		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

func TestGetWebhookHandler(t *testing.T) {
	setupMockDB(t) // Adicionado
	gin.SetMode(gin.TestMode)
	router := getRouterWithOrgAdminContext(testUserID, testOrgID, models.RoleUser)
	router.GET("/orgs/:orgId/webhooks/:webhookId", GetWebhookHandler)

	t.Run("Successful get webhook", func(t *testing.T) {
		mockWH := models.WebhookConfiguration{
			ID: testWebhookID, OrganizationID: testOrgID, Name: "Detail WH", URL: "http://d.com", EventTypes: "risk_created", IsActive: true,
		}
		rows := sqlmock.NewRows([]string{"id", "organization_id", "name", "url", "event_types", "is_active"}).
			AddRow(mockWH.ID, mockWH.OrganizationID, mockWH.Name, mockWH.URL, mockWH.EventTypes, mockWH.IsActive)

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "webhook_configurations" WHERE id = $1 AND organization_id = $2 ORDER BY "webhook_configurations"."id" LIMIT $3`)).
			WithArgs(testWebhookID, testOrgID, 1).WillReturnRows(rows)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/orgs/%s/webhooks/%s", testOrgID.String(), testWebhookID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		var respWH models.WebhookConfiguration
		json.Unmarshal(rr.Body.Bytes(), &respWH)
		assert.Equal(t, mockWH.Name, respWH.Name)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("GetWebhookHandler - Not Found", func(t *testing.T) {
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "webhook_configurations" WHERE id = $1 AND organization_id = $2`)).
			WithArgs(testWebhookID, testOrgID).
			WillReturnError(gorm.ErrRecordNotFound)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/orgs/%s/webhooks/%s", testOrgID.String(), testWebhookID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("GetWebhookHandler - Invalid webhookId format", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/orgs/%s/webhooks/%s", testOrgID.String(), "not-a-uuid"), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Invalid webhook ID format")
	})
}

func TestUpdateWebhookHandler(t *testing.T) {
	setupMockDB(t) // Adicionado
	gin.SetMode(gin.TestMode)
	router := getRouterWithOrgAdminContext(testUserID, testOrgID, models.RoleAdmin)
	router.PUT("/orgs/:orgId/webhooks/:webhookId", UpdateWebhookHandler)

	t.Run("Successful webhook update", func(t *testing.T) {
		isActive := false
		payload := WebhookPayload{
			Name: "Updated Webhook", URL: "https://updated.com/hook", EventTypes: []string{string(models.EventTypeRiskStatusChanged)}, IsActive: &isActive,
		}
		body, _ := json.Marshal(payload)

		originalWH := models.WebhookConfiguration{ID: testWebhookID, OrganizationID: testOrgID}
		rows := sqlmock.NewRows([]string{"id", "organization_id"}).AddRow(originalWH.ID, originalWH.OrganizationID)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "webhook_configurations" WHERE id = $1 AND organization_id = $2 ORDER BY "webhook_configurations"."id" LIMIT $3`)).
			WithArgs(testWebhookID, testOrgID, 1).WillReturnRows(rows)

		sqlMock.ExpectBegin()
		sqlMock.ExpectExec(regexp.QuoteMeta(`UPDATE "webhook_configurations" SET`)).
			WithArgs(sqlmock.AnyArg(), payload.IsActive, payload.Name, testOrgID, payload.URL, sqlmock.AnyArg(), testWebhookID). // EventTypes, UpdatedAt, ID
			WillReturnResult(sqlmock.NewResult(0,1))
		sqlMock.ExpectCommit()

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/orgs/%s/webhooks/%s", testOrgID.String(), testWebhookID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var respWH models.WebhookConfiguration
		json.Unmarshal(rr.Body.Bytes(), &respWH)
		assert.Equal(t, payload.Name, respWH.Name)
		assert.False(t, respWH.IsActive)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("UpdateWebhookHandler - Not Found", func(t *testing.T) {
		isActive := true
		payload := WebhookPayload{Name: "WH Update NF", URL: "http://nf.com", EventTypes: []string{"risk_created"}, IsActive: &isActive}
		body, _ := json.Marshal(payload)
		nonExistentID := uuid.New()

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "webhook_configurations" WHERE id = $1 AND organization_id = $2`)).
			WithArgs(nonExistentID, testOrgID).
			WillReturnError(gorm.ErrRecordNotFound)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/orgs/%s/webhooks/%s", testOrgID.String(), nonExistentID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("UpdateWebhookHandler - Invalid Payload (e.g., empty URL)", func(t *testing.T) {
		// O handler CreateWebhookHandler já testa payload inválido com URL vazia e event_types inválido.
		// A validação do payload para Update é a mesma.
		payload := WebhookPayload{Name: "Valid Name", URL: "", EventTypes: []string{"risk_created"}} // URL vazia
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/orgs/%s/webhooks/%s", testOrgID.String(), testWebhookID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req) // O binding do Gin deve falhar
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Invalid request payload")
	})
}

func TestDeleteWebhookHandler(t *testing.T) {
	setupMockDB(t) // Adicionado
	gin.SetMode(gin.TestMode)
	router := getRouterWithOrgAdminContext(testUserID, testOrgID, models.RoleAdmin)
	router.DELETE("/orgs/:orgId/webhooks/:webhookId", DeleteWebhookHandler)

	t.Run("Successful webhook deletion", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id"}).AddRow(testWebhookID)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "webhook_configurations" WHERE id = $1 AND organization_id = $2 ORDER BY "webhook_configurations"."id" LIMIT $3`)).
			WithArgs(testWebhookID, testOrgID, 1).WillReturnRows(rows)

		sqlMock.ExpectBegin()
		sqlMock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "webhook_configurations" WHERE id = $1 AND organization_id = $2`)).
			WithArgs(testWebhookID, testOrgID).
			WillReturnResult(sqlmock.NewResult(0,1))
		sqlMock.ExpectCommit()

		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/orgs/%s/webhooks/%s", testOrgID.String(), testWebhookID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "Webhook configuration deleted successfully")
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("DeleteWebhookHandler - Not Found", func(t *testing.T) {
		nonExistentID := uuid.New()
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "webhook_configurations" WHERE id = $1 AND organization_id = $2`)).
			WithArgs(nonExistentID, testOrgID).
			WillReturnError(gorm.ErrRecordNotFound)

		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/orgs/%s/webhooks/%s", testOrgID.String(), nonExistentID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}
