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
	if err := database.MigrateDB(); err != nil {
		// database.MigrateDB() already logs the error
		log.Fatalf("Database migration process failed.")
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
