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

	phxmiddleware "phoenixgrc/backend/internal/middleware" // Importar o pacote de middleware
	"phoenixgrc/backend/cmd/setup"                         // Descomentado para permitir a chamada do setup
	// "phoenixgrc/backend/cmd/setup" // Comentado para permitir compilação do server isoladamente. Refatorar setup.

	"github.com/gin-gonic/gin"
	// "golang.org/x/crypto/bcrypt" // Moved to setup package
	phxlog "phoenixgrc/backend/pkg/log" // Importar o novo pacote de logger
	"github.com/prometheus/client_golang/prometheus/promhttp" // Para expor métricas
)

// runSetup() function is now removed from here and exists in backend/cmd/setup/main.go

func startServer() {
	// Inicializar o logger global zap primeiro
	// Usar GIN_MODE para determinar o ambiente (development vs production) para o logger
	// e LOG_LEVEL para o nível de log.
	appEnv := os.Getenv("GIN_MODE") // gin.ReleaseMode ("release") ou gin.DebugMode ("debug")
	if strings.ToLower(appEnv) == gin.ReleaseMode {
		appEnv = "production" // Mapear "release" para "production" para o logger
	} else {
		appEnv = "development" // Default para development se não for release
	}
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info" // Default
	}
	phxlog.Init(logLevel, appEnv) // Usar phxlog.Init para evitar conflito com log.L padrão do Go

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

	notifications.InitEmailService()

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
	// Logger já foi inicializado pela importação de pkg/log ou será re-inicializado em startServer().
	// Se startServer() não for chamado (ex: no fluxo de setup), o logger da importação será usado.
	if len(os.Args) > 1 && os.Args[1] == "setup" {
		// A inicialização do logger em startServer() não ocorrerá.
		// O logger já foi inicializado pelo init() do pacote phxlog.
		// Podemos re-inicializá-lo aqui se quisermos garantir consistência com as vars de env
		// lidas em main, mas o init() do pkg/log já faz isso.
		// Por segurança, podemos chamar phxlog.Init aqui também, não fará mal.
		appEnv := os.Getenv("GIN_MODE")
		if strings.ToLower(appEnv) == gin.ReleaseMode {
			appEnv = "production"
		} else {
			appEnv = "development"
		}
		logLevel := os.Getenv("LOG_LEVEL")
		if logLevel == "" {
			logLevel = "info"
		}
		phxlog.Init(logLevel, appEnv) // Re-inicializa com base nas vars de main.

		phxlog.L.Info("Starting Phoenix GRC setup...")
		setup.RunSetup() // RunSetup usará o logger global phxlog.L / phxlog.S
		phxlog.L.Info("Phoenix GRC setup finished.")
	} else {
		startServer()
	}
}
