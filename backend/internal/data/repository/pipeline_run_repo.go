package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/database"
)

type PipelineRunRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

type PipelineRunLogRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewPipelineRunRepository(db *pgxpool.Pool, logger zerolog.Logger) *PipelineRunRepository {
	return &PipelineRunRepository{db: db, logger: logger}
}

func NewPipelineRunLogRepository(db *pgxpool.Pool, logger zerolog.Logger) *PipelineRunLogRepository {
	return &PipelineRunLogRepository{db: db, logger: logger}
}

func (r *PipelineRunRepository) Create(ctx context.Context, item *model.PipelineRun) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO pipeline_runs (
			id, tenant_id, pipeline_id, status, current_phase, records_extracted, records_transformed,
			records_loaded, records_failed, records_filtered, records_deduplicated, bytes_read, bytes_written,
			quality_gate_results, quality_gates_passed, quality_gates_failed, quality_gates_warned,
			started_at, extract_started_at, extract_completed_at, transform_started_at, transform_completed_at,
			load_started_at, load_completed_at, completed_at, duration_ms, error_phase, error_message,
			error_details, triggered_by, triggered_by_user, retry_count, incremental_from, incremental_to, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13,
			$14, $15, $16, $17,
			$18, $19, $20, $21, $22,
			$23, $24, $25, $26, $27, $28,
			$29, $30, $31, $32, $33, $34, $35
		)`,
		item.ID, item.TenantID, item.PipelineID, item.Status, item.CurrentPhase, item.RecordsExtracted, item.RecordsTransformed,
		item.RecordsLoaded, item.RecordsFailed, item.RecordsFiltered, item.RecordsDeduplicated, item.BytesRead, item.BytesWritten,
		marshalJSONValue(item.QualityGateResults), item.QualityGatesPassed, item.QualityGatesFailed, item.QualityGatesWarned,
		item.StartedAt, item.ExtractStartedAt, item.ExtractCompletedAt, item.TransformStartedAt, item.TransformCompletedAt,
		item.LoadStartedAt, item.LoadCompletedAt, item.CompletedAt, item.DurationMs, item.ErrorPhase, item.ErrorMessage,
		item.ErrorDetails, item.TriggeredBy, item.TriggeredByUser, item.RetryCount, item.IncrementalFrom, item.IncrementalTo, item.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert pipeline run: %w", err)
	}
	return nil
}

func (r *PipelineRunRepository) Update(ctx context.Context, item *model.PipelineRun) error {
	result, err := r.db.Exec(ctx, `
		UPDATE pipeline_runs
		SET status = $4,
		    current_phase = $5,
		    records_extracted = $6,
		    records_transformed = $7,
		    records_loaded = $8,
		    records_failed = $9,
		    records_filtered = $10,
		    records_deduplicated = $11,
		    bytes_read = $12,
		    bytes_written = $13,
		    quality_gate_results = $14,
		    quality_gates_passed = $15,
		    quality_gates_failed = $16,
		    quality_gates_warned = $17,
		    started_at = $18,
		    extract_started_at = $19,
		    extract_completed_at = $20,
		    transform_started_at = $21,
		    transform_completed_at = $22,
		    load_started_at = $23,
		    load_completed_at = $24,
		    completed_at = $25,
		    duration_ms = $26,
		    error_phase = $27,
		    error_message = $28,
		    error_details = $29,
		    triggered_by = $30,
		    triggered_by_user = $31,
		    retry_count = $32,
		    incremental_from = $33,
		    incremental_to = $34
		WHERE tenant_id = $1 AND pipeline_id = $2 AND id = $3`,
		item.TenantID, item.PipelineID, item.ID, item.Status, item.CurrentPhase, item.RecordsExtracted, item.RecordsTransformed,
		item.RecordsLoaded, item.RecordsFailed, item.RecordsFiltered, item.RecordsDeduplicated, item.BytesRead, item.BytesWritten,
		marshalJSONValue(item.QualityGateResults), item.QualityGatesPassed, item.QualityGatesFailed, item.QualityGatesWarned,
		item.StartedAt, item.ExtractStartedAt, item.ExtractCompletedAt, item.TransformStartedAt, item.TransformCompletedAt,
		item.LoadStartedAt, item.LoadCompletedAt, item.CompletedAt, item.DurationMs, item.ErrorPhase, item.ErrorMessage,
		item.ErrorDetails, item.TriggeredBy, item.TriggeredByUser, item.RetryCount, item.IncrementalFrom, item.IncrementalTo,
	)
	if err != nil {
		return fmt.Errorf("update pipeline run: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *PipelineRunRepository) Get(ctx context.Context, tenantID, pipelineID, runID uuid.UUID) (*model.PipelineRun, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, pipeline_id, status, current_phase, records_extracted, records_transformed,
		       records_loaded, records_failed, records_filtered, records_deduplicated, bytes_read, bytes_written,
		       quality_gate_results, quality_gates_passed, quality_gates_failed, quality_gates_warned,
		       started_at, extract_started_at, extract_completed_at, transform_started_at, transform_completed_at,
		       load_started_at, load_completed_at, completed_at, duration_ms, error_phase, error_message,
		       error_details, triggered_by, triggered_by_user, retry_count, incremental_from, incremental_to, created_at
		FROM pipeline_runs
		WHERE tenant_id = $1 AND pipeline_id = $2 AND id = $3`,
		tenantID, pipelineID, runID,
	)
	return scanPipelineRun(row)
}

