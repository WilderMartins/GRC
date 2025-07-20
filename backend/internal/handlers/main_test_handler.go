package handlers

import (
	"database/sql"
	"log"
	"os"
	"phoenixgrc/backend/internal/auth"
	"phoenixgrc/backend/internal/database" // Import for database.DB
	"phoenixgrc/backend/internal/models"
	"regexp" // For sqlmock query matching
	"testing"
	"time"
	"database/sql/driver" // Adicionado para driver.Value

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var mockDB *gorm.DB
var sqlMock sqlmock.Sqlmock

// setupMockDB inicializa sqlMock e mockDB para um teste específico.
func setupMockDB(t *testing.T) {
	// t.Helper() // Helper para não reportar esta função na stack de erro do teste
	var db *sql.DB
	var err error
	// Usar QueryMatcherRegexp para que ExpectQuery com regexp.QuoteMeta funcione de forma mais confiável.
	db, sqlMock, err = sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	// O defer db.Close() aqui fecharia o db após cada teste.
	// Se você quiser que o db mockado persista entre os sub-testes de uma função TestXxx,
	// o defer db.Close() deve estar na função TestXxx que chama setupMockDB.
	// Por enquanto, vamos omitir o defer aqui para simplicidade, mas pode ser necessário.

	dialector := postgres.New(postgres.Config{
		Conn:       db,
		PreferSimpleProtocol: true, // Para compatibilidade com sqlmock
	})
	gormDB, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to open GORM with mock: %v", err)
	}
	mockDB = gormDB       // Atribui à variável global do pacote de teste
	database.DB = mockDB // Configura o DB global usado pelos handlers
}


// TestMain sets up the test environment for handlers.
// Agora foca principalmente no JWT e modo Gin, já que mockDB é por teste.
func TestMain(m *testing.M) {
	log.Println("--- TestMain in main_test_handler.go START ---")
	gin.SetMode(gin.TestMode)

	// Setup JWT - Ainda pode ser global para o pacote de teste
	os.Setenv("JWT_SECRET_KEY", "handler_test_secret_key")
	os.Setenv("JWT_TOKEN_LIFESPAN_HOURS", "1")
	if err := auth.InitializeJWT(); err != nil {
		log.Fatalf("Failed to initialize JWT for handler testing: %v", err)
	}

	// Run tests
	exitVal := m.Run()

	log.Println("--- TestMain in main_test_handler.go END ---") // Log adicionado
	os.Unsetenv("JWT_SECRET_KEY")
	os.Unsetenv("JWT_TOKEN_LIFESPAN_HOURS")
	os.Exit(exitVal)
}

// Helper function to get a Gin engine with context prepared for authenticated requests
func getRouterWithAuthContext(userID uuid.UUID, orgID uuid.UUID, userRole models.UserRole) *gin.Engine {
	r := gin.Default()
	// Middleware to inject user/org IDs and role into context, simulating AuthMiddleware
	r.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Set("organizationID", orgID)
		c.Set("userRole", userRole)
		// Add other claims if your handlers use them
		c.Next()
	})
	return r
}

// escapeSQL takes a SQL string and escapes it for use with sqlmock
func escapeSQL(sql string) string {
	return regexp.QuoteMeta(sql)
}

// Common mock data
var testOrgID = uuid.New()
var testUserID = uuid.New()
var testRiskID = uuid.New()

func anyArg() sqlmock.Argument { return sqlmock.AnyArg() }
func anyTimeArg() sqlmock.Argument { return anyArgOfType("time.Time") } // sqlmock.AnyArg() works for time too usually

// anyArgOfType is a helper for types that sqlmock.AnyArg might not directly cover
// or when you want to be more explicit. For time.Time, GORM often handles it,
// but this is an example.
type anyArgOfType string
func (a anyArgOfType) Match(v driver.Value) bool { // Corrigido para driver.Value
	// For time.Time, you might compare types or just return true if AnyArg isn't enough.
	// This is a placeholder; for most cases, sqlmock.AnyArg() is sufficient.
	// If specific time matching is needed, you'd implement it here.
	// For this example, we'll assume sqlmock.AnyArg covers time.Time well.
	_, ok := v.(time.Time)
	return ok
}
