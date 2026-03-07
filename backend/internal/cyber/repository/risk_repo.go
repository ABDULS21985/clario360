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

type RiskHistoryRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewRiskHistoryRepository(db *pgxpool.Pool, logger zerolog.Logger) *RiskHistoryRepository {
	return &RiskHistoryRepository{db: db, logger: logger}
}

func (r *RiskHistoryRepository) Upsert(ctx context.Context, tenantID uuid.UUID, snapshotType string, triggerEvent *string, score *model.OrganizationRiskScore) error {
	if score == nil {
		return fmt.Errorf("risk score is required")
	}
	componentsJSON, err := json.Marshal(score.Components)
	if err != nil {
		return fmt.Errorf("marshal risk components: %w", err)
	}
	contributorsJSON, err := json.Marshal(score.TopContributors)
	if err != nil {
		return fmt.Errorf("marshal contributors: %w", err)
	}
	recommendationsJSON, err := json.Marshal(score.Recommendations)
	if err != nil {
		return fmt.Errorf("marshal recommendations: %w", err)
	}

	args := []interface{}{
		uuid.New(),
		tenantID,
		score.OverallScore,
		score.Grade,
		score.Components.VulnerabilityRisk.Score,
		score.Components.ThreatExposure.Score,
		score.Components.ConfigurationRisk.Score,
		score.Components.AttackSurfaceRisk.Score,
		score.Components.ComplianceGapRisk.Score,
		score.Context.TotalAssets,
		score.Context.TotalOpenVulns,
		score.Context.TotalOpenAlerts,
		score.Context.TotalActiveThreats,
		componentsJSON,
		contributorsJSON,
		recommendationsJSON,
		snapshotType,
		triggerEvent,
		score.CalculatedAt,
	}

	if snapshotType == "daily" {
		_, err = r.db.Exec(ctx, `
			INSERT INTO risk_score_history (
				id, tenant_id, overall_score, grade, vulnerability_score, threat_score,
				config_score, surface_score, compliance_score, total_assets, total_open_vulns,
				total_open_alerts, total_active_threats, components, top_contributors,
				recommendations, snapshot_type, trigger_event, calculated_at
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19
			)
			ON CONFLICT (tenant_id, snapshot_type, (calculated_at::date)) WHERE snapshot_type = 'daily'
			DO UPDATE SET
				overall_score = EXCLUDED.overall_score,
				grade = EXCLUDED.grade,
				vulnerability_score = EXCLUDED.vulnerability_score,
				threat_score = EXCLUDED.threat_score,
				config_score = EXCLUDED.config_score,
				surface_score = EXCLUDED.surface_score,
				compliance_score = EXCLUDED.compliance_score,
				total_assets = EXCLUDED.total_assets,
				total_open_vulns = EXCLUDED.total_open_vulns,
				total_open_alerts = EXCLUDED.total_open_alerts,
				total_active_threats = EXCLUDED.total_active_threats,
				components = EXCLUDED.components,
				top_contributors = EXCLUDED.top_contributors,
				recommendations = EXCLUDED.recommendations,
				trigger_event = EXCLUDED.trigger_event,
				calculated_at = EXCLUDED.calculated_at`,
			args...,
		)
		return err
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO risk_score_history (
			id, tenant_id, overall_score, grade, vulnerability_score, threat_score,
			config_score, surface_score, compliance_score, total_assets, total_open_vulns,
			total_open_alerts, total_active_threats, components, top_contributors,
			recommendations, snapshot_type, trigger_event, calculated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19
		)`,
		args...,
	)
	return err
}

func (r *RiskHistoryRepository) Latest(ctx context.Context, tenantID uuid.UUID) (*model.RiskScoreHistory, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, overall_score, grade, vulnerability_score, threat_score,
		       config_score, surface_score, compliance_score, total_assets, total_open_vulns,
		       total_open_alerts, total_active_threats, components, top_contributors,
		       recommendations, snapshot_type, trigger_event, calculated_at
		FROM risk_score_history
		WHERE tenant_id = $1
		ORDER BY calculated_at DESC
		LIMIT 1`,
		tenantID,
	)
	item, err := scanRiskHistory(row)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return item, err
}

func (r *RiskHistoryRepository) LatestDaily(ctx context.Context, tenantID uuid.UUID) (*model.RiskScoreHistory, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, overall_score, grade, vulnerability_score, threat_score,
		       config_score, surface_score, compliance_score, total_assets, total_open_vulns,
		       total_open_alerts, total_active_threats, components, top_contributors,
		       recommendations, snapshot_type, trigger_event, calculated_at
		FROM risk_score_history
		WHERE tenant_id = $1 AND snapshot_type = 'daily'
		ORDER BY calculated_at DESC
		LIMIT 1`,
		tenantID,
	)
	item, err := scanRiskHistory(row)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return item, err
}

func (r *RiskHistoryRepository) Trend(ctx context.Context, tenantID uuid.UUID, days int) ([]model.RiskTrendPoint, error) {
	if days <= 0 {
		days = 90
	}
	since := time.Now().UTC().AddDate(0, 0, -days)
	rows, err := r.db.Query(ctx, `
		SELECT calculated_at, overall_score, grade, vulnerability_score, threat_score,
		       config_score, surface_score, compliance_score
		FROM risk_score_history
		WHERE tenant_id = $1 AND calculated_at >= $2
		ORDER BY calculated_at ASC`,
		tenantID, since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]model.RiskTrendPoint, 0)
	for rows.Next() {
		var point model.RiskTrendPoint
		if err := rows.Scan(
			&point.Time,
			&point.OverallScore,
			&point.Grade,
			&point.VulnerabilityRisk,
			&point.ThreatRisk,
			&point.ConfigRisk,
			&point.SurfaceRisk,
			&point.ComplianceRisk,
		); err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	return points, rows.Err()
}

func scanRiskHistory(row interface {
	Scan(dest ...interface{}) error
}) (*model.RiskScoreHistory, error) {
	var item model.RiskScoreHistory
	err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.OverallScore,
		&item.Grade,
		&item.VulnerabilityScore,
		&item.ThreatScore,
		&item.ConfigScore,
		&item.SurfaceScore,
		&item.ComplianceScore,
		&item.TotalAssets,
		&item.TotalOpenVulns,
		&item.TotalOpenAlerts,
		&item.TotalActiveThreats,
		&item.Components,
		&item.TopContributors,
		&item.Recommendations,
		&item.SnapshotType,
		&item.TriggerEvent,
		&item.CalculatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}
