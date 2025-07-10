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

	t.Run("Missing required header", func(t *testing.T) {
		csvContent := `title,description,category,IMPACT_MALFORMED,probability`
		_, fileHeader, err := createTestCSV(t, csvContent)
		assert.NoError(t, err)
		bodyBuf := &bytes.Buffer{}
		writer := multipart.NewWriter(bodyBuf)
		part, _ := writer.CreateFormFile("file", fileHeader.Filename)
		io.Copy(part, strings.NewReader(csvContent))
		writer.Close()
		req, _ := http.NewRequest(http.MethodPost, "/risks/bulk-upload-csv", bodyBuf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Missing required CSV header: impact")
	})

	t.Run("Empty CSV file", func(t *testing.T) {
		csvContent := ``
		_, fileHeader, err := createTestCSV(t, csvContent)
		assert.NoError(t, err)
		bodyBuf := &bytes.Buffer{}
		writer := multipart.NewWriter(bodyBuf)
		part, _ := writer.CreateFormFile("file", fileHeader.Filename)
		io.Copy(part, strings.NewReader(csvContent))
		writer.Close()
		req, _ := http.NewRequest(http.MethodPost, "/risks/bulk-upload-csv", bodyBuf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "CSV file is empty")
	})

	t.Run("CSV with only headers", func(t *testing.T) {
		csvContent := `title,description,category,impact,probability`
		_, fileHeader, err := createTestCSV(t, csvContent)
		assert.NoError(t, err)
		bodyBuf := &bytes.Buffer{}
		writer := multipart.NewWriter(bodyBuf)
		part, _ := writer.CreateFormFile("file", fileHeader.Filename)
		io.Copy(part, strings.NewReader(csvContent))
		writer.Close()
		req, _ := http.NewRequest(http.MethodPost, "/risks/bulk-upload-csv", bodyBuf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		var resp BulkUploadRisksResponse
		err = json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, 0, resp.SuccessfullyImported)
		assert.Empty(t, resp.FailedRows)
	})

	t.Run("CSV with some valid and some invalid rows", func(t *testing.T) {
		csvContent := `title,description,category,impact,probability
Risk Valid,Valid desc,tecnologico,Baixo,Médio
,Invalid - no title,,Crítico,Alto
Risk Valid 2,Valid desc 2,operacional,Médio,Baixo
Risk Invalid Impact,Desc,legal,SUPER ALTO,Médio`
		_, fileHeader, err := createTestCSV(t, csvContent)
		assert.NoError(t, err)
		sqlMock.ExpectBegin()
		sqlMock.ExpectExec(`INSERT INTO "risks"`).
			WillReturnResult(sqlmock.NewResult(0, 2))
		sqlMock.ExpectCommit()
		bodyBuf := &bytes.Buffer{}
		writer := multipart.NewWriter(bodyBuf)
		part, _ := writer.CreateFormFile("file", fileHeader.Filename)
		io.Copy(part, strings.NewReader(csvContent))
		writer.Close()
		req, _ := http.NewRequest(http.MethodPost, "/risks/bulk-upload-csv", bodyBuf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusMultiStatus, rr.Code, "Response: %s", rr.Body.String())
		var resp BulkUploadRisksResponse
		err = json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, 2, resp.SuccessfullyImported)
		assert.Len(t, resp.FailedRows, 2)
		assert.Equal(t, 3, resp.FailedRows[0].LineNumber)
		assert.Contains(t, resp.FailedRows[0].Errors[0], "title is required")
		assert.Equal(t, 5, resp.FailedRows[1].LineNumber)
		assert.Contains(t, resp.FailedRows[1].Errors[0], "invalid impact value")
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

    t.Run("Invalid category uses default", func(t *testing.T) {
		csvContent := `title,description,category,impact,probability
Risk Cat,Desc Cat,INVALID_CATEGORY,Alto,Baixo`
		_, fileHeader, err := createTestCSV(t, csvContent)
		assert.NoError(t, err)
		sqlMock.ExpectBegin()
		sqlMock.ExpectExec(`INSERT INTO "risks"`).
            WithArgs(sqlmock.AnyArg(), testOrgID, "Risk Cat", "Desc Cat", string(models.CategoryTechnological), string(models.ImpactHigh), string(models.ProbabilityLow), string(models.StatusOpen), testUserID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		sqlMock.ExpectCommit()
		bodyBuf := &bytes.Buffer{}
		writer := multipart.NewWriter(bodyBuf)
		part, _ := writer.CreateFormFile("file", fileHeader.Filename)
		io.Copy(part, strings.NewReader(csvContent))
		writer.Close()
		req, _ := http.NewRequest(http.MethodPost, "/risks/bulk-upload-csv", bodyBuf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusMultiStatus, rr.Code, "Response: %s", rr.Body.String())
		var resp BulkUploadRisksResponse
		err = json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, 1, resp.SuccessfullyImported)
        assert.Len(t, resp.FailedRows, 1)
        assert.Equal(t, 2, resp.FailedRows[0].LineNumber)
        assert.Contains(t, resp.FailedRows[0].Errors[0], "invalid category: 'invalid_category'")
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

func TestGetRiskHandler(t *testing.T) {
	setupMockDB(t)
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
		rowsRisk := sqlmock.NewRows([]string{"id", "organization_id", "title", "description", "owner_id", "created_at", "updated_at"}).
			AddRow(mockRisk.ID, mockRisk.OrganizationID, mockRisk.Title, mockRisk.Description, mockRisk.OwnerID, mockRisk.CreatedAt, mockRisk.UpdatedAt)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE id = $1 AND organization_id = $2 ORDER BY "risks"."id" LIMIT $3`)).
			WithArgs(testRiskID, testOrgID, 1).
			WillReturnRows(rowsRisk)
		rowsOwner := sqlmock.NewRows([]string{"id", "name"}).
			AddRow(mockRisk.OwnerID, mockRisk.Owner.Name)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1`)).
			WithArgs(mockRisk.OwnerID).
			WillReturnRows(rowsOwner)
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
			WillReturnError(gorm.ErrRecordNotFound)
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/risks/%s", nonExistentID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

func TestListRisksHandler(t *testing.T) {
	setupMockDB(t) // Adicionado
	gin.SetMode(gin.TestMode)
	router := getRouterWithAuthenticatedContext(testUserID, testOrgID)
	router.GET("/risks", ListRisksHandler)

	defaultPage := 1
	defaultPageSize := 10

	t.Run("Successful list risks - no filters, default pagination", func(t *testing.T) {
		mockRisks := []models.Risk{
			{ID: uuid.New(), OrganizationID: testOrgID, Title: "Risk A", OwnerID: testUserID, CreatedAt: time.Now()},
			{ID: uuid.New(), OrganizationID: testOrgID, Title: "Risk B", OwnerID: testUserID, CreatedAt: time.Now().Add(-time.Hour)},
		}
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "risks" WHERE organization_id = $1`)).
			WithArgs(testOrgID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(len(mockRisks)))
		rows := sqlmock.NewRows([]string{"id", "organization_id", "title", "owner_id", "created_at"}).
			AddRow(mockRisks[0].ID, mockRisks[0].OrganizationID, mockRisks[0].Title, mockRisks[0].OwnerID, mockRisks[0].CreatedAt).
			AddRow(mockRisks[1].ID, mockRisks[1].OrganizationID, mockRisks[1].Title, mockRisks[1].OwnerID, mockRisks[1].CreatedAt)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE organization_id = $1 ORDER BY created_at desc LIMIT $2 OFFSET $3`)).
			WithArgs(testOrgID, defaultPageSize, (defaultPage-1)*defaultPageSize).
			WillReturnRows(rows)
		ownerIDs := []uuid.UUID{mockRisks[0].OwnerID, mockRisks[1].OwnerID}
		var uniqueOwnerIDs []uuid.UUID
		tempOwnerMap := make(map[uuid.UUID]bool)
		for _, id := range ownerIDs {
			if _, value := tempOwnerMap[id]; !value {
				tempOwnerMap[id] = true
				uniqueOwnerIDs = append(uniqueOwnerIDs, id)
			}
		}
		ownerRows := sqlmock.NewRows([]string{"id", "name"})
		for _, uid := range uniqueOwnerIDs {
			ownerRows.AddRow(uid, "Owner "+uid.String()[:4])
		}
		var args []driver.Value
		for _, id := range uniqueOwnerIDs {
			args = append(args, id)
		}
		if len(uniqueOwnerIDs) > 0 {
			sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" IN ($1`)).
				WithArgs(args...).
				WillReturnRows(ownerRows)
		}
		req, _ := http.NewRequest(http.MethodGet, "/risks", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "Response: %s", rr.Body.String())
		var resp PaginatedResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, int64(len(mockRisks)), resp.TotalItems)
		assert.Len(t, resp.Items, len(mockRisks))
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Successful list risks - with pagination params", func(t *testing.T) {
		page := 2
		pageSize := 1
		totalDBRisks := 2
		mockRiskForPage2 := models.Risk{ID: uuid.New(), OrganizationID: testOrgID, Title: "Risk Page 2 Item", OwnerID: testUserID}
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "risks" WHERE organization_id = $1`)).
			WithArgs(testOrgID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(totalDBRisks))
		rows := sqlmock.NewRows([]string{"id", "organization_id", "title", "owner_id"}).
			AddRow(mockRiskForPage2.ID, mockRiskForPage2.OrganizationID, mockRiskForPage2.Title, mockRiskForPage2.OwnerID)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE organization_id = $1 ORDER BY created_at desc LIMIT $2 OFFSET $3`)).
			WithArgs(testOrgID, pageSize, (page-1)*pageSize).
			WillReturnRows(rows)
		ownerRows := sqlmock.NewRows([]string{"id", "name"}).AddRow(mockRiskForPage2.OwnerID, "Owner Paged")
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1`)).
			WithArgs(mockRiskForPage2.OwnerID).
			WillReturnRows(ownerRows)
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/risks?page=%d&page_size=%d", page, pageSize), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		var resp PaginatedResponse
		json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.Equal(t, int64(totalDBRisks), resp.TotalItems)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, page, resp.Page)
		assert.Equal(t, pageSize, resp.PageSize)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Successful list risks - with status filter", func(t *testing.T) {
		filterStatus := models.StatusOpen
		mockRisksFiltered := []models.Risk{
			{ID: uuid.New(), OrganizationID: testOrgID, Title: "Open Risk 1", Status: filterStatus, OwnerID: testUserID},
		}
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "risks" WHERE organization_id = $1 AND status = $2`)).
			WithArgs(testOrgID, filterStatus).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		rows := sqlmock.NewRows([]string{"id", "organization_id", "title", "status", "owner_id"}).
			AddRow(mockRisksFiltered[0].ID, mockRisksFiltered[0].OrganizationID, mockRisksFiltered[0].Title, mockRisksFiltered[0].Status, mockRisksFiltered[0].OwnerID)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE organization_id = $1 AND status = $2 ORDER BY created_at desc LIMIT $3 OFFSET $4`)).
			WithArgs(testOrgID, filterStatus, defaultPageSize, 0).
			WillReturnRows(rows)
		ownerRows := sqlmock.NewRows([]string{"id", "name"}).AddRow(mockRisksFiltered[0].OwnerID, "Owner Filtered")
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1`)).
			WithArgs(mockRisksFiltered[0].OwnerID).
			WillReturnRows(ownerRows)
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/risks?status=%s", filterStatus), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		var resp PaginatedResponse
		json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.Equal(t, int64(1), resp.TotalItems)
		assert.Len(t, resp.Items, 1)
		itemsInterface := resp.Items.([]interface{})
		firstItemMap := itemsInterface[0].(map[string]interface{})
		assert.Equal(t, string(filterStatus), firstItemMap["status"])
		assert.Equal(t, mockRisksFiltered[0].Title, firstItemMap["title"])
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("List risks - empty result with filters", func(t *testing.T) {
		filterStatus := models.StatusMitigated
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "risks" WHERE organization_id = $1 AND status = $2`)).
			WithArgs(testOrgID, filterStatus).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE organization_id = $1 AND status = $2 ORDER BY created_at desc LIMIT $3 OFFSET $4`)).
			WithArgs(testOrgID, filterStatus, defaultPageSize, 0).
			WillReturnRows(sqlmock.NewRows(nil))
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/risks?status=%s", filterStatus), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
		var resp PaginatedResponse
		json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.Equal(t, int64(0), resp.TotalItems)
		assert.Len(t, resp.Items, 0)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
	// TODO: Adicionar testes para outros filtros (impact, probability, category) e combinações de filtros.
}

