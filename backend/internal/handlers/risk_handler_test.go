package handlers

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"phoenixgrc/backend/internal/models"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Conteúdo de risk_handler_test.go ...
func TestCreateRiskHandler(t *testing.T) {
	setupMockDB(t)
	gin.SetMode(gin.TestMode)
	router := getRouterWithAuthenticatedContext(testUserID, testOrgID)
	router.POST("/risks", CreateRiskHandler)

	t.Run("Successful risk creation", func(t *testing.T) {
		payload := RiskPayload{
			Title:       "Test Risk Title",
			Description: "Test Risk Description",
			Category:    models.CategoryTechnological,
			Impact:      models.ImpactMedium,
			Probability: models.ProbabilityHigh,
			Status:      models.StatusOpen,
			OwnerID:     testUserID.String(),
		}
		body, _ := json.Marshal(payload)
		sqlMock.ExpectBegin()
		sqlMock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "risks" ("id","organization_id","title","description","category","impact","probability","status","owner_id","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), testOrgID, payload.Title, payload.Description, payload.Category, payload.Impact, payload.Probability, payload.Status, testUserID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New().String()))
		sqlMock.ExpectCommit()
		var ownerIDForNotification uuid.UUID
		if payload.OwnerID != "" {
			ownerIDForNotification, _ = uuid.Parse(payload.OwnerID)
		} else {
			ownerIDForNotification = testUserID
		}
		ownerRows := sqlmock.NewRows([]string{"id", "email"}).AddRow(ownerIDForNotification, "owner-for-created-risk@example.com")
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
			WithArgs(ownerIDForNotification, 1).
			WillReturnRows(ownerRows)
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
		assert.Equal(t, testUserID, createdRisk.OwnerID)
		assert.NotEqual(t, uuid.Nil, createdRisk.ID, "Risk ID should not be Nil")
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
	})
}

func createTestCSV(t *testing.T, content string) (multipart.File, *multipart.FileHeader, error) {
	t.Helper()
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	part, err := writer.CreateFormFile("file", "test_risks.csv")
	if err != nil {
		return nil, nil, err
	}
	_, err = io.Copy(part, strings.NewReader(content))
	if err != nil {
		return nil, nil, err
	}
	writer.Close()
	req := httptest.NewRequest("POST", "/somepath", &b)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	err = req.ParseMultipartForm(10 << 20)
	if err != nil {
		return nil, nil, err
	}
	file, header, err := req.FormFile("file")
	return file, header, err
}

func TestBulkUploadRisksCSVHandler(t *testing.T) {
	setupMockDB(t)
	gin.SetMode(gin.TestMode)
	router := getRouterWithAuthenticatedContext(testUserID, testOrgID)
	router.POST("/risks/bulk-upload-csv", BulkUploadRisksCSVHandler)

	t.Run("Successful bulk upload", func(t *testing.T) {
		csvContent := `title,description,category,impact,probability
Risk Alpha,Description for Alpha,tecnologico,Alto,Baixo
Risk Beta,Description for Beta,operacional,Médio,Crítico`
		_, fileHeader, err := createTestCSV(t, csvContent)
		assert.NoError(t, err)
		sqlMock.ExpectBegin()
		sqlMock.ExpectExec(`INSERT INTO "risks"`).
			WillReturnResult(sqlmock.NewResult(0, 2))
		sqlMock.ExpectCommit()
		bodyBuf := &bytes.Buffer{}
		writer := multipart.NewWriter(bodyBuf)
		part, err := writer.CreateFormFile("file", fileHeader.Filename)
		assert.NoError(t, err)
		_, err = io.Copy(part, strings.NewReader(csvContent))
		assert.NoError(t, err)
		writer.Close()
		req, _ := http.NewRequest(http.MethodPost, "/risks/bulk-upload-csv", bodyBuf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "Response: %s", rr.Body.String())
		var resp BulkUploadRisksResponse
		err = json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, 2, resp.SuccessfullyImported)
		assert.Empty(t, resp.FailedRows)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
    // ... (outros testes de bulk upload) ...
}

func TestGetRiskHandler(t *testing.T) {
    // ... (código do teste) ...
}

func TestListRisksHandler(t *testing.T) {
    // ... (código do teste) ...
}

func TestSubmitRiskForAcceptanceHandler(t *testing.T) {
    // ... (código do teste) ...
}

func TestApproveOrRejectRiskAcceptanceHandler(t *testing.T) {
    // ... (código do teste) ...
}

func TestUpdateRiskHandler(t *testing.T) {
    // ... (código do teste) ...
}

func TestDeleteRiskHandler(t *testing.T) {
    // ... (código do teste) ...
}

func TestGetRiskApprovalHistoryHandler(t *testing.T) {
    // ... (código do teste) ...
}


// --- Conteúdo de risk_stakeholder_handler_test.go ---

func TestAddRiskStakeholderHandler_Authorization(t *testing.T) {
	setupMockDB(t) // Assume a setup function similar to risk_handler_test.go
	gin.SetMode(gin.TestMode)

	riskID := uuid.New()
	riskOwnerID := uuid.New()
	adminUserID := uuid.New()
	managerUserID := uuid.New()
	regularUserID := uuid.New() // Não é owner, nem admin, nem manager
	stakeholderToAddID := uuid.New()

	testCases := []struct {
		name           string
		actingUserID   uuid.UUID
		actingUserRole models.UserRole
		expectedStatus int
		mockDB         func()
	}{
		{
			name:           "Success - by Owner",
			actingUserID:   riskOwnerID,
			actingUserRole: models.RoleUser,
			expectedStatus: http.StatusCreated,
			mockDB: func() {
				mockRiskFetchForStakeholder(riskID, riskOwnerID)
				mockStakeholderUserFetch(stakeholderToAddID, testOrgID)
				mockStakeholderCreate(riskID, stakeholderToAddID)
			},
		},
		{
			name:           "Success - by Admin",
			actingUserID:   adminUserID,
			actingUserRole: models.RoleAdmin,
			expectedStatus: http.StatusCreated,
			mockDB: func() {
				mockRiskFetchForStakeholder(riskID, riskOwnerID)
				mockStakeholderUserFetch(stakeholderToAddID, testOrgID)
				mockStakeholderCreate(riskID, stakeholderToAddID)
			},
		},
		{
			name:           "Success - by Manager",
			actingUserID:   managerUserID,
			actingUserRole: models.RoleManager,
			expectedStatus: http.StatusCreated,
			mockDB: func() {
				mockRiskFetchForStakeholder(riskID, riskOwnerID)
				mockStakeholderUserFetch(stakeholderToAddID, testOrgID)
				mockStakeholderCreate(riskID, stakeholderToAddID)
			},
		},
		{
			name:           "Forbidden - by Regular User (not owner)",
			actingUserID:   regularUserID,
			actingUserRole: models.RoleUser,
			expectedStatus: http.StatusForbidden,
			mockDB: func() {
				mockRiskFetchForStakeholder(riskID, riskOwnerID)
				// Nenhuma outra chamada ao DB esperada
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := getRouterWithAuthContext(tc.actingUserID, testOrgID, tc.actingUserRole)
			router.POST("/risks/:riskId/stakeholders", AddRiskStakeholderHandler)

			if tc.mockDB != nil {
				tc.mockDB()
			}

			payload := AddStakeholderPayload{UserID: stakeholderToAddID.String()}
			body, _ := json.Marshal(payload)

			req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/risks/%s/stakeholders", riskID), bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code, "Response code mismatch. Body: %s", rr.Body.String())
			if err := sqlMock.ExpectationsWereMet(); err != nil {
				t.Errorf("SQL mock expectations not met for %s: %s", tc.name, err)
			}
		})
	}
}


// --- Helpers para mocks de Stakeholder ---
func mockRiskFetchForStakeholder(riskID, ownerID uuid.UUID) {
	rows := sqlmock.NewRows([]string{"id", "organization_id", "owner_id"}).
		AddRow(riskID, testOrgID, ownerID)
	sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE id = $1 AND organization_id = $2 ORDER BY "risks"."id" LIMIT 1`)).
		WithArgs(riskID, testOrgID).
		WillReturnRows(rows)
}

func mockStakeholderUserFetch(userID, orgID uuid.UUID) {
	rows := sqlmock.NewRows([]string{"id", "organization_id"}).
		AddRow(userID, orgID)
	sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 AND organization_id = $2 ORDER BY "users"."id" LIMIT 1`)).
		WithArgs(userID, orgID).
		WillReturnRows(rows)
}

