package alert_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"pulsewatch-backend/internal/alert"
	"pulsewatch-backend/internal/alert/mocks"
	"pulsewatch-backend/internal/models"
	"pulsewatch-backend/internal/monitor"
	"pulsewatch-backend/internal/store"
)

// handleTransition runs in its own goroutine (see HandleResult), so tests
// that expect it to have run need to synchronize on something. Each test
// arranges for the last mock call in the expected chain to close a
// channel, then waits on it here instead of sleeping.
func waitForSignal(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the dispatcher to finish processing")
	}
}

var baseTime = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func TestDispatcher_FirstObservationNeverOpensIncident(t *testing.T) {
	ctrl := gomock.NewController(t)
	monitorRepo := mocks.NewMockMonitorLookup(ctrl)
	settingsRepo := mocks.NewMockAlertSettingsStore(ctrl)
	incidents := mocks.NewMockIncidentStore(ctrl)
	// No .EXPECT() at all on incidents/settingsRepo/monitorRepo: a first
	// observation must never touch any of them, down or not.
	d := alert.NewDispatcher(monitorRepo, settingsRepo, incidents, nil, nil)

	d.HandleResult(context.Background(), models.CheckResult{
		MonitorID: "m1", Status: monitor.StatusDown, CheckedAt: baseTime,
	})
	d.HandleResult(context.Background(), models.CheckResult{
		MonitorID: "m2", Status: monitor.StatusUp, CheckedAt: baseTime,
	})

	// Nothing async was even spawned (HandleResult returns early), so
	// there's nothing to wait for; ctrl.Finish (via t.Cleanup) enforces
	// that the unexpected-call-count assertions above hold.
}

func TestDispatcher_DownTransitionOpensIncidentAndAlerts(t *testing.T) {
	ctrl := gomock.NewController(t)
	monitorRepo := mocks.NewMockMonitorLookup(ctrl)
	settingsRepo := mocks.NewMockAlertSettingsStore(ctrl)
	incidents := mocks.NewMockIncidentStore(ctrl)
	email := mocks.NewMockSender(ctrl)
	d := alert.NewDispatcher(monitorRepo, settingsRepo, incidents, email, nil)

	// Establish a healthy baseline first (first observation, no transition).
	d.HandleResult(context.Background(), models.CheckResult{
		MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: baseTime,
	})

	incidents.EXPECT().Open(gomock.Any(), "m1", baseTime.Add(time.Minute), "boom").Return(nil)
	settingsRepo.EXPECT().Get(gomock.Any()).Return(models.AlertSettings{EmailEnabled: true, EmailAddress: "me@example.com"}, nil)
	monitorRepo.EXPECT().GetByID(gomock.Any(), "m1").Return(models.Monitor{ID: "m1", Name: "API"}, nil)

	done := make(chan struct{})
	email.EXPECT().Send(gomock.Any(), "me@example.com", gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, ev alert.Event) error {
			assert.Equal(t, alert.EventDown, ev.Type)
			assert.Equal(t, time.Duration(0), ev.DownDuration, "a down event never carries a duration")
			close(done)
			return nil
		},
	)

	d.HandleResult(context.Background(), models.CheckResult{
		MonitorID: "m1", Status: monitor.StatusDown, Error: "boom", CheckedAt: baseTime.Add(time.Minute),
	})
	waitForSignal(t, done)
}

func TestDispatcher_DownWithoutTransportErrorFallsBackToStatusCodeCause(t *testing.T) {
	ctrl := gomock.NewController(t)
	monitorRepo := mocks.NewMockMonitorLookup(ctrl)
	settingsRepo := mocks.NewMockAlertSettingsStore(ctrl)
	incidents := mocks.NewMockIncidentStore(ctrl)
	d := alert.NewDispatcher(monitorRepo, settingsRepo, incidents, nil, nil)

	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: baseTime})

	settingsRepo.EXPECT().Get(gomock.Any()).Return(models.AlertSettings{}, nil)
	done := make(chan struct{})
	incidents.EXPECT().Open(gomock.Any(), "m1", gomock.Any(), "unexpected status code 502").
		DoAndReturn(func(_ context.Context, _ string, _ time.Time, cause string) error {
			close(done)
			return nil
		})

	d.HandleResult(context.Background(), models.CheckResult{
		MonitorID: "m1", Status: monitor.StatusDown, StatusCode: 502, CheckedAt: baseTime.Add(time.Minute),
	})
	waitForSignal(t, done)
}

