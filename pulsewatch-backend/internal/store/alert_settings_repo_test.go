package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"pulsewatch-backend/internal/models"
)

func TestAlertSettingsRepo_Get_DefaultsFromMigrationSeed(t *testing.T) {
	resetDB(t)
	repo := NewAlertSettingsRepo(testPool)

	settings, err := repo.Get(context.Background())
	require.NoError(t, err)
	assert.False(t, settings.EmailEnabled)
	assert.Equal(t, "", settings.EmailAddress)
	assert.False(t, settings.WebhookEnabled)
	assert.Equal(t, "", settings.WebhookURL)
}

func TestAlertSettingsRepo_Get_SelfHealsWhenRowMissing(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	// Simulate a wiped table (or a fresh DB where the migration's seed
	// insert somehow never ran): Get must recover, not error forever.
	_, err := testPool.Exec(ctx, "DELETE FROM alert_settings")
	require.NoError(t, err)

	repo := NewAlertSettingsRepo(testPool)
	settings, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.False(t, settings.EmailEnabled)

	// And the row genuinely exists now, not just a returned zero value.
	var count int
	require.NoError(t, testPool.QueryRow(ctx, "SELECT COUNT(*) FROM alert_settings").Scan(&count))
	assert.Equal(t, 1, count)
}

func TestAlertSettingsRepo_Update(t *testing.T) {
	resetDB(t)
	repo := NewAlertSettingsRepo(testPool)
	ctx := context.Background()

	updated, err := repo.Update(ctx, models.UpdateAlertSettingsInput{
		EmailEnabled: true, EmailAddress: "me@example.com",
		WebhookEnabled: true, WebhookURL: "https://hooks.example.com",
	})
	require.NoError(t, err)
	assert.True(t, updated.EmailEnabled)
	assert.Equal(t, "me@example.com", updated.EmailAddress)
	assert.True(t, updated.WebhookEnabled)
	assert.Equal(t, "https://hooks.example.com", updated.WebhookURL)

	got, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, updated, got)
}

func TestAlertSettingsRepo_Update_SelfHealsWhenRowMissing(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	_, err := testPool.Exec(ctx, "DELETE FROM alert_settings")
	require.NoError(t, err)

	repo := NewAlertSettingsRepo(testPool)
	updated, err := repo.Update(ctx, models.UpdateAlertSettingsInput{EmailEnabled: true, EmailAddress: "a@b.com"})
	require.NoError(t, err)
	assert.True(t, updated.EmailEnabled)
}

func TestAlertSettingsRepo_Update_OverwritesPreviousValues(t *testing.T) {
	resetDB(t)
	repo := NewAlertSettingsRepo(testPool)
	ctx := context.Background()

	_, err := repo.Update(ctx, models.UpdateAlertSettingsInput{
		EmailEnabled: true, EmailAddress: "old@example.com",
	})
	require.NoError(t, err)

	updated, err := repo.Update(ctx, models.UpdateAlertSettingsInput{
		EmailEnabled: false, EmailAddress: "",
	})
	require.NoError(t, err)
	assert.False(t, updated.EmailEnabled)
	assert.Equal(t, "", updated.EmailAddress)
}
