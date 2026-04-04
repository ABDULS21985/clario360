package aggregation

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

// ScheduleConfig controls the intervals for periodic aggregation jobs.
type ScheduleConfig struct {
	FullInterval      time.Duration // default: 5 min
	ExecutiveInterval time.Duration // default: 2 min
	CleanupInterval   time.Duration // default: 1 hour
	MaxAggregationAge time.Duration // default: 7 days
}

// DefaultScheduleConfig provides sensible defaults for development and production.
var DefaultScheduleConfig = ScheduleConfig{
	FullInterval:      5 * time.Minute,
	ExecutiveInterval: 2 * time.Minute,
	CleanupInterval:   1 * time.Hour,
	MaxAggregationAge: 7 * 24 * time.Hour,
}

// Scheduler runs CTI aggregation jobs on a periodic schedule.
// It complements the event-driven AggregationTriggerConsumer by ensuring
// dashboards stay fresh even during low event-flow periods.
type Scheduler struct {
	engine *Engine
	config ScheduleConfig
	logger zerolog.Logger
}

func NewScheduler(engine *Engine, config ScheduleConfig, logger zerolog.Logger) *Scheduler {
	return &Scheduler{
		engine: engine,
		config: config,
		logger: logger.With().Str("component", "cti-aggregation-scheduler").Logger(),
	}
}

// Start runs the scheduler until ctx is cancelled.
// Signature matches the errgroup pattern used throughout cyber-service.
func (s *Scheduler) Start(ctx context.Context) error {
	s.logger.Info().
		Dur("full_interval", s.config.FullInterval).
		Dur("executive_interval", s.config.ExecutiveInterval).
		Dur("cleanup_interval", s.config.CleanupInterval).
		Msg("CTI aggregation scheduler started")

	fullTicker := time.NewTicker(s.config.FullInterval)
	execTicker := time.NewTicker(s.config.ExecutiveInterval)
	cleanupTicker := time.NewTicker(s.config.CleanupInterval)

	defer fullTicker.Stop()
	defer execTicker.Stop()
	defer cleanupTicker.Stop()

	// Run once immediately on startup
	s.runSafe(ctx, "initial_full", func() error { return s.engine.RunAllTenants(ctx) })

	for {
		select {
		case <-ctx.Done():
			s.logger.Info().Msg("CTI aggregation scheduler stopped")
			return ctx.Err()

		case <-fullTicker.C:
			go s.runSafe(ctx, "scheduled_full", func() error { return s.engine.RunAllTenants(ctx) })

		case <-execTicker.C:
			go s.runSafe(ctx, "executive_refresh", func() error {
				tenants, err := s.engine.GetActiveTenants(ctx)
				if err != nil {
					return err
				}
				for _, tid := range tenants {
					if err := s.engine.execAgg.Aggregate(ctx, tid); err != nil {
						s.logger.Warn().Err(err).Str("tenant_id", tid).Msg("executive refresh failed")
					}
				}
				return nil
			})

		case <-cleanupTicker.C:
			go s.runSafe(ctx, "cleanup", func() error { return s.cleanup(ctx) })
		}
	}
}

func (s *Scheduler) runSafe(ctx context.Context, jobName string, fn func() error) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error().Interface("panic", r).Str("job", jobName).Msg("aggregation job panicked")
		}
	}()

	start := time.Now()
	if err := fn(); err != nil {
		s.logger.Error().Err(err).Str("job", jobName).Dur("elapsed", time.Since(start)).Msg("aggregation job failed")
	} else {
		s.logger.Info().Str("job", jobName).Dur("elapsed", time.Since(start)).Msg("aggregation job completed")
	}
}

func (s *Scheduler) cleanup(ctx context.Context) error {
	cutoff := time.Now().UTC().Add(-s.config.MaxAggregationAge)
	if _, err := s.engine.db.Exec(ctx, `DELETE FROM cti_geo_threat_summary WHERE computed_at < $1`, cutoff); err != nil {
		s.logger.Warn().Err(err).Msg("cleanup geo summary")
	}
	if _, err := s.engine.db.Exec(ctx, `DELETE FROM cti_sector_threat_summary WHERE computed_at < $1`, cutoff); err != nil {
		s.logger.Warn().Err(err).Msg("cleanup sector summary")
	}
	s.logger.Info().Time("cutoff", cutoff).Msg("aggregation cleanup complete")
	return nil
}
