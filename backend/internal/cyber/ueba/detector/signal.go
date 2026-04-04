package detector

import (
	"context"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

type SignalDetector interface {
	Name() model.SignalType
	Detect(ctx context.Context, event *model.DataAccessEvent, profile *model.UEBAProfile) *model.AnomalySignal
}

func severityRank(severity string) int {
	switch severity {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func higherSeverity(left, right string) string {
	if severityRank(left) >= severityRank(right) {
		return left
	}
	return right
}

func escalateSeverity(severity string, levels int) string {
	if levels <= 0 {
		return severity
	}
	order := []string{"low", "medium", "high", "critical"}
	index := 0
	for i, value := range order {
		if value == severity {
			index = i
			break
		}
	}
	index += levels
	if index >= len(order) {
		index = len(order) - 1
	}
	return order[index]
}

func clampConfidence(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 0.99:
		return 0.99
	default:
		return value
	}
}
