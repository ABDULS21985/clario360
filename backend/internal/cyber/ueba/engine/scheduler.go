package engine

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/ueba/collector"
	uebarepo "github.com/clario360/platform/internal/cyber/ueba/repository"
)

type Scheduler struct {
	engine      *UEBAEngine
	profileRepo *uebarepo.ProfileRepository
	collector   *collector.AccessEventCollector
	logger      zerolog.Logger

	mu      sync.Mutex
	running map[uuid.UUID]context.CancelFunc
}

func NewScheduler(engine *UEBAEngine, profileRepo *uebarepo.ProfileRepository, collector *collector.AccessEventCollector, logger zerolog.Logger) *Scheduler {
	return &Scheduler{
		engine:      engine,
		profileRepo: profileRepo,
		collector:   collector,
		logger:      logger.With().Str("component", "ueba-scheduler").Logger(),
		running:     make(map[uuid.UUID]context.CancelFunc),
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	if err := s.syncTenants(ctx); err != nil {
		s.logger.Warn().Err(err).Msg("initial ueba tenant sync failed")
	}

	for {
		select {
		case <-ctx.Done():
			s.stopAll()
			return nil
		case <-ticker.C:
			if err := s.syncTenants(ctx); err != nil {
				s.logger.Warn().Err(err).Msg("scheduled ueba tenant sync failed")
			}
		}
	}
}

func (s *Scheduler) syncTenants(ctx context.Context) error {
	profileTenants, err := s.profileRepo.ListTenantIDs(ctx)
	if err != nil {
		return err
	}
	collectorTenants, err := s.collector.ListTenantIDs(ctx)
	if err != nil {
		return err
	}
	tenantSet := make(map[uuid.UUID]struct{}, len(profileTenants)+len(collectorTenants))
	for _, tenantID := range profileTenants {
		tenantSet[tenantID] = struct{}{}
	}
	for _, tenantID := range collectorTenants {
		tenantSet[tenantID] = struct{}{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for tenantID := range tenantSet {
		if _, ok := s.running[tenantID]; ok {
			continue
		}
		tenantCtx, cancel := context.WithCancel(ctx)
		s.running[tenantID] = cancel
		go func(id uuid.UUID) {
			if err := s.engine.Run(tenantCtx, id); err != nil && tenantCtx.Err() == nil {
				s.logger.Error().Err(err).Str("tenant_id", id.String()).Msg("tenant ueba loop stopped with error")
			}
		}(tenantID)
	}
	return nil
}

func (s *Scheduler) stopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for tenantID, cancel := range s.running {
		cancel()
		delete(s.running, tenantID)
	}
}
