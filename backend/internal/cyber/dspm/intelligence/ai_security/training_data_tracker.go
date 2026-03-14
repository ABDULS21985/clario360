package ai_security

import (
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// trainingMetadataKeys maps metadata keys to whether they indicate training usage.
var trainingMetadataKeys = []string{
	"ml_training",
	"training_dataset",
	"model_training_source",
	"training_data",
	"ml_training_data",
	"training_corpus",
	"fine_tuning_data",
	"calibration_data",
}

// TrainingDataTracker detects when data assets are being used for AI/ML
// model training, fine-tuning, or calibration.
type TrainingDataTracker struct {
	logger zerolog.Logger
}

// NewTrainingDataTracker creates a new tracker instance.
func NewTrainingDataTracker(logger zerolog.Logger) *TrainingDataTracker {
	return &TrainingDataTracker{
		logger: logger.With().Str("component", "training_data_tracker").Logger(),
	}
}

// DetectTrainingUsage examines an asset's metadata and characteristics to
// determine if it is being used for AI/ML training. Returns nil if no training
// usage is detected.
func (t *TrainingDataTracker) DetectTrainingUsage(asset *cybermodel.DSPMDataAsset) *model.AIDataUsage {
	if asset == nil || asset.Metadata == nil {
		return nil
	}

	// Check direct metadata keys indicating training usage.
	for _, key := range trainingMetadataKeys {
		if val, ok := asset.Metadata[key]; ok {
			if boolVal, isBool := val.(bool); isBool && !boolVal {
				continue
			}
			t.logger.Debug().
				Str("asset_id", asset.AssetID.String()).
				Str("key", key).
				Msg("detected training data usage via metadata key")
			return t.buildTrainingUsage(asset, model.AIUsageTrainingData)
		}
	}

	// Check ai_usage metadata field.
	if usageVal, ok := asset.Metadata["ai_usage"]; ok {
		if str, isStr := usageVal.(string); isStr {
			lower := strings.ToLower(str)
			if strings.Contains(lower, "training") || strings.Contains(lower, "fine_tuning") ||
				strings.Contains(lower, "fine-tuning") || strings.Contains(lower, "calibration") {
				t.logger.Debug().
					Str("asset_id", asset.AssetID.String()).
					Str("ai_usage", str).
					Msg("detected training data usage via ai_usage metadata")
				return t.buildTrainingUsage(asset, model.AIUsageTrainingData)
			}
		}
	}

	// Check ai_tags list.
	if tags, ok := asset.Metadata["ai_tags"]; ok {
		if tagList, isList := tags.([]interface{}); isList {
			for _, tag := range tagList {
				tagStr, isStr := tag.(string)
				if !isStr {
					continue
				}
				lower := strings.ToLower(tagStr)
				if lower == "training_data" || lower == "training_dataset" ||
					lower == "fine_tuning_data" || lower == "calibration_data" ||
					lower == "ml_training" {
					t.logger.Debug().
						Str("asset_id", asset.AssetID.String()).
						Str("tag", tagStr).
						Msg("detected training data usage via ai_tags")
					return t.buildTrainingUsage(asset, model.AIUsageTrainingData)
				}
			}
		}
	}

	// Check for evaluation data (a related but distinct training use case).
	if val, ok := asset.Metadata["evaluation_data"]; ok {
		if boolVal, isBool := val.(bool); !isBool || boolVal {
			return t.buildTrainingUsage(asset, model.AIUsageEvaluationData)
		}
	}
	if val, ok := asset.Metadata["eval_dataset"]; ok {
		if boolVal, isBool := val.(bool); !isBool || boolVal {
			return t.buildTrainingUsage(asset, model.AIUsageEvaluationData)
		}
	}

	// Check for feature store usage.
	if val, ok := asset.Metadata["feature_store"]; ok {
		if boolVal, isBool := val.(bool); !isBool || boolVal {
			return t.buildTrainingUsage(asset, model.AIUsageFeatureStore)
		}
	}
	if val, ok := asset.Metadata["ml_feature_store"]; ok {
		if boolVal, isBool := val.(bool); !isBool || boolVal {
			return t.buildTrainingUsage(asset, model.AIUsageFeatureStore)
		}
	}

	return nil
}

// buildTrainingUsage constructs an AIDataUsage record for training-related usage.
func (t *TrainingDataTracker) buildTrainingUsage(asset *cybermodel.DSPMDataAsset, usageType model.AIUsageType) *model.AIDataUsage {
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

	// Extract model and pipeline metadata.
	if modelName := metadataString(asset.Metadata, "model_name"); modelName != "" {
		usage.ModelName = modelName
	}
	if modelSlug := metadataString(asset.Metadata, "model_slug"); modelSlug != "" {
		usage.ModelSlug = modelSlug
	}
	if modelIDStr := metadataString(asset.Metadata, "model_id"); modelIDStr != "" {
		if id, err := uuid.Parse(modelIDStr); err == nil {
			usage.ModelID = &id
		}
	}
	if pipelineID := metadataString(asset.Metadata, "pipeline_id"); pipelineID != "" {
		usage.PipelineID = pipelineID
	}
	if pipelineName := metadataString(asset.Metadata, "pipeline_name"); pipelineName != "" {
		usage.PipelineName = pipelineName
	}

	// Check consent and anonymization metadata.
	usage.ConsentVerified = metadataBool(asset.Metadata, "consent_verified")
	usage.DataMinimization = metadataBool(asset.Metadata, "data_minimization")
	usage.RetentionCompliant = metadataBool(asset.Metadata, "retention_compliant")
	usage.AnonymizationLevel = extractAnonymizationLevel(asset.Metadata)

	return usage
}
