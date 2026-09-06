package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"pulsewatch-backend/internal/models"
	"pulsewatch-backend/internal/store"
)

type MonitorHandler struct {
	repo        MonitorRepository
	maxMonitors int
}

// NewMonitorHandler takes maxMonitors as a hard cap on total monitor count
// (0 = unlimited), a "demo limit reached" guard so a publicly hosted
// instance can't be mass-filled with monitors by a stranger.
func NewMonitorHandler(repo MonitorRepository, maxMonitors int) *MonitorHandler {
	return &MonitorHandler{repo: repo, maxMonitors: maxMonitors}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (h *MonitorHandler) List(w http.ResponseWriter, r *http.Request) {
	monitors, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list monitors")
		return
	}
	writeJSON(w, http.StatusOK, monitors)
}

func (h *MonitorHandler) Create(w http.ResponseWriter, r *http.Request) {
	if h.maxMonitors > 0 {
		count, err := h.repo.Count(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to check monitor count")
			return
		}
		if count >= h.maxMonitors {
			writeError(w, http.StatusForbidden, "Demo limit reached. Reach out if you'd like a full live walkthrough.")
			return
		}
	}

	var in models.CreateMonitorInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if err := validateName(in.Name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}
	if err := validateHTTPURL(in.URL); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if in.Method == "" {
		in.Method = "GET"
	}
	normalizedMethod, err := validateMethod(in.Method)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	in.Method = normalizedMethod

	if in.IntervalSeconds <= 0 {
		in.IntervalSeconds = 60
	}
	timeoutProvided := in.TimeoutSeconds > 0
	if in.TimeoutSeconds <= 0 {
		in.TimeoutSeconds = 10
	}
	if in.TimeoutSeconds > in.IntervalSeconds {
		if timeoutProvided {
			// The caller explicitly asked for this combination, so reject
			// it rather than silently changing what they typed.
			writeError(w, http.StatusBadRequest, "timeout_seconds cannot be greater than interval_seconds")
			return
		}
		// Just the platform default (10s) not fitting a short custom
		// interval, so clamp it down instead of rejecting an edit that
		// never mentioned timeout at all.
		in.TimeoutSeconds = in.IntervalSeconds
	}

	if in.ExpectedStatusCode <= 0 {
		in.ExpectedStatusCode = 200
	} else if err := validateStatusCode(in.ExpectedStatusCode); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	monitor, err := h.repo.Create(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create monitor")
		return
	}
	writeJSON(w, http.StatusCreated, monitor)
}

func (h *MonitorHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !requireValidID(w, id) {
		return
	}
	monitor, err := h.repo.GetByID(r.Context(), id)
	if errors.Is(err, store.ErrMonitorNotFound) {
		writeError(w, http.StatusNotFound, "monitor not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get monitor")
		return
	}
	writeJSON(w, http.StatusOK, monitor)
}

func (h *MonitorHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !requireValidID(w, id) {
		return
	}

	var in models.UpdateMonitorInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if in.Name != nil {
		if *in.Name == "" {
			writeError(w, http.StatusBadRequest, "name cannot be empty")
			return
		}
		if err := validateName(*in.Name); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if in.URL != nil {
		if *in.URL == "" {
			writeError(w, http.StatusBadRequest, "url cannot be empty")
			return
		}
		if err := validateHTTPURL(*in.URL); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if in.Method != nil {
		normalized, err := validateMethod(*in.Method)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		in.Method = &normalized
	}
	if in.ExpectedStatusCode != nil {
		if err := validateStatusCode(*in.ExpectedStatusCode); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if in.IntervalSeconds != nil && *in.IntervalSeconds <= 0 {
		writeError(w, http.StatusBadRequest, "interval_seconds must be positive")
		return
	}
	if in.TimeoutSeconds != nil && *in.TimeoutSeconds <= 0 {
		writeError(w, http.StatusBadRequest, "timeout_seconds must be positive")
		return
	}

	// The timeout<=interval check needs the *effective* values, including
	// whichever of the two isn't being changed by this request.
	current, err := h.repo.GetByID(r.Context(), id)
	if errors.Is(err, store.ErrMonitorNotFound) {
		writeError(w, http.StatusNotFound, "monitor not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load monitor")
		return
	}
	effectiveInterval := current.IntervalSeconds
	if in.IntervalSeconds != nil {
		effectiveInterval = *in.IntervalSeconds
	}
	effectiveTimeout := current.TimeoutSeconds
	if in.TimeoutSeconds != nil {
		effectiveTimeout = *in.TimeoutSeconds
	}
	if effectiveTimeout > effectiveInterval {
		if in.TimeoutSeconds != nil {
			// Timeout was explicitly part of this edit and doesn't fit,
			// which is a real mistake worth surfacing.
			writeError(w, http.StatusBadRequest, "timeout_seconds cannot be greater than interval_seconds")
			return
		}
		// Only interval shrank below the untouched, carried-over timeout,
		// so clamp the timeout down rather than rejecting an edit that
		// never mentioned it.
		effectiveTimeout = effectiveInterval
		in.TimeoutSeconds = &effectiveTimeout
	}

	monitor, err := h.repo.Update(r.Context(), id, in)
	if errors.Is(err, store.ErrMonitorNotFound) {
		writeError(w, http.StatusNotFound, "monitor not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update monitor")
		return
	}
	writeJSON(w, http.StatusOK, monitor)
}

func (h *MonitorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !requireValidID(w, id) {
		return
	}
	err := h.repo.Delete(r.Context(), id)
	if errors.Is(err, store.ErrMonitorNotFound) {
		writeError(w, http.StatusNotFound, "monitor not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete monitor")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