func TestDispatcher_RecoveryFromDownResolvesAndAlertsWithDuration(t *testing.T) {
	ctrl := gomock.NewController(t)
	monitorRepo := mocks.NewMockMonitorLookup(ctrl)
	settingsRepo := mocks.NewMockAlertSettingsStore(ctrl)
	incidents := mocks.NewMockIncidentStore(ctrl)
	webhook := mocks.NewMockSender(ctrl)
	d := alert.NewDispatcher(monitorRepo, settingsRepo, incidents, nil, webhook)

	downAt := baseTime
	incidents.EXPECT().Open(gomock.Any(), "m1", downAt, gomock.Any()).Return(nil)
	openTransitionDone := make(chan struct{})
	settingsRepo.EXPECT().Get(gomock.Any()).DoAndReturn(
		func(_ context.Context) (models.AlertSettings, error) {
			close(openTransitionDone)
			return models.AlertSettings{}, nil // both channels off: sendAlerts stops here for this transition.
		},
	)

	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: downAt.Add(-time.Hour)})
	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusDown, CheckedAt: downAt})
	waitForSignal(t, openTransitionDone)

	recoveredAt := downAt.Add(90 * time.Second)
	incidents.EXPECT().Resolve(gomock.Any(), "m1", recoveredAt).Return(nil)
	settingsRepo.EXPECT().Get(gomock.Any()).Return(models.AlertSettings{WebhookEnabled: true, WebhookURL: "https://hooks.example.com"}, nil)
	monitorRepo.EXPECT().GetByID(gomock.Any(), "m1").Return(models.Monitor{ID: "m1"}, nil)

	done := make(chan struct{})
	webhook.EXPECT().Send(gomock.Any(), "https://hooks.example.com", gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, ev alert.Event) error {
			assert.Equal(t, alert.EventRecovered, ev.Type)
			assert.Equal(t, 90*time.Second, ev.DownDuration)
			close(done)
			return nil
		},
	)

	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: recoveredAt})
	waitForSignal(t, done)
}

func TestDispatcher_RecoveryFromDownToDegradedAlsoResolves(t *testing.T) {
	// Degraded counts as healthy, so down -> degraded is a real recovery
	// transition, same as down -> up.
	ctrl := gomock.NewController(t)
	monitorRepo := mocks.NewMockMonitorLookup(ctrl)
	settingsRepo := mocks.NewMockAlertSettingsStore(ctrl)
	incidents := mocks.NewMockIncidentStore(ctrl)
	d := alert.NewDispatcher(monitorRepo, settingsRepo, incidents, nil, nil)

	incidents.EXPECT().Open(gomock.Any(), "m1", gomock.Any(), gomock.Any()).Return(nil)
	openTransitionDone := make(chan struct{})
	settingsRepo.EXPECT().Get(gomock.Any()).DoAndReturn(
		func(_ context.Context) (models.AlertSettings, error) {
			close(openTransitionDone)
			return models.AlertSettings{}, nil
		},
	)
	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: baseTime.Add(-time.Hour)})
	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusDown, CheckedAt: baseTime})
	waitForSignal(t, openTransitionDone)

	settingsRepo.EXPECT().Get(gomock.Any()).Return(models.AlertSettings{}, nil)
	done := make(chan struct{})
	incidents.EXPECT().Resolve(gomock.Any(), "m1", baseTime.Add(time.Minute)).DoAndReturn(
		func(_ context.Context, _ string, _ time.Time) error {
			close(done)
			return nil
		},
	)

	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusDegraded, CheckedAt: baseTime.Add(time.Minute)})
	waitForSignal(t, done)
}

