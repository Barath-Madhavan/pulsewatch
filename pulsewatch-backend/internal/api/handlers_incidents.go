package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"pulsewatch-backend/internal/models"
)

type IncidentHandler struct {
	repo IncidentRepository
}

func NewIncidentHandler(repo IncidentRepository) *IncidentHandler {
	return &IncidentHandler{repo: repo}
}

const (
	defaultIncidentLimit = 10
	maxIncidentLimit     = 200
)

type incidentPage struct {
	Incidents []models.Incident `json:"incidents"`
	Total     int               `json:"total"`
}

// List handles GET /api/incidents, the global feed across every monitor.
func (h *IncidentHandler) List(w http.ResponseWriter, r *http.Request) {
	incidents, total, err := h.repo.List(r.Context(), "", parseLimit(r), parseOffset(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list incidents")
		return
	}
	writeJSON(w, http.StatusOK, incidentPage{Incidents: incidents, Total: total})
}

// ListForMonitor handles GET /api/monitors/{id}/incidents.
func (h *IncidentHandler) ListForMonitor(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !requireValidID(w, id) {
		return
	}
	incidents, total, err := h.repo.List(r.Context(), id, parseLimit(r), parseOffset(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list incidents")
		return
	}
	writeJSON(w, http.StatusOK, incidentPage{Incidents: incidents, Total: total})
}

func parseLimit(r *http.Request) int {
	v := r.URL.Query().Get("limit")
	if v == "" {
		return defaultIncidentLimit
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return defaultIncidentLimit
	}
	if n > maxIncidentLimit {
		return maxIncidentLimit
	}
	return n
}

func parseOffset(r *http.Request) int {
	v := r.URL.Query().Get("offset")
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0
	}
	return n
}
