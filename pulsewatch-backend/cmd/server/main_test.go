package main

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"pulsewatch-backend/internal/models"
)

func TestResultCache_SetAndSnapshot(t *testing.T) {
	cache := newResultCache()
	cache.set(models.CheckResult{MonitorID: "m1", Status: "up"})
	cache.set(models.CheckResult{MonitorID: "m2", Status: "down"})

	snap := cache.snapshot()
	require.Len(t, snap, 2)

	seen := map[string]bool{}
	for _, msg := range snap {
		var decoded statusUpdateMessage
		require.NoError(t, json.Unmarshal(msg, &decoded))
		assert.Equal(t, "status_update", decoded.Type)
		seen[decoded.Payload.MonitorID] = true
	}
	assert.True(t, seen["m1"])
	assert.True(t, seen["m2"])
}

func TestResultCache_SetOverwritesPreviousForSameMonitor(t *testing.T) {
	cache := newResultCache()
	cache.set(models.CheckResult{MonitorID: "m1", Status: "up"})
	cache.set(models.CheckResult{MonitorID: "m1", Status: "down"})

	snap := cache.snapshot()
	require.Len(t, snap, 1)

	var decoded statusUpdateMessage
	require.NoError(t, json.Unmarshal(snap[0], &decoded))
	assert.Equal(t, "down", decoded.Payload.Status)
}

func TestResultCache_SnapshotEmpty(t *testing.T) {
	cache := newResultCache()
	assert.Empty(t, cache.snapshot())
}

func TestResultCache_ConcurrentAccess(t *testing.T) {
	cache := newResultCache()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cache.set(models.CheckResult{MonitorID: string(rune('a' + i%26)), Status: "up"})
			_ = cache.snapshot()
		}(i)
	}
	require.NotPanics(t, wg.Wait)
}

func TestNewResultHandler_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	checkResults := NewMockcheckResultCreator(ctrl)
	hub := NewMockbroadcaster(ctrl)
	dispatcher := NewMockresultDispatcher(ctrl)
	cache := newResultCache()

	result := models.CheckResult{MonitorID: "m1", Status: "up", StatusCode: 200, ResponseTimeMs: 42, CheckedAt: time.Now()}

	checkResults.EXPECT().Create(gomock.Any(), result).Return(nil)
	hub.EXPECT().Broadcast(statusUpdateMessage{Type: "status_update", Payload: result})
	dispatcher.EXPECT().HandleResult(gomock.Any(), result)

	handler := newResultHandler(context.Background(), hub, cache, checkResults, dispatcher)
	handler(result)

	snap := cache.snapshot()
	require.Len(t, snap, 1)
}

func TestNewResultHandler_PersistenceErrorDoesNotStopTheRest(t *testing.T) {
	ctrl := gomock.NewController(t)
	checkResults := NewMockcheckResultCreator(ctrl)
	hub := NewMockbroadcaster(ctrl)
	dispatcher := NewMockresultDispatcher(ctrl)
	cache := newResultCache()

	result := models.CheckResult{MonitorID: "m1", Status: "down", Error: "connection refused", CheckedAt: time.Now()}

	checkResults.EXPECT().Create(gomock.Any(), result).Return(errors.New("db down"))
	hub.EXPECT().Broadcast(gomock.Any())
	dispatcher.EXPECT().HandleResult(gomock.Any(), result)

	handler := newResultHandler(context.Background(), hub, cache, checkResults, dispatcher)
	require.NotPanics(t, func() { handler(result) })

	// Despite the persistence failure, the result is still cached and
	// dispatched: a transient DB write failure shouldn't blind the live
	// dashboard or the alert dispatcher to a real status change.
	assert.Len(t, cache.snapshot(), 1)
}

func TestNewResultHandler_LogsBothWithAndWithoutError(t *testing.T) {
	// Exercises both of newResultHandler's log.Printf call sites (the
	// checkResult itself doesn't branch on this, only the log format
	// does): nothing to assert beyond "it doesn't panic either way".
	for _, result := range []models.CheckResult{
		{MonitorID: "m1", Status: "up", StatusCode: 200},
		{MonitorID: "m1", Status: "down", Error: "timeout"},
	} {
		ctrl := gomock.NewController(t)
		checkResults := NewMockcheckResultCreator(ctrl)
		hub := NewMockbroadcaster(ctrl)
		dispatcher := NewMockresultDispatcher(ctrl)
		cache := newResultCache()

		checkResults.EXPECT().Create(gomock.Any(), result).Return(nil)
		hub.EXPECT().Broadcast(gomock.Any())
		dispatcher.EXPECT().HandleResult(gomock.Any(), result)

		handler := newResultHandler(context.Background(), hub, cache, checkResults, dispatcher)
		require.NotPanics(t, func() { handler(result) })
	}
}
