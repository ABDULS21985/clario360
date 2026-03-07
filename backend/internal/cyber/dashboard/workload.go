package dashboard

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clario360/platform/internal/cyber/model"
)

type WorkloadCalculator struct {
	db *pgxpool.Pool
}

func NewWorkloadCalculator(db *pgxpool.Pool) *WorkloadCalculator {
	return &WorkloadCalculator{db: db}
}

func (c *WorkloadCalculator) Calculate(ctx context.Context, tenantID uuid.UUID) ([]model.AnalystWorkloadEntry, error) {
	rows, err := c.db.Query(ctx, `
		SELECT
			u.id,
			TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')) AS analyst_name,
			COUNT(*) FILTER (WHERE a.status IN ('new','acknowledged','investigating'))::int,
			COUNT(*) FILTER (WHERE a.status = 'resolved' AND a.resolved_at > now() - interval '7 days')::int,
			AVG(EXTRACT(EPOCH FROM (a.resolved_at - a.created_at)) / 3600)
				FILTER (WHERE a.resolved_at IS NOT NULL AND a.resolved_at > now() - interval '30 days')::float8,
			COUNT(*) FILTER (WHERE a.severity = 'critical' AND a.status IN ('new','acknowledged'))::int
		FROM alerts a
		JOIN users u ON u.id = a.assigned_to
		WHERE a.tenant_id = $1
		  AND a.assigned_to IS NOT NULL
		  AND a.deleted_at IS NULL
		GROUP BY u.id, u.first_name, u.last_name
		ORDER BY COUNT(*) FILTER (WHERE a.status IN ('new','acknowledged','investigating')) DESC, analyst_name ASC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("analyst workload: %w", err)
	}
	defer rows.Close()

	items := make([]model.AnalystWorkloadEntry, 0)
	for rows.Next() {
		var item model.AnalystWorkloadEntry
		if err := rows.Scan(
			&item.UserID,
			&item.Name,
			&item.OpenAssigned,
			&item.ResolvedThisWeek,
			&item.AvgResolveHours,
			&item.CriticalOpen,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
