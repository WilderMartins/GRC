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
	// "gorm.io/gorm" // Não é necessário aqui se não formos simular gorm.ErrRecordNotFound diretamente
)

// testOrgAdminID, testUserID (que é um admin/manager de testOrgID) são definidos em main_test_handler.go
// testUserNonAdminID (um usuário comum de testOrgID)
var testUserNonAdminID = uuid.New()
var testTargetUserID = uuid.New() // O usuário que será gerenciado

func TestListOrganizationUsersHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Roteador com contexto de usuário admin (testUserID) para testOrgID
	router := getRouterWithOrgAdminContext(testUserID, testOrgID, models.RoleAdmin)
	router.GET("/orgs/:orgId/users", ListOrganizationUsersHandler)

	t.Run("Successful list users", func(t *testing.T) {
		mockUsers := []models.User{
			{ID: testTargetUserID, OrganizationID: testOrgID, Name: "User One", Email: "one@org.com", Role: models.RoleUser, IsActive: true, CreatedAt: time.Now()},
			{ID: uuid.New(), OrganizationID: testOrgID, Name: "User Two", Email: "two@org.com", Role: models.RoleManager, IsActive: true, CreatedAt: time.Now()},
		}

		// Mock para Count
		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users" WHERE organization_id = $1`)).
			WithArgs(testOrgID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(len(mockUsers)))

		// Mock para Find
		rows := sqlmock.NewRows([]string{"id", "organization_id", "name", "email", "role", "is_active", "created_at", "updated_at"}).
			AddRow(mockUsers[0].ID, mockUsers[0].OrganizationID, mockUsers[0].Name, mockUsers[0].Email, mockUsers[0].Role, mockUsers[0].IsActive, mockUsers[0].CreatedAt, time.Now()).
			AddRow(mockUsers[1].ID, mockUsers[1].OrganizationID, mockUsers[1].Name, mockUsers[1].Email, mockUsers[1].Role, mockUsers[1].IsActive, mockUsers[1].CreatedAt, time.Now())

		sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE organization_id = $1 ORDER BY name asc LIMIT $2 OFFSET $3`)).
			WithArgs(testOrgID, DefaultPageSize, 0). // Assumindo DefaultPage=1, DefaultPageSize=10
			WillReturnRows(rows)

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/orgs/%s/users", testOrgID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code, "Response: %s", rr.Body.String())
		var resp PaginatedResponse
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, int64(len(mockUsers)), resp.TotalItems)
		assert.Len(t, resp.Items, len(mockUsers))
		// TODO: Verificar conteúdo de resp.Items se necessário
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

    t.Run("Unauthorized access by non-admin/manager", func(t *testing.T) {
        nonAdminRouter := getRouterWithOrgAdminContext(testUserNonAdminID, testOrgID, models.RoleUser)
        nonAdminRouter.GET("/orgs/:orgId/users", ListOrganizationUsersHandler)

        req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/orgs/%s/users", testOrgID.String()), nil)
		rr := httptest.NewRecorder()
		nonAdminRouter.ServeHTTP(rr, req)
        assert.Equal(t, http.StatusForbidden, rr.Code)
    })
}

func TestGetOrganizationUserHandler(t *testing.T) {
    gin.SetMode(gin.TestMode)
	router := getRouterWithOrgAdminContext(testUserID, testOrgID, models.RoleAdmin)
	router.GET("/orgs/:orgId/users/:userId", GetOrganizationUserHandler)

    t.Run("Successful get user details", func(t *testing.T) {
        mockUser := models.User{ID: testTargetUserID, OrganizationID: testOrgID, Name: "Target User", Email: "target@org.com", Role: models.RoleUser, IsActive: true}
        rows := sqlmock.NewRows([]string{"id", "organization_id", "name", "email", "role", "is_active"}).
            AddRow(mockUser.ID, mockUser.OrganizationID, mockUser.Name, mockUser.Email, mockUser.Role, mockUser.IsActive)

        sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 AND organization_id = $2 ORDER BY "users"."id" LIMIT $3`)).
            WithArgs(testTargetUserID, testOrgID, 1).
            WillReturnRows(rows)

        req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/orgs/%s/users/%s", testOrgID.String(), testTargetUserID.String()), nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

        assert.Equal(t, http.StatusOK, rr.Code)
        var respUser UserResponse
        err := json.Unmarshal(rr.Body.Bytes(), &respUser)
        assert.NoError(t, err)
        assert.Equal(t, mockUser.Name, respUser.Name)
        assert.NoError(t, sqlMock.ExpectationsWereMet())
    })

    t.Run("GetOrganizationUserHandler - User Not Found", func(t *testing.T) {
        nonExistentUserID := uuid.New()
        sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 AND organization_id = $2`)).
            WithArgs(nonExistentUserID, testOrgID).
            WillReturnError(gorm.ErrRecordNotFound)

        req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/orgs/%s/users/%s", testOrgID.String(), nonExistentUserID.String()), nil)
        rr := httptest.NewRecorder()
        router.ServeHTTP(rr, req)
        assert.Equal(t, http.StatusNotFound, rr.Code)
        assert.NoError(t, sqlMock.ExpectationsWereMet())
    })

    t.Run("GetOrganizationUserHandler - Invalid userId format", func(t *testing.T) {
        req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/orgs/%s/users/%s", testOrgID.String(), "not-a-uuid"), nil)
        rr := httptest.NewRecorder()
        router.ServeHTTP(rr, req)
        assert.Equal(t, http.StatusBadRequest, rr.Code)
        assert.Contains(t, rr.Body.String(), "Formato de ID do usuário inválido")
    })
}


func TestUpdateOrganizationUserRoleHandler(t *testing.T) {
    gin.SetMode(gin.TestMode)
    // Usuário admin (testUserID) atualizando outro usuário (testTargetUserID)
	router := getRouterWithOrgAdminContext(testUserID, testOrgID, models.RoleAdmin)
	router.PUT("/orgs/:orgId/users/:userId/role", UpdateOrganizationUserRoleHandler)

    t.Run("Successful role update", func(t *testing.T) {
        payload := UpdateUserRolePayload{Role: models.RoleManager}
        body, _ := json.Marshal(payload)

        userToUpdate := models.User{ID: testTargetUserID, OrganizationID: testOrgID, Name: "User To Update Role", Role: models.RoleUser, IsActive: true}
        rowsUser := sqlmock.NewRows([]string{"id", "organization_id", "name", "role", "is_active"}).
            AddRow(userToUpdate.ID, userToUpdate.OrganizationID, userToUpdate.Name, userToUpdate.Role, userToUpdate.IsActive)

        sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 AND organization_id = $2 ORDER BY "users"."id" LIMIT $3`)).
            WithArgs(testTargetUserID, testOrgID, 1).
            WillReturnRows(rowsUser)

        sqlMock.ExpectBegin()
        sqlMock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET "role"=$1,"updated_at"=$2 WHERE "id" = $3`)).
            WithArgs(payload.Role, sqlmock.AnyArg(), testTargetUserID).
            WillReturnResult(sqlmock.NewResult(0,1))
        sqlMock.ExpectCommit()

        req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/orgs/%s/users/%s/role", testOrgID.String(), testTargetUserID.String()), bytes.NewBuffer(body))
        req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

        assert.Equal(t, http.StatusOK, rr.Code, "Response: %s", rr.Body.String())
        var respUser UserResponse
        err := json.Unmarshal(rr.Body.Bytes(), &respUser)
        assert.NoError(t, err)
        assert.Equal(t, payload.Role, respUser.Role)
        assert.NoError(t, sqlMock.ExpectationsWereMet())
    })

    t.Run("Fail to demote last admin/manager (self)", func(t *testing.T) {
        // testUserID é o admin fazendo a ação em si mesmo
        payload := UpdateUserRolePayload{Role: models.RoleUser}
        body, _ := json.Marshal(payload)

        adminUser := models.User{ID: testUserID, OrganizationID: testOrgID, Name: "Admin User", Role: models.RoleAdmin, IsActive: true}
        rowsAdmin := sqlmock.NewRows([]string{"id", "organization_id", "name", "role", "is_active"}).
            AddRow(adminUser.ID, adminUser.OrganizationID, adminUser.Name, adminUser.Role, adminUser.IsActive)

        sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 AND organization_id = $2 ORDER BY "users"."id" LIMIT $3`)).
            WithArgs(testUserID, testOrgID, 1).
            WillReturnRows(rowsAdmin)

        // Mock para a contagem de admins/managers
        sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users" WHERE organization_id = $1 AND (role = $2 OR role = $3)`)).
            WithArgs(testOrgID, models.RoleAdmin, models.RoleManager).
            WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1)) // Só existe 1 admin/manager

        req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/orgs/%s/users/%s/role", testOrgID.String(), testUserID.String()), bytes.NewBuffer(body))
        req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

        assert.Equal(t, http.StatusForbidden, rr.Code)
        assert.Contains(t, rr.Body.String(), "Não é possível rebaixar o último administrador/gerente")
        assert.NoError(t, sqlMock.ExpectationsWereMet())
    })
}

