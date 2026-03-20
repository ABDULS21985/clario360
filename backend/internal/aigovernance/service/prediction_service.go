package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	aigovdto "github.com/clario360/platform/internal/aigovernance/dto"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/clario360/platform/internal/aigovernance/repository"
	"github.com/clario360/platform/internal/events"
)

type PredictionService struct {
	repo         *repository.PredictionLogRepository
	registryRepo *repository.ModelRegistryRepository
	producer     *events.Producer
	metrics      *Metrics
	logger       zerolog.Logger
}

func NewPredictionService(repo *repository.PredictionLogRepository, registryRepo *repository.ModelRegistryRepository, producer *events.Producer, metrics *Metrics, logger zerolog.Logger) *PredictionService {
	return &PredictionService{
		repo:         repo,
		registryRepo: registryRepo,
		producer:     producer,
		metrics:      metrics,
		logger:       logger.With().Str("component", "ai_prediction_service").Logger(),
	}
}

func (s *PredictionService) List(ctx context.Context, tenantID uuid.UUID, params aigovdto.PredictionQuery) ([]aigovmodel.PredictionLog, int, error) {
	return s.repo.List(ctx, tenantID, params)
}

func (s *PredictionService) Get(ctx context.Context, tenantID, predictionID uuid.UUID) (*aigovmodel.PredictionLog, error) {
	return s.repo.Get(ctx, tenantID, predictionID)
}

func (s *PredictionService) SubmitFeedback(ctx context.Context, tenantID, userID, predictionID uuid.UUID, req aigovdto.PredictionFeedbackRequest) error {
	entry, err := s.repo.Get(ctx, tenantID, predictionID)
	if err != nil {
		return err
	}
	if err := s.repo.SubmitFeedback(ctx, tenantID, predictionID, userID, req); err != nil {
		return err
	}
	if err := s.registryRepo.UpdateVersionAggregates(ctx, tenantID, entry.ModelVersionID); err != nil {
		s.logger.Warn().Err(err).Str("version_id", entry.ModelVersionID.String()).Msg("failed to refresh version aggregates after feedback")
	}
	if s.metrics != nil {
		s.metrics.PredictionFeedbackTotal.WithLabelValues(entry.ModelSlug, fmt.Sprintf("%t", req.Correct)).Inc()
	}
	s.publish(ctx, "com.clario360.ai.prediction.feedback_received", tenantID, map[string]any{
		"prediction_id": predictionID,
		"model_id":      entry.ModelID,
		"correct":       req.Correct,
	})
	return nil
}

func (s *PredictionService) Stats(ctx context.Context, tenantID uuid.UUID) ([]aigovmodel.PredictionStats, error) {
	return s.repo.Stats(ctx, tenantID)
}

func (s *PredictionService) SearchExplanations(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]aigovmodel.ExplanationSearchResult, error) {
	return s.repo.SearchExplanations(ctx, tenantID, query, limit)
}

func (s *PredictionService) PerformanceSeries(ctx context.Context, tenantID, modelID uuid.UUID, period string) ([]aigovmodel.PerformancePoint, error) {
	return s.repo.PerformanceSeries(ctx, tenantID, modelID, sinceForPeriod(period))
}

func (s *PredictionService) publish(ctx context.Context, eventType string, tenantID uuid.UUID, payload map[string]any) {
	if s.producer == nil {
		return
	}
	event, err := events.NewEvent(eventType, "iam-service", tenantID.String(), payload)
	if err != nil {
		s.logger.Warn().Err(err).Str("event_type", eventType).Msg("failed to build ai prediction event")
		return
	}
	if err := s.producer.Publish(ctx, events.Topics.AIEvents, event); err != nil {
		s.logger.Warn().Err(err).Str("event_type", eventType).Msg("failed to publish ai prediction event")
	}
}

func sinceForPeriod(period string) time.Time {
	now := time.Now().UTC()
	switch period {
	case "7d":
		return now.AddDate(0, 0, -7)
	case "90d":
		return now.AddDate(0, 0, -90)
	default:
		return now.AddDate(0, 0, -30)
	}
}
