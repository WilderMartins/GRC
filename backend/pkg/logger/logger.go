package logger

import (
	"log"
	"os"
	"phoenixgrc/backend/pkg/config" // Importar para acessar Cfg.Environment

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

func init() {
	var err error
	var zapConfig zap.Config

	// Determinar o ambiente para configurar o logger apropriadamente
	env := config.Cfg.Environment // Acessa a config já carregada

	if env == "production" {
		zapConfig = zap.NewProductionConfig()
		zapConfig.EncoderConfig.TimeKey = "timestamp"
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		zapConfig.EncoderConfig.MessageKey = "message"
		// Adicionar mais campos se necessário, como "service", "version"
		// zapConfig.InitialFields = map[string]interface{}{
		// 	"service": "phoenix-grc-backend",
		// }
	} else { // development ou qualquer outro
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // Níveis coloridos para dev
		zapConfig.EncoderConfig.TimeKey = "T"                                  // Chave de tempo mais curta para dev
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// Definir o nível de log (pode vir de config também)
	// zapConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel) // Exemplo

	Log, err = zapConfig.Build()
	if err != nil {
		// Fallback para o logger padrão do Go se o Zap falhar ao inicializar
		log.Printf("Falha ao inicializar o logger Zap: %v. Usando logger padrão do Go.", err)
		log.SetOutput(os.Stderr)
		log.SetPrefix("ERROR: ")
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
		// Não podemos usar Log.Fatal aqui, pois Log não seria um *zap.Logger
		// Em vez disso, usamos o log padrão para sair se for um erro crítico de inicialização.
		// No entanto, para a inicialização do logger, é melhor apenas logar o erro e continuar
		// com um logger de fallback, ou fazer o programa entrar em pânico se o logger for essencial.
		// Para este caso, vamos usar um logger de fallback simples.
		// Esta parte do fallback é complexa porque Log é *zap.Logger.
		// Uma abordagem mais simples seria:
		// if err != nil { panic(err) }
		// Ou usar um logger global que é uma interface e pode ter diferentes implementações.

		// Simplificando: se o Zap falhar, use o logger padrão do Go e não atribua a `Log`
		// Os chamadores de `logger.Log` precisariam verificar se é nil, o que não é ideal.
		// A melhor prática é garantir que `Log` seja sempre um logger válido.
		// Se o Zap falhar, o programa deve provavelmente entrar em pânico, pois o logging é fundamental.
		panic(err)
	}

	Log.Info("Logger Zap inicializado.", zap.String("ambiente", env))

	// Redirecionar o log padrão do Go para o Zap (opcional, mas útil para dependências que usam `log`)
	// zap.RedirectStdLog(Log) // Cuidado: isso pode ter implicações de performance ou formatação.
}

// Funções helper para acesso fácil (opcional)
func Info(message string, fields ...zap.Field) {
	Log.Info(message, fields...)
}

func Debug(message string, fields ...zap.Field) {
	Log.Debug(message, fields...)
}

func Warn(message string, fields ...zap.Field) {
	Log.Warn(message, fields...)
}

func Error(message string, fields ...zap.Field) {
	Log.Error(message, fields...)
}

func Fatal(message string, fields ...zap.Field) {
	// Log.Fatal também chama os.Exit(1)
	Log.Fatal(message, fields...)
}

// Sync chama Log.Sync(). Útil para garantir que logs em buffer sejam escritos antes de sair.
// Chamar no defer main() por exemplo.
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
