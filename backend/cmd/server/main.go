package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"github.com/google/uuid"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("--- Phoenix GRC Setup ---")

	// 1. Coletar credenciais do banco de dados
	fmt.Println("\n--- Database Configuration ---")
	fmt.Print("Enter Database Host (e.g., localhost): ")
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

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	// 2. Conectar ao banco de dados
	if err := database.ConnectDB(dsn); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	fmt.Println("Successfully connected to the database.")

	// 3. Executar migrações
	if err := database.MigrateDB(); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}
	fmt.Println("Database migrations completed successfully.")

	// 4. Criar a primeira organização
	fmt.Println("\n--- Organization Setup ---")
	fmt.Print("Enter the name for the first organization: ")
	orgName, _ := reader.ReadString('\n')
	orgName = strings.TrimSpace(orgName)

	organization := models.Organization{
		Name: orgName,
		// ID will be set by BeforeCreate hook
	}
	db := database.GetDB()
	if err := db.Create(&organization).Error; err != nil {
		log.Fatalf("Failed to create organization: %v", err)
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
		log.Fatalf("Failed to hash password: %v", err)
	}

	adminUser := models.User{
		OrganizationID: organization.ID, // Link admin to the created organization
		Name:           adminName,
		Email:          adminEmail,
		PasswordHash:   string(hashedPassword),
		Role:           models.RoleAdmin,
		// ID will be set by BeforeCreate hook
	}

	if err := db.Create(&adminUser).Error; err != nil {
		log.Fatalf("Failed to create admin user: %v", err)
	}
	fmt.Printf("Admin user '%s' created successfully.\n", adminUser.Email)

	fmt.Println("\n--- Setup Complete ---")
	fmt.Println("Phoenix GRC initial setup is complete. You can now run the main application.")
}