func (r *PipelineRunRepository) ListByPipeline(ctx context.Context, tenantID, pipelineID uuid.UUID, params dto.ListPipelineRunsParams) ([]*model.PipelineRun, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.tenant_id, a.pipeline_id, a.status, a.current_phase, a.records_extracted, a.records_transformed,
		       a.records_loaded, a.records_failed, a.records_filtered, a.records_deduplicated, a.bytes_read, a.bytes_written,
		       a.quality_gate_results, a.quality_gates_passed, a.quality_gates_failed, a.quality_gates_warned,
		       a.started_at, a.extract_started_at, a.extract_completed_at, a.transform_started_at, a.transform_completed_at,
		       a.load_started_at, a.load_completed_at, a.completed_at, a.duration_ms, a.error_phase, a.error_message,
		       a.error_details, a.triggered_by, a.triggered_by_user, a.retry_count, a.incremental_from, a.incremental_to, a.created_at
		FROM pipeline_runs a`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.Where("a.pipeline_id = ?", pipelineID)
	qb.WhereIf(params.Status != "", "a.status = ?", params.Status)
	qb.OrderBy("started_at", "desc", []string{"started_at", "completed_at", "status"})
	qb.Paginate(params.Page, params.PerPage)

	query, args := qb.Build()
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list pipeline runs: %w", err)
	}
	defer rows.Close()

	items := make([]*model.PipelineRun, 0)
	for rows.Next() {
		item, err := scanPipelineRun(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate pipeline runs: %w", err)
	}

	countQuery, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count pipeline runs: %w", err)
	}
	return items, total, nil
}

func (r *PipelineRunRepository) HasRunningRun(ctx context.Context, tenantID, pipelineID uuid.UUID) (bool, error) {
	var exists bool
	if err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM pipeline_runs WHERE tenant_id = $1 AND pipeline_id = $2 AND status = 'running'
		)`,
		tenantID, pipelineID,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check running pipeline run: %w", err)
	}
	return exists, nil
}

