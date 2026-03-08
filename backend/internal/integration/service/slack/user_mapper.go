package slack

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	iamdto "github.com/clario360/platform/internal/iam/dto"
)

type UserLookupFunc func(ctx context.Context, tenantID, email string) (*iamdto.UserResponse, error)

type UserMapper struct {
	client *Client
	redis  *redis.Client
	logger zerolog.Logger
}

func NewUserMapper(client *Client, redis *redis.Client, logger zerolog.Logger) *UserMapper {
	return &UserMapper{
		client: client,
		redis:  redis,
		logger: logger.With().Str("component", "slack_user_mapper").Logger(),
	}
}

func (m *UserMapper) MapSlackUser(ctx context.Context, botToken, tenantID, teamID, slackUserID string, lookup UserLookupFunc) (*iamdto.UserResponse, error) {
	if lookup == nil {
		return nil, fmt.Errorf("user lookup is required")
	}
	cacheKey := fmt.Sprintf("slack_user_map:%s:%s", teamID, slackUserID)
	if m.redis != nil {
		if cachedEmail, err := m.redis.Get(ctx, cacheKey).Result(); err == nil && cachedEmail != "" {
			return lookup(ctx, tenantID, cachedEmail)
		}
	}

	userInfo, err := m.client.UsersInfo(ctx, botToken, slackUserID)
	if err != nil {
		return nil, err
	}
	userSection, _ := userInfo["user"].(map[string]any)
	profile, _ := userSection["profile"].(map[string]any)
	email := strings.TrimSpace(stringValue(profile["email"]))
	if email == "" {
		return nil, fmt.Errorf("slack user email is unavailable")
	}
	user, err := lookup(ctx, tenantID, email)
	if err != nil {
		return nil, err
	}
	if m.redis != nil {
		_ = m.redis.Set(ctx, cacheKey, email, time.Hour).Err()
	}
	return user, nil
}
