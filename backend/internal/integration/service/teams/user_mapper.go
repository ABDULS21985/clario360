package teams

import (
	"context"
	"fmt"
	"strings"

	iamdto "github.com/clario360/platform/internal/iam/dto"
)

type LookupFunc func(ctx context.Context, tenantID, email string) (*iamdto.UserResponse, error)

func MapTeamsUser(ctx context.Context, tenantID string, activity map[string]any, lookup LookupFunc) (*iamdto.UserResponse, error) {
	if lookup == nil {
		return nil, fmt.Errorf("teams lookup func is required")
	}
	email := extractTeamsEmail(activity)
	if email == "" {
		return nil, fmt.Errorf("teams activity did not include a user email")
	}
	return lookup(ctx, tenantID, email)
}

func extractTeamsEmail(activity map[string]any) string {
	for _, path := range [][]string{
		{"from", "userPrincipalName"},
		{"from", "email"},
		{"from", "aadObjectId"},
		{"channelData", "email"},
	} {
		if value := nestedString(activity, path...); strings.Contains(value, "@") {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func nestedString(payload map[string]any, path ...string) string {
	current := payload
	for idx, key := range path {
		value, ok := current[key]
		if !ok {
			return ""
		}
		if idx == len(path)-1 {
			if str, ok := value.(string); ok {
				return str
			}
			return ""
		}
		next, ok := value.(map[string]any)
		if !ok {
			return ""
		}
		current = next
	}
	return ""
}
