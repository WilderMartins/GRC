package main

import (
	"fmt"
	"log" // Será gradualmente substituído
	"net/http"
	"os"
	"strings" // Para ToLower em appEnv
	"phoenixgrc/backend/internal/auth"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/handlers"
	// "phoenixgrc/backend/internal/models" // No longer directly used here for setup
	"phoenixgrc/backend/internal/oauth2auth"
	"phoenixgrc/backend/internal/samlauth"   // Descomentado
	// "phoenixgrc/backend/internal/seeders" // Setup will handle its own seeding call
	"phoenixgrc/backend/internal/filestorage"
	"phoenixgrc/backend/internal/notifications"
	// "strings" // No longer needed for setup here
	"go.uber.org/zap" // Adicionar import do zap
	"time"            // Para time.RFC3339 no middleware de log

	"phoenixgrc/backend/pkg/config"                        // Importar config para Cfg.Environment
	phxmiddleware "phoenixgrc/backend/internal/middleware" // Importar o pacote de middleware
	"phoenixgrc/backend/cmd/setup"                         // Descomentado para permitir a chamada do setup
	// "phoenixgrc/backend/cmd/setup" // Comentado para permitir compilação do server isoladamente. Refatorar setup.

	"github.com/gin-gonic/gin"
	// "golang.org/x/crypto/bcrypt" // Moved to setup package
	phxlog "phoenixgrc/backend/pkg/log" // Importar o novo pacote de logger
	"crypto/rand"
	"encoding/hex"
	"github.com/prometheus/client_golang/prometheus/promhttp" // Para expor métricas
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

		// Gerar nova JWT_SECRET_KEY (64 bytes, codificado em base64)
		jwtBytes := make([]byte, 64)
		if _, err := rand.Read(jwtBytes); err != nil {
			log.Fatal("Falha ao gerar a chave JWT.", zap.Error(err))
		}
		newJwtKey := hex.EncodeToString(jwtBytes) // Usando hex para simplicidade de string

		// Gerar nova ENCRYPTION_KEY_HEX (32 bytes, 64 caracteres hex)
		encBytes := make([]byte, 32)
		if _, err := rand.Read(encBytes); err != nil {
			log.Fatal("Falha ao gerar a chave de criptografia.", zap.Error(err))
		}
		newEncKey := hex.EncodeToString(encBytes)

		// Exibe as novas chaves e encerra.
		// Não tentamos escrever no .env para evitar problemas de permissão e
		// para garantir que o usuário esteja ciente da alteração.
		fmt.Println("------------------------------------------------------------------")
		fmt.Println("ATENÇÃO: As chaves de segurança não foram configuradas.")
		fmt.Println("Por favor, adicione as seguintes linhas ao seu arquivo .env:")
		fmt.Println("------------------------------------------------------------------")
		fmt.Printf("JWT_SECRET_KEY=%s\n", newJwtKey)
		fmt.Printf("ENCRYPTION_KEY_HEX=%s\n", newEncKey)
		fmt.Println("------------------------------------------------------------------")
		fmt.Println("A aplicação será encerrada. Após atualizar o arquivo .env, inicie-a novamente.")

		// Encerra a aplicação para forçar o usuário a atualizar as chaves.
		os.Exit(1)
	}
}

