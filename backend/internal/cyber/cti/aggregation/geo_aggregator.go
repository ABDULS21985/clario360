package aggregation

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// GeoAggregator computes cti_geo_threat_summary rows from raw threat events.
type GeoAggregator struct {
	db      *pgxpool.Pool
	logger  zerolog.Logger
	metrics *Metrics
}

func NewGeoAggregator(db *pgxpool.Pool, logger zerolog.Logger, m *Metrics) *GeoAggregator {
	return &GeoAggregator{db: db, logger: logger.With().Str("aggregator", "geo").Logger(), metrics: m}
}

func (g *GeoAggregator) Aggregate(ctx context.Context, tenantID string, periodStart, periodEnd time.Time, periodLabel string) error {
	start := time.Now()

	tx, err := g.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		return fmt.Errorf("set tenant: %w", err)
	}

	result, err := tx.Exec(ctx, `
		WITH geo_stats AS (
			SELECT
				e.tenant_id,
				COALESCE(e.origin_country_code, 'XX') AS country_code,
				COALESCE(e.origin_city, 'Unknown') AS city,
				AVG(e.origin_latitude) AS latitude,
				AVG(e.origin_longitude) AS longitude,
				(array_agg(e.origin_region_id) FILTER (WHERE e.origin_region_id IS NOT NULL))[1] AS region_id,
				COUNT(*) FILTER (WHERE sl.code = 'critical') AS crit,
				COUNT(*) FILTER (WHERE sl.code = 'high') AS high,
				COUNT(*) FILTER (WHERE sl.code = 'medium') AS med,
				COUNT(*) FILTER (WHERE sl.code = 'low') AS low,
				COUNT(*) AS total,
				MODE() WITHIN GROUP (ORDER BY e.category_id) AS top_cat
			FROM cti_threat_events e
			LEFT JOIN cti_threat_severity_levels sl ON e.severity_id = sl.id
			WHERE e.tenant_id = $1
			  AND e.deleted_at IS NULL
			  AND e.is_false_positive = false
			  AND e.first_seen_at >= $2
			  AND e.first_seen_at < $3
			GROUP BY e.tenant_id, e.origin_country_code, e.origin_city
		)
		INSERT INTO cti_geo_threat_summary (
			tenant_id, country_code, city, latitude, longitude, region_id,
			severity_critical_count, severity_high_count, severity_medium_count, severity_low_count,
			total_count, top_category_id, period_start, period_end, computed_at
		)
		SELECT tenant_id, country_code, city, latitude, longitude, region_id,
			   crit, high, med, low, total, top_cat, $2, $3, NOW()
		FROM geo_stats
		ON CONFLICT (tenant_id, country_code, city, period_start, period_end)
		DO UPDATE SET
			latitude = EXCLUDED.latitude,
			longitude = EXCLUDED.longitude,
			region_id = EXCLUDED.region_id,
			severity_critical_count = EXCLUDED.severity_critical_count,
			severity_high_count = EXCLUDED.severity_high_count,
			severity_medium_count = EXCLUDED.severity_medium_count,
			severity_low_count = EXCLUDED.severity_low_count,
			total_count = EXCLUDED.total_count,
			top_category_id = EXCLUDED.top_category_id,
			computed_at = NOW()`,
		tenantID, periodStart, periodEnd)
	if err != nil {
		g.metrics.Errors.WithLabelValues(tenantID, "geo_"+periodLabel).Inc()
		return fmt.Errorf("geo aggregation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	elapsed := time.Since(start)
	g.metrics.Duration.WithLabelValues(tenantID, "geo_"+periodLabel).Observe(elapsed.Seconds())
	g.logger.Debug().
		Str("tenant_id", tenantID).
		Str("period", periodLabel).
		Int64("rows", result.RowsAffected()).
		Dur("elapsed", elapsed).
		Msg("geo aggregation complete")
	return nil
}

// BackfillTopThreatTypes resolves top_category_id → label for rows missing the label.
func (g *GeoAggregator) BackfillTopThreatTypes(ctx context.Context, tenantID string) error {
	_, err := g.db.Exec(ctx, `
		UPDATE cti_geo_threat_summary gs
		SET top_threat_type = tc.label
		FROM cti_threat_categories tc
		WHERE gs.top_category_id = tc.id
		  AND gs.tenant_id = $1
		  AND (gs.top_threat_type IS NULL OR gs.top_threat_type = '')`, tenantID)
	return err
}
