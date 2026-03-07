package explanation

import (
	"fmt"

	"github.com/clario360/platform/internal/cyber/model"
)

// DetectFalsePositiveIndicators returns human-readable hints that an alert may be benign.
func DetectFalsePositiveIndicators(event model.SecurityEvent, asset *model.Asset) []string {
	indicators := make([]string, 0, 6)
	raw := event.RawMap()

	if safeList, ok := raw["safe_list_match"].(bool); ok && safeList && event.SourceIP != nil {
		indicators = append(indicators, fmt.Sprintf("Source IP %s is in the internal safe list", *event.SourceIP))
	}
	if windowName, ok := raw["maintenance_window_name"].(string); ok && windowName != "" {
		indicators = append(indicators, fmt.Sprintf("Activity occurred during scheduled maintenance window (%s)", windowName))
	}
	if serviceAccount, ok := raw["service_account"].(string); ok && serviceAccount != "" {
		indicators = append(indicators, fmt.Sprintf("Action was performed by service account %q", serviceAccount))
	}
	if knownSaaS, ok := raw["known_saas_destination"].(bool); ok && knownSaaS {
		indicators = append(indicators, "Destination is a known CDN/SaaS endpoint")
	}
	if historicalFPs, ok := raw["historical_false_positive_count"].(float64); ok && historicalFPs > 0 {
		indicators = append(indicators, fmt.Sprintf("Similar alerts from this rule were marked false positive %.0f times in the past 30 days", historicalFPs))
	}
	if asset != nil && asset.Name != "" {
		if typicalTraffic, ok := raw["typical_for_asset"].(bool); ok && typicalTraffic {
			indicators = append(indicators, fmt.Sprintf("Asset %s typically generates this type of traffic", asset.Name))
		}
	}
	return indicators
}
