package teams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type TokenProvider struct {
	httpClient *http.Client
	mu         sync.Mutex
	token      string
	expiresAt  time.Time
}

func NewTokenProvider(timeout time.Duration) *TokenProvider {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &TokenProvider{
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (p *TokenProvider) Token(ctx context.Context, appID, appPassword string) (string, error) {
	p.mu.Lock()
	if p.token != "" && time.Until(p.expiresAt) > time.Minute {
		token := p.token
		p.mu.Unlock()
		return token, nil
	}
	p.mu.Unlock()

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", appID)
	form.Set("client_secret", appPassword)
	form.Set("scope", "https://api.botframework.com/.default")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://login.microsoftonline.com/botframework.com/oauth2/v2.0/token", bytes.NewBufferString(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("teams token endpoint returned %d", resp.StatusCode)
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.AccessToken == "" {
		return "", fmt.Errorf("teams token response missing access_token")
	}

	p.mu.Lock()
	p.token = payload.AccessToken
	p.expiresAt = time.Now().UTC().Add(time.Duration(payload.ExpiresIn) * time.Second)
	p.mu.Unlock()
	return payload.AccessToken, nil
}
