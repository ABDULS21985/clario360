package jira

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
	values.Set("audience", "api.atlassian.com")
	values.Set("client_id", cfg.ClientID)
	values.Set("scope", strings.Join(cfg.Scopes, " "))
	values.Set("redirect_uri", cfg.RedirectURI)
	values.Set("state", state)
	values.Set("response_type", "code")
	values.Set("prompt", "consent")
	return "https://auth.atlassian.com/authorize?" + values.Encode()
}

func ExchangeCode(ctx context.Context, cfg OAuthConfig, code string) (map[string]any, error) {
	payload := map[string]any{
		"grant_type":    "authorization_code",
		"client_id":     cfg.ClientID,
		"client_secret": cfg.ClientSecret,
		"code":          code,
		"redirect_uri":  cfg.RedirectURI,
	}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://auth.atlassian.com/oauth/token", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("atlassian token exchange returned %d", resp.StatusCode)
	}
	var tokenPayload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&tokenPayload); err != nil {
		return nil, err
	}
	return tokenPayload, nil
}
