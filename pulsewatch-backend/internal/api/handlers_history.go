package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"pulsewatch-backend/internal/store"
)

type HistoryHandler struct {
	checkResults CheckResultRepository
}

func NewHistoryHandler(checkResults CheckResultRepository) *HistoryHandler {
	return &HistoryHandler{checkResults: checkResults}
}

type historyRangeConfig struct {
	lookback time.Duration
	bucket   time.Duration
}

// Bucket widths are chosen so each range returns a manageable number of
// points for a chart: ~288 for 24h, ~168 for 7d, ~120 for 30d.
var historyRanges = map[string]historyRangeConfig{
	"24h": {lookback: 24 * time.Hour, bucket: 5 * time.Minute},
	"7d":  {lookback: 7 * 24 * time.Hour, bucket: time.Hour},
	"30d": {lookback: 30 * 24 * time.Hour, bucket: 6 * time.Hour},
}

type historyResponse struct {
	Range            string                `json:"range"`
	Points           []store.HistoryBucket `json:"points"`
	UptimePercentage float64               `json:"uptime_percentage"`
}

func (h *HistoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !requireValidID(w, id) {
		return
	}

	rangeParam := r.URL.Query().Get("range")
	if rangeParam == "" {
		rangeParam = "24h"
	}
	cfg, ok := historyRanges[rangeParam]
	if !ok {
		writeError(w, http.StatusBadRequest, "range must be one of 24h, 7d, 30d")
		return
	}

	since := time.Now().Add(-cfg.lookback)

	points, err := h.checkResults.History(r.Context(), id, since, cfg.bucket)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load history")
		return
	}

	uptime, err := h.checkResults.UptimePercentage(r.Context(), id, since)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to compute uptime")
		return
	}

	writeJSON(w, http.StatusOK, historyResponse{
		Range:            rangeParam,
		Points:           points,
		UptimePercentage: uptime,
	})
}
