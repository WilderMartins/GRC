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
// ATENÇÃO: Estes dados são exemplificativos e incompletos. Para um ambiente de produção,
// é crucial popular esta seção com os controles completos e precisos de cada framework.
// A obtenção e formatação desses dados é uma tarefa de curadoria de conteúdo.
func getFrameworksData() []FrameworkData {
	return []FrameworkData{
		{
			Name: "NIST Cybersecurity Framework 2.0",
			Controls: []ControlData{
				// Govern (GV)
				{ControlID: "GV.OC-1", Description: "Papéis e responsabilidades organizacionais para cibersegurança são estabelecidos e comunicados.", Family: "Governança Organizacional (GV.OC)"},
				{ControlID: "GV.OC-2", Description: "A estratégia de cibersegurança organizacional, incluindo o apetite a risco, é aprovada e comunicada.", Family: "Governança Organizacional (GV.OC)"},
				{ControlID: "GV.RM-1", Description: "O processo de gestão de riscos da organização é usado para informar a gestão de riscos de cibersegurança.", Family: "Gestão de Riscos (GV.RM)"},
				{ControlID: "GV.SC-1", Description: "Os requisitos de cibersegurança para fornecedores são estabelecidos, comunicados e monitorados.", Family: "Gestão de Riscos da Cadeia de Suprimentos (GV.SC)"},
				// Identify (ID)
				{ControlID: "ID.AM-1", Description: "Inventário de ativos de hardware gerenciados pela organização.", Family: "Gestão de Ativos (ID.AM)"},
				{ControlID: "ID.AM-2", Description: "Inventário de ativos de software e serviços gerenciados pela organização.", Family: "Gestão de Ativos (ID.AM)"},
				{ControlID: "ID.RA-1", Description: "Vulnerabilidades em ativos são identificadas e documentadas.", Family: "Avaliação de Riscos (ID.RA)"},
				// Protect (PR)
				{ControlID: "PR.AA-1", Description: "Acesso a ativos físicos é gerenciado e protegido.", Family: "Gestão de Identidade e Controle de Acesso (PR.AA)"},
				{ControlID: "PR.AA-2", Description: "Acesso a ativos lógicos é gerenciado e protegido.", Family: "Gestão de Identidade e Controle de Acesso (PR.AA)"},
				{ControlID: "PR.AT-1", Description: "Todos os usuários são informados e treinados.", Family: "Conscientização e Treinamento (PR.AT)"},
				{ControlID: "PR.DS-1", Description: "Dados em repouso são protegidos.", Family: "Segurança de Dados (PR.DS)"},
				// Detect (DE)
				{ControlID: "DE.CM-1", Description: "Redes são monitoradas para detectar eventos de cibersegurança.", Family: "Monitoramento Contínuo (DE.CM)"},
				// Respond (RS)
				{ControlID: "RS.RP-1", Description: "Plano de resposta a incidentes é executado durante ou após um evento.", Family: "Planejamento de Resposta (RS.RP)"},
				// Recover (RC)
				{ControlID: "RC.RP-1", Description: "Plano de recuperação é executado durante ou após um evento de cibersegurança.", Family: "Planejamento de Recuperação (RC.RP)"},
			},
		},
		{
			Name: "CIS Critical Security Controls v8",
			Controls: []ControlData{
				{ControlID: "CIS-1.1", Description: "Estabelecer e Manter Inventário Detalhado de Ativos Empresariais.", Family: "CIS Control 1: Inventário e Controle de Ativos Empresariais"},
				{ControlID: "CIS-1.2", Description: "Endereçar Ativos Não Autorizados.", Family: "CIS Control 1: Inventário e Controle de Ativos Empresariais"},
				{ControlID: "CIS-2.1", Description: "Estabelecer e Manter Inventário Detalhado de Ativos de Software.", Family: "CIS Control 2: Inventário e Controle de Ativos de Software"},
				{ControlID: "CIS-2.2", Description: "Garantir que Software Não Autorizado Seja Removido ou Colocado em Quarentena.", Family: "CIS Control 2: Inventário e Controle de Ativos de Software"},
				{ControlID: "CIS-3.1", Description: "Estabelecer e Manter um Processo de Gerenciamento Seguro de Configuração de Ativos Empresariais e Software.", Family: "CIS Control 3: Proteção de Dados"}, // Nota: CIS v8 reorganizou, 3.1 é sobre config.
				{ControlID: "CIS-3.3", Description: "Configurar Listas de Controle de Acesso à Rede.", Family: "CIS Control 3: Proteção de Dados"}, // Exemplo, verificar numeração exata
				{ControlID: "CIS-4.1", Description: "Estabelecer e Manter um Processo de Gerenciamento Seguro de Configuração para Dispositivos de Rede, como Firewalls e Roteadores.", Family: "CIS Control 4: Configuração Segura de Ativos e Software Empresariais"},
				{ControlID: "CIS-7.1", Description: "Estabelecer e Manter um Processo de Gerenciamento de Vulnerabilidades.", Family: "CIS Control 7: Gerenciamento Contínuo de Vulnerabilidades"},
			},
		},
		{
			Name: "ISO/IEC 27001:2022 (Anexo A)",
			Controls: []ControlData{
				// Controles Organizacionais
				{ControlID: "A.5.1", Description: "Políticas para segurança da informação.", Family: "5.1 Políticas para segurança da informação"},
				{ControlID: "A.5.2", Description: "Papéis e responsabilidades em segurança da informação.", Family: "5.2 Papéis e responsabilidades em segurança da informação"},
				{ControlID: "A.5.15", Description: "Acesso a informações e outros ativos associados.", Family: "5.15 Gerenciamento de acesso"}, // Reagrupado em 2022
				{ControlID: "A.5.23", Description: "Segurança da informação para uso de serviços em nuvem.", Family: "5.23 Segurança da informação para uso de serviços em nuvem"},
				// Controles de Pessoas
				{ControlID: "A.6.3", Description: "Termos e condições de emprego.", Family: "6.3 Termos e condições de emprego"},
				// Controles Físicos
				{ControlID: "A.7.4", Description: "Monitoramento da segurança física.", Family: "7.4 Monitoramento da segurança física"},
				// Controles Tecnológicos
				{ControlID: "A.8.1", Description: "Equipamento do usuário final.", Family: "8.1 Equipamento do usuário final"}, // Anteriormente A.11.2.1, A.6.2.1 etc.
				{ControlID: "A.8.2", Description: "Direitos de acesso privilegiado.", Family: "8.2 Direitos de acesso privilegiado"},
				{ControlID: "A.8.9", Description: "Configuração.", Family: "8.9 Configuração"}, // Novo em 2022
				{ControlID: "A.8.16", Description: "Monitoramento de atividades.", Family: "8.16 Monitoramento de atividades"},
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
