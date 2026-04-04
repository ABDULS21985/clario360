package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/clario360/platform/internal/events"
	intmodel "github.com/clario360/platform/internal/integration/model"
)

type Client struct {
	httpClient *http.Client
	appURL     string
}

func NewClient(timeout time.Duration, appURL string) *Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		appURL:     strings.TrimRight(appURL, "/"),
	}
}

func (c *Client) Send(ctx context.Context, cfg intmodel.SlackConfig, event *events.Event) (int, string, error) {
	payload := c.buildPayload(event)
	channelID := cfg.ChannelID
	if channelID == "" && cfg.IncomingWebhookURL != "" {
		return c.postWebhook(ctx, cfg.IncomingWebhookURL, payload)
	}
	return c.postMessage(ctx, cfg.BotToken, channelID, payload)
}

func (c *Client) SendTest(ctx context.Context, cfg intmodel.SlackConfig) (int, string, error) {
	payload := map[string]any{
		"text": "Clario 360 integration test",
		"blocks": []map[string]any{
			headerBlock("✅ Clario 360 integration test"),
			sectionBlock("This message confirms the Slack integration is authenticated and able to deliver messages."),
		},
	}
	if cfg.IncomingWebhookURL != "" {
		return c.postWebhook(ctx, cfg.IncomingWebhookURL, payload)
	}
	return c.postMessage(ctx, cfg.BotToken, cfg.ChannelID, payload)
}

func (c *Client) UsersInfo(ctx context.Context, botToken, userID string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://slack.com/api/users.info?user="+url.QueryEscape(userID), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+botToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 16384))
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode slack users.info response: %w", err)
	}
	if ok, _ := payload["ok"].(bool); !ok {
		return nil, fmt.Errorf("slack users.info failed: %v", payload["error"])
	}
	return payload, nil
}

func (c *Client) UpdateMessage(ctx context.Context, botToken, channelID, ts string, payload map[string]any) (int, string, error) {
	payload["channel"] = channelID
	payload["ts"] = ts
	return c.slackAPI(ctx, botToken, "https://slack.com/api/chat.update", payload)
}

func (c *Client) PostMessage(ctx context.Context, botToken, channelID string, payload map[string]any) (int, string, error) {
	return c.postMessage(ctx, botToken, channelID, payload)
}

func (c *Client) PostThreadReply(ctx context.Context, botToken, channelID, threadTS string, payload map[string]any) (int, string, error) {
	payload["thread_ts"] = threadTS
	return c.postMessage(ctx, botToken, channelID, payload)
}

func (c *Client) PostResponseURL(ctx context.Context, responseURL string, payload map[string]any) error {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, responseURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1000))
		return fmt.Errorf("slack response_url returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *Client) postMessage(ctx context.Context, botToken, channelID string, payload map[string]any) (int, string, error) {
	payload["channel"] = channelID
	return c.slackAPI(ctx, botToken, "https://slack.com/api/chat.postMessage", payload)
}

func (c *Client) postWebhook(ctx context.Context, webhookURL string, payload map[string]any) (int, string, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1000))
	if resp.StatusCode >= 400 {
		return resp.StatusCode, string(body), fmt.Errorf("slack webhook returned %d", resp.StatusCode)
	}
	return resp.StatusCode, string(body), nil
}

func (c *Client) slackAPI(ctx context.Context, botToken, endpoint string, payload map[string]any) (int, string, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Authorization", "Bearer "+botToken)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1000))
	if resp.StatusCode >= 400 {
		return resp.StatusCode, string(body), fmt.Errorf("slack api returned %d", resp.StatusCode)
	}
	var responsePayload map[string]any
	if len(body) > 0 {
		_ = json.Unmarshal(body, &responsePayload)
		if ok, exists := responsePayload["ok"].(bool); exists && !ok {
			return resp.StatusCode, string(body), fmt.Errorf("slack api error: %v", responsePayload["error"])
		}
	}
	return resp.StatusCode, string(body), nil
}

func (c *Client) buildPayload(event *events.Event) map[string]any {
	var data map[string]any
	if len(event.Data) > 0 {
		_ = json.Unmarshal(event.Data, &data)
	}

	var blocks []map[string]any
	switch {
	case strings.Contains(event.Type, "alert"):
		blocks = BuildAlertMessage(data, c.appURL)
	default:
		data["title"] = firstNonEmpty(stringValue(data["title"]), trimEventType(event.Type))
		blocks = BuildNotificationMessage(data)
	}

	text := trimEventType(event.Type)
	if title := stringValue(data["title"]); title != "" && title != "<nil>" {
		text = title
	}
	return map[string]any{
		"text":   text,
		"blocks": blocks,
	}
}

func trimEventType(eventType string) string {
	return strings.TrimPrefix(eventType, "com.clario360.")
}
