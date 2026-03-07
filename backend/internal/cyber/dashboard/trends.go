package dashboard

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clario360/platform/internal/cyber/model"
)

type TrendCalculator struct {
	db *pgxpool.Pool
}

func NewTrendCalculator(db *pgxpool.Pool) *TrendCalculator {
	return &TrendCalculator{db: db}
}

func (c *TrendCalculator) AlertTrend(ctx context.Context, tenantID uuid.UUID, days int) ([]model.DailyMetric, error) {
	return c.dailySeries(ctx, tenantID, days, `
		SELECT date_trunc('day', created_at) AS bucket, COUNT(*)::int
		FROM alerts
		WHERE tenant_id = $1 AND created_at >= $2 AND deleted_at IS NULL
		GROUP BY bucket
		ORDER BY bucket ASC`)
}

func (c *TrendCalculator) VulnTrend(ctx context.Context, tenantID uuid.UUID, days int) ([]model.DailyMetric, error) {
	return c.dailySeries(ctx, tenantID, days, `
		SELECT date_trunc('day', detected_at) AS bucket, COUNT(*)::int
		FROM vulnerabilities
		WHERE tenant_id = $1 AND detected_at >= $2 AND deleted_at IS NULL
		GROUP BY bucket
		ORDER BY bucket ASC`)
}

func (c *TrendCalculator) ThreatTrend(ctx context.Context, tenantID uuid.UUID, days int) ([]model.DailyMetric, error) {
	return c.dailySeries(ctx, tenantID, days, `
		SELECT date_trunc('day', created_at) AS bucket, COUNT(*)::int
		FROM threats
		WHERE tenant_id = $1 AND created_at >= $2 AND deleted_at IS NULL
		GROUP BY bucket
		ORDER BY bucket ASC`)
}

func (c *TrendCalculator) dailySeries(ctx context.Context, tenantID uuid.UUID, days int, query string) ([]model.DailyMetric, error) {
	if days <= 0 {
		days = 30
	}
	start := time.Now().UTC().AddDate(0, 0, -(days - 1)).Truncate(24 * time.Hour)
	rows, err := c.db.Query(ctx, query, tenantID, start)
	if err != nil {
		return nil, fmt.Errorf("daily trend query: %w", err)
	}
	defer rows.Close()

	counts := map[time.Time]int{}
	for rows.Next() {
		var bucket time.Time
		var count int
		if err := rows.Scan(&bucket, &count); err != nil {
			return nil, err
		}
		counts[bucket.UTC()] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	series := make([]model.DailyMetric, 0, days)
	for bucket := start; !bucket.After(time.Now().UTC().Truncate(24 * time.Hour)); bucket = bucket.Add(24 * time.Hour) {
		series = append(series, model.DailyMetric{Date: bucket, Count: counts[bucket]})
	}
	return series, nil
}
