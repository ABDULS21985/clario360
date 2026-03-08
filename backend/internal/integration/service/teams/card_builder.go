package teams

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/clario360/platform/internal/events"
)

func BuildAdaptiveCard(event *events.Event, appURL string) map[string]any {
	title := strings.TrimPrefix(event.Type, "com.clario360.")
	subtitle := "Clario 360 event"
	var data map[string]any
	if len(event.Data) > 0 {
		_ = jsonUnmarshal(event.Data, &data)
		if customTitle := stringValue(data["title"]); customTitle != "" && customTitle != "<nil>" {
			title = customTitle
		}
		if severity := stringValue(data["severity"]); severity != "" && severity != "<nil>" {
			subtitle = fmt.Sprintf("%s severity alert", strings.Title(strings.ToLower(severity)))
		}
	}

	body := []map[string]any{
		{"type": "TextBlock", "size": "Medium", "weight": "Bolder", "text": title, "wrap": true},
		{"type": "TextBlock", "text": subtitle, "isSubtle": true, "wrap": true},
	}
	if summary := stringValue(data["description"]); summary != "" && summary != "<nil>" {
		body = append(body, map[string]any{"type": "TextBlock", "text": summary, "wrap": true})
	}

	actions := []map[string]any{
		{"type": "Action.OpenUrl", "title": "View in Clario 360", "url": strings.TrimRight(appURL, "/")},
	}
	if id := stringValue(data["id"]); id != "" && id != "<nil>" {
		actions[0]["url"] = strings.TrimRight(appURL, "/") + "/cyber/alerts/" + id
	}

	return map[string]any{
		"type":    "AdaptiveCard",
		"version": "1.5",
		"body":    body,
		"actions": actions,
		"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
	}
}

func jsonUnmarshal(data []byte, target any) error {
	return json.Unmarshal(data, target)
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", value)
}
