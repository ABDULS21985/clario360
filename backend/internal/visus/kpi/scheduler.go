package kpi

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/repository"
)

type Scheduler struct {
	engine   *KPIEngine
	kpiRepo  *repository.KPIRepository
	interval time.Duration
	logger   zerolog.Logger
}

func NewScheduler(engine *KPIEngine, kpiRepo *repository.KPIRepository, interval time.Duration, logger zerolog.Logger) *Scheduler {
	if interval <= 0 {
		interval = time.Minute
	}
	return &Scheduler{
		engine:   engine,
		kpiRepo:  kpiRepo,
		interval: interval,
		logger:   logger.With().Str("component", "visus_kpi_scheduler").Logger(),
	}
}

func (s *Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		if err := s.runOnce(ctx); err != nil && ctx.Err() == nil {
			s.logger.Error().Err(err).Msg("kpi scheduler iteration failed")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Scheduler) RunOnce(ctx context.Context) error {
	return s.runOnce(ctx)
}

func (s *Scheduler) runOnce(ctx context.Context) error {
	tenantIDs, err := s.kpiRepo.ListDueTenantIDs(ctx)
	if err != nil {
		return err
	}
	if len(tenantIDs) == 0 {
		return nil
	}
	semaphore := make(chan struct{}, 5)
	var wg sync.WaitGroup
	var firstErr error
	var mu sync.Mutex
	for _, tenantID := range tenantIDs {
		tenantID := tenantID
		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			if err := s.engine.TakeSnapshots(ctx, tenantID); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return firstErr
}

func IntervalForFrequency(frequency model.KPISnapshotFrequency) time.Duration {
	switch frequency {
	case model.KPIFrequency15m:
		return 15 * time.Minute
	case model.KPIFrequency4h:
		return 4 * time.Hour
	case model.KPIFrequencyDay:
		return 24 * time.Hour
	case model.KPIFrequencyWeek:
		return 7 * 24 * time.Hour
	default:
		return time.Hour
	}
}

func snapshotPeriodStart(now time.Time, frequency model.KPISnapshotFrequency) time.Time {
	return now.Add(-IntervalForFrequency(frequency))
}

func TenantIDsToStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}
