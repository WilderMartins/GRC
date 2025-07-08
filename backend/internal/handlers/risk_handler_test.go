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

		// Mock para buscar o Owner para notificação por email
		// (assumindo que payload.OwnerID é o testUserID neste caso ou um ID mockável)
		// Se OwnerID no payload for vazio, o owner será o testUserID (usuário que fez a request)
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


// --- Bulk Upload Risks CSV Handler Tests ---

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
	writer.Close() // Importante para finalizar o corpo multipart

	// Para simular um FormFile, precisamos de um http.Request com este corpo
	req := httptest.NewRequest("POST", "/somepath", &b)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	err = req.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		return nil, nil, err
	}

	file, header, err := req.FormFile("file")
	return file, header, err
}


func TestBulkUploadRisksCSVHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := getRouterWithAuthenticatedContext(testUserID, testOrgID) // Assume testUserID and testOrgID from main_test_handler.go
	router.POST("/risks/bulk-upload-csv", BulkUploadRisksCSVHandler)

	t.Run("Successful bulk upload", func(t *testing.T) {
		csvContent := `title,description,category,impact,probability
Risk Alpha,Description for Alpha,tecnologico,Alto,Baixo
Risk Beta,Description for Beta,operacional,Médio,Crítico`

		_, fileHeader, err := createTestCSV(t, csvContent)
		assert.NoError(t, err)

		sqlMock.ExpectBegin()
		// Esperamos duas inserções. O GORM pode fazer isso em uma única query `INSERT INTO ... VALUES (...), (...)`
		// ou múltiplas. `Create` com uma slice geralmente tenta uma única query otimizada.
		// A regex aqui precisa ser genérica o suficiente para cobrir a inserção de múltiplos registros.
		// Ou podemos esperar `ExpectExec` para cada inserção se o GORM fizer individualmente dentro da tx.
		// Para `tx.Create(&risksToCreate)`, GORM geralmente faz uma única query com múltiplos VALUES.
		// A regex exata para múltiplos VALUES pode ser complexa.
		// Vamos simplificar assumindo que o mock pode verificar o número de execuções ou um padrão mais genérico.
		// Uma abordagem comum é mockar `sqlmock.AnyArg()` para os valores e verificar o número de linhas afetadas.
		// Ou, se o driver suportar, o número de `Exec`s.
		// Com `pq` driver e GORM, `Create(&slice)` faz uma query `INSERT ... VALUES (...), (...), ...`.
		// O `regexp.QuoteMeta` não vai funcionar bem com isso.
		// Vamos usar `sqlmock.New οποιοδήποτεArg()` e verificar o número de argumentos ou o resultado.
		// No entanto, `ExpectQuery` não é usado para `INSERT` sem `RETURNING`. `ExpectExec` é mais apropriado.
		// Se `BeforeCreate` com `RETURNING id` estivesse em jogo para cada, seria `ExpectQuery`.
		// Como `risk.ID` é gerado antes do `Create(&risksToCreate)`, a query de Create não retorna ID.
		sqlMock.ExpectExec(`INSERT INTO "risks"`). // Regex mais genérica
			// WithArgs não é trivial para múltiplas inserções com uma única query.
			// Em vez disso, vamos confiar no resultado.
			WillReturnResult(sqlmock.NewResult(0, 2)) // 2 linhas afetadas
		sqlMock.ExpectCommit()

		// Criar o corpo da requisição multipart
		bodyBuf := &bytes.Buffer{}
		writer := multipart.NewWriter(bodyBuf)
		part, err := writer.CreateFormFile("file", fileHeader.Filename)
		assert.NoError(t, err)

		// Reabrir o arquivo simulado para copiar o conteúdo para o 'part'
		// Isso é um pouco artificial porque createTestCSV já "consumiu" o reader original.
		// Em um teste real, você teria o arquivo e o passaria.
		// Para este setup, vamos recriar o conteúdo do CSV para o 'part'.
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
		csvContent := `title,description,category,IMPACT_MALFORMED,probability` // Impact header errado
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
		// No DB interaction expected
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

		assert.Equal(t, http.StatusOK, rr.Code) // OK, mas 0 importados
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
		// Linha 2: OK
		// Linha 3: Erro (title vazio)
		// Linha 4: OK
		// Linha 5: Erro (impact inválido)

		_, fileHeader, err := createTestCSV(t, csvContent)
		assert.NoError(t, err)

		sqlMock.ExpectBegin()
		sqlMock.ExpectExec(`INSERT INTO "risks"`).
			WillReturnResult(sqlmock.NewResult(0, 2)) // Espera que 2 riscos sejam inseridos
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
		assert.Equal(t, 3, resp.FailedRows[0].LineNumber) // Linha 3 do CSV (após cabeçalho)
		assert.Contains(t, resp.FailedRows[0].Errors[0], "title is required")
		assert.Equal(t, 5, resp.FailedRows[1].LineNumber) // Linha 5 do CSV
		assert.Contains(t, resp.FailedRows[1].Errors[0], "invalid impact value")
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

    t.Run("Invalid category uses default", func(t *testing.T) {
		csvContent := `title,description,category,impact,probability
Risk Cat,Desc Cat,INVALID_CATEGORY,Alto,Baixo`

		_, fileHeader, err := createTestCSV(t, csvContent)
		assert.NoError(t, err)

		sqlMock.ExpectBegin()
        // Espera-se que o risco seja inserido com a categoria default (models.CategoryTechnological)
		sqlMock.ExpectExec(`INSERT INTO "risks"`).
            WithArgs(sqlmock.AnyArg(), testOrgID, "Risk Cat", "Desc Cat", string(models.CategoryTechnological), string(models.ImpactHigh), string(models.ProbabilityLow), string(models.StatusOpen), testUserID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1)) // Supondo que ID é o primeiro campo, e 1 linha afetada
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

        // A resposta deve ser StatusOK porque a linha foi processada (usando categoria default)
        // mas o failedRows deve conter o aviso sobre a categoria.
		assert.Equal(t, http.StatusMultiStatus, rr.Code, "Response: %s", rr.Body.String())
		var resp BulkUploadRisksResponse
		err = json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, 1, resp.SuccessfullyImported)
        assert.Len(t, resp.FailedRows, 1) // A "falha" aqui é o aviso da categoria
        assert.Equal(t, 2, resp.FailedRows[0].LineNumber)
        assert.Contains(t, resp.FailedRows[0].Errors[0], "invalid category: 'invalid_category'")
		assert.NoError(t, sqlMock.ExpectationsWereMet())
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


// --- Approval Workflow Handler Tests ---

var testRiskForApprovalID = uuid.New()
var testRiskOwnerID = uuid.New() // Deve ser diferente de testUserID se quisermos testar permissões
var testManagerUserID = uuid.New() // Um usuário com role Manager
var testApprovalWorkflowID = uuid.New()


func TestSubmitRiskForAcceptanceHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Roteador com contexto de usuário Manager (testManagerUserID, testOrgID)
	router := getRouterWithOrgAdminContext(testManagerUserID, testOrgID, models.RoleManager)
	router.POST("/risks/:riskId/submit-acceptance", SubmitRiskForAcceptanceHandler)

	// Setup: Criar um usuário que será o "owner" do risco
	// No mundo real, este usuário já existiria. Para o teste, podemos mockar sua existência.
	// A notificação simulada no handler tentará buscar este usuário.
	ownerUserRows := sqlmock.NewRows([]string{"id", "email"}).AddRow(testRiskOwnerID, "owner@example.com")
	managerUserRows := sqlmock.NewRows([]string{"id", "email"}).AddRow(testManagerUserID, "manager@example.com")


	t.Run("Successful submission for acceptance", func(t *testing.T) {
		mockRisk := models.Risk{
			ID:             testRiskForApprovalID,
			OrganizationID: testOrgID,
			Title:          "Risk to be Approved",
			OwnerID:        testRiskOwnerID, // Owner que receberá a aprovação
			Status:         models.StatusOpen,
		}

		// Mock para buscar o risco
		riskRows := sqlmock.NewRows([]string{"id", "organization_id", "title", "owner_id", "status"}).
			AddRow(mockRisk.ID, mockRisk.OrganizationID, mockRisk.Title, mockRisk.OwnerID, mockRisk.Status)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE id = $1 AND organization_id = $2`)).
			WithArgs(testRiskForApprovalID, testOrgID).
			WillReturnRows(riskRows)

		// Mock para verificar workflow pendente existente (espera-se não encontrar)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "approval_workflows" WHERE risk_id = $1 AND status = $2`)).
			WithArgs(testRiskForApprovalID, models.ApprovalPending).
			WillReturnError(gorm.ErrRecordNotFound)

		// Mock para criar o ApprovalWorkflow
		sqlMock.ExpectBegin()
		sqlMock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "approval_workflows" ("id","risk_id","requester_id","approver_id","status","comments","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), testRiskForApprovalID, testManagerUserID, testRiskOwnerID, models.ApprovalPending, "", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(testApprovalWorkflowID))
		sqlMock.ExpectCommit()

		// Mocks para buscar emails para a notificação placeholder
		// Para SubmitRiskForAcceptanceHandler, o owner do risco é o approverId, e o manager é o requesterId
		// A notificação simulada busca ambos.
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

		// Mock para workflow pendente existente
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
		// Usuário com role 'user' tentando submeter
		userRouter := getRouterWithOrgAdminContext(testUserID, testOrgID, models.RoleUser)
		userRouter.POST("/risks/:riskId/submit-acceptance", SubmitRiskForAcceptanceHandler)

		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/risks/%s/submit-acceptance", testRiskForApprovalID.String()), nil)
		rr := httptest.NewRecorder()
		userRouter.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusForbidden, rr.Code)
		assert.Contains(t, rr.Body.String(), "Only admins or managers can submit")
		// No DB interaction expected
	})
}


func TestApproveOrRejectRiskAcceptanceHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Roteador com contexto do APROVADOR (testRiskOwnerID)
	router := getRouterWithOrgAdminContext(testRiskOwnerID, testOrgID, models.RoleUser) // Role do aprovador pode ser User
	router.POST("/risks/:riskId/approval/:approvalId/decide", ApproveOrRejectRiskAcceptanceHandler)

	t.Run("Successful approval", func(t *testing.T) {
		payload := DecisionPayload{Decision: models.ApprovalApproved, Comments: "Looks good to me."}
		body, _ := json.Marshal(payload)

		mockAWF := models.ApprovalWorkflow{
			ID:          testApprovalWorkflowID,
			RiskID:      testRiskForApprovalID,
			ApproverID:  testRiskOwnerID, // O usuário logado é o aprovador
			Status:      models.ApprovalPending,
			Risk:        models.Risk{OrganizationID: testOrgID}, // Para a verificação de organização no Join
		}
		awfRows := sqlmock.NewRows([]string{"id", "risk_id", "approver_id", "status", "Risk__organization_id"}).
			AddRow(mockAWF.ID, mockAWF.RiskID, mockAWF.ApproverID, mockAWF.Status, mockAWF.Risk.OrganizationID)

		// Mock para buscar o ApprovalWorkflow
		// A query exata com Joins pode ser complexa para mockar com regexp.QuoteMeta se a ordem das colunas não for garantida.
		// Simplificando a query esperada para o essencial.
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT "approval_workflows"."id","approval_workflows"."risk_id","approval_workflows"."requester_id","approval_workflows"."approver_id","approval_workflows"."status","approval_workflows"."comments","approval_workflows"."created_at","approval_workflows"."updated_at","Risk"."id" AS "Risk__id","Risk"."organization_id" AS "Risk__organization_id","Risk"."title" AS "Risk__title","Risk"."description" AS "Risk__description","Risk"."category" AS "Risk__category","Risk"."impact" AS "Risk__impact","Risk"."probability" AS "Risk__probability","Risk"."status" AS "Risk__status","Risk"."owner_id" AS "Risk__owner_id","Risk"."created_at" AS "Risk__created_at","Risk"."updated_at" AS "Risk__updated_at" FROM "approval_workflows" LEFT JOIN "risks" "Risk" ON "approval_workflows"."risk_id" = "Risk"."id" WHERE "approval_workflows"."id" = $1 AND "approval_workflows"."risk_id" = $2 AND "Risk"."organization_id" = $3`)).
			WithArgs(testApprovalWorkflowID, testRiskForApprovalID, testOrgID).
			WillReturnRows(awfRows)

		sqlMock.ExpectBegin()
		// Mock para salvar ApprovalWorkflow atualizado
		sqlMock.ExpectExec(regexp.QuoteMeta(`UPDATE "approval_workflows" SET "risk_id"=$1,"requester_id"=$2,"approver_id"=$3,"status"=$4,"comments"=$5,"updated_at"=$6 WHERE "id" = $7`)).
			WithArgs(mockAWF.RiskID, sqlmock.AnyArg(), mockAWF.ApproverID, payload.Decision, payload.Comments, sqlmock.AnyArg(), mockAWF.ID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Mock para buscar Risco para atualizar status
		riskToUpdateRows := sqlmock.NewRows([]string{"id", "status"}).AddRow(testRiskForApprovalID, models.StatusOpen)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE id = $1`)).
			WithArgs(testRiskForApprovalID).
			WillReturnRows(riskToUpdateRows)
		// Mock para salvar Risco atualizado
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
			RequesterID: testManagerUserID, // Assumindo que o manager submeteu
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
		// Não há atualização no Risco em caso de rejeição
		sqlMock.ExpectCommit()

		// Mock para buscar o Risco para o título na notificação de email
		rejectedRiskRows := sqlmock.NewRows([]string{"id", "title"}).AddRow(mockAWF.RiskID, mockAWF.Risk.Title)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE id = $1 ORDER BY "risks"."id" LIMIT $2`)).
			WithArgs(mockAWF.RiskID, 1).
			WillReturnRows(rejectedRiskRows)

		// Mock para buscar o Requisitante para notificação por email
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

	// TODO: Testar caso de usuário não autorizado (não é o approver_id)
	// TODO: Testar caso de workflow não pendente
}

func TestUpdateRiskHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := getRouterWithAuthenticatedContext(testUserID, testOrgID)
	router.PUT("/risks/:riskId", UpdateRiskHandler)

	// testRiskID já definido globalmente para testes
	// testUserID é o admin/manager que faz a ação
	// testRiskOwnerID pode ser o owner do risco a ser atualizado

	t.Run("Successful risk update - status changed", func(t *testing.T) {
		payload := RiskPayload{
			Title:       "Updated Risk Title",
			Description: "Updated Description",
			Category:    models.CategoryOperational,
			Impact:      models.ImpactCritical,
			Probability: models.ProbabilityCrítico, // Corrigido para Crítico
			Status:      models.StatusInProgress, // Novo status
			OwnerID:     testRiskOwnerID.String(),
		}
		body, _ := json.Marshal(payload)

		originalRisk := models.Risk{
			ID:             testRiskID,
			OrganizationID: testOrgID,
			Title:          "Original Risk Title",
			Status:         models.StatusOpen, // Status original
			OwnerID:        testRiskOwnerID,
		}
		riskRows := sqlmock.NewRows([]string{"id", "organization_id", "title", "status", "owner_id"}).
			AddRow(originalRisk.ID, originalRisk.OrganizationID, originalRisk.Title, originalRisk.Status, originalRisk.OwnerID)

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "risks" WHERE id = $1 AND organization_id = $2 ORDER BY "risks"."id" LIMIT $3`)).
			WithArgs(testRiskID, testOrgID, 1).
			WillReturnRows(riskRows)

		sqlMock.ExpectBegin()
		// A ordem dos campos no SET pode variar, ou pode ser mockado de forma mais genérica
		sqlMock.ExpectExec(regexp.QuoteMeta(`UPDATE "risks" SET`)). // Regex mais genérico para UPDATE
			WithArgs(payload.Category, payload.Description, payload.Impact, testOrgID, testRiskOwnerID, payload.Probability, payload.Status, payload.Title, sqlmock.AnyArg(), testRiskID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		sqlMock.ExpectCommit()

		// Mock para o Preload("Owner") na resposta
		ownerRows := sqlmock.NewRows([]string{"id", "name"}).AddRow(testRiskOwnerID, "Risk Owner Name")
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1`)).
			WithArgs(testRiskOwnerID).
			WillReturnRows(ownerRows)

		// Mock para a notificação de email devido à mudança de status
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 ORDER BY "users"."id" LIMIT $2`)).
			WithArgs(testRiskOwnerID, 1).
			WillReturnRows(ownerRows) // Reutiliza ownerRows para simplificar

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/risks/%s", testRiskID.String()), bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "Response: %s", rr.Body.String())
		var updatedRiskResp UserResponse // Assumindo que UpdateRiskHandler retorna o risco, não UserResponse
		// Corrigir para models.Risk ou um RiskResponseDTO se existir
		var updatedRisk models.Risk
		err := json.Unmarshal(rr.Body.Bytes(), &updatedRisk)
		assert.NoError(t, err)
		assert.Equal(t, payload.Title, updatedRisk.Title)
		assert.Equal(t, payload.Status, updatedRisk.Status)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
	// TODO: Testar UpdateRiskHandler sem mudança de status (sem notificação)
	// TODO: Testar UpdateRiskHandler para risco não encontrado
	// TODO: Testar UpdateRiskHandler com payload inválido
}

func TestDeleteRiskHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := getRouterWithAuthenticatedContext(testUserID, testOrgID)
	router.DELETE("/risks/:riskId", DeleteRiskHandler)

	// testRiskID já definido

	t.Run("Successful risk deletion", func(t *testing.T) {
		// Mock para buscar o risco antes de deletar (verificação de existência e org)
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
	// TODO: Testar DeleteRiskHandler para risco não encontrado
}


func TestGetRiskApprovalHistoryHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := getRouterWithAuthenticatedContext(testUserID, testOrgID) // Qualquer usuário da org pode ver
	router.GET("/risks/:riskId/approval-history", GetRiskApprovalHistoryHandler)

	t.Run("Successful get approval history", func(t *testing.T) {
		// Mock para verificar a existência do risco e se pertence à organização
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

		// Mocks para Preload("Requester") e Preload("Approver")
		// Assumindo que os IDs testManagerUserID e testRiskOwnerID são usados
		userRows := sqlmock.NewRows([]string{"id", "name", "email"}).
			AddRow(testManagerUserID, "Manager User", "manager@example.com").
			AddRow(testRiskOwnerID, "Owner User", "owner@example.com")

		// O GORM pode fazer uma query com IN para os IDs únicos de Requester e Approver
		// Ou queries separadas. Vamos mockar para IN, que é mais comum para preloads.
		// A ordem dos IDs no IN pode variar.
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" IN ($1,$2)`)).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()). // Usar AnyArg se a ordem não for garantida ou testar combinações
			WillReturnRows(userRows)


		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/risks/%s/approval-history", testRiskForApprovalID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "Response: %s", rr.Body.String())
		var history []models.ApprovalWorkflow
		err := json.Unmarshal(rr.Body.Bytes(), &history)
		assert.NoError(t, err)
		assert.Len(t, history, 2)
		// Adicionar mais asserções se necessário, ex: verificar dados do requester/approver
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}
