package handlers

import (
	"fmt"
	"net/http"
	"phoenixgrc/backend/pkg/config" // Para acessar config.Cfg
	phxlog "phoenixgrc/backend/pkg/log"  // Importar o logger zap
	"go.uber.org/zap"                 // Importar zap
	"strings"

	"phoenixgrc/backend/internal/models"
	"phoenixgrc/backend/internal/seeders"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// SetupPayload define a estrutura de dados para o setup inicial via API.
type SetupPayload struct {
	OrganizationName string `json:"organization_name" binding:"required,min=3,max=100"`
	AdminName        string `json:"admin_name" binding:"required,min=3,max=100"`
	AdminEmail       string `json:"admin_email" binding:"required,email"`
	AdminPassword    string `json:"admin_password" binding:"required,min=8"`
}

// PerformSetupHandler lida com a requisição para executar o setup inicial.
func PerformSetupHandler(c *gin.Context) {
	log := phxlog.L.Named("PerformSetupHandler")

	// 1. Verificar se o setup já foi concluído para evitar execuções múltiplas
	db := database.GetDB()
	var orgCount int64
	db.Model(&models.Organization{}).Count(&orgCount)
	if orgCount > 0 {
		log.Warn("Setup attempt on an already configured system.")
		c.JSON(http.StatusConflict, gin.H{"error": "O sistema já parece estar configurado. Setup não pode ser executado novamente."})
		return
	}

	// 2. Validar o payload da requisição
	var payload SetupPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Error("Invalid setup payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload da requisição inválido: " + err.Error()})
		return
	}

	// 3. Executar as migrações do banco de dados
	log.Info("Starting database migrations...")
	if err := seeders.RunMigrations(db); err != nil {
		log.Error("Failed to run database migrations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao executar as migrações do banco de dados: " + err.Error()})
		return
	}
	log.Info("Database migrations completed successfully.")

	// 4. Criar a organização
	log.Info("Creating organization", zap.String("name", payload.OrganizationName))
	org := models.Organization{
		Name: payload.OrganizationName,
	}
	if err := db.Create(&org).Error; err != nil {
		log.Error("Failed to create organization", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao criar a organização: " + err.Error()})
		return
	}
	log.Info("Organization created successfully", zap.String("org_id", org.ID.String()))

	// 5. Criar o usuário administrador
	log.Info("Creating admin user", zap.String("email", payload.AdminEmail))
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Error("Failed to hash admin password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao processar a senha do administrador."})
		return
	}

	adminUser := models.User{
		Name:           payload.AdminName,
		Email:          payload.AdminEmail,
		PasswordHash:   string(hashedPassword),
		Role:           models.RoleAdmin,
		IsActive:       true,
		OrganizationID: org.ID,
	}
	if err := db.Create(&adminUser).Error; err != nil {
		log.Error("Failed to create admin user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao criar o usuário administrador: " + err.Error()})
		return
	}
	log.Info("Admin user created successfully", zap.String("user_id", adminUser.ID.String()))

	// 6. Popular dados iniciais (se houver)
	log.Info("Seeding initial data...")
	if err := seeders.SeedInitialData(db); err != nil {
		log.Error("Failed to seed initial data", zap.Error(err))
		// Não retorna erro fatal, pois o setup principal foi concluído
	} else {
		log.Info("Initial data seeded successfully.")
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Setup concluído com sucesso!",
		"organization_id": org.ID,
		"admin_user_id":   adminUser.ID,
	})
}

// GlobalIdPResponse define a estrutura para provedores de identidade sociais globais.
type GlobalIdPResponse struct {
	Key      string `json:"key"`       // ex: "google", "github" (para uso no frontend)
	Type     string `json:"type"`      // ex: "oauth2_google", "oauth2_github"
	Name     string `json:"name"`      // ex: "Login com Google", "Login com GitHub"
	LoginURL string `json:"login_url"` // URL para iniciar o fluxo de login OAuth2
}

