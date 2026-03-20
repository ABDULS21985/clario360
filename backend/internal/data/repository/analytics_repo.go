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

type AnalyticsRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewAnalyticsRepository(db *pgxpool.Pool, logger zerolog.Logger) *AnalyticsRepository {
	return &AnalyticsRepository{db: db, logger: logger}
}

func (r *AnalyticsRepository) CreateSavedQuery(ctx context.Context, item *model.SavedQuery) error {
	queryJSON, _ := json.Marshal(item.QueryDefinition)
	_, err := r.db.Exec(ctx, `
		INSERT INTO saved_queries (
			id, tenant_id, name, description, model_id, query_definition, last_run_at, run_count, visibility, tags,
			created_by, created_at, updated_at, deleted_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14
		)`,
		item.ID, item.TenantID, item.Name, item.Description, item.ModelID, queryJSON, item.LastRunAt, item.RunCount, item.Visibility, ensureStringSlice(item.Tags),
		item.CreatedBy, item.CreatedAt, item.UpdatedAt, item.DeletedAt,
	)
	if err != nil {
		return fmt.Errorf("insert saved query: %w", err)
	}
	return nil
}

func (r *AnalyticsRepository) GetSavedQuery(ctx context.Context, tenantID, id uuid.UUID) (*model.SavedQuery, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, description, model_id, query_definition, last_run_at, run_count, visibility, tags,
		       created_by, created_at, updated_at, deleted_at
		FROM saved_queries
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id,
	)
	return scanSavedQuery(row)
}

func (r *AnalyticsRepository) ListSavedQueries(ctx context.Context, tenantID, userID uuid.UUID, params dto.ListSavedQueriesParams) ([]*model.SavedQuery, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.tenant_id, a.name, a.description, a.model_id, a.query_definition, a.last_run_at, a.run_count, a.visibility, a.tags,
		       a.created_by, a.created_at, a.updated_at, a.deleted_at
		FROM saved_queries a`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.Where("a.deleted_at IS NULL")
	qb.WhereIf(params.ModelID != "", "a.model_id = ?", params.ModelID)
	qb.WhereIf(strings.TrimSpace(params.Search) != "", "a.name ILIKE ?", "%"+strings.TrimSpace(params.Search)+"%")
	switch params.Visibility {
	case string(model.SavedQueryVisibilityPrivate):
		qb.Where("a.visibility = ?", model.SavedQueryVisibilityPrivate)
		qb.Where("a.created_by = ?", userID)
	case string(model.SavedQueryVisibilityTeam), string(model.SavedQueryVisibilityOrganization):
		qb.Where("a.visibility = ?", params.Visibility)
	default:
		qb.Where("(a.visibility <> ? OR a.created_by = ?)", model.SavedQueryVisibilityPrivate, userID)
	}
	qb.OrderBy(coalesce(params.Sort, "updated_at"), coalesce(params.Order, "desc"), []string{"name", "created_at", "updated_at", "run_count"})
	qb.Paginate(params.Page, params.PerPage)
	query, args := qb.Build()
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list saved queries: %w", err)
	}
	defer rows.Close()

	items := make([]*model.SavedQuery, 0)
	for rows.Next() {
		item, err := scanSavedQuery(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate saved queries: %w", err)
	}

	countQuery, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count saved queries: %w", err)
	}
	return items, total, nil
}

func (r *AnalyticsRepository) UpdateSavedQuery(ctx context.Context, item *model.SavedQuery) error {
	queryJSON, _ := json.Marshal(item.QueryDefinition)
	result, err := r.db.Exec(ctx, `
		UPDATE saved_queries
		SET description = $4,
		    query_definition = $5,
		    visibility = $6,
		    tags = $7,
		    updated_at = $8,
		    last_run_at = $9,
		    run_count = $10
		WHERE tenant_id = $1 AND id = $2 AND created_by = $3 AND deleted_at IS NULL`,
		item.TenantID, item.ID, item.CreatedBy, item.Description, queryJSON, item.Visibility, ensureStringSlice(item.Tags), item.UpdatedAt, item.LastRunAt, item.RunCount,
	)
	if err != nil {
		return fmt.Errorf("update saved query: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *AnalyticsRepository) SoftDeleteSavedQuery(ctx context.Context, tenantID, id, userID uuid.UUID, deletedAt time.Time) error {
	result, err := r.db.Exec(ctx, `
		UPDATE saved_queries
		SET deleted_at = $4, updated_at = $4
		WHERE tenant_id = $1 AND id = $2 AND created_by = $3 AND deleted_at IS NULL`,
		tenantID, id, userID, deletedAt,
	)
	if err != nil {
		return fmt.Errorf("delete saved query: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *AnalyticsRepository) TouchSavedQueryRun(ctx context.Context, tenantID, id uuid.UUID, ranAt time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE saved_queries
		SET last_run_at = $3, run_count = run_count + 1, updated_at = $3
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id, ranAt,
	)
	if err != nil {
		return fmt.Errorf("touch saved query run: %w", err)
	}
	return nil
}

func (r *AnalyticsRepository) CreateAuditLog(ctx context.Context, item *model.AnalyticsAuditLog) error {
	queryJSON, _ := json.Marshal(item.QueryDefinition)
	_, err := r.db.Exec(ctx, `
		INSERT INTO analytics_audit_log (
			id, tenant_id, user_id, model_id, source_id, query_definition, columns_accessed, filters_applied,
			data_classification, pii_columns_accessed, pii_masking_applied, rows_returned, truncated, execution_time_ms,
			error_occurred, error_message, saved_query_id, ip_address, user_agent, executed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20
		)`,
		item.ID, item.TenantID, item.UserID, item.ModelID, item.SourceID, queryJSON, ensureStringSlice(item.ColumnsAccessed), item.FiltersApplied,
		item.DataClassification, ensureStringSlice(item.PIIColumnsAccessed), item.PIIMaskingApplied, item.RowsReturned, item.Truncated, item.ExecutionTimeMs,
		item.ErrorOccurred, item.ErrorMessage, item.SavedQueryID, item.IPAddress, item.UserAgent, item.ExecutedAt,
	)
	if err != nil {
		return fmt.Errorf("insert analytics audit log: %w", err)
	}
	return nil
}

func (r *AnalyticsRepository) ListAuditLogs(ctx context.Context, tenantID uuid.UUID, params dto.ListAnalyticsAuditParams) ([]*model.AnalyticsAuditLog, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id, a.tenant_id, a.user_id, a.model_id, a.source_id, a.query_definition, a.columns_accessed, a.filters_applied,
		       a.data_classification, a.pii_columns_accessed, a.pii_masking_applied, a.rows_returned, a.truncated, a.execution_time_ms,
		       a.error_occurred, a.error_message, a.saved_query_id, a.ip_address, a.user_agent, a.executed_at
		FROM analytics_audit_log a`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.WhereIf(params.ModelID != "", "a.model_id = ?", params.ModelID)
	qb.WhereIf(params.UserID != "", "a.user_id = ?", params.UserID)
	qb.WhereIf(params.Classification != "", "a.data_classification = ?", params.Classification)
	if params.PIIAccessed != nil {
		if *params.PIIAccessed {
			qb.Where("cardinality(a.pii_columns_accessed) > 0")
		} else {
			qb.Where("cardinality(a.pii_columns_accessed) = 0")
		}
	}
	qb.OrderBy(coalesce(params.Sort, "executed_at"), coalesce(params.Order, "desc"), []string{"executed_at", "rows_returned"})
	qb.Paginate(params.Page, params.PerPage)
	query, args := qb.Build()
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list analytics audit logs: %w", err)
	}
	defer rows.Close()

	items := make([]*model.AnalyticsAuditLog, 0)
	for rows.Next() {
		item, err := scanAnalyticsAuditLog(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate analytics audit logs: %w", err)
	}
	countQuery, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count analytics audit logs: %w", err)
	}
	return items, total, nil
}