func (r *PipelineRunRepository) ListActive(ctx context.Context, tenantID uuid.UUID) ([]*model.PipelineRun, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, pipeline_id, status, current_phase, records_extracted, records_transformed,
		       records_loaded, records_failed, records_filtered, records_deduplicated, bytes_read, bytes_written,
		       quality_gate_results, quality_gates_passed, quality_gates_failed, quality_gates_warned,
		       started_at, extract_started_at, extract_completed_at, transform_started_at, transform_completed_at,
		       load_started_at, load_completed_at, completed_at, duration_ms, error_phase, error_message,
		       error_details, triggered_by, triggered_by_user, retry_count, incremental_from, incremental_to, created_at
		FROM pipeline_runs
		WHERE tenant_id = $1 AND status = 'running'
		ORDER BY started_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list active pipeline runs: %w", err)
	}
	defer rows.Close()

	items := make([]*model.PipelineRun, 0)
	for rows.Next() {
		item, err := scanPipelineRun(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PipelineRunRepository) LatestCompleted(ctx context.Context, tenantID, pipelineID uuid.UUID) (*model.PipelineRun, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, pipeline_id, status, current_phase, records_extracted, records_transformed,
		       records_loaded, records_failed, records_filtered, records_deduplicated, bytes_read, bytes_written,
		       quality_gate_results, quality_gates_passed, quality_gates_failed, quality_gates_warned,
		       started_at, extract_started_at, extract_completed_at, transform_started_at, transform_completed_at,
		       load_started_at, load_completed_at, completed_at, duration_ms, error_phase, error_message,
		       error_details, triggered_by, triggered_by_user, retry_count, incremental_from, incremental_to, created_at
		FROM pipeline_runs
		WHERE tenant_id = $1 AND pipeline_id = $2 AND status = 'completed'
		ORDER BY started_at DESC
		LIMIT 1`,
		tenantID, pipelineID,
	)
	return scanPipelineRun(row)
}

func (r *PipelineRunRepository) ConsecutiveFailures(ctx context.Context, tenantID, pipelineID uuid.UUID, limit int) (int, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.Query(ctx, `
		SELECT status
		FROM pipeline_runs
		WHERE tenant_id = $1 AND pipeline_id = $2
		ORDER BY started_at DESC
		LIMIT $3`,
		tenantID, pipelineID, limit,
	)
	if err != nil {
		return 0, fmt.Errorf("list recent pipeline statuses: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var status string
		if err := rows.Scan(&status); err != nil {
			return 0, fmt.Errorf("scan recent pipeline status: %w", err)
		}
		if status != string(model.PipelineRunStatusFailed) {
			break
		}
		count++
	}
	return count, rows.Err()
}

func (r *PipelineRunLogRepository) Create(ctx context.Context, item *model.PipelineRunLog) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO pipeline_run_logs (id, tenant_id, run_id, level, phase, message, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		item.ID, item.TenantID, item.RunID, item.Level, item.Phase, item.Message, item.Details, item.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert pipeline run log: %w", err)
	}
	return nil
}

func (r *PipelineRunLogRepository) ListByRun(ctx context.Context, tenantID, runID uuid.UUID, limit int) ([]*model.PipelineRunLog, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := r.db.Query(ctx, `
		SELECT l.id, l.tenant_id, l.run_id, l.level, l.phase, l.message, l.details, l.created_at
		FROM pipeline_run_logs l
		JOIN pipeline_runs pr ON pr.id = l.run_id
		WHERE l.tenant_id = $1 AND l.run_id = $2 AND pr.tenant_id = $1
		ORDER BY l.created_at ASC
		LIMIT $3`,
		tenantID, runID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list pipeline run logs: %w", err)
	}
	defer rows.Close()

	items := make([]*model.PipelineRunLog, 0)
	for rows.Next() {
		item, err := scanPipelineRunLog(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

type pipelineRunScanner interface {
	Scan(dest ...any) error
}

func scanPipelineRun(scanner pipelineRunScanner) (*model.PipelineRun, error) {
	item := &model.PipelineRun{}
	var qualityJSON []byte
	if err := scanner.Scan(
		&item.ID, &item.TenantID, &item.PipelineID, &item.Status, &item.CurrentPhase, &item.RecordsExtracted, &item.RecordsTransformed,
		&item.RecordsLoaded, &item.RecordsFailed, &item.RecordsFiltered, &item.RecordsDeduplicated, &item.BytesRead, &item.BytesWritten,
		&qualityJSON, &item.QualityGatesPassed, &item.QualityGatesFailed, &item.QualityGatesWarned,
		&item.StartedAt, &item.ExtractStartedAt, &item.ExtractCompletedAt, &item.TransformStartedAt, &item.TransformCompletedAt,
		&item.LoadStartedAt, &item.LoadCompletedAt, &item.CompletedAt, &item.DurationMs, &item.ErrorPhase, &item.ErrorMessage,
		&item.ErrorDetails, &item.TriggeredBy, &item.TriggeredByUser, &item.RetryCount, &item.IncrementalFrom, &item.IncrementalTo, &item.CreatedAt,
	); err != nil {
		return nil, err
	}
	if len(qualityJSON) > 0 && string(qualityJSON) != "null" {
		if err := json.Unmarshal(qualityJSON, &item.QualityGateResults); err != nil {
			return nil, fmt.Errorf("decode pipeline quality gate results: %w", err)
		}
	}
	return item, nil
}

func scanPipelineRunLog(scanner interface{ Scan(dest ...any) error }) (*model.PipelineRunLog, error) {
	item := &model.PipelineRunLog{}
	if err := scanner.Scan(&item.ID, &item.TenantID, &item.RunID, &item.Level, &item.Phase, &item.Message, &item.Details, &item.CreatedAt); err != nil {
		return nil, err
	}
	return item, nil
}
