package slack

import (
	"fmt"
	"strings"
	"time"
)

func BuildAlertMessage(alert map[string]any, appURL string) []map[string]any {
	alertID := stringValue(alert["id"])
	title := firstNonEmpty(stringValue(alert["title"]), "Alert")
	severity := strings.ToLower(firstNonEmpty(stringValue(alert["severity"]), "info"))
	summary := firstNonEmpty(extractNestedString(alert, "explanation", "summary"), stringValue(alert["description"]))
	confidence := confidencePercent(alert)
	status := firstNonEmpty(stringValue(alert["status"]), "new")
	technique := firstNonEmpty(stringValue(alert["mitre_technique_name"]), stringValue(alert["mitre_technique_id"]))

	details := []string{
		fmt.Sprintf("*Confidence:* %d%%", confidence),
		fmt.Sprintf("*Status:* %s", status),
	}
	if technique != "" {
		details = append(details, fmt.Sprintf("*MITRE Technique:* %s", technique))
	}

	blocks := []map[string]any{
		headerBlock(fmt.Sprintf("%s %s Alert: %s", severityEmoji(severity), strings.Title(severity), title)),
		sectionBlock(strings.Join(details, "\n")),
	}
	if summary != "" {
		blocks = append(blocks, sectionBlock("*Summary:* "+truncate(summary, 300)))
	}
	blocks = append(blocks, contextBlock(fmt.Sprintf("Detected %s", relativeTime(alert["created_at"]))))

	actions := []map[string]any{
		buttonAction("clario_ack", "Acknowledge", alertID, "primary", ""),
		buttonAction("clario_investigate", "Investigate", alertID, "", ""),
		buttonAction("", "View in Clario 360", "", "", strings.TrimRight(appURL, "/")+"/cyber/alerts/"+alertID),
	}
	blocks = append(blocks, map[string]any{"type": "actions", "elements": actions})
	return blocks
}

func BuildNotificationMessage(notification map[string]any) []map[string]any {
	title := firstNonEmpty(stringValue(notification["title"]), "Notification")
	body := firstNonEmpty(stringValue(notification["body"]), stringValue(notification["description"]))
	return []map[string]any{
		headerBlock("🔔 " + title),
		sectionBlock(truncate(body, 300)),
	}
}

func BuildStatusMessage(payload map[string]any) []map[string]any {
	text := []string{"*Clario 360 Platform Status*"}
	for key, value := range payload {
		text = append(text, fmt.Sprintf("*%s:* %v", strings.ReplaceAll(strings.Title(strings.ReplaceAll(key, "_", " ")), "Id", "ID"), value))
	}
	return []map[string]any{sectionBlock(strings.Join(text, "\n"))}
}

func BuildAlertListMessage(alerts []map[string]any, filter string) []map[string]any {
	lines := []string{fmt.Sprintf("*Recent Alerts* %s", filter)}
	for _, alert := range alerts {
		lines = append(lines, fmt.Sprintf("%s *%s* — %s — %s%% confidence",
			severityEmoji(strings.ToLower(stringValue(alert["severity"]))),
			firstNonEmpty(stringValue(alert["title"]), "Alert"),
			firstNonEmpty(stringValue(alert["status"]), "new"),
			confidencePercent(alert),
		))
	}
	return []map[string]any{sectionBlock(strings.Join(lines, "\n"))}
}

func BuildAlertDetailMessage(alert map[string]any) []map[string]any {
	title := firstNonEmpty(stringValue(alert["title"]), "Alert")
	summary := firstNonEmpty(extractNestedString(alert, "explanation", "summary"), stringValue(alert["description"]))
	details := []string{
		fmt.Sprintf("*Severity:* %s", firstNonEmpty(stringValue(alert["severity"]), "info")),
		fmt.Sprintf("*Confidence:* %d%%", confidencePercent(alert)),
		fmt.Sprintf("*Status:* %s", firstNonEmpty(stringValue(alert["status"]), "new")),
	}
	if summary != "" {
		details = append(details, "", "*Summary:*", truncate(summary, 1500))
	}
	return []map[string]any{
		headerBlock("🔍 Investigation: " + title),
		sectionBlock(strings.Join(details, "\n")),
	}
}

func severityEmoji(severity string) string {
	switch severity {
	case "critical":
		return "🔴"
	case "high":
		return "🟠"
	case "medium":
		return "🟡"
	case "low":
		return "🔵"
	default:
		return "⚪"
	}
}

func headerBlock(text string) map[string]any {
	return map[string]any{
		"type": "header",
		"text": map[string]any{
			"type": "plain_text",
			"text": text,
		},
	}
}

func sectionBlock(text string) map[string]any {
	return map[string]any{
		"type": "section",
		"text": map[string]any{
			"type": "mrkdwn",
			"text": text,
		},
	}
}

func contextBlock(text string) map[string]any {
	return map[string]any{
		"type": "context",
		"elements": []map[string]any{
			{
				"type": "mrkdwn",
				"text": text,
			},
		},
	}
}

func buttonAction(actionID, text, value, style, url string) map[string]any {
	block := map[string]any{
		"type": "button",
		"text": map[string]any{
			"type": "plain_text",
			"text": text,
		},
	}
	if actionID != "" {
		block["action_id"] = actionID
	}
	if value != "" {
		block["value"] = value
	}
	if style != "" {
		block["style"] = style
	}
	if url != "" {
		block["url"] = url
	}
	return block
}

func confidencePercent(alert map[string]any) int {
	for _, key := range []string{"confidence_score", "confidence"} {
		switch value := alert[key].(type) {
		case float64:
			if value <= 1 {
				return int(value * 100)
			}
			return int(value)
		case int:
			return value
		}
	}
	return 0
}

func extractNestedString(payload map[string]any, parent, child string) string {
	if nested, ok := payload[parent].(map[string]any); ok {
		return stringValue(nested[child])
	}
	return ""
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprintf("%v", value)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}

func relativeTime(value any) string {
	str := stringValue(value)
	if str == "" || str == "<nil>" {
		return "just now"
	}
	parsed, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return "recently"
	}
	diff := time.Since(parsed)
	switch {
	case diff < time.Minute:
		return "moments ago"
	case diff < time.Hour:
		return fmt.Sprintf("%d minutes ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(diff.Hours()))
	default:
		return parsed.Format(time.RFC822)
	}
}
