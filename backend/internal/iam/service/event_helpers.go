package service

import "strings"

func normalizeIAMEventType(eventType string) string {
	trimmed := strings.TrimSpace(eventType)
	switch {
	case trimmed == "":
		return "iam.unknown"
	case strings.HasPrefix(trimmed, "com.clario360.iam."):
		return trimmed
	case strings.HasPrefix(trimmed, "com.clario360."):
		return "iam." + strings.TrimPrefix(trimmed, "com.clario360.")
	case strings.HasPrefix(trimmed, "iam."):
		return trimmed
	default:
		return "iam." + trimmed
	}
}
