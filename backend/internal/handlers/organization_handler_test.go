package handlers

import (
	// "bytes"
	// "encoding/json"
	// "fmt"
	// "mime/multipart"
	// "net/http"
	// "net/http/httptest"
	// "phoenixgrc/backend/internal/filestorage"
	// "phoenixgrc/backend/internal/models"
	// "regexp"
	"testing"
	// "time"

	// "github.com/DATA-DOG/go-sqlmock"
	// "github.com/gin-gonic/gin"
	// "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"phoenixgrc/backend/internal/filestorage"
	"phoenixgrc/backend/internal/models"
	"regexp"
	"strings"
	// "time" // Removido

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MockFileStorageProvider para simular o FileStorageProvider nos testes de branding
type MockBrandingFileStorageProvider struct {
	UploadFileFunc func(ctx context.Context, organizationID string, objectName string, fileContent io.Reader) (string, error)
}

func (m *MockBrandingFileStorageProvider) UploadFile(ctx context.Context, organizationID string, objectName string, fileContent io.Reader) (string, error) {
	if m.UploadFileFunc != nil {
		return m.UploadFileFunc(ctx, organizationID, objectName, fileContent)
	}
	return "http://mockurl.com/logos/" + objectName, nil // Default mock URL
}


func TestUpdateOrganizationBrandingHandler(t *testing.T) {
	setupMockDB(t) // Adicionado
	gin.SetMode(gin.TestMode)
	// testUserID (admin/manager), testOrgID são definidos em main_test_handler.go
	router := getRouterWithOrgAdminContext(testUserID, testOrgID, models.RoleAdmin) // Usuário é admin da testOrgID
	router.PUT("/orgs/:orgId/branding", UpdateOrganizationBrandingHandler)

	originalFileStorageProvider := filestorage.DefaultFileStorageProvider
	defer func() { filestorage.DefaultFileStorageProvider = originalFileStorageProvider }()

	t.Run("Successful update with logo and colors", func(t *testing.T) {
		mockUploader := &MockBrandingFileStorageProvider{}
		expectedLogoURL := ""
		mockUploader.UploadFileFunc = func(ctx context.Context, organizationID string, objectName string, fileContent io.Reader) (string, error) {
			assert.Equal(t, testOrgID.String(), organizationID)
			assert.True(t, strings.HasPrefix(objectName, testOrgID.String()+"/branding/logo_"))
			assert.True(t, strings.HasSuffix(objectName, ".png"))
			expectedLogoURL = "http://mockgcs.com/logos/" + objectName
			return expectedLogoURL, nil
		}
		filestorage.DefaultFileStorageProvider = mockUploader

		brandingPayload := BrandingPayload{
			PrimaryColor:   "#123456",
			SecondaryColor: "#ABCDEF",
		}
		payloadJSON, _ := json.Marshal(brandingPayload)

		bodyBuf := &bytes.Buffer{}
		mpWriter := multipart.NewWriter(bodyBuf)
		_ = mpWriter.WriteField("data", string(payloadJSON))
		fileWriter, _ := mpWriter.CreateFormFile("logo_file", "test_logo.png")
		_, _ = fileWriter.Write([]byte("dummy png content - must be valid image for real test, but mock doesn't care")) // Conteúdo do arquivo
		mpWriter.Close()

		// Mock para buscar a organização
		orgRows := sqlmock.NewRows([]string{"id", "name", "logo_url", "primary_color", "secondary_color"}).
			AddRow(testOrgID, "Test Org", "", "", "")
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "organizations" WHERE id = $1 ORDER BY "organizations"."id" LIMIT $2`)).
			WithArgs(testOrgID, 1).WillReturnRows(orgRows)

		// Mock para salvar a organização
		sqlMock.ExpectBegin()
		sqlMock.ExpectExec(regexp.QuoteMeta(`UPDATE "organizations" SET "logo_url"=$1,"primary_color"=$2,"secondary_color"=$3,"updated_at"=$4 WHERE "id" = $5`)).
			WithArgs(sqlmock.AnyArg(), brandingPayload.PrimaryColor, brandingPayload.SecondaryColor, sqlmock.AnyArg(), testOrgID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		sqlMock.ExpectCommit()

		// Mock para re-fetch da organização (para a resposta)
		updatedOrgRows := sqlmock.NewRows([]string{"id", "name", "logo_url", "primary_color", "secondary_color"}).
			AddRow(testOrgID, "Test Org", expectedLogoURL, brandingPayload.PrimaryColor, brandingPayload.SecondaryColor)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "organizations" WHERE id = $1 ORDER BY "organizations"."id" LIMIT $2`)).
		    WithArgs(testOrgID,1).WillReturnRows(updatedOrgRows)


		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/orgs/%s/branding", testOrgID.String()), bodyBuf)
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

	t.Run("Successful update with only colors", func(t *testing.T) {
		filestorage.DefaultFileStorageProvider = nil // Nenhum upload de arquivo esperado

		brandingPayload := BrandingPayload{
			PrimaryColor:   "#FF0000",
			SecondaryColor: "#00FF00",
		}
		payloadJSON, _ := json.Marshal(brandingPayload)

		bodyBuf := &bytes.Buffer{}
		mpWriter := multipart.NewWriter(bodyBuf)
		_ = mpWriter.WriteField("data", string(payloadJSON))
		// Nenhum arquivo "logo_file" adicionado
		mpWriter.Close()

		orgRows := sqlmock.NewRows([]string{"id", "name", "primary_color", "secondary_color", "logo_url"}).
			AddRow(testOrgID, "Test Org", "", "", "http://oldlogo.com/logo.png") // Assume um logo_url existente
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "organizations" WHERE id = $1`)).
			WithArgs(testOrgID).WillReturnRows(orgRows)

		sqlMock.ExpectBegin()
		// LogoURL não deve ser alterado se nenhum arquivo for enviado
		sqlMock.ExpectExec(regexp.QuoteMeta(`UPDATE "organizations" SET "primary_color"=$1,"secondary_color"=$2,"updated_at"=$3 WHERE "id" = $4`)).
			WithArgs(brandingPayload.PrimaryColor, brandingPayload.SecondaryColor, sqlmock.AnyArg(), testOrgID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		sqlMock.ExpectCommit()

		updatedOrgRows := sqlmock.NewRows([]string{"id", "name", "logo_url", "primary_color", "secondary_color"}).
			AddRow(testOrgID, "Test Org", "http://oldlogo.com/logo.png", brandingPayload.PrimaryColor, brandingPayload.SecondaryColor)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "organizations" WHERE id = $1`)).
		    WithArgs(testOrgID).WillReturnRows(updatedOrgRows)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/orgs/%s/branding", testOrgID.String()), bodyBuf)
		req.Header.Set("Content-Type", mpWriter.FormDataContentType())
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "Response: %s", rr.Body.String())
		var respOrg models.Organization
		json.Unmarshal(rr.Body.Bytes(), &respOrg)
		assert.Equal(t, brandingPayload.PrimaryColor, respOrg.PrimaryColor)
		assert.Equal(t, "http://oldlogo.com/logo.png", respOrg.LogoURL) // Verifica se o logo antigo permaneceu
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Fail: Organization not found", func(t *testing.T) {
		filestorage.DefaultFileStorageProvider = nil
		nonExistentOrgID := uuid.New()
		payloadJSON, _ := json.Marshal(BrandingPayload{PrimaryColor: "#111111"})
		bodyBuf := &bytes.Buffer{}
		mpWriter := multipart.NewWriter(bodyBuf)
		_ = mpWriter.WriteField("data", string(payloadJSON))
		mpWriter.Close()

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "organizations" WHERE id = $1`)).
			WithArgs(nonExistentOrgID).WillReturnError(gorm.ErrRecordNotFound)

		// Roteador precisa ser configurado para a nonExistentOrgID no path, mas o token ainda é de testOrgID (admin)
		// A verificação de `checkOrgAdminOrManager` vai falhar primeiro se o orgId do path for diferente do token,
		// a menos que o usuário seja um superadmin (não implementado).
		// Para testar o "Org not found" do DB, o checkOrgAdminOrManager deve passar.
		// Vamos usar um router específico para este caso, onde o token e o path orgID são nonExistentOrgID,
		// mas o usuário do token é ainda um "admin" (para passar o check de role).
		tempRouter := getRouterWithOrgAdminContext(testUserID, nonExistentOrgID, models.RoleAdmin)
		tempRouter.PUT("/orgs/:orgId/branding", UpdateOrganizationBrandingHandler)

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/orgs/%s/branding", nonExistentOrgID.String()), bodyBuf)
		req.Header.Set("Content-Type", mpWriter.FormDataContentType())
		rr := httptest.NewRecorder()
		tempRouter.ServeHTTP(rr, req) // Usa tempRouter

		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Fail: User not authorized (not admin/manager of target org)", func(t *testing.T) {
		filestorage.DefaultFileStorageProvider = nil
		// Usuário (testUserID) é admin da testOrgID, mas tenta atualizar otherOrgID
		otherOrgID := uuid.New()
		payloadJSON, _ := json.Marshal(BrandingPayload{PrimaryColor: "#111111"})
		bodyBuf := &bytes.Buffer{}
		mpWriter := multipart.NewWriter(bodyBuf)
		_ = mpWriter.WriteField("data", string(payloadJSON))
		mpWriter.Close()

		// Nenhuma chamada ao DB esperada, pois a autorização deve falhar antes
		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/orgs/%s/branding", otherOrgID.String()), bodyBuf)
		req.Header.Set("Content-Type", mpWriter.FormDataContentType())
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req) // Usa o router original (testUserID é admin de testOrgID)

		assert.Equal(t, http.StatusForbidden, rr.Code) // checkOrgAdminOrManager falha
		assert.Contains(t, rr.Body.String(), "Você não pertence a esta organização")
	})

	t.Run("Fail: Invalid primary_color format", func(t *testing.T) {
		filestorage.DefaultFileStorageProvider = nil
		payloadJSON := `{"primary_color": "invalidcolor", "secondary_color": "#ABCDEF"}`
		bodyBuf := &bytes.Buffer{}
		mpWriter := multipart.NewWriter(bodyBuf)
		_ = mpWriter.WriteField("data", payloadJSON) // Envia JSON string diretamente
		mpWriter.Close()

		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/orgs/%s/branding", testOrgID.String()), bodyBuf)
		req.Header.Set("Content-Type", mpWriter.FormDataContentType())
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Formato de Cor Primária inválido")
	})

	t.Run("Fail: FileStorageProvider not configured but logo uploaded", func(t *testing.T) {
		filestorage.DefaultFileStorageProvider = nil // Simula não configurado

		payloadJSON, _ := json.Marshal(BrandingPayload{PrimaryColor: "#123123"})
		bodyBuf := &bytes.Buffer{}
		mpWriter := multipart.NewWriter(bodyBuf)
		_ = mpWriter.WriteField("data", string(payloadJSON))
		fileWriter, _ := mpWriter.CreateFormFile("logo_file", "logo.png")
		_, _ = fileWriter.Write([]byte("dummy"))
		mpWriter.Close()

		// Mock para buscar a organização (o handler chega até aqui antes de checar o provider)
		orgRows := sqlmock.NewRows([]string{"id"}).AddRow(testOrgID)
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "organizations" WHERE id = $1`)).
			WithArgs(testOrgID).WillReturnRows(orgRows)


		req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/orgs/%s/branding", testOrgID.String()), bodyBuf)
		req.Header.Set("Content-Type", mpWriter.FormDataContentType())
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "Serviço de armazenamento de arquivos não configurado")
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	// TODO: Adicionar mais testes para os outros cenários listados no TODO original:
	// 2. Sucesso apenas com logo, sem cores (parcialmente coberto, mas um teste explícito seria bom)
	// 8. Falha: Upload de logo - arquivo muito grande
	// 9. Falha: Upload de logo - tipo de arquivo não permitido
	// 10. Falha: Upload de logo - erro no FileStorageProvider.UploadFile
	// 12. Falha: JSON 'data' malformado ou ausente (ou campo 'data' não é JSON válido)
}


