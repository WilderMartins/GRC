package config

import (
	"log" // Manter log padrão para mensagens de bootstrap antes do logger zap ser configurado
	"os"
	"strconv"
	"strings" // Adicionado para HasPrefix e TrimPrefix
	"time"

	"github.com/joho/godotenv"
	// phxlog "phoenixgrc/backend/pkg/log" // Não importar aqui para evitar ciclo de importação
	// "go.uber.org/zap"                 // Não importar aqui
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
	Environment         string // "development", "staging", "production" (carregado de APP_ENV)
	AppVersion          string // Versão da aplicação (carregado de APP_VERSION)
	GCSProjectID        string
	GCSBucketName       string
	AWSRegion           string
	AWSSESEmailSender   string
	TOTPIssuerName      string
	AWSS3Bucket         string // Novo para S3
	FileStorageProvider string // "gcs" ou "s3"
	FrontendBaseURL     string // Adicionado para links em emails/notificações
	DefaultOrganizationIDForGlobalSSO string `mapstructure:"DEFAULT_ORGANIZATION_ID_FOR_GLOBAL_SSO"`
	AllowSAMLUserCreation             bool   `mapstructure:"ALLOW_SAML_USER_CREATION"` // Nova config para SAML
	GithubClientID                    string `mapstructure:"GITHUB_CLIENT_ID"`
	GithubClientSecret                string `mapstructure:"GITHUB_CLIENT_SECRET"`
	GoogleClientID                    string `mapstructure:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret                string `mapstructure:"GOOGLE_CLIENT_SECRET"`
	AllowGlobalSSOUserCreation        bool   `mapstructure:"ALLOW_GLOBAL_SSO_USER_CREATION"`
	FeatureToggles                    map[string]bool
	// Adicionar outras configurações aqui
}

var Cfg AppConfig

// LoadConfig carrega a configuração da aplicação de variáveis de ambiente.
func LoadConfig() {
	// Carregar .env para desenvolvimento local, ignorar erro se não existir (para produção)
	if err := godotenv.Load(); err != nil {
		// Usar phxlog aqui pode ser problemático se o logger ainda não estiver 100% configurado
		// ou se houver dependência cíclica. O log padrão do Go é seguro para esta fase inicial.
		log.Println("Warning: .env file not found or error loading it:", err)
	}

	Cfg.Port = getEnv("SERVER_PORT", "8080")
	Cfg.JWTSecret = getEnv("JWT_SECRET_KEY", "a_very_secure_secret_key_please_change_me_32_chars_long")
	jwtLifespanHoursStr := getEnv("JWT_TOKEN_LIFESPAN_HOURS", "24")
	jwtLifespanHours, err := strconv.Atoi(jwtLifespanHoursStr)
	if err != nil {
		log.Printf("Warning: Invalid JWT_TOKEN_LIFESPAN_HOURS ('%s'), using default 24h. Error: %v", jwtLifespanHoursStr, err)
		jwtLifespanHours = 24
	}
	Cfg.JWTTokenLifespan = time.Duration(jwtLifespanHours) * time.Hour

	Cfg.DBHost = getEnv("POSTGRES_HOST", "db")
	Cfg.DBPort = getEnv("POSTGRES_PORT", "5432")
	Cfg.DBUser = getEnv("POSTGRES_USER", "admin")
	Cfg.DBPassword = getEnv("POSTGRES_PASSWORD", "password123")
	Cfg.DBName = getEnv("POSTGRES_DB", "phoenix_grc_dev")
	Cfg.DBSchema = getEnv("DB_SCHEMA", "phoenix_grc")
	Cfg.EnableDBSSL = getEnvAsBool("POSTGRES_SSLMODE_ENABLE", false)

	// Padronizar para APP_ENV. GIN_MODE ainda pode ser usado pelo Gin, mas nossa config usa APP_ENV.
	Cfg.Environment = strings.ToLower(getEnv("APP_ENV", "development"))
	if Cfg.Environment != "development" && Cfg.Environment != "staging" && Cfg.Environment != "production" {
		log.Printf("Warning: Invalid APP_ENV value '%s'. Defaulting to 'development'. Allowed: development, staging, production.", Cfg.Environment)
		Cfg.Environment = "development"
	}
	Cfg.AppVersion = getEnv("APP_VERSION", "0.0.0-dev") // Default version

	Cfg.GCSProjectID = getEnv("GCS_PROJECT_ID", "")
	Cfg.GCSBucketName = getEnv("GCS_BUCKET_NAME", "")

	Cfg.AWSRegion = getEnv("AWS_REGION", "")
	Cfg.AWSSESEmailSender = getEnv("AWS_SES_EMAIL_SENDER", "")
	Cfg.TOTPIssuerName = getEnv("TOTP_ISSUER_NAME", "PhoenixGRC")
	Cfg.AWSS3Bucket = getEnv("AWS_S3_BUCKET", "")
	Cfg.FileStorageProvider = strings.ToLower(getEnv("FILE_STORAGE_PROVIDER", "gcs")) // Default para GCS
	Cfg.FrontendBaseURL = getEnv("FRONTEND_BASE_URL", "http://localhost:3000")
	Cfg.DefaultOrganizationIDForGlobalSSO = getEnv("DEFAULT_ORGANIZATION_ID_FOR_GLOBAL_SSO", "")
	Cfg.AllowSAMLUserCreation = getEnvAsBool("ALLOW_SAML_USER_CREATION", false) // Default false
	Cfg.GithubClientID = getEnv("GITHUB_CLIENT_ID", "")
	Cfg.GithubClientSecret = getEnv("GITHUB_CLIENT_SECRET", "")
	Cfg.GoogleClientID = getEnv("GOOGLE_CLIENT_ID", "")
	Cfg.GoogleClientSecret = getEnv("GOOGLE_CLIENT_SECRET", "")
	Cfg.AllowGlobalSSOUserCreation = getEnvAsBool("ALLOW_GLOBAL_SSO_USER_CREATION", false)

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
