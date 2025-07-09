package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"phoenixgrc/backend/internal/database" // Usado por setupMockDB indiretamente
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
)

// MockFileStorageProvider para simular o FileStorageProvider
type MockFileStorageProvider struct {
	UploadFileFunc func(ctx context.Context, organizationID string, objectName string, fileContent io.Reader) (string, error)
	DeleteFileFunc func(ctx context.Context, fileURL string) error
}

func (m *MockFileStorageProvider) UploadFile(ctx context.Context, organizationID string, objectName string, fileContent io.Reader) (string, error) {
	if m.UploadFileFunc != nil {
		return m.UploadFileFunc(ctx, organizationID, objectName, fileContent)
	}
	// Retorna uma URL mockada que inclui partes do objectName para verificação
	return "http://mocked.storage.url/" + objectName, nil
}

func (m *MockFileStorageProvider) DeleteFile(ctx context.Context, fileURL string) error {
	if m.DeleteFileFunc != nil {
		return m.DeleteFileFunc(ctx, fileURL)
	}
	return nil
}

// getRouterWithOrgAdminContext cria um router com contexto de usuário que é admin/manager da org especificada.
// Adaptado de getRouterWithAuthenticatedContext em main_test_handler.go
func getRouterWithOrgRoleContext(userID uuid.UUID, orgID uuid.UUID, role models.UserRole) *gin.Engine {
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Set("organizationID", orgID)
		c.Set("userRole", role)
		c.Next()
	})
	return r
}


