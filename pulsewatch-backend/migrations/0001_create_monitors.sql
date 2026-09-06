CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS monitors (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                  TEXT NOT NULL,
    url                   TEXT NOT NULL,
    method                TEXT NOT NULL DEFAULT 'GET',
    interval_seconds      INTEGER NOT NULL DEFAULT 60,
    timeout_seconds       INTEGER NOT NULL DEFAULT 10,
    expected_status_code  INTEGER NOT NULL DEFAULT 200,
    is_active             BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_monitors_is_active ON monitors (is_active);
