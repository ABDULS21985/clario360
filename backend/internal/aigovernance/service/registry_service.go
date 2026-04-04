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
	aigovdto "github.com/clario360/platform/internal/aigovernance/dto"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/clario360/platform/internal/aigovernance/repository"
	"github.com/clario360/platform/internal/events"
)

type RegistryService struct {
	repo     *repository.ModelRegistryRepository
	producer *events.Producer
	metrics  *Metrics
	logger   zerolog.Logger
}

func NewRegistryService(repo *repository.ModelRegistryRepository, producer *events.Producer, metrics *Metrics, logger zerolog.Logger) *RegistryService {
	return &RegistryService{
		repo:     repo,
		producer: producer,
		metrics:  metrics,
		logger:   logger.With().Str("component", "ai_registry_service").Logger(),
	}
}

func (s *RegistryService) RegisterModel(ctx context.Context, tenantID, userID uuid.UUID, req aigovdto.RegisterModelRequest) (*aigovmodel.RegisteredModel, error) {
	if err := validateModelRequest(req); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	item := &aigovmodel.RegisteredModel{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        strings.TrimSpace(req.Name),
		Slug:        normalizeSlug(req.Slug),
		Description: strings.TrimSpace(req.Description),
		ModelType:   req.ModelType,
		Suite:       req.Suite,
		OwnerUserID: req.OwnerUserID,
		OwnerTeam:   strings.TrimSpace(req.OwnerTeam),
		RiskTier:    defaultRiskTier(req.RiskTier),
		Status:      aigovmodel.ModelStatusActive,
		Tags:        dedupeStrings(req.Tags),
		Metadata:    defaultJSON(req.Metadata, `{}`),
		CreatedBy:   userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.CreateModel(ctx, item); err != nil {
		return nil, err
	}
	s.refreshMetrics(ctx, tenantID)
	s.publish(ctx, "com.clario360.ai.model.registered", tenantID, map[string]any{
		"model_id": item.ID,
		"name":     item.Name,
		"slug":     item.Slug,
		"suite":    item.Suite,
		"type":     item.ModelType,
	})
	return item, nil
}

func (s *RegistryService) ListModels(ctx context.Context, tenantID uuid.UUID, params repository.ListModelsParams) ([]aigovmodel.ModelWithVersions, int, error) {
	items, total, err := s.repo.ListModels(ctx, tenantID, params)
	if err != nil {
		return nil, 0, err
	}
	out := make([]aigovmodel.ModelWithVersions, 0, len(items))
	for idx := range items {
		modelItem := items[idx]
		entry, err := s.attachVersions(ctx, tenantID, &modelItem)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *entry)
	}
	return out, total, nil
}

func (s *RegistryService) GetModel(ctx context.Context, tenantID, modelID uuid.UUID) (*aigovmodel.ModelWithVersions, error) {
	item, err := s.repo.GetModel(ctx, tenantID, modelID)
	if err != nil {
		return nil, err
	}
	return s.attachVersions(ctx, tenantID, item)
}

func (s *RegistryService) UpdateModel(ctx context.Context, tenantID, modelID uuid.UUID, req aigovdto.UpdateModelRequest) (*aigovmodel.RegisteredModel, error) {
	item, err := s.repo.GetModel(ctx, tenantID, modelID)
	if err != nil {
		return nil, err
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		item.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		item.Description = strings.TrimSpace(*req.Description)
	}
	if req.OwnerUserID != nil {
		item.OwnerUserID = req.OwnerUserID
	}
	if req.OwnerTeam != nil {
		item.OwnerTeam = strings.TrimSpace(*req.OwnerTeam)
	}
	if req.RiskTier != nil {
		item.RiskTier = *req.RiskTier
	}
	if req.Status != nil {
		item.Status = *req.Status
	}
	if req.Tags != nil {
		item.Tags = dedupeStrings(*req.Tags)
	}
	if req.Metadata != nil {
		item.Metadata = defaultJSON(*req.Metadata, `{}`)
	}
	item.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateModelMetadata(ctx, item); err != nil {
		return nil, err
	}
	s.refreshMetrics(ctx, tenantID)
	return item, nil
}

