package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// WebhookSender POSTs a JSON event payload to a user-configured URL.
type WebhookSender struct {
	client   *http.Client
	demoMode bool
}

// NewWebhookSender builds a sender. In demoMode, Send never actually POSTs
// anywhere. See the demoMode check below for why.
func NewWebhookSender(demoMode bool) *WebhookSender {
	return &WebhookSender{client: &http.Client{Timeout: 10 * time.Second}, demoMode: demoMode}
}

type webhookPayload struct {
	Event            EventType `json:"event"`
	MonitorID        string    `json:"monitor_id"`
	MonitorName      string    `json:"monitor_name"`
	MonitorURL       string    `json:"monitor_url"`
	StatusCode       int       `json:"status_code,omitempty"`
	Error            string    `json:"error,omitempty"`
	CheckedAt        time.Time `json:"checked_at"`
	DownDurationSecs *int64    `json:"down_duration_seconds,omitempty"`
}

func (s *WebhookSender) Send(ctx context.Context, url string, ev Event) error {
	// A publicly hosted demo can't be allowed to actually POST to whatever
	// URL a visitor types in, since that's an open request-proxy pointed
	// at arbitrary targets. Log what would have been sent and stop here.
	if s.demoMode {
		log.Printf("[DEMO] would POST webhook to %s: event=%s monitor=%s", url, ev.Type, ev.Monitor.Name)
		return nil
	}

	if url == "" {
		return fmt.Errorf("no webhook URL configured")
	}

	payload := webhookPayload{
		Event:       ev.Type,
		MonitorID:   ev.Monitor.ID,
		MonitorName: ev.Monitor.Name,
		MonitorURL:  ev.Monitor.URL,
		StatusCode:  ev.StatusCode,
		Error:       ev.Error,
		CheckedAt:   ev.CheckedAt,
	}
	if ev.Type == EventRecovered && ev.DownDuration > 0 {
		secs := int64(ev.DownDuration.Seconds())
		payload.DownDurationSecs = &secs
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook endpoint returned status %d", resp.StatusCode)
	}
	return nil
}
