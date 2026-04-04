package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// RecipientResolver resolves role-based recipients by calling the IAM service.
type RecipientResolver struct {
	iamBaseURL   string
	dataBaseURL  string
	actaBaseURL  string
	cyberBaseURL string
	client       *http.Client
	logger       zerolog.Logger
}

// NewRecipientResolver creates a new RecipientResolver.
func NewRecipientResolver(iamBaseURL, dataBaseURL, actaBaseURL, cyberBaseURL string, logger zerolog.Logger) *RecipientResolver {
	return &RecipientResolver{
		iamBaseURL:   iamBaseURL,
		dataBaseURL:  dataBaseURL,
		actaBaseURL:  actaBaseURL,
		cyberBaseURL: cyberBaseURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: logger.With().Str("component", "recipient_resolver").Logger(),
	}
}

// ResolveByRoles queries the IAM service for user IDs that have any of the given roles in the tenant.
func (r *RecipientResolver) ResolveByRoles(ctx context.Context, tenantID string, roles []string) ([]string, error) {
	if len(roles) == 0 {
		return nil, nil
	}

	var allUserIDs []string
	var firstErr error
	seen := make(map[string]bool)

	for _, role := range roles {
		userIDs, err := r.resolveRole(ctx, tenantID, role)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			r.logger.Warn().Err(err).Str("role", role).Str("tenant_id", tenantID).Msg("failed to resolve role")
			continue
		}
		for _, uid := range userIDs {
			if !seen[uid] {
				seen[uid] = true
				allUserIDs = append(allUserIDs, uid)
			}
		}
	}

	if firstErr != nil {
		return nil, firstErr
	}
	return allUserIDs, nil
}

func (r *RecipientResolver) resolveRole(ctx context.Context, tenantID, role string) ([]string, error) {
	endpoint := fmt.Sprintf("%s/api/v1/internal/users/by-role?tenant_id=%s&role=%s", r.iamBaseURL, tenantID, role)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-Internal-Service", "notification-service")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("IAM service returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		UserIDs []string `json:"user_ids"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.UserIDs, nil
}

// GetUserEmail queries the IAM service for a user's email address.
func (r *RecipientResolver) GetUserEmail(ctx context.Context, userID string) (string, error) {
	endpoint := fmt.Sprintf("%s/api/v1/internal/users/%s/email", r.iamBaseURL, userID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-Internal-Service", "notification-service")

	resp, err := r.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("IAM service returned %d", resp.StatusCode)
	}

	var result struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return result.Email, nil
}

func (r *RecipientResolver) ResolveComputedRecipients(ctx context.Context, tenantID, computation string, data map[string]interface{}) ([]string, error) {
	switch computation {
	case "asset_owners_from_event":
		assetIDs := extractStringList(data, "affected_asset_ids", "asset_ids")
		if len(assetIDs) == 0 {
			if assetID := extractFirstString(data, "asset_id"); assetID != "" {
				assetIDs = append(assetIDs, assetID)
			}
		}
		if len(assetIDs) == 0 {
			return nil, nil
		}
		query := url.Values{}
		query.Set("tenant_id", tenantID)
		for _, assetID := range assetIDs {
			query.Add("asset_id", assetID)
		}
		var resp struct {
			UserIDs []string `json:"user_ids"`
		}
		if err := r.getJSON(ctx, r.cyberBaseURL, "/api/v1/internal/assets/owners", query, &resp); err != nil {
			return nil, err
		}
		return uniqueStrings(resp.UserIDs), nil
	case "pipeline_owner_from_event":
		pipelineID := extractFirstString(data, "pipeline_id")
		if pipelineID == "" {
			return nil, nil
		}
		query := url.Values{}
		query.Set("tenant_id", tenantID)
		query.Set("pipeline_id", pipelineID)
		var resp struct {
			UserID string `json:"user_id"`
		}
		if err := r.getJSON(ctx, r.dataBaseURL, "/api/v1/internal/pipeline-owner", query, &resp); err != nil {
			return nil, err
		}
		if strings.TrimSpace(resp.UserID) == "" {
			return nil, nil
		}
		return []string{resp.UserID}, nil
	case "committee_members_from_event":
		committeeID := extractFirstString(data, "committee_id")
		if committeeID == "" {
			return nil, nil
		}
		query := url.Values{}
		query.Set("tenant_id", tenantID)
		query.Set("committee_id", committeeID)
		var resp struct {
			UserIDs []string `json:"user_ids"`
		}
		if err := r.getJSON(ctx, r.actaBaseURL, "/api/v1/internal/committee-members", query, &resp); err != nil {
			return nil, err
		}
		return uniqueStrings(resp.UserIDs), nil
	case "meeting_attendees_from_event":
		meetingID := extractFirstString(data, "meeting_id")
		if meetingID == "" {
			return nil, nil
		}
		query := url.Values{}
		query.Set("tenant_id", tenantID)
		query.Set("meeting_id", meetingID)
		var resp struct {
			UserIDs []string `json:"user_ids"`
		}
		if err := r.getJSON(ctx, r.actaBaseURL, "/api/v1/internal/meeting-attendees", query, &resp); err != nil {
			return nil, err
		}
		return uniqueStrings(resp.UserIDs), nil
	default:
		return nil, fmt.Errorf("unsupported computed recipient resolution: %s", computation)
	}
}

func (r *RecipientResolver) getJSON(ctx context.Context, baseURL, path string, query url.Values, target any) error {
	if strings.TrimSpace(baseURL) == "" {
		return fmt.Errorf("base URL is required for internal lookup %s", path)
	}

	endpoint := strings.TrimRight(baseURL, "/") + path
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-Internal-Service", "notification-service")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("internal service returned %d: %s", resp.StatusCode, string(body))
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func extractFirstString(data map[string]interface{}, fields ...string) string {
	for _, field := range fields {
		if value, ok := data[field]; ok {
			if str := stringValue(value); strings.TrimSpace(str) != "" {
				return str
			}
		}
	}
	return ""
}

func extractStringList(data map[string]interface{}, fields ...string) []string {
	var out []string
	for _, field := range fields {
		value, ok := data[field]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				out = append(out, typed)
			}
		case []string:
			out = append(out, typed...)
		case []interface{}:
			for _, item := range typed {
				if str := stringValue(item); strings.TrimSpace(str) != "" {
					out = append(out, str)
				}
			}
		}
	}
	return uniqueStrings(out)
}
