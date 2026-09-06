CREATE TABLE IF NOT EXISTS check_results (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id        UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    status            TEXT NOT NULL,
    status_code       INTEGER NOT NULL,
    response_time_ms  BIGINT NOT NULL,
    error             TEXT NOT NULL DEFAULT '',
    checked_at        TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_check_results_monitor_checked_at ON check_results (monitor_id, checked_at DESC);
