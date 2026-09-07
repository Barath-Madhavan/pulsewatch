package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPostgresPool_Success(t *testing.T) {
	pool, err := NewPostgresPool(context.Background(), resolveTestDatabaseURL())
	require.NoError(t, err)
	defer pool.Close()

	assert.NoError(t, pool.Ping(context.Background()))
}

func TestNewPostgresPool_PingFailureClosesPoolAndErrors(t *testing.T) {
	// A syntactically valid connection string pointing at a port nothing
	// is listening on: pgxpool.New itself succeeds (it doesn't connect
	// eagerly), but the explicit Ping must fail, and NewPostgresPool must
	// return that error rather than a pool that looks fine but isn't.
	_, err := NewPostgresPool(context.Background(), "postgres://postgres:postgres@localhost:1/nope?sslmode=disable&connect_timeout=1")
	assert.Error(t, err)
}

func TestNewPostgresPool_InvalidConnectionString(t *testing.T) {
	_, err := NewPostgresPool(context.Background(), "not-a-valid-connection-string")
	assert.Error(t, err)
}