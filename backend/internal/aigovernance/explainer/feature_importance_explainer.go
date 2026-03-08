package explainer

import (
	"context"
	"fmt"
	"sort"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
)

type FeatureImportanceExplainer struct{}

func NewFeatureImportanceExplainer() *FeatureImportanceExplainer {
	return &FeatureImportanceExplainer{}
}

func (e *FeatureImportanceExplainer) Explain(_ context.Context, version *aigovmodel.ModelVersion, _ any, output *aigovernance.ModelOutput) (*aigovmodel.Explanation, error) {
	componentScores := numericMap(output.Metadata["component_scores"])
	componentWeights := numericMap(output.Metadata["component_weights"])
	overallScore := numeric(output.Metadata["overall_score"])
	if overallScore == 0 {
		overallScore = 1
	}

	type contribution struct {
		Name  string
		Value float64
		Share float64
	}
	contributions := make([]contribution, 0, len(componentScores))
	for name, score := range componentScores {
		weight := componentWeights[name]
		if weight == 0 {
			weight = 1
		}
		share := ((score * weight) / overallScore) * 100
		contributions = append(contributions, contribution{Name: name, Value: score, Share: share})
	}
	sort.SliceStable(contributions, func(i, j int) bool {
		return contributions[i].Share > contributions[j].Share
	})

	factors := make([]aigovmodel.Factor, 0, min(len(contributions), 5))
	for idx, item := range contributions {
		if idx >= 5 {
			break
		}
		factors = append(factors, aigovmodel.Factor{
			Name:        item.Name,
			Value:       fmt.Sprintf("%.2f", item.Value),
			Impact:      item.Share / 100,
			Direction:   "positive",
			Description: fmt.Sprintf("%s contributed %.1f%% of the total score.", item.Name, item.Share),
		})
	}

	human, err := renderTemplate(version, map[string]any{
		"contributions": contributions,
		"overall_score": overallScore,
		"confidence":    output.Confidence,
	})
	if err != nil {
		return nil, err
	}
	if human == "" {
		human = humanizeFactors(factors)
	}

	return &aigovmodel.Explanation{
		Structured: map[string]any{
			"overall_score":  overallScore,
			"contributions":  contributions,
			"component_count": len(contributions),
		},
		HumanReadable: human,
		Factors:       factors,
		Confidence:    output.Confidence,
		ExplainerType: string(aigovmodel.ExplainabilityFeatureImportance),
		ModelSlug:     version.ModelSlug,
		ModelVersion:  version.VersionNumber,
	}, nil
}
