package models

import "time"

// CheckResult is the outcome of a single HTTP check for a monitor.
type CheckResult struct {
	MonitorID      string    `json:"monitor_id"`
	Status         string    `json:"status"`
	StatusCode     int       `json:"status_code"`
	ResponseTimeMs int64     `json:"response_time_ms"`
	Error          string    `json:"error,omitempty"`
	CheckedAt      time.Time `json:"checked_at"`
}
