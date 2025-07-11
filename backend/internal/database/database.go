package database

import (
	"fmt"
	"log"
	"os"
	"phoenixgrc/backend/internal/models"

	"github.com/golang-migrate/migrate/v4"
	postgresdriver "github.com/golang-migrate/migrate/v4/database/postgres" // Renomeado para evitar conflito com gorm/driver/postgres
	_ "github.com/golang-migrate/migrate/v4/source/file"                   // Importar driver source file
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// ConnectDB initializes the database connection and runs migrations.
func ConnectDB(dsn string) error {
	var err error
	logLevel := logger.Silent
	if os.Getenv("APP_ENV") == "development" {
		logLevel = logger.Info
	}

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connection established.")
	return nil
}

// RunPhoenixMigrations aplica migrações SQL usando golang-migrate.
// dbURL deve ser a string de conexão completa para o banco de dados.
func RunPhoenixMigrations(dbURL string, gormInstance *gorm.DB) error {
	if gormInstance == nil {
		return fmt.Errorf("GORM DB instance is nil")
	}
	sqlDB, err := gormInstance.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// O path para as migrações deve ser relativo ao diretório de execução do binário.
	// Se o binário está na raiz do projeto 'backend/', o path é correto.
	// Se estiver em 'backend/cmd/server/', o path precisa ser '../internal/database/migrations'
	// Para robustez, considere usar um path absoluto ou configurável.
	// Por agora, assumimos que o binário é executado da raiz do projeto 'backend/'.
	sourceURL := "file://internal/database/migrations"
	log.Printf("Attempting to run migrations from source: %s", sourceURL)

	driver, err := postgresdriver.WithInstance(sqlDB, &postgresdriver.Config{})
	if err != nil {
		return fmt.Errorf("could not create postgres driver for migrate: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		sourceURL,
		"postgres", // Nome do banco de dados para o driver (pode ser qualquer string, mas "postgres" é comum)
		driver,
	)
	if err != nil {
		// Tentar um path relativo comum se o primeiro falhar (ex: se rodando de cmd/server)
		// Esta é uma tentativa de fallback, idealmente o path é configurado corretamente.
		log.Printf("Failed to initialize migrate with source '%s': %v. Trying alternative path '../internal/database/migrations'", sourceURL, err)
		sourceURL = "file://../internal/database/migrations"
		m, err = migrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
		if err != nil {
			return fmt.Errorf("failed to initialize migrate with source '%s' and alternative path: %w", sourceURL, err)
		}
	}

	log.Println("Applying database migrations...")
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	// Logar versão atual
	version, dirty, err := m.Version()
	if err != nil {
		log.Printf("Warning: could not get migration version after applying: %v", err)
	} else {
		log.Printf("Database migration applied. Current version: %d, Dirty: %t", version, dirty)
	}

	if err == migrate.ErrNoChange {
		log.Println("No new database migrations to apply.")
		return nil
	}

	log.Println("Database migrations applied successfully.")
	return nil
}


// MigrateDB agora usa golang-migrate em vez de GORM auto-migration.
// A dbURL é necessária para o golang-migrate.
func MigrateDB(dbURL string) error {
	if DB == nil {
		return fmt.Errorf("database connection is not initialized. Call ConnectDB first")
	}
	log.Println("Running database migrations via golang-migrate...")

	err := RunPhoenixMigrations(dbURL, DB)
	if err != nil {
		return fmt.Errorf("golang-migrate migration failed: %w", err)
	}

	// A chamada AutoMigrate original foi comentada/removida.
	// As migrações agora são gerenciadas exclusivamente por arquivos SQL.
	// err = DB.AutoMigrate(
	// 	&models.Organization{},
	// 	&models.User{},
	// 	&models.Risk{},
	// 	&models.Vulnerability{},
	// 	&models.RiskStakeholder{},
	// 	&models.ApprovalWorkflow{},
	// 	&models.AuditFramework{},
	// 	&models.AuditControl{},
	// 	&models.AuditAssessment{},
	// 	&models.IdentityProvider{},
	// 	&models.WebhookConfiguration{},
	// )
	// if err != nil {
	// 	return fmt.Errorf("database migration failed: %w", err)
	// }
	log.Println("golang-migrate database migrations process completed.")
	return nil
}

// GetDB returns the current database instance.
func GetDB() *gorm.DB {
	return DB
}
