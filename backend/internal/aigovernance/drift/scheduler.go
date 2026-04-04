package drift

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

type DriftRunner interface {
	RunAllProductionModels(ctx context.Context) error
}

type Scheduler struct {
	runner   DriftRunner
	interval time.Duration
	logger   zerolog.Logger
}

func NewScheduler(runner DriftRunner, interval time.Duration, logger zerolog.Logger) *Scheduler {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	return &Scheduler{
		runner:   runner,
		interval: interval,
		logger:   logger.With().Str("component", "ai_drift_scheduler").Logger(),
	}
}

func (s *Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		if err := s.runner.RunAllProductionModels(ctx); err != nil {
			s.logger.Error().Err(err).Msg("drift calculation run failed")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
