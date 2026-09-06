package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"pulsewatch-backend/internal/models"
)

func insertCheck(t *testing.T, repo *CheckResultRepo, monitorID, status string, responseMs int64, checkedAt time.Time) {
	t.Helper()
	require.NoError(t, repo.Create(context.Background(), models.CheckResult{
		MonitorID:      monitorID,
		Status:         status,
		StatusCode:     200,
		ResponseTimeMs: responseMs,
		CheckedAt:      checkedAt,
	}))
}

func TestCheckResultRepo_Create(t *testing.T) {
	resetDB(t)
	monitorRepo := NewMonitorRepo(testPool)
	repo := NewCheckResultRepo(testPool)
	monitorID := createTestMonitor(t, monitorRepo, "M1")

	err := repo.Create(context.Background(), models.CheckResult{
		MonitorID: monitorID, Status: "up", StatusCode: 200, ResponseTimeMs: 42,
		Error: "", CheckedAt: time.Now(),
	})
	require.NoError(t, err)
}

func TestCheckResultRepo_UptimePercentage_NoDataDefaultsTo100(t *testing.T) {
	resetDB(t)
	monitorRepo := NewMonitorRepo(testPool)
	repo := NewCheckResultRepo(testPool)
	monitorID := createTestMonitor(t, monitorRepo, "M1")

	pct, err := repo.UptimePercentage(context.Background(), monitorID, time.Now().Add(-24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 100.0, pct)
}

func TestCheckResultRepo_UptimePercentage_DegradedCountsAsUp(t *testing.T) {
	resetDB(t)
	monitorRepo := NewMonitorRepo(testPool)
	repo := NewCheckResultRepo(testPool)
	ctx := context.Background()
	monitorID := createTestMonitor(t, monitorRepo, "M1")
	since := time.Now().Add(-time.Hour)

	insertCheck(t, repo, monitorID, "up", 50, time.Now())
	insertCheck(t, repo, monitorID, "degraded", 3000, time.Now())
	insertCheck(t, repo, monitorID, "down", 0, time.Now())
	insertCheck(t, repo, monitorID, "down", 0, time.Now())

	pct, err := repo.UptimePercentage(ctx, monitorID, since)
	require.NoError(t, err)
	// 2 of 4 checks are "down"; only down counts against uptime.
	assert.InDelta(t, 50.0, pct, 0.01)
}

func TestCheckResultRepo_UptimePercentage_OnlyCountsWithinRange(t *testing.T) {
	resetDB(t)
	monitorRepo := NewMonitorRepo(testPool)
	repo := NewCheckResultRepo(testPool)
	ctx := context.Background()
	monitorID := createTestMonitor(t, monitorRepo, "M1")

	insertCheck(t, repo, monitorID, "down", 0, time.Now().Add(-48*time.Hour)) // outside the 24h window
	insertCheck(t, repo, monitorID, "up", 50, time.Now())

	pct, err := repo.UptimePercentage(ctx, monitorID, time.Now().Add(-24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 100.0, pct, "the old down check must not count against a 24h window")
}

func TestCheckResultRepo_History_BucketsAndAverages(t *testing.T) {
	resetDB(t)
	monitorRepo := NewMonitorRepo(testPool)
	repo := NewCheckResultRepo(testPool)
	ctx := context.Background()
	monitorID := createTestMonitor(t, monitorRepo, "M1")

	now := time.Now().Truncate(time.Minute)
	insertCheck(t, repo, monitorID, "up", 100, now)
	insertCheck(t, repo, monitorID, "up", 200, now.Add(time.Second))

	buckets, err := repo.History(ctx, monitorID, now.Add(-time.Hour), 5*time.Minute)
	require.NoError(t, err)
	require.Len(t, buckets, 1)
	assert.Equal(t, 150.0, buckets[0].AvgResponseTimeMs)
	assert.Equal(t, "up", buckets[0].WorstStatus)
	assert.Equal(t, 2, buckets[0].TotalChecks)
}

func TestCheckResultRepo_History_AnomalyRatioThreshold(t *testing.T) {
	resetDB(t)
	monitorRepo := NewMonitorRepo(testPool)
	repo := NewCheckResultRepo(testPool)
	ctx := context.Background()
	now := time.Now().Truncate(time.Minute)

	t.Run("a single failure among many stays up (below the 10% ratio)", func(t *testing.T) {
		resetDB(t)
		monitorID := createTestMonitor(t, monitorRepo, "M1")
		insertCheck(t, repo, monitorID, "down", 0, now)
		for i := 0; i < 20; i++ {
			insertCheck(t, repo, monitorID, "up", 50, now.Add(time.Duration(i)*time.Second))
		}

		buckets, err := repo.History(ctx, monitorID, now.Add(-time.Hour), 5*time.Minute)
		require.NoError(t, err)
		require.Len(t, buckets, 1)
		assert.Equal(t, "up", buckets[0].WorstStatus)
	})

	t.Run("enough failures to cross the ratio marks the bucket down", func(t *testing.T) {
		resetDB(t)
		monitorID := createTestMonitor(t, monitorRepo, "M1")
		for i := 0; i < 2; i++ {
			insertCheck(t, repo, monitorID, "down", 0, now.Add(time.Duration(i)*time.Second))
		}
		for i := 0; i < 8; i++ {
			insertCheck(t, repo, monitorID, "up", 50, now.Add(time.Duration(i+2)*time.Second))
		}

		buckets, err := repo.History(ctx, monitorID, now.Add(-time.Hour), 5*time.Minute)
		require.NoError(t, err)
		require.Len(t, buckets, 1)
		assert.Equal(t, "down", buckets[0].WorstStatus, "2 of 10 checks down is exactly the 10%% threshold")
	})

	t.Run("degraded-only failures mark the bucket degraded, not down", func(t *testing.T) {
		resetDB(t)
		monitorID := createTestMonitor(t, monitorRepo, "M1")
		for i := 0; i < 2; i++ {
			insertCheck(t, repo, monitorID, "degraded", 3000, now.Add(time.Duration(i)*time.Second))
		}
		for i := 0; i < 8; i++ {
			insertCheck(t, repo, monitorID, "up", 50, now.Add(time.Duration(i+2)*time.Second))
		}

		buckets, err := repo.History(ctx, monitorID, now.Add(-time.Hour), 5*time.Minute)
		require.NoError(t, err)
		require.Len(t, buckets, 1)
		assert.Equal(t, "degraded", buckets[0].WorstStatus)
	})
}

func TestCheckResultRepo_History_EmptyWhenNoData(t *testing.T) {
	resetDB(t)
	monitorRepo := NewMonitorRepo(testPool)
	repo := NewCheckResultRepo(testPool)
	monitorID := createTestMonitor(t, monitorRepo, "M1")

	buckets, err := repo.History(context.Background(), monitorID, time.Now().Add(-time.Hour), 5*time.Minute)
	require.NoError(t, err)
	assert.Empty(t, buckets)
}
