package monitor

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"pulsewatch-backend/internal/models"
	"pulsewatch-backend/internal/monitor/mocks"
)

func newTestPool(t *testing.T, results chan<- models.CheckResult) (*Pool, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	checker := newTestChecker(2 * time.Second)
	pool := NewPool(checker, 4, func(r models.CheckResult) { results <- r })
	return pool, srv
}

func expectSubmit(t *testing.T, results <-chan models.CheckResult, monitorID string) {
	t.Helper()
	select {
	case r := <-results:
		assert.Equal(t, monitorID, r.MonitorID)
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for a submitted check for monitor %s", monitorID)
	}
}

func assertNoFurtherSubmit(t *testing.T, results <-chan models.CheckResult) {
	t.Helper()
	select {
	case r := <-results:
		t.Fatalf("unexpected submit for monitor %s after it should have stopped", r.MonitorID)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestNewScheduler_DefaultsNonPositiveReconcileInterval(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockMonitorLister(ctrl)
	s := NewScheduler(repo, nil, 0)
	assert.Equal(t, DefaultReconcileInterval, s.reconcileInterval)
}

func TestScheduler_Reconcile_SubmitsOnlyActiveMonitors(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockMonitorLister(ctrl)
	results := make(chan models.CheckResult, 4)
	pool, srv := newTestPool(t, results)

	repo.EXPECT().List(gomock.Any()).Return([]models.Monitor{
		{ID: "active-1", URL: srv.URL, IntervalSeconds: 60, IsActive: true, UpdatedAt: time.Unix(1, 0)},
		{ID: "active-2", URL: srv.URL, IntervalSeconds: 60, IsActive: true, UpdatedAt: time.Unix(1, 0)},
		{ID: "paused", URL: srv.URL, IntervalSeconds: 60, IsActive: false, UpdatedAt: time.Unix(1, 0)},
	}, nil)

	s := NewScheduler(repo, pool, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)
	s.reconcile(ctx)

	seen := map[string]bool{}
	for i := 0; i < 2; i++ {
		select {
		case r := <-results:
			seen[r.MonitorID] = true
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for active monitors to be submitted")
		}
	}
	assert.True(t, seen["active-1"])
	assert.True(t, seen["active-2"])
	assertNoFurtherSubmit(t, results) // "paused" must never be submitted
}

func TestScheduler_Reconcile_StopsTrackerForRemovedMonitor(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockMonitorLister(ctrl)
	results := make(chan models.CheckResult, 4)
	pool, srv := newTestPool(t, results)

	m := models.Monitor{ID: "m1", URL: srv.URL, IntervalSeconds: 1, IsActive: true, UpdatedAt: time.Unix(1, 0)}
	repo.EXPECT().List(gomock.Any()).Return([]models.Monitor{m}, nil)

	s := NewScheduler(repo, pool, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)

	s.reconcile(ctx)
	expectSubmit(t, results, "m1") // the immediate first check on start

	repo.EXPECT().List(gomock.Any()).Return([]models.Monitor{}, nil) // m1 deleted
	s.reconcile(ctx)

	s.mu.Lock()
	_, stillTracked := s.tracked["m1"]
	s.mu.Unlock()
	assert.False(t, stillTracked)

	// m1's IntervalSeconds=1 would fire another submit within a second if
	// its ticker goroutine weren't actually cancelled.
	assertNoFurtherSubmit(t, results)
}

func TestScheduler_Reconcile_StopsTrackerForDeactivatedMonitor(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockMonitorLister(ctrl)
	results := make(chan models.CheckResult, 4)
	pool, srv := newTestPool(t, results)

	repo.EXPECT().List(gomock.Any()).Return([]models.Monitor{
		{ID: "m1", URL: srv.URL, IntervalSeconds: 1, IsActive: true, UpdatedAt: time.Unix(1, 0)},
	}, nil)
	s := NewScheduler(repo, pool, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)

	s.reconcile(ctx)
	expectSubmit(t, results, "m1")

	repo.EXPECT().List(gomock.Any()).Return([]models.Monitor{
		{ID: "m1", URL: srv.URL, IntervalSeconds: 1, IsActive: false, UpdatedAt: time.Unix(1, 0)},
	}, nil)
	s.reconcile(ctx)

	assertNoFurtherSubmit(t, results)
}

func TestScheduler_Reconcile_EditRestartsTicker(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockMonitorLister(ctrl)
	results := make(chan models.CheckResult, 4)
	pool, srv := newTestPool(t, results)

	repo.EXPECT().List(gomock.Any()).Return([]models.Monitor{
		{ID: "m1", URL: srv.URL, IntervalSeconds: 60, IsActive: true, UpdatedAt: time.Unix(1, 0)},
	}, nil)
	s := NewScheduler(repo, pool, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)

	s.reconcile(ctx)
	expectSubmit(t, results, "m1") // generation 1's immediate check

	// Same monitor, but UpdatedAt moved: reconcile must treat this as an
	// edit, cancel the old ticker goroutine, and start a fresh one (which
	// itself submits an immediate check).
	repo.EXPECT().List(gomock.Any()).Return([]models.Monitor{
		{ID: "m1", URL: srv.URL, IntervalSeconds: 60, IsActive: true, UpdatedAt: time.Unix(2, 0)},
	}, nil)
	s.reconcile(ctx)
	expectSubmit(t, results, "m1") // generation 2's immediate check

	s.mu.Lock()
	tm, tracked := s.tracked["m1"]
	s.mu.Unlock()
	require.True(t, tracked)
	assert.True(t, tm.updatedAt.Equal(time.Unix(2, 0)))
}

func TestScheduler_Reconcile_UnchangedMonitorIsNotRestarted(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockMonitorLister(ctrl)
	results := make(chan models.CheckResult, 4)
	pool, srv := newTestPool(t, results)

	monitorList := []models.Monitor{
		{ID: "m1", URL: srv.URL, IntervalSeconds: 60, IsActive: true, UpdatedAt: time.Unix(1, 0)},
	}
	repo.EXPECT().List(gomock.Any()).Return(monitorList, nil).Times(2)
	s := NewScheduler(repo, pool, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)

	s.reconcile(ctx)
	expectSubmit(t, results, "m1")

	// Same UpdatedAt on the second reconcile: must NOT restart (no second
	// immediate submit).
	s.reconcile(ctx)
	assertNoFurtherSubmit(t, results)
}

func TestScheduler_RunMonitor_ResubmitsOnItsOwnTicker(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockMonitorLister(ctrl)
	results := make(chan models.CheckResult, 4)
	pool, srv := newTestPool(t, results)

	repo.EXPECT().List(gomock.Any()).Return([]models.Monitor{
		{ID: "m1", URL: srv.URL, IntervalSeconds: 1, IsActive: true, UpdatedAt: time.Unix(1, 0)},
	}, nil)
	s := NewScheduler(repo, pool, time.Hour) // reconcile itself never re-fires; only m1's own 1s ticker should.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)
	s.reconcile(ctx)

	expectSubmit(t, results, "m1") // immediate check on start
	expectSubmit(t, results, "m1") // its own ticker firing ~1s later
}

func TestScheduler_Reconcile_RepoErrorIsNonFatal(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockMonitorLister(ctrl)
	repo.EXPECT().List(gomock.Any()).Return(nil, errors.New("db down"))

	s := NewScheduler(repo, nil, time.Hour)
	require.NotPanics(t, func() { s.reconcile(context.Background()) })

	s.mu.Lock()
	defer s.mu.Unlock()
	assert.Empty(t, s.tracked)
}

func TestScheduler_StopAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockMonitorLister(ctrl)
	results := make(chan models.CheckResult, 4)
	pool, srv := newTestPool(t, results)

	repo.EXPECT().List(gomock.Any()).Return([]models.Monitor{
		{ID: "m1", URL: srv.URL, IntervalSeconds: 60, IsActive: true, UpdatedAt: time.Unix(1, 0)},
		{ID: "m2", URL: srv.URL, IntervalSeconds: 60, IsActive: true, UpdatedAt: time.Unix(1, 0)},
	}, nil)
	s := NewScheduler(repo, pool, time.Hour)
	ctx := context.Background()
	pool.Start(ctx)
	s.reconcile(ctx)

	seen := map[string]bool{}
	for i := 0; i < 2; i++ {
		select {
		case r := <-results:
			seen[r.MonitorID] = true
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for initial submits")
		}
	}
	require.Len(t, seen, 2)

	s.stopAll()

	s.mu.Lock()
	defer s.mu.Unlock()
	assert.Empty(t, s.tracked)
}

func TestScheduler_Start_ReconcilesImmediatelyAndOnCancelStopsAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockMonitorLister(ctrl)
	results := make(chan models.CheckResult, 4)
	pool, srv := newTestPool(t, results)

	repo.EXPECT().List(gomock.Any()).Return([]models.Monitor{
		{ID: "m1", URL: srv.URL, IntervalSeconds: 60, IsActive: true, UpdatedAt: time.Unix(1, 0)},
	}, nil).AnyTimes()

	// A short reconcileInterval so Start's own ticker also fires during
	// this test, in addition to its guaranteed reconcile-on-entry.
	s := NewScheduler(repo, pool, 20*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	startDone := make(chan struct{})
	go func() {
		s.Start(ctx)
		close(startDone)
	}()

	expectSubmit(t, results, "m1") // the guaranteed immediate reconcile on entry

	// Give Start's own ticker (20ms) a real chance to fire at least once
	// before cancelling, so its <-ticker.C branch is actually exercised
	// too, not just the pre-loop reconcile and the ctx.Done() exit.
	time.Sleep(60 * time.Millisecond)

	cancel()
	select {
	case <-startDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	assert.Empty(t, s.tracked, "Start must stopAll before returning")
}
