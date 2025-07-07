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

func TestCreateRiskHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Prepare router and context
	router := getRouterWithAuthenticatedContext(testUserID, testOrgID)
	router.POST("/risks", CreateRiskHandler)

	t.Run("Successful risk creation", func(t *testing.T) {
		payload := RiskPayload{
			Title:       "Test Risk Title",
			Description: "Test Risk Description",
			Category:    models.CategoryTechnological,
			Impact:      models.ImpactMedium,    // Uses "Médio"
			Probability: models.ProbabilityHigh,    // Uses "Alto"
			Status:      models.StatusOpen,
			OwnerID:     testUserID.String(),
		}
		body, _ := json.Marshal(payload)

		// --- Mocking GORM Create ---
		// GORM typically does something like:
		// INSERT INTO "risks" ("id","organization_id","title",...) VALUES ('uuid', 'org_uuid', 'title',...) RETURNING "id"
		// The exact SQL can vary. Use logger.Info for GORM to see the generated SQL if needed.
		// For `BeforeCreate` hooks generating UUID, the ID in `VALUES` might be a placeholder or the actual generated one.
		// We'll mock based on the assumption that the ID is generated before the INSERT.

		// The regex needs to be flexible for UUIDs and timestamps.
		// sqlmock.AnyArg() is your friend here.
		sqlMock.ExpectBegin()
		sqlMock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "risks" ("id","organization_id","title","description","category","impact","probability","status","owner_id","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), testOrgID, payload.Title, payload.Description, payload.Category, payload.Impact, payload.Probability, payload.Status, testUserID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New().String())) // Mock returning the new ID
		sqlMock.ExpectCommit()
		// --- End Mocking ---

		req, _ := http.NewRequest(http.MethodPost, "/risks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code, "Response code should be 201 Created")

		var createdRisk models.Risk
		err := json.Unmarshal(rr.Body.Bytes(), &createdRisk)
		assert.NoError(t, err, "Should unmarshal response body")
		assert.Equal(t, payload.Title, createdRisk.Title)
		assert.Equal(t, testOrgID, createdRisk.OrganizationID)
		assert.Equal(t, testUserID, createdRisk.OwnerID) // Assuming owner is correctly set
		assert.NotEqual(t, uuid.Nil, createdRisk.ID, "Risk ID should not be Nil")

		// Ensure all expectations were met
		assert.NoError(t, sqlMock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

	t.Run("Invalid payload - missing title", func(t *testing.T) {
		payload := RiskPayload{Description: "Only description"}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPost, "/risks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		// No DB interaction expected, so no sqlmock expectations here.
	})
}

func TestGetRiskHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := getRouterWithAuthenticatedContext(testUserID, testOrgID)
	router.GET("/risks/:riskId", GetRiskHandler)

	t.Run("Successful get risk", func(t *testing.T) {
		mockRisk := models.Risk{
			ID:             testRiskID,
			OrganizationID: testOrgID,
			Title:          "Fetched Risk",
			Description:    "Details of fetched risk",
			OwnerID:        testUserID,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			Owner:          models.User{ID: testUserID, Name: "Test Owner"},
		}

		// Mocking GORM Preload("Owner").Where("id = ? AND organization_id = ?", riskID, orgID).First(&risk)
		// This involves two queries typically:
		// 1. SELECT * FROM "risks" WHERE id = 'risk_id' AND organization_id = 'org_id' LIMIT 1
		// 2. SELECT * FROM "users" WHERE "users"."id" = 'owner_id_from_risk'

		// Query for the risk itself
		rowsRisk := sqlmock.NewRows([]string{"id", "organization_id", "title", "description", "owner_id", "created_at", "updated_at"}).
			AddRow(mockRisk.ID, mockRisk.OrganizationID, mockRisk.Title, mockRisk.Description, mockRisk.OwnerID, mockRisk.CreatedAt, mockRisk.UpdatedAt)

		// Note: GORM's behavior with Preload can be complex. It might use `IN` for multiple parent records.
		// For a single record, it's typically `WHERE id = ?`.
		// The regex matching is crucial here.
		// For "Owner" preload, it will query the "users" table.
		// Adjust the fields based on your User model.
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE id = $1 AND organization_id = $2 ORDER BY "risks"."id" LIMIT $3`)).
			WithArgs(testRiskID, testOrgID, 1).
			WillReturnRows(rowsRisk)

		// Query for the preloaded Owner
		rowsOwner := sqlmock.NewRows([]string{"id", "name" /* add other relevant user fields */}).
			AddRow(mockRisk.OwnerID, mockRisk.Owner.Name)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1`)).
			WithArgs(mockRisk.OwnerID).
			WillReturnRows(rowsOwner)
		// --- End Mocking ---

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/risks/%s", testRiskID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "Response code should be 200 OK")
		var fetchedRisk models.Risk
		err := json.Unmarshal(rr.Body.Bytes(), &fetchedRisk)
		assert.NoError(t, err)
		assert.Equal(t, mockRisk.Title, fetchedRisk.Title)
		assert.Equal(t, mockRisk.ID, fetchedRisk.ID)
		assert.Equal(t, mockRisk.Owner.Name, fetchedRisk.Owner.Name, "Owner name should be preloaded")


		assert.NoError(t, sqlMock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

	t.Run("Risk not found", func(t *testing.T) {
		nonExistentID := uuid.New()
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE id = $1 AND organization_id = $2`)).
			WithArgs(nonExistentID, testOrgID).
			WillReturnError(gorm.ErrRecordNotFound) // Simulate GORM's record not found

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/risks/%s", nonExistentID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

// TODO: Add tests for ListRisksHandler, UpdateRiskHandler, DeleteRiskHandler
// These will follow similar patterns:
// 1. Setup router and authenticated context.
// 2. Define payload (for Update).
// 3. Mock database interactions using sqlmock.
//    - ListRisks: Expect a SELECT query, return multiple rows.
//    - UpdateRisk: Expect SELECT (to find record), then UPDATE. Return updated row.
//    - DeleteRisk: Expect SELECT (to find record), then DELETE.
// 4. Create HTTP request.
// 5. Record response.
// 6. Assert response code and body.
// 7. Assert sqlMock.ExpectationsWereMet().
// Consider edge cases like invalid input, unauthorized attempts (if adding role checks), etc.