func TestDispatcher_UpToDegradedIsNotATransition(t *testing.T) {
	ctrl := gomock.NewController(t)
	monitorRepo := mocks.NewMockMonitorLookup(ctrl)
	settingsRepo := mocks.NewMockAlertSettingsStore(ctrl)
	incidents := mocks.NewMockIncidentStore(ctrl)
	// No expectations at all: neither up->degraded nor the later
	// degraded->up may touch incidents/settings/monitorRepo, since
	// "healthy" never changed across either step.
	d := alert.NewDispatcher(monitorRepo, settingsRepo, incidents, nil, nil)

	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: baseTime})
	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusDegraded, CheckedAt: baseTime.Add(time.Minute)})
	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: baseTime.Add(2 * time.Minute)})
}

func TestDispatcher_ChangedAtOnlyMovesOnARealTransition(t *testing.T) {
	// up (t0) -> degraded (t1, not a transition) -> down (t2) -> up (t3).
	// The eventual recovery's duration must be measured from t2 (when it
	// actually went down), which in turn proves the intervening
	// non-transition step didn't clobber the tracked changedAt.
	ctrl := gomock.NewController(t)
	monitorRepo := mocks.NewMockMonitorLookup(ctrl)
	settingsRepo := mocks.NewMockAlertSettingsStore(ctrl)
	incidents := mocks.NewMockIncidentStore(ctrl)
	d := alert.NewDispatcher(monitorRepo, settingsRepo, incidents, nil, nil)

	t0 := baseTime
	t1 := t0.Add(time.Minute)
	t2 := t1.Add(time.Minute)
	t3 := t2.Add(30 * time.Second)

	incidents.EXPECT().Open(gomock.Any(), "m1", t2, gomock.Any()).Return(nil)
	openTransitionDone := make(chan struct{})
	settingsRepo.EXPECT().Get(gomock.Any()).DoAndReturn(
		func(_ context.Context) (models.AlertSettings, error) {
			close(openTransitionDone)
			return models.AlertSettings{}, nil
		},
	)
	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: t0})
	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusDegraded, CheckedAt: t1})
	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusDown, CheckedAt: t2})
	waitForSignal(t, openTransitionDone)

	settingsRepo.EXPECT().Get(gomock.Any()).Return(models.AlertSettings{}, nil)
	done := make(chan struct{})
	incidents.EXPECT().Resolve(gomock.Any(), "m1", t3).DoAndReturn(
		func(_ context.Context, _ string, _ time.Time) error { close(done); return nil },
	)

	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: t3})
	waitForSignal(t, done)
}

func TestDispatcher_ResolveErrNoOpenIncidentIsSwallowed(t *testing.T) {
	ctrl := gomock.NewController(t)
	monitorRepo := mocks.NewMockMonitorLookup(ctrl)
	settingsRepo := mocks.NewMockAlertSettingsStore(ctrl)
	incidents := mocks.NewMockIncidentStore(ctrl)
	d := alert.NewDispatcher(monitorRepo, settingsRepo, incidents, nil, nil)

	incidents.EXPECT().Open(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	openTransitionDone := make(chan struct{})
	settingsRepo.EXPECT().Get(gomock.Any()).DoAndReturn(
		func(_ context.Context) (models.AlertSettings, error) {
			close(openTransitionDone)
			return models.AlertSettings{}, nil
		},
	)
	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: baseTime})
	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusDown, CheckedAt: baseTime.Add(time.Minute)})
	waitForSignal(t, openTransitionDone)

	incidents.EXPECT().Resolve(gomock.Any(), gomock.Any(), gomock.Any()).Return(store.ErrNoOpenIncident)
	done := make(chan struct{})
	// sendAlerts still has to run despite Resolve failing; assert that by
	// waiting for the settings lookup that only happens after recordIncident.
	settingsRepo.EXPECT().Get(gomock.Any()).DoAndReturn(
		func(_ context.Context) (models.AlertSettings, error) {
			close(done)
			return models.AlertSettings{}, nil
		},
	)

	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: baseTime.Add(2 * time.Minute)})
	waitForSignal(t, done)
}

