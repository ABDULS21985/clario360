package collector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

type AuditLogCollector struct {
	platformDB *pgxpool.Pool
	logger     zerolog.Logger
}

func NewAuditLogCollector(platformDB *pgxpool.Pool, logger zerolog.Logger) *AuditLogCollector {
	return &AuditLogCollector{
		platformDB: platformDB,
		logger:     logger.With().Str("component", "ueba-audit-collector").Logger(),
	}
}

func (c *AuditLogCollector) Collect(ctx context.Context, tenantID uuid.UUID, since time.Time, limit int) ([]*model.DataAccessEvent, error) {
	if c.platformDB == nil || limit <= 0 {
		return nil, nil
	}
	rows, err := c.platformDB.Query(ctx, `
		SELECT COALESCE(user_id::text, ''), COALESCE(ip_address::text, ''), COALESCE(user_agent, ''), action, resource_type, COALESCE(resource_id::text, ''), metadata, created_at
		FROM audit_logs
		WHERE tenant_id = $1
		  AND created_at > $2
		  AND service IN ('cyber-service', 'data-service')
		  AND (
			action ILIKE '%export%' OR
			action ILIKE '%download%' OR
			action ILIKE '%api%' OR
			COALESCE(metadata ->> 'action', '') IN ('export', 'download', 'api_call')
		  )
		ORDER BY created_at ASC
		LIMIT $3`,
		tenantID, since, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("collect ueba audit log events: %w", err)
	}
	defer rows.Close()
	items := make([]*model.DataAccessEvent, 0, limit)
	for rows.Next() {
		var userID, ipAddress, userAgent, action, resourceType, resourceID string
		var metadata []byte
		var createdAt time.Time
		if err := rows.Scan(&userID, &ipAddress, &userAgent, &action, &resourceType, &resourceID, &metadata, &createdAt); err != nil {
			return nil, err
		}
		meta := decodeMetadata(metadata)
		entityID := firstNonEmpty(userID, stringValue(meta, "user_id", "user"))
		if entityID == "" {
			continue
		}
		tableName := stringValue(meta, "table", "table_name")
		schemaName := stringValue(meta, "schema", "schema_name")
		if schemaName == "" && strings.Contains(tableName, ".") {
			schemaName = tableName[:strings.Index(tableName, ".")]
			tableName = tableName[strings.Index(tableName, ".")+1:]
		}
		items = append(items, &model.DataAccessEvent{
			ID:             uuid.New(),
			TenantID:       tenantID,
			EntityType:     inferEntityType(entityID),
			EntityID:       entityID,
			SourceType:     "audit_log",
			Action:         mapAuditAction(action, meta),
			DatabaseName:   stringValue(meta, "database", "database_name", "source_name"),
			SchemaName:     schemaName,
			TableName:      tableName,
			QueryHash:      firstNonEmpty(stringValue(meta, "query_hash"), resourceID),
			BytesAccessed:  int64Value(meta, "bytes_accessed", "bytes_read", "bytes_written"),
			RowsAccessed:   int64Value(meta, "rows_accessed", "rows_read", "rows_written"),
			DurationMS:     int(int64Value(meta, "duration_ms")),
			SourceIP:       firstNonEmpty(ipAddress, stringValue(meta, "source_ip")),
			UserAgent:      firstNonEmpty(userAgent, stringValue(meta, "user_agent")),
			Success:        boolValue(meta, "success", true),
			ErrorMessage:   stringValue(meta, "error_message"),
			EventTimestamp: createdAt,
			QueryPreview:   stringValue(meta, "query_preview"),
		})
	}
	return items, rows.Err()
}

func mapAccessAction(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "select", "insert", "update", "delete", "login", "logout", "export", "download", "api_call":
		return strings.ToLower(strings.TrimSpace(action))
	case "create", "alter", "drop":
		return strings.ToLower(strings.TrimSpace(action))
	default:
		return "select"
	}
}

func mapAuditAction(action string, meta map[string]any) string {
	override := stringValue(meta, "action")
	if override != "" {
		return mapAccessAction(override)
	}
	lower := strings.ToLower(action)
	switch {
	case strings.Contains(lower, "download"):
		return "download"
	case strings.Contains(lower, "export"):
		return "export"
	case strings.Contains(lower, "api"):
		return "api_call"
	default:
		return "select"
	}
}

func parseOptionalUUID(value string) *uuid.UUID {
	if value == "" {
		return nil
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return nil
	}
	return &parsed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
