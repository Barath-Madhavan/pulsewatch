package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"pulsewatch-backend/internal/websocket"
)

func NewRouter(
	monitorHandler *MonitorHandler,
	historyHandler *HistoryHandler,
	alertSettingsHandler *AlertSettingsHandler,
	incidentHandler *IncidentHandler,
	hub *websocket.Hub,
	wsSnapshot func() [][]byte,
) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(Logging)
	r.Use(CORS)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.ServeWS(hub, w, r, wsSnapshot)
	})

	r.Route("/api/monitors", func(r chi.Router) {
		r.Get("/", monitorHandler.List)
		r.Post("/", monitorHandler.Create)
		r.Get("/{id}", monitorHandler.Get)
		r.Put("/{id}", monitorHandler.Update)
		r.Delete("/{id}", monitorHandler.Delete)
		r.Get("/{id}/history", historyHandler.Get)
		r.Get("/{id}/incidents", incidentHandler.ListForMonitor)
	})

	r.Route("/api/settings", func(r chi.Router) {
		r.Get("/alerts", alertSettingsHandler.Get)
		r.Put("/alerts", alertSettingsHandler.Update)
	})

	r.Get("/api/incidents", incidentHandler.List)

	return r
}
