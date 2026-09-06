package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestMonitor(t *testing.T, repo *MonitorRepo, name string) string {
	t.Helper()
	in := sampleCreateInput()
	in.Name = name
	m, err := repo.Create(context.Background(), in)
	require.NoError(t, err)
	return m.ID
}

func TestIncidentRepo_OpenAndListForMonitor(t *testing.T) {
	resetDB(t)
	monitorRepo := NewMonitorRepo(testPool)
	repo := NewIncidentRepo(testPool)
	ctx := context.Background()

	monitorID := createTestMonitor(t, monitorRepo, "M1")
	startedAt := time.Now().Add(-time.Hour).Truncate(time.Microsecond)

	require.NoError(t, repo.Open(ctx, monitorID, startedAt, "connection refused"))

	incidents, total, err := repo.List(ctx, monitorID, 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, incidents, 1)
	assert.Equal(t, monitorID, incidents[0].MonitorID)
	assert.Equal(t, "M1", incidents[0].MonitorName)
	assert.Equal(t, "connection refused", incidents[0].Cause)
	assert.Nil(t, incidents[0].ResolvedAt)
	assert.WithinDuration(t, startedAt, incidents[0].StartedAt, time.Second)
}

func TestIncidentRepo_Resolve(t *testing.T) {
	resetDB(t)
	monitorRepo := NewMonitorRepo(testPool)
	repo := NewIncidentRepo(testPool)
	ctx := context.Background()

	monitorID := createTestMonitor(t, monitorRepo, "M1")
	require.NoError(t, repo.Open(ctx, monitorID, time.Now().Add(-time.Hour), "boom"))

	resolvedAt := time.Now()
	require.NoError(t, repo.Resolve(ctx, monitorID, resolvedAt))

	incidents, _, err := repo.List(ctx, monitorID, 10, 0)
	require.NoError(t, err)
	require.Len(t, incidents, 1)
	require.NotNil(t, incidents[0].ResolvedAt)
	assert.WithinDuration(t, resolvedAt, *incidents[0].ResolvedAt, time.Second)
}

func TestIncidentRepo_Resolve_NoOpenIncident(t *testing.T) {
	resetDB(t)
	monitorRepo := NewMonitorRepo(testPool)
	repo := NewIncidentRepo(testPool)
	ctx := context.Background()

	monitorID := createTestMonitor(t, monitorRepo, "M1")

	err := repo.Resolve(ctx, monitorID, time.Now())
	assert.ErrorIs(t, err, ErrNoOpenIncident)
}

func TestIncidentRepo_OpenIncidents(t *testing.T) {
	resetDB(t)
	monitorRepo := NewMonitorRepo(testPool)
	repo := NewIncidentRepo(testPool)
	ctx := context.Background()

	m1 := createTestMonitor(t, monitorRepo, "M1")
	m2 := createTestMonitor(t, monitorRepo, "M2")

	require.NoError(t, repo.Open(ctx, m1, time.Now().Add(-time.Hour), "still down"))
	require.NoError(t, repo.Open(ctx, m2, time.Now().Add(-time.Hour), "also down"))
	require.NoError(t, repo.Resolve(ctx, m2, time.Now())) // m2 recovers, m1 doesn't

	open, err := repo.OpenIncidents(ctx)
	require.NoError(t, err)
	require.Len(t, open, 1)
	assert.Equal(t, m1, open[0].MonitorID)
}

func TestIncidentRepo_List_GlobalFeedAcrossMonitors(t *testing.T) {
	resetDB(t)
	monitorRepo := NewMonitorRepo(testPool)
	repo := NewIncidentRepo(testPool)
	ctx := context.Background()

	m1 := createTestMonitor(t, monitorRepo, "M1")
	m2 := createTestMonitor(t, monitorRepo, "M2")
	require.NoError(t, repo.Open(ctx, m1, time.Now().Add(-2*time.Hour), "a"))
	require.NoError(t, repo.Open(ctx, m2, time.Now().Add(-time.Hour), "b"))

	// Empty monitorID means the global feed, not scoped to one monitor.
	incidents, total, err := repo.List(ctx, "", 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, incidents, 2)
	// Most-recent-first.
	assert.Equal(t, m2, incidents[0].MonitorID)
	assert.Equal(t, m1, incidents[1].MonitorID)
}

func TestIncidentRepo_List_Pagination(t *testing.T) {
	resetDB(t)
	monitorRepo := NewMonitorRepo(testPool)
	repo := NewIncidentRepo(testPool)
	ctx := context.Background()

	monitorID := createTestMonitor(t, monitorRepo, "M1")
	for i := 0; i < 5; i++ {
		startedAt := time.Now().Add(-time.Duration(5-i) * time.Hour)
		require.NoError(t, repo.Open(ctx, monitorID, startedAt, "x"))
		require.NoError(t, repo.Resolve(ctx, monitorID, startedAt.Add(time.Minute)))
	}

	page1, total, err := repo.List(ctx, monitorID, 2, 0)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, page1, 2)

	page2, total, err := repo.List(ctx, monitorID, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, page2, 2)

	assert.NotEqual(t, page1[0].ID, page2[0].ID)
}
