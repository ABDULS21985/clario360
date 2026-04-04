package shadow

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

type ComparisonRunner interface {
	AggregateAllShadowModels(ctx context.Context) error
}

type Scheduler struct {
	runner   ComparisonRunner
	interval time.Duration
	logger   zerolog.Logger
}

func NewScheduler(runner ComparisonRunner, interval time.Duration, logger zerolog.Logger) *Scheduler {
	if interval <= 0 {
		interval = time.Hour
	}
	return &Scheduler{
		runner:   runner,
		interval: interval,
		logger:   logger.With().Str("component", "ai_shadow_scheduler").Logger(),
	}
}

func (s *Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		if err := s.runner.AggregateAllShadowModels(ctx); err != nil {
			s.logger.Error().Err(err).Msg("shadow comparison run failed")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
