package monitor

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"pulsewatch-backend/internal/models"
	"pulsewatch-backend/internal/netguard"
)

// ResultHandler receives every completed check. It runs on a worker
// goroutine, so it must be safe for concurrent use.
type ResultHandler func(models.CheckResult)

// Checker performs a single HTTP check against a monitor's target URL.
type Checker struct {
	client        *http.Client
	slowThreshold time.Duration
}

func NewChecker(slowThreshold time.Duration) *Checker {
	return &Checker{
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 20,
				// A monitor's URL is arbitrary visitor input; this refuses
				// to connect to loopback, private, link-local (including
				// the cloud metadata address), or otherwise non-public
				// destinations, checked on every dial rather than once at
				// creation time. See internal/netguard for why that
				// matters.
				DialContext: (&net.Dialer{
					Timeout: 10 * time.Second,
					Control: netguard.DialControl,
				}).DialContext,
			},
			// Redirects are followed by default; each check still bounds
			// total time via the per-monitor timeout on the request context.
			// A redirect target goes through the same DialContext, so it's
			// covered by the same guard, not just the original URL.
		},
		slowThreshold: slowThreshold,
	}
}

func (c *Checker) Check(ctx context.Context, m models.Monitor) models.CheckResult {
	timeout := time.Duration(m.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	req, err := http.NewRequestWithContext(checkCtx, m.Method, m.URL, nil)
	if err != nil {
		return models.CheckResult{
			MonitorID: m.ID,
			Status:    StatusDown,
			Error:     err.Error(),
			CheckedAt: start,
		}
	}

	resp, err := c.client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		return models.CheckResult{
			MonitorID:      m.ID,
			Status:         StatusDown,
			ResponseTimeMs: elapsed.Milliseconds(),
			Error:          err.Error(),
			CheckedAt:      start,
		}
	}
	defer resp.Body.Close()
	// Drain the body so the underlying connection can be reused by the pool.
	_, _ = io.Copy(io.Discard, resp.Body)

	status := StatusUp
	switch {
	case resp.StatusCode != m.ExpectedStatusCode:
		status = StatusDown
	case elapsed >= c.slowThreshold:
		status = StatusDegraded
	}

	return models.CheckResult{
		MonitorID:      m.ID,
		Status:         status,
		StatusCode:     resp.StatusCode,
		ResponseTimeMs: elapsed.Milliseconds(),
		CheckedAt:      start,
	}
}

// Pool runs a fixed number of worker goroutines that pull monitors off a
// bounded queue and check them concurrently, capping how many HTTP checks
// can run at once regardless of how many monitors are scheduled.
type Pool struct {
	checker  *Checker
	jobs     chan models.Monitor
	onResult ResultHandler
	workers  int
}

func NewPool(checker *Checker, workers int, onResult ResultHandler) *Pool {
	if workers <= 0 {
		workers = DefaultWorkerCount
	}
	return &Pool{
		checker:  checker,
		jobs:     make(chan models.Monitor, workers*4),
		onResult: onResult,
		workers:  workers,
	}
}

func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.workers; i++ {
		go p.worker(ctx)
	}
}

func (p *Pool) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case m, ok := <-p.jobs:
			if !ok {
				return
			}
			result := p.checker.Check(ctx, m)
			p.onResult(result)
		}
	}
}

// Submit enqueues a check. If the queue is full (the pool can't keep up),
// the check is dropped for this cycle rather than blocking the scheduler;
// the next tick will try again.
func (p *Pool) Submit(m models.Monitor) {
	select {
	case p.jobs <- m:
	default:
		log.Printf("monitor pool: queue full, dropping check for monitor %s", m.ID)
	}
}
