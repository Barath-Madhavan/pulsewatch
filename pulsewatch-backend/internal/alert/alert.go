package alert

import (
	"time"

	"pulsewatch-backend/internal/models"
)

type EventType string

const (
	EventDown      EventType = "down"
	EventRecovered EventType = "recovered"
)

// Event describes a monitor's healthy<->unhealthy transition, the only
// thing that triggers a notification. DownDuration is only meaningful on
// EventRecovered.
type Event struct {
	Type         EventType
	Monitor      models.Monitor
	StatusCode   int
	Error        string
	CheckedAt    time.Time
	DownDuration time.Duration
}
