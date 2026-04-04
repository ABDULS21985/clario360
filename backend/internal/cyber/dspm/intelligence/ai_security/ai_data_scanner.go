package ai_security

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/dto"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// aiMetadataKeys are metadata keys that indicate AI/ML usage.
var aiMetadataKeys = map[string]model.AIUsageType{
	"training_data":        model.AIUsageTrainingData,
	"training_dataset":     model.AIUsageTrainingData,
	"ml_training":          model.AIUsageTrainingData,
	"model_training_source": model.AIUsageTrainingData,
	"feature_store":        model.AIUsageFeatureStore,
	"ml_pipeline":          model.AIUsageTrainingData,
	"embedding_source":     model.AIUsageEmbeddingSource,
	"rag_source":           model.AIUsageRAGKnowledgeBase,
	"rag_knowledge_base":   model.AIUsageRAGKnowledgeBase,
	"prompt_context":       model.AIUsagePromptContext,
	"inference_input":      model.AIUsageInferenceInput,
	"evaluation_data":      model.AIUsageEvaluationData,
}

// AssetLister retrieves active data assets for a tenant.
type AssetLister interface {
	ListAllActive(ctx context.Context, tenantID uuid.UUID) ([]*cybermodel.DSPMDataAsset, error)
}

// AIUsageRepository persists and queries AI data usage records.
type AIUsageRepository interface {
	Upsert(ctx context.Context, usage *model.AIDataUsage) error
	ListByTenant(ctx context.Context, tenantID uuid.UUID, params *dto.AIUsageListParams) ([]model.AIDataUsage, int, error)
	ListByAsset(ctx context.Context, tenantID, assetID uuid.UUID) ([]model.AIDataUsage, error)
	ListByModel(ctx context.Context, tenantID uuid.UUID, modelSlug string) ([]model.AIDataUsage, error)
	Dashboard(ctx context.Context, tenantID uuid.UUID) (*model.AISecurityDashboard, error)
}

// AIDataScanner discovers and catalogues AI/ML data usage across data assets.
type AIDataScanner struct {
	assets AssetLister
	repo   AIUsageRepository
	logger zerolog.Logger
}

// NewAIDataScanner creates a new AI data scanner instance.
func NewAIDataScanner(assets AssetLister, repo AIUsageRepository, logger zerolog.Logger) *AIDataScanner {
	return &AIDataScanner{
		assets: assets,
		repo:   repo,
		logger: logger.With().Str("component", "ai_data_scanner").Logger(),
	}
}

