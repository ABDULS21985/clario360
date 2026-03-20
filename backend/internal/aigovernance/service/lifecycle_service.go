package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/clario360/platform/internal/aigovernance/repository"
	"github.com/clario360/platform/internal/events"
)

type LifecycleService struct {
	repo           *repository.ModelRegistryRepository
	comparisonRepo *repository.ShadowComparisonRepository
	producer       *events.Producer
	metrics        *Metrics
	logger         zerolog.Logger
}

func NewLifecycleService(repo *repository.ModelRegistryRepository, comparisonRepo *repository.ShadowComparisonRepository, producer *events.Producer, metrics *Metrics, logger zerolog.Logger) *LifecycleService {
	return &LifecycleService{
		repo:           repo,
		comparisonRepo: comparisonRepo,
		producer:       producer,
		metrics:        metrics,
		logger:         logger.With().Str("component", "ai_lifecycle_service").Logger(),
	}
}

func (s *LifecycleService) Promote(ctx context.Context, tenantID, modelID, versionID uuid.UUID, approvedBy *uuid.UUID, override bool) (*aigovmodel.ModelVersion, error) {
	version, err := s.repo.GetVersion(ctx, tenantID, modelID, versionID)
	if err != nil {
		return nil, err
	}
	from := version.Status
	target, err := s.nextStatus(ctx, version, approvedBy, override)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	switch target {
	case aigovmodel.VersionStatusStaging:
		version.PromotedToStagingAt = &now
	case aigovmodel.VersionStatusShadow:
		version.PromotedToShadowAt = &now
	case aigovmodel.VersionStatusProduction:
		currentProd, prodErr := s.repo.GetCurrentProductionVersion(ctx, tenantID, modelID)
		if prodErr == nil && currentProd.ID != version.ID {
			reason := fmt.Sprintf("replaced by v%d", version.VersionNumber)
			currentProd.Status = aigovmodel.VersionStatusRetired
			currentProd.RetiredAt = &now
			currentProd.RetiredBy = approvedBy
			currentProd.RetirementReason = &reason
			currentProd.UpdatedAt = now
			if err := s.repo.UpdateVersionStatus(ctx, currentProd); err != nil {
				return nil, err
			}
			version.ReplacedVersionID = &currentProd.ID
		}
		version.PromotedToProductionAt = &now
		version.PromotedBy = approvedBy
	}
	version.Status = target
	version.PromotedBy = approvedBy
	version.UpdatedAt = now
	if err := s.repo.UpdateVersionStatus(ctx, version); err != nil {
		return nil, err
	}
	if s.metrics != nil {
		s.metrics.LifecyclePromotionsTotal.WithLabelValues(version.ModelSlug, string(target)).Inc()
	}
	s.publish(ctx, "com.clario360.ai.model.version.promoted", tenantID, map[string]any{
		"model_id":    modelID,
		"model_slug":  version.ModelSlug,
		"version_id":  versionID,
		"from_status": from,
		"to_status":   target,
		"promoted_by": approvedBy,
	})
	return version, nil
}

func (s *LifecycleService) Retire(ctx context.Context, tenantID, modelID, versionID, retiredBy uuid.UUID, reason string) (*aigovmodel.ModelVersion, error) {
	version, err := s.repo.GetVersion(ctx, tenantID, modelID, versionID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	version.Status = aigovmodel.VersionStatusRetired
	version.RetiredAt = &now
	version.RetiredBy = &retiredBy
	reason = strings.TrimSpace(reason)
	version.RetirementReason = &reason
	version.UpdatedAt = now
	if err := s.repo.UpdateVersionStatus(ctx, version); err != nil {
		return nil, err
	}
	s.publish(ctx, "com.clario360.ai.model.version.retired", tenantID, map[string]any{
		"model_id":   modelID,
		"model_slug": version.ModelSlug,
		"version_id": versionID,
		"reason":     reason,
	})
	return version, nil
}

func (s *LifecycleService) Fail(ctx context.Context, tenantID, modelID, versionID, failedBy uuid.UUID, reason string) (*aigovmodel.ModelVersion, error) {
	version, err := s.repo.GetVersion(ctx, tenantID, modelID, versionID)
	if err != nil {
		return nil, err
	}
	if err := validateFailureTransition(version); err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, fmt.Errorf("reason is required")
	}
	from := version.Status
	now := time.Now().UTC()
	version.Status = aigovmodel.VersionStatusFailed
	version.FailedAt = &now
	version.FailedBy = &failedBy
	version.FailedFromStatus = versionStatusPtr(from)
	version.FailureReason = &reason
	version.UpdatedAt = now
	if err := s.repo.UpdateVersionStatus(ctx, version); err != nil {
		return nil, err
	}
	s.publish(ctx, "com.clario360.ai.model.version.failed", tenantID, map[string]any{
		"model_id":     modelID,
		"model_slug":   version.ModelSlug,
		"version_id":   versionID,
		"from_status":  from,
		"to_status":    version.Status,
		"failed_by":    failedBy,
		"failure_note": reason,
	})
	return version, nil
}

