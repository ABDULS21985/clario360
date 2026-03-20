package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/visus/model"
)

type KPISnapshotRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewKPISnapshotRepository(db *pgxpool.Pool, logger zerolog.Logger) *KPISnapshotRepository {
	return &KPISnapshotRepository{db: db, logger: logger.With().Str("repo", "visus_kpi_snapshots").Logger()}
}

func (r *KPISnapshotRepository) Create(ctx context.Context, item *model.KPISnapshot) (*model.KPISnapshot, error) {
	if item == nil {
		return nil, ErrValidation
	}
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO visus_kpi_snapshots (
			tenant_id, kpi_id, value, previous_value, delta, delta_percent, status, period_start, period_end,
			fetch_success, fetch_latency_ms, fetch_error, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id`,
		item.TenantID, item.KPIID, item.Value, item.PreviousValue, item.Delta, item.DeltaPercent, item.Status,
		item.PeriodStart, item.PeriodEnd, item.FetchSuccess, item.FetchLatencyMS, item.FetchError, createdAt,
	).Scan(&id)
	if err != nil {
		return nil, wrapErr("create kpi snapshot", err)
	}
	return r.Get(ctx, item.TenantID, id)
}

func (r *KPISnapshotRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.KPISnapshot, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, kpi_id, value, previous_value, delta, delta_percent, status, period_start, period_end,
		       fetch_success, fetch_latency_ms, fetch_error, created_at
		FROM visus_kpi_snapshots
		WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	item, err := scanSnapshot(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return item, nil
}

func (r *KPISnapshotRepository) LatestByKPI(ctx context.Context, tenantID, kpiID uuid.UUID) (*model.KPISnapshot, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, kpi_id, value, previous_value, delta, delta_percent, status, period_start, period_end,
		       fetch_success, fetch_latency_ms, fetch_error, created_at
		FROM visus_kpi_snapshots
		WHERE tenant_id = $1 AND kpi_id = $2
		ORDER BY created_at DESC
		LIMIT 1`, tenantID, kpiID)
	item, err := scanSnapshot(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return item, nil
}

func (r *KPISnapshotRepository) ListByKPI(ctx context.Context, tenantID, kpiID uuid.UUID, query model.KPIQuery) ([]model.KPISnapshot, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 100
	}
	start := time.Time{}
	end := time.Time{}
	hasStart := query.Start != nil
	hasEnd := query.End != nil
	if hasStart {
		start = *query.Start
	}
	if hasEnd {
		end = *query.End
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, kpi_id, value, previous_value, delta, delta_percent, status, period_start, period_end,
		       fetch_success, fetch_latency_ms, fetch_error, created_at
		FROM visus_kpi_snapshots
		WHERE tenant_id = $1
		  AND kpi_id = $2
		  AND ($3::timestamptz = '0001-01-01 00:00:00+00'::timestamptz OR period_start >= $3)
		  AND ($4::timestamptz = '0001-01-01 00:00:00+00'::timestamptz OR period_end <= $4)
		ORDER BY created_at DESC
		LIMIT $5`,
		tenantID, kpiID, start, end, limit,
	)
	if err != nil {
		return nil, wrapErr("list kpi snapshots", err)
	}
	defer rows.Close()
	out := make([]model.KPISnapshot, 0, limit)
	for rows.Next() {
		item, err := scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (r *KPISnapshotRepository) ListLatestByTenant(ctx context.Context, tenantID uuid.UUID) ([]model.KPISnapshot, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT ON (kpi_id)
		       id, tenant_id, kpi_id, value, previous_value, delta, delta_percent, status, period_start, period_end,
		       fetch_success, fetch_latency_ms, fetch_error, created_at
		FROM visus_kpi_snapshots
		WHERE tenant_id = $1
		ORDER BY kpi_id, created_at DESC`, tenantID)
	if err != nil {
		return nil, wrapErr("list latest kpi snapshots", err)
	}
	defer rows.Close()
	out := make([]model.KPISnapshot, 0)
	for rows.Next() {
		item, err := scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (r *KPISnapshotRepository) ListForPeriod(ctx context.Context, tenantID uuid.UUID, start, end time.Time) ([]model.KPISnapshot, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, kpi_id, value, previous_value, delta, delta_percent, status, period_start, period_end,
		       fetch_success, fetch_latency_ms, fetch_error, created_at
		FROM visus_kpi_snapshots
		WHERE tenant_id = $1
		  AND period_start >= $2
		  AND period_end <= $3
		ORDER BY created_at DESC`, tenantID, start, end)
	if err != nil {
		return nil, wrapErr("list kpi snapshots for period", err)
	}
	defer rows.Close()
	out := make([]model.KPISnapshot, 0)
	for rows.Next() {
		item, err := scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func scanSnapshot(row interface{ Scan(...any) error }) (*model.KPISnapshot, error) {
	item := &model.KPISnapshot{}
	var status string
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.KPIID,
		&item.Value,
		&item.PreviousValue,
		&item.Delta,
		&item.DeltaPercent,
		&status,
		&item.PeriodStart,
		&item.PeriodEnd,
		&item.FetchSuccess,
		&item.FetchLatencyMS,
		&item.FetchError,
		&item.CreatedAt,
	); err != nil {
		return nil, wrapErr("scan kpi snapshot", err)
	}
	item.Status = model.KPIStatus(status)
	return item, nil
}