func (s *RegistryService) CreateVersion(ctx context.Context, tenantID, userID, modelID uuid.UUID, req aigovdto.CreateVersionRequest) (*aigovmodel.ModelVersion, error) {
	if strings.TrimSpace(req.Description) == "" {
		return nil, fmt.Errorf("version description is required")
	}
	if len(req.ArtifactConfig) == 0 || string(req.ArtifactConfig) == "null" {
		return nil, fmt.Errorf("artifact_config is required")
	}
	modelItem, err := s.repo.GetModel(ctx, tenantID, modelID)
	if err != nil {
		return nil, err
	}
	versionNumber, err := s.repo.NextVersionNumber(ctx, tenantID, modelID)
	if err != nil {
		return nil, err
	}
	var decoded any
	if err := json.Unmarshal(req.ArtifactConfig, &decoded); err != nil {
		return nil, fmt.Errorf("artifact_config must be valid JSON: %w", err)
	}
	hash, err := aigovernance.HashJSON(decoded)
	if err != nil {
		return nil, fmt.Errorf("compute artifact hash: %w", err)
	}
	now := time.Now().UTC()
	version := &aigovmodel.ModelVersion{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		ModelID:            modelID,
		ModelSlug:          modelItem.Slug,
		ModelName:          modelItem.Name,
		ModelType:          modelItem.ModelType,
		ModelSuite:         modelItem.Suite,
		ModelRiskTier:      modelItem.RiskTier,
		VersionNumber:      versionNumber,
		Status:             aigovmodel.VersionStatusDevelopment,
		Description:        strings.TrimSpace(req.Description),
		ArtifactType:       req.ArtifactType,
		ArtifactConfig:     req.ArtifactConfig,
		ArtifactHash:       hash,
		ExplainabilityType: req.ExplainabilityType,
		ExplanationTemplate: req.ExplanationTemplate,
		TrainingDataDesc:   req.TrainingDataDesc,
		TrainingDataHash:   req.TrainingDataHash,
		TrainingMetrics:    defaultJSON(req.TrainingMetrics, `{}`),
		CreatedBy:          userID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.repo.CreateVersion(ctx, version); err != nil {
		return nil, err
	}
	s.refreshMetrics(ctx, tenantID)
	s.publish(ctx, "com.clario360.ai.model.version.created", tenantID, map[string]any{
		"model_id":       modelID,
		"version_id":     version.ID,
		"version_number": version.VersionNumber,
	})
	return version, nil
}

func (s *RegistryService) ListVersions(ctx context.Context, tenantID, modelID uuid.UUID) ([]aigovmodel.ModelVersion, error) {
	return s.repo.ListVersions(ctx, tenantID, modelID)
}

func (s *RegistryService) GetVersion(ctx context.Context, tenantID, modelID, versionID uuid.UUID) (*aigovmodel.ModelVersion, error) {
	return s.repo.GetVersion(ctx, tenantID, modelID, versionID)
}

func (s *RegistryService) attachVersions(ctx context.Context, tenantID uuid.UUID, item *aigovmodel.RegisteredModel) (*aigovmodel.ModelWithVersions, error) {
	out := &aigovmodel.ModelWithVersions{Model: item}
	production, err := s.repo.GetCurrentProductionVersion(ctx, tenantID, item.ID)
	if err == nil {
		out.ProductionVersion = production
	} else if err != repository.ErrNotFound {
		return nil, err
	}
	shadowVersion, err := s.repo.GetCurrentShadowVersion(ctx, tenantID, item.ID)
	if err == nil {
		out.ShadowVersion = shadowVersion
	} else if err != repository.ErrNotFound {
		return nil, err
	}
	return out, nil
}

func (s *RegistryService) refreshMetrics(ctx context.Context, tenantID uuid.UUID) {
	if s.metrics == nil {
		return
	}
	modelCounts, err := s.repo.CountModelsByStatus(ctx, tenantID)
	if err == nil {
		for status, count := range modelCounts {
			s.metrics.ModelsTotal.WithLabelValues(tenantID.String(), "", status).Set(float64(count))
		}
	}
	versionCounts, err := s.repo.CountVersionsByStatus(ctx, tenantID)
	if err == nil {
		for status, count := range versionCounts {
			s.metrics.ModelVersionsTotal.WithLabelValues(tenantID.String(), status).Set(float64(count))
		}
	}
}

func (s *RegistryService) publish(ctx context.Context, eventType string, tenantID uuid.UUID, payload map[string]any) {
	if s.producer == nil {
		return
	}
	event, err := events.NewEvent(eventType, "iam-service", tenantID.String(), payload)
	if err != nil {
		s.logger.Warn().Err(err).Str("event_type", eventType).Msg("failed to build ai registry event")
		return
	}
	if err := s.producer.Publish(ctx, events.Topics.AIEvents, event); err != nil {
		s.logger.Warn().Err(err).Str("event_type", eventType).Msg("failed to publish ai registry event")
	}
}

func validateModelRequest(req aigovdto.RegisterModelRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(req.Slug) == "" {
		return fmt.Errorf("slug is required")
	}
	if req.ModelType == "" {
		return fmt.Errorf("model_type is required")
	}
	if req.Suite == "" {
		return fmt.Errorf("suite is required")
	}
	return nil
}

func normalizeSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "-")
	return value
}

func defaultRiskTier(value aigovmodel.RiskTier) aigovmodel.RiskTier {
	if value == "" {
		return aigovmodel.RiskTierMedium
	}
	return value
}

func defaultJSON(value json.RawMessage, fallback string) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(fallback)
	}
	return value
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
