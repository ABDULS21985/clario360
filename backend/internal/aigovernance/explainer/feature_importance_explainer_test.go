package explainer

import (
	"context"
	"testing"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/clario360/platform/internal/aigovernance"
)

func TestFeatureImportanceRanksContributions(t *testing.T) {
	version := &aigovmodel.ModelVersion{ModelSlug: "cyber-risk-scorer", VersionNumber: 1}
	output := &aigovernance.ModelOutput{
		Confidence: 0.92,
		Metadata: map[string]any{
			"overall_score": 80.0,
			"component_scores": map[string]any{
				"vulnerability": 85.0,
				"threat":        70.0,
				"configuration": 55.0,
			},
			"component_weights": map[string]any{
				"vulnerability": 0.30,
				"threat":        0.25,
				"configuration": 0.20,
			},
		},
	}

	explanation, err := NewFeatureImportanceExplainer().Explain(context.Background(), version, nil, output)
	if err != nil {
		t.Fatalf("Explain() error = %v", err)
	}
	if explanation.Factors[0].Name != "vulnerability" {
		t.Fatalf("expected vulnerability as top factor, got %q", explanation.Factors[0].Name)
	}
}

func TestFeatureImportanceTop5(t *testing.T) {
	version := &aigovmodel.ModelVersion{ModelSlug: "data-quality-scorer", VersionNumber: 1}
	componentScores := map[string]any{}
	componentWeights := map[string]any{}
	for idx := 0; idx < 10; idx++ {
		key := string(rune('a' + idx))
		componentScores[key] = float64(100 - idx)
		componentWeights[key] = 1.0
	}
	output := &aigovernance.ModelOutput{
		Confidence: 0.8,
		Metadata: map[string]any{
			"overall_score":     100.0,
			"component_scores":  componentScores,
			"component_weights": componentWeights,
		},
	}

	explanation, err := NewFeatureImportanceExplainer().Explain(context.Background(), version, nil, output)
	if err != nil {
		t.Fatalf("Explain() error = %v", err)
	}
	if len(explanation.Factors) != 5 {
		t.Fatalf("len(Factors) = %d, want 5", len(explanation.Factors))
	}
}