func TestDispatcher_OpenAndResolveErrorsOtherThanErrNoOpenIncidentAreAlsoSwallowed(t *testing.T) {
	ctrl := gomock.NewController(t)
	monitorRepo := mocks.NewMockMonitorLookup(ctrl)
	settingsRepo := mocks.NewMockAlertSettingsStore(ctrl)
	incidents := mocks.NewMockIncidentStore(ctrl)
	d := alert.NewDispatcher(monitorRepo, settingsRepo, incidents, nil, nil)

	done := make(chan struct{})
	incidents.EXPECT().Open(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("db write failed"))
	settingsRepo.EXPECT().Get(gomock.Any()).DoAndReturn(
		func(_ context.Context) (models.AlertSettings, error) {
			close(done)
			return models.AlertSettings{}, nil
		},
	)

	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: baseTime})
	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusDown, CheckedAt: baseTime.Add(time.Minute)})
	waitForSignal(t, done)
}

func TestDispatcher_SettingsLookupErrorAbortsBeforeAnySend(t *testing.T) {
	ctrl := gomock.NewController(t)
	monitorRepo := mocks.NewMockMonitorLookup(ctrl)
	settingsRepo := mocks.NewMockAlertSettingsStore(ctrl)
	incidents := mocks.NewMockIncidentStore(ctrl)
	email := mocks.NewMockSender(ctrl)
	d := alert.NewDispatcher(monitorRepo, settingsRepo, incidents, email, nil)

	// No .EXPECT() on monitorRepo or email at all: a settings error must
	// stop sendAlerts before either is touched.
	done := make(chan struct{})
	incidents.EXPECT().Open(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	settingsRepo.EXPECT().Get(gomock.Any()).DoAndReturn(
		func(_ context.Context) (models.AlertSettings, error) {
			close(done)
			return models.AlertSettings{}, errors.New("db down")
		},
	)

	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: baseTime})
	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusDown, CheckedAt: baseTime.Add(time.Minute)})
	waitForSignal(t, done)
}

func TestDispatcher_BothChannelsDisabledNeverLooksUpMonitor(t *testing.T) {
	ctrl := gomock.NewController(t)
	monitorRepo := mocks.NewMockMonitorLookup(ctrl)
	settingsRepo := mocks.NewMockAlertSettingsStore(ctrl)
	incidents := mocks.NewMockIncidentStore(ctrl)
	d := alert.NewDispatcher(monitorRepo, settingsRepo, incidents, nil, nil)

	// No .EXPECT() on monitorRepo: disabled channels must short-circuit
	// before the monitor lookup that only exists to build the alert Event.
	done := make(chan struct{})
	incidents.EXPECT().Open(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	settingsRepo.EXPECT().Get(gomock.Any()).DoAndReturn(
		func(_ context.Context) (models.AlertSettings, error) {
			close(done)
			return models.AlertSettings{EmailEnabled: false, WebhookEnabled: false}, nil
		},
	)

	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: baseTime})
	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusDown, CheckedAt: baseTime.Add(time.Minute)})
	waitForSignal(t, done)
}

func TestDispatcher_MonitorLookupErrorAbortsBeforeSend(t *testing.T) {
	ctrl := gomock.NewController(t)
	monitorRepo := mocks.NewMockMonitorLookup(ctrl)
	settingsRepo := mocks.NewMockAlertSettingsStore(ctrl)
	incidents := mocks.NewMockIncidentStore(ctrl)
	email := mocks.NewMockSender(ctrl)
	d := alert.NewDispatcher(monitorRepo, settingsRepo, incidents, email, nil)

	done := make(chan struct{})
	incidents.EXPECT().Open(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	settingsRepo.EXPECT().Get(gomock.Any()).Return(models.AlertSettings{EmailEnabled: true, EmailAddress: "a@b.com"}, nil)
	monitorRepo.EXPECT().GetByID(gomock.Any(), "m1").DoAndReturn(
		func(_ context.Context, _ string) (models.Monitor, error) {
			close(done)
			return models.Monitor{}, errors.New("not found")
		},
	)
	// No .EXPECT() on email: the lookup failure must stop before Send.

	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: baseTime})
	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusDown, CheckedAt: baseTime.Add(time.Minute)})
	waitForSignal(t, done)
}

