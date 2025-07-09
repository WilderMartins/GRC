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
// TestGetFrameworkControlsHandler, TestGetAssessmentForControlHandler, TestListOrgAssessmentsByFrameworkHandler
// would follow similar patterns.
// Remember to:
// 1. Call setupTestEnvironment(t)
// 2. Mock auth context if the route is protected.
// 3. Set up sqlMock expectations for any DB calls.
// 4. Create a request (httptest.NewRequest).
// 5. Record the response (httptest.NewRecorder).
// 6. router.ServeHTTP(rr, req).
// 7. Assert status code and response body.
// 8. Assert sqlMock.ExpectationsWereMet().
```
