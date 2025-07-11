package config

import (
	"log"
	"os"
	"strconv"
	"strings" // Adicionado para HasPrefix e TrimPrefix
	"time"

	"github.com/joho/godotenv"
)

// AppConfig detém a configuração da aplicação.
type AppConfig struct {
	Port                string
	JWTSecret           string
	JWTTokenLifespan    time.Duration
	DBHost              string
	DBPort              string
	DBUser              string
	DBPassword          string
	DBName              string
	DBSchema            string
	EnableDBSSL         bool
	Environment         string // "development", "staging", "production"
	GCSProjectID        string
	GCSBucketName       string
	AWSRegion           string
	AWSSESEmailSender   string
	TOTPIssuerName      string
	AWSS3Bucket         string // Novo para S3
	FileStorageProvider                string // "gcs" ou "s3"
	FrontendBaseURL                    string // Adicionado para links em emails/notificações
	DefaultOrganizationIDForGlobalSSO string // UUID da organização padrão para novos usuários de SSO global
	FeatureToggles                     map[string]bool
	// Adicionar outras configurações aqui
}

var Cfg AppConfig

// LoadConfig carrega a configuração da aplicação de variáveis de ambiente.
func LoadConfig() {
	// Carregar .env para desenvolvimento local, ignorar erro se não existir (para produção)
	if err := godotenv.Load(); err != nil {
		log.Println("Aviso: Arquivo .env não encontrado ou erro ao carregar:", err)
	}

	Cfg.Port = getEnv("SERVER_PORT", "8080") // Mudado de PORT para SERVER_PORT para consistência com docker-compose
	Cfg.JWTSecret = getEnv("JWT_SECRET_KEY", "a_very_secure_secret_key_please_change_me_32_chars_long")
	jwtLifespanHours, err := strconv.Atoi(getEnv("JWT_TOKEN_LIFESPAN_HOURS", "24"))
	if err != nil {
		log.Printf("Aviso: JWT_TOKEN_LIFESPAN_HOURS inválido, usando default 24h. Erro: %v", err)
		jwtLifespanHours = 24
	}
	Cfg.JWTTokenLifespan = time.Duration(jwtLifespanHours) * time.Hour

	Cfg.DBHost = getEnv("POSTGRES_HOST", "db") // Consistente com docker-compose
	Cfg.DBPort = getEnv("POSTGRES_PORT", "5432")
	Cfg.DBUser = getEnv("POSTGRES_USER", "admin")
	Cfg.DBPassword = getEnv("POSTGRES_PASSWORD", "password123")
	Cfg.DBName = getEnv("POSTGRES_DB", "phoenix_grc_dev")
	Cfg.DBSchema = getEnv("DB_SCHEMA", "phoenix_grc") // Esquema padrão, verificar se é usado
	Cfg.EnableDBSSL = getEnvAsBool("POSTGRES_SSLMODE_ENABLE", false) // POSTGRES_SSLMODE é string, EnableDBSSL é bool

	Cfg.Environment = getEnv("GIN_MODE", "development") // GIN_MODE (debug/release) ou APP_ENV

	Cfg.GCSProjectID = getEnv("GCS_PROJECT_ID", "")
	Cfg.GCSBucketName = getEnv("GCS_BUCKET_NAME", "")

	Cfg.AWSRegion = getEnv("AWS_REGION", "")
	Cfg.AWSSESEmailSender = getEnv("AWS_SES_EMAIL_SENDER", "")
	Cfg.TOTPIssuerName = getEnv("TOTP_ISSUER_NAME", "PhoenixGRC")
	Cfg.AWSS3Bucket = getEnv("AWS_S3_BUCKET", "")
	Cfg.FileStorageProvider = strings.ToLower(getEnv("FILE_STORAGE_PROVIDER", "gcs")) // Default para GCS
	Cfg.FrontendBaseURL = getEnv("FRONTEND_BASE_URL", "http://localhost:3000")
	Cfg.DefaultOrganizationIDForGlobalSSO = getEnv("DEFAULT_ORGANIZATION_ID_FOR_GLOBAL_SSO", "")


	// Carregar Feature Toggles
	Cfg.FeatureToggles = make(map[string]bool)
	const featurePrefix = "FEATURE_"
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, featurePrefix) {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				featureName := strings.TrimPrefix(parts[0], featurePrefix)
				featureValue, err := strconv.ParseBool(parts[1])
				if err == nil {
					Cfg.FeatureToggles[featureName] = featureValue
					log.Printf("Feature Toggle carregado: %s = %t", featureName, featureValue)
				} else {
					log.Printf("Aviso: Valor inválido para Feature Toggle %s: %s (esperado true/false)", featureName, parts[1])
				}
			}
		}
	}

	log.Printf("Configuração carregada para o ambiente: %s", Cfg.Environment)
}

// getEnv retorna o valor de uma variável de ambiente ou um valor default.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("Variável de ambiente '%s' não definida, usando default: '%s'", key, defaultValue)
	return defaultValue
}

// getEnvAsBool retorna o valor booleano de uma variável de ambiente ou um valor default.
func getEnvAsBool(key string, defaultValue bool) bool {
	valStr := getEnv(key, "")
	if valStr == "" {
		return defaultValue
	}
	valBool, err := strconv.ParseBool(valStr)
	if err != nil {
		log.Printf("Aviso: Variável de ambiente booleana '%s' com valor inválido '%s', usando default: %t. Erro: %v", key, valStr, defaultValue, err)
		return defaultValue
	}
	return valBool
}

func init() {
	LoadConfig() // Carregar config automaticamente na inicialização do pacote
}
