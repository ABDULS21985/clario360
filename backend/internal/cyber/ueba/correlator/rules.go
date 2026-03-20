package correlator

import (
	"github.com/clario360/platform/internal/cyber/ueba/model"
)

type ruleMatch struct {
	alertType   model.AlertType
	signals     []model.AnomalySignal
	severity    string
	mitreTactic string
}

func severityFromSignals(signals []model.AnomalySignal, escalate int, floor string) string {
	current := floor
	for _, signal := range signals {
		current = maxSeverity(current, signal.Severity)
	}
	return escalateSeverity(current, escalate)
}

func maxSeverity(left, right string) string {
	if severityRank(left) >= severityRank(right) {
		return left
	}
	return right
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

func escalateSeverity(severity string, levels int) string {
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
