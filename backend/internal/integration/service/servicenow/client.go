package servicenow

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	intmodel "github.com/clario360/platform/internal/integration/model"
)

type Client struct {
	httpClient *http.Client
}

func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 25 * time.Second
	}
	return &Client{httpClient: &http.Client{Timeout: timeout}}
}

func (c *Client) CreateIncident(ctx context.Context, cfg intmodel.ServiceNowConfig, payload map[string]any) (map[string]any, int, string, error) {
	var out map[string]any
	code, body, err := c.request(ctx, cfg, http.MethodPost, "/api/now/table/incident", payload, &out)
	return out, code, body, err
}

func (c *Client) GetIncident(ctx context.Context, cfg intmodel.ServiceNowConfig, sysID string) (map[string]any, int, string, error) {
	var out map[string]any
	code, body, err := c.request(ctx, cfg, http.MethodGet, "/api/now/table/incident/"+sysID, nil, &out)
	return out, code, body, err
}

func (c *Client) request(ctx context.Context, cfg intmodel.ServiceNowConfig, method, path string, payload any, out any) (int, string, error) {
	var bodyReader io.Reader
	if payload != nil {
		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return 0, "", err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(cfg.InstanceURL, "/")+path, bodyReader)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	switch strings.ToLower(cfg.AuthType) {
	case "oauth":
		req.Header.Set("Authorization", "Bearer "+cfg.OAuthToken)
	default:
		creds := base64.StdEncoding.EncodeToString([]byte(cfg.Username + ":" + cfg.Password))
		req.Header.Set("Authorization", "Basic "+creds)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1000))
	if resp.StatusCode >= 400 {
		return resp.StatusCode, string(body), fmt.Errorf("servicenow api returned %d", resp.StatusCode)
	}
	if out != nil && len(body) > 0 {
		if err := json.Unmarshal(body, out); err != nil {
			return resp.StatusCode, string(body), err
		}
	}
	return resp.StatusCode, string(body), nil
}
