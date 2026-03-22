package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/clario360/platform/internal/aigovernance/repository"
	"github.com/clario360/platform/internal/events"
)

type ShadowService struct {
	registryRepo   *repository.ModelRegistryRepository
	comparisonRepo *repository.ShadowComparisonRepository
	predictionRepo *repository.PredictionLogRepository
	producer       *events.Producer
	metrics        *Metrics
	logger         zerolog.Logger
}

func NewShadowService(registryRepo *repository.ModelRegistryRepository, comparisonRepo *repository.ShadowComparisonRepository, predictionRepo *repository.PredictionLogRepository, producer *events.Producer, metrics *Metrics, logger zerolog.Logger) *ShadowService {
	return &ShadowService{
		registryRepo:   registryRepo,
		comparisonRepo: comparisonRepo,
		predictionRepo: predictionRepo,
		producer:       producer,
		metrics:        metrics,
		logger:         logger.With().Str("component", "ai_shadow_service").Logger(),
	}
}

func (s *ShadowService) Start(ctx context.Context, tenantID, modelID, versionID uuid.UUID, promotedBy *uuid.UUID) (*aigovmodel.ModelVersion, error) {
	version, err := s.registryRepo.GetVersion(ctx, tenantID, modelID, versionID)
	if err != nil {
		return nil, err
	}
	production, err := s.registryRepo.GetCurrentProductionVersion(ctx, tenantID, modelID)
	if err != nil {
		return nil, fmt.Errorf("shadow mode requires an active production version")
	}
	if existing, err := s.registryRepo.GetCurrentShadowVersion(ctx, tenantID, modelID); err == nil && existing.ID != version.ID {
		existing.Status = aigovmodel.VersionStatusStaging
		existing.UpdatedAt = time.Now().UTC()
		if updateErr := s.registryRepo.UpdateVersionStatus(ctx, existing); updateErr != nil {
			return nil, updateErr
		}
	}
	now := time.Now().UTC()
	version.Status = aigovmodel.VersionStatusShadow
	version.PromotedToShadowAt = &now
	version.PromotedBy = promotedBy
	version.UpdatedAt = now
	if err := s.registryRepo.UpdateVersionStatus(ctx, version); err != nil {
		return nil, err
	}
	s.publish(ctx, "com.clario360.ai.shadow.started", tenantID, map[string]any{
		"model_id":              version.ModelID,
		"model_slug":            version.ModelSlug,
		"shadow_version_id":     version.ID,
		"production_version_id": production.ID,
	})
	return version, nil
}

func (s *ShadowService) Stop(ctx context.Context, tenantID, modelID, versionID uuid.UUID, reason string) (*aigovmodel.ModelVersion, error) {
	version, err := s.registryRepo.GetVersion(ctx, tenantID, modelID, versionID)
	if err != nil {
		return nil, err
	}
	if version.Status != aigovmodel.VersionStatusShadow {
		return nil, fmt.Errorf("version %d is not in shadow mode", version.VersionNumber)
	}
	version.Status = aigovmodel.VersionStatusStaging
	version.UpdatedAt = time.Now().UTC()
	if err := s.registryRepo.UpdateVersionStatus(ctx, version); err != nil {
		return nil, err
	}
	s.publish(ctx, "com.clario360.ai.shadow.stopped", tenantID, map[string]any{
		"model_id":          version.ModelID,
		"model_slug":        version.ModelSlug,
		"shadow_version_id": version.ID,
		"reason":            strings.TrimSpace(reason),
	})
	return version, nil
}

func (s *ShadowService) LatestComparison(ctx context.Context, tenantID, modelID uuid.UUID) (*aigovmodel.ShadowComparison, error) {
	return s.comparisonRepo.LatestByModel(ctx, tenantID, modelID)
}

func (s *ShadowService) ComparisonHistory(ctx context.Context, tenantID, modelID uuid.UUID, limit int) ([]aigovmodel.ShadowComparison, error) {
	return s.comparisonRepo.History(ctx, tenantID, modelID, limit)
}

func (s *ShadowService) Divergences(ctx context.Context, tenantID, modelID uuid.UUID, page, perPage int) ([]aigovmodel.ShadowDivergence, int, error) {
	logs, total, err := s.predictionRepo.ListDivergences(ctx, tenantID, modelID, page, perPage)
	if err != nil {
		return nil, 0, err
	}
	out := make([]aigovmodel.ShadowDivergence, 0, len(logs))
	for i := range logs {
		out = append(out, predictionLogToDivergence(&logs[i]))
	}
	return out, total, nil
}

// predictionLogToDivergence converts a shadow PredictionLog (which has a
// non-null shadow_divergence JSONB) into the ShadowDivergence struct the
// frontend expects.
func predictionLogToDivergence(log *aigovmodel.PredictionLog) aigovmodel.ShadowDivergence {
	d := aigovmodel.ShadowDivergence{
		PredictionID: log.ID,
		InputHash:    log.InputHash,
		UseCase:      log.UseCase,
		EntityID:     log.EntityID,
		ShadowOutput: log.Prediction,
		ShadowConfidence: log.Confidence,
		CreatedAt:    log.CreatedAt,
	}
	// The shadow_divergence JSONB column may carry production-side data
	// and a human-readable reason produced by the comparison service.
	if len(log.ShadowDivergence) > 0 {
		var embedded struct {
			ProductionOutput     json.RawMessage `json:"production_output"`
			ProductionConfidence *float64        `json:"production_confidence"`
			Reason               string          `json:"reason"`
		}
		if err := json.Unmarshal(log.ShadowDivergence, &embedded); err == nil {
			d.ProductionOutput = embedded.ProductionOutput
			d.ProductionConfidence = embedded.ProductionConfidence
			if embedded.Reason != "" {
				d.Reason = embedded.Reason
			}
		}
	}
	// Fall back to a generic reason when comparison service didn't populate one.
	if d.Reason == "" {
		d.Reason = "Shadow prediction diverged from production"
	}
	return d
}

func (s *ShadowService) publish(ctx context.Context, eventType string, tenantID uuid.UUID, payload map[string]any) {
	if s.producer == nil {
		return
	}
	event, err := events.NewEvent(eventType, "iam-service", tenantID.String(), payload)
	if err != nil {
		s.logger.Warn().Err(err).Str("event_type", eventType).Msg("failed to build ai shadow lifecycle event")
		return
	}
	if err := s.producer.Publish(ctx, events.Topics.AIEvents, event); err != nil {
		s.logger.Warn().Err(err).Str("event_type", eventType).Msg("failed to publish ai shadow lifecycle event")
	}
}
