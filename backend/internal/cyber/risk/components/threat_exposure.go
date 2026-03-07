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

type ThreatExposure struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewThreatExposure(db *pgxpool.Pool, logger zerolog.Logger) *ThreatExposure {
	return &ThreatExposure{db: db, logger: logger.With().Str("risk_component", "threat").Logger()}
}

func (c *ThreatExposure) Name() string {
	return "threat_exposure"
}

func (c *ThreatExposure) Weight() float64 {
	return 0.25
}

func (c *ThreatExposure) Calculate(ctx context.Context, tenantID uuid.UUID) (*model.RiskComponentResult, error) {
	var totalOpen, criticalOpen, highOpen, mediumOpen, lowOpen int
	var criticalSLABreached, highSLABreached, uniqueTactics int
	if err := c.db.QueryRow(ctx, `
		SELECT
			COUNT(*)::int,
			COUNT(*) FILTER (WHERE severity = 'critical')::int,
			COUNT(*) FILTER (WHERE severity = 'high')::int,
			COUNT(*) FILTER (WHERE severity = 'medium')::int,
			COUNT(*) FILTER (WHERE severity = 'low')::int,
			COUNT(*) FILTER (WHERE severity = 'critical' AND status = 'new' AND created_at < now() - interval '4 hours')::int,
			COUNT(*) FILTER (WHERE severity = 'high' AND status = 'new' AND created_at < now() - interval '8 hours')::int,
			COUNT(DISTINCT mitre_tactic_id) FILTER (
				WHERE mitre_tactic_id IS NOT NULL AND created_at > now() - interval '30 days'
			)::int
		FROM alerts
		WHERE tenant_id = $1 AND status IN ('new', 'acknowledged', 'investigating') AND deleted_at IS NULL`,
		tenantID,
	).Scan(
		&totalOpen,
		&criticalOpen,
		&highOpen,
		&mediumOpen,
		&lowOpen,
		&criticalSLABreached,
		&highSLABreached,
		&uniqueTactics,
	); err != nil {
		return nil, fmt.Errorf("aggregate threat exposure: %w", err)
	}

	var activeThreats, totalAssets int
	if err := c.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*)::int FROM threats WHERE tenant_id = $1 AND status = 'active' AND deleted_at IS NULL),
			(SELECT COUNT(*)::int FROM assets WHERE tenant_id = $1 AND status = 'active' AND deleted_at IS NULL)`,
		tenantID,
	).Scan(&activeThreats, &totalAssets); err != nil {
		return nil, fmt.Errorf("load active threats context: %w", err)
	}

	base := float64((criticalOpen*10)+(highOpen*7)+(mediumOpen*3)+lowOpen) / math.Max(float64(totalAssets), 1) * 10
	slaPenalty := float64((criticalSLABreached * 8) + (highSLABreached * 4))
	tacticBreadth := float64(uniqueTactics * 3)
	threatPenalty := math.Min(float64(activeThreats*5), 20)
	score := math.Min(base+slaPenalty+tacticBreadth+threatPenalty, 100)

	return &model.RiskComponentResult{
		Score: score,
		Description: fmt.Sprintf(
			"%d unresolved alerts (%d critical, %d high). %d alerts breached SLA. %d distinct MITRE ATT&CK tactics observed in last 30 days.",
			totalOpen, criticalOpen, highOpen, criticalSLABreached+highSLABreached, uniqueTactics,
		),
		Details: map[string]interface{}{
			"total_open":             totalOpen,
			"critical_open":          criticalOpen,
			"high_open":              highOpen,
			"medium_open":            mediumOpen,
			"low_open":               lowOpen,
			"critical_sla_breached":  criticalSLABreached,
			"high_sla_breached":      highSLABreached,
			"unique_tactics":         uniqueTactics,
			"active_threats":         activeThreats,
			"total_assets":           totalAssets,
			"sla_penalty":            slaPenalty,
			"tactic_breadth_penalty": tacticBreadth,
		},
	}, nil
}
