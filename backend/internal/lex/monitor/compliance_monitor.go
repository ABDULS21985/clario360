package monitor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/lex/repository"
	"github.com/clario360/platform/internal/lex/service"
)

type ComplianceMonitor struct {
	contracts  *repository.ContractRepository
	compliance *service.ComplianceService
	interval   time.Duration
	logger     zerolog.Logger
}

func NewComplianceMonitor(contracts *repository.ContractRepository, compliance *service.ComplianceService, interval time.Duration, logger zerolog.Logger) *ComplianceMonitor {
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	return &ComplianceMonitor{
		contracts:  contracts,
		compliance: compliance,
		interval:   interval,
		logger:     logger.With().Str("component", "lex-compliance-monitor").Logger(),
	}
}

func (m *ComplianceMonitor) Run(ctx context.Context) error {
	if err := m.RunOnce(ctx); err != nil && ctx.Err() == nil {
		m.logger.Error().Err(err).Msg("compliance monitor iteration failed")
	}

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := m.RunOnce(ctx); err != nil && ctx.Err() == nil {
				m.logger.Error().Err(err).Msg("compliance monitor iteration failed")
			}
		}
	}
}

func (m *ComplianceMonitor) RunOnce(ctx context.Context) error {
	tenantIDs, err := m.contracts.ListTenantIDs(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, tenantID := range tenantIDs {
		if _, err := m.compliance.RunChecks(ctx, tenantID, nil); err != nil {
			errs = append(errs, fmt.Errorf("tenant %s: %w", tenantID, err))
		}
	}
	return errors.Join(errs...)
}