type savedQueryScanner interface {
	Scan(dest ...any) error
}

func scanSavedQuery(scanner savedQueryScanner) (*model.SavedQuery, error) {
	item := &model.SavedQuery{}
	var queryJSON []byte
	var tags []string
	if err := scanner.Scan(
		&item.ID, &item.TenantID, &item.Name, &item.Description, &item.ModelID, &queryJSON, &item.LastRunAt, &item.RunCount, &item.Visibility, &tags,
		&item.CreatedBy, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt,
	); err != nil {
		return nil, err
	}
	item.Tags = tags
	if err := json.Unmarshal(queryJSON, &item.QueryDefinition); err != nil {
		return nil, fmt.Errorf("decode saved query definition: %w", err)
	}
	return item, nil
}

type analyticsAuditScanner interface {
	Scan(dest ...any) error
}

func scanAnalyticsAuditLog(scanner analyticsAuditScanner) (*model.AnalyticsAuditLog, error) {
	item := &model.AnalyticsAuditLog{}
	var queryJSON []byte
	var columns []string
	var piiColumns []string
	var filtersJSON []byte
	if err := scanner.Scan(
		&item.ID, &item.TenantID, &item.UserID, &item.ModelID, &item.SourceID, &queryJSON, &columns, &filtersJSON,
		&item.DataClassification, &piiColumns, &item.PIIMaskingApplied, &item.RowsReturned, &item.Truncated, &item.ExecutionTimeMs,
		&item.ErrorOccurred, &item.ErrorMessage, &item.SavedQueryID, &item.IPAddress, &item.UserAgent, &item.ExecutedAt,
	); err != nil {
		return nil, err
	}
	item.ColumnsAccessed = columns
	item.PIIColumnsAccessed = piiColumns
	item.FiltersApplied = filtersJSON
	if err := json.Unmarshal(queryJSON, &item.QueryDefinition); err != nil {
		return nil, fmt.Errorf("decode analytics audit query definition: %w", err)
	}
	return item, nil
}
