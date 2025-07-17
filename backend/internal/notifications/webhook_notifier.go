package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	phxlog "phoenixgrc/backend/pkg/log"
	"go.uber.org/zap"
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
			bodyBytes, _ := io.ReadAll(resp.Body)
			bodyText = bodyBytes
			resp.Body.Close()
		}

		phxlog.L.Warn("Webhook send failed",
			zap.String("url", webhookURL),
			zap.Int("attempt", i+1),
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
