package engine

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Scheduler runs periodic maintenance tasks for the DSPM remediation module.
// It coordinates SLA breach detection, exception expiry checking, and policy
// evaluation across all tenants on a configurable interval.
type Scheduler struct {
	engine *RemediationEngine
	cfg    Config
	logger zerolog.Logger

	cancel context.CancelFunc
	mu     sync.Mutex
	done   chan struct{}
}

// NewScheduler creates a Scheduler bound to the given engine and configuration.
func NewScheduler(engine *RemediationEngine, cfg Config, logger zerolog.Logger) *Scheduler {
	return &Scheduler{
		engine: engine,
		cfg:    cfg,
		logger: logger.With().Str("component", "remediation_scheduler").Logger(),
	}
}

// Start launches the scheduler's background goroutine. It runs a ticker at the
// configured ScheduleIntervalMinutes and executes maintenance tasks for each
// tenant returned by tenantProvider on every tick.
//
// The tenantProvider function is called on each tick to obtain the current list
// of tenant IDs. This allows the scheduler to adapt to tenants being added or
// removed without restart.
//
// Start is safe to call only once; subsequent calls have no effect until Stop
// is called.
func (s *Scheduler) Start(ctx context.Context, tenantProvider func(ctx context.Context) ([]uuid.UUID, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Prevent double-start.
	if s.cancel != nil {
		return
	}

	schedulerCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.done = make(chan struct{})

	interval := time.Duration(s.cfg.ScheduleIntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = 15 * time.Minute
	}

	s.logger.Info().
		Dur("interval", interval).
		Bool("sla_checking", s.cfg.EnableSLAChecking).
		Bool("auto_remediation", s.cfg.EnableAutoRemediation).
		Msg("scheduler starting")

	go s.run(schedulerCtx, interval, tenantProvider)
}

// Stop cancels the scheduler's background goroutine and waits for it to exit.
// It is safe to call Stop multiple times.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel == nil {
		return
	}

	s.logger.Info().Msg("scheduler stopping")

	s.cancel()
	s.cancel = nil

	// Wait for the goroutine to finish.
	if s.done != nil {
		<-s.done
		s.done = nil
	}

	s.logger.Info().Msg("scheduler stopped")
}

// run is the main loop executed in a background goroutine.
func (s *Scheduler) run(ctx context.Context, interval time.Duration, tenantProvider func(ctx context.Context) ([]uuid.UUID, error)) {
	defer close(s.done)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info().Msg("scheduler context cancelled; exiting")
			return

		case <-ticker.C:
			s.tick(ctx, tenantProvider)
		}
	}
}

// tick performs one round of maintenance tasks across all tenants.
func (s *Scheduler) tick(ctx context.Context, tenantProvider func(ctx context.Context) ([]uuid.UUID, error)) {
	start := time.Now()

	tenants, err := tenantProvider(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("scheduler tick: failed to retrieve tenant list")
		return
	}

	if len(tenants) == 0 {
		s.logger.Debug().Msg("scheduler tick: no tenants to process")
		return
	}

	s.logger.Info().
		Int("tenant_count", len(tenants)).
		Msg("scheduler tick starting")

	totalSLABreaches := 0
	totalExceptionsExpired := 0
	totalViolations := 0

	for _, tenantID := range tenants {
		// Check for context cancellation between tenants.
		if ctx.Err() != nil {
			s.logger.Info().Msg("scheduler tick: context cancelled; aborting tenant iteration")
			return
		}

		// 1. Check SLA breaches.
		if s.cfg.EnableSLAChecking {
			breaches, slaErr := s.engine.CheckSLABreaches(ctx, tenantID)
			if slaErr != nil {
				s.logger.Error().Err(slaErr).
					Str("tenant_id", tenantID.String()).
					Msg("scheduler tick: SLA breach check failed")
			} else {
				totalSLABreaches += breaches
			}
		}

		// 2. Check exception expiry.
		expired, expErr := s.engine.CheckExceptionExpiry(ctx, tenantID)
		if expErr != nil {
			s.logger.Error().Err(expErr).
				Str("tenant_id", tenantID.String()).
				Msg("scheduler tick: exception expiry check failed")
		} else {
			totalExceptionsExpired += expired
		}

		// 3. Evaluate policies (only if auto-remediation is enabled).
		if s.cfg.EnableAutoRemediation {
			violations, polErr := s.engine.EvaluatePolicies(ctx, tenantID)
			if polErr != nil {
				s.logger.Error().Err(polErr).
					Str("tenant_id", tenantID.String()).
					Msg("scheduler tick: policy evaluation failed")
			} else {
				totalViolations += len(violations)
			}
		}
	}

	elapsed := time.Since(start)

	s.logger.Info().
		Int("tenant_count", len(tenants)).
		Int("sla_breaches", totalSLABreaches).
		Int("exceptions_expired", totalExceptionsExpired).
		Int("violations_found", totalViolations).
		Dur("elapsed", elapsed).
		Msg("scheduler tick complete")
}
