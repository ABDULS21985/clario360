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

	"github.com/clario360/platform/internal/cyber/ueba/model"
	"github.com/clario360/platform/internal/database"
)

type EventRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewEventRepository(db *pgxpool.Pool, logger zerolog.Logger) *EventRepository {
	return &EventRepository{
		db:     db,
		logger: logger.With().Str("component", "ueba-event-repo").Logger(),
	}
}

func (r *EventRepository) EnsurePartitions(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `SELECT manage_ueba_event_partitions()`)
	return err
}

func (r *EventRepository) InsertBatch(ctx context.Context, tenantID uuid.UUID, events []*model.DataAccessEvent) error {
	if len(events) == 0 {
		return nil
	}
	return database.RunWithTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		batch := &pgx.Batch{}
		for _, event := range events {
			if event == nil {
				continue
			}
			if event.ID == uuid.Nil {
				event.ID = uuid.New()
			}
			if event.CreatedAt.IsZero() {
				event.CreatedAt = time.Now().UTC()
			}
			signalsJSON, err := json.Marshal(event.AnomalySignals)
			if err != nil {
				return err
			}
			batch.Queue(`
				INSERT INTO ueba_access_events (
					id, tenant_id, entity_type, entity_id, source_type, source_id, action,
					database_name, schema_name, table_name, query_hash, rows_accessed, bytes_accessed,
					duration_ms, source_ip, user_agent, success, error_message, table_sensitivity,
					contains_pii, anomaly_signals, anomaly_count, event_timestamp, created_at
				) VALUES (
					$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,COALESCE($24, now())
				)`,
				event.ID, event.TenantID, event.EntityType, event.EntityID, event.SourceType, event.SourceID, event.Action,
				nullString(event.DatabaseName), nullString(event.SchemaName), nullString(event.TableName), nullString(event.QueryHash),
				nullInt64(event.RowsAccessed), nullInt64(event.BytesAccessed), nullInt(event.DurationMS),
				nullString(event.SourceIP), nullString(event.UserAgent), event.Success, nullString(event.ErrorMessage),
				nullString(event.TableSensitivity), event.ContainsPII, signalsJSON, event.AnomalyCount, event.EventTimestamp, event.CreatedAt,
			)
		}
		results := tx.SendBatch(ctx, batch)
		defer results.Close()
		for range events {
			if _, err := results.Exec(); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *EventRepository) ExistsDedupKey(ctx context.Context, tenantID uuid.UUID, event *model.DataAccessEvent) (bool, error) {
	var exists bool
	err := database.RunReadWithTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1
				FROM ueba_access_events
				WHERE tenant_id = $1
				  AND entity_id = $2
				  AND source_type = $3
				  AND action = $4
				  AND COALESCE(query_hash, '') = COALESCE($5, '')
				  AND event_timestamp = $6
			)`,
			tenantID, event.EntityID, event.SourceType, event.Action, nullString(event.QueryHash), event.EventTimestamp,
		).Scan(&exists)
	})
	return exists, err
}

func (r *EventRepository) UpdateAnomalyFlags(ctx context.Context, tenantID, eventID uuid.UUID, signals []model.AnomalySignal) error {
	return database.RunWithTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		payload, err := json.Marshal(signals)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE ueba_access_events
			SET anomaly_signals = $3, anomaly_count = $4
			WHERE tenant_id = $1 AND id = $2`,
			tenantID, eventID, payload, len(signals),
		)
		return err
	})
}

func (r *EventRepository) ListTimeline(ctx context.Context, tenantID uuid.UUID, entityID string, limit, offset int) ([]*model.DataAccessEvent, int, error) {
	if limit <= 0 {
		limit = 50
	}
	var (
		items []*model.DataAccessEvent
		total int
	)
	err := database.RunReadWithTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM ueba_access_events WHERE tenant_id = $1 AND entity_id = $2`, tenantID, entityID).Scan(&total); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
			SELECT
				id, tenant_id, entity_type, entity_id, source_type, source_id, action,
				COALESCE(database_name, ''), COALESCE(schema_name, ''), COALESCE(table_name, ''), COALESCE(query_hash, ''),
				COALESCE(rows_accessed, 0), COALESCE(bytes_accessed, 0), COALESCE(duration_ms, 0),
				COALESCE(source_ip, ''), COALESCE(user_agent, ''), success, COALESCE(error_message, ''),
				COALESCE(table_sensitivity, ''), COALESCE(contains_pii, false),
				anomaly_signals, anomaly_count, event_timestamp, created_at
			FROM ueba_access_events
			WHERE tenant_id = $1 AND entity_id = $2
			ORDER BY event_timestamp DESC
			LIMIT $3 OFFSET $4`,
			tenantID, entityID, limit, offset,
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			item, scanErr := scanEvent(rows)
			if scanErr != nil {
				return scanErr
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, total, err
}

func (r *EventRepository) GetByIDs(ctx context.Context, tenantID uuid.UUID, eventIDs []uuid.UUID) ([]*model.DataAccessEvent, error) {
	if len(eventIDs) == 0 {
		return []*model.DataAccessEvent{}, nil
	}
	items := make([]*model.DataAccessEvent, 0, len(eventIDs))
	err := database.RunReadWithTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT
				id, tenant_id, entity_type, entity_id, source_type, source_id, action,
				COALESCE(database_name, ''), COALESCE(schema_name, ''), COALESCE(table_name, ''), COALESCE(query_hash, ''),
				COALESCE(rows_accessed, 0), COALESCE(bytes_accessed, 0), COALESCE(duration_ms, 0),
				COALESCE(source_ip, ''), COALESCE(user_agent, ''), success, COALESCE(error_message, ''),
				COALESCE(table_sensitivity, ''), COALESCE(contains_pii, false),
				anomaly_signals, anomaly_count, event_timestamp, created_at
			FROM ueba_access_events
			WHERE tenant_id = $1 AND id = ANY($2)
			ORDER BY event_timestamp ASC`,
			tenantID, eventIDs,
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			item, scanErr := scanEvent(rows)
			if scanErr != nil {
				return scanErr
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func (r *EventRepository) ListSignalsWithinWindow(ctx context.Context, tenantID uuid.UUID, entityID string, since time.Time) ([]model.AnomalySignal, error) {
	signals := make([]model.AnomalySignal, 0)
	err := database.RunReadWithTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, anomaly_signals, event_timestamp
			FROM ueba_access_events
			WHERE tenant_id = $1
			  AND entity_id = $2
			  AND anomaly_count > 0
			  AND event_timestamp >= $3`,
			tenantID, entityID, since,
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var (
				eventID      uuid.UUID
				payload      []byte
				eventTime    time.Time
				eventSignals []model.AnomalySignal
			)
			if err := rows.Scan(&eventID, &payload, &eventTime); err != nil {
				return err
			}
			if err := json.Unmarshal(payload, &eventSignals); err != nil {
				return fmt.Errorf("decode stored ueba signals: %w", err)
			}
			for i := range eventSignals {
				eventSignals[i].EventID = eventID
				eventSignals[i].EventTimestamp = eventTime
				signals = append(signals, eventSignals[i])
			}
		}
		return rows.Err()
	})
	return signals, err
}

