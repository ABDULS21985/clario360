package report

import (
	"fmt"
	"strings"
	"time"
)

func GenerateNarrative(sections map[string]interface{}, period [2]time.Time) string {
	start := period[0].Format("2006-01-02")
	end := period[1].Format("2006-01-02")
	var b strings.Builder
	fmt.Fprintf(&b, "## Executive Summary — %s to %s\n\n", start, end)
	b.WriteString("### Security Posture\n")
	b.WriteString(sectionNarrative(sections["security_posture"], func(section map[string]any) string {
		return fmt.Sprintf(
			"The organization's security risk score is currently %.2f/100 (Grade %s), %s from %.2f in the previous period. %.0f security alerts are currently open, including %.0f critical alerts. The security team's mean time to respond is %.2f hours. MITRE ATT&CK detection coverage stands at %.2f%%.",
			number(section["risk_score"]),
			stringValue(section["grade"]),
			stringValue(section["trend_word"]),
			number(section["prev_risk_score"]),
			number(section["open_alerts"]),
			number(section["critical_alerts"]),
			number(section["mttr_hours"]),
			number(section["coverage"]),
		)
	}))
	b.WriteString("\n\n### Data Intelligence\n")
	b.WriteString(sectionNarrative(sections["data_intelligence"], func(section map[string]any) string {
		return fmt.Sprintf(
			"Data quality score is %.2f%% (Grade %s). Pipeline operations achieved a %.2f%% success rate over the period, with %.0f failures in the last 24 hours. %.0f data contradictions remain open for resolution.",
			number(section["quality_score"]),
			stringValue(section["quality_grade"]),
			number(section["success_rate"]),
			number(section["failed_count"]),
			number(section["contradiction_count"]),
		)
	}))
	b.WriteString("\n\n### Governance\n")
	b.WriteString(sectionNarrative(sections["governance"], func(section map[string]any) string {
		return fmt.Sprintf(
			"Governance compliance score is %.2f%%. %.0f committee meetings are upcoming in the next 30 days. %.0f action items are overdue. %.0f sets of meeting minutes are pending approval.",
			number(section["compliance_score"]),
			number(section["meeting_count"]),
			number(section["overdue_count"]),
			number(section["minutes_pending"]),
		)
	}))
	b.WriteString("\n\n### Legal Operations\n")
	b.WriteString(sectionNarrative(sections["legal"], func(section map[string]any) string {
		return fmt.Sprintf(
			"%.0f contracts are currently active with a total value of %.2f. %.0f contracts are expiring within 30 days. %.0f contracts are flagged as high-risk and require attention.",
			number(section["active_contracts"]),
			number(section["value"]),
			number(section["expiring_count"]),
			number(section["high_risk_count"]),
		)
	}))
	b.WriteString("\n\n### Key Recommendations\n")
	recommendations, ok := sections["recommendations"].(map[string]any)
	if !ok {
		b.WriteString("Data unavailable for this section.")
		return b.String()
	}
	items, ok := recommendations["items"].([]string)
	if !ok || len(items) == 0 {
		b.WriteString("No immediate executive recommendations generated for this period.")
		return b.String()
	}
	for _, item := range items {
		fmt.Fprintf(&b, "- %s\n", item)
	}
	return b.String()
}

func TrendWord(delta float64, higherIsBetter bool) string {
	adjusted := delta
	if !higherIsBetter {
		adjusted = -delta
	}
	switch {
	case adjusted > 5:
		return "significantly improved"
	case adjusted > 0:
		return "improved"
	case adjusted == 0:
		return "remained stable"
	case adjusted < -5:
		return "significantly declined"
	default:
		return "declined"
	}
}

func sectionNarrative(value interface{}, render func(map[string]any) string) string {
	section, ok := value.(map[string]any)
	if !ok || !available(section) {
		return "Data unavailable for this section."
	}
	return render(section)
}

func available(section map[string]any) bool {
	if section == nil {
		return false
	}
	available, ok := section["available"].(bool)
	return ok && available
}

func number(value interface{}) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case int32:
		return float64(typed)
	default:
		return 0
	}
}

func stringValue(value interface{}) string {
	if value == nil {
		return ""
	}
	if typed, ok := value.(string); ok {
		return typed
	}
	return fmt.Sprint(value)
}
