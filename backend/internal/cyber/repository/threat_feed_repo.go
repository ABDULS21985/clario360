package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

type ThreatFeedRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewThreatFeedRepository(db *pgxpool.Pool, logger zerolog.Logger) *ThreatFeedRepository {
	return &ThreatFeedRepository{db: db, logger: logger}
}

func (r *ThreatFeedRepository) Create(ctx context.Context, item *model.ThreatFeedConfig) (*model.ThreatFeedConfig, error) {
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `
		INSERT INTO threat_feed_configs (
			id, tenant_id, name, type, url, auth_type, auth_config, sync_interval,
			default_severity, default_confidence, default_tags, indicator_types,
			enabled, status, last_sync_at, last_sync_status, last_error, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18, $19, $19
		)`,
		item.ID, item.TenantID, item.Name, item.Type, item.URL, item.AuthType, ensureRawMessage(item.AuthConfig, "{}"),
		item.SyncInterval, item.DefaultSeverity, item.DefaultConfidence, item.DefaultTags, item.IndicatorTypes,
		item.Enabled, item.Status, item.LastSyncAt, item.LastSyncStatus, item.LastError, item.CreatedBy, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create threat feed config: %w", err)
	}
	return r.GetByID(ctx, item.TenantID, item.ID)
}

func (r *ThreatFeedRepository) Update(ctx context.Context, item *model.ThreatFeedConfig) (*model.ThreatFeedConfig, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE threat_feed_configs
		SET
			name = $3,
			type = $4,
			url = $5,
			auth_type = $6,
			auth_config = $7,
			sync_interval = $8,
			default_severity = $9,
			default_confidence = $10,
			default_tags = $11,
			indicator_types = $12,
			enabled = $13,
			status = $14,
			updated_at = now()
		WHERE tenant_id = $1 AND id = $2`,
		item.TenantID, item.ID, item.Name, item.Type, item.URL, item.AuthType, ensureRawMessage(item.AuthConfig, "{}"),
		item.SyncInterval, item.DefaultSeverity, item.DefaultConfidence, item.DefaultTags, item.IndicatorTypes,
		item.Enabled, item.Status,
	)
	if err != nil {
		return nil, fmt.Errorf("update threat feed config: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, item.TenantID, item.ID)
}

func (r *ThreatFeedRepository) GetByID(ctx context.Context, tenantID, feedID uuid.UUID) (*model.ThreatFeedConfig, error) {
	row := r.db.QueryRow(ctx, `
		SELECT
			id, tenant_id, name, type, url, auth_type, auth_config, sync_interval,
			default_severity, default_confidence, default_tags, indicator_types,
			enabled, status, last_sync_at, last_sync_status, last_error, created_by, created_at, updated_at
		FROM threat_feed_configs
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, feedID,
	)
	item, err := scanThreatFeedConfig(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get threat feed config: %w", err)
	}
	return item, nil
}

func (r *ThreatFeedRepository) List(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]*model.ThreatFeedConfig, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 25
	}
	if perPage > 200 {
		perPage = 200
	}

	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM threat_feed_configs
		WHERE tenant_id = $1`,
		tenantID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count threat feed configs: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT
			id, tenant_id, name, type, url, auth_type, auth_config, sync_interval,
			default_severity, default_confidence, default_tags, indicator_types,
			enabled, status, last_sync_at, last_sync_status, last_error, created_by, created_at, updated_at
		FROM threat_feed_configs
		WHERE tenant_id = $1
		ORDER BY updated_at DESC, name ASC
		LIMIT $2 OFFSET $3`,
		tenantID, perPage, (page-1)*perPage,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list threat feed configs: %w", err)
	}
	defer rows.Close()

	items := make([]*model.ThreatFeedConfig, 0, perPage)
	for rows.Next() {
		item, err := scanThreatFeedConfig(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *ThreatFeedRepository) AppendHistory(ctx context.Context, item *model.ThreatFeedSyncHistory) error {
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO threat_feed_sync_history (
			id, tenant_id, feed_id, status, indicators_parsed, indicators_imported,
			indicators_skipped, indicators_failed, duration_ms, error_message, metadata,
			started_at, completed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13
		)`,
		item.ID, item.TenantID, item.FeedID, item.Status, item.IndicatorsParsed, item.IndicatorsImported,
		item.IndicatorsSkipped, item.IndicatorsFailed, item.DurationMs, item.ErrorMessage, ensureRawMessage(item.Metadata, "{}"),
		item.StartedAt, item.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("append threat feed history: %w", err)
	}
	return nil
}

func (r *ThreatFeedRepository) UpdateSyncState(ctx context.Context, tenantID, feedID uuid.UUID, status model.ThreatFeedStatus, lastSyncStatus string, lastError *string, syncedAt time.Time) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE threat_feed_configs
		SET
			status = $3,
			last_sync_at = $4,
			last_sync_status = $5,
			last_error = $6,
			updated_at = now()
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, feedID, status, syncedAt, lastSyncStatus, lastError,
	)
	if err != nil {
		return fmt.Errorf("update threat feed sync state: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ThreatFeedRepository) ListHistory(ctx context.Context, tenantID, feedID uuid.UUID, limit int) ([]*model.ThreatFeedSyncHistory, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.Query(ctx, `
		SELECT
			id, tenant_id, feed_id, status, indicators_parsed, indicators_imported,
			indicators_skipped, indicators_failed, duration_ms, error_message, metadata,
			started_at, completed_at
		FROM threat_feed_sync_history
		WHERE tenant_id = $1 AND feed_id = $2
		ORDER BY started_at DESC
		LIMIT $3`,
		tenantID, feedID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list threat feed history: %w", err)
	}
	defer rows.Close()

	items := make([]*model.ThreatFeedSyncHistory, 0, limit)
	for rows.Next() {
		item, err := scanThreatFeedSyncHistory(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanThreatFeedConfig(row scanner) (*model.ThreatFeedConfig, error) {
	var item model.ThreatFeedConfig
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.Name,
		&item.Type,
		&item.URL,
		&item.AuthType,
		&item.AuthConfig,
		&item.SyncInterval,
		&item.DefaultSeverity,
		&item.DefaultConfidence,
		&item.DefaultTags,
		&item.IndicatorTypes,
		&item.Enabled,
		&item.Status,
		&item.LastSyncAt,
		&item.LastSyncStatus,
		&item.LastError,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.AuthConfig = ensureRawMessage(item.AuthConfig, "{}")
	if item.DefaultTags == nil {
		item.DefaultTags = []string{}
	}
	if item.IndicatorTypes == nil {
		item.IndicatorTypes = []string{}
	}
	return &item, nil
}

func scanThreatFeedSyncHistory(row scanner) (*model.ThreatFeedSyncHistory, error) {
	var item model.ThreatFeedSyncHistory
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.FeedID,
		&item.Status,
		&item.IndicatorsParsed,
		&item.IndicatorsImported,
		&item.IndicatorsSkipped,
		&item.IndicatorsFailed,
		&item.DurationMs,
		&item.ErrorMessage,
		&item.Metadata,
		&item.StartedAt,
		&item.CompletedAt,
	); err != nil {
		return nil, err
	}
	item.Metadata = ensureRawMessage(item.Metadata, "{}")
	return &item, nil
}
