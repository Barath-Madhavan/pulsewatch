package api

import (
	"encoding/json"
	"net/http"
	"net/mail"

	"pulsewatch-backend/internal/models"
)

type AlertSettingsHandler struct {
	repo     AlertSettingsRepository
	demoMode bool
}

// NewAlertSettingsHandler takes demoMode so it can tell the frontend
// whether this deployment simulates alert sends, letting the Settings
// page show a banner explaining why alerts won't actually arrive.
func NewAlertSettingsHandler(repo AlertSettingsRepository, demoMode bool) *AlertSettingsHandler {
	return &AlertSettingsHandler{repo: repo, demoMode: demoMode}
}

// alertSettingsResponse embeds the stored settings and adds the read-only
// demo_mode flag, which isn't part of the DB row; it's server config.
type alertSettingsResponse struct {
	models.AlertSettings
	DemoMode bool `json:"demo_mode"`
}

func (h *AlertSettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	settings, err := h.repo.Get(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load alert settings")
		return
	}
	writeJSON(w, http.StatusOK, alertSettingsResponse{AlertSettings: settings, DemoMode: h.demoMode})
}

func (h *AlertSettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var in models.UpdateAlertSettingsInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if in.EmailEnabled {
		if in.EmailAddress == "" {
			writeError(w, http.StatusBadRequest, "email_address is required when email_enabled is true")
			return
		}
		if _, err := mail.ParseAddress(in.EmailAddress); err != nil {
			writeError(w, http.StatusBadRequest, "email_address is not a valid email address")
			return
		}
	}
	if in.WebhookEnabled {
		if in.WebhookURL == "" {
			writeError(w, http.StatusBadRequest, "webhook_url is required when webhook_enabled is true")
			return
		}
		if err := validateHTTPURL(in.WebhookURL); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	settings, err := h.repo.Update(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update alert settings")
		return
	}
	writeJSON(w, http.StatusOK, alertSettingsResponse{AlertSettings: settings, DemoMode: h.demoMode})
}