func TestUpdateOrganizationBrandingHandler(t *testing.T) {
	setupMockDB(t) // Usa o setupMockDB de main_test_handler.go
	gin.SetMode(gin.TestMode)

	// testUserID e testOrgID são definidos globalmente em main_test_handler.go
	router := getRouterWithOrgRoleContext(testUserID, testOrgID, models.RoleAdmin) // Usuário é admin da testOrgID
	router.PUT("/organizations/:orgId/branding", UpdateOrganizationBrandingHandler)

	originalFileStorageProvider := filestorage.DefaultFileStorageProvider
	mockUploader := &MockFileStorageProvider{}
	filestorage.DefaultFileStorageProvider = mockUploader
	defer func() { filestorage.DefaultFileStorageProvider = originalFileStorageProvider }()

	t.Run("Success - Update with logo and colors", func(t *testing.T) {
		expectedLogoURL := ""
		mockUploader.UploadFileFunc = func(ctx context.Context, organizationID string, objectName string, fileContent io.Reader) (string, error) {
			assert.Equal(t, testOrgID.String(), organizationID)
			assert.True(t, strings.HasPrefix(objectName, testOrgID.String()+"/branding/logo_"))
			assert.True(t, strings.HasSuffix(objectName, ".png"))
			expectedLogoURL = "http://mocked.storage.url/" + objectName
			return expectedLogoURL, nil
		}

		brandingPayload := BrandingPayload{PrimaryColor: "#123456", SecondaryColor: "#ABCDEF"}
		payloadJSON, _ := json.Marshal(brandingPayload)

		bodyBuf := &bytes.Buffer{}
		mpWriter := multipart.NewWriter(bodyBuf)
		_ = mpWriter.WriteField("data", string(payloadJSON))
		fileWriter, _ := mpWriter.CreateFormFile("logo_file", "test_logo.png")
		_, _ = fileWriter.Write([]byte("dummy_png_content"))
		mpWriter.Close()

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "organizations" WHERE id = $1 ORDER BY "organizations"."id" LIMIT 1`)).
			WithArgs(testOrgID).WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(testOrgID, "Old Org Name"))

		sqlMock.ExpectBegin()
		sqlMock.ExpectExec(regexp.QuoteMeta(`UPDATE "organizations" SET "logo_url"=$1,"primary_color"=$2,"secondary_color"=$3,"updated_at"=$4 WHERE "id" = $5`)).
			WithArgs(sqlmock.AnyArg(), // logo_url é dinâmico por causa do timestamp
				brandingPayload.PrimaryColor, brandingPayload.SecondaryColor, anyTimeArg(), testOrgID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		sqlMock.ExpectCommit()

		// Re-fetch
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "organizations" WHERE "organizations"."id" = $1 ORDER BY "organizations"."id" LIMIT 1`)).
			WithArgs(testOrgID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "logo_url", "primary_color", "secondary_color"}).
				AddRow(testOrgID, "Old Org Name", expectedLogoURL, brandingPayload.PrimaryColor, brandingPayload.SecondaryColor))

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/organizations/%s/branding", testOrgID), bodyBuf)
		req.Header.Set("Content-Type", mpWriter.FormDataContentType())
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "Response: %s", rr.Body.String())
		var respOrg models.Organization
		err := json.Unmarshal(rr.Body.Bytes(), &respOrg)
		assert.NoError(t, err)
		assert.Equal(t, brandingPayload.PrimaryColor, respOrg.PrimaryColor)
		assert.Equal(t, brandingPayload.SecondaryColor, respOrg.SecondaryColor)
		assert.Equal(t, expectedLogoURL, respOrg.LogoURL)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Success - Update only colors", func(t *testing.T) {
		currentLogo := "http://example.com/current_logo.png"
		brandingPayload := BrandingPayload{PrimaryColor: "#FF0000", SecondaryColor: "#00FF00"}
		payloadJSON, _ := json.Marshal(brandingPayload)

		bodyBuf := &bytes.Buffer{}
		mpWriter := multipart.NewWriter(bodyBuf)
		_ = mpWriter.WriteField("data", string(payloadJSON))
		// No logo_file
		mpWriter.Close()

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "organizations" WHERE id = $1 ORDER BY "organizations"."id" LIMIT 1`)).
			WithArgs(testOrgID).WillReturnRows(sqlmock.NewRows([]string{"id", "name", "logo_url"}).AddRow(testOrgID, "Org Name", currentLogo))

		sqlMock.ExpectBegin()
		// LogoURL não deve estar no SET se nenhum arquivo for enviado e o payload "data" existir
		sqlMock.ExpectExec(regexp.QuoteMeta(`UPDATE "organizations" SET "primary_color"=$1,"secondary_color"=$2,"updated_at"=$3 WHERE "id" = $4`)).
			WithArgs(brandingPayload.PrimaryColor, brandingPayload.SecondaryColor, anyTimeArg(), testOrgID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		sqlMock.ExpectCommit()

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "organizations" WHERE "organizations"."id" = $1 ORDER BY "organizations"."id" LIMIT 1`)).
			WithArgs(testOrgID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "logo_url", "primary_color", "secondary_color"}).
				AddRow(testOrgID, "Org Name", currentLogo, brandingPayload.PrimaryColor, brandingPayload.SecondaryColor))

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/organizations/%s/branding", testOrgID), bodyBuf)
		req.Header.Set("Content-Type", mpWriter.FormDataContentType())
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "Response: %s", rr.Body.String())
		var respOrg models.Organization
		json.Unmarshal(rr.Body.Bytes(), &respOrg)
		assert.Equal(t, brandingPayload.PrimaryColor, respOrg.PrimaryColor)
		assert.Equal(t, currentLogo, respOrg.LogoURL) // Logo should remain unchanged
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Fail - Invalid color format", func(t *testing.T) {
		brandingPayload := BrandingPayload{PrimaryColor: "INVALIDCOLOR", SecondaryColor: "#ABCDEF"}
		payloadJSON, _ := json.Marshal(brandingPayload)
		bodyBuf := &bytes.Buffer{}
		mpWriter := multipart.NewWriter(bodyBuf)
		_ = mpWriter.WriteField("data", string(payloadJSON))
		mpWriter.Close()

		// No DB calls expected as validation should fail first
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/organizations/%s/branding", testOrgID), bodyBuf)
		req.Header.Set("Content-Type", mpWriter.FormDataContentType())
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Formato de Cor Primária inválido")
	})

	// TODO: Add more tests for UpdateOrganizationBrandingHandler:
    // - Success: Only logo, no colors in 'data'
    // - Success: No 'data' field, only logo_file
    // - Fail: Organization not found (DB returns ErrRecordNotFound)
    // - Fail: User not admin/manager of the target organization
    // - Fail: Logo file too large
    // - Fail: Invalid logo MIME type
    // - Fail: FileStorageProvider not configured but logo_file sent
    // - Fail: Error during file upload by FileStorageProvider
}


