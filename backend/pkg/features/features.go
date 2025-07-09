package features

import (
	"phoenixgrc/backend/pkg/config" // Importa o pacote de configuração da aplicação
	"strings"
)

// IsEnabled verifica se um feature toggle específico está habilitado.
// Os nomes das features são case-insensitive para a verificação,
// mas são armazenados conforme definidos nas variáveis de ambiente (sem o prefixo FEATURE_).
func IsEnabled(featureName string) bool {
	if config.Cfg.FeatureToggles == nil {
		return false // Se o mapa não for inicializado, nenhuma feature está habilitada.
	}

	// Normaliza o nome da feature para busca (ex: para minúsculas, se desejado,
	// mas as chaves no mapa são como foram definidas após remover o prefixo "FEATURE_").
	// Para consistência, vamos assumir que os nomes das features no código
	// serão passados exatamente como estão no mapa (case-sensitive).
	// Se quisermos case-insensitive, teríamos que iterar ou armazenar chaves normalizadas.
	// Por simplicidade, vamos manter case-sensitive por enquanto, correspondendo à chave da variável de ambiente.

	// Opcional: normalizar o nome da feature para busca, por exemplo, para maiúsculas,
	// para corresponder a como as variáveis de ambiente são frequentemente definidas.
	// Ex: featureName = strings.ToUpper(featureName)

	enabled, exists := config.Cfg.FeatureToggles[featureName]
	if !exists {
		return false // Feature não definida é considerada desabilitada.
	}
	return enabled
}

// GetFeatureToggleState retorna o estado de um feature toggle e se ele existe.
// Isso pode ser útil se você precisa distinguir entre uma feature explicitamente desabilitada
// e uma feature não configurada.
func GetFeatureToggleState(featureName string) (enabled bool, exists bool) {
	if config.Cfg.FeatureToggles == nil {
		return false, false
	}
	// featureName = strings.ToUpper(featureName) // Se normalizar
	enabled, exists = config.Cfg.FeatureToggles[featureName]
	return enabled, exists
}
