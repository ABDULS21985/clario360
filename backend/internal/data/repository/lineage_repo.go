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

type LineageRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewLineageRepository(db *pgxpool.Pool, logger zerolog.Logger) *LineageRepository {
	return &LineageRepository{db: db, logger: logger}
}

func (r *LineageRepository) Upsert(ctx context.Context, edge *model.LineageEdgeRecord) error {
	if edge.ID == uuid.Nil {
		edge.ID = uuid.New()
	}
	if edge.RecordedBy == "" {
		edge.RecordedBy = model.LineageRecordedBySystem
	}
	now := time.Now().UTC()
	if edge.FirstSeenAt.IsZero() {
		edge.FirstSeenAt = now
	}
	edge.LastSeenAt = now
	if edge.CreatedAt.IsZero() {
		edge.CreatedAt = now
	}
	edge.UpdatedAt = now
	if edge.Metadata == nil {
		edge.Metadata = json.RawMessage(`{}`)
	}

	row := r.db.QueryRow(ctx, `
		INSERT INTO data_lineage_edges (
			id, tenant_id, source_type, source_id, source_name, target_type, target_id, target_name,
			relationship, transformation_desc, transformation_type, columns_affected, pipeline_id, pipeline_run_id,
			recorded_by, active, first_seen_at, last_seen_at, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21
		)
		ON CONFLICT (tenant_id, source_type, source_id, target_type, target_id, relationship)
		DO UPDATE SET
			source_name = EXCLUDED.source_name,
			target_name = EXCLUDED.target_name,
			transformation_desc = EXCLUDED.transformation_desc,
			transformation_type = EXCLUDED.transformation_type,
			columns_affected = EXCLUDED.columns_affected,
			pipeline_id = COALESCE(EXCLUDED.pipeline_id, data_lineage_edges.pipeline_id),
			pipeline_run_id = COALESCE(EXCLUDED.pipeline_run_id, data_lineage_edges.pipeline_run_id),
			recorded_by = EXCLUDED.recorded_by,
			active = true,
			last_seen_at = EXCLUDED.last_seen_at,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
		RETURNING id, first_seen_at, created_at, updated_at`,
		edge.ID, edge.TenantID, edge.SourceType, edge.SourceID, edge.SourceName, edge.TargetType, edge.TargetID, edge.TargetName,
		edge.Relationship, edge.TransformationDesc, edge.TransformationType, ensureStringSlice(edge.ColumnsAffected), edge.PipelineID, edge.PipelineRunID,
		edge.RecordedBy, true, edge.FirstSeenAt, edge.LastSeenAt, edge.Metadata, edge.CreatedAt, edge.UpdatedAt,
	)
	if err := row.Scan(&edge.ID, &edge.FirstSeenAt, &edge.CreatedAt, &edge.UpdatedAt); err != nil {
		return fmt.Errorf("upsert data lineage edge: %w", err)
	}
	edge.Active = true
	return nil
}

func (r *LineageRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.LineageEdgeRecord, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, source_type, source_id, source_name, target_type, target_id, target_name,
		       relationship, transformation_desc, transformation_type, columns_affected, pipeline_id, pipeline_run_id,
		       recorded_by, active, first_seen_at, last_seen_at, metadata, created_at, updated_at
		FROM data_lineage_edges
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	return scanLineageEdge(row)
}

func (r *LineageRepository) ListActive(ctx context.Context, tenantID uuid.UUID) ([]*model.LineageEdgeRecord, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, source_type, source_id, source_name, target_type, target_id, target_name,
		       relationship, transformation_desc, transformation_type, columns_affected, pipeline_id, pipeline_run_id,
		       recorded_by, active, first_seen_at, last_seen_at, metadata, created_at, updated_at
		FROM data_lineage_edges
		WHERE tenant_id = $1 AND active = true
		ORDER BY created_at ASC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list active data lineage edges: %w", err)
	}
	defer rows.Close()

	edges := make([]*model.LineageEdgeRecord, 0)
	for rows.Next() {
		edge, err := scanLineageEdge(rows)
		if err != nil {
			return nil, err
		}
		edges = append(edges, edge)
	}
	return edges, rows.Err()
}

func (r *LineageRepository) ListEntityEdges(ctx context.Context, tenantID uuid.UUID, entityType model.LineageEntityType, entityID uuid.UUID) ([]*model.LineageEdgeRecord, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, source_type, source_id, source_name, target_type, target_id, target_name,
		       relationship, transformation_desc, transformation_type, columns_affected, pipeline_id, pipeline_run_id,
		       recorded_by, active, first_seen_at, last_seen_at, metadata, created_at, updated_at
		FROM data_lineage_edges
		WHERE tenant_id = $1
		  AND active = true
		  AND ((source_type = $2 AND source_id = $3) OR (target_type = $2 AND target_id = $3))
		ORDER BY created_at ASC`,
		tenantID, entityType, entityID,
	)
	if err != nil {
		return nil, fmt.Errorf("list data lineage entity edges: %w", err)
	}
	defer rows.Close()

	items := make([]*model.LineageEdgeRecord, 0)
	for rows.Next() {
		item, err := scanLineageEdge(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *LineageRepository) Deactivate(ctx context.Context, tenantID, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `
		UPDATE data_lineage_edges
		SET active = false, updated_at = now()
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	if err != nil {
		return fmt.Errorf("deactivate data lineage edge: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

type lineageScanner interface {
	Scan(dest ...any) error
}

func scanLineageEdge(scanner lineageScanner) (*model.LineageEdgeRecord, error) {
	item := &model.LineageEdgeRecord{}
	var columns []string
	var metadata []byte
	if err := scanner.Scan(
		&item.ID, &item.TenantID, &item.SourceType, &item.SourceID, &item.SourceName, &item.TargetType, &item.TargetID, &item.TargetName,
		&item.Relationship, &item.TransformationDesc, &item.TransformationType, &columns, &item.PipelineID, &item.PipelineRunID,
		&item.RecordedBy, &item.Active, &item.FirstSeenAt, &item.LastSeenAt, &metadata, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.ColumnsAffected = columns
	item.Metadata = metadata
	return item, nil
}