func startServer() {
	// Inicializar o logger global zap primeiro
	// Usar config.Cfg.Environment (carregado de APP_ENV) e LOG_LEVEL.
	phxlog.Init(os.Getenv("LOG_LEVEL"), os.Getenv("APP_ENV")) // Simplificado

	// Verificar chaves de segurança ANTES de qualquer outra coisa
	checkAndGenerateKeys()
	// A função Init do pacote log já lê LOG_LEVEL e APP_ENV em seu próprio init(),
	// mas chamá-la aqui explicitamente com os valores da config garante que
	// a configuração carregada por LoadConfig() seja usada.
	logLevel := os.Getenv("LOG_LEVEL") // LOG_LEVEL pode ser sobrescrito por env var direta
	if logLevel == "" {
		logLevel = "info" // Default se LOG_LEVEL não estiver diretamente no env para esta chamada
	}
	// config.Cfg.Environment já foi carregado e padronizado por config.LoadConfig()
	phxlog.Init(logLevel, config.Cfg.Environment)

	// Agora usar phxlog.L ou phxlog.S para logging
	if err := auth.InitializeJWT(); err != nil {
		phxlog.L.Fatal("Failed to initialize JWT", zap.Error(err))
	}
	phxlog.L.Info("JWT Initialized.")

	// SAML Global Config Initialization
	if err := samlauth.InitializeSAMLSPGlobalConfig(); err != nil {
		phxlog.L.Warn("Failed to initialize SAML SP Global Config. SAML logins may not work.", zap.Error(err))
	} else {
		phxlog.L.Info("SAML SP Global Config Initialized.")
	}

	if err := oauth2auth.InitializeOAuth2GlobalConfig(); err != nil {
		phxlog.L.Fatal("Failed to initialize OAuth2 Global Config", zap.Error(err))
	}
	phxlog.L.Info("OAuth2 Global Config Initialized.")

	if err := filestorage.InitFileStorage(); err != nil {
		phxlog.L.Warn("File storage initialization failed. Uploads may not work.", zap.Error(err))
	}

	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")
	dbSSLMode := os.Getenv("POSTGRES_SSLMODE")

	if dbHost == "" { dbHost = "db" }
	if dbPort == "" { dbPort = "5432" }
	if dbSSLMode == "" { dbSSLMode = "disable" }
	if dbUser == "" || dbPassword == "" || dbName == "" {
		phxlog.L.Fatal("Database credentials (POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB) must be set for the server.")
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	if err := database.ConnectDB(dsn); err != nil {
		phxlog.L.Fatal("Failed to connect to database for the server", zap.Error(err))
	}
	phxlog.L.Info("Database connection established for the server.")

	// Inicializar serviços que dependem do DB
	notifications.InitEmailService()

	router := gin.New() // Usar gin.New() para controle explícito de middleware

	// Adicionar middlewares globais:
	// 1. Metrics middleware - deve vir primeiro ou no início para medir a latência total.
	router.Use(phxmiddleware.Metrics())
	// 2. GinZap para logging estruturado de requisições
	// 3. GinRecovery para capturar panics, logá-los com zap e retornar 500
	// O formato de tempo RFC3339 é um bom padrão. UTC para consistência.
	router.Use(phxmiddleware.GinZap(phxlog.L, time.RFC3339, true))
	router.Use(phxmiddleware.GinRecovery(phxlog.L, time.RFC3339, true, true)) // true para recovery (retornar 500)

	// Endpoint para métricas Prometheus
	// Deve ser público e não agrupado sob /api ou /auth
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Rotas Públicas (sem autenticação JWT)
	publicApi := router.Group("/api/public")
	{
		publicApi.GET("/social-identity-providers", handlers.ListGlobalSocialIdentityProvidersHandler)
		publicApi.GET("/saml-identity-providers", handlers.ListGlobalSAMLIdentityProvidersHandler)
		publicApi.GET("/setup-status", handlers.GetSetupStatusHandler) // Novo endpoint de status
		// Outras rotas públicas podem ser adicionadas aqui no futuro
	}

	router.GET("/health", func(c *gin.Context) {
		sqlDB, err := database.DB.DB()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "database instance error"})
			return
		}
		err = sqlDB.Ping()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "database ping failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":   "ok",
			"database": "connected",
		})
	})

	authRoutes := router.Group("/auth")
	{
		// Endpoint para o setup inicial via API. Deve ser uma das primeiras rotas.
		authRoutes.POST("/setup", handlers.PerformSetupHandler)

		authRoutes.POST("/login", handlers.LoginHandler) // Restaurado para usar o handler implementado

		samlIdPGroup := authRoutes.Group("/saml/:idpId")
		{
			samlIdPGroup.GET("/metadata", samlauth.MetadataHandler)
			samlIdPGroup.POST("/acs", samlauth.ACSHandler) // O middleware samlsp pode proteger este
			samlIdPGroup.GET("/login", samlauth.SAMLLoginHandler)
		}

		oauth2GoogleGroup := authRoutes.Group("/oauth2/google/:idpId")
		{
			oauth2GoogleGroup.GET("/login", oauth2auth.GoogleLoginHandler)
			oauth2GoogleGroup.GET("/callback", oauth2auth.GoogleCallbackHandler)
		}

		oauth2GithubGroup := authRoutes.Group("/oauth2/github/:idpId")
		{
			oauth2GithubGroup.GET("/login", oauth2auth.GithubLoginHandler)
			oauth2GithubGroup.GET("/callback", oauth2auth.GithubCallbackHandler)
		}
		// 2FA TOTP Verification as part of login
		authRoutes.POST("/login/2fa/verify", handlers.LoginVerifyTOTPHandler)
		// 2FA Backup Code Verification as part of login
		authRoutes.POST("/login/2fa/backup-code/verify", handlers.LoginVerifyBackupCodeHandler)

		// Password Reset
		authRoutes.POST("/forgot-password", handlers.ForgotPasswordHandler)
		authRoutes.POST("/reset-password", handlers.ResetPasswordHandler)
	}

	apiV1 := router.Group("/api/v1")
	apiV1.Use(auth.AuthMiddleware())
	{
		apiV1.GET("/me", func(c *gin.Context) {
			userID, _ := c.Get("userID")
			userEmail, _ := c.Get("userEmail")
			userRole, _ := c.Get("userRole")
			orgID, _ := c.Get("organizationID")
			c.JSON(http.StatusOK, gin.H{
				"message":      "This is a protected route",
				"user_id":      userID,
				"email":        userEmail,
				"role":         userRole,
				"organization_id": orgID,
			})
		})

		riskRoutes := apiV1.Group("/risks")
		{
			riskRoutes.POST("", handlers.CreateRiskHandler)
			riskRoutes.GET("", handlers.ListRisksHandler)
			riskRoutes.GET("/:riskId", handlers.GetRiskHandler)
			riskRoutes.PUT("/:riskId", handlers.UpdateRiskHandler)
			riskRoutes.DELETE("/:riskId", handlers.DeleteRiskHandler)
			riskRoutes.POST("/:riskId/submit-acceptance", handlers.SubmitRiskForAcceptanceHandler)
			riskRoutes.GET("/:riskId/approval-history", handlers.GetRiskApprovalHistoryHandler)
			riskRoutes.POST("/:riskId/approval/:approvalId/decide", handlers.ApproveOrRejectRiskAcceptanceHandler)
			riskRoutes.POST("/bulk-upload-csv", handlers.BulkUploadRisksCSVHandler)

			// Stakeholder routes
			stakeholderRoutes := riskRoutes.Group("/:riskId/stakeholders")
			{
				stakeholderRoutes.POST("", handlers.AddRiskStakeholderHandler)
				stakeholderRoutes.GET("", handlers.ListRiskStakeholdersHandler)
				stakeholderRoutes.DELETE("/:userId", handlers.RemoveRiskStakeholderHandler)
			}
		}

		orgRoutes := apiV1.Group("/organizations/:orgId")
		{
			idpRoutes := orgRoutes.Group("/identity-providers")
			{
				idpRoutes.POST("", handlers.CreateIdentityProviderHandler)
				idpRoutes.GET("", handlers.ListIdentityProvidersHandler)
				idpRoutes.GET("/:idpId", handlers.GetIdentityProviderHandler)
				idpRoutes.PUT("/:idpId", handlers.UpdateIdentityProviderHandler)
				idpRoutes.DELETE("/:idpId", handlers.DeleteIdentityProviderHandler)
			}
			webhookRoutes := orgRoutes.Group("/webhooks")
			{
				webhookRoutes.POST("", handlers.CreateWebhookHandler)
				webhookRoutes.GET("", handlers.ListWebhooksHandler)
				webhookRoutes.GET("/:webhookId", handlers.GetWebhookHandler)
				webhookRoutes.PUT("/:webhookId", handlers.UpdateWebhookHandler)
				webhookRoutes.DELETE("/:webhookId", handlers.DeleteWebhookHandler)
				webhookRoutes.POST("/:webhookId/test", handlers.SendTestWebhookHandler)
			}
			userManagementRoutes := orgRoutes.Group("/users")
			{
				userManagementRoutes.GET("", handlers.ListOrganizationUsersHandler)
				userManagementRoutes.GET("/:userId", handlers.GetOrganizationUserHandler)
				userManagementRoutes.PUT("/:userId/role", handlers.UpdateOrganizationUserRoleHandler)
				userManagementRoutes.PUT("/:userId/status", handlers.UpdateOrganizationUserStatusHandler)
			}
			orgRoutes.PUT("/branding", handlers.UpdateOrganizationBrandingHandler)
			orgRoutes.GET("/branding", handlers.GetOrganizationBrandingHandler)
		}

		vulnerabilityRoutes := apiV1.Group("/vulnerabilities")
		{
			vulnerabilityRoutes.POST("", handlers.CreateVulnerabilityHandler)
			vulnerabilityRoutes.GET("", handlers.ListVulnerabilitiesHandler)
			vulnerabilityRoutes.POST("/import-csv", handlers.ImportVulnerabilitiesCSVHandler)
			vulnerabilityRoutes.GET("/:vulnId", handlers.GetVulnerabilityHandler)
			vulnerabilityRoutes.PUT("/:vulnId", handlers.UpdateVulnerabilityHandler)
			vulnerabilityRoutes.DELETE("/:vulnId", handlers.DeleteVulnerabilityHandler)
		}

		auditRoutes := apiV1.Group("/audit")
		{
			auditRoutes.GET("/frameworks", handlers.ListFrameworksHandler)
			auditRoutes.GET("/frameworks/:frameworkId/controls", handlers.GetFrameworkControlsHandler)
			auditRoutes.GET("/frameworks/:frameworkId/control-families", handlers.GetControlFamiliesForFrameworkHandler) // Nova rota
			auditRoutes.POST("/assessments", handlers.CreateOrUpdateAssessmentHandler)
			auditRoutes.GET("/assessments/control/:controlId", handlers.GetAssessmentForControlHandler)
			auditRoutes.DELETE("/assessments/:assessmentId/evidence", handlers.DeleteAssessmentEvidenceHandler) // Nova rota para deletar evidência
			auditRoutes.GET("/organizations/:orgId/frameworks/:frameworkId/assessments", handlers.ListOrgAssessmentsByFrameworkHandler)
			auditRoutes.GET("/organizations/:orgId/frameworks/:frameworkId/compliance-score", handlers.GetComplianceScoreHandler)
			auditRoutes.GET("/organizations/:orgId/frameworks/:frameworkId/c2m2-maturity-summary", handlers.GetC2M2MaturitySummaryHandler) // Novo endpoint C2M2
		}

		c2m2Routes := apiV1.Group("/c2m2")
		{
			c2m2Routes.GET("/domains", handlers.ListC2M2DomainsHandler)
			c2m2Routes.GET("/domains/:domainId/practices", handlers.ListC2M2PracticesByDomainHandler)
		}

		// MFA Routes (operam no usuário autenticado - /me)
		mfaRoutes := apiV1.Group("/users/me/2fa")
		{
			mfaTOTPRoutes := mfaRoutes.Group("/totp")
			{
				mfaTOTPRoutes.POST("/setup", handlers.SetupTOTPHandler)
				mfaTOTPRoutes.POST("/verify", handlers.VerifyTOTPHandler)
				mfaTOTPRoutes.POST("/disable", handlers.DisableTOTPHandler)
			}
			backupCodeRoutes := mfaRoutes.Group("/backup-codes")
			{
				backupCodeRoutes.POST("/generate", handlers.GenerateBackupCodesHandler) // Usar POST para gerar/regerar
				// backupCodeRoutes.POST("/verify", handlers.VerifyBackupCodeHandler)   // TODO - parte do login 2FA
			}
		}
		apiV1.GET("/me/dashboard/summary", handlers.GetUserDashboardSummaryHandler) // Rota para o sumário do dashboard do usuário

		// Endpoint para lookup de usuários da organização (para filtros, dropdowns, etc.)
		// Colocado no nível /api/v1/ pois não é específico de uma organização via path param,
		// mas opera na organização do usuário autenticado.
		apiV1.GET("/users/organization-lookup", handlers.OrganizationUserLookupHandler)

		// Endpoint para obter URLs assinadas para acesso a arquivos
		fileAccessRoutes := apiV1.Group("/files")
		{
			fileAccessRoutes.GET("/signed-url", handlers.GetSignedURLForObjectHandler)
		}

		// Rotas de Administração do Sistema
		adminRoutes := apiV1.Group("/admin")
		// Adicionar um middleware de verificação de role de admin aqui se necessário
		{
			settingsRoutes := adminRoutes.Group("/settings")
			{
				settingsRoutes.GET("", handlers.ListSystemSettingsHandler)
				settingsRoutes.PUT("", handlers.UpdateSystemSettingsHandler)
				settingsRoutes.POST("/test-email", handlers.SendTestEmailHandler)
			}
		}

		dashboardRoutes := apiV1.Group("/dashboard")
		{
			dashboardRoutes.GET("/risk-matrix", handlers.GetRiskMatrixHandler)
			dashboardRoutes.GET("/vulnerability-summary", handlers.GetVulnerabilitySummaryHandler)
			dashboardRoutes.GET("/compliance-overview", handlers.GetComplianceOverviewHandler)
			dashboardRoutes.GET("/recent-activity", handlers.GetRecentActivityHandler)
		}
	}

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080"
	}
	phxlog.L.Info("Starting server", zap.String("port", serverPort))
	if err := router.Run(":" + serverPort); err != nil {
		phxlog.L.Fatal("Failed to start server", zap.Error(err))
	}
}

func main() {
	// O ponto de entrada agora sempre inicia o servidor.
	// O setup é tratado pelo endpoint da API.
	startServer()
}
