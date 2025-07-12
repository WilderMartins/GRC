package handlers

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"phoenixgrc/backend/internal/auth"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/filestorage"
	"phoenixgrc/backend/internal/models"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// MockFileStorageProvider for testing
type MockFileStorageProvider struct {
	UploadFileFunc func(ctx context.Context, organizationID string, objectName string, fileContent io.Reader) (fileURL string, err error)
}

func (m *MockFileStorageProvider) UploadFile(ctx context.Context, organizationID string, objectName string, fileContent io.Reader) (string, error) {
	if m.UploadFileFunc != nil {
		return m.UploadFileFunc(ctx, organizationID, objectName, fileContent)
	}
	// Ensure the objectName is somewhat dynamic to avoid conflicts if multiple uploads are simulated
	// A more robust mock might use the actual objectName passed.
	// For this test, the key is that it's a predictable, mock-generated URL.
	return "http://mocked.storage.url/" + organizationID + "/" + objectName, nil
}

var mockDB *gorm.DB
var sqlMock sqlmock.Sqlmock

// setupTestEnvironment initializes Gin, mock DB, and other necessary components for testing.
func setupTestEnvironment(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// JWT Init for auth middleware
	// Ensure JWT_SECRET_KEY is set for tests that might go through auth middleware
	// or directly call functions relying on it.
	currentJwtKey := os.Getenv("JWT_SECRET_KEY")
	if currentJwtKey == "" {
		os.Setenv("JWT_SECRET_KEY", "testsecretfortests") // Use a consistent test key
	}
	if err := auth.InitializeJWT(); err != nil {
		t.Fatalf("Failed to initialize JWT for tests: %v", err)
	}
	if currentJwtKey == "" { // Clean up if we set it
		defer os.Unsetenv("JWT_SECRET_KEY")
	}


	var err error
	db, smock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("An error '%s' was not expected when opening a stub database connection", err)
	}
	sqlMock = smock // Assign to global for use in tests

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Silent, // Change to logger.Info for SQL logs during debugging
			Colorful:      true,
		},
	)

	mockDB, err = gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{Logger: gormLogger})
	if err != nil {
		t.Fatalf("An error '%s' was not expected when opening gorm database: %v", err)
	}
	database.SetDB(mockDB) // Use the global DB setter from database package
}

// TestMain can be used if specific setup/teardown is needed for the whole package,
// but individual test setup (like setupTestEnvironment) is often more flexible.
// func TestMain(m *testing.M) {
// 	// Global setup
// 	exitVal := m.Run()
// 	// Global teardown
// 	os.Exit(exitVal)
// }


