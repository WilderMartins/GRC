package router

import (
	"net/http"
	"time"

	"phoenixgrc/backend/internal/auth"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/handlers"
	phxmiddleware "phoenixgrc/backend/internal/middleware"
	"phoenixgrc/backend/internal/oauth2auth"
	"phoenixgrc/backend/internal/samlauth"
	phxlog "phoenixgrc/backend/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// SetupRouter configura e retorna uma instância do Gin Engine.
func SetupRouter(log *zap.Logger) *gin.Engine {
	router := gin.New()

	// Adicionar middlewares globais
	router.Use(phxmiddleware.Metrics())
	router.Use(phxmiddleware.GinZap(log, time.RFC3339, true))
	router.Use(phxmiddleware.GinRecovery(log, time.RFC3339, true, true))

	// Endpoint para métricas Prometheus
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Rotas de Saúde
	router.GET("/health", healthCheckHandler)

	// Rotas Públicas (sem autenticação JWT)
	setupPublicRoutes(router)

	// Rotas de Autenticação
	setupAuthRoutes(router)

	// Rotas da API v1 (protegidas por JWT)
	setupV1Routes(router)

	return router
}

func healthCheckHandler(c *gin.Context) {
	// Obter a instância do banco de dados SQL do GORM
	sqlDB, err := database.DB.DB()
	if err != nil {
		phxlog.L.Error("Erro ao obter a instância do DB para o health check", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "database instance error"})
		return
	}

	// Ping no banco de dados para verificar a conectividade
	err = sqlDB.Ping()
	if err != nil {
		phxlog.L.Error("Falha no ping do banco de dados durante o health check", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "message": "database ping failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"database": "connected",
	})
}

func setupPublicRoutes(r *gin.Engine) {
	publicApi := r.Group("/api/public")
	{
		publicApi.GET("/social-identity-providers", handlers.ListGlobalSocialIdentityProvidersHandler)
		publicApi.GET("/saml-identity-providers", handlers.ListGlobalSAMLIdentityProvidersHandler)
		publicApi.GET("/setup-status", handlers.GetSetupStatusHandler)
	}
}

