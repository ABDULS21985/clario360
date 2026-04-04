package aggregation

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// ExecutiveAggregator computes the single-row cti_executive_snapshot per tenant.
type ExecutiveAggregator struct {
	db      *pgxpool.Pool
	logger  zerolog.Logger
	metrics *Metrics
	trend   *TrendCalculator
}

func NewExecutiveAggregator(db *pgxpool.Pool, logger zerolog.Logger, m *Metrics, trend *TrendCalculator) *ExecutiveAggregator {
	return &ExecutiveAggregator{
		db: db, logger: logger.With().Str("aggregator", "executive").Logger(),
		metrics: m, trend: trend,
	}
}

func (ea *ExecutiveAggregator) Aggregate(ctx context.Context, tenantID string) error {
	start := time.Now()

	tx, err := ea.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenantContext(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("set tenant: %w", err)
	}

	now := time.Now().UTC()

	// Event counts
	var ev24h, ev7d, ev30d int64
	err = tx.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE first_seen_at >= ($2::timestamptz - INTERVAL '24 hours')),
			COUNT(*) FILTER (WHERE first_seen_at >= ($2::timestamptz - INTERVAL '7 days')),
			COUNT(*) FILTER (WHERE first_seen_at >= ($2::timestamptz - INTERVAL '30 days'))
		FROM cti_threat_events
		WHERE tenant_id = $1 AND deleted_at IS NULL AND is_false_positive = false`,
		tenantID, now).Scan(&ev24h, &ev7d, &ev30d)
	if err != nil {
		return fmt.Errorf("event counts: %w", err)
	}

	// Campaign stats
	var activeCampaigns, criticalCampaigns int
	err = tx.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE c.status = 'active'),
			COUNT(*) FILTER (WHERE c.status = 'active' AND sl.code = 'critical')
		FROM cti_campaigns c
		LEFT JOIN cti_threat_severity_levels sl ON c.severity_id = sl.id
		WHERE c.tenant_id = $1 AND c.deleted_at IS NULL`,
		tenantID).Scan(&activeCampaigns, &criticalCampaigns)
	if err != nil {
		return fmt.Errorf("campaign stats: %w", err)
	}

	// Total IOCs across active campaigns
	var totalIOCs int64
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(ioc_count), 0)
		FROM cti_campaigns
		WHERE tenant_id = $1 AND status = 'active' AND deleted_at IS NULL`,
		tenantID).Scan(&totalIOCs)
	if err != nil {
		return fmt.Errorf("ioc count: %w", err)
	}

	// Brand abuse stats (exclude resolved)
	var brandCritical, brandTotal int
	err = tx.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE risk_level = 'critical'),
			COUNT(*)
		FROM cti_brand_abuse_incidents
		WHERE tenant_id = $1 AND deleted_at IS NULL
		  AND takedown_status NOT IN ('taken_down', 'false_positive')`,
		tenantID).Scan(&brandCritical, &brandTotal)
	if err != nil {
		return fmt.Errorf("brand abuse stats: %w", err)
	}

	// Top targeted sector (30 days)
	var topSectorID *string
	err = tx.QueryRow(ctx, `
		SELECT target_sector_id::text
		FROM cti_threat_events
		WHERE tenant_id = $1 AND deleted_at IS NULL AND target_sector_id IS NOT NULL
		  AND first_seen_at >= ($2::timestamptz - INTERVAL '30 days')
		GROUP BY target_sector_id ORDER BY COUNT(*) DESC LIMIT 1`,
		tenantID, now).Scan(&topSectorID)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("top sector: %w", err)
	}

	// Top origin country (30 days)
	var topOriginCountry *string
	err = tx.QueryRow(ctx, `
		SELECT origin_country_code
		FROM cti_threat_events
		WHERE tenant_id = $1 AND deleted_at IS NULL AND origin_country_code IS NOT NULL
		  AND first_seen_at >= ($2::timestamptz - INTERVAL '30 days')
		GROUP BY origin_country_code ORDER BY COUNT(*) DESC LIMIT 1`,
		tenantID, now).Scan(&topOriginCountry)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("top origin: %w", err)
	}

	// MTTD (mean time to detect: first_seen → created_at)
	var mttdHours float64
	_ = tx.QueryRow(ctx, `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (created_at - first_seen_at)) / 3600.0), 0)
		FROM cti_threat_events
		WHERE tenant_id = $1 AND deleted_at IS NULL
		  AND first_seen_at >= ($2::timestamptz - INTERVAL '30 days')
		  AND created_at > first_seen_at`,
		tenantID, now).Scan(&mttdHours)

	// MTTR (mean time to respond: first_seen → resolved_at)
	var mttrHours float64
	_ = tx.QueryRow(ctx, `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (resolved_at - first_seen_at)) / 3600.0), 0)
		FROM cti_threat_events
		WHERE tenant_id = $1 AND deleted_at IS NULL
		  AND resolved_at IS NOT NULL
		  AND first_seen_at >= ($2::timestamptz - INTERVAL '30 days')`,
		tenantID, now).Scan(&mttrHours)

	// Composite risk score
	riskScore := computeRiskScore(ev24h, int64(activeCampaigns), int64(criticalCampaigns), int64(brandCritical))

	// Trend
	trendDir, trendPct := ea.trend.CalculateTx(ctx, tx, tenantID, now)

	// Upsert
	_, err = tx.Exec(ctx, `
		INSERT INTO cti_executive_snapshot (
			tenant_id, total_events_24h, total_events_7d, total_events_30d,
			active_campaigns_count, critical_campaigns_count, total_iocs,
			brand_abuse_critical_count, brand_abuse_total_count,
			top_targeted_sector_id, top_threat_origin_country,
			mean_time_to_detect_hours, mean_time_to_respond_hours,
			risk_score_overall, trend_direction, trend_percentage, computed_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::uuid,$11,$12,$13,$14,$15,$16,NOW())
		ON CONFLICT (tenant_id) DO UPDATE SET
			total_events_24h = EXCLUDED.total_events_24h,
			total_events_7d = EXCLUDED.total_events_7d,
			total_events_30d = EXCLUDED.total_events_30d,
			active_campaigns_count = EXCLUDED.active_campaigns_count,
			critical_campaigns_count = EXCLUDED.critical_campaigns_count,
			total_iocs = EXCLUDED.total_iocs,
			brand_abuse_critical_count = EXCLUDED.brand_abuse_critical_count,
			brand_abuse_total_count = EXCLUDED.brand_abuse_total_count,
			top_targeted_sector_id = EXCLUDED.top_targeted_sector_id,
			top_threat_origin_country = EXCLUDED.top_threat_origin_country,
			mean_time_to_detect_hours = EXCLUDED.mean_time_to_detect_hours,
			mean_time_to_respond_hours = EXCLUDED.mean_time_to_respond_hours,
			risk_score_overall = EXCLUDED.risk_score_overall,
			trend_direction = EXCLUDED.trend_direction,
			trend_percentage = EXCLUDED.trend_percentage,
			computed_at = NOW()`,
		tenantID, ev24h, ev7d, ev30d,
		activeCampaigns, criticalCampaigns, totalIOCs,
		brandCritical, brandTotal,
		topSectorID, topOriginCountry,
		mttdHours, mttrHours,
		riskScore, trendDir, trendPct)
	if err != nil {
		ea.metrics.Errors.WithLabelValues(tenantID, "executive").Inc()
		return fmt.Errorf("executive upsert: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	elapsed := time.Since(start)
	ea.metrics.Duration.WithLabelValues(tenantID, "executive").Observe(elapsed.Seconds())
	ea.metrics.RiskScore.WithLabelValues(tenantID).Set(riskScore)
	ea.metrics.Events24h.WithLabelValues(tenantID).Set(float64(ev24h))
	ea.metrics.LastRun.WithLabelValues(tenantID).SetToCurrentTime()

	ea.logger.Debug().
		Str("tenant_id", tenantID).
		Float64("risk_score", riskScore).
		Int64("events_24h", ev24h).
		Str("trend", trendDir).
		Dur("elapsed", elapsed).
		Msg("executive aggregation complete")
	return nil
}

// computeRiskScore — weighted composite 0-100.
// Weights: event volume (30%), active campaigns (25%), critical campaigns (25%), brand abuse (20%).
func computeRiskScore(events24h, activeCampaigns, criticalCampaigns, brandCritical int64) float64 {
	eventScore := math.Min(float64(events24h)/100.0, 1.0) * 30.0
	campaignScore := math.Min(float64(activeCampaigns)/10.0, 1.0) * 25.0
	criticalScore := math.Min(float64(criticalCampaigns)/5.0, 1.0) * 25.0
	brandScore := math.Min(float64(brandCritical)/5.0, 1.0) * 20.0
	return math.Round((eventScore+campaignScore+criticalScore+brandScore)*100) / 100
}
