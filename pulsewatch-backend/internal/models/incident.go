package models

import "time"

// Incident is one continuous stretch of downtime for a monitor, opened the
// moment it transitions healthy->unhealthy and resolved the moment it
// transitions back. ResolvedAt is nil while the incident is ongoing.
type Incident struct {
	ID          string     `json:"id"`
	MonitorID   string     `json:"monitor_id"`
	MonitorName string     `json:"monitor_name"`
	StartedAt   time.Time  `json:"started_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	Cause       string     `json:"cause,omitempty"`
}
