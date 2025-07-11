package database

import (
	"fmt"
	"os"
	"phoenixgrc/backend/internal/models" // Mantido para referência futura, não usado diretamente aqui
	phxlog "phoenixgrc/backend/pkg/log"  // Importar o logger zap

	"github.com/golang-migrate/migrate/v4"
	postgresdriver "github.com/golang-migrate/migrate/v4/database/postgres" // Renomeado para evitar conflito com gorm/driver/postgres
	_ "github.com/golang-migrate/migrate/v4/source/file"                   // Importar driver source file
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// ConnectDB initializes the database connection.
func ConnectDB(dsn string) error {
	var err error
	gormLogLevel := logger.Silent
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" { // Fallback para GIN_MODE se APP_ENV não estiver setado
		appEnv = os.Getenv("GIN_MODE")
	}

	if appEnv == "development" || appEnv == "debug" {
		gormLogLevel = logger.Info
	}

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(gormLogLevel),
	})

	if err != nil {
		// O chamador (main.go) fará o log fatal com zap.
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	phxlog.L.Info("Database connection established.")
	return nil
}

// RunPhoenixMigrations aplica migrações SQL usando golang-migrate.
func RunPhoenixMigrations(dbURL string, gormInstance *gorm.DB) error {
	if gormInstance == nil {
		return fmt.Errorf("GORM DB instance is nil")
	}
	sqlDB, err := gormInstance.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sourceURL := "file://internal/database/migrations"
	phxlog.L.Info("Attempting to run migrations", zap.String("sourceURL", sourceURL))

	driver, err := postgresdriver.WithInstance(sqlDB, &postgresdriver.Config{})
	if err != nil {
		return fmt.Errorf("could not create postgres driver for migrate: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
	if err != nil {
		phxlog.L.Warn("Failed to initialize migrate with primary source, trying alternative",
			zap.String("source", sourceURL),
			zap.Error(err),
			zap.String("alternative_source", "file://../internal/database/migrations")) // Para quando rodado de cmd/server/
		sourceURL = "file://../internal/database/migrations"
		m, err = migrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
		if err != nil {
			return fmt.Errorf("failed to initialize migrate with source '%s' and alternative path: %w", sourceURL, err)
		}
	}

	phxlog.L.Info("Applying database migrations...")
	errUp := m.Up() // Armazenar o erro de m.Up()
	if errUp != nil && errUp != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", errUp)
	}

	version, dirty, errVersion := m.Version()
	if errVersion != nil {
		phxlog.L.Warn("Could not get migration version after applying", zap.Error(errVersion))
	} else {
		phxlog.L.Info("Database migration status", zap.Uint("version", version), zap.Bool("dirty", dirty))
	}

	if errUp == migrate.ErrNoChange {
		phxlog.L.Info("No new database migrations to apply.")
		return nil
	}

	phxlog.L.Info("Database migrations applied successfully.")
	return nil
}

// MigrateDB agora usa golang-migrate em vez de GORM auto-migration.
func MigrateDB(dbURL string) error {
	if DB == nil {
		return fmt.Errorf("database connection is not initialized. Call ConnectDB first")
	}
	phxlog.L.Info("Running database migrations via golang-migrate...")

	err := RunPhoenixMigrations(dbURL, DB)
	if err != nil {
		// O erro de RunPhoenixMigrations já é bem formatado e logado internamente se necessário.
		// O chamador (setup/main.go) fará o log fatal se este erro for retornado.
		return fmt.Errorf("golang-migrate migration process failed: %w", err)
	}

	// AutoMigrate do GORM foi removido.
	phxlog.L.Info("golang-migrate database migrations process completed.")
	return nil
}

// GetDB returns the current database instance.
func GetDB() *gorm.DB {
	return DB
}
