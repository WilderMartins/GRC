package notifications

import (
	"bytes"
	"context" // Adicionado
	"encoding/json"
	"fmt"
	"io" // Adicionado de volta
	"log"
	"net/http"
	// "os" // Removido, pois appCfg é usado
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	appCfg "phoenixgrc/backend/pkg/config" // Nosso config da aplicação
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsGoConfig "github.com/aws/aws-sdk-go-v2/config" // Config do SDK AWS com alias
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/google/uuid"
)

const maxWebhookRetries = 3
const webhookRetryDelay = 5 * time.Second

// GoogleChatMessage é a estrutura do payload para webhooks do Google Chat.
type GoogleChatMessage struct {
	Text string `json:"text"`
}

// SendWebhookNotification envia uma notificação para uma URL de webhook.
func SendWebhookNotification(webhookURL string, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	var lastErr error
	for i := 0; i < maxWebhookRetries; i++ {
		req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
		if err != nil {
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

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Printf("Webhook sent successfully to %s (status: %s)\n", webhookURL, resp.Status)
			if resp.Body != nil {
				resp.Body.Close()
			}
			return nil
		}

		var bodyText []byte
		if resp.Body != nil {
			bodyBytes, _ := io.ReadAll(resp.Body) // Corrigido para ler o corpo
			bodyText = bodyBytes
			resp.Body.Close()
		}

		log.Printf("Webhook to %s failed (try %d/%d) - Status: %s, Body: %s\n", webhookURL, i+1, maxWebhookRetries, resp.Status, string(bodyText))
		lastErr = fmt.Errorf("request failed with status %s", resp.Status)

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

	eventPattern := string(eventType)
	err := db.Where("organization_id = ? AND is_active = ?", orgID, true).
		Where(
			db.Where("event_types = ?", eventPattern).
				Or("event_types LIKE ?", eventPattern+",%").
				Or("event_types LIKE ?", "%,"+eventPattern+",%").
				Or("event_types LIKE ?", "%,"+eventPattern),
		).
		Find(&webhooks).Error

	if err != nil {
		log.Printf("Error fetching webhooks for org %s, event %s: %v\n", orgID, eventType, err)
		return
	}

	if len(webhooks) == 0 {
		return
	}

	// Construir a URL base do frontend a partir da configuração
	frontendBaseURL := appCfg.Cfg.FrontendBaseURL // Assumindo que esta variável existe em AppConfig e é carregada de FRONTEND_BASE_URL
	if frontendBaseURL == "" {
		frontendBaseURL = "http://localhost:3000" // Fallback se não configurado
		log.Printf("AVISO: FRONTEND_BASE_URL não configurado, usando fallback para notificação: %s", frontendBaseURL)
	}
	riskURLPlaceholder := fmt.Sprintf("%s/risks/%s", strings.TrimSuffix(frontendBaseURL, "/"), risk.ID.String())


	var messageText string
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
		go func(webhookURL string, pld interface{}) {
			if err := SendWebhookNotification(webhookURL, pld); err != nil {
				log.Printf("Falha ao enviar notificação de webhook para %s: %v\n", webhookURL, err)
			}
		}(wh.URL, payload)
	}
}

func SendEmailNotification(toEmail string, subject string, htmlBody string, textBody string) error {
	if toEmail == "" {
		return fmt.Errorf("destinatário do email (toEmail) não pode ser vazio")
	}

	if DefaultEmailNotifier != nil {
		return DefaultEmailNotifier.SendEmail(toEmail, subject, htmlBody, textBody)
	}

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
	return nil
}

func NotifyUserByEmail(userID uuid.UUID, subject string, textBody string) {
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
	htmlBody := fmt.Sprintf("<p>%s</p>", strings.ReplaceAll(textBody, "\n", "<br>")) // Simple HTML version

	go func(email, subj, txtBdy, htmlBdy string) {
		if err := SendEmailNotification(email, subj, htmlBdy, txtBdy); err != nil {
			log.Printf("Falha ao enviar notificação por email para %s: %v\n", email, err)
		}
	}(user.Email, subject, textBody, htmlBody)
}

type EmailNotifier interface {
	SendEmail(toEmail, subject, htmlBody, textBody string) error
}

type SESEmailNotifier struct {
	client      *sesv2.Client
	senderEmail string
}

var awsSDKConfig aws.Config
var sesInitializationError error

func InitializeAWSSession() error {
	region := appCfg.Cfg.AWSRegion // Usar config da app
	if region == "" {
		sesInitializationError = fmt.Errorf("AWS_REGION não está configurada na app config")
		log.Printf("AVISO: %v. O envio de emails reais estará desabilitado.", sesInitializationError)
		return nil
	}

	var err error
	awsSDKConfig, err = awsGoConfig.LoadDefaultConfig(context.TODO(), awsGoConfig.WithRegion(region)) // Usar alias
	if err != nil {
		sesInitializationError = fmt.Errorf("falha ao carregar configuração AWS: %w", err)
		log.Printf("AVISO: %v. O envio de emails reais estará desabilitado.", sesInitializationError)
		return nil
	}
	log.Println("Sessão AWS inicializada com sucesso para a região:", region)
	return nil
}

func NewSESEmailNotifier() (*SESEmailNotifier, error) {
	if sesInitializationError != nil {
		return nil, fmt.Errorf("não é possível criar SESEmailNotifier pois a sessão AWS não foi inicializada: %w", sesInitializationError)
	}
	if awsSDKConfig.Region == "" {
		return nil, fmt.Errorf("configuração AWS não carregada (região vazia). Chame InitializeAWSSession primeiro")
	}

	sender := appCfg.Cfg.AWSSESEmailSender // Usar config da app
	if sender == "" {
		return nil, fmt.Errorf("EMAIL_SENDER_ADDRESS (AWSSESEmailSender na app config) não está configurado")
	}

	sesClient := sesv2.NewFromConfig(awsSDKConfig) // Usar awsSDKConfig

	return &SESEmailNotifier{
		client:      sesClient,
		senderEmail: sender,
	}, nil
}

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
					Data:    aws.String(subject), // Usar aws.String
					Charset: aws.String("UTF-8"),
				},
				Body: &types.Body{},
			},
		},
	}

	if textBody != "" {
		input.Content.Simple.Body.Text = &types.Content{
			Data:    aws.String(textBody), // Usar aws.String
			Charset: aws.String("UTF-8"),
		}
	}
	if htmlBody != "" {
		input.Content.Simple.Body.Html = &types.Content{
			Data:    aws.String(htmlBody), // Usar aws.String
			Charset: aws.String("UTF-8"),
		}
	}

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

var DefaultEmailNotifier EmailNotifier

func InitEmailService() {
	if err := InitializeAWSSession(); err != nil {
		// Log já feito em InitializeAWSSession
	}

	if sesInitializationError == nil && awsSDKConfig.Region != "" {
		notifier, err := NewSESEmailNotifier()
		if err != nil {
			log.Printf("AVISO: Falha ao inicializar o AWS SES Email Notifier: %v. O envio de emails reais estará desabilitado.", err)
		} else {
			DefaultEmailNotifier = notifier
			log.Println("AWS SES Email Notifier inicializado e configurado como padrão.")
		}
	} else {
		log.Println("AVISO: Sessão AWS não inicializada ou região não configurada. AWS SES Email Notifier não será ativado. O envio de emails reais estará desabilitado.")
	}
}
