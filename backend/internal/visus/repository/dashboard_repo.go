package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/visus/model"
)

type DashboardRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewDashboardRepository(db *pgxpool.Pool, logger zerolog.Logger) *DashboardRepository {
	return &DashboardRepository{db: db, logger: logger.With().Str("repo", "visus_dashboards").Logger()}
}

func (r *DashboardRepository) Create(ctx context.Context, dashboard *model.Dashboard) (*model.Dashboard, error) {
	if dashboard == nil {
		return nil, ErrValidation
	}
	normalizeDashboardFields(dashboard)
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO visus_dashboards (
			tenant_id, name, description, grid_columns, visibility, shared_with, is_default, is_system,
			tags, metadata, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id`,
		dashboard.TenantID, dashboard.Name, dashboard.Description, dashboard.GridColumns, dashboard.Visibility,
		dashboard.SharedWith, dashboard.IsDefault, dashboard.IsSystem, dashboard.Tags, marshalJSON(dashboard.Metadata), dashboard.CreatedBy,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, wrapErr("create dashboard", err)
	}
	return r.GetByID(ctx, dashboard.TenantID, nil, id)
}

func (r *DashboardRepository) ListAccessible(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, page, perPage int, sortCol, sortDir, search, visibility string) ([]model.Dashboard, int, error) {
	meta := normalizePagination(page, perPage)
	args := []any{tenantID}
	whereClauses := []string{
		"tenant_id = $1",
		"deleted_at IS NULL",
	}
	if userID != nil {
		args = append(args, *userID)
		whereClauses = append(whereClauses, "(created_by = $2 OR $2 = ANY(shared_with) OR visibility IN ('organization','public','team'))")
	} else {
		whereClauses = append(whereClauses, "visibility IN ('organization','public','team')")
	}
	if search != "" {
		args = append(args, "%"+search+"%")
		position := len(args)
		whereClauses = append(whereClauses, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", position, position))
	}
	if visibility != "" {
		args = append(args, visibility)
		position := len(args)
		whereClauses = append(whereClauses, fmt.Sprintf("visibility = $%d", position))
	}
	orderClause := fmt.Sprintf("is_default DESC, %s %s", sortCol, sortDir)
	args = append(args, meta.Limit, meta.Offset)
	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, description, grid_columns, visibility, shared_with, is_default, is_system,
		       tags, metadata, created_by, created_at, updated_at, deleted_at,
		       COALESCE((SELECT COUNT(*) FROM visus_widgets w WHERE w.dashboard_id = visus_dashboards.id), 0)
		FROM visus_dashboards
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`,
		strings.Join(whereClauses, " AND "),
		orderClause,
		len(args)-1,
		len(args),
	)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, wrapErr("list dashboards", err)
	}
	defer rows.Close()

	items := make([]model.Dashboard, 0, meta.Limit)
	for rows.Next() {
		item, widgetCount, err := scanDashboard(rows)
		if err != nil {
			return nil, 0, err
		}
		item.WidgetCount = widgetCount
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, wrapErr("iterate dashboards", err)
	}

	countArgs := args[:len(args)-2]
	countQuery := fmt.Sprintf(
		`SELECT COUNT(*) FROM visus_dashboards WHERE %s`,
		strings.Join(whereClauses, " AND "),
	)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, wrapErr("count dashboards", err)
	}
	return items, total, nil
}

