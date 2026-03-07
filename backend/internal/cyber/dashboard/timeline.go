package dashboard

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clario360/platform/internal/cyber/model"
)

type TimelineCalculator struct {
	db *pgxpool.Pool
}

func NewTimelineCalculator(db *pgxpool.Pool) *TimelineCalculator {
	return &TimelineCalculator{db: db}
}

func (c *TimelineCalculator) AlertTimeline(ctx context.Context, tenantID uuid.UUID, window time.Duration) (model.AlertTimelineData, error) {
	start := time.Now().UTC().Add(-window).Truncate(time.Hour)
	rows, err := c.db.Query(ctx, `
		SELECT date_trunc('hour', created_at) AS bucket, COUNT(*)::int
		FROM alerts
		WHERE tenant_id = $1 AND created_at >= $2 AND deleted_at IS NULL
		GROUP BY bucket
		ORDER BY bucket ASC`,
		tenantID, start,
	)
	if err != nil {
		return model.AlertTimelineData{}, fmt.Errorf("alert timeline: %w", err)
	}
	defer rows.Close()

	counts := map[time.Time]int{}
	for rows.Next() {
		var bucket time.Time
		var count int
		if err := rows.Scan(&bucket, &count); err != nil {
			return model.AlertTimelineData{}, err
		}
		counts[bucket.UTC()] = count
	}
	if err := rows.Err(); err != nil {
		return model.AlertTimelineData{}, err
	}

	points := make([]model.AlertTimelinePoint, 0, int(window.Hours())+1)
	for bucket := start; !bucket.After(time.Now().UTC().Truncate(time.Hour)); bucket = bucket.Add(time.Hour) {
		points = append(points, model.AlertTimelinePoint{
			Bucket: bucket,
			Count:  counts[bucket],
		})
	}
	return model.AlertTimelineData{
		Granularity: "hour",
		Points:      points,
	}, nil
}

func (c *TimelineCalculator) SeverityDistribution(ctx context.Context, tenantID uuid.UUID) (model.SeverityDistribution, error) {
	rows, err := c.db.Query(ctx, `
		SELECT severity::text, COUNT(*)::int
		FROM alerts
		WHERE tenant_id = $1
		  AND status IN ('new', 'acknowledged', 'investigating')
		  AND deleted_at IS NULL
		GROUP BY severity`,
		tenantID,
	)
	if err != nil {
		return model.SeverityDistribution{}, fmt.Errorf("severity distribution: %w", err)
	}
	defer rows.Close()

	out := model.SeverityDistribution{Counts: map[string]int{}}
	for rows.Next() {
		var severity string
		var count int
		if err := rows.Scan(&severity, &count); err != nil {
			return out, err
		}
		out.Counts[severity] = count
		out.Total += count
	}
	return out, rows.Err()
}
