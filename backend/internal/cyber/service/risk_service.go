package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	cyberrisk "github.com/clario360/platform/internal/cyber/risk"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/events"
)

type RiskService struct {
	scorer      *cyberrisk.RiskScorer
	snapshots   *cyberrisk.SnapshotService
	historyRepo *repository.RiskHistoryRepository
	vulnRepo    *repository.VulnerabilityRepository
	producer    *events.Producer
	logger      zerolog.Logger
}

func NewRiskService(
	scorer *cyberrisk.RiskScorer,
	snapshots *cyberrisk.SnapshotService,
	historyRepo *repository.RiskHistoryRepository,
	vulnRepo *repository.VulnerabilityRepository,
	producer *events.Producer,
	logger zerolog.Logger,
) *RiskService {
	return &RiskService{
		scorer:      scorer,
		snapshots:   snapshots,
		historyRepo: historyRepo,
		vulnRepo:    vulnRepo,
		producer:    producer,
		logger:      logger.With().Str("service", "risk").Logger(),
	}
}

func (s *RiskService) GetCurrentScore(ctx context.Context, tenantID uuid.UUID) (*model.OrganizationRiskScore, error) {
	return s.scorer.CalculateOrganizationRisk(ctx, tenantID)
}

func (s *RiskService) Recalculate(ctx context.Context, tenantID uuid.UUID, actor *Actor) (*model.OrganizationRiskScore, error) {
	previous, _ := s.historyRepo.Latest(ctx, tenantID)
	score, err := s.snapshots.SaveOnDemandSnapshot(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	s.publishScoreEvents(ctx, tenantID, actor, previous, score)
	return score, nil
}

func (s *RiskService) SaveEventTriggeredSnapshot(ctx context.Context, tenantID uuid.UUID, eventType string) (*model.OrganizationRiskScore, error) {
	previous, _ := s.historyRepo.Latest(ctx, tenantID)
	score, err := s.snapshots.SaveEventTriggeredSnapshot(ctx, tenantID, eventType)
	if err != nil {
		return nil, err
	}
	s.publishScoreEvents(ctx, tenantID, nil, previous, score)
	return score, nil
}

func (s *RiskService) Trend(ctx context.Context, tenantID uuid.UUID, days int) ([]model.RiskTrendPoint, error) {
	return s.historyRepo.Trend(ctx, tenantID, days)
}

func (s *RiskService) Heatmap(ctx context.Context, tenantID uuid.UUID) (*model.RiskHeatmap, error) {
	return s.vulnRepo.RiskHeatmap(ctx, tenantID)
}

func (s *RiskService) TopRisks(ctx context.Context, tenantID uuid.UUID) ([]model.RiskContributor, error) {
	score, err := s.scorer.CalculateOrganizationRisk(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return score.TopContributors, nil
}

func (s *RiskService) Recommendations(ctx context.Context, tenantID uuid.UUID) ([]model.RiskRecommendation, error) {
	score, err := s.scorer.CalculateOrganizationRisk(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return score.Recommendations, nil
}

func (s *RiskService) InvalidateCache(ctx context.Context, tenantID uuid.UUID) error {
	return s.scorer.InvalidateCache(ctx, tenantID)
}

func (s *RiskService) publishScoreEvents(ctx context.Context, tenantID uuid.UUID, actor *Actor, previous *model.RiskScoreHistory, current *model.OrganizationRiskScore) {
	if current == nil {
		return
	}
	previousScore := 0.0
	previousGrade := ""
	if previous != nil {
		previousScore = previous.OverallScore
		previousGrade = previous.Grade
	}
	payload := map[string]interface{}{
		"tenant_id":      tenantID.String(),
		"score":          current.OverallScore,
		"grade":          current.Grade,
		"previous_score": previousScore,
		"delta":          current.OverallScore - previousScore,
	}
	if err := publishEvent(ctx, s.producer, events.Topics.RiskEvents, "cyber.risk.score_calculated", tenantID, actor, payload); err != nil {
		s.logger.Error().Err(err).Str("tenant_id", tenantID.String()).Msg("publish risk score event")
	}
	if previousGrade != "" && previousGrade != current.Grade {
		if err := publishEvent(ctx, s.producer, events.Topics.RiskEvents, "cyber.risk.grade_changed", tenantID, actor, map[string]interface{}{
			"tenant_id": tenantID.String(),
			"old_grade": previousGrade,
			"new_grade": current.Grade,
			"score":     current.OverallScore,
		}); err != nil {
			s.logger.Error().Err(err).Str("tenant_id", tenantID.String()).Msg("publish risk grade change event")
		}
	}
}

func (s *RiskService) LatestTrendSafe(ctx context.Context, tenantID uuid.UUID, days int) []model.RiskTrendPoint {
	trend, err := s.Trend(ctx, tenantID, days)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		s.logger.Error().Err(err).Str("tenant_id", tenantID.String()).Msg("load risk trend")
	}
	return trend
}

func (s *RiskService) String() string {
	return fmt.Sprintf("RiskService(%s)", s.logger.GetLevel().String())
}