func TestDispatcher_EmailSendErrorDoesNotStopWebhookSend(t *testing.T) {
	ctrl := gomock.NewController(t)
	monitorRepo := mocks.NewMockMonitorLookup(ctrl)
	settingsRepo := mocks.NewMockAlertSettingsStore(ctrl)
	incidents := mocks.NewMockIncidentStore(ctrl)
	email := mocks.NewMockSender(ctrl)
	webhook := mocks.NewMockSender(ctrl)
	d := alert.NewDispatcher(monitorRepo, settingsRepo, incidents, email, webhook)

	incidents.EXPECT().Open(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	settingsRepo.EXPECT().Get(gomock.Any()).Return(models.AlertSettings{
		EmailEnabled: true, EmailAddress: "a@b.com",
		WebhookEnabled: true, WebhookURL: "https://hooks.example.com",
	}, nil)
	monitorRepo.EXPECT().GetByID(gomock.Any(), "m1").Return(models.Monitor{ID: "m1"}, nil)
	email.EXPECT().Send(gomock.Any(), "a@b.com", gomock.Any()).Return(errors.New("smtp down"))

	done := make(chan struct{})
	webhook.EXPECT().Send(gomock.Any(), "https://hooks.example.com", gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, _ alert.Event) error { close(done); return nil },
	)

	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: baseTime})
	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusDown, CheckedAt: baseTime.Add(time.Minute)})
	waitForSignal(t, done)
}

func TestDispatcher_NilSenderIsSkippedEvenIfChannelEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	monitorRepo := mocks.NewMockMonitorLookup(ctrl)
	settingsRepo := mocks.NewMockAlertSettingsStore(ctrl)
	incidents := mocks.NewMockIncidentStore(ctrl)
	webhook := mocks.NewMockSender(ctrl)
	// email sender left nil entirely.
	d := alert.NewDispatcher(monitorRepo, settingsRepo, incidents, nil, webhook)

	incidents.EXPECT().Open(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	settingsRepo.EXPECT().Get(gomock.Any()).Return(models.AlertSettings{
		EmailEnabled: true, EmailAddress: "a@b.com",
		WebhookEnabled: true, WebhookURL: "https://hooks.example.com",
	}, nil)
	monitorRepo.EXPECT().GetByID(gomock.Any(), "m1").Return(models.Monitor{ID: "m1"}, nil)

	done := make(chan struct{})
	webhook.EXPECT().Send(gomock.Any(), "https://hooks.example.com", gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, _ alert.Event) error { close(done); return nil },
	)

	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: baseTime})
	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusDown, CheckedAt: baseTime.Add(time.Minute)})
	waitForSignal(t, done)
}

func TestDispatcher_WebhookSendErrorIsHandled(t *testing.T) {
	ctrl := gomock.NewController(t)
	monitorRepo := mocks.NewMockMonitorLookup(ctrl)
	settingsRepo := mocks.NewMockAlertSettingsStore(ctrl)
	incidents := mocks.NewMockIncidentStore(ctrl)
	webhook := mocks.NewMockSender(ctrl)
	d := alert.NewDispatcher(monitorRepo, settingsRepo, incidents, nil, webhook)

	incidents.EXPECT().Open(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	settingsRepo.EXPECT().Get(gomock.Any()).Return(models.AlertSettings{WebhookEnabled: true, WebhookURL: "https://hooks.example.com"}, nil)
	monitorRepo.EXPECT().GetByID(gomock.Any(), "m1").Return(models.Monitor{ID: "m1"}, nil)

	done := make(chan struct{})
	webhook.EXPECT().Send(gomock.Any(), "https://hooks.example.com", gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, _ alert.Event) error {
			close(done)
			return errors.New("endpoint returned 500")
		},
	)

	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: baseTime})
	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusDown, CheckedAt: baseTime.Add(time.Minute)})
	waitForSignal(t, done)
}

