package models

import "time"

type Monitor struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	URL                string    `json:"url"`
	Method             string    `json:"method"`
	IntervalSeconds    int       `json:"interval_seconds"`
	TimeoutSeconds     int       `json:"timeout_seconds"`
	ExpectedStatusCode int       `json:"expected_status_code"`
	IsActive           bool      `json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type CreateMonitorInput struct {
	Name               string `json:"name"`
	URL                string `json:"url"`
	Method             string `json:"method"`
	IntervalSeconds    int    `json:"interval_seconds"`
	TimeoutSeconds     int    `json:"timeout_seconds"`
	ExpectedStatusCode int    `json:"expected_status_code"`
}

type UpdateMonitorInput struct {
	Name               *string `json:"name"`
	URL                *string `json:"url"`
	Method             *string `json:"method"`
	IntervalSeconds    *int    `json:"interval_seconds"`
	TimeoutSeconds     *int    `json:"timeout_seconds"`
	ExpectedStatusCode *int    `json:"expected_status_code"`
	IsActive           *bool   `json:"is_active"`
}
