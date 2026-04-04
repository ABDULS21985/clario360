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

	"github.com/clario360/platform/internal/cyber/dspm/intelligence/dto"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// ComplianceRepository handles persistence for compliance posture records.
type ComplianceRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewComplianceRepository creates a new ComplianceRepository.
func NewComplianceRepository(db *pgxpool.Pool, logger zerolog.Logger) *ComplianceRepository {
	return &ComplianceRepository{db: db, logger: logger}
}

// Upsert inserts or updates a compliance posture snapshot.
func (r *ComplianceRepository) Upsert(ctx context.Context, posture *model.CompliancePosture) error {
	if posture.ID == uuid.Nil {
		posture.ID = uuid.New()
	}
	now := time.Now().UTC()
	posture.UpdatedAt = now
	if posture.EvaluatedAt.IsZero() {
		posture.EvaluatedAt = now
	}

	controlDetailsJSON, err := json.Marshal(posture.ControlDetails)
	if err != nil {
		return fmt.Errorf("marshal control details: %w", err)
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO dspm_compliance_posture (
			id, tenant_id, framework,
			overall_score, controls_total, controls_compliant,
			controls_partial, controls_non_compliant, controls_not_applicable,
			control_details,
			score_7d_ago, score_30d_ago, score_90d_ago,
			trend_direction, estimated_fine_exposure, fine_currency,
			evaluated_at, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$18
		)
		ON CONFLICT (tenant_id, framework)
		DO UPDATE SET
			overall_score = EXCLUDED.overall_score,
			controls_total = EXCLUDED.controls_total,
			controls_compliant = EXCLUDED.controls_compliant,
			controls_partial = EXCLUDED.controls_partial,
			controls_non_compliant = EXCLUDED.controls_non_compliant,
			controls_not_applicable = EXCLUDED.controls_not_applicable,
			control_details = EXCLUDED.control_details,
			score_7d_ago = EXCLUDED.score_7d_ago,
			score_30d_ago = EXCLUDED.score_30d_ago,
			score_90d_ago = EXCLUDED.score_90d_ago,
			trend_direction = EXCLUDED.trend_direction,
			estimated_fine_exposure = EXCLUDED.estimated_fine_exposure,
			fine_currency = EXCLUDED.fine_currency,
			evaluated_at = EXCLUDED.evaluated_at,
			updated_at = EXCLUDED.updated_at`,
		posture.ID, posture.TenantID, posture.Framework,
		posture.OverallScore, posture.ControlsTotal, posture.ControlsCompliant,
		posture.ControlsPartial, posture.ControlsNonCompliant, posture.ControlsNotApplicable,
		controlDetailsJSON,
		posture.Score7dAgo, posture.Score30dAgo, posture.Score90dAgo,
		posture.TrendDirection, posture.EstimatedFineExposure, posture.FineCurrency,
		posture.EvaluatedAt, now,
	)
	if err != nil {
		return fmt.Errorf("upsert compliance posture: %w", err)
	}
	return nil
}

// GetByFramework returns the compliance posture for a specific framework.
func (r *ComplianceRepository) GetByFramework(ctx context.Context, tenantID uuid.UUID, framework string) (*model.CompliancePosture, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, framework,
		       overall_score, controls_total, controls_compliant,
		       controls_partial, controls_non_compliant, controls_not_applicable,
		       control_details,
		       score_7d_ago, score_30d_ago, score_90d_ago,
		       trend_direction, estimated_fine_exposure, fine_currency,
		       evaluated_at, created_at, updated_at
		FROM dspm_compliance_posture
		WHERE tenant_id = $1 AND framework = $2`,
		tenantID, framework,
	)
	cp, err := scanCompliancePosture(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("compliance posture not found")
		}
		return nil, err
	}
	return cp, nil
}

// ListByTenant returns all compliance postures for a tenant.
func (r *ComplianceRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]model.CompliancePosture, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, framework,
		       overall_score, controls_total, controls_compliant,
		       controls_partial, controls_non_compliant, controls_not_applicable,
		       control_details,
		       score_7d_ago, score_30d_ago, score_90d_ago,
		       trend_direction, estimated_fine_exposure, fine_currency,
		       evaluated_at, created_at, updated_at
		FROM dspm_compliance_posture
		WHERE tenant_id = $1
		ORDER BY framework ASC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list compliance postures: %w", err)
	}
	defer rows.Close()

	items := make([]model.CompliancePosture, 0)
	for rows.Next() {
		cp, err := scanCompliancePosture(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *cp)
	}
	return items, rows.Err()
}

