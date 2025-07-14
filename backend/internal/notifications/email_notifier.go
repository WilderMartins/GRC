package notifications

import (
	"context"
	"fmt"

	"phoenixgrc/backend/pkg/config"
	phxlog "phoenixgrc/backend/pkg/log"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsGoConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"go.uber.org/zap"
)

// Notifier é uma interface genérica para enviar notificações.
type Notifier interface {
	Send(ctx context.Context, to, subject, body string) error
}

// SESEmailNotifier implementa a interface Notifier para o Amazon SES.
type SESEmailNotifier struct {
	client      *sesv2.Client
	senderEmail string
}

// DefaultEmailNotifier é a instância padrão do notificador de e-mail.
var DefaultEmailNotifier Notifier

// InitEmailService inicializa o notificador de e-mail padrão.
func InitEmailService() {
	log := phxlog.L.Named("EmailService")
	region := config.Cfg.AWSRegion
	sender := config.Cfg.AWSSESEmailSender

	if region == "" || sender == "" {
		log.Warn("AWS SES email service is not configured (missing AWS_REGION or AWS_SENDER_EMAIL). Email notifications will be disabled.")
		DefaultEmailNotifier = &logNotifier{} // Fallback para um notificador que apenas loga
		return
	}

	sdkConfig, err := awsGoConfig.LoadDefaultConfig(context.TODO(), awsGoConfig.WithRegion(region))
	if err != nil {
		log.Error("Failed to load AWS SDK config for SES", zap.Error(err))
		DefaultEmailNotifier = &logNotifier{}
		return
	}

	DefaultEmailNotifier = &SESEmailNotifier{
		client:      sesv2.NewFromConfig(sdkConfig),
		senderEmail: sender,
	}
	log.Info("AWS SES email service initialized successfully.", zap.String("sender", sender), zap.String("region", region))
}

// Send envia um e-mail usando o Amazon SES.
func (s *SESEmailNotifier) Send(ctx context.Context, to, subject, body string) error {
	input := &sesv2.SendEmailInput{
		FromEmailAddress: &s.senderEmail,
		Destination: &types.Destination{
			ToAddresses: []string{to},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data:    aws.String(subject),
					Charset: aws.String("UTF-8"),
				},
				Body: &types.Body{
					Text: &types.Content{
						Data:    aws.String(body),
						Charset: aws.String("UTF-8"),
					},
					Html: &types.Content{
						Data:    aws.String(body), // Usando o mesmo corpo para HTML por simplicidade
						Charset: aws.String("UTF-8"),
					},
				},
			},
		},
	}

	_, err := s.client.SendEmail(ctx, input)
	if err != nil {
		phxlog.L.Error("Failed to send email via SES",
			zap.String("to", to),
			zap.String("subject", subject),
			zap.Error(err))
		return fmt.Errorf("failed to send email via SES: %w", err)
	}

	phxlog.L.Info("Email sent successfully via AWS SES",
		zap.String("to", to),
		zap.String("subject", subject))
	return nil
}

// logNotifier é um notificador que apenas registra as mensagens, usado como fallback.
type logNotifier struct{}

func (n *logNotifier) Send(ctx context.Context, to, subject, body string) error {
	phxlog.L.Info("--- SIMULATING EMAIL SEND (Fallback) ---",
		zap.String("to", to),
		zap.String("subject", subject),
		zap.String("body", body))
	return nil
}