// Scan discovers AI/ML data usage across all active assets for a tenant.
// It checks asset metadata for AI-related tags and access patterns, calculates
// risk scores, and persists the results.
func (s *AIDataScanner) Scan(ctx context.Context, tenantID uuid.UUID) ([]model.AIDataUsage, error) {
	s.logger.Info().Str("tenant_id", tenantID.String()).Msg("starting AI data scan")

	assets, err := s.assets.ListAllActive(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var usages []model.AIDataUsage
	now := time.Now().UTC()

	trainingTracker := NewTrainingDataTracker(s.logger)
	promptMonitor := NewPromptDataMonitor(s.logger)
	riskAssessor := NewAIRiskAssessor(s.logger)

	for _, asset := range assets {
		// Try training data detection first.
		if usage := trainingTracker.DetectTrainingUsage(asset); usage != nil {
			usage.TenantID = tenantID
			usage.FirstDetectedAt = now
			usage.LastDetectedAt = now
			usage.CreatedAt = now
			usage.UpdatedAt = now

			score, level, factors := riskAssessor.AssessRisk(usage)
			usage.AIRiskScore = score
			usage.AIRiskLevel = level
			usage.RiskFactors = factors

			usages = append(usages, *usage)
			continue
		}

		// Try prompt/RAG data detection.
		if usage := promptMonitor.DetectPromptUsage(asset); usage != nil {
			usage.TenantID = tenantID
			usage.FirstDetectedAt = now
			usage.LastDetectedAt = now
			usage.CreatedAt = now
			usage.UpdatedAt = now

			score, level, factors := riskAssessor.AssessRisk(usage)
			usage.AIRiskScore = score
			usage.AIRiskLevel = level
			usage.RiskFactors = factors

			usages = append(usages, *usage)
			continue
		}

		// Generic metadata-based detection for remaining AI usage types.
		if usage := s.detectFromMetadata(asset); usage != nil {
			usage.TenantID = tenantID
			usage.FirstDetectedAt = now
			usage.LastDetectedAt = now
			usage.CreatedAt = now
			usage.UpdatedAt = now

			score, level, factors := riskAssessor.AssessRisk(usage)
			usage.AIRiskScore = score
			usage.AIRiskLevel = level
			usage.RiskFactors = factors

			usages = append(usages, *usage)
		}
	}

	// Persist all discovered usages.
	for i := range usages {
		if err := s.repo.Upsert(ctx, &usages[i]); err != nil {
			s.logger.Error().Err(err).
				Str("asset_id", usages[i].DataAssetID.String()).
				Msg("failed to persist AI usage record")
		}
	}

	s.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("assets_scanned", len(assets)).
		Int("ai_usages_found", len(usages)).
		Msg("AI data scan complete")

	return usages, nil
}

// detectFromMetadata examines asset metadata for AI-related tags.
func (s *AIDataScanner) detectFromMetadata(asset *cybermodel.DSPMDataAsset) *model.AIDataUsage {
	if asset.Metadata == nil {
		return nil
	}

	// Check metadata keys for AI usage indicators.
	for key, usageType := range aiMetadataKeys {
		if val, ok := asset.Metadata[key]; ok {
			if boolVal, isBool := val.(bool); isBool && !boolVal {
				continue
			}
			return s.buildUsage(asset, usageType)
		}
	}

	// Check for ai_usage or ai_tags metadata.
	if tags, ok := asset.Metadata["ai_tags"]; ok {
		if tagList, isList := tags.([]interface{}); isList {
			for _, tag := range tagList {
				tagStr, isStr := tag.(string)
				if !isStr {
					continue
				}
				tagLower := strings.ToLower(tagStr)
				if usageType, found := aiMetadataKeys[tagLower]; found {
					return s.buildUsage(asset, usageType)
				}
			}
		}
	}

	if usageStr, ok := asset.Metadata["ai_usage"]; ok {
		if str, isStr := usageStr.(string); isStr {
			usageLower := strings.ToLower(str)
			if usageType, found := aiMetadataKeys[usageLower]; found {
				return s.buildUsage(asset, usageType)
			}
			// Default to training data if we see any AI usage tag we don't recognize.
			return s.buildUsage(asset, model.AIUsageTrainingData)
		}
	}

	return nil
}

// buildUsage constructs an AIDataUsage record from an asset and usage type.
func (s *AIDataScanner) buildUsage(asset *cybermodel.DSPMDataAsset, usageType model.AIUsageType) *model.AIDataUsage {
	usage := &model.AIDataUsage{
		ID:                 uuid.New(),
		DataAssetID:        asset.AssetID,
		DataAssetName:      asset.AssetName,
		DataClassification: asset.DataClassification,
		ContainsPII:        asset.ContainsPII,
		PIITypes:           asset.PIITypes,
		UsageType:          usageType,
		Status:             model.AIUsageStatusActive,
	}

	// Extract model info from metadata if available.
	if modelName, ok := asset.Metadata["model_name"]; ok {
		if str, isStr := modelName.(string); isStr {
			usage.ModelName = str
		}
	}
	if modelSlug, ok := asset.Metadata["model_slug"]; ok {
		if str, isStr := modelSlug.(string); isStr {
			usage.ModelSlug = str
		}
	}
	if modelID, ok := asset.Metadata["model_id"]; ok {
		if str, isStr := modelID.(string); isStr {
			if id, err := uuid.Parse(str); err == nil {
				usage.ModelID = &id
			}
		}
	}
	if pipelineID, ok := asset.Metadata["pipeline_id"]; ok {
		if str, isStr := pipelineID.(string); isStr {
			usage.PipelineID = str
		}
	}
	if pipelineName, ok := asset.Metadata["pipeline_name"]; ok {
		if str, isStr := pipelineName.(string); isStr {
			usage.PipelineName = str
		}
	}

	// Check consent and anonymization metadata.
	usage.ConsentVerified = metadataBool(asset.Metadata, "consent_verified")
	usage.DataMinimization = metadataBool(asset.Metadata, "data_minimization")
	usage.RetentionCompliant = metadataBool(asset.Metadata, "retention_compliant")
	usage.AnonymizationLevel = extractAnonymizationLevel(asset.Metadata)

	return usage
}

// classificationWeight returns the risk weight for a data classification level.
func classificationWeight(classification string) float64 {
	switch strings.ToLower(classification) {
	case "restricted":
		return 40
	case "confidential":
		return 25
	case "internal":
		return 10
	case "public":
		return 0
	default:
		return 10
	}
}

// metadataBool extracts a boolean from metadata, defaulting to false.
func metadataBool(md map[string]interface{}, key string) bool {
	if md == nil {
		return false
	}
	if val, ok := md[key]; ok {
		if b, isBool := val.(bool); isBool {
			return b
		}
	}
	return false
}

// metadataString extracts a string from metadata, defaulting to empty.
func metadataString(md map[string]interface{}, key string) string {
	if md == nil {
		return ""
	}
	if val, ok := md[key]; ok {
		if s, isStr := val.(string); isStr {
			return s
		}
	}
	return ""
}

// extractAnonymizationLevel reads the anonymization level from asset metadata.
func extractAnonymizationLevel(md map[string]interface{}) model.AnonymizationLevel {
	if md == nil {
		return model.AnonymizationNone
	}
	val, ok := md["anonymization_level"]
	if !ok {
		return model.AnonymizationNone
	}
	str, isStr := val.(string)
	if !isStr {
		return model.AnonymizationNone
	}
	switch strings.ToLower(str) {
	case "pseudonymized":
		return model.AnonymizationPseudonymized
	case "anonymized":
		return model.AnonymizationAnonymized
	case "differential_privacy":
		return model.AnonymizationDifferentialPrivacy
	default:
		return model.AnonymizationNone
	}
}

// clamp constrains a value to [min, max].
func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
