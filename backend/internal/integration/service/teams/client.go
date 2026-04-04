package teams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/clario360/platform/internal/events"
	intmodel "github.com/clario360/platform/internal/integration/model"
)

type Client struct {
	httpClient    *http.Client
	tokenProvider *TokenProvider
	appURL        string
}

func NewClient(timeout time.Duration, appURL string) *Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &Client{
		httpClient:    &http.Client{Timeout: timeout},
		tokenProvider: NewTokenProvider(timeout),
		appURL:        appURL,
	}
}

func (c *Client) Send(ctx context.Context, cfg intmodel.TeamsConfig, event *events.Event) (int, string, error) {
	card := BuildAdaptiveCard(event, c.appURL)
	return c.sendActivity(ctx, cfg, map[string]any{
		"type": "message",
		"attachments": []map[string]any{
			{
				"contentType": "application/vnd.microsoft.card.adaptive",
				"content":     card,
			},
		},
	})
}

func (c *Client) SendTest(ctx context.Context, cfg intmodel.TeamsConfig) (int, string, error) {
	return c.sendActivity(ctx, cfg, map[string]any{
		"type": "message",
		"text": "Clario 360 integration test",
		"attachments": []map[string]any{
			{
				"contentType": "application/vnd.microsoft.card.adaptive",
				"content": map[string]any{
					"type":    "AdaptiveCard",
					"version": "1.5",
					"body": []map[string]any{
						{"type": "TextBlock", "size": "Medium", "weight": "Bolder", "text": "Clario 360 integration test"},
						{"type": "TextBlock", "text": "The Teams connector can send messages successfully.", "wrap": true},
					},
					"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
				},
			},
		},
	})
}

func (c *Client) SendActivityToConversation(ctx context.Context, cfg intmodel.TeamsConfig, serviceURL, conversationID string, activity map[string]any) (int, string, error) {
	cfg.ServiceURL = serviceURL
	cfg.ConversationID = conversationID
	return c.sendActivity(ctx, cfg, activity)
}

func (c *Client) sendActivity(ctx context.Context, cfg intmodel.TeamsConfig, activity map[string]any) (int, string, error) {
	token, err := c.tokenProvider.Token(ctx, cfg.BotAppID, cfg.BotPassword)
	if err != nil {
		return 0, "", err
	}

	bodyBytes, err := json.Marshal(activity)
	if err != nil {
		return 0, "", err
	}

	endpoint := strings.TrimRight(cfg.ServiceURL, "/") + "/v3/conversations/" + cfg.ConversationID + "/activities"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1000))
	if resp.StatusCode >= 400 {
		return resp.StatusCode, string(body), fmt.Errorf("teams api returned %d", resp.StatusCode)
	}
	return resp.StatusCode, string(body), nil
}
