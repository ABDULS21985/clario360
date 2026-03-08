package drift

import aigovmodel "github.com/clario360/platform/internal/aigovernance/model"

func AccuracyChange(reference, current *float64) *float64 {
	if reference == nil || current == nil {
		return nil
	}
	value := *current - *reference
	return &value
}

func BuildAlerts(outputLevel, confidenceLevel aigovmodel.DriftLevel, volumeChange, latencyChange, accuracyChange *float64) []aigovmodel.DriftAlert {
	alerts := make([]aigovmodel.DriftAlert, 0)
	if outputLevel == aigovmodel.DriftLevelSignificant || confidenceLevel == aigovmodel.DriftLevelSignificant {
		alerts = append(alerts, aigovmodel.DriftAlert{
			Type:        "psi",
			Severity:    "high",
			Message:     "Prediction distribution drift is significant and requires investigation.",
			Recommended: "Review input distribution and recent model behavior changes.",
		})
	}
	if volumeChange != nil && (*volumeChange >= 50 || *volumeChange <= -50) {
		alerts = append(alerts, aigovmodel.DriftAlert{
			Type:        "volume",
			Severity:    "warning",
			Message:     "Prediction volume changed by more than 50% versus the reference period.",
			Recommended: "Validate upstream traffic and feature availability.",
		})
	}
	if latencyChange != nil && *latencyChange >= 100 {
		alerts = append(alerts, aigovmodel.DriftAlert{
			Type:        "latency",
			Severity:    "warning",
			Message:     "Model latency doubled compared with the reference period.",
			Recommended: "Inspect recent infrastructure or dependency regressions.",
		})
	}
	if accuracyChange != nil && *accuracyChange <= -0.05 {
		alerts = append(alerts, aigovmodel.DriftAlert{
			Type:        "accuracy",
			Severity:    "high",
			Message:     "Feedback-backed accuracy dropped by more than 5 points.",
			Recommended: "Investigate recent data or ruleset shifts before promotion decisions.",
		})
	}
	return alerts
}
