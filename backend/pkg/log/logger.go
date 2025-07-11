package log

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// L é o logger global estruturado (zap.Logger). Use para logging de alta performance.
	L *zap.Logger
	// S é o logger global sugarizado (zap.SugaredLogger). Use para conveniência (printf-style logging).
	S *zap.SugaredLogger
)

// Init inicializa os loggers globais L e S.
// logLevel pode ser "debug", "info", "warn", "error", "dpanic", "panic", "fatal".
// env pode ser "development" ou "production" (ou qualquer outra string para default para production).
func Init(logLevel string, env string) {
	var cfg zap.Config
	if strings.ToLower(env) == "development" {
		cfg = zap.NewDevelopmentConfig()
		// Cores para desenvolvimento podem ser habilitadas aqui se desejado,
		// mas NewDevelopmentConfig já tem um encoder de console amigável.
		// cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		cfg = zap.NewProductionConfig()
	}

	// Parse e define o nível de log
	level, err := zapcore.ParseLevel(strings.ToLower(logLevel))
	if err != nil {
		// Default para info se o nível for inválido
		level = zapcore.InfoLevel
		// Logar um aviso sobre o nível inválido usando um logger temporário ou o logger padrão do Go.
		// Não podemos usar nosso logger L/S aqui pois ainda não está inicializado.
		zap.L().Warn("Nível de log inválido fornecido, usando 'info' como padrão.", zap.String("invalid_level", logLevel))
	}
	cfg.Level = zap.NewAtomicLevelAt(level)

	// Construir o logger
	logger, err := cfg.Build(zap.AddCallerSkip(1)) // zap.AddCallerSkip(1) para que o caller seja o local da chamada a L.Info, etc.
	if err != nil {
		// Em caso de falha ao construir o logger, usar o logger padrão do Go para logar o erro
		// e entrar em pânico, pois logging é fundamental.
		// Ou, poderia tentar um fallback para um logger Nop ou um logger básico.
		panic(fmt.Sprintf("Falha ao construir o logger zap: %v", err))
	}

	L = logger
	S = logger.Sugar()

	// Substituir o logger global do zap para que possa ser acessado via zap.L() e zap.S() em outros pacotes.
	// Isso é opcional se você sempre importar seu pacote log e usar log.L ou log.S.
	// No entanto, é uma prática comum para conveniência.
	zap.ReplaceGlobals(L)
	// Se quiser substituir o logger padrão do Go também (para que `log.Print` use zap):
	// zap.RedirectStdLog(L) // Cuidado com loops se o próprio zap logar para o std log em algum momento.
}

// init é chamado quando o pacote é importado pela primeira vez.
// Configura um logger padrão inicial. Pode ser re-inicializado explicitamente em main.go.
func init() {
	// Valores padrão podem vir de variáveis de ambiente
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info" // Default log level
	}
	appEnv := os.Getenv("APP_ENV") // Ou GIN_MODE, se preferir
	if appEnv == "" {
		appEnv = "development" // Default para ambiente de desenvolvimento
	}

	// Inicialização inicial do logger.
	// Precisamos de um fmt.Sprintf para o panic dentro de Init se ele ocorrer ANTES de L ser setado.
	// Para evitar isso, podemos ter uma lógica de inicialização mais simples aqui
	// ou garantir que o panic use o logger padrão do Go.
	// A função Init como está agora, lida com o panic usando fmt.Sprintf.
	Init(logLevel, appEnv)
	L.Info("Logger global zap inicializado na importação do pacote.", zap.String("initial_level", logLevel), zap.String("initial_env", appEnv))
}

// Helper para adicionar fmt.Sprintf ao panic dentro de Init
// (já está lá, mas como uma nota mental).
// Este comentário pode ser removido.
var _ = fmt.Sprintf // Dummy para manter o import de fmt se o panic for a única utilização.
