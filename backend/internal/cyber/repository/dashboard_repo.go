package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

type DashboardRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewDashboardRepository(db *pgxpool.Pool, logger zerolog.Logger) *DashboardRepository {
	return &DashboardRepository{db: db, logger: logger}
}

func (r *DashboardRepository) RecentAlerts(ctx context.Context, tenantID uuid.UUID, limit int) ([]model.AlertSummary, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.Query(ctx, `
		SELECT a.id, a.title, a.severity::text, a.status::text, a.asset_id, asset.name, a.assigned_to,
		       a.created_at, a.mitre_technique_id, a.mitre_technique_name
		FROM alerts a
		LEFT JOIN assets asset ON asset.id = a.asset_id
		WHERE a.tenant_id = $1 AND a.deleted_at IS NULL
		ORDER BY a.created_at DESC
		LIMIT $2`,
		tenantID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("recent alerts: %w", err)
	}
	defer rows.Close()

	items := make([]model.AlertSummary, 0, limit)
	for rows.Next() {
		var item model.AlertSummary
		if err := rows.Scan(
			&item.ID,
			&item.Title,
			&item.Severity,
			&item.Status,
			&item.AssetID,
			&item.AssetName,
			&item.AssignedTo,
			&item.CreatedAt,
			&item.MITRETechniqueID,
			&item.MITRETechniqueName,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ActiveUsersToday counts distinct users who acted on alerts today
// (assigned_to or escalated_to).
func (r *DashboardRepository) ActiveUsersToday(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(DISTINCT u)::int
		FROM alerts a, LATERAL (
			VALUES (a.assigned_to), (a.escalated_to)
		) AS t(u)
		WHERE a.tenant_id = $1
		  AND a.deleted_at IS NULL
		  AND a.updated_at >= CURRENT_DATE
		  AND u IS NOT NULL`,
		tenantID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("active users today: %w", err)
	}
	return count, nil
}

// PendingReviews counts open alerts that await acknowledgement or investigation.
func (r *DashboardRepository) PendingReviews(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM alerts
		WHERE tenant_id = $1
		  AND deleted_at IS NULL
		  AND status = 'new'
		  AND severity IN ('critical', 'high')`,
		tenantID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("pending reviews: %w", err)
	}
	return count, nil
}

// ActiveIncidents counts open alerts with critical or high severity.
func (r *DashboardRepository) ActiveIncidents(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM alerts
		WHERE tenant_id = $1
		  AND deleted_at IS NULL
		  AND status IN ('new', 'acknowledged', 'investigating', 'in_progress', 'escalated')
		  AND severity IN ('critical', 'high')`,
		tenantID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("active incidents: %w", err)
	}
	return count, nil
}

func (r *DashboardRepository) TopAttackedAssets(ctx context.Context, tenantID uuid.UUID, limit int) ([]model.AssetAlertSummary, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.Query(ctx, `
		SELECT asset.id, asset.name, asset.type::text, asset.criticality::text,
		       COUNT(*)::int AS alert_count,
		       COUNT(*) FILTER (WHERE a.severity = 'critical' AND a.status IN ('new','acknowledged'))::int AS critical_open
		FROM alerts a
		JOIN assets asset ON asset.id = a.asset_id AND asset.deleted_at IS NULL
		WHERE a.tenant_id = $1
		  AND a.status IN ('new','acknowledged','investigating')
		  AND a.deleted_at IS NULL
		  AND a.asset_id IS NOT NULL
		GROUP BY asset.id, asset.name, asset.type, asset.criticality
		ORDER BY alert_count DESC, critical_open DESC, asset.name ASC
		LIMIT $2`,
		tenantID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("top attacked assets: %w", err)
	}
	defer rows.Close()

	items := make([]model.AssetAlertSummary, 0, limit)
	for rows.Next() {
		var item model.AssetAlertSummary
		if err := rows.Scan(
			&item.AssetID,
			&item.AssetName,
			&item.AssetType,
			&item.Criticality,
			&item.AlertCount,
			&item.CriticalOpen,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
