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
)

func TestWebhookSender_DemoModeNeverCallsOut(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := NewWebhookSender(true)
	err := s.Send(context.Background(), srv.URL, Event{Type: EventDown, Monitor: testMonitor()})

	require.NoError(t, err)
	assert.False(t, called, "demo mode must never make a real request")
}

func TestWebhookSender_MissingURL(t *testing.T) {
	s := NewWebhookSender(false)
	err := s.Send(context.Background(), "", Event{Type: EventDown, Monitor: testMonitor()})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no webhook URL configured")
}

func TestWebhookSender_Success(t *testing.T) {
	var gotPayload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		_ = json.NewDecoder(r.Body).Decode(&gotPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := NewWebhookSender(false)
	err := s.Send(context.Background(), srv.URL, Event{
		Type:      EventDown,
		Monitor:   testMonitor(),
		Error:     "boom",
		CheckedAt: time.Now(),
	})

	require.NoError(t, err)
	assert.Equal(t, "down", gotPayload["event"])
	assert.Equal(t, "m1", gotPayload["monitor_id"])
	assert.Equal(t, "boom", gotPayload["error"])
	assert.NotContains(t, gotPayload, "down_duration_seconds", "a down event never carries a duration")
}

func TestWebhookSender_RecoveredIncludesDownDuration(t *testing.T) {
	var gotPayload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := NewWebhookSender(false)
	err := s.Send(context.Background(), srv.URL, Event{
		Type:         EventRecovered,
		Monitor:      testMonitor(),
		DownDuration: 45 * time.Second,
		CheckedAt:    time.Now(),
	})

	require.NoError(t, err)
	assert.Equal(t, "recovered", gotPayload["event"])
	assert.Equal(t, float64(45), gotPayload["down_duration_seconds"])
}

func TestWebhookSender_RecoveredWithZeroDurationOmitsField(t *testing.T) {
	var gotPayload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := NewWebhookSender(false)
	err := s.Send(context.Background(), srv.URL, Event{
		Type:         EventRecovered,
		Monitor:      testMonitor(),
		DownDuration: 0,
	})

	require.NoError(t, err)
	assert.NotContains(t, gotPayload, "down_duration_seconds")
}

func TestWebhookSender_NonSuccessResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	s := NewWebhookSender(false)
	err := s.Send(context.Background(), srv.URL, Event{Type: EventDown, Monitor: testMonitor()})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestWebhookSender_TransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	s := NewWebhookSender(false)
	err := s.Send(context.Background(), srv.URL, Event{Type: EventDown, Monitor: testMonitor()})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "send webhook")
}

func TestWebhookSender_InvalidURLFailsToBuildRequest(t *testing.T) {
	s := NewWebhookSender(false)
	err := s.Send(context.Background(), "://not-a-valid-url", Event{Type: EventDown, Monitor: testMonitor()})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "build webhook request")
}
