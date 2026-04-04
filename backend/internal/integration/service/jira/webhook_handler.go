package jira

import "encoding/json"

type WebhookEvent struct {
	WebhookEvent string `json:"webhookEvent"`
	Issue        struct {
		ID     string `json:"id"`
		Key    string `json:"key"`
		Fields struct {
			Status struct {
				Name string `json:"name"`
			} `json:"status"`
		} `json:"fields"`
	} `json:"issue"`
	Changelog struct {
		Items []struct {
			Field      string `json:"field"`
			FromString string `json:"fromString"`
			ToString   string `json:"toString"`
		} `json:"items"`
	} `json:"changelog"`
}

func ParseWebhookEvent(body []byte) (*WebhookEvent, error) {
	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, err
	}
	return &event, nil
}
