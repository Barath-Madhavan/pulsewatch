package store

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testPool is shared across every test in this package: opening a fresh
// pool per test would be slow and isn't necessary since each test starts
// by truncating its own tables (see resetDB).
var testPool *pgxpool.Pool

const defaultTestDatabaseURL = "postgres://postgres:postgres@localhost:5433/pulsewatch_test?sslmode=disable"

// resolveTestDatabaseURL is the single source of truth for which database
// this package's integration tests hit - anything that needs a DB URL
// (TestMain's shared pool, or a test that opens its own pool directly)
// must go through this rather than referencing defaultTestDatabaseURL on
// its own, or it silently stops respecting TEST_DATABASE_URL in CI/other
// environments where the port differs from this machine's default.
func resolveTestDatabaseURL() string {
	if dbURL := os.Getenv("TEST_DATABASE_URL"); dbURL != "" {
		return dbURL
	}
	return defaultTestDatabaseURL
}

// TestMain requires a real, already-migrated Postgres database (see
// migrations/*.sql): these are integration tests for the store package,
// not unit tests, and mocking pgx would just re-implement SQL semantics
// badly instead of verifying them. Point TEST_DATABASE_URL at a
// different instance if this machine's default (matching every other
// isolated-test-DB step used throughout this project) doesn't apply.
func TestMain(m *testing.M) {
	dbURL := resolveTestDatabaseURL()

	ctx := context.Background()
	pool, err := NewPostgresPool(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "store integration tests: could not connect to %s: %v\n"+
			"Set TEST_DATABASE_URL to point at a migrated Postgres database, or ensure one is reachable at the default.\n", dbURL, err)
		os.Exit(1)
	}
	testPool = pool

	code := m.Run()
	pool.Close()
	os.Exit(code)
}

// resetDB truncates every table this package's tests touch, run at the
// start of each test so tests are independent of each other and of
// execution order. monitors cascades into check_results and incidents via
// their FKs; alert_settings is a separate singleton re-seeded to the same
// default row the migration itself seeds.
func resetDB(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	_, err := testPool.Exec(ctx, "TRUNCATE monitors CASCADE")
	if err != nil {
		t.Fatalf("resetDB: truncating monitors: %v", err)
	}
	_, err = testPool.Exec(ctx, "TRUNCATE alert_settings")
	if err != nil {
		t.Fatalf("resetDB: truncating alert_settings: %v", err)
	}
	_, err = testPool.Exec(ctx, "INSERT INTO alert_settings (id) VALUES (true)")
	if err != nil {
		t.Fatalf("resetDB: reseeding alert_settings: %v", err)
	}
}
