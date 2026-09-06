package models

import "time"

// AlertSettings is global config (not per-monitor) for where down/recovery
// notifications go.
type AlertSettings struct {
	EmailEnabled   bool      `json:"email_enabled"`
	EmailAddress   string    `json:"email_address"`
	WebhookEnabled bool      `json:"webhook_enabled"`
	WebhookURL     string    `json:"webhook_url"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type UpdateAlertSettingsInput struct {
	EmailEnabled   bool   `json:"email_enabled"`
	EmailAddress   string `json:"email_address"`
	WebhookEnabled bool   `json:"webhook_enabled"`
	WebhookURL     string `json:"webhook_url"`
}