func TestCreateOrUpdateAssessmentHandler_UploadFile(t *testing.T) {
	setupTestEnvironment(t)

	mockFileProvider := &MockFileStorageProvider{}
	originalFileStorageProvider := filestorage.DefaultFileStorageProvider
	filestorage.DefaultFileStorageProvider = mockFileProvider
	defer func() { filestorage.DefaultFileStorageProvider = originalFileStorageProvider }()

	router := gin.Default()
	orgID := uuid.New()
	userID := uuid.New() // Define userID, even if not directly used in this specific test logic path of handler
	router.Use(func(c *gin.Context) {
		c.Set("organizationID", orgID)
		c.Set("userID", userID)
		c.Set("userRole", models.RoleAdmin)
		c.Next()
	})
	router.POST("/assessments", CreateOrUpdateAssessmentHandler)

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	controlID := uuid.New()
	assessmentData := AssessmentPayload{
		AuditControlID: controlID.String(),
		Status:         models.ControlStatusConformant,
		Score:          pointyInt(100),
		AssessmentDate: "2023-01-15",
	}
	jsonData, _ := json.Marshal(assessmentData)
	_ = writer.WriteField("data", string(jsonData))

	fileContents := "dummy file content for testing"
	part, _ := writer.CreateFormFile("evidence_file", "test_evidence.txt")
	_, _ = io.Copy(part, strings.NewReader(fileContents))
	writer.Close()

	sqlMock.ExpectBegin()
	// The RETURNING "id" (or other fields) is common for GORM to get the ID of the inserted/updated row.
	// The exact SQL for ON CONFLICT can vary slightly based on GORM version and specific usage.
	// This regex is made more flexible for column order and potential differences.
	sqlMock.ExpectQuery(regexp.QuoteMeta(
		`INSERT INTO "audit_assessments" ("id","organization_id","audit_control_id","status","evidence_url","score","assessment_date","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT ("organization_id","audit_control_id") DO UPDATE SET "status"=EXCLUDED."status","evidence_url"=EXCLUDED."evidence_url","score"=EXCLUDED."score","assessment_date"=EXCLUDED."assessment_date","updated_at"=EXCLUDED."updated_at" RETURNING "id"`)).
		WithArgs(sqlmock.AnyArg(), orgID, controlID, assessmentData.Status, sqlmock.AnyArg(), *assessmentData.Score, AnyTime{}, AnyTime{}, AnyTime{}).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	sqlMock.ExpectCommit()

	// Mock the re-fetch call after upsert
	// The actual uploaded URL will have a UUID prepended to the filename
	// We need to match this pattern rather than a fixed string.
	// The mock UploadFile returns "http://mocked.storage.url/" + orgID.String() + "/" + objectPath
	// where objectPath is orgID/audit_evidences/controlID/uuid_filename

	// This is the path constructed in the handler:
	// objectPath := fmt.Sprintf("%s/audit_evidences/%s/%s", organizationID.String(), auditControlUUID.String(), newFileName)
	// We can't know newFileName (uuid.New()) exactly, so we match part of the URL.

	sqlMock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "audit_assessments" WHERE organization_id = $1 AND audit_control_id = $2 ORDER BY "audit_assessments"."id" LIMIT 1`)).
		WithArgs(orgID, controlID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "audit_control_id", "status", "evidence_url", "score", "assessment_date"}).
			AddRow(uuid.New(), orgID, controlID, assessmentData.Status, "http://mocked.storage.url/some_org/some_path/test_evidence.txt", *assessmentData.Score, time.Date(2023,1,15,0,0,0,0,time.UTC)))


	req, _ := http.NewRequest(http.MethodPost, "/assessments", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Response code should be OK")

	var responseAssessment models.AuditAssessment
	err := json.Unmarshal(rr.Body.Bytes(), &responseAssessment)
	assert.NoError(t, err, "Failed to unmarshal response")
	assert.Equal(t, assessmentData.Status, responseAssessment.Status, "Status mismatch")
	assert.Equal(t, *assessmentData.Score, responseAssessment.Score, "Score mismatch")

	// Check that the URL starts with the mock provider's base and includes part of the expected path structure
	assert.True(t, strings.HasPrefix(responseAssessment.EvidenceURL, "http://mocked.storage.url/"+orgID.String()), "Evidence URL prefix mismatch")
	assert.True(t, strings.Contains(responseAssessment.EvidenceURL, "/audit_evidences/"+controlID.String()+"/"), "Evidence URL path structure mismatch")
	assert.True(t, strings.HasSuffix(responseAssessment.EvidenceURL, "test_evidence.txt"), "Evidence URL suffix mismatch")


	if err := sqlMock.ExpectationsWereMet(); err != nil {
		t.Errorf("SQL mock expectations not met: %s", err)
	}
}


func TestCreateOrUpdateAssessmentHandler_NoFile(t *testing.T) {
	setupTestEnvironment(t)

	mockFileProvider := &MockFileStorageProvider{}
	originalFileStorageProvider := filestorage.DefaultFileStorageProvider
	filestorage.DefaultFileStorageProvider = mockFileProvider
	defer func() { filestorage.DefaultFileStorageProvider = originalFileStorageProvider }()

	router := gin.Default()
	orgID := uuid.New()
	userID := uuid.New()
	router.Use(func(c *gin.Context) {
		c.Set("organizationID", orgID)
		c.Set("userID", userID)
		c.Set("userRole", models.RoleAdmin)
		c.Next()
	})
	router.POST("/assessments", CreateOrUpdateAssessmentHandler)

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	controlID := uuid.New()
	assessmentDataPayload := AssessmentPayload{
		AuditControlID: controlID.String(),
		Status:         models.ControlStatusNonConformant,
		EvidenceURL:    "http://manual.evidence.url/evidence.pdf",
		Score:          pointyInt(10),
		AssessmentDate: "2023-02-20",
	}
	jsonData, _ := json.Marshal(assessmentDataPayload)
	_ = writer.WriteField("data", string(jsonData))
	writer.Close()

	sqlMock.ExpectBegin()
	sqlMock.ExpectQuery(regexp.QuoteMeta(
		`INSERT INTO "audit_assessments" ("id","organization_id","audit_control_id","status","evidence_url","score","assessment_date","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT ("organization_id","audit_control_id") DO UPDATE SET "status"=EXCLUDED."status","evidence_url"=EXCLUDED."evidence_url","score"=EXCLUDED."score","assessment_date"=EXCLUDED."assessment_date","updated_at"=EXCLUDED."updated_at" RETURNING "id"`)).
		WithArgs(sqlmock.AnyArg(), orgID, controlID, assessmentDataPayload.Status, assessmentDataPayload.EvidenceURL, *assessmentDataPayload.Score, AnyTime{}, AnyTime{}, AnyTime{}).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	sqlMock.ExpectCommit()

	sqlMock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "audit_assessments" WHERE organization_id = $1 AND audit_control_id = $2 ORDER BY "audit_assessments"."id" LIMIT 1`)).
		WithArgs(orgID, controlID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "audit_control_id", "status", "evidence_url", "score", "assessment_date"}).
			AddRow(uuid.New(), orgID, controlID, assessmentDataPayload.Status, assessmentDataPayload.EvidenceURL, *assessmentDataPayload.Score, time.Date(2023,2,20,0,0,0,0,time.UTC)))


	req, _ := http.NewRequest(http.MethodPost, "/assessments", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var responseAssessment models.AuditAssessment
	err := json.Unmarshal(rr.Body.Bytes(), &responseAssessment)
	assert.NoError(t, err)
	assert.Equal(t, assessmentDataPayload.Status, responseAssessment.Status)
	assert.Equal(t, assessmentDataPayload.EvidenceURL, responseAssessment.EvidenceURL)

	if err := sqlMock.ExpectationsWereMet(); err != nil {
		t.Errorf("SQL mock expectations not met: %s", err)
	}
}

func pointyInt(i int) *int {
	return &i
}

type AnyTime struct{}

func (a AnyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}


func TestListFrameworksHandler(t *testing.T) {
	setupTestEnvironment(t)

	router := gin.Default()
	// Assuming ListFrameworksHandler does not require auth for this example
	// If it does, add the mock auth middleware similar to other tests
	router.GET("/frameworks", ListFrameworksHandler)

	expectedFrameworks := []models.AuditFramework{
		{ID: uuid.New(), Name: "NIST CSF 2.0"},
		{ID: uuid.New(), Name: "ISO 27001:2022"},
	}

	rows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"})
	for _, f := range expectedFrameworks {
		// Ensure correct number of values for AddRow based on your actual model & query
		rows.AddRow(f.ID, f.Name, time.Now(), time.Now())
	}
	// Adjust query if your ListFrameworksHandler has specific WHERE clauses or ORDER BY
	sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_frameworks"`)).WillReturnRows(rows)

	req, _ := http.NewRequest(http.MethodGet, "/frameworks", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var actualFrameworks []models.AuditFramework
	err := json.Unmarshal(rr.Body.Bytes(), &actualFrameworks)
	assert.NoError(t, err)
	assert.Len(t, actualFrameworks, len(expectedFrameworks))
	if len(actualFrameworks) > 0 && len(expectedFrameworks) > 0 {
		assert.Equal(t, expectedFrameworks[0].Name, actualFrameworks[0].Name)
	}


	if err := sqlMock.ExpectationsWereMet(); err != nil {
		t.Errorf("SQL mock expectations not met: %s", err)
	}
}
// TODO: Add more tests for CreateOrUpdateAssessmentHandler:
// - File too large
// - Invalid MIME type
// - File upload error from provider
// - DB error during upsert
// - Invalid payload (missing fields, bad UUIDs, etc.)
// - Test case for when the assessment already exists (update path of upsert)

