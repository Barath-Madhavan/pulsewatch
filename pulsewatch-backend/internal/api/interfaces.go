package api

import (
	"context"
	"time"

	"pulsewatch-backend/internal/models"
	"pulsewatch-backend/internal/store"
)

// These interfaces exist purely so handlers can be unit-tested against a
// generated mock instead of a real Postgres pool. Each is narrowed to
// exactly the methods its handler calls (interface segregation), declared
// here in the consuming package per Go convention ("accept interfaces,
// return structs"). The concrete *store.XRepo types already satisfy them
// structurally, so nothing in the store package or in main.go's call
// sites needs to change.

//go:generate mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks

type MonitorRepository interface {
	Create(ctx context.Context, in models.CreateMonitorInput) (models.Monitor, error)
	Count(ctx context.Context) (int, error)
	List(ctx context.Context) ([]models.Monitor, error)
	GetByID(ctx context.Context, id string) (models.Monitor, error)
	Update(ctx context.Context, id string, in models.UpdateMonitorInput) (models.Monitor, error)
	Delete(ctx context.Context, id string) error
}

type IncidentRepository interface {
	List(ctx context.Context, monitorID string, limit, offset int) ([]models.Incident, int, error)
}

type AlertSettingsRepository interface {
	Get(ctx context.Context) (models.AlertSettings, error)
	Update(ctx context.Context, in models.UpdateAlertSettingsInput) (models.AlertSettings, error)
}

type CheckResultRepository interface {
	History(ctx context.Context, monitorID string, since time.Time, bucketWidth time.Duration) ([]store.HistoryBucket, error)
	UptimePercentage(ctx context.Context, monitorID string, since time.Time) (float64, error)
}
