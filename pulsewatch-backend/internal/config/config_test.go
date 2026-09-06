package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Defaults(t *testing.T) {
	cfg := Load()

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "postgres://postgres:postgres@localhost:5432/pulsewatch?sslmode=disable", cfg.DatabaseURL)
	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, "", cfg.ResendAPIKey)
	assert.Equal(t, "PulseWatch <onboarding@resend.dev>", cfg.ResendFromEmail)
	assert.Equal(t, "", cfg.ResendAPIURL)
	assert.Equal(t, 30, cfg.MaxMonitors)
	assert.False(t, cfg.DemoMode)
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DATABASE_URL", "postgres://u:p@host:5432/db")
	t.Setenv("APP_ENV", "production")
	t.Setenv("RESEND_API_KEY", "key-123")
	t.Setenv("RESEND_FROM_EMAIL", "Alerts <alerts@example.com>")
	t.Setenv("RESEND_API_URL", "https://mock.local/emails")
	t.Setenv("MAX_MONITORS", "5")
	t.Setenv("DEMO_MODE", "true")

	cfg := Load()

	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "postgres://u:p@host:5432/db", cfg.DatabaseURL)
	assert.Equal(t, "production", cfg.Env)
	assert.Equal(t, "key-123", cfg.ResendAPIKey)
	assert.Equal(t, "Alerts <alerts@example.com>", cfg.ResendFromEmail)
	assert.Equal(t, "https://mock.local/emails", cfg.ResendAPIURL)
	assert.Equal(t, 5, cfg.MaxMonitors)
	assert.True(t, cfg.DemoMode)
}

func TestLoad_EmptyEnvValueFallsBackToDefault(t *testing.T) {
	// getEnv/getEnvInt/getEnvBool all treat a present-but-empty variable the
	// same as unset, rather than as a literal empty override.
	t.Setenv("PORT", "")
	t.Setenv("MAX_MONITORS", "")
	t.Setenv("DEMO_MODE", "")

	cfg := Load()

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, 30, cfg.MaxMonitors)
	assert.False(t, cfg.DemoMode)
}

func TestLoad_InvalidIntFallsBackToDefault(t *testing.T) {
	t.Setenv("MAX_MONITORS", "not-a-number")

	cfg := Load()

	assert.Equal(t, 30, cfg.MaxMonitors)
}

func TestLoad_InvalidBoolFallsBackToDefault(t *testing.T) {
	t.Setenv("DEMO_MODE", "not-a-bool")

	cfg := Load()

	assert.False(t, cfg.DemoMode)
}

func TestLoad_ZeroMaxMonitorsMeansUnlimited(t *testing.T) {
	// Not a distinct code path in Load itself, but documents the contract
	// getEnvInt hands back: "0" is a valid, parseable override, not treated
	// as falling back to the default.
	t.Setenv("MAX_MONITORS", "0")

	cfg := Load()

	assert.Equal(t, 0, cfg.MaxMonitors)
}
