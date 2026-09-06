package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pulsewatch-backend/internal/models"
)

type AlertSettingsRepo struct {
	pool *pgxpool.Pool
}

func NewAlertSettingsRepo(pool *pgxpool.Pool) *AlertSettingsRepo {
	return &AlertSettingsRepo{pool: pool}
}

func (r *AlertSettingsRepo) Get(ctx context.Context) (models.AlertSettings, error) {
	var s models.AlertSettings
	err := r.pool.QueryRow(ctx, `
		SELECT email_enabled, email_address, webhook_enabled, webhook_url, updated_at
		FROM alert_settings
		WHERE id = true`).
		Scan(&s.EmailEnabled, &s.EmailAddress, &s.WebhookEnabled, &s.WebhookURL, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		// Self-heal: the singleton seed row is missing, either a wiped
		// table or a fresh DB where the migration's seed INSERT never ran,
		// rather than leaving Settings permanently broken with no way to
		// recover short of a manual SQL insert.
		return r.createDefault(ctx)
	}
	return s, err
}

// createDefault inserts the singleton row if absent and returns whatever
// row ends up existing either way. The no-op ON CONFLICT update (rather
// than DO NOTHING) is what lets RETURNING still produce a row if a
// concurrent caller won the insert race first.
func (r *AlertSettingsRepo) createDefault(ctx context.Context) (models.AlertSettings, error) {
	var s models.AlertSettings
	err := r.pool.QueryRow(ctx, `
		INSERT INTO alert_settings (id) VALUES (true)
		ON CONFLICT (id) DO UPDATE SET id = EXCLUDED.id
		RETURNING email_enabled, email_address, webhook_enabled, webhook_url, updated_at`).
		Scan(&s.EmailEnabled, &s.EmailAddress, &s.WebhookEnabled, &s.WebhookURL, &s.UpdatedAt)
	return s, err
}

// Update is a genuine upsert, self-healing the same way Get is, in one
// atomic statement, so a PUT arriving before any GET (or after the row was
// somehow lost) still succeeds instead of silently updating zero rows.
func (r *AlertSettingsRepo) Update(ctx context.Context, in models.UpdateAlertSettingsInput) (models.AlertSettings, error) {
	var s models.AlertSettings
	err := r.pool.QueryRow(ctx, `
		INSERT INTO alert_settings (id, email_enabled, email_address, webhook_enabled, webhook_url, updated_at)
		VALUES (true, $1, $2, $3, $4, now())
		ON CONFLICT (id) DO UPDATE SET
			email_enabled = EXCLUDED.email_enabled,
			email_address = EXCLUDED.email_address,
			webhook_enabled = EXCLUDED.webhook_enabled,
			webhook_url = EXCLUDED.webhook_url,
			updated_at = EXCLUDED.updated_at
		RETURNING email_enabled, email_address, webhook_enabled, webhook_url, updated_at`,
		in.EmailEnabled, in.EmailAddress, in.WebhookEnabled, in.WebhookURL).
		Scan(&s.EmailEnabled, &s.EmailAddress, &s.WebhookEnabled, &s.WebhookURL, &s.UpdatedAt)
	return s, err
}
