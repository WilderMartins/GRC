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
	FileStorageProvider string // "gcs" ou "s3"
	FeatureToggles      map[string]bool
	// Adicionar outras configurações aqui
}

var Cfg AppConfig

// LoadConfig carrega a configuração da aplicação de variáveis de ambiente.
func LoadConfig() {
	// Carregar .env para desenvolvimento local, ignorar erro se não existir (para produção)
	if err := godotenv.Load(); err != nil {
		log.Println("Aviso: Arquivo .env não encontrado ou erro ao carregar:", err)
	}

	Cfg.Port = getEnv("PORT", "8080")
	Cfg.JWTSecret = getEnv("JWT_SECRET_KEY", "a_very_secure_secret_key_please_change_me_32_chars_long")
	jwtLifespanHours, err := strconv.Atoi(getEnv("JWT_TOKEN_LIFESPAN_HOURS", "24"))
	if err != nil {
		log.Printf("Aviso: JWT_TOKEN_LIFESPAN_HOURS inválido, usando default 24h. Erro: %v", err)
		jwtLifespanHours = 24
	}
	Cfg.JWTTokenLifespan = time.Duration(jwtLifespanHours) * time.Hour

	Cfg.DBHost = getEnv("DB_HOST", "localhost")
	Cfg.DBPort = getEnv("DB_PORT", "5432")
	Cfg.DBUser = getEnv("DB_USER", "phoenix_user")
	Cfg.DBPassword = getEnv("DB_PASSWORD", "phoenix_pass")
	Cfg.DBName = getEnv("DB_NAME", "phoenix_grc_db")
	Cfg.DBSchema = getEnv("DB_SCHEMA", "phoenix_grc") // Esquema padrão
	Cfg.EnableDBSSL = getEnvAsBool("DB_SSL_ENABLE", false)

	Cfg.Environment = getEnv("ENVIRONMENT", "development")

	Cfg.GCSProjectID = getEnv("GCS_PROJECT_ID", "")
	Cfg.GCSBucketName = getEnv("GCS_BUCKET_NAME", "")

	Cfg.AWSRegion = getEnv("AWS_REGION", "")
	Cfg.AWSSESEmailSender = getEnv("AWS_SES_EMAIL_SENDER", "")
	Cfg.TOTPIssuerName = getEnv("TOTP_ISSUER_NAME", "PhoenixGRC")
	Cfg.AWSS3Bucket = getEnv("AWS_S3_BUCKET", "")
	Cfg.FileStorageProvider = strings.ToLower(getEnv("FILE_STORAGE_PROVIDER", "gcs")) // Default para GCS

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
