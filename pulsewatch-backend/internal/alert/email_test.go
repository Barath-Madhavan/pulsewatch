package alert

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"pulsewatch-backend/internal/models"
)

func TestEmailSender_DemoModeNeverCallsOut(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := NewEmailSender("key", "from@example.com", srv.URL, true)
	err := s.Send(context.Background(), "to@example.com", Event{Type: EventDown, Monitor: testMonitor()})

	require.NoError(t, err)
	assert.False(t, called, "demo mode must never make a real request")
}

func TestEmailSender_MissingAPIKey(t *testing.T) {
	s := NewEmailSender("", "from@example.com", "http://unused.invalid", false)
	err := s.Send(context.Background(), "to@example.com", Event{Type: EventDown, Monitor: testMonitor()})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "RESEND_API_KEY is empty")
}

func TestEmailSender_MissingRecipient(t *testing.T) {
	s := NewEmailSender("key", "from@example.com", "http://unused.invalid", false)
	err := s.Send(context.Background(), "", Event{Type: EventDown, Monitor: testMonitor()})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no recipient")
}

func TestEmailSender_Success(t *testing.T) {
	var gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		gotBody, _ = payload["text"].(string)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := NewEmailSender("secret-key", "from@example.com", srv.URL, false)
	err := s.Send(context.Background(), "to@example.com", Event{
		Type:    EventDown,
		Monitor: testMonitor(),
		Error:   "connection refused",
	})

	require.NoError(t, err)
	assert.Equal(t, "Bearer secret-key", gotAuth)
	assert.Contains(t, gotBody, "not responding as expected")
	assert.Contains(t, gotBody, "connection refused", "a non-empty error must appear verbatim, not as \"none\"")
}

func TestEmailSender_DownWithNoErrorTextReadsAsNone(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		gotBody, _ = payload["text"].(string)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := NewEmailSender("key", "from@example.com", srv.URL, false)
	err := s.Send(context.Background(), "to@example.com", Event{Type: EventDown, Monitor: testMonitor()})

	require.NoError(t, err)
	assert.Contains(t, gotBody, "Error: none")
}

func TestEmailSender_RecoveredSubjectAndBody(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		gotBody, _ = payload["text"].(string)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := NewEmailSender("key", "from@example.com", srv.URL, false)
	err := s.Send(context.Background(), "to@example.com", Event{
		Type:         EventRecovered,
		Monitor:      testMonitor(),
		DownDuration: 90 * time.Second,
		CheckedAt:    time.Now(),
	})

	require.NoError(t, err)
	assert.Contains(t, gotBody, "1m30s")
}

func TestEmailSender_RecoveredWithZeroDurationReadsAsUnknown(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		gotBody, _ = payload["text"].(string)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := NewEmailSender("key", "from@example.com", srv.URL, false)
	err := s.Send(context.Background(), "to@example.com", Event{
		Type:         EventRecovered,
		Monitor:      testMonitor(),
		DownDuration: 0,
		CheckedAt:    time.Now(),
	})

	require.NoError(t, err)
	assert.Contains(t, gotBody, "Downtime: unknown")
}

func TestEmailSender_InvalidBaseURLFailsToBuildRequest(t *testing.T) {
	s := NewEmailSender("key", "from@example.com", "://not-a-valid-url", false)
	err := s.Send(context.Background(), "to@example.com", Event{Type: EventDown, Monitor: testMonitor()})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "build email request")
}

func TestEmailSender_NonSuccessResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("bad key"))
	}))
	defer srv.Close()

	s := NewEmailSender("key", "from@example.com", srv.URL, false)
	err := s.Send(context.Background(), "to@example.com", Event{Type: EventDown, Monitor: testMonitor()})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestEmailSender_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // closed immediately: connections to it fail.

	s := NewEmailSender("key", "from@example.com", srv.URL, false)
	err := s.Send(context.Background(), "to@example.com", Event{Type: EventDown, Monitor: testMonitor()})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "send email")
}

func TestEmailSender_DefaultBaseURL(t *testing.T) {
	// Passing an empty baseURL should fall back to the real Resend
	// endpoint, not an empty string. Verified indirectly: demo mode still
	// short-circuits before ever dialing it, so this just exercises the
	// defaulting branch without making a real network call.
	s := NewEmailSender("key", "from@example.com", "", true)
	err := s.Send(context.Background(), "to@example.com", Event{Type: EventDown, Monitor: testMonitor()})
	require.NoError(t, err)
	assert.Equal(t, defaultResendURL, s.baseURL)
}

func testMonitor() models.Monitor {
	return models.Monitor{ID: "m1", Name: "Test Monitor", URL: "https://example.com/health"}
}
