package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pulsewatch-backend/internal/models"
)

var ErrMonitorNotFound = errors.New("monitor not found")

type MonitorRepo struct {
	pool *pgxpool.Pool
}

func NewMonitorRepo(pool *pgxpool.Pool) *MonitorRepo {
	return &MonitorRepo{pool: pool}
}

const monitorColumns = `id, name, url, method, interval_seconds, timeout_seconds, expected_status_code, is_active, created_at, updated_at`

func scanMonitor(row pgx.Row) (models.Monitor, error) {
	var m models.Monitor
	err := row.Scan(&m.ID, &m.Name, &m.URL, &m.Method, &m.IntervalSeconds, &m.TimeoutSeconds, &m.ExpectedStatusCode, &m.IsActive, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}

func (r *MonitorRepo) Create(ctx context.Context, in models.CreateMonitorInput) (models.Monitor, error) {
	query := fmt.Sprintf(`
		INSERT INTO monitors (name, url, method, interval_seconds, timeout_seconds, expected_status_code)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING %s`, monitorColumns)

	row := r.pool.QueryRow(ctx, query, in.Name, in.URL, in.Method, in.IntervalSeconds, in.TimeoutSeconds, in.ExpectedStatusCode)
	return scanMonitor(row)
}

// Count returns how many monitors currently exist, regardless of
// active/paused state. Used to enforce the demo monitor cap.
func (r *MonitorRepo) Count(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM monitors`).Scan(&count)
	return count, err
}

func (r *MonitorRepo) List(ctx context.Context) ([]models.Monitor, error) {
	query := fmt.Sprintf(`SELECT %s FROM monitors ORDER BY created_at DESC`, monitorColumns)
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	monitors := make([]models.Monitor, 0)
	for rows.Next() {
		m, err := scanMonitor(rows)
		if err != nil {
			return nil, err
		}
		monitors = append(monitors, m)
	}
	return monitors, rows.Err()
}

func (r *MonitorRepo) GetByID(ctx context.Context, id string) (models.Monitor, error) {
	query := fmt.Sprintf(`SELECT %s FROM monitors WHERE id = $1`, monitorColumns)
	row := r.pool.QueryRow(ctx, query, id)
	m, err := scanMonitor(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Monitor{}, ErrMonitorNotFound
	}
	return m, err
}

func (r *MonitorRepo) Update(ctx context.Context, id string, in models.UpdateMonitorInput) (models.Monitor, error) {
	query := fmt.Sprintf(`
		UPDATE monitors SET
			name = COALESCE($2, name),
			url = COALESCE($3, url),
			method = COALESCE($4, method),
			interval_seconds = COALESCE($5, interval_seconds),
			timeout_seconds = COALESCE($6, timeout_seconds),
			expected_status_code = COALESCE($7, expected_status_code),
			is_active = COALESCE($8, is_active),
			updated_at = now()
		WHERE id = $1
		RETURNING %s`, monitorColumns)

	row := r.pool.QueryRow(ctx, query, id, in.Name, in.URL, in.Method, in.IntervalSeconds, in.TimeoutSeconds, in.ExpectedStatusCode, in.IsActive)
	m, err := scanMonitor(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Monitor{}, ErrMonitorNotFound
	}
	return m, err
}

func (r *MonitorRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM monitors WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrMonitorNotFound
	}
	return nil
}
