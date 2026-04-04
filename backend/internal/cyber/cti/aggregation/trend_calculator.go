package aggregation

import (
	"context"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// TrendCalculator compares current vs previous period event volume to determine trend direction.
type TrendCalculator struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewTrendCalculator(db *pgxpool.Pool, logger zerolog.Logger) *TrendCalculator {
	return &TrendCalculator{db: db, logger: logger}
}

type rowQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Calculate compares the last 24 h against the preceding 24 h and returns a direction label
// ("increasing", "stable", "decreasing") plus the percentage change.
func (tc *TrendCalculator) Calculate(ctx context.Context, tenantID string, now time.Time) (string, float64) {
	return tc.calculateWithQueryer(ctx, tc.db, tenantID, now)
}

// CalculateTx performs the same comparison using the caller's tenant-scoped transaction.
func (tc *TrendCalculator) CalculateTx(ctx context.Context, tx pgx.Tx, tenantID string, now time.Time) (string, float64) {
	return tc.calculateWithQueryer(ctx, tx, tenantID, now)
}

func (tc *TrendCalculator) calculateWithQueryer(ctx context.Context, q rowQueryer, tenantID string, now time.Time) (string, float64) {
	var current, previous int64
	err := q.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE first_seen_at >= ($2::timestamptz - INTERVAL '24 hours')),
			COUNT(*) FILTER (WHERE first_seen_at >= ($2::timestamptz - INTERVAL '48 hours') AND first_seen_at < ($2::timestamptz - INTERVAL '24 hours'))
		FROM cti_threat_events
		WHERE tenant_id = $1 AND deleted_at IS NULL AND is_false_positive = false`,
		tenantID, now).Scan(&current, &previous)
	if err != nil && err != pgx.ErrNoRows {
		tc.logger.Warn().Err(err).Str("tenant_id", tenantID).Msg("trend calculation query failed")
		return "stable", 0
	}
	return classifyTrend(current, previous)
}

// CalculateForPeriod compares an arbitrary window against its predecessor of the same length.
func (tc *TrendCalculator) CalculateForPeriod(ctx context.Context, tenantID string, period time.Duration) (string, float64) {
	now := time.Now().UTC()
	var current, previous int64
	err := tc.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE first_seen_at >= $2),
			COUNT(*) FILTER (WHERE first_seen_at >= $3 AND first_seen_at < $2)
		FROM cti_threat_events
		WHERE tenant_id = $1 AND deleted_at IS NULL AND is_false_positive = false`,
		tenantID, now.Add(-period), now.Add(-2*period)).Scan(&current, &previous)
	if err != nil {
		return "stable", 0
	}
	return classifyTrend(current, previous)
}

func classifyTrend(current, previous int64) (string, float64) {
	if previous == 0 {
		if current > 0 {
			return "increasing", 100.0
		}
		return "stable", 0
	}
	pct := math.Round((float64(current-previous)/float64(previous))*10000) / 100
	switch {
	case pct > 10:
		return "increasing", pct
	case pct < -10:
		return "decreasing", pct
	default:
		return "stable", pct
	}
}