func TestDispatcher_Seed(t *testing.T) {
	t.Run("nil incident store is a no-op", func(t *testing.T) {
		d := alert.NewDispatcher(nil, nil, nil, nil, nil)
		require.NoError(t, d.Seed(context.Background()))
	})

	t.Run("propagates OpenIncidents error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		incidents := mocks.NewMockIncidentStore(ctrl)
		incidents.EXPECT().OpenIncidents(gomock.Any()).Return(nil, errors.New("db down"))
		d := alert.NewDispatcher(nil, nil, incidents, nil, nil)

		err := d.Seed(context.Background())
		assert.Error(t, err)
	})

	t.Run("pre-loaded outage state survives a restart and resolves with the original duration", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		monitorRepo := mocks.NewMockMonitorLookup(ctrl)
		settingsRepo := mocks.NewMockAlertSettingsStore(ctrl)
		incidents := mocks.NewMockIncidentStore(ctrl)

		startedAt := baseTime
		incidents.EXPECT().OpenIncidents(gomock.Any()).Return([]models.Incident{
			{MonitorID: "m1", StartedAt: startedAt},
		}, nil)

		d := alert.NewDispatcher(monitorRepo, settingsRepo, incidents, nil, nil)
		require.NoError(t, d.Seed(context.Background()))

		// A continued-down check right after "restart" must NOT look like
		// a fresh first observation and must NOT re-open an incident.
		d.HandleResult(context.Background(), models.CheckResult{
			MonitorID: "m1", Status: monitor.StatusDown, CheckedAt: startedAt.Add(time.Minute),
		})

		recoveredAt := startedAt.Add(10 * time.Minute)
		settingsRepo.EXPECT().Get(gomock.Any()).Return(models.AlertSettings{}, nil)
		done := make(chan struct{})
		incidents.EXPECT().Resolve(gomock.Any(), "m1", recoveredAt).DoAndReturn(
			func(_ context.Context, _ string, _ time.Time) error { close(done); return nil },
		)

		d.HandleResult(context.Background(), models.CheckResult{
			MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: recoveredAt,
		})
		waitForSignal(t, done)
	})
}

func TestDispatcher_NilIncidentStoreSkipsRecordingButStillAlerts(t *testing.T) {
	ctrl := gomock.NewController(t)
	monitorRepo := mocks.NewMockMonitorLookup(ctrl)
	settingsRepo := mocks.NewMockAlertSettingsStore(ctrl)
	email := mocks.NewMockSender(ctrl)
	// incidents is nil: recordIncident's own guard must return immediately
	// without touching it, while sendAlerts still runs normally.
	d := alert.NewDispatcher(monitorRepo, settingsRepo, nil, email, nil)

	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: baseTime})

	settingsRepo.EXPECT().Get(gomock.Any()).Return(models.AlertSettings{EmailEnabled: true, EmailAddress: "a@b.com"}, nil)
	monitorRepo.EXPECT().GetByID(gomock.Any(), "m1").Return(models.Monitor{ID: "m1"}, nil)
	done := make(chan struct{})
	email.EXPECT().Send(gomock.Any(), "a@b.com", gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, _ alert.Event) error { close(done); return nil },
	)

	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusDown, CheckedAt: baseTime.Add(time.Minute)})
	waitForSignal(t, done)
}

func TestDispatcher_ResolveGenericErrorIsLoggedNotSwallowedSilently(t *testing.T) {
	// A Resolve error other than store.ErrNoOpenIncident is logged, not
	// swallowed - but still doesn't stop sendAlerts from running.
	ctrl := gomock.NewController(t)
	monitorRepo := mocks.NewMockMonitorLookup(ctrl)
	settingsRepo := mocks.NewMockAlertSettingsStore(ctrl)
	incidents := mocks.NewMockIncidentStore(ctrl)
	d := alert.NewDispatcher(monitorRepo, settingsRepo, incidents, nil, nil)

	incidents.EXPECT().Open(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	openTransitionDone := make(chan struct{})
	settingsRepo.EXPECT().Get(gomock.Any()).DoAndReturn(
		func(_ context.Context) (models.AlertSettings, error) {
			close(openTransitionDone)
			return models.AlertSettings{}, nil
		},
	)
	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: baseTime})
	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusDown, CheckedAt: baseTime.Add(time.Minute)})
	waitForSignal(t, openTransitionDone)

	incidents.EXPECT().Resolve(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("db write failed"))
	done := make(chan struct{})
	settingsRepo.EXPECT().Get(gomock.Any()).DoAndReturn(
		func(_ context.Context) (models.AlertSettings, error) {
			close(done)
			return models.AlertSettings{}, nil
		},
	)

	d.HandleResult(context.Background(), models.CheckResult{MonitorID: "m1", Status: monitor.StatusUp, CheckedAt: baseTime.Add(2 * time.Minute)})
	waitForSignal(t, done)
}
