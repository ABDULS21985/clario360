package pipeline

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
)

type Scheduler struct {
	pipelineRepo *repository.PipelineRepository
	engine       *Engine
	logger       zerolog.Logger
	interval     time.Duration
	stopCh       chan struct{}
	doneCh       chan struct{}
}

func NewScheduler(pipelineRepo *repository.PipelineRepository, engine *Engine, logger zerolog.Logger, interval time.Duration) *Scheduler {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	return &Scheduler{
		pipelineRepo: pipelineRepo,
		engine:       engine,
		logger:       logger,
		interval:     interval,
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
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
	due, err := s.pipelineRepo.ListDue(ctx, time.Now().UTC(), 50)
	if err != nil {
		s.logger.Error().Err(err).Msg("poll due pipelines")
		return
	}
	for _, item := range due {
		if _, err := s.engine.ExecutePipeline(ctx, item.ID, string(model.PipelineTriggerSchedule), nil); err != nil {
			s.logger.Error().Err(err).Str("pipeline_id", item.ID.String()).Msg("execute scheduled pipeline")
		}
	}
}

func NextRunTime(schedule *string, now time.Time) (*time.Time, error) {
	if schedule == nil || strings.TrimSpace(*schedule) == "" {
		return nil, nil
	}
	raw := strings.TrimSpace(*schedule)
	switch raw {
	case "@hourly":
		next := now.Truncate(time.Hour).Add(time.Hour)
		return &next, nil
	case "@daily":
		next := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Add(24 * time.Hour)
		return &next, nil
	case "@weekly":
		next := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Add(7 * 24 * time.Hour)
		return &next, nil
	}
	fields := strings.Fields(raw)
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron expression must have 5 fields")
	}
	candidate := now.Truncate(time.Minute).Add(time.Minute)
	deadline := now.Add(366 * 24 * time.Hour)
	for !candidate.After(deadline) {
		if matchesCronField(fields[0], candidate.Minute()) &&
			matchesCronField(fields[1], candidate.Hour()) &&
			matchesCronField(fields[2], candidate.Day()) &&
			matchesCronField(fields[3], int(candidate.Month())) &&
			matchesCronField(fields[4], int(candidate.Weekday())) {
			return &candidate, nil
		}
		candidate = candidate.Add(time.Minute)
	}
	return nil, fmt.Errorf("could not compute next run time")
}

func matchesCronField(field string, value int) bool {
	if field == "*" {
		return true
	}
	for _, part := range strings.Split(field, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "*/") {
			step, err := strconv.Atoi(strings.TrimPrefix(part, "*/"))
			return err == nil && step > 0 && value%step == 0
		}
		if strings.Contains(part, "-") {
			rangeParts := strings.SplitN(part, "-", 2)
			start, err1 := strconv.Atoi(rangeParts[0])
			end, err2 := strconv.Atoi(rangeParts[1])
			if err1 == nil && err2 == nil && value >= start && value <= end {
				return true
			}
			continue
		}
		if parsed, err := strconv.Atoi(part); err == nil && parsed == value {
			return true
		}
	}
	return false
}

func tenantSlotKey(tenantID uuid.UUID) string {
	return tenantID.String()
}

