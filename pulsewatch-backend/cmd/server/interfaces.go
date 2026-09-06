package main

import (
	"context"

	"pulsewatch-backend/internal/models"
)

// Narrowed to exactly what newResultHandler calls, so it can be
// unit-tested against generated mocks instead of a real Postgres pool, a
// real WebSocket hub, and a real Dispatcher. The concrete *websocket.Hub,
// *store.CheckResultRepo, and *alert.Dispatcher types already satisfy
// these structurally, so main()'s own call site doesn't change.

//go:generate mockgen -source=interfaces.go -destination=mocks_test.go -package=main

type checkResultCreator interface {
	Create(ctx context.Context, result models.CheckResult) error
}

type broadcaster interface {
	Broadcast(v any)
}

type resultDispatcher interface {
	HandleResult(ctx context.Context, r models.CheckResult)
}
