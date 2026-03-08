package service

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

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	iamdto "github.com/clario360/platform/internal/iam/dto"
)

type ClarioAPIClient struct {
	httpClient *http.Client
	gatewayURL string
	iamURL     string
	jwtMgr     *auth.JWTManager
	logger     zerolog.Logger
}

func NewClarioAPIClient(gatewayURL, iamURL string, jwtMgr *auth.JWTManager, logger zerolog.Logger) *ClarioAPIClient {
	return &ClarioAPIClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		gatewayURL: strings.TrimRight(gatewayURL, "/"),
		iamURL:     strings.TrimRight(iamURL, "/"),
		jwtMgr:     jwtMgr,
		logger:     logger.With().Str("component", "integration_clario_client").Logger(),
	}
}

func (c *ClarioAPIClient) LookupUserByEmail(ctx context.Context, tenantID, email string) (*iamdto.UserResponse, error) {
	params := url.Values{}
	params.Set("email", email)
	if tenantID != "" {
		params.Set("tenant_id", tenantID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.iamURL+"/api/v1/internal/users/by-email?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("build user lookup request: %w", err)
	}
	req.Header.Set("X-Internal-Service", "notification-service")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lookup user by email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("lookup user by email returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var user iamdto.UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode user lookup response: %w", err)
	}
	return &user, nil
}

func (c *ClarioAPIClient) LookupUserEmail(ctx context.Context, userID string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/v1/internal/users/%s/email", c.iamURL, userID), nil)
	if err != nil {
		return "", fmt.Errorf("build user email request: %w", err)
	}
	req.Header.Set("X-Internal-Service", "notification-service")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("lookup user email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("lookup user email returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode user email response: %w", err)
	}
	return payload.Email, nil
}

func (c *ClarioAPIClient) MintUserToken(user *iamdto.UserResponse) (string, error) {
	if user == nil {
		return "", fmt.Errorf("user is required")
	}
	roles := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		roles = append(roles, role.Slug)
	}
	pair, err := c.jwtMgr.GenerateTokenPair(user.ID, user.TenantID, user.Email, roles)
	if err != nil {
		return "", fmt.Errorf("mint user token: %w", err)
	}
	return pair.AccessToken, nil
}

func (c *ClarioAPIClient) MintSystemToken(tenantID string) (string, error) {
	pair, err := c.jwtMgr.GenerateTokenPair("00000000-0000-0000-0000-000000000000", tenantID, "integrations@clario360.local", []string{"tenant-admin"})
	if err != nil {
		return "", fmt.Errorf("mint system token: %w", err)
	}
	return pair.AccessToken, nil
}

func (c *ClarioAPIClient) GatewayRequest(ctx context.Context, method, path, token string, body any, out any) (int, []byte, error) {
	var payload io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, fmt.Errorf("marshal gateway body: %w", err)
		}
		payload = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.gatewayURL+path, payload)
	if err != nil {
		return 0, nil, fmt.Errorf("build gateway request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("gateway request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("read gateway response: %w", err)
	}

	if out != nil && len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, out); err != nil {
			return resp.StatusCode, bodyBytes, fmt.Errorf("decode gateway response: %w", err)
		}
	}
	if resp.StatusCode >= 400 {
		return resp.StatusCode, bodyBytes, fmt.Errorf("gateway returned status %d", resp.StatusCode)
	}
	return resp.StatusCode, bodyBytes, nil
}

func (c *ClarioAPIClient) FetchEntity(ctx context.Context, token, entityType, entityID string) (map[string]any, error) {
	path, err := entityPath(entityType, entityID)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	if _, _, err := c.GatewayRequest(ctx, http.MethodGet, path, token, nil, &envelope); err != nil {
		return nil, err
	}
	if envelope.Data == nil {
		return map[string]any{}, nil
	}
	return envelope.Data, nil
}

func (c *ClarioAPIClient) UpdateAlertStatus(ctx context.Context, token, alertID, status string, notes, reason *string) error {
	payload := map[string]any{"status": status}
	if notes != nil {
		payload["notes"] = *notes
	}
	if reason != nil {
		payload["reason"] = *reason
	}
	_, _, err := c.GatewayRequest(ctx, http.MethodPut, "/api/v1/cyber/alerts/"+alertID+"/status", token, payload, nil)
	return err
}

func (c *ClarioAPIClient) AssignAlert(ctx context.Context, token, alertID, assignedTo string) error {
	payload := map[string]any{"assigned_to": assignedTo}
	_, _, err := c.GatewayRequest(ctx, http.MethodPut, "/api/v1/cyber/alerts/"+alertID+"/assign", token, payload, nil)
	return err
}

func (c *ClarioAPIClient) AddAlertComment(ctx context.Context, token, alertID, content string, metadata map[string]any) error {
	payload := map[string]any{"content": content}
	if metadata != nil {
		payload["metadata"] = metadata
	}
	_, _, err := c.GatewayRequest(ctx, http.MethodPost, "/api/v1/cyber/alerts/"+alertID+"/comments", token, payload, nil)
	return err
}

func entityPath(entityType, entityID string) (string, error) {
	switch entityType {
	case "alert":
		return "/api/v1/cyber/alerts/" + entityID, nil
	case "remediation":
		return "/api/v1/cyber/remediation/" + entityID, nil
	case "action_item":
		return "/api/v1/acta/action-items/" + entityID, nil
	case "contract":
		return "/api/v1/lex/contracts/" + entityID, nil
	default:
		return "", fmt.Errorf("unsupported entity type %q", entityType)
	}
}
