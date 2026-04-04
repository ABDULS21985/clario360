package explainer

import (
	"context"
	"strings"
	"testing"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/clario360/platform/internal/aigovernance"
)

func TestStatisticalExplainerZScore(t *testing.T) {
	version := &aigovmodel.ModelVersion{ModelSlug: "cyber-anomaly-detector", VersionNumber: 1}
	output := &aigovernance.ModelOutput{
		Confidence: 0.88,
		Metadata: map[string]any{
			"current_value":   19.0,
			"baseline_mean":   4.2,
			"baseline_stddev": 4.2,
			"z_score":         3.5,
			"threshold":       3.0,
		},
	}

	explanation, err := NewStatisticalExplainer().Explain(context.Background(), version, nil, output)
	if err != nil {
		t.Fatalf("Explain() error = %v", err)
	}
	if !strings.Contains(explanation.HumanReadable, "3.50 standard deviations") {
		t.Fatalf("expected z-score explanation, got %q", explanation.HumanReadable)
	}
}

func TestStatisticalExplainerFactors(t *testing.T) {
	version := &aigovmodel.ModelVersion{ModelSlug: "visus-kpi-monitor", VersionNumber: 1}
	output := &aigovernance.ModelOutput{
		Confidence: 0.9,
		Metadata: map[string]any{
			"current_value":   85.0,
			"baseline_mean":   63.0,
			"baseline_stddev": 6.3,
			"z_score":         3.49,
			"threshold":       2.5,
		},
	}

	explanation, err := NewStatisticalExplainer().Explain(context.Background(), version, nil, output)
	if err != nil {
		t.Fatalf("Explain() error = %v", err)
	}
	if len(explanation.Factors) != 3 {
		t.Fatalf("len(Factors) = %d, want 3", len(explanation.Factors))
	}
	if explanation.Factors[1].Name != "Baseline Mean" {
		t.Fatalf("expected baseline factor, got %q", explanation.Factors[1].Name)
	}
}
