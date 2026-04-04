package sections

import "fmt"

func BuildRecommendations(sectionData map[string]interface{}) map[string]any {
	items := make([]string, 0)
	if kpiSummary, ok := sectionData["kpi_summary"].(map[string]any); ok {
		if kpis, ok := kpiSummary["kpis"].([]map[string]any); ok {
			for _, item := range kpis {
				status, _ := item["status"].(string)
				if status == "critical" {
					items = append(items, recommendationForKPI(item))
				}
			}
		}
	}
	if governance, ok := sectionData["governance"].(map[string]any); ok && available(governance) {
		if overdue := number(governance["overdue_count"]); overdue > 0 {
			items = append(items, recommendationf("Resolve %.0f overdue action items before the next board meeting.", overdue))
		}
	}
	if legal, ok := sectionData["legal"].(map[string]any); ok && available(legal) {
		if expiring := number(legal["expiring_count"]); expiring > 0 {
			items = append(items, recommendationf("Review %.0f contracts expiring in 30 days.", expiring))
		}
	}
	if security, ok := sectionData["security_posture"].(map[string]any); ok && available(security) {
		if coverage := number(security["coverage"]); coverage < 100 {
			items = append(items, recommendationf("Close MITRE ATT&CK coverage gaps; current coverage is %.2f%%.", coverage))
		}
	}
	return map[string]any{
		"available": true,
		"items":     uniqueStrings(items),
	}
}

func recommendationForKPI(item map[string]any) string {
	return recommendationf("Address %s: currently at %.2f, threshold is %.2f.", stringValue(item["name"]), number(item["value"]), number(item["critical"]))
}

func recommendationf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok || value == "" {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
