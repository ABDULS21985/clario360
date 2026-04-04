package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

type sourceCollector interface {
	Collect(ctx context.Context, tenantID uuid.UUID, since time.Time, limit int) ([]*model.DataAccessEvent, error)
}

type AccessEventCollector struct {
	platformDB     *pgxpool.Pool
	redis          *redis.Client
	deduplicator   *Deduplicator
	auditCollector sourceCollector
	dataCollector  sourceCollector
	iamCollector   sourceCollector
	logger         zerolog.Logger
}

func New(
	platformDB *pgxpool.Pool,
	redisClient *redis.Client,
	eventRepo dedupRepository,
	logger zerolog.Logger,
) *AccessEventCollector {
	return &AccessEventCollector{
		platformDB:     platformDB,
		redis:          redisClient,
		deduplicator:   NewDeduplicator(eventRepo),
		auditCollector: NewAuditLogCollector(platformDB, logger),
		dataCollector:  NewDataConnectorCollector(platformDB, logger),
		iamCollector:   NewIAMCollector(platformDB, logger),
		logger:         logger.With().Str("component", "ueba-collector").Logger(),
	}
}

func (c *AccessEventCollector) CollectSinceLastRun(ctx context.Context, tenantID uuid.UUID, maxEvents int) ([]*model.DataAccessEvent, error) {
	if maxEvents <= 0 {
		maxEvents = 10000
	}
	c.deduplicator.Reset()
	since, err := c.LastRun(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	collectors := []sourceCollector{c.dataCollector, c.iamCollector, c.auditCollector}
	events := make([]*model.DataAccessEvent, 0, maxEvents)
	for _, instance := range collectors {
		remaining := maxEvents - len(events)
		if remaining <= 0 {
			break
		}
		items, err := instance.Collect(ctx, tenantID, since, remaining)
		if err != nil {
			return nil, err
		}
		for _, event := range items {
			duplicate, err := c.deduplicator.IsDuplicate(ctx, tenantID, event)
			if err != nil {
				return nil, err
			}
			if duplicate {
				continue
			}
			events = append(events, event)
			if len(events) == maxEvents {
				break
			}
		}
	}
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].EventTimestamp.Before(events[j].EventTimestamp)
	})
	return events, nil
}

func (c *AccessEventCollector) LastRun(ctx context.Context, tenantID uuid.UUID) (time.Time, error) {
	if c.redis == nil {
		return time.Now().UTC().Add(-6 * time.Hour), nil
	}
	value, err := c.redis.Get(ctx, cursorKey(tenantID)).Result()
	if err != nil {
		if err == redis.Nil {
			return time.Now().UTC().Add(-6 * time.Hour), nil
		}
		return time.Time{}, fmt.Errorf("load ueba collector cursor: %w", err)
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse ueba collector cursor: %w", err)
	}
	return parsed.UTC(), nil
}

func (c *AccessEventCollector) MarkRun(ctx context.Context, tenantID uuid.UUID, ts time.Time) error {
	if c.redis == nil || ts.IsZero() {
		return nil
	}
	return c.redis.Set(ctx, cursorKey(tenantID), ts.UTC().Format(time.RFC3339Nano), 30*24*time.Hour).Err()
}

func (c *AccessEventCollector) ListTenantIDs(ctx context.Context) ([]uuid.UUID, error) {
	if c.platformDB == nil {
		return nil, nil
	}
	rows, err := c.platformDB.Query(ctx, `
		SELECT tenant_id FROM (
			SELECT DISTINCT tenant_id FROM audit_logs WHERE created_at >= now() - interval '30 days'
			UNION
			SELECT DISTINCT tenant_id FROM sessions WHERE created_at >= now() - interval '30 days'
		) tenants`)
	if err != nil {
		return nil, fmt.Errorf("list collector tenant ids: %w", err)
	}
	defer rows.Close()
	items := make([]uuid.UUID, 0)
	for rows.Next() {
		var tenantID uuid.UUID
		if err := rows.Scan(&tenantID); err != nil {
			return nil, err
		}
		items = append(items, tenantID)
	}
	return items, rows.Err()
}

func cursorKey(tenantID uuid.UUID) string {
	return "cyber:ueba:cursor:" + tenantID.String()
}

func decodeMetadata(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func inferEntityType(value string) model.EntityType {
	switch {
	case value == "":
		return model.EntityTypeUser
	case len(value) > 4 && value[:4] == "key_":
		return model.EntityTypeAPIKey
	case containsInsensitive(value, "svc") || containsInsensitive(value, "service"):
		return model.EntityTypeServiceAccount
	case containsInsensitive(value, "app"):
		return model.EntityTypeApplication
	default:
		return model.EntityTypeUser
	}
}

func containsInsensitive(value, needle string) bool {
	return len(value) >= len(needle) && (len(needle) == 0 || indexInsensitive(value, needle) >= 0)
}

func indexInsensitive(value, needle string) int {
	for i := 0; i+len(needle) <= len(value); i++ {
		if equalFoldASCII(value[i:i+len(needle)], needle) {
			return i
		}
	}
	return -1
}

func equalFoldASCII(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := 0; i < len(left); i++ {
		l := left[i]
		r := right[i]
		if l >= 'A' && l <= 'Z' {
			l += 'a' - 'A'
		}
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		if l != r {
			return false
		}
	}
	return true
}

func stringValue(meta map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := meta[key]; ok {
			switch typed := value.(type) {
			case string:
				return typed
			case fmt.Stringer:
				return typed.String()
			}
		}
	}
	return ""
}

func int64Value(meta map[string]any, keys ...string) int64 {
	for _, key := range keys {
		if value, ok := meta[key]; ok {
			switch typed := value.(type) {
			case float64:
				return int64(typed)
			case int64:
				return typed
			case int:
				return int64(typed)
			}
		}
	}
	return 0
}

func boolValue(meta map[string]any, key string, defaultValue bool) bool {
	value, ok := meta[key]
	if !ok {
		return defaultValue
	}
	typed, ok := value.(bool)
	if !ok {
		return defaultValue
	}
	return typed
}