// Placeholder for other audit handler tests
// TestGetFrameworkControlsHandler, TestGetAssessmentForControlHandler
// would follow similar patterns.

func TestListOrgAssessmentsByFrameworkHandler_Auth(t *testing.T) {
	setupTestEnvironment(t)
	gin.SetMode(gin.TestMode)

	targetOrgID := uuid.New()
	otherOrgID := uuid.New()
	frameworkID := uuid.New()
	actingUserID := uuid.New() // User performing the action

	testCases := []struct {
		name           string
		requestOrgID   uuid.UUID // Org ID in the URL path
		tokenOrgID     uuid.UUID // Org ID in the user's token
		userRole       models.UserRole
		expectedStatus int
		mockDB         func()
	}{
		{
			name:           "User accesses own org's assessments - Allowed",
			requestOrgID:   targetOrgID,
			tokenOrgID:     targetOrgID, // Token org matches path org
			userRole:       models.RoleUser, // Any role in the org can list
			expectedStatus: http.StatusOK,
			mockDB: func() {
				// Mock DB calls for a successful listing
				sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT "id" FROM "audit_controls" WHERE framework_id = $1`)).
					WithArgs(frameworkID).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New())) // At least one control
				sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "audit_assessments" WHERE organization_id = $1 AND audit_control_id IN ($2)`)).
					WithArgs(targetOrgID, sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0)) // No assessments for simplicity
				sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_assessments" WHERE organization_id = $1 AND audit_control_id IN ($2) ORDER BY assessment_date desc LIMIT $3 OFFSET $4`)).
					WithArgs(targetOrgID, sqlmock.AnyArg(), 10, 0). // Default page size 10, offset 0
					WillReturnRows(sqlmock.NewRows(nil)) // Empty result set
			},
		},
		{
			name:           "User (even admin) accesses other org's assessments - Forbidden",
			requestOrgID:   otherOrgID,  // Path org is different
			tokenOrgID:     targetOrgID, // Token org
			userRole:       models.RoleAdmin, // Even as admin of own org
			expectedStatus: http.StatusForbidden,
			mockDB:         func() { /* No DB calls expected */ },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := getRouterWithAuthContext(actingUserID, tc.tokenOrgID, tc.userRole)
			router.GET("/organizations/:orgId/frameworks/:frameworkId/assessments", ListOrgAssessmentsByFrameworkHandler)

			if tc.mockDB != nil {
				tc.mockDB()
			}

			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/organizations/%s/frameworks/%s/assessments", tc.requestOrgID, frameworkID), nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code, "Response code mismatch. Body: %s", rr.Body.String())
			if err := sqlMock.ExpectationsWereMet(); err != nil {
				t.Errorf("SQL mock expectations not met for %s: %s", tc.name, err)
			}
		})
	}
}