func TestGetOrganizationBrandingHandler(t *testing.T) {
	setupMockDB(t) // Adicionado
	gin.SetMode(gin.TestMode)
	// Para este handler, o usuário não precisa ser admin, apenas pertencer à organização (ou ser público)
	// O handler GetOrganizationBrandingHandler tem uma lógica de autorização comentada.
	// Se for público, o token não importa. Se for protegido por org, o token importa.
	// Vamos testar o caso protegido onde o usuário (testUserID) pertence à testOrgID.
	router := getRouterWithAuthenticatedContext(testUserID, testOrgID) // testUserID pertence a testOrgID
	router.GET("/orgs/:orgId/branding", GetOrganizationBrandingHandler)

	t.Run("Successful get branding for existing organization", func(t *testing.T) {
		expectedBranding := models.Organization{
			ID:             testOrgID,
			Name:           "Test Organization",
			LogoURL:        "http://example.com/logo.png",
			PrimaryColor:   "#AAAAAA",
			SecondaryColor: "#BBBBBB",
		}
		rows := sqlmock.NewRows([]string{"id", "name", "logo_url", "primary_color", "secondary_color"}).
			AddRow(expectedBranding.ID, expectedBranding.Name, expectedBranding.LogoURL, expectedBranding.PrimaryColor, expectedBranding.SecondaryColor)

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT "id","name","logo_url","primary_color","secondary_color" FROM "organizations" WHERE id = $1 ORDER BY "organizations"."id" LIMIT $2`)).
			WithArgs(testOrgID, 1).WillReturnRows(rows)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/orgs/%s/branding", testOrgID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "Response: %s", rr.Body.String())
		var respData map[string]string // O handler retorna um gin.H
		err := json.Unmarshal(rr.Body.Bytes(), &respData)
		assert.NoError(t, err)
		assert.Equal(t, expectedBranding.LogoURL, respData["logo_url"])
		assert.Equal(t, expectedBranding.PrimaryColor, respData["primary_color"])
		assert.Equal(t, expectedBranding.SecondaryColor, respData["secondary_color"])
		assert.Equal(t, expectedBranding.Name, respData["name"])
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Fail: Organization not found", func(t *testing.T) {
		nonExistentOrgID := uuid.New()

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT "id","name","logo_url","primary_color","secondary_color" FROM "organizations" WHERE id = $1`)).
			WithArgs(nonExistentOrgID).WillReturnError(gorm.ErrRecordNotFound)

		// Roteador para simular acesso à nonExistentOrgID. O token ainda é de testUserID@testOrgID.
		// A lógica de autorização no handler GetOrganizationBrandingHandler está comentada,
		// então ele tentará buscar a org pelo ID do path diretamente.
		// Se a autorização estivesse ativa e o usuário não fosse da nonExistentOrgID, daria Forbidden.
		// Como está, deve dar NotFound do DB.

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/orgs/%s/branding", nonExistentOrgID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req) // Usa o router original, o path orgId é o que importa para o DB query

		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Fail: Accessing other organization's branding (if auth was strict)", func(t *testing.T) {
		// Este teste depende da lógica de autorização no handler.
		// Atualmente, a verificação `tokenOrgID.(uuid.UUID) != targetOrgID` está comentada.
		// Se ela fosse ativa, este teste seria relevante.
		// otherOrgID := uuid.New() // Comentado pois o teste está pulado
		// O usuário testUserID@testOrgID tenta acessar otherOrgID
		// Nenhuma chamada ao DB deve ocorrer se a autorização falhar primeiro.

		// Para simular que a autorização está ativa, precisaríamos de um handler modificado
		// ou testar a função de autorização separadamente.
		// Por agora, este teste pode ser mais conceitual ou adaptado se a auth for descomentada.
		t.Skip("Skipping test as current GetOrganizationBrandingHandler auth is permissive / commented out for public-like access")

		// Se a auth fosse:
		// req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/orgs/%s/branding", otherOrgID.String()), nil)
		// rr := httptest.NewRecorder()
		// router.ServeHTTP(rr, req) // testUserID@testOrgID acessando otherOrgID
		// assert.Equal(t, http.StatusForbidden, rr.Code)
	})
}
