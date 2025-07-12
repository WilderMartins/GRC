package c2m2logic

import (
	"fmt"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	phxlog "phoenixgrc/backend/pkg/log"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CalculateAndUpdateMaturityLevel calcula e atualiza o nível de maturidade C2M2 para uma avaliação.
// Retorna o nível calculado e um erro, se houver.
func CalculateAndUpdateMaturityLevel(assessmentID uuid.UUID, db *gorm.DB) (int, error) {
	phxlog.L.Info("Starting C2M2 maturity level calculation", zap.String("assessmentID", assessmentID.String()))

	// 1. Buscar todas as práticas C2M2 e mapeá-las por TargetMIL
	var allPractices []models.C2M2Practice
	if err := db.Find(&allPractices).Error; err != nil {
		phxlog.L.Error("Failed to fetch all C2M2 practices for calculation", zap.Error(err))
		return -1, fmt.Errorf("failed to fetch C2M2 practices: %w", err)
	}

	practicesByMIL := make(map[int][]uuid.UUID)
	for _, p := range allPractices {
		practicesByMIL[p.TargetMIL] = append(practicesByMIL[p.TargetMIL], p.ID)
	}

	// 2. Buscar todas as avaliações de práticas para este assessment
	var practiceEvaluations []models.C2M2PracticeEvaluation
	if err := db.Where("audit_assessment_id = ?", assessmentID).Find(&practiceEvaluations).Error; err != nil {
		phxlog.L.Error("Failed to fetch C2M2 practice evaluations for assessment",
			zap.String("assessmentID", assessmentID.String()), zap.Error(err))
		return -1, fmt.Errorf("failed to fetch practice evaluations: %w", err)
	}

	// Criar um mapa de práticas avaliadas para fácil lookup
	evaluatedPractices := make(map[uuid.UUID]string)
	for _, eval := range practiceEvaluations {
		evaluatedPractices[eval.PracticeID] = eval.Status
	}

	// 3. Lógica de Cálculo de MIL
	achievedMIL := 0
	for mil := 1; mil <= 3; mil++ {
		practicesForMIL, ok := practicesByMIL[mil]
		if !ok || len(practicesForMIL) == 0 {
			// Não há práticas para este MIL, então não pode ser alcançado.
			// Ou, se for o caso, pode-se considerar que foi alcançado se o nível anterior foi.
			// A lógica C2M2 implica que se não há práticas, não há nada a fazer para este nível.
			// Vamos assumir que se o nível anterior foi alcançado, este também é se não houver práticas.
			// UPDATE: A regra correta é que para alcançar um MIL, TODAS as práticas daquele MIL devem ser "fully_implemented".
			// Se não houver práticas para um MIL, a condição é trivialmente verdadeira, mas isso não faz sentido.
			// Vamos assumir que um framework C2M2 completo tem práticas para todos os MILs.
			// Se não houver práticas para um MIL, ele não pode ser alcançado.
			break
		}

		allPracticesForMILMet := true
		for _, practiceID := range practicesForMIL {
			status, found := evaluatedPractices[practiceID]
			if !found || status != models.PracticeStatusFullyImplemented {
				allPracticesForMILMet = false
				break // Uma prática não atendida é suficiente para falhar o nível
			}
		}

		if allPracticesForMILMet {
			achievedMIL = mil // Nível alcançado
		} else {
			break // Não pode prosseguir para o próximo nível
		}
	}

	// 4. Salvar o resultado na AuditAssessment
	phxlog.L.Info("C2M2 calculation complete",
		zap.String("assessmentID", assessmentID.String()),
		zap.Int("calculatedMIL", achievedMIL))

	result := db.Model(&models.AuditAssessment{}).Where("id = ?", assessmentID).Update("c2m2_maturity_level", achievedMIL)
	if result.Error != nil {
		phxlog.L.Error("Failed to update C2M2 maturity level in database",
			zap.String("assessmentID", assessmentID.String()),
			zap.Int("calculatedMIL", achievedMIL),
			zap.Error(result.Error))
		return -1, fmt.Errorf("failed to save calculated maturity level: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		phxlog.L.Warn("C2M2 maturity level was calculated but no assessment record was updated.",
			zap.String("assessmentID", assessmentID.String()))
		// Não é necessariamente um erro fatal, o assessment pode ter sido deletado.
	}

	return achievedMIL, nil
}