func TestGetComplianceScoreHandler_Auth(t *testing.T) {
	setupTestEnvironment(t)
	gin.SetMode(gin.TestMode)

	targetOrgID := uuid.New()
	otherOrgID := uuid.New()
	frameworkID := uuid.New()
	actingUserID := uuid.New()

	testCases := []struct {
		name           string
		requestOrgID   uuid.UUID
		tokenOrgID     uuid.UUID
		userRole       models.UserRole
		expectedStatus int
		mockDB         func()
	}{
		{
			name:           "User accesses own org's compliance score - Allowed",
			requestOrgID:   targetOrgID,
			tokenOrgID:     targetOrgID,
			userRole:       models.RoleUser, // Any role
			expectedStatus: http.StatusOK,
			mockDB: func() {
				// Mock DB calls for a successful score calculation
				sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_frameworks" WHERE id = $1 ORDER BY "audit_frameworks"."id" LIMIT 1`)).
					WithArgs(frameworkID).
					WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(frameworkID, "Test Framework"))
				sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_controls" WHERE framework_id = $1`)).
					WithArgs(frameworkID).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New())) // At least one control
				sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_assessments" WHERE organization_id = $1 AND audit_control_id IN ($2)`)).
					WithArgs(targetOrgID, sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows(nil)) // No assessments for simplicity, leads to 0 score
			},
		},
		{
			name:           "User (even admin) accesses other org's compliance score - Forbidden",
			requestOrgID:   otherOrgID,
			tokenOrgID:     targetOrgID,
			userRole:       models.RoleAdmin,
			expectedStatus: http.StatusForbidden,
			mockDB:         func() { /* No DB calls expected */ },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := getRouterWithAuthContext(actingUserID, tc.tokenOrgID, tc.userRole)
			router.GET("/organizations/:orgId/frameworks/:frameworkId/compliance-score", GetComplianceScoreHandler)

			if tc.mockDB != nil {
				tc.mockDB()
			}

			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/organizations/%s/frameworks/%s/compliance-score", tc.requestOrgID, frameworkID), nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code, "Response code mismatch. Body: %s", rr.Body.String())
			if err := sqlMock.ExpectationsWereMet(); err != nil {
				t.Errorf("SQL mock expectations not met for %s: %s", tc.name, err)
			}
		})
	}
}

func TestCreateOrUpdateAssessmentHandler_FileTooLarge(t *testing.T) {
	setupTestEnvironment(t)
	// Usar getRouterWithAuthContext de main_test_handler.go se padronizado,
	// ou garantir que o contexto seja configurado com userRole.
	router := getRouterWithAuthContext(testUserID, testOrgID, models.RoleAdmin)
	router.POST("/assessments", CreateOrUpdateAssessmentHandler)

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	controlID := uuid.New()
	assessmentData := AssessmentPayload{AuditControlID: controlID.String(), Status: models.ControlStatusConformant}
	jsonData, _ := json.Marshal(assessmentData)
	_ = writer.WriteField("data", string(jsonData))

	// Criar um arquivo mock grande (excedendo maxEvidenceFileSize)
	largeFileContent := make([]byte, maxEvidenceFileSize+1) // maxEvidenceFileSize é 10MB
	part, _ := writer.CreateFormFile("evidence_file", "large_evidence.txt")
	_, _ = io.Copy(part, bytes.NewReader(largeFileContent))
	writer.Close()

	req, _ := http.NewRequest(http.MethodPost, "/assessments", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "File size exceeds limit")
	// Nenhuma expectativa de DB, pois deve falhar antes
	assert.NoError(t, sqlMock.ExpectationsWereMet()) // Verificar se não houve interações inesperadas
}

