package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const defaultResendURL = "https://api.resend.com/emails"

// EmailSender sends alert emails through Resend's HTTP API.
type EmailSender struct {
	apiKey   string
	from     string
	baseURL  string
	client   *http.Client
	demoMode bool
}

// NewEmailSender builds a sender. baseURL is overridable (for tests
// pointing at a mock server); it defaults to the real Resend endpoint. In
// demoMode, Send never actually calls Resend. See the demoMode check
// below for why.
func NewEmailSender(apiKey, from, baseURL string, demoMode bool) *EmailSender {
	if baseURL == "" {
		baseURL = defaultResendURL
	}
	return &EmailSender{
		apiKey:   apiKey,
		from:     from,
		baseURL:  baseURL,
		client:   &http.Client{Timeout: 10 * time.Second},
		demoMode: demoMode,
	}
}

func (s *EmailSender) Send(ctx context.Context, to string, ev Event) error {
	subject, text := formatEmail(ev)

	// A publicly hosted demo can't be allowed to actually email whatever
	// address a visitor types in, since that's a free spam relay pointed
	// at strangers. Log what would have been sent and stop here; every
	// other part of the feature (the form, validation, saving) stays
	// fully real.
	if s.demoMode {
		log.Printf("[DEMO] would send email to %s: %q", to, subject)
		return nil
	}

	if s.apiKey == "" {
		return fmt.Errorf("email sending not configured: RESEND_API_KEY is empty")
	}
	if to == "" {
		return fmt.Errorf("no recipient email address configured")
	}

	payload, err := json.Marshal(map[string]any{
		"from":    s.from,
		"to":      []string{to},
		"subject": subject,
		"text":    text,
	})
	if err != nil {
		return fmt.Errorf("marshal email payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build email request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("resend API returned %d: %s", resp.StatusCode, body)
	}
	return nil
}

func formatEmail(ev Event) (subject, body string) {
	switch ev.Type {
	case EventDown:
		subject = fmt.Sprintf("\U0001F534 %s is down", ev.Monitor.Name)
		body = fmt.Sprintf(
			"%s (%s) is not responding as expected.\n\nStatus code: %d\nError: %s\nChecked at: %s\n",
			ev.Monitor.Name, ev.Monitor.URL, ev.StatusCode, orNone(ev.Error), ev.CheckedAt.Format(time.RFC1123),
		)
	case EventRecovered:
		subject = fmt.Sprintf("✅ %s is back up", ev.Monitor.Name)
		body = fmt.Sprintf(
			"%s (%s) is responding normally again.\n\nDowntime: %s\nRecovered at: %s\n",
			ev.Monitor.Name, ev.Monitor.URL, formatDuration(ev.DownDuration), ev.CheckedAt.Format(time.RFC1123),
		)
	}
	return subject, body
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "unknown"
	}
	return d.Round(time.Second).String()
}