func setupAuthRoutes(r *gin.Engine) {
	authRoutes := r.Group("/auth")
	{
		authRoutes.POST("/setup", handlers.PerformSetupHandler)
		authRoutes.POST("/login", handlers.LoginHandler)

		samlIdPGroup := authRoutes.Group("/saml/:idpId")
		{
			samlIdPGroup.GET("/metadata", samlauth.MetadataHandler)
			samlIdPGroup.POST("/acs", samlauth.ACSHandler)
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

		authRoutes.POST("/login/2fa/verify", handlers.LoginVerifyTOTPHandler)
		authRoutes.POST("/login/2fa/backup-code/verify", handlers.LoginVerifyBackupCodeHandler)
		authRoutes.POST("/forgot-password", handlers.ForgotPasswordHandler)
		authRoutes.POST("/reset-password", handlers.ResetPasswordHandler)
	}
}

func setupV1Routes(r *gin.Engine) {
	apiV1 := r.Group("/api/v1")
	apiV1.Use(auth.AuthMiddleware())
	{
		apiV1.GET("/me", func(c *gin.Context) {
			userID, _ := c.Get("userID")
			userEmail, _ := c.Get("userEmail")
			userRole, _ := c.Get("userRole")
			orgID, _ := c.Get("organizationID")
			c.JSON(http.StatusOK, gin.H{
				"message":         "This is a protected route",
				"user_id":         userID,
				"email":           userEmail,
				"role":            userRole,
				"organization_id": orgID,
			})
		})

		// Risk Routes
		riskRoutes := apiV1.Group("/risks")
		{
			riskRoutes.POST("", handlers.CreateRiskHandler)
			riskRoutes.GET("", handlers.ListRisksHandler)
			riskRoutes.GET("/:riskId", handlers.GetRiskHandler)
			riskRoutes.PUT("/:riskId", handlers.UpdateRiskHandler)
			riskRoutes.DELETE("/:riskId", handlers.DeleteRiskHandler)
			riskRoutes.POST("/bulk-upload-csv", handlers.BulkUploadRisksCSVHandler)
			riskRoutes.POST("/:riskId/submit-acceptance", handlers.SubmitRiskForAcceptanceHandler)
			riskRoutes.GET("/:riskId/approval-history", handlers.GetRiskApprovalHistoryHandler)
			riskRoutes.POST("/:riskId/approval/:approvalId/decide", handlers.ApproveOrRejectRiskAcceptanceHandler)

			stakeholderRoutes := riskRoutes.Group("/:riskId/stakeholders")
			{
				stakeholderRoutes.POST("", handlers.AddRiskStakeholderHandler)
				stakeholderRoutes.GET("", handlers.ListRiskStakeholdersHandler)
				stakeholderRoutes.DELETE("/:userId", handlers.RemoveRiskStakeholderHandler)
			}
		}

		// Organization Routes
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

		// Vulnerability Routes
		vulnerabilityRoutes := apiV1.Group("/vulnerabilities")
		{
			vulnerabilityRoutes.POST("", handlers.CreateVulnerabilityHandler)
			vulnerabilityRoutes.GET("", handlers.ListVulnerabilitiesHandler)
			vulnerabilityRoutes.POST("/import-csv", handlers.ImportVulnerabilitiesCSVHandler)
			vulnerabilityRoutes.GET("/:vulnId", handlers.GetVulnerabilityHandler)
			vulnerabilityRoutes.PUT("/:vulnId", handlers.UpdateVulnerabilityHandler)
			vulnerabilityRoutes.DELETE("/:vulnId", handlers.DeleteVulnerabilityHandler)
		}

		// Audit Routes
		auditRoutes := apiV1.Group("/audit")
		{
			auditRoutes.GET("/frameworks", handlers.ListFrameworksHandler)
			auditRoutes.GET("/frameworks/:frameworkId/controls", handlers.GetFrameworkControlsHandler)
			auditRoutes.GET("/frameworks/:frameworkId/control-families", handlers.GetControlFamiliesForFrameworkHandler)
			auditRoutes.POST("/assessments", handlers.CreateOrUpdateAssessmentHandler)
			auditRoutes.GET("/assessments/control/:controlId", handlers.GetAssessmentForControlHandler)
			auditRoutes.DELETE("/assessments/:assessmentId/evidence", handlers.DeleteAssessmentEvidenceHandler)
			auditRoutes.GET("/organizations/:orgId/frameworks/:frameworkId/assessments", handlers.ListOrgAssessmentsByFrameworkHandler)
			auditRoutes.GET("/organizations/:orgId/frameworks/:frameworkId/compliance-score", handlers.GetComplianceScoreHandler)
			auditRoutes.GET("/organizations/:orgId/frameworks/:frameworkId/c2m2-maturity-summary", handlers.GetC2M2MaturitySummaryHandler)
		}

		// C2M2 Routes
		c2m2Routes := apiV1.Group("/c2m2")
		{
			c2m2Routes.GET("/domains", handlers.ListC2M2DomainsHandler)
			c2m2Routes.GET("/domains/:domainId/practices", handlers.ListC2M2PracticesByDomainHandler)
		}

		// MFA Routes
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
				backupCodeRoutes.POST("/generate", handlers.GenerateBackupCodesHandler)
			}
		}

		// User-specific routes
		apiV1.GET("/me/dashboard/summary", handlers.GetUserDashboardSummaryHandler)
		apiV1.GET("/users/organization-lookup", handlers.OrganizationUserLookupHandler)

		// File Access Routes
		fileAccessRoutes := apiV1.Group("/files")
		{
			fileAccessRoutes.GET("/signed-url", handlers.GetSignedURLForObjectHandler)
		}

		// System Admin Routes
		adminRoutes := apiV1.Group("/admin")
		{
			settingsRoutes := adminRoutes.Group("/settings")
			{
				settingsRoutes.GET("", handlers.ListSystemSettingsHandler)
				settingsRoutes.PUT("", handlers.UpdateSystemSettingsHandler)
				settingsRoutes.POST("/test-email", handlers.SendTestEmailHandler)
			}
		}

		// Dashboard Routes
		dashboardRoutes := apiV1.Group("/dashboard")
		{
			dashboardRoutes.GET("/risk-matrix", handlers.GetRiskMatrixHandler)
			dashboardRoutes.GET("/vulnerability-summary", handlers.GetVulnerabilitySummaryHandler)
			dashboardRoutes.GET("/compliance-overview", handlers.GetComplianceOverviewHandler)
			dashboardRoutes.GET("/recent-activity", handlers.GetRecentActivityHandler)
		}
	}
}
