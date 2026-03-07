package explanation

import (
	"strings"

	"github.com/clario360/platform/internal/cyber/model"
)

// GenerateRecommendedActions returns severity and threat-specific analyst guidance.
func GenerateRecommendedActions(alert *model.Alert, asset *model.Asset) []string {
	if alert == nil {
		return nil
	}
	actions := make([]string, 0, 8)
	switch alert.Severity {
	case model.SeverityCritical:
		actions = append(actions,
			"Immediately isolate the affected asset from the network",
			"Escalate to the security incident response team",
			"Preserve evidence: capture memory dump and disk image",
			"Begin incident response procedure per IR playbook",
		)
	case model.SeverityHigh:
		actions = append(actions,
			"Investigate within 4 hours per SLA",
			"Check for indicators of lateral movement",
			"Review affected user's recent activity",
		)
	case model.SeverityMedium:
		actions = append(actions,
			"Review during next analyst shift",
			"Check if this matches known benign behavior",
			"Update detection rule if confirmed false positive",
		)
	default:
		actions = append(actions,
			"Log for weekly trending analysis",
			"No immediate action required",
		)
	}

	lowerTitle := strings.ToLower(alert.Title)
	if strings.Contains(lowerTitle, "ransomware") && alert.Severity == model.SeverityCritical {
		actions = append(actions,
			"Verify backup integrity for affected systems",
			"Check for lateral movement to connected assets",
		)
	}

	if asset != nil && asset.Criticality == model.CriticalityCritical && alert.Severity.Rank() < model.SeverityHigh.Rank() {
		actions = append(actions, "Prioritize triage because the impacted asset is business critical")
	}
	return uniqueStrings(actions)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
