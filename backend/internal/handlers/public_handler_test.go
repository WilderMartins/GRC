package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// setupMockDB é necessário aqui também. Assumindo que está disponível de um main_test.go ou helper.
// Se não, precisaria ser definido aqui.

func TestGetSetupStatusHandler(t *testing.T) {
	setupMockDB(t) // Configura o mock DB globalmente para o teste
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/setup-status", GetSetupStatusHandler)

	testCases := []struct {
		name           string
		mockDB         func(sqlMock sqlmock.Sqlmock)
		expectedStatus int
		expectedResp   SetupStatusResponse
	}{
		{
			name: "Database Not Connected",
			mockDB: func(sm sqlmock.Sqlmock) {
				// Simular erro de Ping
				sm.ExpectPing().WillReturnError(errors.New("connection failed"))
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedResp: SetupStatusResponse{
				Status:  "database_not_connected",
				Message: "Não foi possível conectar ao banco de dados. Verifique as credenciais e a conectividade.",
			},
		},
		{
			name: "Migrations Not Run",
			mockDB: func(sm sqlmock.Sqlmock) {
				sm.ExpectPing()
				// A query para HasTable pode variar com o dialeto GORM, regex é mais seguro.
				sm.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1`)).
					WithArgs("users").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0)) // Tabela não existe
			},
			expectedStatus: http.StatusOK,
			expectedResp: SetupStatusResponse{
				Status:  "migrations_not_run",
				Message: "Conexão com o banco de dados OK, mas as tabelas da aplicação não foram criadas. Execute o setup.",
			},
		},
		{
			name: "Setup Pending - No Organizations",
			mockDB: func(sm sqlmock.Sqlmock) {
				sm.ExpectPing()
				sm.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1`)).
					WithArgs("users").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1)) // Tabela existe
				sm.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "organizations"`)).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0)) // Nenhuma organização
			},
			expectedStatus: http.StatusOK,
			expectedResp: SetupStatusResponse{
				Status:  "setup_pending_org",
				Message: "Migrações concluídas, mas a primeira organização e o usuário administrador precisam ser criados.",
			},
		},
		{
			name: "Setup Pending - No Admin User",
			mockDB: func(sm sqlmock.Sqlmock) {
				sm.ExpectPing()
				sm.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1`)).
					WithArgs("users").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				sm.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "organizations"`)).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1)) // 1 organização
				sm.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users" WHERE role = $1`)).
					WithArgs(models.RoleAdmin).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0)) // Nenhum admin
			},
			expectedStatus: http.StatusOK,
			expectedResp: SetupStatusResponse{
				Status:  "setup_pending_admin",
				Message: "Organização criada, mas o usuário administrador não foi encontrado. Complete o setup.",
			},
		},
		{
			name: "Setup Complete",
			mockDB: func(sm sqlmock.Sqlmock) {
				sm.ExpectPing()
				sm.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = $1`)).
					WithArgs("users").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				sm.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "organizations"`)).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				sm.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users" WHERE role = $1`)).
					WithArgs(models.RoleAdmin).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1)) // 1 admin
			},
			expectedStatus: http.StatusOK,
			expectedResp: SetupStatusResponse{
				Status:  "setup_complete",
				Message: "A aplicação está configurada e pronta para uso.",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// A função setupMockDB já foi chamada, mas precisamos registrar as expectativas específicas para este caso.
			tc.mockDB(sqlMock)

			req, _ := http.NewRequest(http.MethodGet, "/setup-status", nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code, "Response code mismatch")

			var respBody SetupStatusResponse
			err := json.Unmarshal(rr.Body.Bytes(), &respBody)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedResp, respBody)

			// Verificar se todas as expectativas do mock foram atendidas para este caso de teste
			assert.NoError(t, sqlMock.ExpectationsWereMet())
		})
	}
}