func TestCreateOrUpdateAssessmentHandler_InvalidMimeType(t *testing.T) {
	setupTestEnvironment(t)
	router := getRouterWithAuthContext(testUserID, testOrgID, models.RoleAdmin)
	router.POST("/assessments", CreateOrUpdateAssessmentHandler)

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	controlID := uuid.New()
	assessmentData := AssessmentPayload{AuditControlID: controlID.String(), Status: models.ControlStatusConformant}
	jsonData, _ := json.Marshal(assessmentData)
	_ = writer.WriteField("data", string(jsonData))

	invalidFileContent := []byte{0xDE, 0xAD, 0xBE, 0xEF} // Exemplo de bytes binários
	part, _ := writer.CreateFormFile("evidence_file", "invalid_type.exe")
	_, _ = io.Copy(part, bytes.NewReader(invalidFileContent))
	writer.Close()

	req, _ := http.NewRequest(http.MethodPost, "/assessments", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "File type")
	assert.Contains(t, rr.Body.String(), "is not allowed")
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}


func TestCreateOrUpdateAssessmentHandler_WithC2M2Fields(t *testing.T) {
	setupTestEnvironment(t)

	mockFileProvider := &MockFileStorageProvider{} // Não vamos fazer upload de arquivo neste teste
	originalFileStorageProvider := filestorage.DefaultFileStorageProvider
	filestorage.DefaultFileStorageProvider = mockFileProvider
	defer func() { filestorage.DefaultFileStorageProvider = originalFileStorageProvider }()

	router := getRouterWithAuthContext(testUserID, testOrgID, models.RoleAdmin)
	router.POST("/assessments", CreateOrUpdateAssessmentHandler)

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	controlID := uuid.New()
	c2m2Level := 2
	c2m2DateStr := "2024-03-10"
	c2m2Comments := "C2M2 assessment comments here."

	assessmentDataPayload := AssessmentPayload{
		AuditControlID:    controlID.String(),
		Status:            models.ControlStatusPartiallyConformant,
		Score:             pointyInt(60),
		AssessmentDate:    "2024-03-15",
		Comments:          pointyStr("Main assessment comments."),
		// C2M2MaturityLevel é calculado, então não enviamos mais
		C2M2AssessmentDate: &c2m2DateStr,
		C2M2Comments:      &c2m2Comments,
		C2M2PracticeEvaluations: map[string]string{ // Enviar avaliações de práticas
			practice1_mil1.ID.String(): models.PracticeStatusFullyImplemented,
			practice2_mil1.ID.String(): models.PracticeStatusFullyImplemented,
			practice1_mil2.ID.String(): models.PracticeStatusPartiallyImplemented,
		},
	}
	jsonData, _ := json.Marshal(assessmentDataPayload)
	_ = writer.WriteField("data", string(jsonData))
	writer.Close()

	// --- Mock DB ---

	// 1. Upsert do Assessment principal
	sqlMock.ExpectBegin()
	// O upsert inicial não incluirá C2M2MaturityLevel se não estiver no payload, ou será nulo.
	// A lógica do handler agora o calcula depois.
	sqlMock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "audit_assessments"`)).
		WithArgs(sqlmock.AnyArg(), testOrgID, controlID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	sqlMock.ExpectCommit()

	// 2. Re-fetch do Assessment para obter o ID
	fetchedAssessmentID := uuid.New()
	sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_assessments" WHERE organization_id = $1 AND audit_control_id = $2`)).
		WithArgs(testOrgID, controlID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(fetchedAssessmentID))

	// 3. Upsert das PracticeEvaluations
	sqlMock.ExpectBegin()
	// sqlmock não lida bem com múltiplos inserts em uma única query (Create(&evalsToUpsert)).
	// GORM pode fazer isso com uma query longa ou em um loop. Vamos mockar como um loop para ser seguro.
	// Ou, se GORM usar um INSERT ... VALUES (...), (...), (...), podemos mockar isso.
	// Vamos assumir uma query de upsert em lote para ser mais otimista.
	sqlMock.ExpectExec(regexp.QuoteMeta(`INSERT INTO "c2m2_practice_evaluations"`)).
		WillReturnResult(sqlmock.NewResult(3, 3)) // 3 linhas afetadas
	sqlMock.ExpectCommit()

	// 4. Lógica de Cálculo de MIL (chamada dentro do handler)
	// 4a. Buscar todas as práticas C2M2
	practiceRows := sqlmock.NewRows([]string{"id", "code", "target_mil"}).
		AddRow(practice1_mil1.ID, "RM.1.1", 1).
		AddRow(practice2_mil1.ID, "SA.1.1", 1).
		AddRow(practice1_mil2.ID, "RM.2.1", 2)
	sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "c2m2_practices"`)).WillReturnRows(practiceRows)
	// 4b. Buscar as avaliações de práticas para este assessment
	evalRows := sqlmock.NewRows([]string{"practice_id", "status"}).
		AddRow(practice1_mil1.ID, models.PracticeStatusFullyImplemented).
		AddRow(practice2_mil1.ID, models.PracticeStatusFullyImplemented).
		AddRow(practice1_mil2.ID, models.PracticeStatusPartiallyImplemented)
	sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "c2m2_practice_evaluations" WHERE audit_assessment_id = $1`)).
		WithArgs(fetchedAssessmentID).WillReturnRows(evalRows)
	// 4c. Atualizar o C2M2MaturityLevel no assessment
	expectedCalculatedMIL := 1
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec(regexp.QuoteMeta(`UPDATE "audit_assessments" SET "c2m2_maturity_level"=$1,"updated_at"=$2 WHERE id = $3`)).
		WithArgs(expectedCalculatedMIL, sqlmock.AnyArg(), fetchedAssessmentID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	sqlMock.ExpectCommit()

	// 5. Re-fetch final para a resposta
	finalRows := sqlmock.NewRows([]string{"id", "c2m2_maturity_level"}).AddRow(fetchedAssessmentID, expectedCalculatedMIL)
	sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_assessments" WHERE "audit_assessments"."id" = $1`)).
		WithArgs(fetchedAssessmentID).WillReturnRows(finalRows)


	req, _ := http.NewRequest(http.MethodPost, "/assessments", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Response code should be OK. Body: %s", rr.Body.String())

	var responseAssessment models.AuditAssessment
	err := json.Unmarshal(rr.Body.Bytes(), &responseAssessment)
	assert.NoError(t, err, "Failed to unmarshal response")
	assert.Equal(t, assessmentDataPayload.Status, responseAssessment.Status)
	assert.NotNil(t, responseAssessment.Score)
	assert.Equal(t, *assessmentDataPayload.Score, *responseAssessment.Score)
	assert.NotNil(t, responseAssessment.Comments)
	assert.Equal(t, *assessmentDataPayload.Comments, *responseAssessment.Comments)

	assert.NotNil(t, responseAssessment.C2M2MaturityLevel)
	assert.Equal(t, *assessmentDataPayload.C2M2MaturityLevel, *responseAssessment.C2M2MaturityLevel)
	assert.NotNil(t, responseAssessment.C2M2AssessmentDate)
	assert.True(t, parsedC2M2Date.Equal(*responseAssessment.C2M2AssessmentDate), "C2M2AssessmentDate mismatch")
	assert.NotNil(t, responseAssessment.C2M2Comments)
	assert.Equal(t, *assessmentDataPayload.C2M2Comments, *responseAssessment.C2M2Comments)


	if errDbMock := sqlMock.ExpectationsWereMet(); errDbMock != nil {
		t.Errorf("SQL mock expectations not met: %s", errDbMock)
	}
}

func TestGetC2M2MaturitySummaryHandler_Success(t *testing.T) {
	setupTestEnvironment(t)
	orgID := testOrgID // Usar testOrgID globalmente definido nos testes
	frameworkID := uuid.New()
	frameworkName := "Test NIST CSF for C2M2"

	router := getRouterWithAuthContext(testUserID, orgID, models.RoleUser)
	router.GET("/audit/organizations/:orgId/frameworks/:frameworkId/c2m2-maturity-summary", GetC2M2MaturitySummaryHandler)

	// Mock Framework
	sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_frameworks" WHERE id = $1 ORDER BY "audit_frameworks"."id" LIMIT 1`)).
		WithArgs(frameworkID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(frameworkID, frameworkName))

	// Mock Controls
	ctrl1ID, ctrl2ID, ctrl3ID, ctrl4ID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	ctrlRows := sqlmock.NewRows([]string{"id", "framework_id", "control_id", "family"}).
		AddRow(ctrl1ID, frameworkID, "ID.AM-1", "Identify (ID.AM)"). // Função Identify
		AddRow(ctrl2ID, frameworkID, "PR.IP-1", "Protect (PR.IP)"). // Função Protect
		AddRow(ctrl3ID, frameworkID, "ID.RA-1", "Identify (ID.RA)"). // Função Identify
		AddRow(ctrl4ID, frameworkID, "DE.CM-1", "Detect (DE.CM)")   // Função Detect (sem avaliação C2M2)
	sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_controls" WHERE framework_id = $1 ORDER BY control_id asc`)).
		WithArgs(frameworkID).
		WillReturnRows(ctrlRows)

	// Mock Assessments (apenas para controles com C2M2MaturityLevel)
	mil1, mil2, mil3 := 1, 2, 3
	assessRows := sqlmock.NewRows([]string{"audit_control_id", "c2m2_maturity_level"}).
		AddRow(ctrl1ID, &mil2). // ID.AM-1 -> MIL 2
		AddRow(ctrl2ID, &mil3). // PR.IP-1 -> MIL 3
		AddRow(ctrl3ID, &mil1)  // ID.RA-1 -> MIL 1
		// ctrl4ID (Detect) não tem assessment C2M2
	sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_assessments" WHERE organization_id = $1 AND audit_control_id IN ($2,$3,$4,$5) AND c2m2_maturity_level IS NOT NULL`)).
		WithArgs(orgID, ctrl1ID, ctrl2ID, ctrl3ID, ctrl4ID). // A ordem dos IDs pode variar
		WillReturnRows(assessRows)

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/audit/organizations/%s/frameworks/%s/c2m2-maturity-summary", orgID, frameworkID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Response code. Body: %s", rr.Body.String())

	var response C2M2MaturityFrameworkSummaryResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err, "Failed to unmarshal response")

	assert.Equal(t, frameworkID, response.FrameworkID)
	assert.Equal(t, frameworkName, response.FrameworkName)
	assert.Equal(t, orgID, response.OrganizationID)
	assert.Len(t, response.SummaryByFunction, 3, "Deveria haver sumários para 3 funções com avaliações C2M2 (Identify, Protect, Detect - embora Detect não tenha avaliações, deve aparecer com 0)")

	foundIdentify := false
	foundProtect := false
	foundDetect := false

	for _, summary := range response.SummaryByFunction {
		if summary.NISTComponentName == "Identify" {
			foundIdentify = true
			assert.Equal(t, "Function", summary.NISTComponentType)
			// Controles: ID.AM-1 (MIL2), ID.RA-1 (MIL1). Total 2. Avaliados 2.
			// Moda: MIL2 (ocorre 1 vez), MIL1 (ocorre 1 vez). Desempate pelo maior: MIL2
			assert.Equal(t, 2, summary.AchievedMIL)
			assert.Equal(t, 2, summary.EvaluatedControls)
			assert.Equal(t, 2, summary.TotalControls) // ctrl1, ctrl3
			assert.Equal(t, 0, summary.MILDistribution.MIL0)
			assert.Equal(t, 1, summary.MILDistribution.MIL1)
			assert.Equal(t, 1, summary.MILDistribution.MIL2)
			assert.Equal(t, 0, summary.MILDistribution.MIL3)
		} else if summary.NISTComponentName == "Protect" {
			foundProtect = true
			assert.Equal(t, "Function", summary.NISTComponentType)
			// Controles: PR.IP-1 (MIL3). Total 1. Avaliados 1.
			// Moda: MIL3
			assert.Equal(t, 3, summary.AchievedMIL)
			assert.Equal(t, 1, summary.EvaluatedControls)
			assert.Equal(t, 1, summary.TotalControls) // ctrl2
			assert.Equal(t, 0, summary.MILDistribution.MIL0)
			assert.Equal(t, 0, summary.MILDistribution.MIL1)
			assert.Equal(t, 0, summary.MILDistribution.MIL2)
			assert.Equal(t, 1, summary.MILDistribution.MIL3)
		} else if summary.NISTComponentName == "Detect" {
			foundDetect = true
			assert.Equal(t, "Function", summary.NISTComponentType)
			// Controles: DE.CM-1 (sem avaliação C2M2). Total 1. Avaliados 0.
			// Moda: default MIL0
			assert.Equal(t, 0, summary.AchievedMIL)
			assert.Equal(t, 0, summary.EvaluatedControls)
			assert.Equal(t, 1, summary.TotalControls) // ctrl4
			assert.Equal(t, 0, summary.MILDistribution.MIL0)
			assert.Equal(t, 0, summary.MILDistribution.MIL1)
			assert.Equal(t, 0, summary.MILDistribution.MIL2)
			assert.Equal(t, 0, summary.MILDistribution.MIL3)
		}
	}
	assert.True(t, foundIdentify, "Sumário para Função 'Identify' não encontrado")
	assert.True(t, foundProtect, "Sumário para Função 'Protect' não encontrado")
	assert.True(t, foundDetect, "Sumário para Função 'Detect' não encontrado")


	if errDbMock := sqlMock.ExpectationsWereMet(); errDbMock != nil {
		t.Errorf("SQL mock expectations not met: %s", errDbMock)
	}
}

