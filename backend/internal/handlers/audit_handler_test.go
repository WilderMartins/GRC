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

// Assumindo que testUserID, testOrgID são definidos em main_test_handler.go
// e getRouterWithAuthenticatedContext está disponível.

var testFrameworkID = uuid.New()
var testControlID = uuid.New() // UUID do AuditControl
var testAssessmentID = uuid.New()

func TestListFrameworksHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := getRouterWithAuthenticatedContext(testUserID, testOrgID) // Auth pode não ser estritamente necessário para listar frameworks públicos
	router.GET("/audit/frameworks", ListFrameworksHandler)

	t.Run("Successful list frameworks", func(t *testing.T) {
		mockFrameworks := []models.AuditFramework{
			{ID: testFrameworkID, Name: "NIST CSF Test", CreatedAt: time.Now()},
			{ID: uuid.New(), Name: "ISO 27001 Test", CreatedAt: time.Now()},
		}

		rows := sqlmock.NewRows([]string{"id", "name", "created_at"}).
			AddRow(mockFrameworks[0].ID, mockFrameworks[0].Name, mockFrameworks[0].CreatedAt).
			AddRow(mockFrameworks[1].ID, mockFrameworks[1].Name, mockFrameworks[1].CreatedAt)

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_frameworks"`)).WillReturnRows(rows)

		req, _ := http.NewRequest(http.MethodGet, "/audit/frameworks", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var frameworks []models.AuditFramework
		err := json.Unmarshal(rr.Body.Bytes(), &frameworks)
		assert.NoError(t, err)
		assert.Len(t, frameworks, 2)
		assert.Equal(t, "NIST CSF Test", frameworks[0].Name)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

func TestGetFrameworkControlsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := getRouterWithAuthenticatedContext(testUserID, testOrgID)
	router.GET("/audit/frameworks/:frameworkId/controls", GetFrameworkControlsHandler)

	t.Run("Successful get controls for framework", func(t *testing.T) {
		mockControls := []models.AuditControl{
			{ID: testControlID, FrameworkID: testFrameworkID, ControlID: "TC-1", Description: "Test Control 1", Family: "Test Family"},
			{ID: uuid.New(), FrameworkID: testFrameworkID, ControlID: "TC-2", Description: "Test Control 2", Family: "Test Family"},
		}

		rows := sqlmock.NewRows([]string{"id", "framework_id", "control_id", "description", "family"}).
			AddRow(mockControls[0].ID, mockControls[0].FrameworkID, mockControls[0].ControlID, mockControls[0].Description, mockControls[0].Family).
			AddRow(mockControls[1].ID, mockControls[1].FrameworkID, mockControls[1].ControlID, mockControls[1].Description, mockControls[1].Family)

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_controls" WHERE framework_id = $1 ORDER BY control_id asc`)).
			WithArgs(testFrameworkID).
			WillReturnRows(rows)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/audit/frameworks/%s/controls", testFrameworkID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var controls []models.AuditControl
		err := json.Unmarshal(rr.Body.Bytes(), &controls)
		assert.NoError(t, err)
		assert.Len(t, controls, 2)
		assert.Equal(t, "TC-1", controls[0].ControlID)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Framework not found when getting controls", func(t *testing.T) {
		nonExistentFrameworkID := uuid.New()
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_controls" WHERE framework_id = $1`)).
			WithArgs(nonExistentFrameworkID).
			WillReturnRows(sqlmock.NewRows(nil)) // No controls

		// Mock para a verificação de existência do framework
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_frameworks" WHERE id = $1`)).
			WithArgs(nonExistentFrameworkID).
			WillReturnError(gorm.ErrRecordNotFound)


		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/audit/frameworks/%s/controls", nonExistentFrameworkID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

func TestCreateOrUpdateAssessmentHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := getRouterWithAuthenticatedContext(testUserID, testOrgID)
	router.POST("/audit/assessments", CreateOrUpdateAssessmentHandler)

	assessmentDate := time.Now().Format("2006-01-02")
	score := 50

	payload := AssessmentPayload{
		AuditControlID: testControlID.String(),
		Status:         models.ControlStatusPartiallyConformant,
		EvidenceURL:    "http://example.com/evidence.pdf",
		Score:          &score,
		AssessmentDate: assessmentDate,
	}
	body, _ := json.Marshal(payload)

	t.Run("Successful assessment upsert - create", func(t *testing.T) {
		sqlMock.ExpectBegin()
		// Mock para o ON CONFLICT ... DO UPDATE
		// A query exata pode variar um pouco com base na versão do GORM/Postgres.
		// O importante é que ele tente inserir e, em caso de conflito, atualize.
		// Para CREATE (sem conflito):
		sqlMock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "audit_assessments" ("id","organization_id","audit_control_id","status","evidence_url","score","assessment_date","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT ("organization_id","audit_control_id") DO UPDATE SET "status"=$10,"evidence_url"=$11,"score"=$12,"assessment_date"=$13,"updated_at"=$14 RETURNING "id"`)).
			WithArgs(sqlmock.AnyArg(), testOrgID, testControlID, payload.Status, payload.EvidenceURL, *payload.Score, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), payload.Status, payload.EvidenceURL, *payload.Score, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(testAssessmentID)) // Simula a criação
		sqlMock.ExpectCommit()

		// Mock para o re-fetch após o upsert
		// Colunas: id, organization_id, audit_control_id, status, evidence_url, score, assessment_date
		refetchRows := sqlmock.NewRows([]string{"id", "organization_id", "audit_control_id", "status", "evidence_url", "score", "assessment_date"}).
			AddRow(testAssessmentID, testOrgID, testControlID, payload.Status, payload.EvidenceURL, *payload.Score, time.Now())
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_assessments" WHERE organization_id = $1 AND audit_control_id = $2 ORDER BY "audit_assessments"."id" LIMIT $3`)).
			WithArgs(testOrgID, testControlID, 1).
			WillReturnRows(refetchRows)


		req, _ := http.NewRequest(http.MethodPost, "/audit/assessments", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "Response code should be 200 OK: %s", rr.Body.String())
		var createdAssessment models.AuditAssessment
		err := json.Unmarshal(rr.Body.Bytes(), &createdAssessment)
		assert.NoError(t, err)
		assert.Equal(t, testAssessmentID, createdAssessment.ID) // ID retornado pelo re-fetch
		assert.Equal(t, payload.Status, createdAssessment.Status)
		assert.Equal(t, *payload.Score, createdAssessment.Score)

		assert.NoError(t, sqlMock.ExpectationsWereMet(), "SQL mock expectations were not met")
	})

    // TODO: Adicionar teste para o caso de UPDATE do upsert.
    // Isso exigiria que o primeiro ExpectQuery simulasse um conflito ou que a lógica do GORM fosse mockada de forma diferente.
    // Para simplificar, o teste acima cobre o fluxo geral do upsert que resulta em uma criação ou atualização.
    // Um teste específico de UPDATE envolveria GORM detectando o conflito e aplicando o DO UPDATE path.
}


func TestGetAssessmentForControlHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := getRouterWithAuthenticatedContext(testUserID, testOrgID)
	router.GET("/audit/assessments/control/:controlId", GetAssessmentForControlHandler)

	t.Run("Successful get assessment for control", func(t *testing.T) {
		mockAssessment := models.AuditAssessment{
			ID:             testAssessmentID,
			OrganizationID: testOrgID,
			AuditControlID: testControlID,
			Status:         models.ControlStatusConformant,
			Score:          100,
			AssessmentDate: time.Now(),
		}
		// Mock para o preload de AuditControl (simplificado, apenas o ID)
		mockAssessment.AuditControl.ID = testControlID

		rows := sqlmock.NewRows([]string{"id", "organization_id", "audit_control_id", "status", "score", "assessment_date"}).
			AddRow(mockAssessment.ID, mockAssessment.OrganizationID, mockAssessment.AuditControlID, mockAssessment.Status, mockAssessment.Score, mockAssessment.AssessmentDate)

		// A query exata para preload pode variar.
		// Simulação básica:
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_assessments" WHERE organization_id = $1 AND audit_control_id = $2`)).
			WithArgs(testOrgID, testControlID).
			WillReturnRows(rows)

        // Mock para o Preload("AuditControl")
        // Assumindo que o GORM faz uma query separada para o preload após buscar o assessment.
        controlRows := sqlmock.NewRows([]string{"id", "control_id", "description"}).
            AddRow(testControlID, "TEST-C1", "Test control description")
        sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_controls" WHERE "audit_controls"."id" = $1`)).
            WithArgs(testControlID).
            WillReturnRows(controlRows)


		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/audit/assessments/control/%s", testControlID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "Response code should be 200 OK: %s", rr.Body.String())
		var fetchedAssessment models.AuditAssessment
		err := json.Unmarshal(rr.Body.Bytes(), &fetchedAssessment)
		assert.NoError(t, err)
		assert.Equal(t, mockAssessment.ID, fetchedAssessment.ID)
		assert.Equal(t, mockAssessment.Status, fetchedAssessment.Status)
        assert.Equal(t, testControlID, fetchedAssessment.AuditControl.ID) // Verifica se o preload funcionou (pelo menos o ID)

		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Assessment not found for control", func(t *testing.T) {
		nonExistentControlID := uuid.New()
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_assessments" WHERE organization_id = $1 AND audit_control_id = $2`)).
			WithArgs(testOrgID, nonExistentControlID).
			WillReturnError(gorm.ErrRecordNotFound)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/audit/assessments/control/%s", nonExistentControlID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

func TestListOrgAssessmentsByFrameworkHandler(t *testing.T) {
    gin.SetMode(gin.TestMode)
    // Roteador com contexto de usuário admin para testOrgID
    router := getRouterWithOrgAdminContext(testUserID, testOrgID, models.RoleAdmin)
    router.GET("/audit/organizations/:orgId/frameworks/:frameworkId/assessments", ListOrgAssessmentsByFrameworkHandler)

    t.Run("Successful list assessments by framework", func(t *testing.T) {
        control1ID := uuid.New()
        control2ID := uuid.New()

        mockControls := []models.AuditControl{
            {ID: control1ID},
            {ID: control2ID},
        }
        // Mock para buscar IDs dos controles do framework
        // GORM Pluck: SELECT "id" FROM "audit_controls" WHERE framework_id = $1
        controlRows := sqlmock.NewRows([]string{"id"}).AddRow(control1ID).AddRow(control2ID)
        sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT "id" FROM "audit_controls" WHERE framework_id = $1`)).
            WithArgs(testFrameworkID).
            WillReturnRows(controlRows)

        mockAssessments := []models.AuditAssessment{
            {ID: uuid.New(), OrganizationID: testOrgID, AuditControlID: control1ID, Status: models.ControlStatusConformant, AuditControl: models.AuditControl{ID: control1ID, ControlID: "C1"}},
            {ID: uuid.New(), OrganizationID: testOrgID, AuditControlID: control2ID, Status: models.ControlStatusPartiallyConformant, AuditControl: models.AuditControl{ID: control2ID, ControlID: "C2"}},
        }
        assessmentRows := sqlmock.NewRows([]string{"id", "organization_id", "audit_control_id", "status"}).
            AddRow(mockAssessments[0].ID, mockAssessments[0].OrganizationID, mockAssessments[0].AuditControlID, mockAssessments[0].Status).
            AddRow(mockAssessments[1].ID, mockAssessments[1].OrganizationID, mockAssessments[1].AuditControlID, mockAssessments[1].Status)

        // Mock para buscar as avaliações
        // A query exata do GORM com IN pode variar um pouco na formatação do placeholder.
        // Usando regexp.QuoteMeta para partes fixas e .* para o IN.
        sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_assessments" WHERE organization_id = $1 AND audit_control_id IN ($2,$3)`)).
            WithArgs(testOrgID, control1ID, control2ID). // O order dos UUIDs no IN pode variar, sqlmock pode precisar de mais flexibilidade aqui se falhar.
            WillReturnRows(assessmentRows)

        // Mocks para os Preloads de AuditControl para cada assessment
        preloadControl1Rows := sqlmock.NewRows([]string{"id", "control_id"}).AddRow(control1ID, "C1")
        sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_controls" WHERE "audit_controls"."id" = $1`)).WithArgs(control1ID).WillReturnRows(preloadControl1Rows)

        preloadControl2Rows := sqlmock.NewRows([]string{"id", "control_id"}).AddRow(control2ID, "C2")
        sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "audit_controls" WHERE "audit_controls"."id" = $1`)).WithArgs(control2ID).WillReturnRows(preloadControl2Rows)


        req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/audit/organizations/%s/frameworks/%s/assessments", testOrgID.String(), testFrameworkID.String()), nil)
        rr := httptest.NewRecorder()
        router.ServeHTTP(rr, req)

        assert.Equal(t, http.StatusOK, rr.Code, "Response code should be 200 OK: %s", rr.Body.String())
        var assessments []models.AuditAssessment
        err := json.Unmarshal(rr.Body.Bytes(), &assessments)
        assert.NoError(t, err)
        assert.Len(t, assessments, 2)
        assert.Equal(t, "C1", assessments[0].AuditControl.ControlID) // Verifica se o preload funcionou

        assert.NoError(t, sqlMock.ExpectationsWereMet())
    })
}
