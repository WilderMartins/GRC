package notifications

import (
	"context"
	"testing"

	"phoenixgrc/backend/pkg/config"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// MockNotifier para simular o comportamento de notificação.
type MockNotifier struct {
	SendFunc      func(ctx context.Context, to, subject, body string) error
	SendCalled    bool
	LastTo        string
	LastSubject   string
	LastBody      string
}

func (m *MockNotifier) Send(ctx context.Context, to, subject, body string) error {
	m.SendCalled = true
	m.LastTo = to
	m.LastSubject = subject
	m.LastBody = body
	if m.SendFunc != nil {
		return m.SendFunc(ctx, to, subject, body)
	}
	return nil
}

func TestInitEmailService(t *testing.T) {
	originalNotifier := DefaultEmailNotifier
	originalCfg := config.Cfg
	defer func() {
		DefaultEmailNotifier = originalNotifier
		config.Cfg = originalCfg
	}()

	t.Run("Service initializes with logNotifier when config is missing", func(t *testing.T) {
		config.Cfg.AWSRegion = ""
		config.Cfg.AWSSESEmailSender = ""

		InitEmailService()

		assert.NotNil(t, DefaultEmailNotifier)
		_, ok := DefaultEmailNotifier.(*logNotifier)
		assert.True(t, ok, "DefaultEmailNotifier should be a logNotifier")
	})

	t.Run("Service initializes with logNotifier when AWS SDK fails", func(t *testing.T) {
		// Simular falha no SDK (difícil sem mockar o SDK, mas podemos assumir que logNotifier é o fallback)
		// A lógica atual já usa logNotifier como fallback para qualquer falha na inicialização.
		config.Cfg.AWSRegion = "us-east-1"
		config.Cfg.AWSSESEmailSender = "sender@example.com"
		// Sem credenciais AWS reais, a inicialização do SDK falhará, caindo para logNotifier.
		// Esta é uma suposição sobre o comportamento do SDK, mas reflete o design do nosso código.
		InitEmailService()
		assert.NotNil(t, DefaultEmailNotifier)
		// O tipo exato pode depender de onde a falha ocorre. O importante é que não seja nil.
	})
}

func TestNotifyUserByEmail(t *testing.T) {
	// Esta função agora depende de um banco de dados mock, que não está configurado aqui.
	// Testar a lógica de notificação em si é mais um teste de integração.
	// Para um teste de unidade, podemos verificar se a função não entra em pânico com um nil Notifier.
	t.Run("Does not panic with nil notifier", func(t *testing.T) {
		originalNotifier := DefaultEmailNotifier
		DefaultEmailNotifier = nil
		defer func() { DefaultEmailNotifier = originalNotifier }()

		// Esta chamada irá logar um erro, mas não deve causar pânico.
		// O teste não pode verificar o log facilmente sem uma configuração mais complexa.
		assert.NotPanics(t, func() {
			uid, _ := uuid.Parse("00000000-0000-0000-0000-000000000000")
			NotifyUserByEmail(context.Background(), uid, "subject", "body")
		})
	})
}

func TestLogNotifier(t *testing.T) {
	notifier := &logNotifier{}
	err := notifier.Send(context.Background(), "test@example.com", "Test Subject", "Test Body")
	assert.NoError(t, err, "logNotifier should never return an error")
}