func mockStakeholderCreate(riskID, userID uuid.UUID) {
	sqlMock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "risk_stakeholders" ("risk_id","user_id","created_at") VALUES ($1,$2,$3) ON CONFLICT DO NOTHING RETURNING "risk_id"`)).
		WithArgs(riskID, userID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"risk_id"}).AddRow(riskID))
}

func TestRemoveRiskStakeholderHandler_Authorization(t *testing.T) {
	setupMockDB(t)
	gin.SetMode(gin.TestMode)

	riskID := uuid.New()
	riskOwnerID := uuid.New()
	adminUserID := uuid.New()
	managerUserID := uuid.New()
	regularUserID := uuid.New()
	stakeholderToRemoveID := uuid.New()

	testCases := []struct {
		name           string
		actingUserID   uuid.UUID
		actingUserRole models.UserRole
		expectedStatus int
		mockDB         func()
	}{
		{
			name:           "Success - by Owner",
			actingUserID:   riskOwnerID,
			actingUserRole: models.RoleUser,
			expectedStatus: http.StatusOK,
			mockDB: func() {
				mockRiskFetchForStakeholder(riskID, riskOwnerID)
				mockStakeholderDelete(riskID, stakeholderToRemoveID)
			},
		},
		{
			name:           "Success - by Admin",
			actingUserID:   adminUserID,
			actingUserRole: models.RoleAdmin,
			expectedStatus: http.StatusOK,
			mockDB: func() {
				mockRiskFetchForStakeholder(riskID, riskOwnerID)
				mockStakeholderDelete(riskID, stakeholderToRemoveID)
			},
		},
		{
			name:           "Forbidden - by Regular User (not owner)",
			actingUserID:   regularUserID,
			actingUserRole: models.RoleUser,
			expectedStatus: http.StatusForbidden,
			mockDB: func() {
				mockRiskFetchForStakeholder(riskID, riskOwnerID)
				// Nenhuma chamada de delete esperada
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := getRouterWithAuthContext(tc.actingUserID, testOrgID, tc.actingUserRole)
			router.DELETE("/risks/:riskId/stakeholders/:userId", RemoveRiskStakeholderHandler)

			if tc.mockDB != nil {
				tc.mockDB()
			}

			req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/risks/%s/stakeholders/%s", riskID, stakeholderToRemoveID), nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code, "Response code mismatch. Body: %s", rr.Body.String())
			if err := sqlMock.ExpectationsWereMet(); err != nil {
				t.Errorf("SQL mock expectations not met for %s: %s", tc.name, err)
			}
		})
	}
}

func mockStakeholderDelete(riskID, userID uuid.UUID) {
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "risk_stakeholders" WHERE risk_id = $1 AND user_id = $2`)).
		WithArgs(riskID, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	sqlMock.ExpectCommit()
}
// ... (outros helpers de mock) ...
// (omitindo o corpo dos testes existentes por brevidade)
// ... (o corpo completo dos testes existentes vai aqui) ...
// ... (o corpo completo dos novos testes vai aqui) ...
// ... (o corpo completo dos novos helpers vai aqui) ...
