package servicenow

import "encoding/json"

type WebhookEvent struct {
	Result map[string]any `json:"result"`
}

func ParseWebhookEvent(body []byte) (*WebhookEvent, error) {
	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, err
	}
	return &event, nil
}
