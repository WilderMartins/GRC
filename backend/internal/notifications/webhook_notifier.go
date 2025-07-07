package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxWebhookRetries = 3
const webhookRetryDelay = 5 * time.Second

// GoogleChatMessage é a estrutura do payload para webhooks do Google Chat.
type GoogleChatMessage struct {
	Text string `json:"text"`
}

// SendWebhookNotification envia uma notificação para uma URL de webhook.
// Ele tenta algumas vezes em caso de falha.
func SendWebhookNotification(webhookURL string, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	var lastErr error
	for i := 0; i < maxWebhookRetries; i++ {
		req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
		if err != nil {
			// Este erro é improvável se a URL e o payload estiverem corretos
			log.Printf("Error creating webhook request to %s (try %d/%d): %v\n", webhookURL, i+1, maxWebhookRetries, err)
			lastErr = fmt.Errorf("failed to create request: %w", err)
			time.Sleep(webhookRetryDelay)
			continue
		}
		req.Header.Set("Content-Type", "application/json; charset=UTF-8")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error sending webhook to %s (try %d/%d): %v\n", webhookURL, i+1, maxWebhookRetries, err)
			lastErr = fmt.Errorf("request failed: %w", err)
			time.Sleep(webhookRetryDelay)
			continue
		}

		// Sucesso se status code for 2xx
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Printf("Webhook sent successfully to %s (status: %s)\n", webhookURL, resp.Status)
			// É importante fechar o corpo da resposta para reutilizar a conexão
			if resp.Body != nil {
				resp.Body.Close()
			}
			return nil
		}

		// Ler corpo da resposta em caso de erro para logging
		var bodyText []byte
		if resp.Body != nil {
			bodyText, _ = json.Marshal(resp.Body) // Simplificado, idealmente ler o corpo
			resp.Body.Close()
		}

		log.Printf("Webhook to %s failed (try %d/%d) - Status: %s, Body: %s\n", webhookURL, i+1, maxWebhookRetries, resp.Status, string(bodyText))
		lastErr = fmt.Errorf("request failed with status %s", resp.Status)

		// Não tentar novamente para erros 4xx (exceto talvez 429 Too Many Requests, se quisermos tratar)
        if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
            break
        }
		time.Sleep(webhookRetryDelay)
	}
	return fmt.Errorf("failed to send webhook to %s after %d retries: %w", webhookURL, maxWebhookRetries, lastErr)
}

