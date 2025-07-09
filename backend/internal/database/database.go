package database

import (
	"fmt"
	"log"
	"os"
	"phoenixgrc/backend/internal/models"

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

// MigrateDB runs the GORM auto-migration for the defined models.
func MigrateDB() error {
	if DB == nil {
		return fmt.Errorf("database connection is not initialized. Call ConnectDB first")
	}
	log.Println("Running database migrations...")
	err := DB.AutoMigrate(
		&models.Organization{},
		&models.User{},
		&models.Risk{},
		&models.Vulnerability{},
		&models.RiskStakeholder{},
		&models.ApprovalWorkflow{},
		&models.AuditFramework{},
		&models.AuditControl{},
		&models.AuditAssessment{},
		&models.IdentityProvider{},     // Added
		&models.WebhookConfiguration{}, // Added
	)
	if err != nil {
		return fmt.Errorf("database migration failed: %w", err)
	}
	log.Println("Database migrations completed successfully.")
	return nil
}

// GetDB returns the current database instance.
func GetDB() *gorm.DB {
	return DB
}
