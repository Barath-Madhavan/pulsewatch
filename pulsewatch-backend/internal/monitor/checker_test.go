package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"pulsewatch-backend/internal/models"
)

// newTestChecker builds a Checker without the netguard dial guard NewChecker
// wires in. httptest.NewServer always binds to 127.0.0.1, which the real
// guard correctly refuses to dial (SSRF protection) - so a checker built
// with the real constructor can never actually reach a local test server
// at all, let alone exercise its handler.
func newTestChecker(slowThreshold time.Duration) *Checker {
	return &Checker{
		client:        &http.Client{Transport: &http.Transport{MaxIdleConnsPerHost: 20}},
		slowThreshold: slowThreshold,
	}
}

func testMonitorFor(url string) models.Monitor {
	return models.Monitor{
		ID:                 "m1",
		Method:             http.MethodGet,
		URL:                url,
		TimeoutSeconds:     5,
		ExpectedStatusCode: http.StatusOK,
	}
}

func TestChecker_Up(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestChecker(2 * time.Second)
	result := c.Check(context.Background(), testMonitorFor(srv.URL))

	assert.Equal(t, StatusUp, result.Status)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Empty(t, result.Error)
}

func TestChecker_DownOnUnexpectedStatusCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestChecker(2 * time.Second)
	result := c.Check(context.Background(), testMonitorFor(srv.URL))

	assert.Equal(t, StatusDown, result.Status)
	assert.Equal(t, http.StatusInternalServerError, result.StatusCode)
}

func TestChecker_DegradedWhenSlow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(30 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestChecker(10 * time.Millisecond) // slowThreshold well below the handler's sleep
	m := testMonitorFor(srv.URL)
	m.TimeoutSeconds = 2
	result := c.Check(context.Background(), m)

	assert.Equal(t, StatusDegraded, result.Status)
	assert.Equal(t, http.StatusOK, result.StatusCode)
}

func TestChecker_DownOnTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // closed immediately: nothing is listening there anymore.

	c := newTestChecker(2 * time.Second)
	result := c.Check(context.Background(), testMonitorFor(srv.URL))

	assert.Equal(t, StatusDown, result.Status)
	assert.NotEmpty(t, result.Error)
}

func TestChecker_DownOnInvalidRequest(t *testing.T) {
	c := NewChecker(2 * time.Second)
	m := testMonitorFor("://not-a-valid-url")
	result := c.Check(context.Background(), m)

	assert.Equal(t, StatusDown, result.Status)
	assert.NotEmpty(t, result.Error)
	assert.Zero(t, result.ResponseTimeMs, "a request-build failure happens before any timing starts")
}

func TestChecker_DefaultTimeoutAppliesWhenMonitorHasNone(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	defer close(release)

	c := newTestChecker(2 * time.Second)
	m := testMonitorFor(srv.URL)
	m.TimeoutSeconds = 0 // falls back to the checker's own 10s default

	done := make(chan models.CheckResult)
	go func() { done <- c.Check(context.Background(), m) }()

	<-started
	// The request is now genuinely in flight; release it immediately so
	// the test doesn't actually wait out a 10s timeout, it just confirms
	// the check didn't fail with "context deadline exceeded" from some
	// unexpectedly-short default.
	release <- struct{}{}
	result := <-done

	assert.Equal(t, StatusUp, result.Status)
}

// SSRF regression: the checker's transport must refuse to dial loopback/
// private/link-local addresses at actual dial time, which can't be
// bypassed by a hostname that only resolves to a private address later.
func TestChecker_RefusesNonPublicAddresses(t *testing.T) {
	c := NewChecker(2 * time.Second)
	m := testMonitorFor("http://127.0.0.1:1/") // port 1 is never listening, but that's irrelevant: the dial is refused before it would even matter.
	result := c.Check(context.Background(), m)

	assert.Equal(t, StatusDown, result.Status)
	assert.Contains(t, result.Error, "refusing to connect")
}
