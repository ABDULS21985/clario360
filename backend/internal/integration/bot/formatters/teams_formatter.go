package formatters

import bottypes "github.com/clario360/platform/internal/integration/bot/types"

func TeamsResponse(resp *bottypes.BotResponse) map[string]any {
	return map[string]any{
		"type": "message",
		"attachments": []map[string]any{
			{
				"contentType": "application/vnd.microsoft.card.adaptive",
				"content": map[string]any{
					"type":    "AdaptiveCard",
					"version": "1.5",
					"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
					"body": []map[string]any{
						{"type": "TextBlock", "text": resp.Text, "wrap": true},
					},
				},
			},
		},
	}
}
