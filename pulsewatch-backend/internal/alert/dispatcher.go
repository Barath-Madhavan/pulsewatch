package alert

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"pulsewatch-backend/internal/models"
	"pulsewatch-backend/internal/monitor"
	"pulsewatch-backend/internal/store"
)

type healthState struct {
	healthy   bool
	changedAt time.Time
}

// Dispatcher watches check results for healthy<->unhealthy transitions and,
// on one, unconditionally records an incident and, if the user has
// configured a channel, sends an alert. "Degraded" counts as healthy here;
// only genuine downtime opens an incident or fires a notification, so a
// slow-but-reachable endpoint doesn't spam either.
//
// A monitor's very first observed result never triggers either, even if
// it's down: with no prior state, there's no way to tell a fresh failure
// from a monitor that's simply been down since before anyone was watching.
type Dispatcher struct {
	monitorRepo  MonitorLookup
	settingsRepo AlertSettingsStore
	incidents    IncidentStore
	email        Sender
	webhook      Sender

	mu    sync.Mutex
	state map[string]healthState
}

func NewDispatcher(
	monitorRepo MonitorLookup,
	settingsRepo AlertSettingsStore,
	incidents IncidentStore,
	email Sender,
	webhook Sender,
) *Dispatcher {
	return &Dispatcher{
		monitorRepo:  monitorRepo,
		settingsRepo: settingsRepo,
		incidents:    incidents,
		email:        email,
		webhook:      webhook,
		state:        make(map[string]healthState),
	}
}

// Seed loads currently-open incidents so a server restart doesn't lose
// track of a monitor that was already down. Without this, the in-memory
// state map starts empty, the next check for a still-down monitor looks
// like a fresh "first observation" (see the type doc above), and the
// eventual recovery is never noticed, so the incident opened before the
// restart is orphaned open forever and no recovery alert ever fires.
// Call once, after construction and before the checker starts running.
func (d *Dispatcher) Seed(ctx context.Context) error {
	if d.incidents == nil {
		return nil
	}
	open, err := d.incidents.OpenIncidents(ctx)
	if err != nil {
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	for _, inc := range open {
		d.state[inc.MonitorID] = healthState{healthy: false, changedAt: inc.StartedAt}
	}
	return nil
}

func (d *Dispatcher) HandleResult(ctx context.Context, result models.CheckResult) {
	healthy := result.Status != monitor.StatusDown

	d.mu.Lock()
	prev, existed := d.state[result.MonitorID]
	transitioned := existed && prev.healthy != healthy

	changedAt := prev.changedAt
	if !existed || transitioned {
		changedAt = result.CheckedAt
	}
	d.state[result.MonitorID] = healthState{healthy: healthy, changedAt: changedAt}
	d.mu.Unlock()

	if !transitioned {
		return
	}

	var downDuration time.Duration
	if healthy {
		downDuration = result.CheckedAt.Sub(prev.changedAt)
	}

	go d.handleTransition(ctx, result, healthy, downDuration)
}

func (d *Dispatcher) handleTransition(ctx context.Context, result models.CheckResult, healthy bool, downDuration time.Duration) {
	// Incident history shouldn't depend on whether the user has bothered
	// to configure alerting, so this runs unconditionally, before the
	// settings-gated alert sending below.
	d.recordIncident(ctx, result, healthy)
	d.sendAlerts(ctx, result, healthy, downDuration)
}

func (d *Dispatcher) recordIncident(ctx context.Context, result models.CheckResult, healthy bool) {
	if d.incidents == nil {
		return
	}

	if healthy {
		if err := d.incidents.Resolve(ctx, result.MonitorID, result.CheckedAt); err != nil && !errors.Is(err, store.ErrNoOpenIncident) {
			log.Printf("incident: failed to resolve for monitor %s: %v", result.MonitorID, err)
		}
		return
	}

	cause := result.Error
	if cause == "" {
		cause = fmt.Sprintf("unexpected status code %d", result.StatusCode)
	}
	if err := d.incidents.Open(ctx, result.MonitorID, result.CheckedAt, cause); err != nil {
		log.Printf("incident: failed to open for monitor %s: %v", result.MonitorID, err)
	}
}

func (d *Dispatcher) sendAlerts(ctx context.Context, result models.CheckResult, healthy bool, downDuration time.Duration) {
	settings, err := d.settingsRepo.Get(ctx)
	if err != nil {
		log.Printf("alert: failed to load settings: %v", err)
		return
	}
	if !settings.EmailEnabled && !settings.WebhookEnabled {
		return
	}

	mon, err := d.monitorRepo.GetByID(ctx, result.MonitorID)
	if err != nil {
		log.Printf("alert: failed to load monitor %s: %v", result.MonitorID, err)
		return
	}

	ev := Event{
		Monitor:      mon,
		StatusCode:   result.StatusCode,
		Error:        result.Error,
		CheckedAt:    result.CheckedAt,
		DownDuration: downDuration,
	}
	if healthy {
		ev.Type = EventRecovered
	} else {
		ev.Type = EventDown
	}

	if settings.EmailEnabled && d.email != nil {
		if err := d.email.Send(ctx, settings.EmailAddress, ev); err != nil {
			log.Printf("alert: email send failed for monitor %s: %v", mon.ID, err)
		} else {
			log.Printf("alert: email sent for monitor %s (%s)", mon.ID, ev.Type)
		}
	}
	if settings.WebhookEnabled && d.webhook != nil {
		if err := d.webhook.Send(ctx, settings.WebhookURL, ev); err != nil {
			log.Printf("alert: webhook send failed for monitor %s: %v", mon.ID, err)
		} else {
			log.Printf("alert: webhook sent for monitor %s (%s)", mon.ID, ev.Type)
		}
	}
}
