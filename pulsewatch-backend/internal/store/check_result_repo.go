package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"pulsewatch-backend/internal/models"
)

type CheckResultRepo struct {
	pool *pgxpool.Pool
}

func NewCheckResultRepo(pool *pgxpool.Pool) *CheckResultRepo {
	return &CheckResultRepo{pool: pool}
}

func (r *CheckResultRepo) Create(ctx context.Context, result models.CheckResult) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO check_results (monitor_id, status, status_code, response_time_ms, error, checked_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		result.MonitorID, result.Status, result.StatusCode, result.ResponseTimeMs, result.Error, result.CheckedAt)
	return err
}

// anomalyRatioThreshold gates how much of a bucket's checks must have been
// down/degraded before the bucket itself is marked anomalous. Flagging on
// "any single check failed" makes coarse buckets (e.g. 30d at 6h width,
// ~100+ checks each) paint almost every point red from one-off blips;
// a ratio keeps the marker meaning "this window had a real problem," not
// "one request out of hundreds hiccupped."
const anomalyRatioThreshold = 0.10

// HistoryBucket is one time-bucketed point in a monitor's response-time
// history. WorstStatus is "down" when at least anomalyRatioThreshold of the
// bucket's checks were down, "degraded" when that share were down or
// degraded but not enough were down alone, else "up".
type HistoryBucket struct {
	BucketStart       time.Time `json:"bucket_start"`
	AvgResponseTimeMs float64   `json:"avg_response_time_ms"`
	WorstStatus       string    `json:"worst_status"`
	TotalChecks       int       `json:"total_checks"`
}

func (r *CheckResultRepo) History(ctx context.Context, monitorID string, since time.Time, bucketWidth time.Duration) ([]HistoryBucket, error) {
	// bucketWidth is passed as seconds and turned into an interval via
	// make_interval. Go's Duration.String() (e.g. "5m0s") isn't valid
	// Postgres interval syntax, so it can't be cast directly.
	rows, err := r.pool.Query(ctx, `
		SELECT
			date_bin(make_interval(secs => $1), checked_at, TIMESTAMPTZ 'epoch') AS bucket,
			AVG(response_time_ms)::float8 AS avg_response_time_ms,
			CASE
				WHEN COUNT(*) FILTER (WHERE status = 'down')::float8 / COUNT(*) >= $4 THEN 'down'
				WHEN COUNT(*) FILTER (WHERE status IN ('down', 'degraded'))::float8 / COUNT(*) >= $4 THEN 'degraded'
				ELSE 'up'
			END AS worst_status,
			COUNT(*) AS total_checks
		FROM check_results
		WHERE monitor_id = $2 AND checked_at >= $3
		GROUP BY bucket
		ORDER BY bucket ASC`,
		bucketWidth.Seconds(), monitorID, since, anomalyRatioThreshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	buckets := make([]HistoryBucket, 0)
	for rows.Next() {
		var b HistoryBucket
		if err := rows.Scan(&b.BucketStart, &b.AvgResponseTimeMs, &b.WorstStatus, &b.TotalChecks); err != nil {
			return nil, err
		}
		buckets = append(buckets, b)
	}
	return buckets, rows.Err()
}

// UptimePercentage treats "degraded" (slow but correct) as up; only
// "down" counts against uptime, matching standard uptime-monitor
// convention. Returns 100 when there are no checks in range yet.
func (r *CheckResultRepo) UptimePercentage(ctx context.Context, monitorID string, since time.Time) (float64, error) {
	var pct float64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(
			COUNT(*) FILTER (WHERE status != 'down') * 100.0 / NULLIF(COUNT(*), 0),
			100
		)
		FROM check_results
		WHERE monitor_id = $1 AND checked_at >= $2`,
		monitorID, since).Scan(&pct)
	return pct, err
}
