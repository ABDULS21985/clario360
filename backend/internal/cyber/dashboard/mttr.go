package dashboard

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clario360/platform/internal/cyber/metrics"
	"github.com/clario360/platform/internal/cyber/model"
)

type MTTRCalculator struct {
	db      *pgxpool.Pool
	metrics *metrics.Metrics
}

func NewMTTRCalculator(db *pgxpool.Pool, m *metrics.Metrics) *MTTRCalculator {
	return &MTTRCalculator{db: db, metrics: m}
}

func (c *MTTRCalculator) Calculate(ctx context.Context, tenantID uuid.UUID) (*model.MTTRReport, error) {
	report := &model.MTTRReport{
		BySeverity: map[string]model.MTTREntry{},
		Period:     "Last 30 days",
	}
	rows, err := c.db.Query(ctx, `
		SELECT severity::text,
			AVG(EXTRACT(EPOCH FROM (COALESCE(acknowledged_at, resolved_at, now()) - created_at)) / 3600)::float8,
			PERCENTILE_CONT(0.5) WITHIN GROUP (
				ORDER BY EXTRACT(EPOCH FROM (COALESCE(acknowledged_at, resolved_at, now()) - created_at)) / 3600
			)::float8,
			PERCENTILE_CONT(0.95) WITHIN GROUP (
				ORDER BY EXTRACT(EPOCH FROM (COALESCE(acknowledged_at, resolved_at, now()) - created_at)) / 3600
			)::float8,
			AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 3600)
				FILTER (WHERE resolved_at IS NOT NULL)::float8,
			COUNT(*)::int,
			100.0 * AVG(
				CASE
					WHEN severity = 'critical' AND EXTRACT(EPOCH FROM (COALESCE(acknowledged_at, resolved_at, now()) - created_at)) / 3600 <= 4 THEN 1
					WHEN severity = 'high' AND EXTRACT(EPOCH FROM (COALESCE(acknowledged_at, resolved_at, now()) - created_at)) / 3600 <= 8 THEN 1
					WHEN severity = 'medium' AND EXTRACT(EPOCH FROM (COALESCE(acknowledged_at, resolved_at, now()) - created_at)) / 3600 <= 24 THEN 1
					WHEN severity = 'low' AND EXTRACT(EPOCH FROM (COALESCE(acknowledged_at, resolved_at, now()) - created_at)) / 3600 <= 72 THEN 1
					ELSE 0
				END
			)::float8
		FROM alerts
		WHERE tenant_id = $1
		  AND created_at > now() - interval '30 days'
		  AND deleted_at IS NULL
		GROUP BY severity`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("mttr by severity: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			severity   string
			entry      model.MTTREntry
			resolveAvg *float64
		)
		if err := rows.Scan(
			&severity,
			&entry.AvgResponseHours,
			&entry.MedianResponseHours,
			&entry.P95ResponseHours,
			&resolveAvg,
			&entry.SampleSize,
			&entry.SLACompliance,
		); err != nil {
			return nil, err
		}
		entry.AvgResolveHours = resolveAvg
		report.BySeverity[severity] = entry
		if c.metrics != nil && c.metrics.MTTRHours != nil {
			c.metrics.MTTRHours.WithLabelValues(severity, "avg").Set(entry.AvgResponseHours)
			c.metrics.MTTRHours.WithLabelValues(severity, "p50").Set(entry.MedianResponseHours)
			c.metrics.MTTRHours.WithLabelValues(severity, "p95").Set(entry.P95ResponseHours)
		}
		if c.metrics != nil && c.metrics.SLAComplianceRate != nil {
			c.metrics.SLAComplianceRate.WithLabelValues(severity).Set(entry.SLACompliance)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var overallResolve *float64
	if err := c.db.QueryRow(ctx, `
		SELECT
			AVG(EXTRACT(EPOCH FROM (COALESCE(acknowledged_at, resolved_at, now()) - created_at)) / 3600)::float8,
			PERCENTILE_CONT(0.5) WITHIN GROUP (
				ORDER BY EXTRACT(EPOCH FROM (COALESCE(acknowledged_at, resolved_at, now()) - created_at)) / 3600
			)::float8,
			PERCENTILE_CONT(0.95) WITHIN GROUP (
				ORDER BY EXTRACT(EPOCH FROM (COALESCE(acknowledged_at, resolved_at, now()) - created_at)) / 3600
			)::float8,
			AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 3600)
				FILTER (WHERE resolved_at IS NOT NULL)::float8,
			COUNT(*)::int,
			100.0 * AVG(
				CASE
					WHEN severity = 'critical' AND EXTRACT(EPOCH FROM (COALESCE(acknowledged_at, resolved_at, now()) - created_at)) / 3600 <= 4 THEN 1
					WHEN severity = 'high' AND EXTRACT(EPOCH FROM (COALESCE(acknowledged_at, resolved_at, now()) - created_at)) / 3600 <= 8 THEN 1
					WHEN severity = 'medium' AND EXTRACT(EPOCH FROM (COALESCE(acknowledged_at, resolved_at, now()) - created_at)) / 3600 <= 24 THEN 1
					WHEN severity = 'low' AND EXTRACT(EPOCH FROM (COALESCE(acknowledged_at, resolved_at, now()) - created_at)) / 3600 <= 72 THEN 1
					ELSE 0
				END
			)::float8
		FROM alerts
		WHERE tenant_id = $1
		  AND created_at > now() - interval '30 days'
		  AND deleted_at IS NULL`,
		tenantID,
	).Scan(
		&report.Overall.AvgResponseHours,
		&report.Overall.MedianResponseHours,
		&report.Overall.P95ResponseHours,
		&overallResolve,
		&report.Overall.SampleSize,
		&report.Overall.SLACompliance,
	); err != nil {
		return nil, fmt.Errorf("mttr overall: %w", err)
	}
	report.Overall.AvgResolveHours = overallResolve
	return report, nil
}
