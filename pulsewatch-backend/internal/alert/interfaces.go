package alert

import (
	"context"
	"time"

	"pulsewatch-backend/internal/models"
)

// Narrowed to exactly what Dispatcher calls, so it can be unit-tested
// against a generated mock instead of a real Postgres pool and real
// senders. The concrete *store.XRepo / *EmailSender / *WebhookSender
// types already satisfy these structurally; main.go's call sites don't
// need to change.

//go:generate mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks

type MonitorLookup interface {
	GetByID(ctx context.Context, id string) (models.Monitor, error)
}

type AlertSettingsStore interface {
	Get(ctx context.Context) (models.AlertSettings, error)
}

type IncidentStore interface {
	Open(ctx context.Context, monitorID string, startedAt time.Time, cause string) error
	Resolve(ctx context.Context, monitorID string, resolvedAt time.Time) error
	OpenIncidents(ctx context.Context) ([]models.Incident, error)
}

// Sender is satisfied by both *EmailSender and *WebhookSender, whose Send
// signatures already match exactly (only the meaning of "target" differs:
// an email address vs. a URL).
type Sender interface {
	Send(ctx context.Context, target string, ev Event) error
}
