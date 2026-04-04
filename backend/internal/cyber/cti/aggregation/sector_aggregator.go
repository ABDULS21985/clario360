package aggregation

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// SectorAggregator computes cti_sector_threat_summary rows from raw threat events.
type SectorAggregator struct {
	db      *pgxpool.Pool
	logger  zerolog.Logger
	metrics *Metrics
}

func NewSectorAggregator(db *pgxpool.Pool, logger zerolog.Logger, m *Metrics) *SectorAggregator {
	return &SectorAggregator{db: db, logger: logger.With().Str("aggregator", "sector").Logger(), metrics: m}
}

func (s *SectorAggregator) Aggregate(ctx context.Context, tenantID string, periodStart, periodEnd time.Time, periodLabel string) error {
	start := time.Now()

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenantContext(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("set tenant: %w", err)
	}

	result, err := tx.Exec(ctx, `
		WITH sector_stats AS (
			SELECT
				e.tenant_id,
				e.target_sector_id AS sector_id,
				COUNT(*) FILTER (WHERE sl.code = 'critical') AS crit,
				COUNT(*) FILTER (WHERE sl.code = 'high') AS high,
				COUNT(*) FILTER (WHERE sl.code = 'medium') AS med,
				COUNT(*) FILTER (WHERE sl.code = 'low') AS low,
				COUNT(*) AS total
			FROM cti_threat_events e
			LEFT JOIN cti_threat_severity_levels sl ON e.severity_id = sl.id
			WHERE e.tenant_id = $1
			  AND e.deleted_at IS NULL
			  AND e.is_false_positive = false
			  AND e.target_sector_id IS NOT NULL
			  AND e.first_seen_at >= $2
			  AND e.first_seen_at < $3
			GROUP BY e.tenant_id, e.target_sector_id
		)
		INSERT INTO cti_sector_threat_summary (
			tenant_id, sector_id,
			severity_critical_count, severity_high_count, severity_medium_count, severity_low_count,
			total_count, period_start, period_end, computed_at
		)
		SELECT tenant_id, sector_id, crit, high, med, low, total, $2, $3, NOW()
		FROM sector_stats
		ON CONFLICT (tenant_id, sector_id, period_start, period_end)
		DO UPDATE SET
			severity_critical_count = EXCLUDED.severity_critical_count,
			severity_high_count = EXCLUDED.severity_high_count,
			severity_medium_count = EXCLUDED.severity_medium_count,
			severity_low_count = EXCLUDED.severity_low_count,
			total_count = EXCLUDED.total_count,
			computed_at = NOW()`,
		tenantID, periodStart, periodEnd)
	if err != nil {
		s.metrics.Errors.WithLabelValues(tenantID, "sector_"+periodLabel).Inc()
		return fmt.Errorf("sector aggregation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	elapsed := time.Since(start)
	s.metrics.Duration.WithLabelValues(tenantID, "sector_"+periodLabel).Observe(elapsed.Seconds())
	s.logger.Debug().
		Str("tenant_id", tenantID).
		Str("period", periodLabel).
		Int64("rows", result.RowsAffected()).
		Dur("elapsed", elapsed).
		Msg("sector aggregation complete")
	return nil
}
