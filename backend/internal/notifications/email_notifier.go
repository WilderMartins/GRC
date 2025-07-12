package notifications

import (
	"context"
	"errors"
	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	phxlog "phoenixgrc/backend/pkg/log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"go.uber.org/zap"
)

// EmailNotifier é responsável por enviar e-mails.
type EmailNotifier struct {
	client *sesv2.Client
	sender string
}

var emailNotifier *EmailNotifier

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
		awsRegion = phxlog.GetEnv("AWS_REGION", "")
		senderEmail = phxlog.GetEnv("AWS_SES_EMAIL_SENDER", "")
	}

	if awsRegion == "" || senderEmail == "" {
		log.Warn("AWS SES email service is not configured (missing AWS_REGION or AWS_SES_EMAIL_SENDER). Email notifications will be disabled.")
		emailNotifier = nil
		return
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(awsRegion))
	if err != nil {
		log.Error("Failed to load AWS SDK config for SES", zap.Error(err))
		emailNotifier = nil
		return
	}

	emailNotifier = &EmailNotifier{
		client: sesv2.NewFromConfig(cfg),
		sender: senderEmail,
	}
	log.Info("AWS SES email service initialized successfully.", zap.String("sender", senderEmail), zap.String("region", awsRegion))
}

// SendEmail envia um e-mail usando o serviço configurado.
func SendEmail(to, subject, bodyHTML, bodyText string) error {
	if emailNotifier == nil {
		return errors.New("email service is not initialized")
	}

	input := &sesv2.SendEmailInput{
		FromEmailAddress: &emailNotifier.sender,
		Destination: &types.Destination{
			ToAddresses: []string{to},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Body: &types.Body{
					Html: &types.Content{
						Data: &bodyHTML,
					},
					Text: &types.Content{
						Data: &bodyText,
					},
				},
				Subject: &types.Content{
					Data: &subject,
				},
			},
		},
	}

	_, err := emailNotifier.client.SendEmail(context.TODO(), input)
	if err != nil {
		phxlog.L.Error("Failed to send email via SES", zap.Error(err), zap.String("recipient", to))
		return err
	}

	phxlog.L.Info("Successfully sent email", zap.String("recipient", to), zap.String("subject", subject))
	return nil
}
