package explainer

import (
	"fmt"
	"strings"

	predictmodel "github.com/clario360/platform/internal/cyber/vciso/predict/model"
)

type PredictionNarrator struct{}

func NewPredictionNarrator() *PredictionNarrator {
	return &PredictionNarrator{}
}

func (n *PredictionNarrator) Explain(
	predictionType predictmodel.PredictionType,
	confidence float64,
	interval predictmodel.ConfidenceInterval,
	top []predictmodel.FeatureContribution,
	target string,
) (string, []string) {
	intro := fmt.Sprintf("%s generated with %.0f%% confidence (P10 %.2f, P50 %.2f, P90 %.2f).",
		humanizePredictionType(predictionType), confidence*100, interval.P10, interval.P50, interval.P90)
	if strings.TrimSpace(target) != "" {
		intro = fmt.Sprintf("%s Target: %s.", intro, target)
	}
	features := make([]string, 0, len(top))
	for _, item := range top {
		features = append(features, fmt.Sprintf("%s (%s %.2f)", item.Feature, item.Direction, item.SHAPValue))
	}
	text := intro
	if len(features) > 0 {
		text += " Main drivers: " + strings.Join(features, ", ") + "."
	}
	steps := defaultVerificationSteps(predictionType)
	if confidence < 0.6 {
		steps = append([]string{"Treat this as an early signal and confirm with current telemetry before escalation."}, steps...)
	}
	return text, steps
}

func humanizePredictionType(value predictmodel.PredictionType) string {
	switch value {
	case predictmodel.PredictionTypeAlertVolumeForecast:
		return "Alert volume forecast"
	case predictmodel.PredictionTypeAssetRisk:
		return "Asset targeting forecast"
	case predictmodel.PredictionTypeVulnerabilityExploit:
		return "Vulnerability exploit forecast"
	case predictmodel.PredictionTypeAttackTechniqueTrend:
		return "Attack technique trend forecast"
	case predictmodel.PredictionTypeInsiderThreatTrajectory:
		return "Insider threat trajectory forecast"
	case predictmodel.PredictionTypeCampaignDetection:
		return "Campaign clustering forecast"
	default:
		return "Predictive assessment"
	}
}

func defaultVerificationSteps(value predictmodel.PredictionType) []string {
	switch value {
	case predictmodel.PredictionTypeAlertVolumeForecast:
		return []string{"Compare forecast against current alert queue growth.", "Validate recent rule changes and maintenance windows."}
	case predictmodel.PredictionTypeAssetRisk:
		return []string{"Review exposed services and patch age for the top-ranked assets.", "Confirm whether recent alerts already indicate active probing."}
	case predictmodel.PredictionTypeVulnerabilityExploit:
		return []string{"Validate KEV and EPSS status for the highest-ranked CVEs.", "Check whether the affected products are internet-facing and business critical."}
	case predictmodel.PredictionTypeAttackTechniqueTrend:
		return []string{"Verify emerging techniques against current detections and threat feeds.", "Confirm coverage gaps before escalating the trend to leadership."}
	case predictmodel.PredictionTypeInsiderThreatTrajectory:
		return []string{"Review UEBA evidence, HR signals, and access anomalies before escalation.", "Have an analyst validate peer-group deviation and recent policy violations."}
	case predictmodel.PredictionTypeCampaignDetection:
		return []string{"Validate shared IOCs and timeline proximity across the clustered alerts.", "Confirm campaign stage with an investigator before broad incident messaging."}
	default:
		return []string{"Validate the prediction against live telemetry."}
	}
}
