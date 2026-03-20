package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/model"
)

type SyncRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewSyncRepository(db *pgxpool.Pool, logger zerolog.Logger) *SyncRepository {
	return &SyncRepository{db: db, logger: logger}
}

func (r *SyncRepository) Create(ctx context.Context, item *model.SyncHistory) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO sync_history (
			id, tenant_id, source_id, status, sync_type, tables_synced, rows_read,
			rows_written, bytes_transferred, errors, error_count, started_at,
			completed_at, duration_ms, triggered_by, triggered_by_user, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17
		)`,
		item.ID, item.TenantID, item.SourceID, item.Status, item.SyncType, item.TablesSynced, item.RowsRead,
		item.RowsWritten, item.BytesTransferred, item.Errors, item.ErrorCount, item.StartedAt,
		item.CompletedAt, item.DurationMs, item.TriggeredBy, item.TriggeredByUser, item.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert sync history: %w", err)
	}
	return nil
}

func (r *SyncRepository) Update(ctx context.Context, item *model.SyncHistory) error {
	result, err := r.db.Exec(ctx, `
		UPDATE sync_history
		SET status = $4,
		    sync_type = $5,
		    tables_synced = $6,
		    rows_read = $7,
		    rows_written = $8,
		    bytes_transferred = $9,
		    errors = $10,
		    error_count = $11,
		    started_at = $12,
		    completed_at = $13,
		    duration_ms = $14,
		    triggered_by = $15,
		    triggered_by_user = $16
		WHERE tenant_id = $1 AND source_id = $2 AND id = $3`,
		item.TenantID, item.SourceID, item.ID, item.Status, item.SyncType, item.TablesSynced, item.RowsRead,
		item.RowsWritten, item.BytesTransferred, item.Errors, item.ErrorCount, item.StartedAt,
		item.CompletedAt, item.DurationMs, item.TriggeredBy, item.TriggeredByUser,
	)
	if err != nil {
		return fmt.Errorf("update sync history: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *SyncRepository) ListBySource(ctx context.Context, tenantID, sourceID uuid.UUID, limit int) ([]*model.SyncHistory, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, source_id, status, sync_type, tables_synced, rows_read,
		       rows_written, bytes_transferred, errors, error_count, started_at,
		       completed_at, duration_ms, triggered_by, triggered_by_user, created_at
		FROM sync_history
		WHERE tenant_id = $1 AND source_id = $2
		ORDER BY started_at DESC
		LIMIT $3`,
		tenantID, sourceID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list sync history: %w", err)
	}
	defer rows.Close()

	values := make([]*model.SyncHistory, 0)
	for rows.Next() {
		item, err := scanSyncHistory(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, item)
	}
	return values, rows.Err()
}

func scanSyncHistory(scanner interface{ Scan(dest ...any) error }) (*model.SyncHistory, error) {
	item := &model.SyncHistory{}
	var errorsPayload []byte
	if err := scanner.Scan(
		&item.ID, &item.TenantID, &item.SourceID, &item.Status, &item.SyncType, &item.TablesSynced, &item.RowsRead,
		&item.RowsWritten, &item.BytesTransferred, &errorsPayload, &item.ErrorCount, &item.StartedAt,
		&item.CompletedAt, &item.DurationMs, &item.TriggeredBy, &item.TriggeredByUser, &item.CreatedAt,
	); err != nil {
		return nil, err
	}
	item.Errors = errorsPayload
	if len(item.Errors) == 0 {
		item.Errors = json.RawMessage(`[]`)
	}
	return item, nil
}

func NewRunningSync(tenantID, sourceID uuid.UUID, syncType model.SyncType, trigger model.SyncTrigger, userID *uuid.UUID, now time.Time) *model.SyncHistory {
	return &model.SyncHistory{
		ID:              uuid.New(),
		TenantID:        tenantID,
		SourceID:        sourceID,
		Status:          model.SyncStatusRunning,
		SyncType:        syncType,
		Errors:          json.RawMessage(`[]`),
		StartedAt:       now,
		TriggeredBy:     trigger,
		TriggeredByUser: userID,
		CreatedAt:       now,
	}
}
