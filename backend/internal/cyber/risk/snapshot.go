package risk

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/metrics"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/events"
)

type SnapshotService struct {
	db          *pgxpool.Pool
	scorer      *RiskScorer
	historyRepo *repository.RiskHistoryRepository
	producer    *events.Producer
	logger      zerolog.Logger
	metrics     *metrics.Metrics
}

func NewSnapshotService(
	db *pgxpool.Pool,
	scorer *RiskScorer,
	historyRepo *repository.RiskHistoryRepository,
	producer *events.Producer,
	m *metrics.Metrics,
	logger zerolog.Logger,
) *SnapshotService {
	return &SnapshotService{
		db:          db,
		scorer:      scorer,
		historyRepo: historyRepo,
		producer:    producer,
		logger:      logger.With().Str("component", "risk-snapshot").Logger(),
		metrics:     m,
	}
}

func (s *SnapshotService) RunDailySnapshot(ctx context.Context) error {
	for {
		nextRun := nextTwoAMUTC(time.Now().UTC())
		timer := time.NewTimer(time.Until(nextRun))
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
			if err := s.snapshotAllTenants(ctx, "daily", nil); err != nil && ctx.Err() == nil {
				s.logger.Error().Err(err).Msg("daily risk snapshot run failed")
			}
		}
	}
}

func (s *SnapshotService) SaveEventTriggeredSnapshot(ctx context.Context, tenantID uuid.UUID, eventType string) (*model.OrganizationRiskScore, error) {
	return s.captureSnapshot(ctx, tenantID, "event_triggered", &eventType)
}

func (s *SnapshotService) SaveOnDemandSnapshot(ctx context.Context, tenantID uuid.UUID) (*model.OrganizationRiskScore, error) {
	return s.captureSnapshot(ctx, tenantID, "on_demand", nil)
}

func (s *SnapshotService) captureSnapshot(ctx context.Context, tenantID uuid.UUID, snapshotType string, triggerEvent *string) (*model.OrganizationRiskScore, error) {
	start := time.Now()
	score, err := s.scorer.CalculateOrganizationRiskFresh(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if err := s.historyRepo.Upsert(ctx, tenantID, snapshotType, triggerEvent, score); err != nil {
		return nil, fmt.Errorf("persist risk snapshot: %w", err)
	}
	if s.metrics != nil && s.metrics.RiskSnapshotDuration != nil {
		s.metrics.RiskSnapshotDuration.Observe(time.Since(start).Seconds())
	}
	s.publishEvent(ctx, "cyber.risk.snapshot_saved", tenantID.String(), map[string]interface{}{
		"tenant_id":     tenantID.String(),
		"snapshot_type": snapshotType,
		"score":         score.OverallScore,
	})
	return score, nil
}

func (s *SnapshotService) snapshotAllTenants(ctx context.Context, snapshotType string, triggerEvent *string) error {
	start := time.Now()
	rows, err := s.db.Query(ctx, `SELECT DISTINCT tenant_id FROM assets WHERE deleted_at IS NULL`)
	if err != nil {
		return fmt.Errorf("list tenants for risk snapshot: %w", err)
	}
	defer rows.Close()

	tenantIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var tenantID uuid.UUID
		if err := rows.Scan(&tenantID); err != nil {
			return err
		}
		tenantIDs = append(tenantIDs, tenantID)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, tenantID := range tenantIDs {
		score, err := s.captureSnapshot(ctx, tenantID, snapshotType, triggerEvent)
		if err != nil {
			s.logger.Error().Err(err).Str("tenant_id", tenantID.String()).Msg("risk snapshot failed for tenant")
			continue
		}
		s.logger.Info().
			Str("tenant_id", tenantID.String()).
			Float64("score", score.OverallScore).
			Str("grade", score.Grade).
			Msg("risk snapshot saved")
	}

	if s.metrics != nil && s.metrics.RiskSnapshotDuration != nil {
		s.metrics.RiskSnapshotDuration.Observe(time.Since(start).Seconds())
	}
	return nil
}

func (s *SnapshotService) publishEvent(ctx context.Context, eventType, tenantID string, data interface{}) {
	if s.producer == nil {
		return
	}
	event, err := events.NewEvent(eventType, "cyber-service", tenantID, data)
	if err != nil {
		s.logger.Error().Err(err).Str("event_type", eventType).Msg("build risk snapshot event")
		return
	}
	if err := s.producer.Publish(ctx, events.Topics.RiskEvents, event); err != nil {
		s.logger.Error().Err(err).Str("event_type", eventType).Msg("publish risk snapshot event")
	}
}

func nextTwoAMUTC(now time.Time) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, time.UTC)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}