// ListGaps returns compliance gaps (non-compliant controls with affected assets).
func (r *ComplianceRepository) ListGaps(ctx context.Context, tenantID uuid.UUID, params *dto.ComplianceGapParams) ([]model.ComplianceGap, int, error) {
	if params == nil {
		params = &dto.ComplianceGapParams{}
	}
	params.SetDefaults()

	// Build conditions for filtering postures
	conds := []string{"tenant_id = $1"}
	args := []interface{}{tenantID}
	i := 2

	if params.Framework != nil && *params.Framework != "" {
		conds = append(conds, fmt.Sprintf("framework = $%d", i))
		args = append(args, *params.Framework)
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	// Load all matching postures with their control_details
	query := fmt.Sprintf(`
		SELECT id, tenant_id, framework,
		       overall_score, controls_total, controls_compliant,
		       controls_partial, controls_non_compliant, controls_not_applicable,
		       control_details,
		       score_7d_ago, score_30d_ago, score_90d_ago,
		       trend_direction, estimated_fine_exposure, fine_currency,
		       evaluated_at, created_at, updated_at
		FROM dspm_compliance_posture
		%s
		ORDER BY framework ASC`, where)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list compliance gaps: %w", err)
	}
	defer rows.Close()

	// Extract gaps from control_details JSONB
	var allGaps []model.ComplianceGap
	for rows.Next() {
		cp, err := scanCompliancePosture(rows)
		if err != nil {
			return nil, 0, err
		}

		for _, ctrl := range cp.ControlDetails {
			if ctrl.Status == model.ControlNonCompliant || ctrl.Status == model.ControlPartial {
				severity := "medium"
				if ctrl.Status == model.ControlNonCompliant {
					severity = "high"
				}

				// Apply severity filter if provided
				if params.Severity != nil && *params.Severity != "" && severity != *params.Severity {
					continue
				}

				gap := model.ComplianceGap{
					Framework:   cp.Framework,
					ControlID:   ctrl.ControlID,
					ControlName: ctrl.Name,
					Severity:    severity,
					AssetCount:  ctrl.AssetsNonCompliant,
					Gaps:        ctrl.Gaps,
				}
				if gap.Gaps == nil {
					gap.Gaps = []model.ControlGap{}
				}
				allGaps = append(allGaps, gap)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	total := len(allGaps)

	// Apply pagination in memory (gaps are extracted from JSONB, not direct SQL rows)
	start := (params.Page - 1) * params.PerPage
	if start >= total {
		return []model.ComplianceGap{}, total, nil
	}
	end := start + params.PerPage
	if end > total {
		end = total
	}

	return allGaps[start:end], total, nil
}

// scanCompliancePosture scans a single compliance posture row.
func scanCompliancePosture(row interface{ Scan(...interface{}) error }) (*model.CompliancePosture, error) {
	var cp model.CompliancePosture
	var controlDetailsJSON []byte

	err := row.Scan(
		&cp.ID, &cp.TenantID, &cp.Framework,
		&cp.OverallScore, &cp.ControlsTotal, &cp.ControlsCompliant,
		&cp.ControlsPartial, &cp.ControlsNonCompliant, &cp.ControlsNotApplicable,
		&controlDetailsJSON,
		&cp.Score7dAgo, &cp.Score30dAgo, &cp.Score90dAgo,
		&cp.TrendDirection, &cp.EstimatedFineExposure, &cp.FineCurrency,
		&cp.EvaluatedAt, &cp.CreatedAt, &cp.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("scan compliance posture: %w", err)
	}

	if len(controlDetailsJSON) > 0 {
		_ = json.Unmarshal(controlDetailsJSON, &cp.ControlDetails)
	}
	if cp.ControlDetails == nil {
		cp.ControlDetails = []model.ControlDetail{}
	}

	return &cp, nil
}
