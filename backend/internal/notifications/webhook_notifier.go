package notifications

import (
	"bytes"
	"context" // Adicionado
	"bytes"
	"context" // Adicionado
	"encoding/json"
	"fmt"
	"io" // Adicionado de volta
	"net/http"
	// "os" // Removido, pois appCfg é usado
	phxlog "phoenixgrc/backend/pkg/log" // Importar o logger zap
	"go.uber.org/zap"                 // Importar zap
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
			phxlog.L.Error("Error creating webhook request",
				zap.String("url", webhookURL),
				zap.Int("attempt", i+1),
				zap.Int("max_attempts", maxWebhookRetries),
				zap.Error(err))
			lastErr = fmt.Errorf("failed to create request: %w", err)
			time.Sleep(webhookRetryDelay)
			continue
		}
		req.Header.Set("Content-Type", "application/json; charset=UTF-8")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			phxlog.L.Error("Error sending webhook",
				zap.String("url", webhookURL),
				zap.Int("attempt", i+1),
				zap.Int("max_attempts", maxWebhookRetries),
				zap.Error(err))
			lastErr = fmt.Errorf("request failed: %w", err)
			time.Sleep(webhookRetryDelay)
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			phxlog.L.Info("Webhook sent successfully",
				zap.String("url", webhookURL),
				zap.String("status", resp.Status))
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

		phxlog.L.Warn("Webhook send failed",
			zap.String("url", webhookURL),
			zap.Int("attempt", i+1),
			zap.Int("max_attempts", maxWebhookRetries),
			zap.String("status", resp.Status),
			zap.ByteString("response_body", bodyText))
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
		phxlog.L.Error("Error fetching webhooks for notification",
			zap.String("organizationID", orgID.String()),
			zap.String("eventType", string(eventType)),
			zap.Error(err))
		return
	}

	if len(webhooks) == 0 {
		return // Nenhum webhook configurado para este evento/org
	}

	// Construir a URL base do frontend a partir da configuração
	frontendBaseURL := appCfg.Cfg.FrontendBaseURL
	if frontendBaseURL == "" {
		frontendBaseURL = "http://localhost:3000" // Fallback se não configurado
		phxlog.L.Warn("FRONTEND_BASE_URL not configured, using fallback for webhook notification link.",
			zap.String("fallback_url", frontendBaseURL))
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
		phxlog.L.Warn("Unknown risk event type for notification", zap.String("eventType", string(eventType)))
		return
	}

	payload := GoogleChatMessage{Text: messageText}

	for _, wh := range webhooks {
		go func(webhookURL string, pld interface{}, webhookName string) { // Adicionar webhookName para log
			if err := SendWebhookNotification(webhookURL, pld); err != nil {
				phxlog.L.Error("Failed to send webhook notification",
					zap.String("webhookURL", webhookURL),
					zap.String("webhookName", webhookName),
					zap.String("eventType", string(eventType)),
					zap.String("riskID", risk.ID.String()),
					zap.Error(err))
			}
		}(wh.URL, payload, wh.Name)
	}
}

func SendEmailNotification(toEmail string, subject string, htmlBody string, textBody string) error {
	if toEmail == "" {
		return fmt.Errorf("destinatário do email (toEmail) não pode ser vazio")
	}

	if DefaultEmailNotifier != nil {
		return DefaultEmailNotifier.SendEmail(toEmail, subject, htmlBody, textBody)
	}

	phxlog.L.Info("--- SIMULATING EMAIL SEND (Fallback) ---",
		zap.String("to", toEmail),
		zap.String("subject", subject))
	if textBody != "" {
		phxlog.L.Debug("Email Body (Text)", zap.String("body", textBody))
	}
	if htmlBody != "" {
		phxlog.L.Debug("Email Body (HTML)", zap.String("body", htmlBody))
	}
	phxlog.L.Info("--- END OF EMAIL SIMULATION (Fallback) ---")
	return nil
}

