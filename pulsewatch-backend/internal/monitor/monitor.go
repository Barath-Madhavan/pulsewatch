package monitor

import "time"

const (
	StatusUp       = "up"
	StatusDown     = "down"
	StatusDegraded = "degraded"
)

const (
	DefaultWorkerCount       = 10
	DefaultReconcileInterval = 30 * time.Second
	DefaultSlowThreshold     = 2 * time.Second
)