func TestGetC2M2MaturitySummaryHandler_FrameworkNotFound(t *testing.T) {
	setupTestEnvironment(t)
	orgID := testOrgID
	frameworkID := uuid.New()

	router := getRouterWithAuthContext(testUserID, orgID, models.RoleUser)
	router.GET("/audit/organizations/:orgId/frameworks/:frameworkId/c2m2-maturity-summary", GetC2M2MaturitySummaryHandler)

	sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_frameworks" WHERE id = $1 ORDER BY "audit_frameworks"."id" LIMIT 1`)).
		WithArgs(frameworkID).
		WillReturnError(gorm.ErrRecordNotFound)

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/audit/organizations/%s/frameworks/%s/c2m2-maturity-summary", orgID, frameworkID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "Framework not found")

	if errDbMock := sqlMock.ExpectationsWereMet(); errDbMock != nil {
		t.Errorf("SQL mock expectations not met: %s", errDbMock)
	}
}


func TestGetFrameworkControlsHandler_Success_WithAssessments(t *testing.T) {
	setupTestEnvironment(t)
	orgID := testOrgID
	frameworkID := uuid.New()

	router := getRouterWithAuthContext(testUserID, orgID, models.RoleUser) // Auth context para pegar orgID
	router.GET("/audit/frameworks/:frameworkId/controls", GetFrameworkControlsHandler)

	// Mock Controls
	ctrl1ID := uuid.New()
	ctrl2ID := uuid.New()
	ctrlRows := sqlmock.NewRows([]string{"id", "framework_id", "control_id", "family", "description"}).
		AddRow(ctrl1ID, frameworkID, "Ctrl1", "Family1", "Desc1").
		AddRow(ctrl2ID, frameworkID, "Ctrl2", "Family2", "Desc2")
	sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_controls" WHERE framework_id = $1 ORDER BY control_id asc`)).
		WithArgs(frameworkID).
		WillReturnRows(ctrlRows)

	// Mock Assessments
	assessScore := 80
	assessDate := time.Now().Truncate(24 * time.Hour)
	assessRows := sqlmock.NewRows([]string{"id", "organization_id", "audit_control_id", "status", "score", "assessment_date"}).
		AddRow(uuid.New(), orgID, ctrl1ID, models.ControlStatusConformant, &assessScore, &assessDate) // Assessment para ctrl1
		// ctrl2 não tem assessment
	sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_assessments" WHERE organization_id = $1 AND audit_control_id IN ($2,$3)`)).
		WithArgs(orgID, ctrl1ID, ctrl2ID). // A ordem dos IDs pode variar
		WillReturnRows(assessRows)


	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/audit/frameworks/%s/controls", frameworkID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Response code. Body: %s", rr.Body.String())

	// Definir a struct localmente para o teste, pois ela é definida no handler
	type AuditControlWithAssessmentResponse struct {
		models.AuditControl
		Assessment *models.AuditAssessment `json:"assessment,omitempty"`
	}
	var response []AuditControlWithAssessmentResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err, "Failed to unmarshal response")
	assert.Len(t, response, 2, "Deveria haver 2 controles")

	// Verificar Controle 1 (com assessment)
	assert.Equal(t, ctrl1ID, response[0].AuditControl.ID)
	assert.NotNil(t, response[0].Assessment, "Assessment para ctrl1 não deveria ser nulo")
	if response[0].Assessment != nil {
		assert.Equal(t, models.ControlStatusConformant, response[0].Assessment.Status)
		assert.Equal(t, &assessScore, response[0].Assessment.Score)
	}

	// Verificar Controle 2 (sem assessment)
	assert.Equal(t, ctrl2ID, response[1].AuditControl.ID)
	assert.Nil(t, response[1].Assessment, "Assessment para ctrl2 deveria ser nulo")


	if errDbMock := sqlMock.ExpectationsWereMet(); errDbMock != nil {
		t.Errorf("SQL mock expectations not met: %s", errDbMock)
	}
}

func TestGetFrameworkControlsHandler_NoFramework(t *testing.T) {
	setupTestEnvironment(t)
	orgID := testOrgID
	frameworkID := uuid.New()

	router := getRouterWithAuthContext(testUserID, orgID, models.RoleUser)
	router.GET("/audit/frameworks/:frameworkId/controls", GetFrameworkControlsHandler)

	sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_controls" WHERE framework_id = $1 ORDER BY control_id asc`)).
		WithArgs(frameworkID).
		WillReturnRows(sqlmock.NewRows(nil)) // Nenhum controle retornado

	sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_frameworks" WHERE id = $1 ORDER BY "audit_frameworks"."id" LIMIT 1`)).
		WithArgs(frameworkID).
		WillReturnError(gorm.ErrRecordNotFound) // Framework não encontrado

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/audit/frameworks/%s/controls", frameworkID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "Framework not found")

	if errDbMock := sqlMock.ExpectationsWereMet(); errDbMock != nil {
		t.Errorf("SQL mock expectations not met: %s", errDbMock)
	}
}
// Ensure newline at end of file
