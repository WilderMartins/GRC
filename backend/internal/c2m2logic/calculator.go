package c2m2logic

import (
	"fmt"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CalculateAndUpdateMaturityLevel calcula o nível de maturidade C2M2 para uma avaliação
// e atualiza o registro no banco de dados.
func CalculateAndUpdateMaturityLevel(assessmentID uuid.UUID, db *gorm.DB) (int, error) {
	if db == nil {
		db = database.GetDB()
	}

	// 1. Obter todas as práticas C2M2
	var allPractices []models.C2M2Practice
	if err := db.Find(&allPractices).Error; err != nil {
		return 0, fmt.Errorf("failed to fetch all C2M2 practices: %w", err)
	}

	// 2. Obter as avaliações de práticas para este assessment
	var evaluations []models.C2M2PracticeEvaluation
	if err := db.Where("audit_assessment_id = ?", assessmentID).Find(&evaluations).Error; err != nil {
		return 0, fmt.Errorf("failed to fetch C2M2 practice evaluations for assessment %s: %w", assessmentID, err)
	}

	// Mapear avaliações por ID da prática para fácil acesso
	evaluationsMap := make(map[uuid.UUID]models.PracticeStatus)
	for _, e := range evaluations {
		evaluationsMap[e.PracticeID] = e.Status
	}

	// 3. Calcular o MIL alcançado
	achievedMIL := 0
	for mil := 1; mil <= 3; mil++ {
		allPracticesForMILMet := true
		for _, p := range allPractices {
			if p.TargetMIL == mil {
				status, exists := evaluationsMap[p.ID]
				if !exists || status != models.PracticeStatusFullyImplemented {
					allPracticesForMILMet = false
					break
				}
			}
		}

		if allPracticesForMILMet {
			achievedMIL = mil
		} else {
			break // Não pode alcançar um MIL maior se o atual não foi alcançado
		}
	}

	// 4. Atualizar o assessment no banco de dados
	err := db.Model(&models.AuditAssessment{}).
		Where("id = ?", assessmentID).
		Updates(map[string]interface{}{
			"c2m2_maturity_level": achievedMIL,
			"updated_at":          time.Now(),
		}).Error

	if err != nil {
		return 0, fmt.Errorf("failed to update audit assessment with calculated MIL: %w", err)
	}

	return achievedMIL, nil
}
