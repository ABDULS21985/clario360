package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/visus/model"
)

type WidgetRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewWidgetRepository(db *pgxpool.Pool, logger zerolog.Logger) *WidgetRepository {
	return &WidgetRepository{db: db, logger: logger.With().Str("repo", "visus_widgets").Logger()}
}

func (r *WidgetRepository) Create(ctx context.Context, widget *model.Widget) (*model.Widget, error) {
	if widget == nil {
		return nil, ErrValidation
	}
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO visus_widgets (
			tenant_id, dashboard_id, title, subtitle, type, config, pos_x, pos_y, pos_w, pos_h, refresh_interval_seconds
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id`,
		widget.TenantID, widget.DashboardID, widget.Title, widget.Subtitle, widget.Type, marshalJSON(widget.Config),
		widget.Position.X, widget.Position.Y, widget.Position.W, widget.Position.H, widget.RefreshIntervalSeconds,
	).Scan(&id)
	if err != nil {
		return nil, wrapErr("create widget", err)
	}
	return r.GetByID(ctx, widget.TenantID, widget.DashboardID, id)
}

func (r *WidgetRepository) ListByDashboard(ctx context.Context, tenantID, dashboardID uuid.UUID) ([]model.Widget, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, dashboard_id, title, subtitle, type, config,
		       pos_x, pos_y, pos_w, pos_h, refresh_interval_seconds, created_at, updated_at
		FROM visus_widgets
		WHERE tenant_id = $1
		  AND dashboard_id = $2
		ORDER BY pos_y ASC, pos_x ASC, created_at ASC`, tenantID, dashboardID)
	if err != nil {
		return nil, wrapErr("list widgets", err)
	}
	defer rows.Close()
	items := make([]model.Widget, 0)
	for rows.Next() {
		item, err := scanWidget(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapErr("iterate widgets", err)
	}
	return items, nil
}

func (r *WidgetRepository) GetByID(ctx context.Context, tenantID, dashboardID, id uuid.UUID) (*model.Widget, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, dashboard_id, title, subtitle, type, config,
		       pos_x, pos_y, pos_w, pos_h, refresh_interval_seconds, created_at, updated_at
		FROM visus_widgets
		WHERE tenant_id = $1 AND dashboard_id = $2 AND id = $3`, tenantID, dashboardID, id)
	item, err := scanWidget(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return item, nil
}

func (r *WidgetRepository) Update(ctx context.Context, widget *model.Widget) (*model.Widget, error) {
	if widget == nil {
		return nil, ErrValidation
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE visus_widgets
		SET title = $4,
		    subtitle = $5,
		    config = $6,
		    pos_x = $7,
		    pos_y = $8,
		    pos_w = $9,
		    pos_h = $10,
		    refresh_interval_seconds = $11,
		    updated_at = now()
		WHERE tenant_id = $1 AND dashboard_id = $2 AND id = $3`,
		widget.TenantID, widget.DashboardID, widget.ID, widget.Title, widget.Subtitle, marshalJSON(widget.Config),
		widget.Position.X, widget.Position.Y, widget.Position.W, widget.Position.H, widget.RefreshIntervalSeconds,
	)
	if err != nil {
		return nil, wrapErr("update widget", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, widget.TenantID, widget.DashboardID, widget.ID)
}

func (r *WidgetRepository) Delete(ctx context.Context, tenantID, dashboardID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM visus_widgets WHERE tenant_id = $1 AND dashboard_id = $2 AND id = $3`, tenantID, dashboardID, id)
	if err != nil {
		return wrapErr("delete widget", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *WidgetRepository) UpdateLayout(ctx context.Context, tenantID, dashboardID uuid.UUID, positions map[uuid.UUID]model.WidgetPosition) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return wrapErr("begin layout tx", err)
	}
	defer tx.Rollback(ctx)
	for id, pos := range positions {
		tag, execErr := tx.Exec(ctx, `
			UPDATE visus_widgets
			SET pos_x = $4, pos_y = $5, pos_w = $6, pos_h = $7, updated_at = now()
			WHERE tenant_id = $1 AND dashboard_id = $2 AND id = $3`,
			tenantID, dashboardID, id, pos.X, pos.Y, pos.W, pos.H,
		)
		if execErr != nil {
			return wrapErr("update widget layout", execErr)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return wrapErr("commit layout tx", err)
	}
	return nil
}

func (r *WidgetRepository) CountByType(ctx context.Context, tenantID uuid.UUID) (map[string]int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT type, COUNT(*)
		FROM visus_widgets
		WHERE tenant_id = $1
		GROUP BY type`, tenantID)
	if err != nil {
		return nil, wrapErr("count widgets", err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var typ string
		var count int
		if err := rows.Scan(&typ, &count); err != nil {
			return nil, wrapErr("scan widget counts", err)
		}
		out[typ] = count
	}
	return out, rows.Err()
}

func scanWidget(row interface{ Scan(...any) error }) (*model.Widget, error) {
	item := &model.Widget{}
	var config []byte
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.DashboardID,
		&item.Title,
		&item.Subtitle,
		&item.Type,
		&config,
		&item.Position.X,
		&item.Position.Y,
		&item.Position.W,
		&item.Position.H,
		&item.RefreshIntervalSeconds,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, wrapErr("scan widget", err)
	}
	item.Config = unmarshalMap(config)
	return item, nil
}