func TestGetOrganizationBrandingHandler(t *testing.T) {
	setupMockDB(t)
	gin.SetMode(gin.TestMode)

	// User (testUserID) belongs to testOrgID. Role doesn't matter for GET if they belong to org.
	router := getRouterWithOrgRoleContext(testUserID, testOrgID, models.RoleUser)
	router.GET("/organizations/:orgId/branding", GetOrganizationBrandingHandler)

	t.Run("Success - Get branding for own organization", func(t *testing.T) {
		expectedBranding := models.Organization{
			ID:             testOrgID,
			Name:           "Test Org",
			LogoURL:        "http://logo.url/logo.png",
			PrimaryColor:   "#112233",
			SecondaryColor: "#AABBCC",
		}
		rows := sqlMock.NewRows([]string{"id", "name", "logo_url", "primary_color", "secondary_color"}).
			AddRow(expectedBranding.ID, expectedBranding.Name, expectedBranding.LogoURL, expectedBranding.PrimaryColor, expectedBranding.SecondaryColor)

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT "id","name","logo_url","primary_color","secondary_color" FROM "organizations" WHERE id = $1 ORDER BY "organizations"."id" LIMIT 1`)).
			WithArgs(testOrgID).WillReturnRows(rows)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/organizations/%s/branding", testOrgID), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "Response: %s", rr.Body.String())
		var respData map[string]interface{} // Handler returns gin.H
		err := json.Unmarshal(rr.Body.Bytes(), &respData)
		assert.NoError(t, err)
		assert.Equal(t, expectedBranding.ID.String(), respData["id"])
		assert.Equal(t, expectedBranding.LogoURL, respData["logo_url"])
		assert.Equal(t, expectedBranding.PrimaryColor, respData["primary_color"])
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Fail - Organization not found", func(t *testing.T) {
		nonExistentOrgID := uuid.New()

		// Setup router where the user's token org matches the path org, but DB returns not found
		tempRouter := getRouterWithOrgRoleContext(testUserID, nonExistentOrgID, models.RoleUser)
		tempRouter.GET("/organizations/:orgId/branding", GetOrganizationBrandingHandler)

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT "id","name","logo_url","primary_color","secondary_color" FROM "organizations" WHERE id = $1 ORDER BY "organizations"."id" LIMIT 1`)).
			WithArgs(nonExistentOrgID).WillReturnError(gorm.ErrRecordNotFound)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/organizations/%s/branding", nonExistentOrgID), nil)
		rr := httptest.NewRecorder()
		tempRouter.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Fail - User from other org attempts to access", func(t *testing.T) {
		otherOrgID := uuid.New() // User testUserID@testOrgID tries to access otherOrgID's branding

		// No DB call expected as authorization should fail first.
		// The router context has testUserID@testOrgID. The path has otherOrgID.
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/organizations/%s/branding", otherOrgID), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req) // Using original router with testUserID@testOrgID context

		assert.Equal(t, http.StatusForbidden, rr.Code)
		assert.Contains(t, rr.Body.String(), "Você não pode visualizar o branding desta organização")
		// No sqlmock expectations to check here
	})
}
```
