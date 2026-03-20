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

type DataConnectorCollector struct {
	platformDB *pgxpool.Pool
	logger     zerolog.Logger
}

func NewDataConnectorCollector(platformDB *pgxpool.Pool, logger zerolog.Logger) *DataConnectorCollector {
	return &DataConnectorCollector{
		platformDB: platformDB,
		logger:     logger.With().Str("component", "ueba-data-collector").Logger(),
	}
}

func (c *DataConnectorCollector) Collect(ctx context.Context, tenantID uuid.UUID, since time.Time, limit int) ([]*model.DataAccessEvent, error) {
	if c.platformDB == nil || limit <= 0 {
		return nil, nil
	}
	rows, err := c.platformDB.Query(ctx, `
		SELECT user_id, COALESCE(ip_address::text, ''), COALESCE(user_agent, ''), metadata, created_at
		FROM audit_logs
		WHERE tenant_id = $1
		  AND service = 'data-service'
		  AND action = 'data.access.event.collected'
		  AND created_at > $2
		ORDER BY created_at ASC
		LIMIT $3`,
		tenantID, since, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("collect ueba data connector events: %w", err)
	}
	defer rows.Close()
	items := make([]*model.DataAccessEvent, 0, limit)
	for rows.Next() {
		var (
			userID    *string
			ipAddress string
			userAgent string
			metadata  []byte
			createdAt time.Time
		)
		if err := rows.Scan(&userID, &ipAddress, &userAgent, &metadata, &createdAt); err != nil {
			return nil, err
		}
		meta := decodeMetadata(metadata)
		entityID := stringValue(meta, "user")
		if entityID == "" && userID != nil {
			entityID = *userID
		}
		if entityID == "" {
			continue
		}
		action := strings.ToLower(strings.TrimSpace(stringValue(meta, "action")))
		if action == "" {
			action = "select"
		}
		tableName := stringValue(meta, "table")
		schemaName := ""
		if dot := strings.Index(tableName, "."); dot > 0 {
			schemaName = tableName[:dot]
			tableName = tableName[dot+1:]
		}
		sourceID := parseOptionalUUID(stringValue(meta, "source_id"))
		eventTime := createdAt
		if ts := stringValue(meta, "timestamp"); ts != "" {
			if parsed, err := time.Parse(time.RFC3339Nano, ts); err == nil {
				eventTime = parsed.UTC()
			}
		}
		items = append(items, &model.DataAccessEvent{
			ID:             uuid.New(),
			TenantID:       tenantID,
			EntityType:     inferEntityType(entityID),
			EntityID:       entityID,
			SourceType:     stringValue(meta, "source_type"),
			SourceID:       sourceID,
			Action:         mapAccessAction(action),
			DatabaseName:   stringValue(meta, "database"),
			SchemaName:     schemaName,
			TableName:      tableName,
			QueryHash:      stringValue(meta, "query_hash"),
			RowsAccessed:   maxInt64(int64Value(meta, "rows_read"), int64Value(meta, "rows_written")),
			BytesAccessed:  maxInt64(int64Value(meta, "bytes_read"), int64Value(meta, "bytes_written")),
			DurationMS:     int(int64Value(meta, "duration_ms")),
			SourceIP:       firstNonEmpty(stringValue(meta, "source_ip"), ipAddress),
			UserAgent:      firstNonEmpty(stringValue(meta, "user_agent"), userAgent),
			Success:        boolValue(meta, "success", true),
			ErrorMessage:   stringValue(meta, "error_message"),
			EventTimestamp: eventTime,
			QueryPreview:   stringValue(meta, "query_preview"),
		})
	}
	return items, rows.Err()
}