var testRiskForApprovalID = uuid.New()
var testRiskOwnerID = uuid.New()
var testManagerUserID = uuid.New()
var testApprovalWorkflowID = uuid.New()

func TestSubmitRiskForAcceptanceHandler(t *testing.T) {
	setupMockDB(t)
	gin.SetMode(gin.TestMode)
	router := getRouterWithOrgAdminContext(testManagerUserID, testOrgID, models.RoleManager)
	router.POST("/risks/:riskId/submit-acceptance", SubmitRiskForAcceptanceHandler)
	ownerUserRows := sqlmock.NewRows([]string{"id", "email"}).AddRow(testRiskOwnerID, "owner@example.com")
	managerUserRows := sqlmock.NewRows([]string{"id", "email"}).AddRow(testManagerUserID, "manager@example.com")

	t.Run("Successful submission for acceptance", func(t *testing.T) {
		mockRisk := models.Risk{
			ID:             testRiskForApprovalID,
			OrganizationID: testOrgID,
			Title:          "Risk to be Approved",
			OwnerID:        testRiskOwnerID,
			Status:         models.StatusOpen,
		}
		riskRows := sqlmock.NewRows([]string{"id", "organization_id", "title", "owner_id", "status"}).
			AddRow(mockRisk.ID, mockRisk.OrganizationID, mockRisk.Title, mockRisk.OwnerID, mockRisk.Status)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE id = $1 AND organization_id = $2`)).
			WithArgs(testRiskForApprovalID, testOrgID).
			WillReturnRows(riskRows)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "approval_workflows" WHERE risk_id = $1 AND status = $2`)).
			WithArgs(testRiskForApprovalID, models.ApprovalPending).
			WillReturnError(gorm.ErrRecordNotFound)
		sqlMock.ExpectBegin()
		sqlMock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "approval_workflows" ("id","risk_id","requester_id","approver_id","status","comments","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), testRiskForApprovalID, testManagerUserID, testRiskOwnerID, models.ApprovalPending, "", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(testApprovalWorkflowID))
		sqlMock.ExpectCommit()
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).WithArgs(testManagerUserID, 1).WillReturnRows(managerUserRows)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).WithArgs(testRiskOwnerID, 1).WillReturnRows(ownerUserRows)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/risks/%s/submit-acceptance", testRiskForApprovalID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusCreated, rr.Code, "Response: %s", rr.Body.String())
		var awf models.ApprovalWorkflow
		err := json.Unmarshal(rr.Body.Bytes(), &awf)
		assert.NoError(t, err)
		assert.Equal(t, testRiskForApprovalID, awf.RiskID)
		assert.Equal(t, testManagerUserID, awf.RequesterID)
		assert.Equal(t, testRiskOwnerID, awf.ApproverID)
		assert.Equal(t, models.ApprovalPending, awf.Status)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Fail if risk has no owner", func(t *testing.T) {
		mockRiskNoOwner := models.Risk{ID: testRiskForApprovalID, OrganizationID: testOrgID, Title: "No Owner Risk", OwnerID: uuid.Nil}
		riskRows := sqlmock.NewRows([]string{"id", "organization_id", "title", "owner_id"}).
			AddRow(mockRiskNoOwner.ID, mockRiskNoOwner.OrganizationID, mockRiskNoOwner.Title, mockRiskNoOwner.OwnerID)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE id = $1 AND organization_id = $2`)).
			WithArgs(testRiskForApprovalID, testOrgID).
			WillReturnRows(riskRows)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/risks/%s/submit-acceptance", testRiskForApprovalID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Risk must have an owner")
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Fail if already pending workflow exists", func(t *testing.T) {
		mockRisk := models.Risk{ID: testRiskForApprovalID, OrganizationID: testOrgID, Title: "Pending Risk", OwnerID: testRiskOwnerID}
		riskRows := sqlmock.NewRows([]string{"id", "organization_id", "title", "owner_id"}).
			AddRow(mockRisk.ID, mockRisk.OrganizationID, mockRisk.Title, mockRisk.OwnerID)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE id = $1 AND organization_id = $2`)).
			WithArgs(testRiskForApprovalID, testOrgID).
			WillReturnRows(riskRows)
		existingAWFRows := sqlmock.NewRows([]string{"id"}).AddRow(uuid.New())
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "approval_workflows" WHERE risk_id = $1 AND status = $2`)).
			WithArgs(testRiskForApprovalID, models.ApprovalPending).
			WillReturnRows(existingAWFRows)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/risks/%s/submit-acceptance", testRiskForApprovalID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusConflict, rr.Code)
		assert.Contains(t, rr.Body.String(), "already pending")
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Fail if user is not manager or admin", func(t *testing.T) {
		userRouter := getRouterWithOrgAdminContext(testUserID, testOrgID, models.RoleUser)
		userRouter.POST("/risks/:riskId/submit-acceptance", SubmitRiskForAcceptanceHandler)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/risks/%s/submit-acceptance", testRiskForApprovalID.String()), nil)
		rr := httptest.NewRecorder()
		userRouter.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusForbidden, rr.Code)
		assert.Contains(t, rr.Body.String(), "Only admins or managers can submit")
	})
}

func TestApproveOrRejectRiskAcceptanceHandler(t *testing.T) {
	setupMockDB(t)
	gin.SetMode(gin.TestMode)
	router := getRouterWithOrgAdminContext(testRiskOwnerID, testOrgID, models.RoleUser)
	router.POST("/risks/:riskId/approval/:approvalId/decide", ApproveOrRejectRiskAcceptanceHandler)

	t.Run("Successful approval", func(t *testing.T) {
		payload := DecisionPayload{Decision: models.ApprovalApproved, Comments: "Looks good to me."}
		body, _ := json.Marshal(payload)
		mockAWF := models.ApprovalWorkflow{
			ID:          testApprovalWorkflowID,
			RiskID:      testRiskForApprovalID,
			ApproverID:  testRiskOwnerID,
			Status:      models.ApprovalPending,
			Risk:        models.Risk{OrganizationID: testOrgID},
		}
		awfRows := sqlmock.NewRows([]string{"id", "risk_id", "approver_id", "status", "Risk__organization_id"}).
			AddRow(mockAWF.ID, mockAWF.RiskID, mockAWF.ApproverID, mockAWF.Status, mockAWF.Risk.OrganizationID)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT "approval_workflows"."id","approval_workflows"."risk_id","approval_workflows"."requester_id","approval_workflows"."approver_id","approval_workflows"."status","approval_workflows"."comments","approval_workflows"."created_at","approval_workflows"."updated_at","Risk"."id" AS "Risk__id","Risk"."organization_id" AS "Risk__organization_id","Risk"."title" AS "Risk__title","Risk"."description" AS "Risk__description","Risk"."category" AS "Risk__category","Risk"."impact" AS "Risk__impact","Risk"."probability" AS "Risk__probability","Risk"."status" AS "Risk__status","Risk"."owner_id" AS "Risk__owner_id","Risk"."created_at" AS "Risk__created_at","Risk"."updated_at" AS "Risk__updated_at" FROM "approval_workflows" LEFT JOIN "risks" "Risk" ON "approval_workflows"."risk_id" = "Risk"."id" WHERE "approval_workflows"."id" = $1 AND "approval_workflows"."risk_id" = $2 AND "Risk"."organization_id" = $3`)).
			WithArgs(testApprovalWorkflowID, testRiskForApprovalID, testOrgID).
			WillReturnRows(awfRows)
		sqlMock.ExpectBegin()
		sqlMock.ExpectExec(regexp.QuoteMeta(`UPDATE "approval_workflows" SET "risk_id"=$1,"requester_id"=$2,"approver_id"=$3,"status"=$4,"comments"=$5,"updated_at"=$6 WHERE "id" = $7`)).
			WithArgs(mockAWF.RiskID, sqlmock.AnyArg(), mockAWF.ApproverID, payload.Decision, payload.Comments, sqlmock.AnyArg(), mockAWF.ID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		riskToUpdateRows := sqlmock.NewRows([]string{"id", "status"}).AddRow(testRiskForApprovalID, models.StatusOpen)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE id = $1`)).
			WithArgs(testRiskForApprovalID).
			WillReturnRows(riskToUpdateRows)
		sqlMock.ExpectExec(regexp.QuoteMeta(`UPDATE "risks" SET "status"=$1,"updated_at"=$2 WHERE "id" = $3`)).
			WithArgs(models.StatusAccepted, sqlmock.AnyArg(), testRiskForApprovalID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		sqlMock.ExpectCommit()
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/risks/%s/approval/%s/decide", testRiskForApprovalID.String(), testApprovalWorkflowID.String()), bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "Response: %s", rr.Body.String())
		var updatedAWF models.ApprovalWorkflow
		err := json.Unmarshal(rr.Body.Bytes(), &updatedAWF)
		assert.NoError(t, err)
		assert.Equal(t, models.ApprovalApproved, updatedAWF.Status)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Successful rejection", func(t *testing.T) {
		payload := DecisionPayload{Decision: models.ApprovalRejected, Comments: "Not acceptable at this time."}
		body, _ := json.Marshal(payload)
		mockAWF := models.ApprovalWorkflow{
			ID:          testApprovalWorkflowID,
			RiskID:      testRiskForApprovalID,
			ApproverID:  testRiskOwnerID,
			RequesterID: testManagerUserID,
			Status:      models.ApprovalPending,
			Risk:        models.Risk{OrganizationID: testOrgID, Title: "Risk Title for Rejection"},
		}
		awfRows := sqlmock.NewRows([]string{"id", "risk_id", "approver_id", "requester_id", "status", "Risk__organization_id"}).
			AddRow(mockAWF.ID, mockAWF.RiskID, mockAWF.ApproverID, mockAWF.RequesterID, mockAWF.Status, mockAWF.Risk.OrganizationID)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT "approval_workflows"."id","approval_workflows"."risk_id","approval_workflows"."requester_id","approval_workflows"."approver_id","approval_workflows"."status","approval_workflows"."comments","approval_workflows"."created_at","approval_workflows"."updated_at","Risk"."id" AS "Risk__id","Risk"."organization_id" AS "Risk__organization_id","Risk"."title" AS "Risk__title","Risk"."description" AS "Risk__description","Risk"."category" AS "Risk__category","Risk"."impact" AS "Risk__impact","Risk"."probability" AS "Risk__probability","Risk"."status" AS "Risk__status","Risk"."owner_id" AS "Risk__owner_id","Risk"."created_at" AS "Risk__created_at","Risk"."updated_at" AS "Risk__updated_at" FROM "approval_workflows" LEFT JOIN "risks" "Risk" ON "approval_workflows"."risk_id" = "Risk"."id" WHERE "approval_workflows"."id" = $1 AND "approval_workflows"."risk_id" = $2 AND "Risk"."organization_id" = $3`)).
			WithArgs(testApprovalWorkflowID, testRiskForApprovalID, testOrgID).
			WillReturnRows(awfRows)
		sqlMock.ExpectBegin()
		sqlMock.ExpectExec(regexp.QuoteMeta(`UPDATE "approval_workflows" SET "risk_id"=$1,"requester_id"=$2,"approver_id"=$3,"status"=$4,"comments"=$5,"updated_at"=$6 WHERE "id" = $7`)).
			WithArgs(mockAWF.RiskID, mockAWF.RequesterID, mockAWF.ApproverID, payload.Decision, payload.Comments, sqlmock.AnyArg(), mockAWF.ID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		sqlMock.ExpectCommit()
		rejectedRiskRows := sqlmock.NewRows([]string{"id", "title"}).AddRow(mockAWF.RiskID, mockAWF.Risk.Title)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE id = $1 ORDER BY "risks"."id" LIMIT $2`)).
			WithArgs(mockAWF.RiskID, 1).
			WillReturnRows(rejectedRiskRows)
		requesterRows := sqlmock.NewRows([]string{"id", "email"}).AddRow(mockAWF.RequesterID, "manager@example.com")
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
			WithArgs(mockAWF.RequesterID, 1).
			WillReturnRows(requesterRows)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/risks/%s/approval/%s/decide", testRiskForApprovalID.String(), testApprovalWorkflowID.String()), bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "Response: %s", rr.Body.String())
		var updatedAWF models.ApprovalWorkflow
		err := json.Unmarshal(rr.Body.Bytes(), &updatedAWF)
		assert.NoError(t, err)
		assert.Equal(t, models.ApprovalRejected, updatedAWF.Status)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Fail if user not authorized (not approver)", func(t *testing.T) {
		anotherUserID := uuid.New()
		otherUserRouter := getRouterWithOrgAdminContext(anotherUserID, testOrgID, models.RoleUser)
		otherUserRouter.POST("/risks/:riskId/approval/:approvalId/decide", ApproveOrRejectRiskAcceptanceHandler)
		payload := DecisionPayload{Decision: models.ApprovalApproved, Comments: "Trying to approve."}
		body, _ := json.Marshal(payload)
		mockAWF := models.ApprovalWorkflow{
			ID: testApprovalWorkflowID, RiskID: testRiskForApprovalID, ApproverID: testRiskOwnerID,
			Status: models.ApprovalPending, Risk: models.Risk{OrganizationID: testOrgID},
		}
		awfRows := sqlmock.NewRows([]string{"id", "risk_id", "approver_id", "status", "Risk__organization_id"}).
			AddRow(mockAWF.ID, mockAWF.RiskID, mockAWF.ApproverID, mockAWF.Status, mockAWF.Risk.OrganizationID)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT "approval_workflows"."id","approval_workflows"."risk_id"`)).
			WithArgs(testApprovalWorkflowID, testRiskForApprovalID, testOrgID).
			WillReturnRows(awfRows)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/risks/%s/approval/%s/decide", testRiskForApprovalID.String(), testApprovalWorkflowID.String()), bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		otherUserRouter.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusForbidden, rr.Code)
		assert.Contains(t, rr.Body.String(), "You are not authorized to decide")
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Fail if workflow not pending", func(t *testing.T) {
		payload := DecisionPayload{Decision: models.ApprovalApproved}
		body, _ := json.Marshal(payload)
		mockAWF_Approved := models.ApprovalWorkflow{
			ID: testApprovalWorkflowID, RiskID: testRiskForApprovalID, ApproverID: testRiskOwnerID,
			Status: models.ApprovalApproved, Risk: models.Risk{OrganizationID: testOrgID},
		}
		awfRows_Approved := sqlmock.NewRows([]string{"id", "risk_id", "approver_id", "status", "Risk__organization_id"}).
			AddRow(mockAWF_Approved.ID, mockAWF_Approved.RiskID, mockAWF_Approved.ApproverID, mockAWF_Approved.Status, mockAWF_Approved.Risk.OrganizationID)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT "approval_workflows"."id","approval_workflows"."risk_id"`)).
			WithArgs(testApprovalWorkflowID, testRiskForApprovalID, testOrgID).
			WillReturnRows(awfRows_Approved)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/risks/%s/approval/%s/decide", testRiskForApprovalID.String(), testApprovalWorkflowID.String()), bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusConflict, rr.Code)
		assert.Contains(t, rr.Body.String(), "already been decided")
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

func TestUpdateRiskHandler(t *testing.T) {
	setupMockDB(t)
	gin.SetMode(gin.TestMode)
	router := getRouterWithAuthenticatedContext(testUserID, testOrgID)
	router.PUT("/risks/:riskId", UpdateRiskHandler)

	t.Run("Successful risk update - status changed", func(t *testing.T) {
		payload := RiskPayload{
			Title:       "Updated Risk Title",
			Description: "Updated Description",
			Category:    models.CategoryOperational,
			Impact:      models.ImpactCritical,
			Probability: models.ProbabilityCritical,
			Status:      models.StatusInProgress,
			OwnerID:     testRiskOwnerID.String(),
		}
		body, _ := json.Marshal(payload)
		originalRisk := models.Risk{
			ID:             testRiskID,
			OrganizationID: testOrgID,
			Title:          "Original Risk Title",
			Status:         models.StatusOpen,
			OwnerID:        testRiskOwnerID,
		}
		riskRows := sqlmock.NewRows([]string{"id", "organization_id", "title", "status", "owner_id"}).
			AddRow(originalRisk.ID, originalRisk.OrganizationID, originalRisk.Title, originalRisk.Status, originalRisk.OwnerID)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE id = $1 AND organization_id = $2 ORDER BY "risks"."id" LIMIT $3`)).
			WithArgs(testRiskID, testOrgID, 1).
			WillReturnRows(riskRows)
		sqlMock.ExpectBegin()
		sqlMock.ExpectExec(regexp.QuoteMeta(`UPDATE "risks" SET`)).
			WithArgs(payload.Category, payload.Description, payload.Impact, testOrgID, testRiskOwnerID, payload.Probability, payload.Status, payload.Title, sqlmock.AnyArg(), testRiskID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		sqlMock.ExpectCommit()
		ownerRows := sqlmock.NewRows([]string{"id", "name"}).AddRow(testRiskOwnerID, "Risk Owner Name")
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1`)).
			WithArgs(testRiskOwnerID).
			WillReturnRows(ownerRows)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
			WithArgs(testRiskOwnerID, 1).
			WillReturnRows(ownerRows)
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/risks/%s", testRiskID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "Response: %s", rr.Body.String())
		var updatedRisk models.Risk
		err := json.Unmarshal(rr.Body.Bytes(), &updatedRisk)
		assert.NoError(t, err)
		assert.Equal(t, payload.Title, updatedRisk.Title)
		assert.Equal(t, payload.Status, updatedRisk.Status)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Successful risk update - no status change, no notification", func(t *testing.T) {
		originalStatusUnchanged := models.StatusInProgress
		payload := RiskPayload{
			Title:       "Updated Risk Title No Status Change",
			Description: "Description for no status change",
			Status:      originalStatusUnchanged,
			OwnerID:     testRiskOwnerID.String(),
		}
		body, _ := json.Marshal(payload)
		originalRisk := models.Risk{
			ID:             testRiskID,
			OrganizationID: testOrgID,
			Title:          "Original Title for No Status Change",
			Status:         originalStatusUnchanged,
			OwnerID:        testRiskOwnerID,
		}
		riskRows := sqlmock.NewRows([]string{"id", "organization_id", "title", "status", "owner_id"}).
			AddRow(originalRisk.ID, originalRisk.OrganizationID, originalRisk.Title, originalRisk.Status, originalRisk.OwnerID)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE id = $1 AND organization_id = $2 ORDER BY "risks"."id" LIMIT $3`)).
			WithArgs(testRiskID, testOrgID, 1).
			WillReturnRows(riskRows)
		sqlMock.ExpectBegin()
		sqlMock.ExpectExec(regexp.QuoteMeta(`UPDATE "risks" SET`)).
			WithArgs(originalRisk.Category, payload.Description, originalRisk.Impact, testOrgID, testRiskOwnerID, originalRisk.Probability, payload.Status, payload.Title, sqlmock.AnyArg(), testRiskID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		sqlMock.ExpectCommit()
		ownerRows := sqlmock.NewRows([]string{"id", "name"}).AddRow(testRiskOwnerID, "Risk Owner Name")
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1`)).
			WithArgs(testRiskOwnerID).
			WillReturnRows(ownerRows)
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/risks/%s", testRiskID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "Response: %s", rr.Body.String())
		var updatedRisk models.Risk
		err := json.Unmarshal(rr.Body.Bytes(), &updatedRisk)
		assert.NoError(t, err)
		assert.Equal(t, payload.Title, updatedRisk.Title)
		assert.Equal(t, payload.Status, updatedRisk.Status)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("UpdateRiskHandler - Risk not found", func(t *testing.T) {
		payload := RiskPayload{Title: "Updated Title", Impact: models.ImpactLow, Probability: models.ProbabilityLow, Status: models.StatusOpen}
		body, _ := json.Marshal(payload)
		nonExistentRiskID := uuid.New()
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE id = $1 AND organization_id = $2`)).
			WithArgs(nonExistentRiskID, testOrgID).
			WillReturnError(gorm.ErrRecordNotFound)
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/risks/%s", nonExistentRiskID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("UpdateRiskHandler - Invalid payload", func(t *testing.T) {
		invalidPayloadJSON := `{"title": "Test", "impact": "INVALID_IMPACT", "probability": "Baixo", "status": "aberto"}`
		body := bytes.NewBuffer([]byte(invalidPayloadJSON))
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/risks/%s", testRiskID.String()), body)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Invalid request payload")
	})
}

