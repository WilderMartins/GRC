package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"phoenixgrc/backend/internal/auth"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/handlers"
	// "phoenixgrc/backend/internal/models" // No longer directly used here for setup
	"phoenixgrc/backend/internal/oauth2auth"
	// "phoenixgrc/backend/internal/samlauth"   // Temporariamente comentado
	// "phoenixgrc/backend/internal/seeders" // Setup will handle its own seeding call
	"phoenixgrc/backend/internal/filestorage"
	"phoenixgrc/backend/internal/notifications"
	// "strings" // No longer needed for setup here

	"phoenixgrc/backend/cmd/setup" // Descomentado para permitir a chamada do setup
	// "phoenixgrc/backend/cmd/setup" // Comentado para permitir compilação do server isoladamente. Refatorar setup.

	"github.com/gin-gonic/gin"
	// "golang.org/x/crypto/bcrypt" // Moved to setup package
)

// runSetup() function is now removed from here and exists in backend/cmd/setup/main.go

func startServer() {
	if err := auth.InitializeJWT(); err != nil {
		log.Fatalf("Failed to initialize JWT: %v", err)
	}
	log.Println("JWT Initialized.")

	/* // SAML Temporariamente Comentado
	if err := samlauth.InitializeSAMLSPGlobalConfig(); err != nil {
		log.Fatalf("Failed to initialize SAML SP Global Config: %v", err)
	}
	log.Println("SAML SP Global Config Initialized.")
	*/

	if err := oauth2auth.InitializeOAuth2GlobalConfig(); err != nil {
		log.Fatalf("Failed to initialize OAuth2 Global Config: %v", err)
	}
	log.Println("OAuth2 Global Config Initialized.")

	if err := filestorage.InitFileStorage(); err != nil {
		log.Printf("Warning: File storage initialization failed: %v. Uploads may not work.", err)
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
		log.Fatal("Database credentials (POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB) must be set for the server.")
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	if err := database.ConnectDB(dsn); err != nil {
		log.Fatalf("Failed to connect to database for the server: %v", err)
	}
	log.Println("Database connection established for the server.")

	router := gin.Default()

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
		/* // SAML Temporariamente Comentado
		samlIdPGroup := authRoutes.Group("/saml/:idpId")
		{
			samlIdPGroup.GET("/metadata", samlauth.MetadataHandler)
			samlIdPGroup.POST("/acs", samlauth.ACSHandler)
			samlIdPGroup.GET("/login", samlauth.SAMLLoginHandler)
		}
		*/
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
			auditRoutes.POST("/assessments", handlers.CreateOrUpdateAssessmentHandler)
			auditRoutes.GET("/assessments/control/:controlId", handlers.GetAssessmentForControlHandler)
			auditRoutes.GET("/organizations/:orgId/frameworks/:frameworkId/assessments", handlers.ListOrgAssessmentsByFrameworkHandler)
			auditRoutes.GET("/organizations/:orgId/frameworks/:frameworkId/compliance-score", handlers.GetComplianceScoreHandler)
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
	}

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080"
	}
	log.Printf("Starting server on port %s", serverPort)
	if err := router.Run(":" + serverPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "setup" {
		// Call RunSetup from the setup package
		// A função RunSetup agora é pública no pacote setup e pode ser chamada.
		// O pacote setup precisa ser ajustado para que main.go possa ser importado,
		// ou a lógica de RunSetup movida para um pacote internal/setupUtils e chamada por ambos os cmd.
		// Assumindo que phoenixgrc/backend/cmd/setup pode ser importado e RunSetup é acessível:
		log.Println("Starting Phoenix GRC setup...")
		setup.RunSetup() // Descomentado para habilitar o setup via ./server setup
		log.Println("Phoenix GRC setup finished.")
	} else {
		startServer()
	}
}
