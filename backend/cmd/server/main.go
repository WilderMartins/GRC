package main

import (
	"fmt"
	"os"

	"phoenixgrc/backend/internal/auth"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/filestorage"
	"phoenixgrc/backend/internal/notifications"
	"phoenixgrc/backend/internal/oauth2auth"
	"phoenixgrc/backend/internal/router"
	"phoenixgrc/backend/internal/samlauth"
	phxlog "phoenixgrc/backend/pkg/log"

	"crypto/rand"
	"encoding/hex"

	"go.uber.org/zap"
)

// checkAndGenerateKeys verifica as chaves de segurança essenciais.
// Se elas estiverem ausentes ou com os valores padrão, gera novas chaves
// e encerra a aplicação com instruções para o usuário.
func checkAndGenerateKeys() {
	log := phxlog.L.Named("KeyCheck")
	jwtKey := os.Getenv("JWT_SECRET_KEY")
	encryptionKey := os.Getenv("ENCRYPTION_KEY_HEX")

	jwtDefault := "mudar_esta_chave_em_producao_com_um_valor_aleatorio_longo"
	encryptionKeyDefault := "mudar_para_64_caracteres_hexadecimais_em_producao"

	if jwtKey == "" || jwtKey == jwtDefault || encryptionKey == "" || encryptionKey == encryptionKeyDefault {
		log.Warn("Chaves de segurança ausentes ou padrão detectadas. Gerando novas chaves.")

		// Gerar nova JWT_SECRET_KEY (64 bytes, codificado em hex)
		jwtBytes := make([]byte, 64)
		if _, err := rand.Read(jwtBytes); err != nil {
			log.Fatal("Falha ao gerar a chave JWT.", zap.Error(err))
		}
		newJwtKey := hex.EncodeToString(jwtBytes)

		// Gerar nova ENCRYPTION_KEY_HEX (32 bytes, 64 caracteres hex)
		encBytes := make([]byte, 32)
		if _, err := rand.Read(encBytes); err != nil {
			log.Fatal("Falha ao gerar a chave de criptografia.", zap.Error(err))
		}
		newEncKey := hex.EncodeToString(encBytes)

		fmt.Println("------------------------------------------------------------------")
		fmt.Println("ATENÇÃO: As chaves de segurança não foram configuradas.")
		fmt.Println("Por favor, adicione as seguintes linhas ao seu arquivo .env:")
		fmt.Println("------------------------------------------------------------------")
		fmt.Printf("JWT_SECRET_KEY=%s\n", newJwtKey)
		fmt.Printf("ENCRYPTION_KEY_HEX=%s\n", newEncKey)
		fmt.Println("------------------------------------------------------------------")
		fmt.Println("A aplicação será encerrada. Após atualizar o arquivo .env, inicie-a novamente.")
		os.Exit(1)
	}
}

// initializeServices coordena a inicialização de todos os serviços principais.
// Retorna um erro se qualquer inicialização crítica falhar.
func initializeServices() error {
	phxlog.Init(os.Getenv("LOG_LEVEL"), os.Getenv("APP_ENV"))
	log := phxlog.L.Named("Initialization")

	// 1. Configuração e Chaves (Crítico)
	checkAndGenerateKeys()
	log.Info("Verificação de chaves de segurança concluída.")

	// 2. JWT (Crítico)
	if err := auth.InitializeJWT(); err != nil {
		return fmt.Errorf("falha ao inicializar JWT: %w", err)
	}
	log.Info("JWT inicializado com sucesso.")

	// 3. Banco de Dados (Crítico)
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")
	dbSSLMode := os.Getenv("POSTGRES_SSLMODE")

	if dbHost == "" {
		dbHost = "db"
	}
	if dbPort == "" {
		dbPort = "5432"
	}
	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}
	if dbUser == "" || dbPassword == "" || dbName == "" {
		return fmt.Errorf("credenciais do banco de dados (POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB) devem ser definidas")
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	if err := database.ConnectDB(dsn); err != nil {
		return fmt.Errorf("falha ao conectar ao banco de dados: %w", err)
	}
	log.Info("Conexão com o banco de dados estabelecida com sucesso.")

	// 4. Serviços Opcionais/Não-Críticos (registram avisos em caso de falha)
	if err := samlauth.InitializeSAMLSPGlobalConfig(); err != nil {
		return fmt.Errorf("falha ao inicializar a configuração global do SAML SP: %w", err)
	}
	log.Info("Configuração global do SAML SP inicializada.")

	if err := oauth2auth.InitializeOAuth2GlobalConfig(); err != nil {
		return fmt.Errorf("falha ao inicializar a configuração global do OAuth2: %w", err)
	}
	log.Info("Configuração global do OAuth2 inicializada.")

	if err := filestorage.InitFileStorage(); err != nil {
		return fmt.Errorf("inicialização do armazenamento de arquivos falhou: %w", err)
	}
	log.Info("Armazenamento de arquivos inicializado.")

	notifications.InitEmailService()
	log.Info("Serviço de e-mail inicializado.")

	return nil
}

func main() {
	// A configuração agora é carregada automaticamente pelo `init()` no pacote config.
	// A chamada explícita não é mais necessária aqui.
	// _ = config.LoadConfig()

	// Inicializa todos os serviços. A aplicação encerra se um serviço crítico falhar.
	if err := initializeServices(); err != nil {
		phxlog.L.Fatal("Falha na inicialização de serviço crítico.", zap.Error(err))
	}

	// Configura o roteador
	appRouter := router.SetupRouter(phxlog.L)

	// Inicia o servidor
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080"
	}

	phxlog.L.Info("Iniciando o servidor", zap.String("port", serverPort))
	if err := appRouter.Run(":" + serverPort); err != nil {
		phxlog.L.Fatal("Falha ao iniciar o servidor", zap.Error(err))
	}
}
