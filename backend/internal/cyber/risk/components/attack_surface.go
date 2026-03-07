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

type AttackSurface struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewAttackSurface(db *pgxpool.Pool, logger zerolog.Logger) *AttackSurface {
	return &AttackSurface{db: db, logger: logger.With().Str("risk_component", "attack_surface").Logger()}
}

func (c *AttackSurface) Name() string {
	return "attack_surface_risk"
}

func (c *AttackSurface) Weight() float64 {
	return 0.15
}

func (c *AttackSurface) Calculate(ctx context.Context, tenantID uuid.UUID) (*model.RiskComponentResult, error) {
	var totalActive, internetFacing, internetFacingVulnerable int
	if err := c.db.QueryRow(ctx, `
		SELECT
			COUNT(*)::int,
			COUNT(*) FILTER (
				WHERE 'internet-facing' = ANY(tags) OR 'dmz' = ANY(tags) OR 'public' = ANY(tags)
			)::int,
			COUNT(*) FILTER (
				WHERE ('internet-facing' = ANY(tags) OR 'dmz' = ANY(tags) OR 'public' = ANY(tags))
				  AND EXISTS (
					SELECT 1
					FROM vulnerabilities v
					WHERE v.asset_id = assets.id
					  AND v.status IN ('open', 'in_progress')
					  AND v.severity IN ('critical', 'high')
					  AND v.deleted_at IS NULL
				  )
			)::int
		FROM assets
		WHERE tenant_id = $1 AND status = 'active' AND deleted_at IS NULL`,
		tenantID,
	).Scan(&totalActive, &internetFacing, &internetFacingVulnerable); err != nil {
		return nil, fmt.Errorf("aggregate attack surface risk: %w", err)
	}

	var attackPathCount int
	if err := c.db.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM ctem_findings
		WHERE tenant_id = $1 AND type = 'attack_path' AND status = 'open'`,
		tenantID,
	).Scan(&attackPathCount); err != nil {
		return nil, fmt.Errorf("count attack paths: %w", err)
	}

	internetRatio := float64(internetFacing) / math.Max(float64(totalActive), 1)
	vulnerableRatio := float64(internetFacingVulnerable) / math.Max(float64(internetFacing), 1)
	attackPathFactor := math.Min(float64(attackPathCount*2), 30)
	score := math.Min((internetRatio*40)+(vulnerableRatio*30)+attackPathFactor, 100)

	return &model.RiskComponentResult{
		Score: score,
		Description: fmt.Sprintf(
			"%d internet-facing assets out of %d active assets, with %d exposed assets carrying critical or high vulnerabilities. %d open attack paths remain from CTEM.",
			internetFacing, totalActive, internetFacingVulnerable, attackPathCount,
		),
		Details: map[string]interface{}{
			"total_active":               totalActive,
			"internet_facing":            internetFacing,
			"internet_facing_vulnerable": internetFacingVulnerable,
			"attack_path_count":          attackPathCount,
			"internet_ratio":             internetRatio,
			"vulnerable_internet_ratio":  vulnerableRatio,
			"attack_path_factor":         attackPathFactor,
		},
	}, nil
}
