package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

type CTEMSnapshotRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewCTEMSnapshotRepository(db *pgxpool.Pool, logger zerolog.Logger) *CTEMSnapshotRepository {
	return &CTEMSnapshotRepository{db: db, logger: logger}
}

func (r *CTEMSnapshotRepository) Create(ctx context.Context, tenantID uuid.UUID, assessmentID *uuid.UUID, snapshotType string, score *model.ExposureScore, assetCount, vulnCount, findingCount int) error {
	breakdownJSON, err := json.Marshal(score.Breakdown)
	if err != nil {
		return fmt.Errorf("marshal score breakdown: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO exposure_score_snapshots (
			id, tenant_id, score, breakdown, asset_count, vuln_count, finding_count, assessment_id, snapshot_type, created_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, now()
		)`,
		tenantID, score.Score, breakdownJSON, assetCount, vulnCount, findingCount, assessmentID, snapshotType,
	)
	if err != nil {
		return fmt.Errorf("create exposure snapshot: %w", err)
	}
	return nil
}

func (r *CTEMSnapshotRepository) Last(ctx context.Context, tenantID uuid.UUID) (*model.ExposureScoreSnapshot, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, score, breakdown, asset_count, vuln_count, finding_count, assessment_id, snapshot_type, created_at
		FROM exposure_score_snapshots
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT 1`,
		tenantID,
	)
	item, err := scanExposureScoreSnapshot(row)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return item, err
}

func (r *CTEMSnapshotRepository) History(ctx context.Context, tenantID uuid.UUID, since time.Time) ([]model.TimeSeriesPoint, error) {
	rows, err := r.db.Query(ctx, `
		SELECT created_at, score
		FROM exposure_score_snapshots
		WHERE tenant_id = $1 AND created_at >= $2
		ORDER BY created_at ASC`,
		tenantID, since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]model.TimeSeriesPoint, 0)
	for rows.Next() {
		var point model.TimeSeriesPoint
		if err := rows.Scan(&point.Time, &point.Value); err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	return points, rows.Err()
}

func scanExposureScoreSnapshot(row interface{ Scan(dest ...any) error }) (*model.ExposureScoreSnapshot, error) {
	var item model.ExposureScoreSnapshot
	err := row.Scan(
		&item.ID, &item.TenantID, &item.Score, &item.Breakdown, &item.AssetCount, &item.VulnCount,
		&item.FindingCount, &item.AssessmentID, &item.SnapshotType, &item.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if item.Breakdown == nil {
		item.Breakdown = json.RawMessage("{}")
	}
	return &item, nil
}
