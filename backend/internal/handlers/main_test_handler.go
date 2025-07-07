package handlers

import (
	"database/sql"
	"log"
	"os"
	"phoenixgrc/backend/internal/auth"
	"phoenixgrc/backend/internal/database" // Import for database.DB
	"regexp" // For sqlmock query matching
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var mockDB *gorm.DB
var sqlMock sqlmock.Sqlmock

// TestMain sets up the test environment for handlers.
// It initializes a mock database and JWT.
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	// Setup mock DB
	var err error
	var db *sql.DB
	db, sqlMock, err = sqlmock.New()
	if err != nil {
		log.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	// Use a GORM dialector that uses the sqlmock connection
	dialector := postgres.New(postgres.Config{
		Conn: db,
		// PreferSimpleProtocol: true, // Avoids implicit prepared statements that can complicate mocking
	})

	mockDB, err = gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Or logger.Info for debugging
	})
	if err != nil {
		log.Fatalf("Failed to open GORM with mock: %v", err)
	}
	database.DB = mockDB // Override the global DB instance with the mock

	// Setup JWT
	os.Setenv("JWT_SECRET_KEY", "handler_test_secret_key")
	os.Setenv("JWT_TOKEN_LIFESPAN_HOURS", "1")
	if err := auth.InitializeJWT(); err != nil {
		log.Fatalf("Failed to initialize JWT for handler testing: %v", err)
	}

	// Run tests
	exitVal := m.Run()

	os.Unsetenv("JWT_SECRET_KEY")
	os.Unsetenv("JWT_TOKEN_LIFESPAN_HOURS")
	os.Exit(exitVal)
}

// Helper function to get a Gin engine with context prepared for authenticated requests
func getRouterWithAuthenticatedContext(userID uuid.UUID, orgID uuid.UUID) *gin.Engine {
	r := gin.Default()
	// Middleware to inject user/org IDs into context, simulating AuthMiddleware
	r.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Set("organizationID", orgID)
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
func (a anyArgOfType) Match(v interface{}) bool {
	// For time.Time, you might compare types or just return true if AnyArg isn't enough.
	// This is a placeholder; for most cases, sqlmock.AnyArg() is sufficient.
	// If specific time matching is needed, you'd implement it here.
	// For this example, we'll assume sqlmock.AnyArg covers time.Time well.
	_, ok := v.(time.Time)
	return ok
}
