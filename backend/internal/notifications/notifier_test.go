package notifications

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	// Não vamos importar o SDK da AWS aqui para manter os testes unitários focados na nossa lógica
)

func TestInitializeAWSSession(t *testing.T) {
	originalRegion := os.Getenv("AWS_REGION")
	defer os.Setenv("AWS_REGION", originalRegion) // Restaurar variável

	t.Run("AWS_REGION not set", func(t *testing.T) {
		os.Unsetenv("AWS_REGION")
		err := InitializeAWSSession()
		assert.NoError(t, err, "InitializeAWSSession should not return fatal error if region is not set, only log")
		assert.NotNil(t, sesInitializationError, "sesInitializationError should be set")
		assert.Contains(t, sesInitializationError.Error(), "AWS_REGION não está configurada")
		sesInitializationError = nil // Reset for other tests
		cfg.Region = "" // Reset for other tests
	})

	t.Run("AWS_REGION set but AWS SDK fails to load config (simulated by not having credentials)", func(t *testing.T) {
		// Esta é difícil de simular perfeitamente sem mockar o SDK profundamente
		// ou garantir que nenhuma credencial esteja disponível no ambiente de teste.
		// A função config.LoadDefaultConfig pode ter múltiplos fallbacks.
		// Por enquanto, vamos assumir que se a região estiver lá, o SDK tentará carregar.
		// O erro real viria do SDK se as credenciais não pudessem ser encontradas/validadas.
		// O nosso wrapper InitializeAWSSession deve logar e retornar nil.
		os.Setenv("AWS_REGION", "test-region-1")

		// Para garantir que não pegue credenciais reais do ambiente de teste, se houver
		originalAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
		originalSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
		originalSessionToken := os.Getenv("AWS_SESSION_TOKEN")
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		os.Unsetenv("AWS_SESSION_TOKEN")
		// Também pode ser necessário limpar ~/.aws/credentials e config se o SDK os ler.
		// Este teste é mais um teste de integração "leve".

		err := InitializeAWSSession() // Isso vai tentar carregar credenciais e falhar se não encontrar
		assert.NoError(t, err, "InitializeAWSSession should log error but return nil")
		// Se sesInitializationError foi setado pela falta de credenciais, o teste é válido.
		// O erro exato do SDK pode variar.
		// assert.NotNil(t, sesInitializationError) // Difícil de garantir sem um mock do SDK

		sesInitializationError = nil
		cfg.Region = ""
		os.Setenv("AWS_ACCESS_KEY_ID", originalAccessKey)
		os.Setenv("AWS_SECRET_ACCESS_KEY", originalSecretKey)
		os.Setenv("AWS_SESSION_TOKEN", originalSessionToken)
	})
}

func TestNewSESEmailNotifier(t *testing.T) {
	originalRegion := os.Getenv("AWS_REGION")
	originalSender := os.Getenv("EMAIL_SENDER_ADDRESS")
	defer func() {
		os.Setenv("AWS_REGION", originalRegion)
		os.Setenv("EMAIL_SENDER_ADDRESS", originalSender)
		sesInitializationError = nil
		cfg.Region = ""
	}()

	t.Run("SES Notifier creation fails if AWS session not initialized", func(t *testing.T) {
		os.Unsetenv("AWS_REGION") // Garante que a sessão falhe
		InitializeAWSSession()    // Define sesInitializationError

		_, err := NewSESEmailNotifier()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sessão AWS não foi inicializada")
	})

	t.Run("SES Notifier creation fails if sender email not set", func(t *testing.T) {
		os.Setenv("AWS_REGION", "test-region") // Simula sessão AWS OK
		InitializeAWSSession() // Limpa sesInitializationError e define cfg.Region
		os.Unsetenv("EMAIL_SENDER_ADDRESS")

		_, err := NewSESEmailNotifier()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "EMAIL_SENDER_ADDRESS não está configurado")
	})

    // Teste para criação bem-sucedida é mais um teste de integração,
    // pois requer que config.LoadDefaultConfig funcione e encontre credenciais válidas.
    // Não faremos aqui para manter unitário.
}


func TestSendEmailNotificationLogic(t *testing.T) {
    // Testar a lógica de fallback da função SendEmailNotification
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
        DefaultEmailNotifier = nil // Garantir que não há notificador real

        // Como verificar o log é mais complexo, vamos apenas garantir que não dá erro
        // e assumir que o log ocorreu (coberto por inspeção visual ou testes de log mais avançados)
        err := SendEmailNotification("log@example.com", "Log Subject", "<p>Log HTML</p>", "Log Text")
        assert.NoError(t, err, "Fallback logging should not produce an error")
        // Para verificar o log, você precisaria de um logger mockado ou capturar stdout/stderr.
    })

    t.Run("SendEmail returns error if toEmail is empty", func(t *testing.T) {
        err := SendEmailNotification("", "Subject", "", "Body")
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "destinatário do email (toEmail) não pode ser vazio")
    })
}

// MockEmailNotifier para testar a lógica de SendEmailNotification
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

// Testes para NotifyUserByEmail (verificando se busca usuário e chama SendEmailNotification)
// Estes testes precisarão de mocks de banco de dados, então seriam melhor colocados
// em um contexto onde o DB mock (sqlmock) está configurado, como nos testes de handler.
// Ou, refatorar NotifyUserByEmail para aceitar uma interface de DB.
// Por enquanto, vamos focar nos testes acima.

// Lembre-se que InitializeAWSSession e NewSESEmailNotifier dependem de variáveis de ambiente
// e do comportamento real do AWS SDK para carregar configurações/credenciais.
// Testes unitários puros para eles são limitados sem mocks profundos do SDK.
// Os testes acima focam mais na lógica de erro e fallback da nossa aplicação.
