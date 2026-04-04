package commands

import bottypes "github.com/clario360/platform/internal/integration/bot/types"

func linkedError() *bottypes.BotResponse {
	return &bottypes.BotResponse{
		Text:      "⚠️ Your account is not linked to a Clario 360 user.",
		Ephemeral: true,
	}
}

func permissionError(permission string) *bottypes.BotResponse {
	return &bottypes.BotResponse{
		Text:      "⚠️ You do not have permission to execute this command. Required: `" + permission + "`.",
		Ephemeral: true,
	}
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func extractNestedString(payload map[string]any, parent, child string) string {
	if nested, ok := payload[parent].(map[string]any); ok {
		return stringValue(nested[child])
	}
	return ""
}
