package notifications

import (
	"context"
	"errors"
	"os"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	phxlog "phoenixgrc/backend/pkg/log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"go.uber.org/zap"
)

// EmailNotifier define a interface para um notificador de email.
type EmailNotifier interface {
	SendEmail(to, subject, bodyHTML, bodyText string) error
}

// SESEmailNotifier implementa EmailNotifier usando AWS SES.
type SESEmailNotifier struct {
	client *sesv2.Client
	sender string
}

// DefaultEmailNotifier é o notificador padrão usado pela aplicação.
var DefaultEmailNotifier EmailNotifier

// InitEmailService inicializa o notificador de e-mail.
// Ele tenta carregar a configuração do banco de dados primeiro,
// e usa as variáveis de ambiente como fallback.
func InitEmailService() {
	log := phxlog.L.Named("InitEmailService")
	db := database.GetDB()

	// Tenta obter configurações do banco de dados
	awsRegion, errRegion := models.GetSystemSetting(db, "AWS_REGION")
	senderEmail, errSender := models.GetSystemSetting(db, "AWS_SES_EMAIL_SENDER")

	if errRegion != nil || errSender != nil {
		log.Warn("Could not retrieve email settings from database, falling back to environment variables.")
		// Fallback para variáveis de ambiente (comportamento original)
		awsRegion = os.Getenv("AWS_REGION")
		senderEmail = os.Getenv("AWS_SES_EMAIL_SENDER")
	}

	if awsRegion == "" || senderEmail == "" {
		log.Warn("AWS SES email service is not configured (missing AWS_REGION or AWS_SES_EMAIL_SENDER). Email notifications will be disabled.")
		DefaultEmailNotifier = nil
		return
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(awsRegion))
	if err != nil {
		log.Error("Failed to load AWS SDK config for SES", zap.Error(err))
		DefaultEmailNotifier = nil
		return
	}

	DefaultEmailNotifier = &SESEmailNotifier{
		client: sesv2.NewFromConfig(cfg),
		sender: senderEmail,
	}
	log.Info("AWS SES email service initialized successfully.", zap.String("sender", senderEmail), zap.String("region", awsRegion))
}

// SendEmailNotification envia um e-mail usando o serviço configurado.
func SendEmailNotification(to, subject, bodyHTML, bodyText string) error {
	if DefaultEmailNotifier == nil {
		phxlog.L.Info("--- SIMULATING EMAIL SEND (Fallback) ---",
			zap.String("to", to),
			zap.String("subject", subject))
		return nil
	}
	return DefaultEmailNotifier.SendEmail(to, subject, bodyHTML, bodyText)
}

// SendEmail é o método da implementação SESEmailNotifier.
func (s *SESEmailNotifier) SendEmail(to, subject, bodyHTML, bodyText string) error {
	if s.client == nil {
		return errors.New("SES client not initialized")
	}

	input := &sesv2.SendEmailInput{
		FromEmailAddress: &s.sender,
		Destination: &types.Destination{
			ToAddresses: []string{to},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Body: &types.Body{
					Html: &types.Content{
						Data:    aws.String(bodyHTML),
						Charset: aws.String("UTF-8"),
					},
					Text: &types.Content{
						Data:    aws.String(bodyText),
						Charset: aws.String("UTF-8"),
					},
				},
				Subject: &types.Content{
					Data:    aws.String(subject),
					Charset: aws.String("UTF-8"),
				},
			},
		},
	}

	_, err := s.client.SendEmail(context.TODO(), input)
	if err != nil {
		phxlog.L.Error("Failed to send email via SES", zap.Error(err), zap.String("recipient", to))
		return err
	}

	phxlog.L.Info("Successfully sent email", zap.String("recipient", to), zap.String("subject", subject))
	return nil
}
