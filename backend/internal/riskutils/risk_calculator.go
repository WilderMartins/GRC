package riskutils

import "phoenixgrc/backend/internal/models"

// CalculateRiskLevel calcula o nível de risco com base no impacto e probabilidade.
func CalculateRiskLevel(impact models.RiskImpact, probability models.RiskProbability) string {
	// Mapear impacto e probabilidade para valores numéricos para facilitar a lógica da matriz
	impactValue := mapImpactToValue(impact)
	probabilityValue := mapProbabilityToValue(probability)

	if impactValue == 0 || probabilityValue == 0 {
		return models.RiskLevelUndefined // Se algum valor for inválido/não mapeado
	}

	// Matriz de Risco 4x4 (Probabilidade x Impacto)
	// Linhas: Probabilidade (1=Baixo, 2=Médio, 3=Alto, 4=Crítico)
	// Colunas: Impacto (1=Baixo, 2=Médio, 3=Alto, 4=Crítico)

	// Probabilidade Baixa
	if probabilityValue == 1 { // Baixa
		if impactValue <= 2 { // Baixo, Médio
			return models.RiskLevelLow
		}
		if impactValue == 3 { // Alto
			return models.RiskLevelModerate
		}
		if impactValue == 4 { // Crítico
			return models.RiskLevelHigh
		}
	}

	// Probabilidade Média
	if probabilityValue == 2 { // Média
		if impactValue == 1 { // Baixo
			return models.RiskLevelLow
		}
		if impactValue == 2 { // Médio
			return models.RiskLevelModerate
		}
		if impactValue >= 3 { // Alto, Crítico
			return models.RiskLevelHigh
		}
	}

	// Probabilidade Alta
	if probabilityValue == 3 { // Alta
		if impactValue == 1 { // Baixo
			return models.RiskLevelModerate
		}
		if impactValue <= 3 { // Médio, Alto
			return models.RiskLevelHigh
		}
		if impactValue == 4 { // Crítico
			return models.RiskLevelExtreme
		}
	}

	// Probabilidade Crítica
	if probabilityValue == 4 { // Crítica
		if impactValue <= 1 { // Baixo (considerando que poderia haver mais granularidade)
			return models.RiskLevelModerate
		}
		if impactValue <= 2 { // Médio
			return models.RiskLevelHigh
		}
		if impactValue >=3 { // Alto, Crítico
			return models.RiskLevelExtreme
		}
	}

	return models.RiskLevelUndefined // Default para combinações não cobertas (não deveria acontecer)
}

func mapImpactToValue(impact models.RiskImpact) int {
	switch impact {
	case models.ImpactLow:
		return 1
	case models.ImpactMedium:
		return 2
	case models.ImpactHigh:
		return 3
	case models.ImpactCritical:
		return 4
	default:
		return 0 // Valor inválido ou não mapeado
	}
}

func mapProbabilityToValue(probability models.RiskProbability) int {
	switch probability {
	case models.ProbabilityLow:
		return 1
	case models.ProbabilityMedium:
		return 2
	case models.ProbabilityHigh:
		return 3
	case models.ProbabilityCritical:
		return 4
	default:
		return 0 // Valor inválido ou não mapeado
	}
}