func TestDeleteRiskHandler(t *testing.T) {
	setupMockDB(t)
	gin.SetMode(gin.TestMode)
	router := getRouterWithAuthenticatedContext(testUserID, testOrgID)
	router.DELETE("/risks/:riskId", DeleteRiskHandler)

	t.Run("Successful risk deletion", func(t *testing.T) {
		riskRows := sqlmock.NewRows([]string{"id"}).AddRow(testRiskID)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE id = $1 AND organization_id = $2 ORDER BY "risks"."id" LIMIT $3`)).
			WithArgs(testRiskID, testOrgID, 1).
			WillReturnRows(riskRows)
		sqlMock.ExpectBegin()
		sqlMock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "risks" WHERE "risks"."id" = $1`)).
			WithArgs(testRiskID).
			WillReturnResult(sqlmock.NewResult(0,1))
		sqlMock.ExpectCommit()
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/risks/%s", testRiskID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "Response: %s", rr.Body.String())
		assert.Contains(t, rr.Body.String(), "Risk deleted successfully")
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("DeleteRiskHandler - Risk not found", func(t *testing.T) {
		nonExistentRiskID := uuid.New()
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE id = $1 AND organization_id = $2`)).
			WithArgs(nonExistentRiskID, testOrgID).
			WillReturnError(gorm.ErrRecordNotFound)
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/risks/%s", nonExistentRiskID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

func TestGetRiskApprovalHistoryHandler(t *testing.T) {
	setupMockDB(t)
	gin.SetMode(gin.TestMode)
	router := getRouterWithAuthenticatedContext(testUserID, testOrgID)
	router.GET("/risks/:riskId/approval-history", GetRiskApprovalHistoryHandler)

	t.Run("Successful get approval history", func(t *testing.T) {
		riskRow := sqlmock.NewRows([]string{"id"}).AddRow(testRiskForApprovalID)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE id = $1 AND organization_id = $2`)).
			WithArgs(testRiskForApprovalID, testOrgID).
			WillReturnRows(riskRow)
		mockHistory := []models.ApprovalWorkflow{
			{ID: uuid.New(), RiskID: testRiskForApprovalID, RequesterID: testManagerUserID, ApproverID: testRiskOwnerID, Status: models.ApprovalApproved, CreatedAt: time.Now().Add(-time.Hour)},
			{ID: testApprovalWorkflowID, RiskID: testRiskForApprovalID, RequesterID: testManagerUserID, ApproverID: testRiskOwnerID, Status: models.ApprovalPending, CreatedAt: time.Now()},
		}
		historyRows := sqlmock.NewRows([]string{"id", "risk_id", "requester_id", "approver_id", "status", "created_at"}).
			AddRow(mockHistory[0].ID, mockHistory[0].RiskID, mockHistory[0].RequesterID, mockHistory[0].ApproverID, mockHistory[0].Status, mockHistory[0].CreatedAt).
			AddRow(mockHistory[1].ID, mockHistory[1].RiskID, mockHistory[1].RequesterID, mockHistory[1].ApproverID, mockHistory[1].Status, mockHistory[1].CreatedAt)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "approval_workflows" WHERE risk_id = $1 ORDER BY created_at desc`)).
			WithArgs(testRiskForApprovalID).
			WillReturnRows(historyRows)
		userRows := sqlmock.NewRows([]string{"id", "name", "email"}).
			AddRow(testManagerUserID, "Manager User", "manager@example.com").
			AddRow(testRiskOwnerID, "Owner User", "owner@example.com")
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" IN ($1,$2)`)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(userRows)
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/risks/%s/approval-history", testRiskForApprovalID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "Response: %s", rr.Body.String())
		var history []models.ApprovalWorkflow
		err := json.Unmarshal(rr.Body.Bytes(), &history)
		assert.NoError(t, err)
		assert.Len(t, history, 2)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}
