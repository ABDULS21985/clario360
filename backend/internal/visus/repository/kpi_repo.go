package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/visus/model"
)

type KPIRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewKPIRepository(db *pgxpool.Pool, logger zerolog.Logger) *KPIRepository {
	return &KPIRepository{db: db, logger: logger.With().Str("repo", "visus_kpis").Logger()}
}

func (r *KPIRepository) Create(ctx context.Context, item *model.KPIDefinition) (*model.KPIDefinition, error) {
	if item == nil {
		return nil, ErrValidation
	}
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO visus_kpi_definitions (
			tenant_id, name, description, category, suite, icon, query_endpoint, query_params, value_path, unit,
			format_pattern, target_value, warning_threshold, critical_threshold, direction, calculation_type,
			calculation_window, snapshot_frequency, enabled, is_default, tags, created_by
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
			$11,$12,$13,$14,$15,$16,
			$17,$18,$19,$20,$21,$22
		)
		RETURNING id`,
		item.TenantID, item.Name, item.Description, item.Category, item.Suite, item.Icon, item.QueryEndpoint, marshalJSON(item.QueryParams),
		item.ValuePath, item.Unit, item.FormatPattern, item.TargetValue, item.WarningThreshold, item.CriticalThreshold,
		item.Direction, item.CalculationType, item.CalculationWindow, item.SnapshotFrequency, item.Enabled, item.IsDefault, item.Tags, item.CreatedBy,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, wrapErr("create kpi", err)
	}
	return r.Get(ctx, item.TenantID, id)
}

func (r *KPIRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.KPIDefinition, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, description, category, suite, icon, query_endpoint, query_params, value_path, unit,
		       format_pattern, target_value, warning_threshold, critical_threshold, direction, calculation_type,
		       calculation_window, snapshot_frequency, enabled, is_default, last_snapshot_at, last_value, last_status,
		       tags, created_by, created_at, updated_at, deleted_at
		FROM visus_kpi_definitions
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	item, err := scanKPI(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return item, nil
}

func (r *KPIRepository) List(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]model.KPIDefinition, int, error) {
	meta := normalizePagination(page, perPage)
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, name, description, category, suite, icon, query_endpoint, query_params, value_path, unit,
		       format_pattern, target_value, warning_threshold, critical_threshold, direction, calculation_type,
		       calculation_window, snapshot_frequency, enabled, is_default, last_snapshot_at, last_value, last_status,
		       tags, created_by, created_at, updated_at, deleted_at
		FROM visus_kpi_definitions
		WHERE tenant_id = $1 AND deleted_at IS NULL
		ORDER BY category, name
		LIMIT $2 OFFSET $3`, tenantID, meta.Limit, meta.Offset)
	if err != nil {
		return nil, 0, wrapErr("list kpis", err)
	}
	defer rows.Close()
	items := make([]model.KPIDefinition, 0, meta.Limit)
	for rows.Next() {
		item, err := scanKPI(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, wrapErr("iterate kpis", err)
	}
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM visus_kpi_definitions WHERE tenant_id = $1 AND deleted_at IS NULL`, tenantID).Scan(&total); err != nil {
		return nil, 0, wrapErr("count kpis", err)
	}
	return items, total, nil
}

func (r *KPIRepository) ListEnabled(ctx context.Context, tenantID uuid.UUID) ([]model.KPIDefinition, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, name, description, category, suite, icon, query_endpoint, query_params, value_path, unit,
		       format_pattern, target_value, warning_threshold, critical_threshold, direction, calculation_type,
		       calculation_window, snapshot_frequency, enabled, is_default, last_snapshot_at, last_value, last_status,
		       tags, created_by, created_at, updated_at, deleted_at
		FROM visus_kpi_definitions
		WHERE tenant_id = $1 AND enabled = true AND deleted_at IS NULL
		ORDER BY name`, tenantID)
	if err != nil {
		return nil, wrapErr("list enabled kpis", err)
	}
	defer rows.Close()
	out := make([]model.KPIDefinition, 0)
	for rows.Next() {
		item, err := scanKPI(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (r *KPIRepository) Update(ctx context.Context, item *model.KPIDefinition) (*model.KPIDefinition, error) {
	if item == nil {
		return nil, ErrValidation
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE visus_kpi_definitions
		SET name = $3,
		    description = $4,
		    category = $5,
		    suite = $6,
		    icon = $7,
		    query_endpoint = $8,
		    query_params = $9,
		    value_path = $10,
		    unit = $11,
		    format_pattern = $12,
		    target_value = $13,
		    warning_threshold = $14,
		    critical_threshold = $15,
		    direction = $16,
		    calculation_type = $17,
		    calculation_window = $18,
		    snapshot_frequency = $19,
		    enabled = $20,
		    tags = $21,
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		item.TenantID, item.ID, item.Name, item.Description, item.Category, item.Suite, item.Icon,
		item.QueryEndpoint, marshalJSON(item.QueryParams), item.ValuePath, item.Unit, item.FormatPattern,
		item.TargetValue, item.WarningThreshold, item.CriticalThreshold, item.Direction, item.CalculationType,
		item.CalculationWindow, item.SnapshotFrequency, item.Enabled, item.Tags,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, wrapErr("update kpi", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.Get(ctx, item.TenantID, item.ID)
}

func (r *KPIRepository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `UPDATE visus_kpi_definitions SET deleted_at = now(), updated_at = now() WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	if err != nil {
		return wrapErr("delete kpi", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *KPIRepository) UpdateSnapshotState(ctx context.Context, tenantID, id uuid.UUID, at time.Time, value float64, status model.KPIStatus) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE visus_kpi_definitions
		SET last_snapshot_at = $3,
		    last_value = $4,
		    last_status = $5,
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id, at, value, status,
	)
	if err != nil {
		return wrapErr("update kpi snapshot state", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *KPIRepository) ListDueTenantIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT tenant_id
		FROM visus_kpi_definitions
		WHERE enabled = true
		  AND deleted_at IS NULL
		  AND (
		      last_snapshot_at IS NULL OR
		      last_snapshot_at < now() - CASE snapshot_frequency
		          WHEN 'every_15m' THEN interval '15 minutes'
		          WHEN 'hourly' THEN interval '1 hour'
		          WHEN 'every_4h' THEN interval '4 hours'
		          WHEN 'daily' THEN interval '1 day'
		          WHEN 'weekly' THEN interval '7 days'
		      END
		  )`)
	if err != nil {
		return nil, wrapErr("list due kpi tenants", err)
	}
	defer rows.Close()
	out := make([]uuid.UUID, 0)
	for rows.Next() {
		var tenantID uuid.UUID
		if err := rows.Scan(&tenantID); err != nil {
			return nil, wrapErr("scan kpi tenant", err)
		}
		out = append(out, tenantID)
	}
	return out, rows.Err()
}

func (r *KPIRepository) CountBySuite(ctx context.Context, tenantID uuid.UUID) (map[string]map[string]int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT suite, enabled, COUNT(*)
		FROM visus_kpi_definitions
		WHERE tenant_id = $1 AND deleted_at IS NULL
		GROUP BY suite, enabled`, tenantID)
	if err != nil {
		return nil, wrapErr("count kpis by suite", err)
	}
	defer rows.Close()
	out := map[string]map[string]int{}
	for rows.Next() {
		var suite string
		var enabled bool
		var count int
		if err := rows.Scan(&suite, &enabled, &count); err != nil {
			return nil, wrapErr("scan suite kpi counts", err)
		}
		if out[suite] == nil {
			out[suite] = map[string]int{}
		}
		out[suite][fmt.Sprintf("%t", enabled)] = count
	}
	return out, rows.Err()
}

func scanKPI(row interface{ Scan(...any) error }) (*model.KPIDefinition, error) {
	item := &model.KPIDefinition{}
	var queryParams []byte
	var lastStatus *string
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.Name,
		&item.Description,
		&item.Category,
		&item.Suite,
		&item.Icon,
		&item.QueryEndpoint,
		&queryParams,
		&item.ValuePath,
		&item.Unit,
		&item.FormatPattern,
		&item.TargetValue,
		&item.WarningThreshold,
		&item.CriticalThreshold,
		&item.Direction,
		&item.CalculationType,
		&item.CalculationWindow,
		&item.SnapshotFrequency,
		&item.Enabled,
		&item.IsDefault,
		&item.LastSnapshotAt,
		&item.LastValue,
		&lastStatus,
		&item.Tags,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.DeletedAt,
	); err != nil {
		return nil, wrapErr("scan kpi", err)
	}
	item.QueryParams = unmarshalMap(queryParams)
	if lastStatus != nil {
		status := model.KPIStatus(*lastStatus)
		item.LastStatus = &status
	}
	if item.Tags == nil {
		item.Tags = []string{}
	}
	return item, nil
}
