CREATE TABLE IF NOT EXISTS incidents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id  UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    started_at  TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ,
    cause       TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_incidents_monitor_started_at ON incidents (monitor_id, started_at DESC);
-- Speeds up "does this monitor have an open incident" / resolve-on-recovery lookups.
CREATE INDEX IF NOT EXISTS idx_incidents_open ON incidents (monitor_id) WHERE resolved_at IS NULL;
