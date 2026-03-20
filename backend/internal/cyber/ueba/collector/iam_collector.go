package collector

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

type IAMCollector struct {
	platformDB *pgxpool.Pool
	logger     zerolog.Logger
}

func NewIAMCollector(platformDB *pgxpool.Pool, logger zerolog.Logger) *IAMCollector {
	return &IAMCollector{
		platformDB: platformDB,
		logger:     logger.With().Str("component", "ueba-iam-collector").Logger(),
	}
}

func (c *IAMCollector) Collect(ctx context.Context, tenantID uuid.UUID, since time.Time, limit int) ([]*model.DataAccessEvent, error) {
	if c.platformDB == nil || limit <= 0 {
		return nil, nil
	}
	items := make([]*model.DataAccessEvent, 0, limit)

	rows, err := c.platformDB.Query(ctx, `
		SELECT COALESCE(user_id::text, ''), COALESCE(ip_address::text, ''), COALESCE(user_agent, ''), action, created_at
		FROM audit_logs
		WHERE tenant_id = $1
		  AND service = 'iam-service'
		  AND action ~ '(login|logout)'
		  AND created_at > $2
		ORDER BY created_at ASC
		LIMIT $3`,
		tenantID, since, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("collect ueba iam audit events: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var userID, ipAddress, userAgent, action string
		var createdAt time.Time
		if err := rows.Scan(&userID, &ipAddress, &userAgent, &action, &createdAt); err != nil {
			return nil, err
		}
		mappedAction := "login"
		success := true
		if strings.Contains(action, "logout") {
			mappedAction = "logout"
		}
		if strings.Contains(action, "failure") {
			success = false
		}
		items = append(items, &model.DataAccessEvent{
			ID:             uuid.New(),
			TenantID:       tenantID,
			EntityType:     model.EntityTypeUser,
			EntityID:       userID,
			SourceType:     "iam",
			Action:         mappedAction,
			QueryHash:      syntheticIAMHash(userID, mappedAction, ipAddress, createdAt),
			SourceIP:       ipAddress,
			UserAgent:      userAgent,
			Success:        success,
			EventTimestamp: createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(items) >= limit {
		return items, nil
	}
	remaining := limit - len(items)
	sessionRows, err := c.platformDB.Query(ctx, `
		SELECT user_id::text, COALESCE(ip_address::text, ''), COALESCE(user_agent, ''), created_at
		FROM sessions
		WHERE tenant_id = $1
		  AND created_at > $2
		ORDER BY created_at ASC
		LIMIT $3`,
		tenantID, since, remaining,
	)
	if err != nil {
		return items, nil
	}
	defer sessionRows.Close()
	for sessionRows.Next() {
		var userID, ipAddress, userAgent string
		var createdAt time.Time
		if err := sessionRows.Scan(&userID, &ipAddress, &userAgent, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, &model.DataAccessEvent{
			ID:             uuid.New(),
			TenantID:       tenantID,
			EntityType:     model.EntityTypeUser,
			EntityID:       userID,
			SourceType:     "iam",
			Action:         "login",
			QueryHash:      syntheticIAMHash(userID, "login", ipAddress, createdAt),
			SourceIP:       ipAddress,
			UserAgent:      userAgent,
			Success:        true,
			EventTimestamp: createdAt,
		})
	}
	return items, sessionRows.Err()
}

func syntheticIAMHash(userID, action, ip string, ts time.Time) string {
	sum := sha256.Sum256([]byte(userID + "|" + action + "|" + ip + "|" + ts.UTC().Format(time.RFC3339Nano)))
	return hex.EncodeToString(sum[:])
}
