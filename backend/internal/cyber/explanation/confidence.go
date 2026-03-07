package explanation

import (
	"fmt"
	"math"
	"strings"

	"github.com/clario360/platform/internal/cyber/model"
)

// ComputeConfidence computes the alert confidence score and its contributing factors.
func ComputeConfidence(rule *model.DetectionRule, match model.RuleMatch, asset *model.Asset) (float64, []model.ConfidenceFactor) {
	base := 0.70
	if rule != nil && rule.BaseConfidence > 0 {
		base = rule.BaseConfidence
	}
	score := base
	factors := make([]model.ConfidenceFactor, 0, 8)

	if count, ok := intFromDetails(match.MatchDetails["matched_condition_count"]); ok {
		extra := clampFloat(float64(maxInt(count-1, 0))*0.05, 0, 0.15)
		if extra > 0 {
			score += extra
			factors = append(factors, model.ConfidenceFactor{
				Factor:      "multiple_conditions",
				Impact:      extra,
				Description: "Multiple detection conditions matched",
			})
		}
	}

	if ageHours, ok := floatFromDetails(match.MatchDetails["indicator_age_hours"]); ok && ageHours <= 24 {
		score += 0.10
		factors = append(factors, model.ConfidenceFactor{
			Factor:      "recent_indicator",
			Impact:      0.10,
			Description: "Threat indicator is recently active",
		})
	}

	if asset != nil && asset.Criticality == model.CriticalityCritical {
		score += 0.05
		factors = append(factors, model.ConfidenceFactor{
			Factor:      "critical_asset",
			Impact:      0.05,
			Description: "Affects a critical asset",
		})
	}

	if len(match.Events) > 10 {
		score += 0.05
		factors = append(factors, model.ConfidenceFactor{
			Factor:      "multiple_events",
			Impact:      0.05,
			Description: "Multiple triggering events were observed",
		})
	}

	if correlated, ok := boolFromDetails(match.MatchDetails["correlated_recent"]); ok && correlated {
		score += 0.10
		factors = append(factors, model.ConfidenceFactor{
			Factor:      "correlated_detection",
			Impact:      0.10,
			Description: "Correlated with other detections on the same asset in the last hour",
		})
	}

	if rule != nil {
		fpRate := rule.FPRate()
		switch {
		case fpRate > 0.40:
			score -= 0.10
			factors = append(factors, model.ConfidenceFactor{
				Factor:      "high_fp_rate",
				Impact:      -0.10,
				Description: fmt.Sprintf("Rule has high false positive rate (%.1f%%)", fpRate*100),
			})
		case fpRate > 0.20:
			score -= 0.05
			factors = append(factors, model.ConfidenceFactor{
				Factor:      "elevated_fp_rate",
				Impact:      -0.05,
				Description: fmt.Sprintf("Rule has elevated false positive rate (%.1f%%)", fpRate*100),
			})
		}
	}

	if maintenance, ok := boolFromDetails(match.MatchDetails["maintenance_window"]); ok && maintenance {
		score -= 0.05
		factors = append(factors, model.ConfidenceFactor{
			Factor:      "maintenance_window",
			Impact:      -0.05,
			Description: "Event occurred during a scheduled maintenance window",
		})
	}

	if serviceAccount, ok := stringFromDetails(match.MatchDetails["service_account"]); ok && serviceAccount != "" {
		score -= 0.05
		factors = append(factors, model.ConfidenceFactor{
			Factor:      "service_account",
			Impact:      -0.05,
			Description: fmt.Sprintf("Action performed by service account %q", serviceAccount),
		})
	}

	score = clampFloat(score, 0.05, 0.99)
	return math.Round(score*100) / 100, factors
}

func clampFloat(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func intFromDetails(value interface{}) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	default:
		return 0, false
	}
}

func floatFromDetails(value interface{}) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	default:
		return 0, false
	}
}

func boolFromDetails(value interface{}) (bool, bool) {
	typed, ok := value.(bool)
	return typed, ok
}

func stringFromDetails(value interface{}) (string, bool) {
	typed, ok := value.(string)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(typed), true
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
