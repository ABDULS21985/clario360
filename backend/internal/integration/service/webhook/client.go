package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/events"
	intmodel "github.com/clario360/platform/internal/integration/model"
)

type Client struct {
	httpClient *http.Client
}

func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *Client) Post(ctx context.Context, cfg intmodel.WebhookConfig, event *events.Event) (int, string, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return 0, "", fmt.Errorf("webhook url is required")
	}
	method := strings.ToUpper(strings.TrimSpace(cfg.Method))
	if method == "" {
		method = http.MethodPost
	}
	if cfg.ContentType == "" {
		cfg.ContentType = "application/json"
	}

	payload := map[string]any{
		"id":          event.ID,
		"type":        event.Type,
		"source":      event.Source,
		"tenant_id":   event.TenantID,
		"user_id":     event.UserID,
		"time":        event.Time,
		"data":        json.RawMessage(event.Data),
		"correlation": event.CorrelationID,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, "", fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, cfg.URL, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, "", fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", cfg.ContentType)
	for key, value := range cfg.Headers {
		req.Header.Set(key, value)
	}

	signedHeaders := SignWebhookRequest(bodyBytes, cfg.Secret, time.Now().UTC().Unix(), uuid.NewString())
	for key, value := range signedHeaders {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1000))
	if resp.StatusCode >= 400 {
		return resp.StatusCode, string(responseBody), fmt.Errorf("webhook receiver returned %d", resp.StatusCode)
	}
	return resp.StatusCode, string(responseBody), nil
}
