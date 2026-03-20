package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/aigovernance"
	"github.com/clario360/platform/internal/aigovernance/explainer"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
)

type ExplanationService struct {
	explainers map[aigovmodel.ExplainabilityType]explainer.Explainer
	logger     zerolog.Logger
}

func NewExplanationService(logger zerolog.Logger) *ExplanationService {
	return &ExplanationService{
		explainers: map[aigovmodel.ExplainabilityType]explainer.Explainer{
			aigovmodel.ExplainabilityRuleTrace:            explainer.NewRuleExplainer(),
			aigovmodel.ExplainabilityFeatureImportance:    explainer.NewFeatureImportanceExplainer(),
			aigovmodel.ExplainabilityStatisticalDeviation: explainer.NewStatisticalExplainer(),
			aigovmodel.ExplainabilityTemplateBased:        explainer.NewTemplateExplainer(),
		},
		logger: logger.With().Str("component", "ai_explanation_service").Logger(),
	}
}

func (s *ExplanationService) Explain(ctx context.Context, version *aigovmodel.ModelVersion, input any, output *aigovernance.ModelOutput) (*aigovmodel.Explanation, error) {
	if version == nil {
		return nil, fmt.Errorf("model version is required")
	}
	if output == nil {
		return nil, fmt.Errorf("model output is required")
	}
	engine, ok := s.explainers[version.ExplainabilityType]
	if !ok {
		s.logger.Warn().
			Str("model_slug", version.ModelSlug).
			Str("explainability_type", string(version.ExplainabilityType)).
			Msg("unsupported explainability type, falling back to rule explainer")
		engine = explainer.NewRuleExplainer()
	}
	return engine.Explain(ctx, version, input, output)
}

func (s *ExplanationService) FromPrediction(log *aigovmodel.PredictionLog) (*aigovmodel.Explanation, error) {
	if log == nil {
		return nil, fmt.Errorf("prediction log is required")
	}
	var structured map[string]any
	if len(log.ExplanationStructured) > 0 {
		if err := json.Unmarshal(log.ExplanationStructured, &structured); err != nil {
			return nil, fmt.Errorf("decode explanation structured payload: %w", err)
		}
	}
	if structured == nil {
		structured = map[string]any{}
	}
	var factors []aigovmodel.Factor
	if len(log.ExplanationFactors) > 0 {
		if err := json.Unmarshal(log.ExplanationFactors, &factors); err != nil {
			return nil, fmt.Errorf("decode explanation factors: %w", err)
		}
	}
	confidence := 0.0
	if log.Confidence != nil {
		confidence = *log.Confidence
	}
	return &aigovmodel.Explanation{
		Structured:    structured,
		HumanReadable: log.ExplanationText,
		Factors:       factors,
		Confidence:    confidence,
		ModelSlug:     log.ModelSlug,
		ModelVersion:  log.ModelVersionNumber,
	}, nil
}
