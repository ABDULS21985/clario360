package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/visus/model"
)

type ReportSnapshotRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewReportSnapshotRepository(db *pgxpool.Pool, logger zerolog.Logger) *ReportSnapshotRepository {
	return &ReportSnapshotRepository{db: db, logger: logger.With().Str("repo", "visus_report_snapshots").Logger()}
}

func (r *ReportSnapshotRepository) Create(ctx context.Context, item *model.ReportSnapshot) (*model.ReportSnapshot, error) {
	if item == nil {
		return nil, ErrValidation
	}
	periodStart := dateOnly(&item.PeriodStart)
	periodEnd := dateOnly(&item.PeriodEnd)
	generatedAt := item.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO visus_report_snapshots (
			tenant_id, report_id, report_data, narrative, file_id, file_format, period_start, period_end,
			sections_included, generation_time_ms, suite_fetch_errors, generated_by, generated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (tenant_id, report_id, period_start, period_end)
		DO NOTHING
		RETURNING id`,
		item.TenantID, item.ReportID, marshalJSON(item.ReportData), item.Narrative, item.FileID, item.FileFormat,
		periodStart, periodEnd, item.SectionsIncluded, item.GenerationTimeMS, marshalJSON(item.SuiteFetchErrors), item.GeneratedBy, generatedAt,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return r.FindByPeriod(ctx, item.TenantID, item.ReportID, *periodStart, *periodEnd)
		}
		return nil, wrapErr("create report snapshot", err)
	}
	return r.Get(ctx, item.TenantID, item.ReportID, id)
}

func (r *ReportSnapshotRepository) Get(ctx context.Context, tenantID, reportID, id uuid.UUID) (*model.ReportSnapshot, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, report_id, report_data, narrative, file_id, file_format, period_start, period_end,
		       sections_included, generation_time_ms, suite_fetch_errors, generated_by, generated_at
		FROM visus_report_snapshots
		WHERE tenant_id = $1 AND report_id = $2 AND id = $3`, tenantID, reportID, id)
	item, err := scanReportSnapshot(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return item, nil
}

func (r *ReportSnapshotRepository) ListByReport(ctx context.Context, tenantID, reportID uuid.UUID) ([]model.ReportSnapshot, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, report_id, report_data, narrative, file_id, file_format, period_start, period_end,
		       sections_included, generation_time_ms, suite_fetch_errors, generated_by, generated_at
		FROM visus_report_snapshots
		WHERE tenant_id = $1 AND report_id = $2
		ORDER BY generated_at DESC`, tenantID, reportID)
	if err != nil {
		return nil, wrapErr("list report snapshots", err)
	}
	defer rows.Close()
	out := make([]model.ReportSnapshot, 0)
	for rows.Next() {
		item, err := scanReportSnapshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (r *ReportSnapshotRepository) LatestByReport(ctx context.Context, tenantID, reportID uuid.UUID) (*model.ReportSnapshot, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, report_id, report_data, narrative, file_id, file_format, period_start, period_end,
		       sections_included, generation_time_ms, suite_fetch_errors, generated_by, generated_at
		FROM visus_report_snapshots
		WHERE tenant_id = $1 AND report_id = $2
		ORDER BY generated_at DESC
		LIMIT 1`, tenantID, reportID)
	item, err := scanReportSnapshot(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return item, nil
}

func (r *ReportSnapshotRepository) FindByPeriod(ctx context.Context, tenantID, reportID uuid.UUID, periodStart, periodEnd time.Time) (*model.ReportSnapshot, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, report_id, report_data, narrative, file_id, file_format, period_start, period_end,
		       sections_included, generation_time_ms, suite_fetch_errors, generated_by, generated_at
		FROM visus_report_snapshots
		WHERE tenant_id = $1
		  AND report_id = $2
		  AND period_start = $3
		  AND period_end = $4
		ORDER BY generated_at DESC
		LIMIT 1`, tenantID, reportID, dateOnly(&periodStart), dateOnly(&periodEnd))
	item, err := scanReportSnapshot(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return item, nil
}

func scanReportSnapshot(row interface{ Scan(...any) error }) (*model.ReportSnapshot, error) {
	item := &model.ReportSnapshot{}
	var reportData, fetchErrors []byte
	var fileFormat string
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.ReportID,
		&reportData,
		&item.Narrative,
		&item.FileID,
		&fileFormat,
		&item.PeriodStart,
		&item.PeriodEnd,
		&item.SectionsIncluded,
		&item.GenerationTimeMS,
		&fetchErrors,
		&item.GeneratedBy,
		&item.GeneratedAt,
	); err != nil {
		return nil, wrapErr("scan report snapshot", err)
	}
	item.FileFormat = model.ReportFileFormat(fileFormat)
	item.ReportData = unmarshalMap(reportData)
	item.SuiteFetchErrors = mapStringMap(fetchErrors)
	if item.SectionsIncluded == nil {
		item.SectionsIncluded = []string{}
	}
	return item, nil
}

func mapStringMap(raw []byte) map[string]string {
	if len(raw) == 0 {
		return map[string]string{}
	}
	out := map[string]string{}
	_ = json.Unmarshal(raw, &out)
	return out
}
