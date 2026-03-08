package formatters

import (
	"strings"

	bottypes "github.com/clario360/platform/internal/integration/bot/types"
)

func SlackResponse(resp *bottypes.BotResponse) map[string]any {
	payload := map[string]any{
		"text": resp.Text,
	}
	if resp.Ephemeral {
		payload["response_type"] = "ephemeral"
	} else {
		payload["response_type"] = "in_channel"
	}
	payload["blocks"] = []map[string]any{
		{
			"type": "section",
			"text": map[string]any{
				"type": "mrkdwn",
				"text": strings.TrimSpace(resp.Text),
			},
		},
	}
	return payload
}
