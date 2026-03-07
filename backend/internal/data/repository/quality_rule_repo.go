package repository

import (
	"context"
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

type QualityRuleRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewQualityRuleRepository(db *pgxpool.Pool, logger zerolog.Logger) *QualityRuleRepository {
	return &QualityRuleRepository{db: db, logger: logger}
}

func (r *QualityRuleRepository) Create(ctx context.Context, item *model.QualityRule) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO quality_rules (
			id, tenant_id, model_id, name, description, rule_type, severity, column_name, config, schedule,
			enabled, last_run_at, last_status, consecutive_failures, tags, created_by, created_at, updated_at, deleted_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19
		)`,
		item.ID, item.TenantID, item.ModelID, item.Name, item.Description, item.RuleType, item.Severity, item.ColumnName, item.Config, item.Schedule,
		item.Enabled, item.LastRunAt, item.LastStatus, item.ConsecutiveFailures, ensureStringSlice(item.Tags), item.CreatedBy, item.CreatedAt, item.UpdatedAt, item.DeletedAt,
	)
	if err != nil {
		return fmt.Errorf("insert quality rule: %w", err)
	}
	return nil
}

func (r *QualityRuleRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.QualityRule, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, model_id, name, description, rule_type, severity, column_name, config, schedule,
		       enabled, last_run_at, last_status, consecutive_failures, tags, created_by, created_at, updated_at, deleted_at
		FROM quality_rules
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id,
	)
	return scanQualityRule(row)
}

func (r *QualityRuleRepository) List(ctx context.Context, tenantID uuid.UUID, params dto.ListQualityRulesParams) ([]*model.QualityRule, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.tenant_id, a.model_id, a.name, a.description, a.rule_type, a.severity, a.column_name, a.config, a.schedule,
		       a.enabled, a.last_run_at, a.last_status, a.consecutive_failures, a.tags, a.created_by, a.created_at, a.updated_at, a.deleted_at
		FROM quality_rules a`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.Where("a.deleted_at IS NULL")
	qb.WhereIf(params.ModelID != "", "a.model_id = ?", params.ModelID)
	qb.WhereIf(params.Severity != "", "a.severity = ?", params.Severity)
	qb.WhereIf(params.Status != "", "a.last_status = ?", params.Status)
	qb.WhereIf(params.Enabled != nil, "a.enabled = ?", *params.Enabled)
	qb.WhereIf(strings.TrimSpace(params.Search) != "", "a.name ILIKE ?", "%"+strings.TrimSpace(params.Search)+"%")
	qb.OrderBy(coalesce(params.Sort, "updated_at"), coalesce(params.Order, "desc"), []string{"name", "severity", "last_run_at", "updated_at", "created_at"})
	qb.Paginate(params.Page, params.PerPage)

	query, args := qb.Build()
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list quality rules: %w", err)
	}
	defer rows.Close()

	items := make([]*model.QualityRule, 0)
	for rows.Next() {
		item, err := scanQualityRule(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate quality rules: %w", err)
	}

	countQuery, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count quality rules: %w", err)
	}
	return items, total, nil
}

func (r *QualityRuleRepository) Update(ctx context.Context, item *model.QualityRule) error {
	result, err := r.db.Exec(ctx, `
		UPDATE quality_rules
		SET name = $3,
		    description = $4,
		    severity = $5,
		    column_name = $6,
		    config = $7,
		    schedule = $8,
		    enabled = $9,
		    tags = $10,
		    updated_at = $11
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		item.TenantID, item.ID, item.Name, item.Description, item.Severity, item.ColumnName, item.Config, item.Schedule,
		item.Enabled, ensureStringSlice(item.Tags), item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update quality rule: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *QualityRuleRepository) UpdateExecutionState(ctx context.Context, tenantID, ruleID uuid.UUID, lastRunAt time.Time, status model.QualityResultStatus, consecutiveFailures int) error {
	result, err := r.db.Exec(ctx, `
		UPDATE quality_rules
		SET last_run_at = $3,
		    last_status = $4,
		    consecutive_failures = $5,
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, ruleID, lastRunAt, status, consecutiveFailures,
	)
	if err != nil {
		return fmt.Errorf("update quality rule execution state: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *QualityRuleRepository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID, deletedAt time.Time) error {
	result, err := r.db.Exec(ctx, `
		UPDATE quality_rules
		SET deleted_at = $3, updated_at = $3
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id, deletedAt,
	)
	if err != nil {
		return fmt.Errorf("soft delete quality rule: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *QualityRuleRepository) ListEnabledByModel(ctx context.Context, tenantID, modelID uuid.UUID) ([]*model.QualityRule, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, model_id, name, description, rule_type, severity, column_name, config, schedule,
		       enabled, last_run_at, last_status, consecutive_failures, tags, created_by, created_at, updated_at, deleted_at
		FROM quality_rules
		WHERE tenant_id = $1 AND model_id = $2 AND enabled = true AND deleted_at IS NULL
		ORDER BY created_at ASC`,
		tenantID, modelID,
	)
	if err != nil {
		return nil, fmt.Errorf("list enabled quality rules by model: %w", err)
	}
	defer rows.Close()

	items := make([]*model.QualityRule, 0)
	for rows.Next() {
		item, err := scanQualityRule(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *QualityRuleRepository) ListDue(ctx context.Context, before time.Time, limit int) ([]*model.QualityRule, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, model_id, name, description, rule_type, severity, column_name, config, schedule,
		       enabled, last_run_at, last_status, consecutive_failures, tags, created_by, created_at, updated_at, deleted_at
		FROM quality_rules
		WHERE enabled = true
		  AND deleted_at IS NULL
		  AND schedule IS NOT NULL
		  AND (last_run_at IS NULL OR last_run_at <= $1)
		ORDER BY COALESCE(last_run_at, created_at) ASC
		LIMIT $2`,
		before, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list due quality rules: %w", err)
	}
	defer rows.Close()

	items := make([]*model.QualityRule, 0)
	for rows.Next() {
		item, err := scanQualityRule(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanQualityRule(scanner interface{ Scan(dest ...any) error }) (*model.QualityRule, error) {
	item := &model.QualityRule{}
	var configJSON []byte
	var lastStatus *string
	var tags []string
	if err := scanner.Scan(
		&item.ID, &item.TenantID, &item.ModelID, &item.Name, &item.Description, &item.RuleType, &item.Severity, &item.ColumnName, &configJSON, &item.Schedule,
		&item.Enabled, &item.LastRunAt, &lastStatus, &item.ConsecutiveFailures, &tags, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt,
	); err != nil {
		return nil, err
	}
	item.Tags = tags
	if lastStatus != nil {
		status := model.QualityResultStatus(*lastStatus)
		item.LastStatus = &status
	}
	if len(configJSON) > 0 && string(configJSON) != "null" {
		item.Config = configJSON
	}
	return item, nil
}