func (r *EventRepository) AggregateHeatmap(ctx context.Context, tenantID uuid.UUID, entityID string, days int) ([7][24]int, error) {
	if days <= 0 {
		days = 30
	}
	var matrix [7][24]int
	err := database.RunReadWithTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT EXTRACT(ISODOW FROM event_timestamp)::int AS dow, EXTRACT(HOUR FROM event_timestamp)::int AS hour, COUNT(*)::int
			FROM ueba_access_events
			WHERE tenant_id = $1 AND entity_id = $2 AND event_timestamp >= $3
			GROUP BY 1, 2`,
			tenantID, entityID, time.Now().UTC().AddDate(0, 0, -days),
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var dow, hour, count int
			if err := rows.Scan(&dow, &hour, &count); err != nil {
				return err
			}
			index := dow - 1
			if index >= 0 && index < 7 && hour >= 0 && hour < 24 {
				matrix[index][hour] = count
			}
		}
		return rows.Err()
	})
	return matrix, err
}

func (r *EventRepository) AggregateEntityVolume(ctx context.Context, tenantID uuid.UUID, entityID string, since time.Time) ([]map[string]any, error) {
	result := make([]map[string]any, 0)
	err := database.RunReadWithTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT date_trunc('day', event_timestamp) AS bucket,
			       COALESCE(SUM(bytes_accessed), 0)::bigint,
			       COALESCE(SUM(rows_accessed), 0)::bigint,
			       COUNT(*)::int
			FROM ueba_access_events
			WHERE tenant_id = $1 AND entity_id = $2 AND event_timestamp >= $3
			GROUP BY 1
			ORDER BY 1`,
			tenantID, entityID, since,
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var (
				bucket time.Time
				bytes  int64
				rowsN  int64
				count  int
			)
			if err := rows.Scan(&bucket, &bytes, &rowsN, &count); err != nil {
				return err
			}
			result = append(result, map[string]any{
				"bucket": bucket,
				"bytes":  bytes,
				"rows":   rowsN,
				"count":  count,
			})
		}
		return rows.Err()
	})
	return result, err
}

type eventScanner interface {
	Scan(dest ...any) error
}

func scanEvent(row eventScanner) (*model.DataAccessEvent, error) {
	var (
		item        model.DataAccessEvent
		signalsJSON []byte
		sourceID    *uuid.UUID
	)
	if err := row.Scan(
		&item.ID, &item.TenantID, &item.EntityType, &item.EntityID, &item.SourceType, &sourceID, &item.Action,
		&item.DatabaseName, &item.SchemaName, &item.TableName, &item.QueryHash,
		&item.RowsAccessed, &item.BytesAccessed, &item.DurationMS, &item.SourceIP, &item.UserAgent,
		&item.Success, &item.ErrorMessage, &item.TableSensitivity, &item.ContainsPII,
		&signalsJSON, &item.AnomalyCount, &item.EventTimestamp, &item.CreatedAt,
	); err != nil {
		return nil, err
	}
	item.SourceID = sourceID
	if len(signalsJSON) > 0 {
		if err := json.Unmarshal(signalsJSON, &item.AnomalySignals); err != nil {
			return nil, fmt.Errorf("decode ueba event anomaly signals: %w", err)
		}
	}
	return &item, nil
}

func nullInt(value int) any {
	if value == 0 {
		return nil
	}
	return value
}

func nullInt64(value int64) any {
	if value == 0 {
		return nil
	}
	return value
}
