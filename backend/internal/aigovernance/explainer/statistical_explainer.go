package explainer

import (
	"context"
	"fmt"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
)

type StatisticalExplainer struct{}

func NewStatisticalExplainer() *StatisticalExplainer {
	return &StatisticalExplainer{}
}

func (e *StatisticalExplainer) Explain(_ context.Context, version *aigovmodel.ModelVersion, _ any, output *aigovernance.ModelOutput) (*aigovmodel.Explanation, error) {
	current := numeric(output.Metadata["current_value"])
	baseline := numeric(output.Metadata["baseline_mean"])
	stddev := numeric(output.Metadata["baseline_stddev"])
	zScore := numeric(output.Metadata["z_score"])
	threshold := numeric(output.Metadata["threshold"])
	anomalyDetected, hasAnomalyFlag := boolValue(output.Metadata["anomaly_detected"])

	factors := []aigovmodel.Factor{
		{
			Name:        "Current Value",
			Value:       fmt.Sprintf("%.2f", current),
			Impact:      normalizedImpact(zScore),
			Direction:   "positive",
			Description: "Observed current value for the monitored metric.",
		},
		{
			Name:        "Baseline Mean",
			Value:       fmt.Sprintf("%.2f", baseline),
			Impact:      0,
			Direction:   "positive",
			Description: "Historical baseline average used as the reference point.",
		},
		{
			Name:        "Deviation",
			Value:       fmt.Sprintf("%.2fσ", zScore),
			Impact:      normalizedImpact(zScore),
			Direction:   "positive",
			Description: "Standard deviation distance from the baseline.",
		},
	}
	structured := map[string]any{
		"current_value":   current,
		"baseline_mean":   baseline,
		"baseline_stddev": stddev,
		"z_score":         zScore,
		"threshold":       threshold,
	}
	if hasAnomalyFlag {
		structured["anomaly_detected"] = anomalyDetected
	}

	human := ""
	if hasAnomalyFlag && !anomalyDetected {
		human = fmt.Sprintf("No statistically significant deviation was detected for this evaluation window (threshold %.2fσ).", threshold)
	}
	if human == "" {
		rendered, err := renderTemplate(version, map[string]any{
			"current_value":    current,
			"baseline_mean":    baseline,
			"baseline_stddev":  stddev,
			"z_score":          zScore,
			"threshold":        threshold,
			"anomaly_detected": structured["anomaly_detected"],
			"confidence":       output.Confidence,
		})
		if err != nil {
			return nil, err
		}
		human = rendered
	}
	if human == "" {
		human = fmt.Sprintf("Observed value %.2f deviates %.2f standard deviations from the baseline of %.2f.", current, zScore, baseline)
	}

	return &aigovmodel.Explanation{
		Structured:    structured,
		HumanReadable: human,
		Factors:       factors,
		Confidence:    output.Confidence,
		ExplainerType: string(aigovmodel.ExplainabilityStatisticalDeviation),
		ModelSlug:     version.ModelSlug,
		ModelVersion:  version.VersionNumber,
	}, nil
}

func numeric(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		return 0
	}
}

func normalizedImpact(value float64) float64 {
	if value < 0 {
		value = -value
	}
	if value > 1 {
		return 1
	}
	return value
}
