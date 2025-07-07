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

// SendEmailNotification envia um email usando o DefaultEmailNotifier se configurado,
// caso contrário, simula o envio com log.
// Agora aceita htmlBody e textBody.
func SendEmailNotification(toEmail string, subject string, htmlBody string, textBody string) error {
	if toEmail == "" {
		return fmt.Errorf("destinatário do email (toEmail) não pode ser vazio")
	}

	if DefaultEmailNotifier != nil {
		// Envia email real usando o notificador configurado
		return DefaultEmailNotifier.SendEmail(toEmail, subject, htmlBody, textBody)
	}

	// Fallback para simulação se DefaultEmailNotifier não estiver configurado
	log.Printf("--- SIMULAÇÃO DE ENVIO DE EMAIL (Fallback) ---")
	log.Printf("Para: %s", toEmail)
	log.Printf("Assunto: %s", subject)
	if textBody != "" {
		log.Printf("Corpo (Texto):\n%s", textBody)
	}
	if htmlBody != "" {
		log.Printf("Corpo (HTML):\n%s", htmlBody)
	}
	log.Printf("--- FIM DA SIMULAÇÃO DE ENVIO DE EMAIL (Fallback) ---")
	return nil // Simula sucesso
}

// NotifyUserByEmail constrói e envia uma notificação por email para um usuário específico.
// Busca o email do usuário pelo ID.
// O corpo do email é passado como textBody, uma versão HTML simples pode ser gerada ou passada.
func NotifyUserByEmail(userID uuid.UUID, subject string, textBody string) { // Alterado para aceitar textBody
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

	// Por enquanto, vamos enviar apenas a versão em texto.
	// Uma versão HTML simples poderia ser: fmt.Sprintf("<p>%s</p>", strings.ReplaceAll(textBody, "\n", "<br>"))
	htmlBody := "" // Deixar vazio por enquanto, ou criar uma versão HTML simples do textBody

	go func(email, subj, txtBdy, htmlBdy string) {
		if err := SendEmailNotification(email, subj, htmlBdy, txtBdy); err != nil {
			log.Printf("Falha ao enviar notificação por email para %s: %v\n", email, err)
		}
	}(user.Email, subject, textBody, htmlBody)
}

// --- Interface e Implementação para Email Real ---

// EmailNotifier define uma interface para serviços de envio de email.
type EmailNotifier interface {
	SendEmail(toEmail, subject, htmlBody, textBody string) error
}

// SESEmailNotifier implementa EmailNotifier usando AWS SES V2.
type SESEmailNotifier struct {
	client      *sesv2.Client
	senderEmail string
}

var cfg aws.Config // Variável de configuração AWS carregada globalmente para o pacote
var sesInitializationError error

// InitializeAWSSession carrega a configuração da AWS.
// Deve ser chamado uma vez durante a inicialização da aplicação.
func InitializeAWSSession() error {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		sesInitializationError = fmt.Errorf("AWS_REGION não está configurada")
		log.Printf("AVISO: %v. O envio de emails reais estará desabilitado.", sesInitializationError)
		return nil // Não retornar erro fatal, permite que a app continue sem email
	}

	// Carrega a configuração padrão da AWS.
	// Isso tentará usar as credenciais do ambiente (variáveis de ambiente, shared config, IAM role).
	var err error
	cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		sesInitializationError = fmt.Errorf("falha ao carregar configuração AWS: %w", err)
		log.Printf("AVISO: %v. O envio de emails reais estará desabilitado.", sesInitializationError)
		return nil
	}
	log.Println("Sessão AWS inicializada com sucesso para a região:", region)
	return nil
}


// NewSESEmailNotifier cria uma nova instância de SESEmailNotifier.
func NewSESEmailNotifier() (*SESEmailNotifier, error) {
	if sesInitializationError != nil {
		return nil, fmt.Errorf("não é possível criar SESEmailNotifier pois a sessão AWS não foi inicializada: %w", sesInitializationError)
	}
	if cfg.Region == "" { // Se InitializeAWSSession não foi chamado ou falhou silenciosamente
		return nil, fmt.Errorf("configuração AWS não carregada (região vazia). Chame InitializeAWSSession primeiro")
	}


	sender := os.Getenv("EMAIL_SENDER_ADDRESS")
	if sender == "" {
		return nil, fmt.Errorf("EMAIL_SENDER_ADDRESS não está configurado")
	}

	sesClient := sesv2.NewFromConfig(cfg)

	return &SESEmailNotifier{
		client:      sesClient,
		senderEmail: sender,
	}, nil
}

// SendEmail envia um email usando AWS SES V2.
func (s *SESEmailNotifier) SendEmail(toEmail, subject, htmlBody, textBody string) error {
	if s.client == nil {
		return fmt.Errorf("cliente SES não inicializado")
	}

	input := &sesv2.SendEmailInput{
		FromEmailAddress: &s.senderEmail,
		Destination: &types.Destination{
			ToAddresses: []string{toEmail},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data:    &subject,
					Charset: aws.String("UTF-8"),
				},
				Body: &types.Body{},
			},
			// TODO: Adicionar suporte para templates de email do SES se necessário no futuro.
			// Raw: &types.RawMessage{ Data: []byte(mimeEmail) }, // Para emails MIME complexos
		},
	}

	if textBody != "" {
		input.Content.Simple.Body.Text = &types.Content{
			Data:    &textBody,
			Charset: aws.String("UTF-8"),
		}
	}
	if htmlBody != "" {
		input.Content.Simple.Body.Html = &types.Content{
			Data:    &htmlBody,
			Charset: aws.String("UTF-8"),
		}
	}

	// Se ambos textBody e htmlBody estiverem vazios, isso será um erro.
    if textBody == "" && htmlBody == "" {
        return fmt.Errorf("o corpo do email (texto ou HTML) não pode estar vazio")
    }


	_, err := s.client.SendEmail(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("falha ao enviar email via SES: %w", err)
	}

	log.Printf("Email enviado com sucesso para %s via AWS SES (Assunto: %s)", toEmail, subject)
	return nil
}

// DefaultEmailNotifier é a instância padrão do notificador de email.
var DefaultEmailNotifier EmailNotifier

// InitEmailService inicializa o provedor de email padrão.
func InitEmailService() {
	// Primeiro, inicializa a sessão AWS (carrega config, credenciais)
	// Isso deve ser chamado apenas uma vez.
	if err := InitializeAWSSession(); err != nil {
		// O erro já foi logado por InitializeAWSSession se for crítico para a sessão em si
		// Se InitializeAWSSession retorna nil mesmo com erro interno, DefaultEmailNotifier ficará nil
	}

	// Se a sessão AWS foi carregada (ou pelo menos não houve erro fatal nela), tenta criar o notifier SES
	if sesInitializationError == nil && cfg.Region != "" {
		notifier, err := NewSESEmailNotifier()
		if err != nil {
			log.Printf("AVISO: Falha ao inicializar o AWS SES Email Notifier: %v. O envio de emails reais estará desabilitado.", err)
			// DefaultEmailNotifier permanecerá nil, e as notificações usarão o fallback de log.
		} else {
			DefaultEmailNotifier = notifier
			log.Println("AWS SES Email Notifier inicializado e configurado como padrão.")
		}
	} else {
		log.Println("AVISO: Sessão AWS não inicializada ou região não configurada. AWS SES Email Notifier não será ativado. O envio de emails reais estará desabilitado.")
	}
}
