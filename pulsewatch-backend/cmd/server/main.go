package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"pulsewatch-backend/internal/alert"
	"pulsewatch-backend/internal/api"
	"pulsewatch-backend/internal/config"
	"pulsewatch-backend/internal/models"
	"pulsewatch-backend/internal/monitor"
	"pulsewatch-backend/internal/store"
	"pulsewatch-backend/internal/websocket"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}

	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbPool, err := store.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	hub := websocket.NewHub()
	go hub.Run(ctx)

	results := newResultCache()

	monitorRepo := store.NewMonitorRepo(dbPool)
	checkResultRepo := store.NewCheckResultRepo(dbPool)
	alertSettingsRepo := store.NewAlertSettingsRepo(dbPool)
	incidentRepo := store.NewIncidentRepo(dbPool)

	emailSender := alert.NewEmailSender(cfg.ResendAPIKey, cfg.ResendFromEmail, cfg.ResendAPIURL, cfg.DemoMode)
	webhookSender := alert.NewWebhookSender(cfg.DemoMode)
	dispatcher := alert.NewDispatcher(monitorRepo, alertSettingsRepo, incidentRepo, emailSender, webhookSender)
	if err := dispatcher.Seed(ctx); err != nil {
		log.Printf("failed to seed alert dispatcher from open incidents: %v", err)
	}

	monitorHandler := api.NewMonitorHandler(monitorRepo, cfg.MaxMonitors)
	historyHandler := api.NewHistoryHandler(checkResultRepo)
	alertSettingsHandler := api.NewAlertSettingsHandler(alertSettingsRepo, cfg.DemoMode)
	incidentHandler := api.NewIncidentHandler(incidentRepo)
	router := api.NewRouter(monitorHandler, historyHandler, alertSettingsHandler, incidentHandler, hub, results.snapshot)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("pulsewatch backend listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	checker := monitor.NewChecker(monitor.DefaultSlowThreshold)
	checkPool := monitor.NewPool(checker, monitor.DefaultWorkerCount, newResultHandler(ctx, hub, results, checkResultRepo, dispatcher))
	checkPool.Start(ctx)

	scheduler := monitor.NewScheduler(monitorRepo, checkPool, monitor.DefaultReconcileInterval)
	go scheduler.Start(ctx)

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

// statusUpdateMessage is the envelope sent to every connected dashboard.
// The "type" field lets the frontend distinguish this from other message
// kinds (e.g. incident events) added in later steps, on the same socket.
type statusUpdateMessage struct {
	Type    string             `json:"type"`
	Payload models.CheckResult `json:"payload"`
}

// resultCache holds the most recent CheckResult per monitor so a newly
// connected dashboard can be caught up immediately instead of waiting for
// each monitor's next check cycle.
type resultCache struct {
	mu      sync.Mutex
	results map[string]models.CheckResult
}

func newResultCache() *resultCache {
	return &resultCache{results: make(map[string]models.CheckResult)}
}

func (c *resultCache) set(r models.CheckResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results[r.MonitorID] = r
}

func (c *resultCache) snapshot() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()

	messages := make([][]byte, 0, len(c.results))
	for _, r := range c.results {
		data, err := json.Marshal(statusUpdateMessage{Type: "status_update", Payload: r})
		if err != nil {
			continue
		}
		messages = append(messages, data)
	}
	return messages
}

// newResultHandler logs each check result, persists it, caches it as the
// latest known state, broadcasts it to every connected WebSocket client,
// and hands it to the alert dispatcher to check for a healthy<->unhealthy
// transition. Persistence uses the app's root context (not a request
// context) since it runs on a background worker goroutine, not in
// response to an HTTP call.
func newResultHandler(ctx context.Context, hub broadcaster, cache *resultCache, checkResults checkResultCreator, dispatcher resultDispatcher) monitor.ResultHandler {
	return func(r models.CheckResult) {
		if r.Error != "" {
			log.Printf("[check] monitor=%s status=%s time=%dms error=%q", r.MonitorID, r.Status, r.ResponseTimeMs, r.Error)
		} else {
			log.Printf("[check] monitor=%s status=%s code=%d time=%dms", r.MonitorID, r.Status, r.StatusCode, r.ResponseTimeMs)
		}

		if err := checkResults.Create(ctx, r); err != nil {
			log.Printf("failed to persist check result for monitor %s: %v", r.MonitorID, err)
		}

		cache.set(r)
		hub.Broadcast(statusUpdateMessage{Type: "status_update", Payload: r})
		dispatcher.HandleResult(ctx, r)
	}
}
