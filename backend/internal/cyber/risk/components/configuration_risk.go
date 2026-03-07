package components

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

type ConfigurationRisk struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewConfigurationRisk(db *pgxpool.Pool, logger zerolog.Logger) *ConfigurationRisk {
	return &ConfigurationRisk{db: db, logger: logger.With().Str("risk_component", "configuration").Logger()}
}

func (c *ConfigurationRisk) Name() string {
	return "configuration_risk"
}

func (c *ConfigurationRisk) Weight() float64 {
	return 0.15
}

func (c *ConfigurationRisk) Calculate(ctx context.Context, tenantID uuid.UUID) (*model.RiskComponentResult, error) {
	var total, criticalCount, highCount, mediumCount, lowCount, affectedAssets int
	if err := c.db.QueryRow(ctx, `
		SELECT
			COUNT(*)::int,
			COUNT(*) FILTER (WHERE severity = 'critical')::int,
			COUNT(*) FILTER (WHERE severity = 'high')::int,
			COUNT(*) FILTER (WHERE severity = 'medium')::int,
			COUNT(*) FILTER (WHERE severity = 'low')::int,
			COUNT(DISTINCT primary_asset_id)::int
		FROM ctem_findings
		WHERE tenant_id = $1
		  AND type IN ('misconfiguration', 'insecure_protocol', 'weak_credential', 'expired_certificate')
		  AND status = 'open'`,
		tenantID,
	).Scan(&total, &criticalCount, &highCount, &mediumCount, &lowCount, &affectedAssets); err != nil {
		return nil, fmt.Errorf("aggregate configuration risk: %w", err)
	}

	var totalAssets, totalAssessments int
	if err := c.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*)::int FROM assets WHERE tenant_id = $1 AND status = 'active' AND deleted_at IS NULL),
			(SELECT COUNT(*)::int FROM ctem_assessments WHERE tenant_id = $1 AND deleted_at IS NULL)`,
		tenantID,
	).Scan(&totalAssets, &totalAssessments); err != nil {
		return nil, fmt.Errorf("load configuration risk context: %w", err)
	}

	if totalAssessments == 0 && total == 0 {
		return &model.RiskComponentResult{
			Score: 50,
			Description: "No CTEM assessment has been completed yet. Configuration risk is set to a moderate default until assessment data exists.",
			Details: map[string]interface{}{
				"total_misconfigs": total,
				"affected_assets":  affectedAssets,
				"assessments":      totalAssessments,
			},
		}, nil
	}

	score := math.Min(
		float64((criticalCount*10)+(highCount*7)+(mediumCount*3)+lowCount)/math.Max(float64(totalAssets), 1)*20,
		100,
	)
	return &model.RiskComponentResult{
		Score: score,
		Description: fmt.Sprintf("%d open misconfigurations across %d assets based on CTEM findings.", total, affectedAssets),
		Details: map[string]interface{}{
			"total_misconfigs": total,
			"critical_count":   criticalCount,
			"high_count":       highCount,
			"medium_count":     mediumCount,
			"low_count":        lowCount,
			"affected_assets":  affectedAssets,
			"assessments":      totalAssessments,
		},
	}, nil
}