// ListGlobalSocialIdentityProvidersHandler retorna uma lista de provedores de identidade sociais
// configurados globalmente através de variáveis de ambiente.
func ListGlobalSocialIdentityProvidersHandler(c *gin.Context) {
	var providers []GlobalIdPResponse

	appRootURL := config.Cfg.AppRootURL
	if appRootURL == "" {
		appRootURL = "http://localhost:8080" // Fallback para desenvolvimento se não configurado via .env
		phxlog.L.Warn("APP_ROOT_URL is not configured. Social login URLs may use an insecure fallback.",
			zap.String("fallback_url", appRootURL))
	}
	// Remover quaisquer barras extras do final para evitar // no path
	appRootURL = strings.TrimSuffix(appRootURL, "/")

	// Verificar Google
	// As variáveis config.Cfg.GoogleClientID e config.Cfg.GoogleClientSecret precisam ser adicionadas à struct AppConfig
	// e carregadas a partir de variáveis de ambiente (ex: GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET).
	if config.Cfg.GoogleClientID != "" && config.Cfg.GoogleClientSecret != "" {
		providers = append(providers, GlobalIdPResponse{
			Key:      "google",
			Type:     "oauth2_google",
			Name:     "Login com Google",
			LoginURL: fmt.Sprintf("%s/auth/oauth2/google/global/login", appRootURL),
		})
	}

	// Verificar GitHub
	// As variáveis config.Cfg.GithubClientID e config.Cfg.GithubClientSecret precisam ser adicionadas.
	if config.Cfg.GithubClientID != "" && config.Cfg.GithubClientSecret != "" {
		providers = append(providers, GlobalIdPResponse{
			Key:      "github",
			Type:     "oauth2_github",
			Name:     "Login com GitHub",
			LoginURL: fmt.Sprintf("%s/auth/oauth2/github/global/login", appRootURL),
		})
	}

	c.JSON(http.StatusOK, providers)
}

// SetupStatusResponse define a resposta para o endpoint de status do setup.
type SetupStatusResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// GetSetupStatusHandler verifica e retorna o estado atual da configuração da aplicação.
func GetSetupStatusHandler(c *gin.Context) {
	db := database.GetDB()

	// 1. Verificar conexão com o DB
	sqlDB, err := db.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, SetupStatusResponse{
			Status:  "database_not_configured",
			Message: "A instância do banco de dados não está disponível.",
		})
		return
	}
	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, SetupStatusResponse{
			Status:  "database_not_connected",
			Message: "Não foi possível conectar ao banco de dados. Verifique as credenciais e a conectividade.",
		})
		return
	}

	// 2. Verificar se as migrações foram executadas
	// Uma forma simples é verificar se a tabela 'users' existe.
	if !db.Migrator().HasTable(&models.User{}) {
		c.JSON(http.StatusOK, SetupStatusResponse{
			Status:  "migrations_not_run",
			Message: "Conexão com o banco de dados OK, mas as tabelas da aplicação não foram criadas. Execute o setup.",
		})
		return
	}

	// 3. Verificar se a primeira organização e o admin foram criados
	var orgCount int64
	db.Model(&models.Organization{}).Count(&orgCount)
	if orgCount == 0 {
		c.JSON(http.StatusOK, SetupStatusResponse{
			Status:  "setup_pending_org",
			Message: "Migrações concluídas, mas a primeira organização e o usuário administrador precisam ser criados.",
		})
		return
	}

	var adminCount int64
	db.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&adminCount)
	if adminCount == 0 {
		c.JSON(http.StatusOK, SetupStatusResponse{
			Status:  "setup_pending_admin",
			Message: "Organização criada, mas o usuário administrador não foi encontrado. Complete o setup.",
		})
		return
	}

	// Se tudo estiver OK
	c.JSON(http.StatusOK, SetupStatusResponse{
		Status:  "setup_complete",
		Message: "A aplicação está configurada e pronta para uso.",
	})
}
