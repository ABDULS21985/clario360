package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/database"
)

type PipelineRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

type PipelineRunStatePatch struct {
	LastRunID             *uuid.UUID
	LastRunAt             *time.Time
	LastRunStatus         *string
	LastRunError          *string
	NextRunAt             *time.Time
	TotalRuns             *int
	SuccessfulRuns        *int
	FailedRuns            *int
	TotalRecordsProcessed *int64
	AvgDurationMs         *int64
	Status                *model.PipelineStatus
	Config                *model.PipelineConfig
}

func NewPipelineRepository(db *pgxpool.Pool, logger zerolog.Logger) *PipelineRepository {
	return &PipelineRepository{db: db, logger: logger}
}

func (r *PipelineRepository) ExistsByName(ctx context.Context, tenantID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
	query := `SELECT EXISTS (
		SELECT 1 FROM pipelines WHERE tenant_id = $1 AND lower(name) = lower($2) AND deleted_at IS NULL`
	args := []any{tenantID, name}
	if excludeID != nil {
		query += ` AND id <> $3`
		args = append(args, *excludeID)
	}
	query += `)`

	var exists bool
	if err := r.db.QueryRow(ctx, query, args...).Scan(&exists); err != nil {
		return false, fmt.Errorf("check pipeline duplicate name: %w", err)
	}
	return exists, nil
}

func (r *PipelineRepository) Create(ctx context.Context, item *model.Pipeline) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO pipelines (
			id, tenant_id, name, description, type, source_id, target_id, config, schedule, status,
			last_run_id, last_run_at, last_run_status, last_run_error, next_run_at, total_runs,
			successful_runs, failed_runs, total_records_processed, avg_duration_ms, tags,
			created_by, created_at, updated_at, deleted_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21,
			$22, $23, $24, $25
		)`,
		item.ID, item.TenantID, item.Name, item.Description, item.Type, item.SourceID, item.TargetID, marshalJSONValue(item.Config), item.Schedule, item.Status,
		item.LastRunID, item.LastRunAt, item.LastRunStatus, item.LastRunError, item.NextRunAt, item.TotalRuns,
		item.SuccessfulRuns, item.FailedRuns, item.TotalRecordsProcessed, item.AvgDurationMs, ensureStringSlice(item.Tags),
		item.CreatedBy, item.CreatedAt, item.UpdatedAt, item.DeletedAt,
	)
	if err != nil {
		return fmt.Errorf("insert pipeline: %w", err)
	}
	return nil
}

func (r *PipelineRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.Pipeline, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, description, type, source_id, target_id, config, schedule, status,
		       last_run_id, last_run_at, last_run_status, last_run_error, next_run_at, total_runs,
		       successful_runs, failed_runs, total_records_processed, avg_duration_ms, tags,
		       created_by, created_at, updated_at, deleted_at
		FROM pipelines
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id,
	)
	return scanPipeline(row)
}

func (r *PipelineRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Pipeline, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, description, type, source_id, target_id, config, schedule, status,
		       last_run_id, last_run_at, last_run_status, last_run_error, next_run_at, total_runs,
		       successful_runs, failed_runs, total_records_processed, avg_duration_ms, tags,
		       created_by, created_at, updated_at, deleted_at
		FROM pipelines
		WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	return scanPipeline(row)
}

func (r *PipelineRepository) List(ctx context.Context, tenantID uuid.UUID, params dto.ListPipelinesParams) ([]*model.Pipeline, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.tenant_id, a.name, a.description, a.type, a.source_id, a.target_id, a.config, a.schedule, a.status,
		       a.last_run_id, a.last_run_at, a.last_run_status, a.last_run_error, a.next_run_at, a.total_runs,
		       a.successful_runs, a.failed_runs, a.total_records_processed, a.avg_duration_ms, a.tags,
		       a.created_by, a.created_at, a.updated_at, a.deleted_at
		FROM pipelines a`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.Where("a.deleted_at IS NULL")
	qb.WhereIf(strings.TrimSpace(params.Search) != "", "a.name ILIKE ?", "%"+strings.TrimSpace(params.Search)+"%")
	qb.WhereIn("a.type", params.Types)
	qb.WhereIn("a.status", params.Statuses)
	qb.WhereIf(params.SourceID != "", "a.source_id = ?", params.SourceID)
	qb.OrderBy(coalesce(params.Sort, "updated_at"), coalesce(params.Order, "desc"), []string{"name", "type", "status", "last_run_at", "next_run_at", "created_at", "updated_at"})
	qb.Paginate(params.Page, params.PerPage)

	query, args := qb.Build()
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list pipelines: %w", err)
	}
	defer rows.Close()

	items := make([]*model.Pipeline, 0)
	for rows.Next() {
		item, err := scanPipeline(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate pipelines: %w", err)
	}

	countQuery, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count pipelines: %w", err)
	}
	return items, total, nil
}

