package handlers

import (
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

func TestListC2M2DomainsHandler(t *testing.T) {
	setupMockDB(t) // Assume setupMockDB está disponível a partir de outro _test.go ou main_test.go
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/c2m2/domains", ListC2M2DomainsHandler)

	mockDomains := []models.C2M2Domain{
		{ID: uuid.New(), Name: "Risk Management", Code: "RM", CreatedAt: time.Now()},
		{ID: uuid.New(), Name: "Situational Awareness", Code: "SA", CreatedAt: time.Now()},
	}

	rows := sqlmock.NewRows([]string{"id", "name", "code", "created_at"}).
		AddRow(mockDomains[0].ID, mockDomains[0].Name, mockDomains[0].Code, mockDomains[0].CreatedAt).
		AddRow(mockDomains[1].ID, mockDomains[1].Name, mockDomains[1].Code, mockDomains[1].CreatedAt)

	sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "c2m2_domains" ORDER BY code asc`)).
		WillReturnRows(rows)

	req, _ := http.NewRequest(http.MethodGet, "/c2m2/domains", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var domains []models.C2M2Domain
	err := json.Unmarshal(rr.Body.Bytes(), &domains)
	assert.NoError(t, err)
	assert.Len(t, domains, 2)
	assert.Equal(t, "Risk Management", domains[0].Name)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestListC2M2PracticesByDomainHandler(t *testing.T) {
	setupMockDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/c2m2/domains/:domainId/practices", ListC2M2PracticesByDomainHandler)

	domainID := uuid.New()

	t.Run("Successful list practices", func(t *testing.T) {
		mockDomain := models.C2M2Domain{ID: domainID, Name: "Risk Management", Code: "RM"}
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "c2m2_domains" WHERE id = $1 ORDER BY "c2m2_domains"."id" LIMIT 1`)).
			WithArgs(domainID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code"}).AddRow(mockDomain.ID, mockDomain.Name, mockDomain.Code))

		mockPractices := []models.C2M2Practice{
			{ID: uuid.New(), DomainID: domainID, Code: "RM.1.1", Description: "Desc 1", TargetMIL: 1},
			{ID: uuid.New(), DomainID: domainID, Code: "RM.2.1", Description: "Desc 2", TargetMIL: 2},
		}
		rows := sqlmock.NewRows([]string{"id", "domain_id", "code", "description", "target_mil"}).
			AddRow(mockPractices[0].ID, mockPractices[0].DomainID, mockPractices[0].Code, mockPractices[0].Description, mockPractices[0].TargetMIL).
			AddRow(mockPractices[1].ID, mockPractices[1].DomainID, mockPractices[1].Code, mockPractices[1].Description, mockPractices[1].TargetMIL)

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "c2m2_practices" WHERE domain_id = $1 ORDER BY code asc`)).
			WithArgs(domainID).
			WillReturnRows(rows)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/c2m2/domains/%s/practices", domainID), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var practices []models.C2M2Practice
		err := json.Unmarshal(rr.Body.Bytes(), &practices)
		assert.NoError(t, err)
		assert.Len(t, practices, 2)
		assert.Equal(t, "RM.1.1", practices[0].Code)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("Domain not found", func(t *testing.T) {
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "c2m2_domains" WHERE id = $1`)).
			WithArgs(domainID).
			WillReturnError(gorm.ErrRecordNotFound)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/c2m2/domains/%s/practices", domainID), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}