func TestUpdateOrganizationUserStatusHandler(t *testing.T) {
    gin.SetMode(gin.TestMode)
	router := getRouterWithOrgAdminContext(testUserID, testOrgID, models.RoleAdmin)
	router.PUT("/orgs/:orgId/users/:userId/status", UpdateOrganizationUserStatusHandler)

    t.Run("Successful deactivate user", func(t *testing.T) {
        isActive := false
        payload := UpdateUserStatusPayload{IsActive: &isActive}
        body, _ := json.Marshal(payload)

        userToUpdate := models.User{ID: testTargetUserID, OrganizationID: testOrgID, Name: "User To Deactivate", Role: models.RoleUser, IsActive: true}
        rowsUser := sqlmock.NewRows([]string{"id", "organization_id", "name", "role", "is_active"}).
            AddRow(userToUpdate.ID, userToUpdate.OrganizationID, userToUpdate.Name, userToUpdate.Role, userToUpdate.IsActive)

        sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 AND organization_id = $2 ORDER BY "users"."id" LIMIT $3`)).
            WithArgs(testTargetUserID, testOrgID, 1).
            WillReturnRows(rowsUser)

        sqlMock.ExpectBegin()
        sqlMock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET "is_active"=$1,"updated_at"=$2 WHERE "id" = $3`)).
            WithArgs(false, sqlmock.AnyArg(), testTargetUserID).
            WillReturnResult(sqlmock.NewResult(0,1))
        sqlMock.ExpectCommit()

        req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/orgs/%s/users/%s/status", testOrgID.String(), testTargetUserID.String()), bytes.NewBuffer(body))
        req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

        assert.Equal(t, http.StatusOK, rr.Code, "Response: %s", rr.Body.String())
        var respUser UserResponse
        err := json.Unmarshal(rr.Body.Bytes(), &respUser)
        assert.NoError(t, err)
        assert.False(t, respUser.IsActive)
        assert.NoError(t, sqlMock.ExpectationsWereMet())
    })


    t.Run("Successful activate user", func(t *testing.T) {
        isActive := true
        payload := UpdateUserStatusPayload{IsActive: &isActive}
        body, _ := json.Marshal(payload)

        userToUpdate := models.User{ID: testTargetUserID, OrganizationID: testOrgID, Name: "User To Activate", Role: models.RoleUser, IsActive: false} // Começa inativo
        rowsUser := sqlmock.NewRows([]string{"id", "organization_id", "name", "role", "is_active"}).
            AddRow(userToUpdate.ID, userToUpdate.OrganizationID, userToUpdate.Name, userToUpdate.Role, userToUpdate.IsActive)

        sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 AND organization_id = $2 ORDER BY "users"."id" LIMIT $3`)).
            WithArgs(testTargetUserID, testOrgID, 1).
            WillReturnRows(rowsUser)

        sqlMock.ExpectBegin()
        sqlMock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET "is_active"=$1,"updated_at"=$2 WHERE "id" = $3`)).
            WithArgs(true, sqlmock.AnyArg(), testTargetUserID).
            WillReturnResult(sqlmock.NewResult(0,1))
        sqlMock.ExpectCommit()

        req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/orgs/%s/users/%s/status", testOrgID.String(), testTargetUserID.String()), bytes.NewBuffer(body))
        req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

        assert.Equal(t, http.StatusOK, rr.Code, "Response: %s", rr.Body.String())
        var respUser UserResponse
        err := json.Unmarshal(rr.Body.Bytes(), &respUser)
        assert.NoError(t, err)
        assert.True(t, respUser.IsActive)
        assert.NoError(t, sqlMock.ExpectationsWereMet())
    })

	t.Run("Fail to deactivate last active admin/manager (self)", func(t *testing.T) {
		isActive := false
        payload := UpdateUserStatusPayload{IsActive: &isActive}
        body, _ := json.Marshal(payload)

		// testUserID é o admin fazendo a ação em si mesmo
        adminUser := models.User{ID: testUserID, OrganizationID: testOrgID, Name: "Admin User", Role: models.RoleAdmin, IsActive: true}
        rowsAdmin := sqlmock.NewRows([]string{"id", "organization_id", "name", "role", "is_active"}).
            AddRow(adminUser.ID, adminUser.OrganizationID, adminUser.Name, adminUser.Role, adminUser.IsActive)

        sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 AND organization_id = $2 ORDER BY "users"."id" LIMIT $3`)).
            WithArgs(testUserID, testOrgID, 1).
            WillReturnRows(rowsAdmin)

        // Mock para a contagem de admins/managers ativos
        sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "users" WHERE organization_id = $1 AND is_active = $2 AND (role = $3 OR role = $4)`)).
            WithArgs(testOrgID, true, models.RoleAdmin, models.RoleManager).
            WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1)) // Só existe 1 admin/manager ativo

        req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/orgs/%s/users/%s/status", testOrgID.String(), testUserID.String()), bytes.NewBuffer(body))
        req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

        assert.Equal(t, http.StatusForbidden, rr.Code)
        assert.Contains(t, rr.Body.String(), "Não é possível desativar o último administrador/gerente ativo")
        assert.NoError(t, sqlMock.ExpectationsWereMet())
    })

	t.Run("UpdateUserStatusHandler - User not found", func(t *testing.T) {
		isActive := true
        payload := UpdateUserStatusPayload{IsActive: &isActive}
        body, _ := json.Marshal(payload)
        nonExistentUserID := uuid.New()

        sqlMock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE id = $1 AND organization_id = $2`)).
            WithArgs(nonExistentUserID, testOrgID).
            WillReturnError(gorm.ErrRecordNotFound)

        req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/orgs/%s/users/%s/status", testOrgID.String(), nonExistentUserID.String()), bytes.NewBuffer(body))
        req.Header.Set("Content-Type", "application/json")
        rr := httptest.NewRecorder()
        router.ServeHTTP(rr, req)
        assert.Equal(t, http.StatusNotFound, rr.Code)
        assert.NoError(t, sqlMock.ExpectationsWereMet())
    })

	t.Run("UpdateUserStatusHandler - Invalid payload (is_active missing)", func(t *testing.T) {
		// Enviar JSON vazio, o binding:"required" para IsActive deve falhar
        body := bytes.NewBuffer([]byte(`{}`))

        req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/orgs/%s/users/%s/status", testOrgID.String(), testTargetUserID.String()), body)
        req.Header.Set("Content-Type", "application/json")
        rr := httptest.NewRecorder()
        router.ServeHTTP(rr, req)
        assert.Equal(t, http.StatusBadRequest, rr.Code)
        assert.Contains(t, rr.Body.String(), "Payload inválido")
	})
    // TODO: Testar prevenção de bloqueio ao tentar desativar *outro* admin/manager se este for o último ativo.
}
