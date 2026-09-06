package monitor

import (
	"context"
	"log"
	"sync"
	"time"

	"pulsewatch-backend/internal/models"
)

// Scheduler keeps one ticker goroutine per active monitor, submitting a
// check to the worker pool on each tick. It periodically reconciles its
// running tickers against the monitor table so that monitors created,
// deleted, deactivated, or edited via the API take effect without a
// server restart.
type Scheduler struct {
	repo              MonitorLister
	pool              *Pool
	reconcileInterval time.Duration

	mu      sync.Mutex
	tracked map[string]trackedMonitor
}

type trackedMonitor struct {
	cancel    context.CancelFunc
	updatedAt time.Time
}

func NewScheduler(repo MonitorLister, pool *Pool, reconcileInterval time.Duration) *Scheduler {
	if reconcileInterval <= 0 {
		reconcileInterval = DefaultReconcileInterval
	}
	return &Scheduler{
		repo:              repo,
		pool:              pool,
		reconcileInterval: reconcileInterval,
		tracked:           make(map[string]trackedMonitor),
	}
}

// Start blocks until ctx is cancelled, reconciling on a timer.
func (s *Scheduler) Start(ctx context.Context) {
	s.reconcile(ctx)

	ticker := time.NewTicker(s.reconcileInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.stopAll()
			return
		case <-ticker.C:
			s.reconcile(ctx)
		}
	}
}

func (s *Scheduler) reconcile(ctx context.Context) {
	monitors, err := s.repo.List(ctx)
	if err != nil {
		log.Printf("scheduler: failed to list monitors: %v", err)
		return
	}

	active := make(map[string]models.Monitor, len(monitors))
	for _, m := range monitors {
		if m.IsActive {
			active[m.ID] = m
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop tickers for monitors that were deleted, deactivated, or edited
	// (an edit restarts the ticker so the new interval/timeout/URL applies).
	for id, tm := range s.tracked {
		m, stillActive := active[id]
		if !stillActive || !m.UpdatedAt.Equal(tm.updatedAt) {
			tm.cancel()
			delete(s.tracked, id)
		}
	}

	// Start tickers for monitors that are active but not yet tracked.
	for id, m := range active {
		if _, exists := s.tracked[id]; exists {
			continue
		}
		monitorCtx, cancel := context.WithCancel(ctx)
		s.tracked[id] = trackedMonitor{cancel: cancel, updatedAt: m.UpdatedAt}
		go s.runMonitor(monitorCtx, m)
	}
}

func (s *Scheduler) runMonitor(ctx context.Context, m models.Monitor) {
	interval := time.Duration(m.IntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 60 * time.Second
	}

	// Check once immediately so a newly created monitor doesn't wait a full
	// interval before its first result.
	s.pool.Submit(m)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// reconcile cancels this goroutine's context and starts a fresh
			// one with updated config in the same breath (on edit/delete/
			// deactivate). If this ticker was already about to fire at that
			// exact moment, select can pick this case over the now-closed
			// ctx.Done(). Go doesn't prioritize between simultaneously
			// ready cases, which would submit one stale check under the
			// old config after cancellation. ctx.Err() is guaranteed
			// non-nil the instant ctx.Done() is observably closed, so this
			// check reliably catches that race.
			if ctx.Err() != nil {
				return
			}
			s.pool.Submit(m)
		}
	}
}

func (s *Scheduler) stopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, tm := range s.tracked {
		tm.cancel()
		delete(s.tracked, id)
	}
}
