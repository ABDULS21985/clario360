package dashboard

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clario360/platform/internal/cyber/model"
)

type KPICalculator struct {
	db *pgxpool.Pool
}

func NewKPICalculator(db *pgxpool.Pool) *KPICalculator {
	return &KPICalculator{db: db}
}

func (c *KPICalculator) Calculate(ctx context.Context, tenantID uuid.UUID) (model.KPICards, error) {
	var out model.KPICards
	if err := c.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*)::int FROM alerts WHERE tenant_id = $1 AND status IN ('new','acknowledged','investigating') AND deleted_at IS NULL),
			(SELECT COUNT(*)::int FROM alerts WHERE tenant_id = $1 AND status IN ('new','acknowledged','investigating') AND severity = 'critical' AND deleted_at IS NULL),
			(SELECT COUNT(*)::int FROM vulnerabilities WHERE tenant_id = $1 AND status IN ('open','in_progress') AND deleted_at IS NULL),
			(SELECT COUNT(*)::int FROM vulnerabilities WHERE tenant_id = $1 AND status IN ('open','in_progress') AND severity = 'critical' AND deleted_at IS NULL),
			(SELECT COUNT(*)::int FROM threats WHERE tenant_id = $1 AND status = 'active' AND deleted_at IS NULL)`,
		tenantID,
	).Scan(
		&out.OpenAlerts,
		&out.CriticalAlerts,
		&out.OpenVulnerabilities,
		&out.CriticalVulnerabilities,
		&out.ActiveThreats,
	); err != nil {
		return out, fmt.Errorf("query kpis: %w", err)
	}

	var alertsToday, alertsYesterday, vulnsToday, vulnsYesterday int
	if err := c.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE created_at > now() - interval '1 day')::int,
			COUNT(*) FILTER (WHERE created_at BETWEEN now() - interval '2 days' AND now() - interval '1 day')::int
		FROM alerts
		WHERE tenant_id = $1 AND deleted_at IS NULL`,
		tenantID,
	).Scan(&alertsToday, &alertsYesterday); err != nil {
		return out, fmt.Errorf("query alert delta: %w", err)
	}
	if err := c.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE discovered_at > now() - interval '1 day')::int,
			COUNT(*) FILTER (WHERE discovered_at BETWEEN now() - interval '2 days' AND now() - interval '1 day')::int
		FROM vulnerabilities
		WHERE tenant_id = $1 AND deleted_at IS NULL`,
		tenantID,
	).Scan(&vulnsToday, &vulnsYesterday); err != nil {
		return out, fmt.Errorf("query vulnerability delta: %w", err)
	}
	out.AlertsDelta = alertsToday - alertsYesterday
	out.VulnsDelta = vulnsToday - vulnsYesterday
	return out, nil
}
