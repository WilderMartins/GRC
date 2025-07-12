package seeders

import (
	"phoenixgrc/backend/internal/models"
	"gorm.io/gorm"
	"log" // Usar log padrão para seeders, pois o contexto pode ser simples
)

// SeedC2M2Data popula o banco de dados com os domínios e práticas do C2M2.
func SeedC2M2Data(db *gorm.DB) error {
	log.Println("Seeding C2M2 Domains and Practices...")

	// --- DOMAINS ---
	domains := []models.C2M2Domain{
		{Name: "Risk Management", Code: "RM"},
		{Name: "Threat and Vulnerability Management", Code: "TVM"},
		{Name: "Situational Awareness", Code: "SA"},
		{Name: "Identity and Access Management", Code: "IAM"},
		{Name: "Incident Response", Code: "IR"},
		// Adicionar os outros 5 domínios se necessário...
	}

	domainMap := make(map[string]models.C2M2Domain)
	for _, domain := range domains {
		// Usar FirstOrCreate para evitar duplicatas em execuções repetidas
		result := db.Where(models.C2M2Domain{Code: domain.Code}).FirstOrCreate(&domain)
		if result.Error != nil {
			log.Printf("Error seeding C2M2 domain '%s': %v", domain.Name, result.Error)
			continue // Continuar mesmo se um falhar
		}
		domainMap[domain.Code] = domain
	}
	log.Println("C2M2 Domains seeded.")

	// --- PRACTICES ---
	practices := []models.C2M2Practice{
		// Risk Management (RM)
		{DomainID: domainMap["RM"].ID, Code: "RM.1.1", Description: "Establish and maintain a risk management strategy that is approved by senior leadership and communicated to organizational personnel.", TargetMIL: 1},
		{DomainID: domainMap["RM"].ID, Code: "RM.2.1", Description: "Manage risks to the organization, including supply chain risks, consistent with the risk management strategy.", TargetMIL: 2},
		{DomainID: domainMap["RM"].ID, Code: "RM.3.1", Description: "Evaluate and improve the risk management strategy and activities based on the results of performance monitoring and the changing threat environment.", TargetMIL: 3},

		// Threat and Vulnerability Management (TVM)
		{DomainID: domainMap["TVM"].ID, Code: "TVM.1.1", Description: "Identify, assess, and mitigate vulnerabilities on organizational assets.", TargetMIL: 1},
		{DomainID: domainMap["TVM"].ID, Code: "TVM.1.2", Description: "Receive, assess, and act on threat and vulnerability information from internal and external sources.", TargetMIL: 1},
		{DomainID: domainMap["TVM"].ID, Code: "TVM.2.1", Description: "Perform proactive activities to identify previously undiscovered threats and vulnerabilities.", TargetMIL: 2},
		{DomainID: domainMap["TVM"].ID, Code: "TVM.3.1", Description: "Share internally and externally discovered threats and vulnerabilities with external entities, such as sector-specific Information Sharing and Analysis Centers (ISACs).", TargetMIL: 3},

		// Situational Awareness (SA)
		{DomainID: domainMap["SA"].ID, Code: "SA.1.1", Description: "Maintain an inventory of assets, including hardware, software, and data.", TargetMIL: 1},
		{DomainID: domainMap["SA"].ID, Code: "SA.1.2", Description: "Maintain an inventory of network connections, including those to external entities.", TargetMIL: 1},
		{DomainID: domainMap["SA"].ID, Code: "SA.2.1", Description: "Monitor and analyze security logs and other sensor data to detect and respond to security events.", TargetMIL: 2},
	}

	for _, practice := range practices {
		// Usar FirstOrCreate para evitar duplicatas em execuções repetidas
		result := db.Where(models.C2M2Practice{Code: practice.Code}).FirstOrCreate(&practice)
		if result.Error != nil {
			log.Printf("Error seeding C2M2 practice '%s': %v", practice.Code, result.Error)
		}
	}
	log.Println("C2M2 Practices seeded.")
	return nil
}
