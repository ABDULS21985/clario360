package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// RecipientResolver resolves role-based recipients by calling the IAM service.
type RecipientResolver struct {
	iamBaseURL string
	client     *http.Client
	logger     zerolog.Logger
}

// NewRecipientResolver creates a new RecipientResolver.
func NewRecipientResolver(iamBaseURL string, logger zerolog.Logger) *RecipientResolver {
	return &RecipientResolver{
		iamBaseURL: iamBaseURL,
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
	seen := make(map[string]bool)

	for _, role := range roles {
		userIDs, err := r.resolveRole(ctx, tenantID, role)
		if err != nil {
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

	return allUserIDs, nil
}

func (r *RecipientResolver) resolveRole(ctx context.Context, tenantID, role string) ([]string, error) {
	url := fmt.Sprintf("%s/api/v1/internal/users/by-role?tenant_id=%s&role=%s", r.iamBaseURL, tenantID, role)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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
	url := fmt.Sprintf("%s/api/v1/internal/users/%s/email", r.iamBaseURL, userID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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
