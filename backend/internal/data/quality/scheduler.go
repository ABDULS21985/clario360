package quality

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/repository"
)

type Scheduler struct {
	ruleRepo  *repository.QualityRuleRepository
	executor  *QualityExecutor
	logger    zerolog.Logger
	interval  time.Duration
	stopCh    chan struct{}
	doneCh    chan struct{}
}

func NewScheduler(ruleRepo *repository.QualityRuleRepository, executor *QualityExecutor, logger zerolog.Logger, interval time.Duration) *Scheduler {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &Scheduler{
		ruleRepo: ruleRepo,
		executor: executor,
		logger:   logger,
		interval: interval,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	go func() {
		defer close(s.doneCh)
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stopCh:
				return
			case <-ticker.C:
				s.poll(ctx)
			}
		}
	}()
}

func (s *Scheduler) Stop() {
	close(s.stopCh)
	<-s.doneCh
}

func (s *Scheduler) poll(ctx context.Context) {
	rules, err := s.ruleRepo.ListDue(ctx, time.Now().UTC(), 50)
	if err != nil {
		s.logger.Error().Err(err).Msg("poll due quality rules")
		return
	}
	for _, rule := range rules {
		if _, err := s.executor.RunCheck(ctx, rule.TenantID, rule.ID, nil); err != nil {
			s.logger.Error().Err(err).Str("rule_id", rule.ID.String()).Msg("execute scheduled quality rule")
		}
	}
}

