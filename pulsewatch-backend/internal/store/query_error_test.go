package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"pulsewatch-backend/internal/models"
)

// cancelledContext returns a context that's already done, which pgx
// checks before ever acquiring a connection. It's a safe, deterministic
// way to force the "the query/exec call itself failed" branch in each
// repo method, without corrupting the schema or the connection pool that
// every other test in this package shares.
func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestMonitorRepo_List_QueryError(t *testing.T) {
	resetDB(t)
	repo := NewMonitorRepo(testPool)
	_, err := repo.List(cancelledContext())
	assert.Error(t, err)
}

func TestMonitorRepo_Delete_ExecError(t *testing.T) {
	resetDB(t)
	repo := NewMonitorRepo(testPool)
	err := repo.Delete(cancelledContext(), "00000000-0000-0000-0000-000000000000")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrMonitorNotFound, "a real execution failure is distinct from a clean not-found result")
}

func TestIncidentRepo_Resolve_ExecError(t *testing.T) {
	resetDB(t)
	repo := NewIncidentRepo(testPool)
	err := repo.Resolve(cancelledContext(), "00000000-0000-0000-0000-000000000000", time.Now())
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrNoOpenIncident)
}

func TestIncidentRepo_OpenIncidents_QueryError(t *testing.T) {
	resetDB(t)
	repo := NewIncidentRepo(testPool)
	_, err := repo.OpenIncidents(cancelledContext())
	assert.Error(t, err)
}

func TestIncidentRepo_List_CountQueryError(t *testing.T) {
	resetDB(t)
	repo := NewIncidentRepo(testPool)

	t.Run("global feed", func(t *testing.T) {
		_, _, err := repo.List(cancelledContext(), "", 10, 0)
		assert.Error(t, err)
	})
	t.Run("scoped to one monitor", func(t *testing.T) {
		_, _, err := repo.List(cancelledContext(), "some-id", 10, 0)
		assert.Error(t, err)
	})
}

func TestCheckResultRepo_History_QueryError(t *testing.T) {
	resetDB(t)
	repo := NewCheckResultRepo(testPool)
	_, err := repo.History(cancelledContext(), "some-id", time.Now().Add(-time.Hour), 5*time.Minute)
	assert.Error(t, err)
}

func TestCheckResultRepo_Create_ExecError(t *testing.T) {
	resetDB(t)
	repo := NewCheckResultRepo(testPool)
	err := repo.Create(cancelledContext(), models.CheckResult{MonitorID: "some-id", Status: "up", CheckedAt: time.Now()})
	assert.Error(t, err)
}

func TestAlertSettingsRepo_Get_QueryError(t *testing.T) {
	resetDB(t)
	repo := NewAlertSettingsRepo(testPool)
	_, err := repo.Get(cancelledContext())
	assert.Error(t, err)
}

func TestAlertSettingsRepo_Update_ExecError(t *testing.T) {
	resetDB(t)
	repo := NewAlertSettingsRepo(testPool)
	_, err := repo.Update(cancelledContext(), models.UpdateAlertSettingsInput{})
	assert.Error(t, err)
}
