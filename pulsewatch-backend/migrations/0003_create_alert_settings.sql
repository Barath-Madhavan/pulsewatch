-- Singleton table: the boolean primary key with a CHECK constraint means
-- exactly one row can ever exist. Alert config is global (applies to every
-- monitor), not per-monitor, matching the Settings page design.
CREATE TABLE IF NOT EXISTS alert_settings (
    id              BOOLEAN PRIMARY KEY DEFAULT true CHECK (id),
    email_enabled   BOOLEAN NOT NULL DEFAULT false,
    email_address   TEXT NOT NULL DEFAULT '',
    webhook_enabled BOOLEAN NOT NULL DEFAULT false,
    webhook_url     TEXT NOT NULL DEFAULT '',
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO alert_settings (id) VALUES (true) ON CONFLICT DO NOTHING;
