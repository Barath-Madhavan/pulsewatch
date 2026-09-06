package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pulsewatch-backend/internal/models"
)

// ErrNoOpenIncident is returned by Resolve when a monitor has no currently
// open incident, a benign case (e.g. its very first observed check was
// already "up", so no incident was ever opened for it to recover from).
var ErrNoOpenIncident = errors.New("no open incident for monitor")

type IncidentRepo struct {
	pool *pgxpool.Pool
}

func NewIncidentRepo(pool *pgxpool.Pool) *IncidentRepo {
	return &IncidentRepo{pool: pool}
}

func (r *IncidentRepo) Open(ctx context.Context, monitorID string, startedAt time.Time, cause string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO incidents (monitor_id, started_at, cause)
		VALUES ($1, $2, $3)`,
		monitorID, startedAt, cause)
	return err
}

// Resolve closes the monitor's currently open incident, if any. By
// construction (Open/Resolve only ever run from the same transition
// detector) at most one incident is open per monitor at a time.
func (r *IncidentRepo) Resolve(ctx context.Context, monitorID string, resolvedAt time.Time) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE incidents SET resolved_at = $2
		WHERE monitor_id = $1 AND resolved_at IS NULL`,
		monitorID, resolvedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNoOpenIncident
	}
	return nil
}

// OpenIncidents returns every currently-open incident, across all
// monitors. Used to seed the alert dispatcher's in-memory transition state
// on startup. Without it, a server restart while a monitor is down makes
// the next check look like a first-ever observation, so the eventual
// recovery is never noticed and the incident is left open forever.
func (r *IncidentRepo) OpenIncidents(ctx context.Context) ([]models.Incident, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+incidentColumns+`
		FROM incidents
		JOIN monitors ON monitors.id = incidents.monitor_id
		WHERE incidents.resolved_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	incidents := make([]models.Incident, 0)
	for rows.Next() {
		var inc models.Incident
		if err := rows.Scan(&inc.ID, &inc.MonitorID, &inc.MonitorName, &inc.StartedAt, &inc.ResolvedAt, &inc.Cause); err != nil {
			return nil, err
		}
		incidents = append(incidents, inc)
	}
	return incidents, rows.Err()
}

const incidentColumns = `incidents.id, incidents.monitor_id, monitors.name, incidents.started_at, incidents.resolved_at, incidents.cause`

// List returns one page of incidents most-recent-first, joined with the
// owning monitor's name for display, plus the total count matching the
// filter (for a "1-10 of 47" style display and computing page count).
// Pass an empty monitorID for the global feed across every monitor.
func (r *IncidentRepo) List(ctx context.Context, monitorID string, limit, offset int) ([]models.Incident, int, error) {
	var total int
	var countErr error
	if monitorID != "" {
		countErr = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM incidents WHERE monitor_id = $1`, monitorID).Scan(&total)
	} else {
		countErr = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM incidents`).Scan(&total)
	}
	if countErr != nil {
		return nil, 0, countErr
	}

	var rows pgx.Rows
	var err error

	if monitorID != "" {
		rows, err = r.pool.Query(ctx, `
			SELECT `+incidentColumns+`
			FROM incidents
			JOIN monitors ON monitors.id = incidents.monitor_id
			WHERE incidents.monitor_id = $1
			ORDER BY incidents.started_at DESC
			LIMIT $2 OFFSET $3`,
			monitorID, limit, offset)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT `+incidentColumns+`
			FROM incidents
			JOIN monitors ON monitors.id = incidents.monitor_id
			ORDER BY incidents.started_at DESC
			LIMIT $1 OFFSET $2`,
			limit, offset)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	incidents := make([]models.Incident, 0)
	for rows.Next() {
		var inc models.Incident
		if err := rows.Scan(&inc.ID, &inc.MonitorID, &inc.MonitorName, &inc.StartedAt, &inc.ResolvedAt, &inc.Cause); err != nil {
			return nil, 0, err
		}
		incidents = append(incidents, inc)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return incidents, total, nil
}
