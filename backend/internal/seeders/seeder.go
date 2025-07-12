package seeders

import (
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	phxlog "phoenixgrc/backend/pkg/log"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RunMigrations executa as migrações do GORM para todos os modelos.
func RunMigrations(db *gorm.DB) error {
	log := phxlog.L.Named("RunMigrations")
	log.Info("Auto-migrating database schema...")

	// Adicione todos os seus modelos aqui para que o GORM possa criar/atualizar suas tabelas.
	err := db.AutoMigrate(
		&models.Organization{},
		&models.User{},
		&models.Risk{},
		&models.RiskStakeholder{},
		&models.ApprovalWorkflow{},
		&models.Vulnerability{},
		&models.AuditFramework{},
		&models.AuditControl{},
		&models.AuditAssessment{},
		&models.IdentityProvider{},
		&models.Webhook{},
		&models.C2M2Domain{},
		&models.C2M2Practice{},
	)

	if err != nil {
		log.Error("GORM AutoMigrate failed", zap.Error(err))
		return err
	}

	log.Info("Database schema migration completed successfully.")
	return nil
}

// SeedInitialData popula o banco de dados com dados iniciais essenciais.
func SeedInitialData(db *gorm.DB) error {
	log := phxlog.L.Named("SeedInitialData")
	log.Info("Seeding initial data...")

	// Cada função de seeder deve verificar se os dados já existem antes de inserir.
	if err := SeedAuditFrameworksAndControls(db); err != nil {
		log.Error("Failed to seed audit frameworks and controls", zap.Error(err))
		return err
	}

	if err := SeedC2M2Data(db); err != nil {
		log.Error("Failed to seed C2M2 data", zap.Error(err))
		return err
	}

	log.Info("Initial data seeding completed successfully.")
	return nil
}

// FullSetup é uma função de conveniência que executa migrações e seeding.
// Útil para o comando de setup CLI.
func FullSetup(db *gorm.DB) error {
	if err := RunMigrations(db); err != nil {
		return err
	}
	if err := SeedInitialData(db); err != nil {
		return err
	}
	return nil
}
