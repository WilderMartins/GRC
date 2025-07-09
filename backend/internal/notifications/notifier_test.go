package notifications

import (
	"os"
	"testing"
	appCfg "phoenixgrc/backend/pkg/config" // Adicionado para acessar a config da app

	"github.com/stretchr/testify/assert"
	// Não vamos importar o SDK da AWS aqui para manter os testes unitários focados na nossa lógica
)

func TestInitializeAWSSession(t *testing.T) {
	// Salvar e restaurar o valor original de appCfg.Cfg.AWSRegion
	originalAppAWSRegion := appCfg.Cfg.AWSRegion
	defer func() {
		appCfg.Cfg.AWSRegion = originalAppAWSRegion
		sesInitializationError = nil
		awsSDKConfig.Region = ""
	}()

	t.Run("AWS_REGION not set in appCfg", func(t *testing.T) {
		appCfg.Cfg.AWSRegion = "" // Simula que não está na config da app
		err := InitializeAWSSession()
		assert.NoError(t, err, "InitializeAWSSession should not return fatal error if region is not set, only log")
		assert.NotNil(t, sesInitializationError, "sesInitializationError should be set")
		assert.Contains(t, sesInitializationError.Error(), "AWS_REGION não está configurada na app config")
		sesInitializationError = nil
		awsSDKConfig.Region = ""
	})

	t.Run("AWS_REGION set but AWS SDK fails to load config (simulated by not having credentials)", func(t *testing.T) {
		appCfg.Cfg.AWSRegion = "test-region-1" // Simula que está na config da app

		originalAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
		originalSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
		originalSessionToken := os.Getenv("AWS_SESSION_TOKEN")
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		os.Unsetenv("AWS_SESSION_TOKEN")
		defer func() {
			os.Setenv("AWS_ACCESS_KEY_ID", originalAccessKey)
			os.Setenv("AWS_SECRET_ACCESS_KEY", originalSecretKey)
			os.Setenv("AWS_SESSION_TOKEN", originalSessionToken)
		}()

		err := InitializeAWSSession()
		assert.NoError(t, err, "InitializeAWSSession should log error but return nil")
		// O erro específico de credenciais é difícil de mockar/garantir sem mais controle sobre o SDK loader.
		// sesInitializationError pode ou não ser setado dependendo de como o SDK lida com a ausência de credenciais.
		// Por enquanto, o importante é que InitializeAWSSession não retorne um erro fatal.
		sesInitializationError = nil
		awsSDKConfig.Region = ""
	})
}

func TestNewSESEmailNotifier(t *testing.T) {
	originalAppAWSRegion := appCfg.Cfg.AWSRegion
	originalAppSender := appCfg.Cfg.AWSSESEmailSender
	defer func() {
		appCfg.Cfg.AWSRegion = originalAppAWSRegion
		appCfg.Cfg.AWSSESEmailSender = originalAppSender
		sesInitializationError = nil
		awsSDKConfig.Region = ""
	}()

	t.Run("SES Notifier creation fails if AWS session not initialized (AWSRegion empty in appCfg)", func(t *testing.T) {
		appCfg.Cfg.AWSRegion = ""  // Garante que a sessão falhe ao ler de appCfg
		InitializeAWSSession()     // Isso vai setar sesInitializationError

		_, err := NewSESEmailNotifier()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sessão AWS não foi inicializada")
		assert.Contains(t, sesInitializationError.Error(), "AWS_REGION não está configurada na app config")
	})

	t.Run("SES Notifier creation fails if sender email not set in appCfg", func(t *testing.T) {
		appCfg.Cfg.AWSRegion = "us-east-1" // Configura região para passar na inicialização da sessão
		sesInitializationError = nil      // Limpa erro anterior
		awsSDKConfig.Region = ""          // Força InitializeAWSSession a tentar carregar

		err := InitializeAWSSession()
		assert.NoError(t, err)
		assert.Nil(t, sesInitializationError)
		assert.Equal(t, "us-east-1", awsSDKConfig.Region)

		appCfg.Cfg.AWSSESEmailSender = "" // Remove o sender da config da app

		_, errNotifier := NewSESEmailNotifier()
		assert.Error(t, errNotifier)
		assert.Contains(t, errNotifier.Error(), "EMAIL_SENDER_ADDRESS (AWSSESEmailSender na app config) não está configurado")
	})
}


func TestSendEmailNotificationLogic(t *testing.T) {
    originalNotifier := DefaultEmailNotifier
    defer func() { DefaultEmailNotifier = originalNotifier }()

    t.Run("SendEmail uses DefaultEmailNotifier if set", func(t *testing.T) {
        mockNotifier := &MockEmailNotifier{}
        DefaultEmailNotifier = mockNotifier

        to := "test@example.com"
        subject := "Test Subject"
        htmlBody := "<p>Test HTML</p>"
        textBody := "Test Text"

        mockNotifier.SendEmailFunc = func(rto, rsubject, rhtmlBody, rtextBody string) error {
            assert.Equal(t, to, rto)
            assert.Equal(t, subject, rsubject)
            assert.Equal(t, htmlBody, rhtmlBody)
            assert.Equal(t, textBody, rtextBody)
            return nil
        }

        err := SendEmailNotification(to, subject, htmlBody, textBody)
        assert.NoError(t, err)
        assert.True(t, mockNotifier.SendEmailCalled, "Expected SendEmail on mock notifier to be called")
    })

    t.Run("SendEmail falls back to logging if DefaultEmailNotifier is nil", func(t *testing.T) {
        DefaultEmailNotifier = nil

        err := SendEmailNotification("log@example.com", "Log Subject", "<p>Log HTML</p>", "Log Text")
        assert.NoError(t, err, "Fallback logging should not produce an error")
    })

    t.Run("SendEmail returns error if toEmail is empty", func(t *testing.T) {
        err := SendEmailNotification("", "Subject", "", "Body")
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "destinatário do email (toEmail) não pode ser vazio")
    })
}

type MockEmailNotifier struct {
    SendEmailFunc   func(toEmail, subject, htmlBody, textBody string) error
    SendEmailCalled bool
}

func (m *MockEmailNotifier) SendEmail(toEmail, subject, htmlBody, textBody string) error {
    m.SendEmailCalled = true
    if m.SendEmailFunc != nil {
        return m.SendEmailFunc(toEmail, subject, htmlBody, textBody)
    }
    return nil
}
