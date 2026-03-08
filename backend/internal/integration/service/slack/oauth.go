package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
}

func BuildOAuthURL(cfg OAuthConfig, state string) string {
	values := url.Values{}
	values.Set("client_id", cfg.ClientID)
	values.Set("scope", strings.Join(cfg.Scopes, ","))
	values.Set("redirect_uri", cfg.RedirectURI)
	values.Set("state", state)
	return "https://slack.com/oauth/v2/authorize?" + values.Encode()
}

func ExchangeCode(ctx context.Context, cfg OAuthConfig, code string) (map[string]any, error) {
	values := url.Values{}
	values.Set("client_id", cfg.ClientID)
	values.Set("client_secret", cfg.ClientSecret)
	values.Set("code", code)
	values.Set("redirect_uri", cfg.RedirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://slack.com/api/oauth.v2.access", bytes.NewBufferString(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("slack oauth exchange returned %d", resp.StatusCode)
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if ok, _ := payload["ok"].(bool); !ok {
		return nil, fmt.Errorf("slack oauth exchange failed: %v", payload["error"])
	}
	return payload, nil
}
