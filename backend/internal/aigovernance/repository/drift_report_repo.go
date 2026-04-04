package repository

import (
	"context"
	"fmt"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type DriftReportRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewDriftReportRepository(db *pgxpool.Pool, logger zerolog.Logger) *DriftReportRepository {
	return &DriftReportRepository{db: db, logger: loggerWithRepo(logger, "ai_drift_report")}
}

func (r *DriftReportRepository) Create(ctx context.Context, item *aigovmodel.DriftReport) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO ai_drift_reports (
			id, tenant_id, model_id, model_version_id, period, period_start, period_end, output_psi,
			output_drift_level, confidence_psi, confidence_drift_level, current_volume, reference_volume,
			volume_change_pct, current_p95_latency_ms, reference_p95_latency_ms, latency_change_pct,
			current_accuracy, reference_accuracy, accuracy_change, alerts, alert_count, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
		)`,
		item.ID, item.TenantID, item.ModelID, item.ModelVersionID, item.Period, item.PeriodStart, item.PeriodEnd,
		item.OutputPSI, item.OutputDriftLevel, item.ConfidencePSI, item.ConfidenceDriftLevel, item.CurrentVolume,
		item.ReferenceVolume, item.VolumeChangePct, item.CurrentP95LatencyMS, item.ReferenceP95LatencyMS,
		item.LatencyChangePct, item.CurrentAccuracy, item.ReferenceAccuracy, item.AccuracyChange,
		item.Alerts, item.AlertCount, item.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert ai drift report: %w", err)
	}
	return nil
}

func (r *DriftReportRepository) LatestByModel(ctx context.Context, tenantID, modelID uuid.UUID) (*aigovmodel.DriftReport, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, model_id, model_version_id, period, period_start, period_end, output_psi,
		       output_drift_level, confidence_psi, confidence_drift_level, current_volume, reference_volume,
		       volume_change_pct, current_p95_latency_ms, reference_p95_latency_ms, latency_change_pct,
		       current_accuracy, reference_accuracy, accuracy_change, alerts, alert_count, created_at
		FROM ai_drift_reports
		WHERE tenant_id = $1 AND model_id = $2
		ORDER BY created_at DESC
		LIMIT 1`,
		tenantID, modelID,
	)
	item, err := scanDriftReport(row)
	if err != nil {
		return nil, rowNotFound(err)
	}
	return item, nil
}

func (r *DriftReportRepository) History(ctx context.Context, tenantID, modelID uuid.UUID, limit int) ([]aigovmodel.DriftReport, error) {
	if limit <= 0 {
		limit = 30
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, model_id, model_version_id, period, period_start, period_end, output_psi,
		       output_drift_level, confidence_psi, confidence_drift_level, current_volume, reference_volume,
		       volume_change_pct, current_p95_latency_ms, reference_p95_latency_ms, latency_change_pct,
		       current_accuracy, reference_accuracy, accuracy_change, alerts, alert_count, created_at
		FROM ai_drift_reports
		WHERE tenant_id = $1 AND model_id = $2
		ORDER BY created_at DESC
		LIMIT $3`,
		tenantID, modelID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list ai drift history: %w", err)
	}
	defer rows.Close()
	items := make([]aigovmodel.DriftReport, 0)
	for rows.Next() {
		item, err := scanDriftReport(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

type driftScannable interface {
	Scan(dest ...any) error
}

func scanDriftReport(row driftScannable) (*aigovmodel.DriftReport, error) {
	item := &aigovmodel.DriftReport{}
	var alerts []byte
	if err := row.Scan(
		&item.ID, &item.TenantID, &item.ModelID, &item.ModelVersionID, &item.Period, &item.PeriodStart,
		&item.PeriodEnd, &item.OutputPSI, &item.OutputDriftLevel, &item.ConfidencePSI,
		&item.ConfidenceDriftLevel, &item.CurrentVolume, &item.ReferenceVolume, &item.VolumeChangePct,
		&item.CurrentP95LatencyMS, &item.ReferenceP95LatencyMS, &item.LatencyChangePct, &item.CurrentAccuracy,
		&item.ReferenceAccuracy, &item.AccuracyChange, &alerts, &item.AlertCount, &item.CreatedAt,
	); err != nil {
		return nil, err
	}
	item.Alerts = nullJSON(alerts, "[]")
	return item, nil
}