func (s *LifecycleService) Rollback(ctx context.Context, tenantID, modelID, rolledBackBy uuid.UUID, reason string) (*aigovmodel.ModelVersion, error) {
	current, err := s.repo.GetCurrentProductionVersion(ctx, tenantID, modelID)
	if err != nil {
		return nil, err
	}
	if current.ReplacedVersionID == nil {
		return nil, fmt.Errorf("no previous version to rollback to")
	}
	previous, err := s.repo.GetVersionByID(ctx, tenantID, *current.ReplacedVersionID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	reason = strings.TrimSpace(reason)
	current.Status = aigovmodel.VersionStatusRolledBack
	current.RolledBackAt = &now
	current.RolledBackBy = &rolledBackBy
	current.RollbackReason = &reason
	current.UpdatedAt = now
	if err := s.repo.UpdateVersionStatus(ctx, current); err != nil {
		return nil, err
	}
	previous.Status = aigovmodel.VersionStatusProduction
	previous.PromotedToProductionAt = &now
	previous.PromotedBy = &rolledBackBy
	previous.UpdatedAt = now
	if err := s.repo.UpdateVersionStatus(ctx, previous); err != nil {
		return nil, err
	}
	if shadowVersion, err := s.repo.GetCurrentShadowVersion(ctx, tenantID, modelID); err == nil {
		shadowVersion.Status = aigovmodel.VersionStatusStaging
		shadowVersion.UpdatedAt = now
		if err := s.repo.UpdateVersionStatus(ctx, shadowVersion); err != nil {
			return nil, err
		}
	}
	if s.metrics != nil {
		s.metrics.LifecycleRollbacksTotal.WithLabelValues(previous.ModelSlug).Inc()
	}
	s.publish(ctx, "com.clario360.ai.model.version.rolled_back", tenantID, map[string]any{
		"model_id":       modelID,
		"model_slug":     previous.ModelSlug,
		"version_id":     current.ID,
		"rolled_back_to": previous.ID,
		"reason":         reason,
	})
	return previous, nil
}

func (s *LifecycleService) History(ctx context.Context, tenantID, modelID uuid.UUID) ([]aigovmodel.LifecycleHistoryEntry, error) {
	return s.repo.LifecycleHistory(ctx, tenantID, modelID)
}

func (s *LifecycleService) nextStatus(ctx context.Context, version *aigovmodel.ModelVersion, approvedBy *uuid.UUID, override bool) (aigovmodel.VersionStatus, error) {
	switch version.Status {
	case aigovmodel.VersionStatusDevelopment:
		if err := validateArtifact(version); err != nil {
			return "", err
		}
		return aigovmodel.VersionStatusStaging, nil
	case aigovmodel.VersionStatusStaging:
		if _, err := s.repo.GetCurrentProductionVersion(ctx, version.TenantID, version.ModelID); err == repository.ErrNotFound {
			return aigovmodel.VersionStatusProduction, nil
		}
		return aigovmodel.VersionStatusShadow, nil
	case aigovmodel.VersionStatusShadow:
		comparison, err := s.comparisonRepo.LatestByShadowVersion(ctx, version.TenantID, version.ID)
		if err != nil {
			return "", fmt.Errorf("shadow mode has not produced comparison results yet")
		}
		if comparison.Recommendation == aigovmodel.ShadowRecommendationReject {
			return "", fmt.Errorf("shadow comparison recommends rejection: %s", comparison.RecommendationReason)
		}
		if version.ModelRiskTier == aigovmodel.RiskTierCritical && comparison.Recommendation == aigovmodel.ShadowRecommendationNeedsReview && approvedBy == nil {
			return "", fmt.Errorf("critical-tier model requires manual approval for promotion")
		}
		if comparison.Recommendation == aigovmodel.ShadowRecommendationKeepShadow && !override && approvedBy == nil {
			return "", fmt.Errorf("shadow comparison recommends keep_shadow; manual override is required")
		}
		return aigovmodel.VersionStatusProduction, nil
	default:
		return "", fmt.Errorf("invalid transition from %s", version.Status)
	}
}

func validateFailureTransition(version *aigovmodel.ModelVersion) error {
	switch version.Status {
	case aigovmodel.VersionStatusDevelopment, aigovmodel.VersionStatusStaging, aigovmodel.VersionStatusShadow:
		return nil
	case aigovmodel.VersionStatusProduction:
		return fmt.Errorf("invalid transition: cannot mark a production version as failed; rollback or retire it instead")
	case aigovmodel.VersionStatusFailed:
		return fmt.Errorf("invalid transition: version %d is already failed", version.VersionNumber)
	default:
		return fmt.Errorf("invalid transition: version %d cannot be marked failed from %s", version.VersionNumber, version.Status)
	}
}

func validateArtifact(version *aigovmodel.ModelVersion) error {
	if len(version.ArtifactConfig) == 0 || string(version.ArtifactConfig) == "null" {
		return fmt.Errorf("artifact_config must not be empty")
	}
	if strings.TrimSpace(version.Description) == "" {
		return fmt.Errorf("description must not be empty")
	}
	var decoded any
	if err := json.Unmarshal(version.ArtifactConfig, &decoded); err != nil {
		return fmt.Errorf("artifact_config must be valid JSON: %w", err)
	}
	hash, err := aigovernance.HashJSON(decoded)
	if err != nil {
		return fmt.Errorf("compute artifact hash: %w", err)
	}
	if hash != version.ArtifactHash {
		return fmt.Errorf("artifact_hash does not match artifact_config")
	}
	return nil
}

func (s *LifecycleService) publish(ctx context.Context, eventType string, tenantID uuid.UUID, payload map[string]any) {
	if s.producer == nil {
		return
	}
	event, err := events.NewEvent(eventType, "iam-service", tenantID.String(), payload)
	if err != nil {
		s.logger.Warn().Err(err).Str("event_type", eventType).Msg("failed to build ai lifecycle event")
		return
	}
	if err := s.producer.Publish(ctx, events.Topics.AIEvents, event); err != nil {
		s.logger.Warn().Err(err).Str("event_type", eventType).Msg("failed to publish ai lifecycle event")
	}
}

func versionStatusPtr(status aigovmodel.VersionStatus) *aigovmodel.VersionStatus {
	value := status
	return &value
}
