package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"phoenixgrc/backend/internal/auth"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/handlers"
	"phoenixgrc/backend/internal/models"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	// "github.com/google/uuid" // uuid.New() is used in models' BeforeCreate
)

func runSetup() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("--- Phoenix GRC Setup ---")

	// 1. Coletar credenciais do banco de dados
	fmt.Println("\n--- Database Configuration ---")
	fmt.Print("Enter Database Host (e.g., localhost or 'db' if using docker-compose): ")
	dbHost, _ := reader.ReadString('\n')
	dbHost = strings.TrimSpace(dbHost)

	fmt.Print("Enter Database Port (e.g., 5432): ")
	dbPort, _ := reader.ReadString('\n')
	dbPort = strings.TrimSpace(dbPort)

	fmt.Print("Enter Database User: ")
	dbUser, _ := reader.ReadString('\n')
	dbUser = strings.TrimSpace(dbUser)

	fmt.Print("Enter Database Password: ")
	dbPassword, _ := reader.ReadString('\n')
	dbPassword = strings.TrimSpace(dbPassword)

	fmt.Print("Enter Database Name: ")
	dbName, _ := reader.ReadString('\n')
	dbName = strings.TrimSpace(dbName)

	fmt.Print("Enter Database SSL Mode (e.g., disable, require): ")
	dbSSLMode, _ := reader.ReadString('\n')
	dbSSLMode = strings.TrimSpace(dbSSLMode)

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	// 2. Conectar ao banco de dados
	if err := database.ConnectDB(dsn); err != nil {
		log.Fatalf("Failed to connect to database during setup: %v", err)
	}
	fmt.Println("Successfully connected to the database during setup.")

	// 3. Executar migrações
	if err := database.MigrateDB(); err != nil {
		log.Fatalf("Failed to run database migrations during setup: %v", err)
	}
	fmt.Println("Database migrations completed successfully during setup.")

	// 4. Criar a primeira organização
	fmt.Println("\n--- Organization Setup ---")
	fmt.Print("Enter the name for the first organization: ")
	orgName, _ := reader.ReadString('\n')
	orgName = strings.TrimSpace(orgName)

	organization := models.Organization{
		Name: orgName,
	}
	db := database.GetDB()
	if err := db.Create(&organization).Error; err != nil {
		log.Fatalf("Failed to create organization during setup: %v", err)
	}
	fmt.Printf("Organization '%s' created successfully with ID: %s\n", organization.Name, organization.ID)

	// 5. Criar o primeiro usuário administrador
	fmt.Println("\n--- Admin User Setup ---")
	fmt.Print("Enter Admin User Name: ")
	adminName, _ := reader.ReadString('\n')
	adminName = strings.TrimSpace(adminName)

	fmt.Print("Enter Admin User Email: ")
	adminEmail, _ := reader.ReadString('\n')
	adminEmail = strings.TrimSpace(adminEmail)

	fmt.Print("Enter Admin User Password: ")
	adminPassword, _ := reader.ReadString('\n')
	adminPassword = strings.TrimSpace(adminPassword)

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password during setup: %v", err)
	}

	adminUser := models.User{
		OrganizationID: organization.ID,
		Name:           adminName,
		Email:          adminEmail,
		PasswordHash:   string(hashedPassword),
		Role:           models.RoleAdmin,
	}

	if err := db.Create(&adminUser).Error; err != nil {
		log.Fatalf("Failed to create admin user during setup: %v", err)
	}
	fmt.Printf("Admin user '%s' created successfully during setup.\n", adminUser.Email)

	fmt.Println("\n--- Setup Complete ---")
	fmt.Println("Phoenix GRC initial setup is complete.")
}

func startServer() {
	// Initialize JWT
	if err := auth.InitializeJWT(); err != nil {
		log.Fatalf("Failed to initialize JWT: %v", err)
	}
	log.Println("JWT Initialized.")

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
		log.Fatal("Database credentials (POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB) must be set in environment variables or .env file for the server.")
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	if err := database.ConnectDB(dsn); err != nil {
		log.Fatalf("Failed to connect to database for the server: %v", err)
	}
	log.Println("Database connection established for the server.")

	router := gin.Default()

	// Public routes
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
		authRoutes.POST("/login", handlers.LoginHandler)
	}

	// Protected API v1 routes
	apiV1 := router.Group("/api/v1")
	apiV1.Use(auth.AuthMiddleware()) // Apply JWT auth middleware to this group
	{
		// Example protected route
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

		// Risk Management Routes
		riskRoutes := apiV1.Group("/risks")
		{
			riskRoutes.POST("", handlers.CreateRiskHandler)
			riskRoutes.GET("", handlers.ListRisksHandler)
			riskRoutes.GET("/:riskId", handlers.GetRiskHandler)
			riskRoutes.PUT("/:riskId", handlers.UpdateRiskHandler)
			riskRoutes.DELETE("/:riskId", handlers.DeleteRiskHandler)
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
		runSetup()
	} else {
		startServer()
	}
}
