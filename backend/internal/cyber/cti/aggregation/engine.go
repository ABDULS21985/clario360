package aggregation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

// DefaultPeriods defines the time windows aggregation is computed for.
var DefaultPeriods = []Period{
	{Label: "24h", Duration: 24 * time.Hour},
	{Label: "7d", Duration: 7 * 24 * time.Hour},
	{Label: "30d", Duration: 30 * 24 * time.Hour},
	{Label: "90d", Duration: 90 * 24 * time.Hour},
}

// Period represents a named time window for aggregation.
type Period struct {
	Label    string
	Duration time.Duration
}

// Engine orchestrates all CTI aggregation jobs.
type Engine struct {
	db        *pgxpool.Pool
	logger    zerolog.Logger
	Metrics   *Metrics
	geoAgg    *GeoAggregator
	sectorAgg *SectorAggregator
	execAgg   *ExecutiveAggregator
	trendCalc *TrendCalculator
}

// NewEngine creates an aggregation engine with its own Prometheus registry.
func NewEngine(db *pgxpool.Pool, parentReg *prometheus.Registry, logger zerolog.Logger) *Engine {
	m := NewMetrics(parentReg)
	tc := NewTrendCalculator(db, logger)
	return &Engine{
		db:        db,
		logger:    logger.With().Str("component", "cti-aggregation-engine").Logger(),
		Metrics:   m,
		geoAgg:    NewGeoAggregator(db, logger, m),
		sectorAgg: NewSectorAggregator(db, logger, m),
		execAgg:   NewExecutiveAggregator(db, logger, m, tc),
		trendCalc: tc,
	}
}

// RunFullAggregation refreshes all aggregation tables for a single tenant.
func (e *Engine) RunFullAggregation(ctx context.Context, tenantID string) error {
	start := time.Now()
	e.logger.Info().Str("tenant_id", tenantID).Msg("starting full CTI aggregation")

	now := time.Now().UTC()
	var errs int

	for _, p := range DefaultPeriods {
		pStart := now.Add(-p.Duration)
		if err := e.geoAgg.Aggregate(ctx, tenantID, pStart, now, p.Label); err != nil {
			e.logger.Error().Err(err).Str("tenant_id", tenantID).Str("period", p.Label).Msg("geo aggregation failed")
			errs++
		}
		if err := e.sectorAgg.Aggregate(ctx, tenantID, pStart, now, p.Label); err != nil {
			e.logger.Error().Err(err).Str("tenant_id", tenantID).Str("period", p.Label).Msg("sector aggregation failed")
			errs++
		}
	}

	// Backfill top_threat_type labels
	_ = e.geoAgg.BackfillTopThreatTypes(ctx, tenantID)

	// Executive snapshot
	if err := e.execAgg.Aggregate(ctx, tenantID); err != nil {
		e.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("executive aggregation failed")
		errs++
	}

	elapsed := time.Since(start)
	e.Metrics.Duration.WithLabelValues(tenantID, "full").Observe(elapsed.Seconds())
	e.Metrics.RunsTotal.WithLabelValues("single_tenant").Inc()
	e.Metrics.LastRun.WithLabelValues(tenantID).SetToCurrentTime()

	e.logger.Info().Str("tenant_id", tenantID).Int("errors", errs).Dur("elapsed", elapsed).Msg("full CTI aggregation complete")

	if errs > 0 {
		return fmt.Errorf("aggregation completed with %d errors", errs)
	}
	return nil
}

// RunAllTenants discovers active tenants and aggregates for each with bounded concurrency.
func (e *Engine) RunAllTenants(ctx context.Context) error {
	tenantIDs, err := e.GetActiveTenants(ctx)
	if err != nil {
		return fmt.Errorf("get active tenants: %w", err)
	}
	if len(tenantIDs) == 0 {
		e.logger.Debug().Msg("no active tenants for aggregation")
		return nil
	}

	e.logger.Info().Int("tenants", len(tenantIDs)).Msg("running CTI aggregation for all tenants")

	sem := make(chan struct{}, 5) // max concurrency
	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalErrors int

	for _, tid := range tenantIDs {
		wg.Add(1)
		sem <- struct{}{}
		go func(tenantID string) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := e.RunFullAggregation(ctx, tenantID); err != nil {
				mu.Lock()
				totalErrors++
				mu.Unlock()
			}
		}(tid)
	}
	wg.Wait()

	e.Metrics.RunsTotal.WithLabelValues("all_tenants").Inc()

	if totalErrors > 0 {
		return fmt.Errorf("aggregation had errors in %d/%d tenants", totalErrors, len(tenantIDs))
	}
	return nil
}

// GetActiveTenants returns distinct tenant IDs that have CTI threat events.
func (e *Engine) GetActiveTenants(ctx context.Context) ([]string, error) {
	rows, err := e.db.Query(ctx, `
		SELECT DISTINCT tenant_id::text FROM cti_threat_events
		WHERE deleted_at IS NULL
		GROUP BY tenant_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