func (r *DashboardRepository) GetByID(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, id uuid.UUID) (*model.Dashboard, error) {
	args := []any{tenantID, id}
	accessClause := ``
	if userID != nil {
		accessClause = `AND (created_by = $3 OR $3 = ANY(shared_with) OR visibility IN ('organization','public','team'))`
		args = []any{tenantID, id, *userID}
	}
	row := r.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT id, tenant_id, name, description, grid_columns, visibility, shared_with, is_default, is_system,
		       tags, metadata, created_by, created_at, updated_at, deleted_at,
		       COALESCE((SELECT COUNT(*) FROM visus_widgets w WHERE w.dashboard_id = visus_dashboards.id), 0)
		FROM visus_dashboards
		WHERE tenant_id = $1
		  AND id = $2
		  AND deleted_at IS NULL
		  %s`, accessClause), args...)
	item, widgetCount, err := scanDashboard(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	item.WidgetCount = widgetCount
	return item, nil
}

func (r *DashboardRepository) Update(ctx context.Context, dashboard *model.Dashboard) (*model.Dashboard, error) {
	if dashboard == nil {
		return nil, ErrValidation
	}
	normalizeDashboardFields(dashboard)
	tag, err := r.db.Exec(ctx, `
		UPDATE visus_dashboards
		SET name = $3,
		    description = $4,
		    grid_columns = $5,
		    visibility = $6,
		    shared_with = $7,
		    is_default = $8,
		    tags = $9,
		    metadata = $10,
		    updated_at = now()
		WHERE tenant_id = $1
		  AND id = $2
		  AND deleted_at IS NULL`,
		dashboard.TenantID, dashboard.ID, dashboard.Name, dashboard.Description, dashboard.GridColumns, dashboard.Visibility,
		dashboard.SharedWith, dashboard.IsDefault, dashboard.Tags, marshalJSON(dashboard.Metadata),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, wrapErr("update dashboard", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, dashboard.TenantID, nil, dashboard.ID)
}

func (r *DashboardRepository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE visus_dashboards
		SET deleted_at = now(), updated_at = now()
		WHERE tenant_id = $1
		  AND id = $2
		  AND deleted_at IS NULL`, tenantID, id,
	)
	if err != nil {
		return wrapErr("delete dashboard", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *DashboardRepository) ClearDefault(ctx context.Context, tenantID uuid.UUID, exceptID *uuid.UUID) error {
	if exceptID == nil {
		_, err := r.db.Exec(ctx, `UPDATE visus_dashboards SET is_default = false, updated_at = now() WHERE tenant_id = $1 AND deleted_at IS NULL`, tenantID)
		return wrapErr("clear default dashboard", err)
	}
	_, err := r.db.Exec(ctx, `UPDATE visus_dashboards SET is_default = false, updated_at = now() WHERE tenant_id = $1 AND id <> $2 AND deleted_at IS NULL`, tenantID, *exceptID)
	return wrapErr("clear default dashboard", err)
}

func (r *DashboardRepository) CountByVisibility(ctx context.Context, tenantID uuid.UUID) (map[string]int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT visibility, COUNT(*)
		FROM visus_dashboards
		WHERE tenant_id = $1 AND deleted_at IS NULL
		GROUP BY visibility`, tenantID)
	if err != nil {
		return nil, wrapErr("count dashboards by visibility", err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var visibility string
		var count int
		if err := rows.Scan(&visibility, &count); err != nil {
			return nil, wrapErr("scan visibility counts", err)
		}
		out[visibility] = count
	}
	return out, rows.Err()
}

func (r *DashboardRepository) ListTenantIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT tenant_id
		FROM visus_dashboards
		WHERE deleted_at IS NULL
		ORDER BY tenant_id`)
	if err != nil {
		return nil, wrapErr("list dashboard tenant ids", err)
	}
	defer rows.Close()

	out := make([]uuid.UUID, 0)
	for rows.Next() {
		var tenantID uuid.UUID
		if err := rows.Scan(&tenantID); err != nil {
			return nil, wrapErr("scan dashboard tenant id", err)
		}
		out = append(out, tenantID)
	}
	return out, rows.Err()
}

func scanDashboard(row interface{ Scan(...any) error }) (*model.Dashboard, int, error) {
	item := &model.Dashboard{}
	var metadata []byte
	var widgetCount int
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.Name,
		&item.Description,
		&item.GridColumns,
		&item.Visibility,
		&item.SharedWith,
		&item.IsDefault,
		&item.IsSystem,
		&item.Tags,
		&metadata,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.DeletedAt,
		&widgetCount,
	); err != nil {
		return nil, 0, err
	}
	item.Metadata = unmarshalMap(metadata)
	if item.SharedWith == nil {
		item.SharedWith = []uuid.UUID{}
	}
	if item.Tags == nil {
		item.Tags = []string{}
	}
	return item, widgetCount, nil
}

func normalizeDashboardFields(dashboard *model.Dashboard) {
	if dashboard.SharedWith == nil {
		dashboard.SharedWith = []uuid.UUID{}
	}
	if dashboard.Tags == nil {
		dashboard.Tags = []string{}
	}
	if dashboard.Metadata == nil {
		dashboard.Metadata = map[string]any{}
	}
}