// NotifyRiskEvent busca webhooks relevantes e envia notificações.
func NotifyRiskEvent(orgID uuid.UUID, risk models.Risk, eventType models.WebhookEventType) {
	db := database.GetDB()
	var webhooks []models.WebhookConfiguration

	// Busca webhooks ativos para a organização e tipo de evento
	// A query precisa verificar se EventTypes (string separada por vírgula) contém o eventType.
	// Usar LIKE é uma forma simples, mas pode ter falsos positivos se um tipo de evento for substring de outro.
	// Ex: "risk_created" LIKE "%risk_create%" (falso positivo).
	// Idealmente, se EventTypes fosse um array no DB ou JSONB, a query seria mais robusta.
	// Com string separada por vírgula:
	// 1. Buscar todos e filtrar na aplicação (ineficiente para muitos webhooks)
	// 2. Usar funções específicas do DB se disponíveis (ex: string_to_array no PostgreSQL)
	// 3. Usar LIKE com vírgulas delimitadoras: (',' || event_types || ',') LIKE '%,event_type,%'
	// Para simplificar, vamos usar LIKE, mas cientes da limitação.

	// Construindo a condição LIKE para encontrar o eventType na string EventTypes
	// Isso garante que "event" não corresponda a "long_event" ou "event_short"
	// Ex: ",risk_created," LIKE "%,risk_created,%"
	// Ou "risk_created," LIKE "risk_created,%" (início)
	// Ou ",risk_created" LIKE "%,risk_created" (fim)
	// Ou "risk_created" = "risk_created" (único)
	eventPattern := string(eventType)
	// Usar GORM com OR e LIKE para cobrir os casos:
	// eventType
	// eventType,...
	// ...,eventType,...
	// ...,eventType
	err := db.Where("organization_id = ? AND is_active = ?", orgID, true).
		Where(
			db.Where("event_types = ?", eventPattern). // Exatamente o evento
				Or("event_types LIKE ?", eventPattern+",%"). // Começa com o evento
				Or("event_types LIKE ?", "%,"+eventPattern+",%"). // Contém o evento no meio
				Or("event_types LIKE ?", "%,"+eventPattern), // Termina com o evento
		).
		Find(&webhooks).Error

	if err != nil {
		log.Printf("Error fetching webhooks for org %s, event %s: %v\n", orgID, eventType, err)
		return
	}

	if len(webhooks) == 0 {
		return // Nenhum webhook configurado para este evento/organização
	}

	var messageText string
	// TODO: Construir URL real para o risco no frontend (requer configuração de URL base do frontend)
	riskURLPlaceholder := fmt.Sprintf("http://phoenix-grc-frontend.example.com/risks/%s", risk.ID.String())

	switch eventType {
	case models.EventTypeRiskCreated:
		messageText = fmt.Sprintf("🚀 Novo risco criado: *%s*\nDescrição: %s\nImpacto: %s, Probabilidade: %s\nLink: %s",
			risk.Title, risk.Description, risk.Impact, risk.Probability, riskURLPlaceholder)
	case models.EventTypeRiskStatusChanged:
		messageText = fmt.Sprintf("🔄 Status do risco '*%s*' alterado para: *%s*\nLink: %s",
			risk.Title, risk.Status, riskURLPlaceholder)
	default:
		log.Printf("Tipo de evento desconhecido para notificação de risco: %s\n", eventType)
		return
	}

	payload := GoogleChatMessage{Text: messageText}

	for _, wh := range webhooks {
		// Checagem mais precisa do EventTypes aqui, já que o LIKE do SQL pode não ser perfeito
		// se não usarmos os delimitadores de vírgula na query.
		// A query SQL já foi ajustada para ser mais precisa com os delimitadores.

		// Disparar em uma goroutine para não bloquear
		go func(webhookURL string, pld interface{}) {
			if err := SendWebhookNotification(webhookURL, pld); err != nil {
				log.Printf("Falha ao enviar notificação de webhook para %s: %v\n", webhookURL, err)
			}
		}(wh.URL, payload)
	}
}

// SendEmailNotification simula o envio de um email.
// No futuro, isso seria integrado com um serviço de email transacional.
func SendEmailNotification(toEmail string, subject string, body string) error {
	if toEmail == "" {
		return fmt.Errorf("destinatário do email (toEmail) não pode ser vazio")
	}
	// Simulação
	log.Printf("--- SIMULAÇÃO DE ENVIO DE EMAIL ---")
	log.Printf("Para: %s", toEmail)
	log.Printf("Assunto: %s", subject)
	log.Printf("Corpo:\n%s", body)
	log.Printf("--- FIM DA SIMULAÇÃO DE ENVIO DE EMAIL ---")
	return nil // Simula sucesso
}

// NotifyUserByEmail envia uma notificação por email para um usuário específico.
// Busca o email do usuário pelo ID.
func NotifyUserByEmail(userID uuid.UUID, subject string, body string) {
	if userID == uuid.Nil {
		log.Println("Tentativa de notificar usuário por email com ID nulo.")
		return
	}
	db := database.GetDB()
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		log.Printf("Erro ao buscar usuário %s para notificação por email: %v\n", userID, err)
		return
	}
	if user.Email == "" {
		log.Printf("Usuário %s não possui email cadastrado para notificação.\n", userID)
		return
	}

	go func(email, subj, bdy string) {
		if err := SendEmailNotification(email, subj, bdy); err != nil {
			log.Printf("Falha ao enviar notificação por email simulada para %s: %v\n", email, err)
		}
	}(user.Email, subject, body)
}
