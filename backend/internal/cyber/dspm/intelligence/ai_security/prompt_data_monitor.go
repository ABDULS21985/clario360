package ai_security

import (
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// promptMetadataKeys maps metadata keys to prompt/RAG-related AI usage types.
var promptMetadataKeys = map[string]model.AIUsageType{
	"rag_source":         model.AIUsageRAGKnowledgeBase,
	"rag_knowledge_base": model.AIUsageRAGKnowledgeBase,
	"rag_corpus":         model.AIUsageRAGKnowledgeBase,
	"rag_index":          model.AIUsageRAGKnowledgeBase,
	"vector_store":       model.AIUsageRAGKnowledgeBase,
	"knowledge_base":     model.AIUsageRAGKnowledgeBase,
	"prompt_context":     model.AIUsagePromptContext,
	"prompt_template":    model.AIUsagePromptContext,
	"system_prompt_data": model.AIUsagePromptContext,
	"context_window":     model.AIUsagePromptContext,
	"embedding_source":   model.AIUsageEmbeddingSource,
	"embedding_index":    model.AIUsageEmbeddingSource,
	"embeddings":         model.AIUsageEmbeddingSource,
	"inference_input":    model.AIUsageInferenceInput,
	"inference_data":     model.AIUsageInferenceInput,
}

// PromptDataMonitor detects when data assets are used as RAG knowledge bases,
// prompt context sources, embedding sources, or inference inputs.
type PromptDataMonitor struct {
	logger zerolog.Logger
}

// NewPromptDataMonitor creates a new prompt data monitor instance.
func NewPromptDataMonitor(logger zerolog.Logger) *PromptDataMonitor {
	return &PromptDataMonitor{
		logger: logger.With().Str("component", "prompt_data_monitor").Logger(),
	}
}

// DetectPromptUsage examines an asset's metadata for RAG, prompt context,
// or embedding source usage patterns. Returns nil if no such usage is detected.
func (m *PromptDataMonitor) DetectPromptUsage(asset *cybermodel.DSPMDataAsset) *model.AIDataUsage {
	if asset == nil || asset.Metadata == nil {
		return nil
	}

	// Check direct metadata keys for prompt/RAG usage.
	for key, usageType := range promptMetadataKeys {
		if val, ok := asset.Metadata[key]; ok {
			if boolVal, isBool := val.(bool); isBool && !boolVal {
				continue
			}
			m.logger.Debug().
				Str("asset_id", asset.AssetID.String()).
				Str("key", key).
				Str("usage_type", string(usageType)).
				Msg("detected prompt/RAG usage via metadata key")
			return m.buildPromptUsage(asset, usageType)
		}
	}

	// Check ai_usage metadata field for prompt/RAG indicators.
	if usageVal, ok := asset.Metadata["ai_usage"]; ok {
		if str, isStr := usageVal.(string); isStr {
			lower := strings.ToLower(str)
			switch {
			case strings.Contains(lower, "rag") || strings.Contains(lower, "knowledge_base") ||
				strings.Contains(lower, "knowledge-base"):
				return m.buildPromptUsage(asset, model.AIUsageRAGKnowledgeBase)
			case strings.Contains(lower, "prompt") || strings.Contains(lower, "context"):
				return m.buildPromptUsage(asset, model.AIUsagePromptContext)
			case strings.Contains(lower, "embedding"):
				return m.buildPromptUsage(asset, model.AIUsageEmbeddingSource)
			case strings.Contains(lower, "inference"):
				return m.buildPromptUsage(asset, model.AIUsageInferenceInput)
			}
		}
	}

	// Check ai_tags list for prompt/RAG indicators.
	if tags, ok := asset.Metadata["ai_tags"]; ok {
		if tagList, isList := tags.([]interface{}); isList {
			for _, tag := range tagList {
				tagStr, isStr := tag.(string)
				if !isStr {
					continue
				}
				tagLower := strings.ToLower(tagStr)
				if usageType, found := promptMetadataKeys[tagLower]; found {
					m.logger.Debug().
						Str("asset_id", asset.AssetID.String()).
						Str("tag", tagStr).
						Msg("detected prompt/RAG usage via ai_tags")
					return m.buildPromptUsage(asset, usageType)
				}
			}
		}
	}

	return nil
}

// buildPromptUsage constructs an AIDataUsage record for prompt/RAG usage.
func (m *PromptDataMonitor) buildPromptUsage(asset *cybermodel.DSPMDataAsset, usageType model.AIUsageType) *model.AIDataUsage {
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

	usage.ConsentVerified = metadataBool(asset.Metadata, "consent_verified")
	usage.DataMinimization = metadataBool(asset.Metadata, "data_minimization")
	usage.RetentionCompliant = metadataBool(asset.Metadata, "retention_compliant")
	usage.AnonymizationLevel = extractAnonymizationLevel(asset.Metadata)

	return usage
}