func (r *PipelineRepository) Update(ctx context.Context, item *model.Pipeline) error {
	result, err := r.db.Exec(ctx, `
		UPDATE pipelines
		SET name = $3,
		    description = $4,
		    type = $5,
		    target_id = $6,
		    config = $7,
		    schedule = $8,
		    status = $9,
		    next_run_at = $10,
		    tags = $11,
		    updated_at = $12
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		item.TenantID, item.ID, item.Name, item.Description, item.Type, item.TargetID, marshalJSONValue(item.Config), item.Schedule,
		item.Status, item.NextRunAt, ensureStringSlice(item.Tags), item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update pipeline: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PipelineRepository) UpdateRunState(ctx context.Context, tenantID, pipelineID uuid.UUID, patch PipelineRunStatePatch) error {
	result, err := r.db.Exec(ctx, `
		UPDATE pipelines
		SET last_run_id = COALESCE($3, last_run_id),
		    last_run_at = COALESCE($4, last_run_at),
		    last_run_status = COALESCE($5, last_run_status),
		    last_run_error = $6,
		    next_run_at = $7,
		    total_runs = COALESCE($8, total_runs),
		    successful_runs = COALESCE($9, successful_runs),
		    failed_runs = COALESCE($10, failed_runs),
		    total_records_processed = COALESCE($11, total_records_processed),
		    avg_duration_ms = COALESCE($12, avg_duration_ms),
		    status = COALESCE($13, status),
		    config = COALESCE($14, config),
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, pipelineID, patch.LastRunID, patch.LastRunAt, patch.LastRunStatus, patch.LastRunError, patch.NextRunAt,
		patch.TotalRuns, patch.SuccessfulRuns, patch.FailedRuns, patch.TotalRecordsProcessed, patch.AvgDurationMs,
		patch.Status, marshalPipelineConfigPtr(patch.Config),
	)
	if err != nil {
		return fmt.Errorf("update pipeline run state: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PipelineRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status model.PipelineStatus) error {
	result, err := r.db.Exec(ctx, `
		UPDATE pipelines SET status = $3, updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id, status,
	)
	if err != nil {
		return fmt.Errorf("update pipeline status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PipelineRepository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID, deletedAt time.Time) error {
	result, err := r.db.Exec(ctx, `
		UPDATE pipelines SET deleted_at = $3, updated_at = $3
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id, deletedAt,
	)
	if err != nil {
		return fmt.Errorf("soft delete pipeline: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PipelineRepository) ListDue(ctx context.Context, before time.Time, limit int) ([]*model.Pipeline, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, name, description, type, source_id, target_id, config, schedule, status,
		       last_run_id, last_run_at, last_run_status, last_run_error, next_run_at, total_runs,
		       successful_runs, failed_runs, total_records_processed, avg_duration_ms, tags,
		       created_by, created_at, updated_at, deleted_at
		FROM pipelines
		WHERE deleted_at IS NULL
		  AND status = 'active'
		  AND schedule IS NOT NULL
		  AND next_run_at IS NOT NULL
		  AND next_run_at <= $1
		ORDER BY next_run_at ASC
		LIMIT $2`,
		before, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list due pipelines: %w", err)
	}
	defer rows.Close()

	items := make([]*model.Pipeline, 0)
	for rows.Next() {
		item, err := scanPipeline(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PipelineRepository) Stats(ctx context.Context, tenantID uuid.UUID) (*model.PipelineStats, error) {
	stats := &model.PipelineStats{
		ByType:   map[string]int{},
		ByStatus: map[string]int{},
		UpdatedAt: time.Now().UTC(),
	}

	rows, err := r.db.Query(ctx, `
		SELECT type, status, COUNT(*)
		FROM pipelines
		WHERE tenant_id = $1 AND deleted_at IS NULL
		GROUP BY type, status`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("query pipeline stats breakdown: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var typeName string
		var status string
		var count int
		if err := rows.Scan(&typeName, &status, &count); err != nil {
			return nil, fmt.Errorf("scan pipeline stats breakdown: %w", err)
		}
		stats.TotalPipelines += count
		stats.ByType[typeName] += count
		stats.ByStatus[status] += count
		switch status {
		case string(model.PipelineStatusActive):
			stats.ActivePipelines += count
		case string(model.PipelineStatusPaused):
			stats.PausedPipelines += count
		case string(model.PipelineStatusError):
			stats.ErrorPipelines += count
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pipeline stats breakdown: %w", err)
	}

	if err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(COUNT(*) FILTER (WHERE status = 'running'), 0),
			COALESCE(COUNT(*) FILTER (WHERE status = 'completed'), 0),
			COALESCE(COUNT(*) FILTER (WHERE status = 'failed'), 0)
		FROM pipeline_runs
		WHERE tenant_id = $1`,
		tenantID,
	).Scan(&stats.RunningPipelines, &stats.CompletedRuns, &stats.FailedRuns); err != nil {
		return nil, fmt.Errorf("query pipeline run stats: %w", err)
	}
	if total := stats.CompletedRuns + stats.FailedRuns; total > 0 {
		stats.SuccessRate = float64(stats.CompletedRuns) / float64(total) * 100
	}
	return stats, nil
}

type pipelineScanner interface {
	Scan(dest ...any) error
}

func scanPipeline(scanner pipelineScanner) (*model.Pipeline, error) {
	item := &model.Pipeline{}
	var configJSON []byte
	var tags []string
	if err := scanner.Scan(
		&item.ID, &item.TenantID, &item.Name, &item.Description, &item.Type, &item.SourceID, &item.TargetID, &configJSON, &item.Schedule, &item.Status,
		&item.LastRunID, &item.LastRunAt, &item.LastRunStatus, &item.LastRunError, &item.NextRunAt, &item.TotalRuns,
		&item.SuccessfulRuns, &item.FailedRuns, &item.TotalRecordsProcessed, &item.AvgDurationMs, &tags,
		&item.CreatedBy, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt,
	); err != nil {
		return nil, err
	}
	item.Tags = tags
	if len(configJSON) > 0 && string(configJSON) != "null" {
		if err := json.Unmarshal(configJSON, &item.Config); err != nil {
			return nil, fmt.Errorf("decode pipeline config: %w", err)
		}
	}
	return item, nil
}

func marshalPipelineConfigPtr(value *model.PipelineConfig) []byte {
	if value == nil {
		return nil
	}
	return marshalJSONValue(*value)
}
