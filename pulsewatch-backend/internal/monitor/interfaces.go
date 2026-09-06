package monitor

import (
	"context"

	"pulsewatch-backend/internal/models"
)

// MonitorLister is the one method Scheduler needs from a monitor repo,
// declared here so Scheduler can be unit-tested against a generated mock
// instead of a real Postgres pool. *store.MonitorRepo already satisfies
// this structurally.

//go:generate mockgen -source=interfaces.go -destination=mocks/mocks.go -package=mocks

type MonitorLister interface {
	List(ctx context.Context) ([]models.Monitor, error)
}
