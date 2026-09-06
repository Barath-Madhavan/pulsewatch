package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port        string
	DatabaseURL string
	Env         string

	// ResendAPIKey is left empty by default. Email alerts are simply
	// skipped (with a logged error) until it's set. ResendAPIURL is
	// overridable for pointing at a mock server in tests.
	ResendAPIKey    string
	ResendFromEmail string
	ResendAPIURL    string

	// MaxMonitors caps how many monitors can exist at once, a "demo limit
	// reached" guard for a publicly hosted instance, so a stranger can't
	// mass-create monitors and run up hosting costs or bloat the database.
	// 0 means unlimited.
	MaxMonitors int

	// DemoMode short-circuits real email/webhook sends into a log line
	// instead. Every other part of the alerting feature (the form,
	// validation, saving settings) stays fully real. Lets a publicly
	// hosted instance show the whole feature without anyone being able to
	// spam a stranger's inbox or hit an arbitrary URL through it.
	DemoMode bool
}

func Load() Config {
	return Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/pulsewatch?sslmode=disable"),
		Env:             getEnv("APP_ENV", "development"),
		ResendAPIKey:    getEnv("RESEND_API_KEY", ""),
		ResendFromEmail: getEnv("RESEND_FROM_EMAIL", "PulseWatch <onboarding@resend.dev>"),
		ResendAPIURL:    getEnv("RESEND_API_URL", ""),
		MaxMonitors:     getEnvInt("MAX_MONITORS", 30),
		DemoMode:        getEnvBool("DEMO_MODE", false),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
