package notifications

import (
	"fmt"
	"context"
	"strings"

	"phoenixgrc/backend/internal/database"
	"phoenixgrc/backend/internal/models"
	"phoenixgrc/backend/pkg/config"
	phxlog "phoenixgrc/backend/pkg/log"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// NotifyRiskEvent busca webhooks e/ou outros métodos de notificação para um evento de risco.
func NotifyRiskEvent(ctx context.Context, orgID uuid.UUID, risk models.Risk, eventType models.WebhookEventType) {
	// Notificação via Webhook
	notifyRiskEventViaWebhook(ctx, orgID, risk, eventType)

	// Outros métodos de notificação (ex: e-mail) podem ser adicionados aqui.
}

func notifyRiskEventViaWebhook(ctx context.Context, orgID uuid.UUID, risk models.Risk, eventType models.WebhookEventType) {
	db := database.GetDB()
	var webhooks []models.WebhookConfiguration

	eventPattern := string(eventType)
	err := db.WithContext(ctx).Where("organization_id = ? AND is_active = ?", orgID, true).
		Where("event_types LIKE ?", "%"+eventPattern+"%").
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

	frontendBaseURL := config.Cfg.FrontendBaseURL
	if frontendBaseURL == "" {
		frontendBaseURL = "http://localhost:3000" // Fallback
	}
	riskURL := fmt.Sprintf("%s/risks/%s", strings.TrimSuffix(frontendBaseURL, "/"), risk.ID.String())

	var messageText string
	switch eventType {
	case models.EventTypeRiskCreated:
		messageText = fmt.Sprintf("🚀 Novo risco criado: *%s*\nDescrição: %s\nImpacto: %s, Probabilidade: %s\nLink: %s",
			risk.Title, risk.Description, risk.Impact, risk.Probability, riskURL)
	case models.EventTypeRiskStatusChanged:
		messageText = fmt.Sprintf("🔄 Status do risco '*%s*' alterado para: *%s*\nLink: %s",
			risk.Title, risk.Status, riskURL)
	default:
		phxlog.L.Warn("Unknown risk event type for notification", zap.String("eventType", string(eventType)))
		return
	}

	payload := GoogleChatMessage{Text: messageText}

	for _, wh := range webhooks {
		go func(webhookURL, webhookName string) {
			if err := SendWebhookNotification(webhookURL, payload); err != nil {
				phxlog.L.Error("Failed to send webhook notification",
					zap.String("webhookURL", webhookURL),
					zap.String("webhookName", webhookName),
					zap.Error(err))
			}
		}(wh.URL, wh.Name)
	}
}

// NotifyUserByEmail envia uma notificação por e-mail para um usuário específico.
func NotifyUserByEmail(ctx context.Context, userID uuid.UUID, subject, body string) {
	if userID == uuid.Nil {
		phxlog.L.Warn("Attempted to notify user by email with nil UserID.")
		return
	}
	db := database.GetDB()
	var user models.User
	if err := db.WithContext(ctx).First(&user, userID).Error; err != nil {
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

	go func(email, subj, bdy string) {
		if DefaultEmailNotifier != nil {
			if err := DefaultEmailNotifier.Send(context.Background(), email, subj, bdy); err != nil {
				phxlog.L.Error("Failed to send email notification",
					zap.String("recipientEmail", email),
					zap.Error(err))
			}
		}
	}(user.Email, subject, body)
}
