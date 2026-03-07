package scheduler

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/acta/repository"
	"github.com/clario360/platform/internal/acta/service"
)

type ComplianceScheduler struct {
	store    *repository.Store
	checker  *service.ComplianceService
	interval time.Duration
	hourUTC  int
	logger   zerolog.Logger
}

func NewComplianceScheduler(store *repository.Store, checker *service.ComplianceService, interval time.Duration, hourUTC int, logger zerolog.Logger) *ComplianceScheduler {
	return &ComplianceScheduler{
		store:    store,
		checker:  checker,
		interval: interval,
		hourUTC:  hourUTC,
		logger:   logger.With().Str("component", "acta_compliance_scheduler").Logger(),
	}
}

func (s *ComplianceScheduler) Run(ctx context.Context) error {
	delay := time.Until(nextRunAt(time.Now().UTC(), s.hourUTC))
	timer := time.NewTimer(delay)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			if err := s.runOnce(ctx); err != nil {
				s.logger.Error().Err(err).Msg("scheduled compliance run failed")
			}
			timer.Reset(s.interval)
		}
	}
}

func (s *ComplianceScheduler) runOnce(ctx context.Context) error {
	tenantIDs, err := s.store.ListTenantIDs(ctx)
	if err != nil {
		return err
	}
	for _, tenantID := range tenantIDs {
		report, err := s.checker.RunChecks(ctx, tenantID)
		if err != nil {
			return err
		}
		if report.NonCompliantCount > 0 {
			s.logger.Warn().
				Str("tenant_id", tenantID.String()).
				Int("non_compliant_count", report.NonCompliantCount).
				Float64("score", report.Score).
				Msg("acta compliance scheduler found non-compliant results")
		}
	}
	return nil
}

func nextRunAt(now time.Time, hourUTC int) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), hourUTC, 0, 0, 0, time.UTC)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}
