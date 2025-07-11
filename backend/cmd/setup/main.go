package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	"phoenixgrc/backend/internal/seeders"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term" // For password masking
)

// readInput reads a line of text from the console.
func readInput(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// readPassword reads a password from the console, masking the input.
func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // Add a newline after password input
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytePassword)), nil
}

func RunSetup() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("--- Phoenix GRC Setup ---")

	// 1. Database Configuration
	fmt.Println("\n--- Database Configuration ---")
	dbHost := readInput(reader, "Enter Database Host (e.g., localhost or 'db' if using docker-compose): ")
	dbPort := readInput(reader, "Enter Database Port (e.g., 5432): ")
	dbUser := readInput(reader, "Enter Database User: ")
	dbPassword, err := readPassword("Enter Database Password: ")
	if err != nil {
		log.Fatalf("Failed to read database password: %v", err)
	}
	dbName := readInput(reader, "Enter Database Name: ")
	dbSSLMode := readInput(reader, "Enter Database SSL Mode (e.g., disable, require): ")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	fmt.Println("Connecting to database...")
	if err := database.ConnectDB(dsn); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	fmt.Println("Successfully connected to the database.")

	// 2. Database Migrations
	fmt.Println("\n--- Running Database Migrations ---")
	// Construir a dbURL para golang-migrate
	// Formato: postgresql://user:password@host:port/dbname?sslmode=disable
	// Nota: o usuário e senha podem conter caracteres especiais que precisam ser URL-encoded.
	// No entanto, para input manual, geralmente não é um problema, mas para produção, sim.
	// A biblioteca lib/pq (usada por database/sql e GORM) lida com isso na DSN.
	// O golang-migrate também usa lib/pq internamente para o driver postgres.
	// A DSN simples que construímos para o GORM deve ser aceitável para o driver do migrate se
	// não houver caracteres muito exóticos.
	// A forma mais segura de construir a URL para migrate é usar `url.URL` e `url.QueryEscape`.
	// Por simplicidade aqui, vou usar uma formatação direta, assumindo que as credenciais não são problemáticas.

	// A dbURL para golang-migrate é um pouco diferente da DSN do GORM.
	// Ex: "postgres://postgres:password@localhost:5432/phoenix_grc_dev?sslmode=disable"
	// Adicionar o schema padrão (public) para a tabela schema_migrations, a menos que outro seja especificado.
	// Por agora, vamos manter simples e assumir que schema_migrations vai para o 'public' ou o search_path padrão do usuário.
	// Se `DB_SCHEMA` for uma variável de ambiente relevante, ela deveria ser usada aqui.
	// Para o migrate, o schema é geralmente controlado pelo search_path do usuário ou pode ser
	// especificado na URL se o driver suportar (ex: ?search_path=myschema).
	// O driver postgres do migrate não parece ter um suporte explícito a `search_path` na URL de forma padrão.
	// Ele opera no search_path padrão da conexão.

	// A DSN que já temos é `host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC`
	// O migrate espera algo como `postgres://user:pass@host:port/dbname?sslmode=val`
	// Vamos construir essa URL.
	migrateDbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser,     // Assumindo que não precisa de URL encoding para setup manual
		dbPassword, // Assumindo que não precisa de URL encoding para setup manual
		dbHost,
		dbPort,
		dbName,
		dbSSLMode,
	)

	if err := database.MigrateDB(migrateDbURL); err != nil {
		log.Fatalf("Database migration process failed: %v", err)
	}
	fmt.Println("Database migrations completed successfully.")

	// 3. Seeding Initial Data (Audit Frameworks)
	fmt.Println("\n--- Seeding Audit Frameworks and Controls ---")
	db := database.GetDB()
	if db == nil {
		log.Fatal("Failed to get database instance for seeding.")
	}
	if err := seeders.SeedAuditFrameworksAndControls(db); err != nil {
		log.Fatalf("Failed to seed audit frameworks and controls: %v", err)
	}
	fmt.Println("Audit frameworks and controls seeded successfully.")

	// 4. First Organization Setup
	fmt.Println("\n--- Creating First Organization ---")
	orgName := readInput(reader, "Enter the name for the first organization: ")
	if orgName == "" {
		orgName = "Default Organization" // Provide a default if empty
		fmt.Printf("No organization name entered, using default: %s\n", orgName)
	}

	organization := models.Organization{
		Name: orgName,
	}
	if err := db.Create(&organization).Error; err != nil {
		log.Fatalf("Failed to create organization: %v", err)
	}
	fmt.Printf("Organization '%s' created successfully with ID: %s\n", organization.Name, organization.ID)

	// 5. Admin User Creation
	fmt.Println("\n--- Creating Admin User ---")
	adminName := readInput(reader, "Enter Admin User Name: ")
	adminEmail := readInput(reader, "Enter Admin User Email: ")

	var adminPassword string
	var adminPasswordConfirm string
	for {
		adminPassword, err = readPassword("Enter Admin User Password: ")
		if err != nil {
			log.Fatalf("Failed to read admin password: %v", err)
		}
		if adminPassword == "" {
			fmt.Println("Password cannot be empty. Please try again.")
			continue
		}
		adminPasswordConfirm, err = readPassword("Confirm Admin User Password: ")
		if err != nil {
			log.Fatalf("Failed to read admin password confirmation: %v", err)
		}
		if adminPassword == adminPasswordConfirm {
			break
		}
		fmt.Println("Passwords do not match. Please try again.")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	adminUser := models.User{
		OrganizationID: organization.ID,
		Name:           adminName,
		Email:          adminEmail,
		PasswordHash:   string(hashedPassword),
		Role:           models.RoleAdmin, // Assuming models.RoleAdmin is defined
		IsActive:       true,
	}

	if err := db.Create(&adminUser).Error; err != nil {
		log.Fatalf("Failed to create admin user: %v. Ensure email is unique.", err)
	}
	fmt.Printf("Admin user '%s' created successfully.\n", adminUser.Email)

	// 6. Completion
	fmt.Println("\n--- Phoenix GRC Setup Complete! ---")
	fmt.Println("You can now start the main application server.")
}
