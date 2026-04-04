package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// ActivityEntry represents a single asset activity record.
type ActivityEntry struct {
	ID          uuid.UUID       `json:"id"`
	TenantID    uuid.UUID       `json:"-"`
	AssetID     uuid.UUID       `json:"alert_id"` // mapped as alert_id for frontend compat
	Action      string          `json:"action"`
	ActorID     *uuid.UUID      `json:"actor_id,omitempty"`
	ActorName   string          `json:"actor_name,omitempty"`
	Description string          `json:"description"`
	OldValue    *string         `json:"old_value,omitempty"`
	NewValue    *string         `json:"new_value,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// ActivityRepository handles the asset_activity table.
type ActivityRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewActivityRepository creates a new ActivityRepository.
func NewActivityRepository(db *pgxpool.Pool, logger zerolog.Logger) *ActivityRepository {
	return &ActivityRepository{db: db, logger: logger}
}

// Insert records a new activity entry.
func (r *ActivityRepository) Insert(ctx context.Context, entry *ActivityEntry) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	meta := entry.Metadata
	if meta == nil {
		meta = json.RawMessage("{}")
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO asset_activity (id, tenant_id, asset_id, action, actor_id, actor_name, description, old_value, new_value, metadata, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		entry.ID, entry.TenantID, entry.AssetID, entry.Action,
		entry.ActorID, entry.ActorName, entry.Description,
		entry.OldValue, entry.NewValue, meta, entry.CreatedAt,
	)
	return err
}

// ListByAsset returns recent activity for an asset, newest first.
func (r *ActivityRepository) ListByAsset(ctx context.Context, tenantID, assetID uuid.UUID, limit int) ([]*ActivityEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT id, tenant_id, asset_id, action, actor_id, actor_name,
		       description, old_value, new_value, metadata, created_at
		FROM asset_activity
		WHERE tenant_id = $1 AND asset_id = $2
		ORDER BY created_at DESC
		LIMIT %d`, limit),
		tenantID, assetID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*ActivityEntry
	for rows.Next() {
		var e ActivityEntry
		if err := rows.Scan(
			&e.ID, &e.TenantID, &e.AssetID, &e.Action,
			&e.ActorID, &e.ActorName, &e.Description,
			&e.OldValue, &e.NewValue, &e.Metadata, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		entries = append(entries, &e)
	}
	return entries, rows.Err()
}
