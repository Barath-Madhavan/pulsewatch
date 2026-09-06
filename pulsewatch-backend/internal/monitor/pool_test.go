package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"pulsewatch-backend/internal/models"
)

func TestPool_ProcessesSubmittedJobs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	checker := newTestChecker(2 * time.Second)
	results := make(chan models.CheckResult, 3)
	pool := NewPool(checker, 2, func(r models.CheckResult) { results <- r })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)

	pool.Submit(testMonitorFor(srv.URL))
	m2 := testMonitorFor(srv.URL)
	m2.ID = "m2"
	pool.Submit(m2)

	seen := map[string]bool{}
	for i := 0; i < 2; i++ {
		select {
		case r := <-results:
			seen[r.MonitorID] = true
			assert.Equal(t, StatusUp, r.Status)
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for pool to process submitted jobs")
		}
	}
	assert.True(t, seen["m1"])
	assert.True(t, seen["m2"])
}

func TestPool_WorkersRunConcurrently(t *testing.T) {
	const workers = 3

	var arrived int32
	allArrived := make(chan struct{})
	var closeOnce sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&arrived, 1) == workers {
			closeOnce.Do(func() { close(allArrived) })
		}
		// If the pool only ran one job at a time, this handler would block
		// forever waiting for siblings that can never arrive (they'd be
		// stuck behind this same request in the queue), and the test would
		// time out. Passing proves `workers` requests were genuinely
		// in flight at once.
		<-allArrived
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	checker := newTestChecker(2 * time.Second)
	results := make(chan models.CheckResult, workers)
	pool := NewPool(checker, workers, func(r models.CheckResult) { results <- r })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)

	for i := 0; i < workers; i++ {
		m := testMonitorFor(srv.URL)
		m.ID = string(rune('a' + i))
		pool.Submit(m)
	}

	for i := 0; i < workers; i++ {
		select {
		case <-results:
		case <-time.After(3 * time.Second):
			t.Fatal("timed out: workers did not run concurrently")
		}
	}
}

func TestPool_DefaultWorkerCountWhenNonPositive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	checker := newTestChecker(2 * time.Second)
	results := make(chan models.CheckResult, 1)
	pool := NewPool(checker, 0, func(r models.CheckResult) { results <- r })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool.Start(ctx)
	pool.Submit(testMonitorFor(srv.URL))

	select {
	case r := <-results:
		assert.Equal(t, StatusUp, r.Status)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a job submitted to a default-sized pool")
	}
}

func TestPool_SubmitDropsWithoutBlockingWhenQueueIsFull(t *testing.T) {
	checker := newTestChecker(2 * time.Second)
	pool := NewPool(checker, 1, func(models.CheckResult) {})
	// Deliberately never call pool.Start: nothing drains the queue, so its
	// capacity (workers*4 = 4) fills up exactly, and the next Submit must
	// hit the queue-full branch and return immediately instead of blocking.
	for i := 0; i < 4; i++ {
		pool.Submit(models.Monitor{ID: "queued"})
	}

	done := make(chan struct{})
	go func() {
		pool.Submit(models.Monitor{ID: "dropped"})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Submit blocked instead of dropping the job when the queue was full")
	}
}

func TestPool_WorkerReturnsWhenJobsChannelCloses(t *testing.T) {
	// Nothing in production ever closes the jobs channel today (Pool has
	// no Stop/Close method), but the worker loop defends against it
	// anyway; exercise that branch directly since jobs is reachable from
	// this white-box test.
	require.NotPanics(t, func() {
		checker := newTestChecker(time.Second)
		pool := NewPool(checker, 1, func(models.CheckResult) {})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		pool.Start(ctx)
		close(pool.jobs)
		time.Sleep(20 * time.Millisecond)
	})
}

func TestPool_StopsWhenContextCancelled(t *testing.T) {
	require.NotPanics(t, func() {
		checker := newTestChecker(time.Second)
		pool := NewPool(checker, 1, func(models.CheckResult) {})
		ctx, cancel := context.WithCancel(context.Background())
		pool.Start(ctx)
		cancel()
		// Give the worker goroutine a moment to observe ctx.Done() and
		// return; nothing to assert directly (no exported state), this
		// just confirms cancellation doesn't panic or deadlock.
		time.Sleep(20 * time.Millisecond)
	})
}
