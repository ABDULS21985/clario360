package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/database"
)

type QualityResultRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewQualityResultRepository(db *pgxpool.Pool, logger zerolog.Logger) *QualityResultRepository {
	return &QualityResultRepository{db: db, logger: logger}
}

func (r *QualityResultRepository) Create(ctx context.Context, item *model.QualityResult) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO quality_results (
			id, tenant_id, rule_id, model_id, pipeline_run_id, status, records_checked, records_passed,
			records_failed, pass_rate, failure_samples, failure_summary, checked_at, duration_ms, error_message, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15, $16
		)`,
		item.ID, item.TenantID, item.RuleID, item.ModelID, item.PipelineRunID, item.Status, item.RecordsChecked, item.RecordsPassed,
		item.RecordsFailed, item.PassRate, item.FailureSamples, item.FailureSummary, item.CheckedAt, item.DurationMs, item.ErrorMessage, item.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert quality result: %w", err)
	}
	return nil
}

func (r *QualityResultRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.QualityResult, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, rule_id, model_id, pipeline_run_id, status, records_checked, records_passed,
		       records_failed, pass_rate, failure_samples, failure_summary, checked_at, duration_ms, error_message, created_at
		FROM quality_results
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	return scanQualityResult(row)
}

func (r *QualityResultRepository) List(ctx context.Context, tenantID uuid.UUID, params dto.ListQualityResultsParams) ([]*model.QualityResult, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.tenant_id, a.rule_id, a.model_id, a.pipeline_run_id, a.status, a.records_checked, a.records_passed,
		       a.records_failed, a.pass_rate, a.failure_samples, a.failure_summary, a.checked_at, a.duration_ms, a.error_message, a.created_at
		FROM quality_results a`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.WhereIf(params.RuleID != "", "a.rule_id = ?", params.RuleID)
	qb.WhereIf(params.ModelID != "", "a.model_id = ?", params.ModelID)
	qb.WhereIf(params.Status != "", "a.status = ?", params.Status)
	qb.OrderBy(coalesce(params.Sort, "checked_at"), coalesce(params.Order, "desc"), []string{"checked_at", "created_at", "status"})
	qb.Paginate(params.Page, params.PerPage)

	query, args := qb.Build()
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list quality results: %w", err)
	}
	defer rows.Close()

	items := make([]*model.QualityResult, 0)
	for rows.Next() {
		item, err := scanQualityResult(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate quality results: %w", err)
	}

	countQuery, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count quality results: %w", err)
	}
	return items, total, nil
}

func (r *QualityResultRepository) LatestByRule(ctx context.Context, tenantID, ruleID uuid.UUID) (*model.QualityResult, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, rule_id, model_id, pipeline_run_id, status, records_checked, records_passed,
		       records_failed, pass_rate, failure_samples, failure_summary, checked_at, duration_ms, error_message, created_at
		FROM quality_results
		WHERE tenant_id = $1 AND rule_id = $2
		ORDER BY checked_at DESC
		LIMIT 1`,
		tenantID, ruleID,
	)
	return scanQualityResult(row)
}

func (r *QualityResultRepository) Trend(ctx context.Context, tenantID uuid.UUID, days int) ([]model.QualityTrendPoint, error) {
	if days <= 0 {
		days = 30
	}
	rows, err := r.db.Query(ctx, `
		SELECT DATE_TRUNC('day', checked_at) AS day,
		       ROUND(AVG(COALESCE(pass_rate, 0)), 2) AS score
		FROM quality_results
		WHERE tenant_id = $1
		  AND checked_at >= NOW() - ($2::int * INTERVAL '1 day')
		GROUP BY DATE_TRUNC('day', checked_at)
		ORDER BY day ASC`,
		tenantID, days,
	)
	if err != nil {
		return nil, fmt.Errorf("query quality trend: %w", err)
	}
	defer rows.Close()

	points := make([]model.QualityTrendPoint, 0)
	for rows.Next() {
		var point model.QualityTrendPoint
		if err := rows.Scan(&point.Day, &point.Score); err != nil {
			return nil, fmt.Errorf("scan quality trend: %w", err)
		}
		points = append(points, point)
	}
	return points, rows.Err()
}

func scanQualityResult(scanner interface{ Scan(dest ...any) error }) (*model.QualityResult, error) {
	item := &model.QualityResult{}
	if err := scanner.Scan(
		&item.ID, &item.TenantID, &item.RuleID, &item.ModelID, &item.PipelineRunID, &item.Status, &item.RecordsChecked, &item.RecordsPassed,
		&item.RecordsFailed, &item.PassRate, &item.FailureSamples, &item.FailureSummary, &item.CheckedAt, &item.DurationMs, &item.ErrorMessage, &item.CreatedAt,
	); err != nil {
		return nil, err
	}
	return item, nil
}
