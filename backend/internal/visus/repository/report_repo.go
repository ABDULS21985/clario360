package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/visus/model"
)

type ReportRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewReportRepository(db *pgxpool.Pool, logger zerolog.Logger) *ReportRepository {
	return &ReportRepository{db: db, logger: logger.With().Str("repo", "visus_reports").Logger()}
}

func (r *ReportRepository) Create(ctx context.Context, item *model.ReportDefinition) (*model.ReportDefinition, error) {
	if item == nil {
		return nil, ErrValidation
	}
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO visus_report_definitions (
			tenant_id, name, description, report_type, sections, period, custom_period_start, custom_period_end,
			schedule, next_run_at, recipients, auto_send, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id`,
		item.TenantID, item.Name, item.Description, item.ReportType, item.Sections, item.Period,
		dateOnly(item.CustomPeriodStart), dateOnly(item.CustomPeriodEnd), item.Schedule, item.NextRunAt, item.Recipients,
		item.AutoSend, item.CreatedBy,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, wrapErr("create report", err)
	}
	return r.Get(ctx, item.TenantID, id)
}

func (r *ReportRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.ReportDefinition, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, description, report_type, sections, period, custom_period_start, custom_period_end,
		       schedule, next_run_at, recipients, auto_send, last_generated_at, total_generated, created_by,
		       created_at, updated_at, deleted_at
		FROM visus_report_definitions
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	item, err := scanReport(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return item, nil
}

func (r *ReportRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ReportDefinition, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, description, report_type, sections, period, custom_period_start, custom_period_end,
		       schedule, next_run_at, recipients, auto_send, last_generated_at, total_generated, created_by,
		       created_at, updated_at, deleted_at
		FROM visus_report_definitions
		WHERE id = $1 AND deleted_at IS NULL`, id)
	item, err := scanReport(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return item, nil
}

func (r *ReportRepository) List(ctx context.Context, tenantID uuid.UUID, page, perPage int, sortCol, sortDir, search, reportType string, autoSend *bool) ([]model.ReportDefinition, int, error) {
	meta := normalizePagination(page, perPage)
	orderClause := fmt.Sprintf("%s %s", sortCol, sortDir)
	args := []any{tenantID}
	whereClauses := []string{
		"tenant_id = $1",
		"deleted_at IS NULL",
	}
	if search != "" {
		args = append(args, "%"+search+"%")
		position := len(args)
		whereClauses = append(whereClauses, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", position, position))
	}
	if reportType != "" {
		args = append(args, reportType)
		position := len(args)
		whereClauses = append(whereClauses, fmt.Sprintf("report_type = $%d", position))
	}
	if autoSend != nil {
		args = append(args, *autoSend)
		position := len(args)
		whereClauses = append(whereClauses, fmt.Sprintf("auto_send = $%d", position))
	}
	args = append(args, meta.Limit, meta.Offset)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT id, tenant_id, name, description, report_type, sections, period, custom_period_start, custom_period_end,
		       schedule, next_run_at, recipients, auto_send, last_generated_at, total_generated, created_by,
		       created_at, updated_at, deleted_at
		FROM visus_report_definitions
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`,
		strings.Join(whereClauses, " AND "),
		orderClause,
		len(args)-1,
		len(args),
	), args...)
	if err != nil {
		return nil, 0, wrapErr("list reports", err)
	}
	defer rows.Close()
	items := make([]model.ReportDefinition, 0, meta.Limit)
	for rows.Next() {
		item, err := scanReport(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, wrapErr("iterate reports", err)
	}
	var total int
	if err := r.db.QueryRow(
		ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM visus_report_definitions WHERE %s`, strings.Join(whereClauses, " AND ")),
		args[:len(args)-2]...,
	).Scan(&total); err != nil {
		return nil, 0, wrapErr("count reports", err)
	}
	return items, total, nil
}

func (r *ReportRepository) Update(ctx context.Context, item *model.ReportDefinition) (*model.ReportDefinition, error) {
	if item == nil {
		return nil, ErrValidation
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE visus_report_definitions
		SET name = $3,
		    description = $4,
		    report_type = $5,
		    sections = $6,
		    period = $7,
		    custom_period_start = $8,
		    custom_period_end = $9,
		    schedule = $10,
		    next_run_at = $11,
		    recipients = $12,
		    auto_send = $13,
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		item.TenantID, item.ID, item.Name, item.Description, item.ReportType, item.Sections, item.Period,
		dateOnly(item.CustomPeriodStart), dateOnly(item.CustomPeriodEnd), item.Schedule, item.NextRunAt, item.Recipients, item.AutoSend,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, wrapErr("update report", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.Get(ctx, item.TenantID, item.ID)
}

func (r *ReportRepository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `UPDATE visus_report_definitions SET deleted_at = now(), updated_at = now() WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	if err != nil {
		return wrapErr("delete report", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ReportRepository) UpdateGeneration(ctx context.Context, tenantID, id uuid.UUID, generatedAt time.Time, nextRunAt *time.Time) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE visus_report_definitions
		SET last_generated_at = $3,
		    total_generated = total_generated + 1,
		    next_run_at = $4,
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id, generatedAt, nextRunAt,
	)
	if err != nil {
		return wrapErr("update report generation", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ReportRepository) ListDue(ctx context.Context, now time.Time, limit int) ([]model.ReportDefinition, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, name, description, report_type, sections, period, custom_period_start, custom_period_end,
		       schedule, next_run_at, recipients, auto_send, last_generated_at, total_generated, created_by,
		       created_at, updated_at, deleted_at
		FROM visus_report_definitions
		WHERE schedule IS NOT NULL
		  AND next_run_at IS NOT NULL
		  AND next_run_at <= $1
		  AND deleted_at IS NULL
		ORDER BY next_run_at ASC
		LIMIT $2`, now, limit)
	if err != nil {
		return nil, wrapErr("list due reports", err)
	}
	defer rows.Close()
	out := make([]model.ReportDefinition, 0, limit)
	for rows.Next() {
		item, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (r *ReportRepository) CountByType(ctx context.Context, tenantID uuid.UUID) (map[string]int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT report_type, COUNT(*)
		FROM visus_report_definitions
		WHERE tenant_id = $1 AND deleted_at IS NULL
		GROUP BY report_type`, tenantID)
	if err != nil {
		return nil, wrapErr("count reports by type", err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var typ string
		var count int
		if err := rows.Scan(&typ, &count); err != nil {
			return nil, wrapErr("scan report counts", err)
		}
		out[typ] = count
	}
	return out, rows.Err()
}

func scanReport(row interface{ Scan(...any) error }) (*model.ReportDefinition, error) {
	item := &model.ReportDefinition{}
	var typ string
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.Name,
		&item.Description,
		&typ,
		&item.Sections,
		&item.Period,
		&item.CustomPeriodStart,
		&item.CustomPeriodEnd,
		&item.Schedule,
		&item.NextRunAt,
		&item.Recipients,
		&item.AutoSend,
		&item.LastGeneratedAt,
		&item.TotalGenerated,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.DeletedAt,
	); err != nil {
		return nil, wrapErr("scan report", err)
	}
	item.ReportType = model.ReportType(typ)
	if item.Sections == nil {
		item.Sections = []string{}
	}
	if item.Recipients == nil {
		item.Recipients = []uuid.UUID{}
	}
	return item, nil
}

func dateOnly(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	normalized := time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
	return &normalized
}
