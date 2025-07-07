package seeders

import (
	"fmt"
	"log"
	"phoenixgrc/backend/internal/models"

	"gorm.io/gorm"
)

// FrameworkData define a estrutura para os dados de um framework e seus controles.
type FrameworkData struct {
	Name     string
	Controls []ControlData
}

// ControlData define a estrutura para os dados de um controle.
type ControlData struct {
	ControlID   string
	Description string
	Family      string
}

// getFrameworksData retorna os dados dos frameworks a serem semeados.
// Estes dados são simplificados. Em uma aplicação real, seriam mais extensos e possivelmente carregados de arquivos.
func getFrameworksData() []FrameworkData {
	return []FrameworkData{
		{
			Name: "NIST CSF 2.0 (Exemplo)", // Usar nomes oficiais e versões completas
			Controls: []ControlData{
				{ControlID: "GV.OC-1", Description: "Estabelecer e comunicar papéis e responsabilidades de cibersegurança.", Family: "Governança (GV)"},
				{ControlID: "GV.RM-1", Description: "Estabelecer, comunicar, e coordenar programa de gestão de risco de cibersegurança.", Family: "Governança (GV)"},
				{ControlID: "ID.AM-1", Description: "Inventariar ativos de hardware e software.", Family: "Identificar (ID)"},
				{ControlID: "PR.AT-1", Description: "Prover treinamento de conscientização em segurança.", Family: "Proteger (PR)"},
			},
		},
		{
			Name: "CIS Controls v8 (Exemplo)",
			Controls: []ControlData{
				{ControlID: "CIS-1.1", Description: "Estabelecer e Manter Inventário Detalhado de Ativos Empresariais.", Family: "Inventário e Controle de Ativos Empresariais"},
				{ControlID: "CIS-2.1", Description: "Estabelecer e Manter Inventário Detalhado de Ativos de Software.", Family: "Inventário e Controle de Ativos de Software"},
				{ControlID: "CIS-3.1", Description: "Estabelecer e Manter um Processo de Gerenciamento Seguro de Configuração.", Family: "Proteção de Dados"},
			},
		},
		{
			Name: "ISO 27001:2022 (Exemplo de Controles do Anexo A)",
			Controls: []ControlData{
				{ControlID: "A.5.1", Description: "Políticas para segurança da informação.", Family: "Controles Organizacionais"},
				{ControlID: "A.5.15", Description: "Acesso a informações e outros ativos associados.", Family: "Controles Organizacionais"},
				{ControlID: "A.8.1", Description: "Ativos de informação.", Family: "Controles de Ativos"},
				{ControlID: "A.8.9", Description: "Gerenciamento de acesso.", Family: "Controles de Acesso"},
			},
		},
	}
}

// SeedAuditFrameworksAndControls popula o banco de dados com frameworks e controles de auditoria.
func SeedAuditFrameworksAndControls(db *gorm.DB) error {
	frameworksData := getFrameworksData()

	for _, fd := range frameworksData {
		// Tenta encontrar o framework pelo nome para evitar duplicatas
		var existingFramework models.AuditFramework
		err := db.Where("name = ?", fd.Name).First(&existingFramework).Error

		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("erro ao verificar framework existente %s: %w", fd.Name, err)
		}

		frameworkToSeed := models.AuditFramework{Name: fd.Name}

		if err == gorm.ErrRecordNotFound { // Framework não existe, cria novo
			log.Printf("Semeando framework: %s", fd.Name)
			if result := db.Create(&frameworkToSeed); result.Error != nil {
				return fmt.Errorf("erro ao semear framework %s: %w", fd.Name, result.Error)
			}
		} else { // Framework já existe, usa o ID existente
			log.Printf("Framework %s já existe, pulando criação do framework.", fd.Name)
			frameworkToSeed.ID = existingFramework.ID
		}

		// Semear controles para este framework
		for _, cd := range fd.Controls {
			var existingControl models.AuditControl
			// Verifica se o controle já existe para este framework e ControlID
			errCtrl := db.Where("framework_id = ? AND control_id = ?", frameworkToSeed.ID, cd.ControlID).First(&existingControl).Error

			if errCtrl != nil && errCtrl != gorm.ErrRecordNotFound {
				return fmt.Errorf("erro ao verificar controle existente %s para framework %s: %w", cd.ControlID, fd.Name, errCtrl)
			}

			if errCtrl == gorm.ErrRecordNotFound { // Controle não existe, cria novo
				log.Printf("Semeando controle: %s (%s) para framework: %s", cd.ControlID, cd.Description, fd.Name)
				controlToSeed := models.AuditControl{
					FrameworkID: frameworkToSeed.ID,
					ControlID:   cd.ControlID,
					Description: cd.Description,
					Family:      cd.Family,
				}
				if result := db.Create(&controlToSeed); result.Error != nil {
					return fmt.Errorf("erro ao semear controle %s para framework %s: %w", cd.ControlID, fd.Name, result.Error)
				}
			} else {
				log.Printf("Controle %s para framework %s já existe, pulando.", cd.ControlID, fd.Name)
			}
		}
	}
	log.Println("Semeação de frameworks e controles de auditoria concluída.")
	return nil
}
