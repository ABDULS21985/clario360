package scorer

import (
	"time"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

func alertSeverityImpact(severity string) float64 {
	switch severity {
	case "critical":
		return 30
	case "high":
		return 20
	case "medium":
		return 10
	default:
		return 5
	}
}

func recencyWeight(createdAt time.Time, now time.Time) float64 {
	age := now.Sub(createdAt)
	switch {
	case age <= 24*time.Hour:
		return 1.0
	case age <= 7*24*time.Hour:
		return 0.5
	case age <= 14*24*time.Hour:
		return 0.3
	default:
		return 0.15
	}
}

func riskLevelForScore(score float64) model.RiskLevel {
	switch {
	case score >= 75:
		return model.RiskLevelCritical
	case score >= 50:
		return model.RiskLevelHigh
	case score >= 25:
		return model.RiskLevelMedium
	default:
		return model.RiskLevelLow
	}
}