func NotifyUserByEmail(userID uuid.UUID, subject string, textBody string) {
	if userID == uuid.Nil {
		phxlog.L.Warn("Attempted to notify user by email with nil UserID.")
		return
	}
	db := database.GetDB()
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		phxlog.L.Error("Error fetching user for email notification",
			zap.String("userID", userID.String()),
			zap.Error(err))
		return
	}
	if user.Email == "" {
		phxlog.L.Warn("User has no email address for notification.",
			zap.String("userID", userID.String()))
		return
	}
	htmlBody := fmt.Sprintf("<p>%s</p>", strings.ReplaceAll(textBody, "\n", "<br>")) // Simple HTML version

	go func(email, subj, txtBdy, htmlBdy string, uID uuid.UUID) { // Adicionar uID para log
		if err := SendEmailNotification(email, subj, htmlBdy, txtBdy); err != nil {
			phxlog.L.Error("Failed to send email notification",
				zap.String("recipientEmail", email),
				zap.String("userID", uID.String()),
				zap.Error(err))
		}
	}(user.Email, subject, textBody, htmlBody, userID)
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
		phxlog.L.Warn("AWS_REGION not configured. Real email sending will be disabled.", zap.Error(sesInitializationError))
		return nil
	}

	var err error
	awsSDKConfig, err = awsGoConfig.LoadDefaultConfig(context.TODO(), awsGoConfig.WithRegion(region)) // Usar alias
	if err != nil {
		sesInitializationError = fmt.Errorf("falha ao carregar configuração AWS: %w", err)
		phxlog.L.Error("Failed to load AWS SDK config. Real email sending will be disabled.", zap.Error(sesInitializationError))
		return nil
	}
	phxlog.L.Info("AWS SDK session initialized successfully", zap.String("region", region))
	return nil
}

func NewSESEmailNotifier() (*SESEmailNotifier, error) {
	if sesInitializationError != nil {
		// Este erro já foi logado em InitializeAWSSession
		return nil, fmt.Errorf("cannot create SESEmailNotifier because AWS session was not initialized: %w", sesInitializationError)
	}
	if awsSDKConfig.Region == "" { // Checagem adicional
		return nil, fmt.Errorf("AWS config not loaded (region is empty). Call InitializeAWSSession first")
	}

	sender := appCfg.Cfg.AWSSESEmailSender // Usar config da app
	if sender == "" {
		return nil, fmt.Errorf("EMAIL_SENDER_ADDRESS (AWSSESEmailSender in app config) is not configured")
	}

	sesClient := sesv2.NewFromConfig(awsSDKConfig) // Usar awsSDKConfig

	return &SESEmailNotifier{
		client:      sesClient,
		senderEmail: sender,
	}, nil
}

func (s *SESEmailNotifier) SendEmail(toEmail, subject, htmlBody, textBody string) error {
	if s.client == nil {
		return fmt.Errorf("SES client not initialized")
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
		phxlog.L.Error("Failed to send email via SES",
			zap.String("to", toEmail),
			zap.String("subject", subject),
			zap.Error(err))
		return fmt.Errorf("falha ao enviar email via SES: %w", err)
	}

	phxlog.L.Info("Email sent successfully via AWS SES",
		zap.String("to", toEmail),
		zap.String("subject", subject))
	return nil
}

var DefaultEmailNotifier EmailNotifier

func InitEmailService() {
	if err := InitializeAWSSession(); err != nil {
		// Erro já logado em InitializeAWSSession se sesInitializationError foi setado.
		// Se InitializeAWSSession retornasse um erro real, poderíamos logá-lo aqui.
		// Como ela retorna nil e seta uma var global de erro, a lógica abaixo lida com isso.
	}

	if sesInitializationError == nil && awsSDKConfig.Region != "" {
		notifier, err := NewSESEmailNotifier()
		if err != nil {
			phxlog.L.Warn("Failed to initialize AWS SES Email Notifier. Real email sending will be disabled.", zap.Error(err))
		} else {
			DefaultEmailNotifier = notifier
			phxlog.L.Info("AWS SES Email Notifier initialized and set as default.")
		}
	} else {
		phxlog.L.Warn("AWS session not initialized or region not configured. AWS SES Email Notifier will not be activated. Real email sending will be disabled.")
	}
}
