package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"pulsewatch-backend/internal/models"
)

func sampleCreateInput() models.CreateMonitorInput {
	return models.CreateMonitorInput{
		Name:               "Production API",
		URL:                "https://example.com/health",
		Method:             "GET",
		IntervalSeconds:    60,
		TimeoutSeconds:     10,
		ExpectedStatusCode: 200,
	}
}

func TestMonitorRepo_CreateAndGetByID(t *testing.T) {
	resetDB(t)
	repo := NewMonitorRepo(testPool)
	ctx := context.Background()

	created, err := repo.Create(ctx, sampleCreateInput())
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "Production API", created.Name)
	assert.True(t, created.IsActive, "new monitors default to active")
	assert.False(t, created.CreatedAt.IsZero())

	got, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created, got)
}

func TestMonitorRepo_GetByID_NotFound(t *testing.T) {
	resetDB(t)
	repo := NewMonitorRepo(testPool)

	_, err := repo.GetByID(context.Background(), "00000000-0000-0000-0000-000000000000")
	assert.ErrorIs(t, err, ErrMonitorNotFound)
}

func TestMonitorRepo_List(t *testing.T) {
	resetDB(t)
	repo := NewMonitorRepo(testPool)
	ctx := context.Background()

	empty, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, empty)

	in1 := sampleCreateInput()
	in1.Name = "First"
	_, err = repo.Create(ctx, in1)
	require.NoError(t, err)

	in2 := sampleCreateInput()
	in2.Name = "Second"
	_, err = repo.Create(ctx, in2)
	require.NoError(t, err)

	list, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestMonitorRepo_Count(t *testing.T) {
	resetDB(t)
	repo := NewMonitorRepo(testPool)
	ctx := context.Background()

	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	_, err = repo.Create(ctx, sampleCreateInput())
	require.NoError(t, err)

	count, err = repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestMonitorRepo_Update(t *testing.T) {
	resetDB(t)
	repo := NewMonitorRepo(testPool)
	ctx := context.Background()

	created, err := repo.Create(ctx, sampleCreateInput())
	require.NoError(t, err)

	newName := "Renamed API"
	newActive := false
	updated, err := repo.Update(ctx, created.ID, models.UpdateMonitorInput{
		Name:     &newName,
		IsActive: &newActive,
	})
	require.NoError(t, err)
	assert.Equal(t, "Renamed API", updated.Name)
	assert.False(t, updated.IsActive)
	// Fields not mentioned in the update must be left untouched.
	assert.Equal(t, created.URL, updated.URL)
	assert.Equal(t, created.Method, updated.Method)
	assert.True(t, updated.UpdatedAt.After(created.UpdatedAt) || updated.UpdatedAt.Equal(created.UpdatedAt))
}

func TestMonitorRepo_Update_NotFound(t *testing.T) {
	resetDB(t)
	repo := NewMonitorRepo(testPool)

	newName := "doesn't matter"
	_, err := repo.Update(context.Background(), "00000000-0000-0000-0000-000000000000", models.UpdateMonitorInput{Name: &newName})
	assert.ErrorIs(t, err, ErrMonitorNotFound)
}

func TestMonitorRepo_Delete(t *testing.T) {
	resetDB(t)
	repo := NewMonitorRepo(testPool)
	ctx := context.Background()

	created, err := repo.Create(ctx, sampleCreateInput())
	require.NoError(t, err)

	require.NoError(t, repo.Delete(ctx, created.ID))

	_, err = repo.GetByID(ctx, created.ID)
	assert.ErrorIs(t, err, ErrMonitorNotFound)
}

func TestMonitorRepo_Delete_NotFound(t *testing.T) {
	resetDB(t)
	repo := NewMonitorRepo(testPool)

	err := repo.Delete(context.Background(), "00000000-0000-0000-0000-000000000000")
	assert.ErrorIs(t, err, ErrMonitorNotFound)
}

func TestMonitorRepo_Delete_CascadesToCheckResultsAndIncidents(t *testing.T) {
	resetDB(t)
	monitorRepo := NewMonitorRepo(testPool)
	checkResultRepo := NewCheckResultRepo(testPool)
	incidentRepo := NewIncidentRepo(testPool)
	ctx := context.Background()

	m, err := monitorRepo.Create(ctx, sampleCreateInput())
	require.NoError(t, err)

	require.NoError(t, checkResultRepo.Create(ctx, models.CheckResult{
		MonitorID: m.ID, Status: "up", StatusCode: 200, ResponseTimeMs: 10, CheckedAt: time.Now(),
	}))
	require.NoError(t, incidentRepo.Open(ctx, m.ID, time.Now(), "boom"))

	require.NoError(t, monitorRepo.Delete(ctx, m.ID))

	incidents, total, err := incidentRepo.List(ctx, m.ID, 10, 0)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, incidents)
}
