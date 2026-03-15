package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/database"
)

// ThreatRepository handles threat persistence.
type ThreatRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewThreatRepository creates a new ThreatRepository.
func NewThreatRepository(db *pgxpool.Pool, logger zerolog.Logger) *ThreatRepository {
	return &ThreatRepository{db: db, logger: logger}
}

// List returns paginated threats for a tenant.
func (r *ThreatRepository) List(ctx context.Context, tenantID uuid.UUID, params *dto.ThreatListParams) ([]*model.Threat, int, error) {
	baseSelect := `
		SELECT
			a.id, a.tenant_id, a.name, a.description, a.type, a.severity, a.status,
			a.threat_actor, a.campaign, a.mitre_tactic_ids, a.mitre_technique_ids,
			COALESCE((
				SELECT COUNT(*)
				FROM threat_indicators ti
				WHERE ti.tenant_id = a.tenant_id
				  AND ti.threat_id = a.id
			), 0) AS indicator_count,
			a.affected_asset_count, a.alert_count, a.first_seen_at, a.last_seen_at,
			a.contained_at, a.tags, a.metadata, a.created_by, a.created_at, a.updated_at, a.deleted_at
		FROM threats a`
	qb := database.NewQueryBuilder(baseSelect)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.Where("a.deleted_at IS NULL")
	if params.Search != nil && strings.TrimSpace(*params.Search) != "" {
		search := "%" + strings.TrimSpace(*params.Search) + "%"
		qb.Where("(a.name ILIKE ? OR a.description ILIKE ?)", search, search)
	}
	if len(params.Types) > 0 {
		qb.WhereIn("a.type", params.Types)
	}
	if len(params.Statuses) > 0 {
		qb.WhereIn("a.status", params.Statuses)
	}
	if len(params.Severities) > 0 {
		qb.WhereIn("a.severity", params.Severities)
	}
	qb.OrderBy(params.Sort, params.Order, []string{"last_seen_at", "created_at", "severity", "status", "affected_asset_count", "alert_count", "name"})
	qb.Paginate(params.Page, params.PerPage)

	var total int
	items := make([]*model.Threat, 0)
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		countSQL, countArgs := qb.BuildCount()
		if err := db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
			return fmt.Errorf("count threats: %w", err)
		}

		sql, args := qb.Build()
		rows, err := db.Query(ctx, sql, args...)
		if err != nil {
			return fmt.Errorf("list threats: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			threat, err := scanThreat(rows)
			if err != nil {
				return err
			}
			items = append(items, threat)
		}
		return rows.Err()
	})
	return items, total, err
}

// GetByID fetches a single threat.
func (r *ThreatRepository) GetByID(ctx context.Context, tenantID, threatID uuid.UUID) (*model.Threat, error) {
	var item *model.Threat
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		row := db.QueryRow(ctx, `
			SELECT
				id, tenant_id, name, description, type, severity, status,
				threat_actor, campaign, mitre_tactic_ids, mitre_technique_ids,
				COALESCE((
					SELECT COUNT(*)
					FROM threat_indicators ti
					WHERE ti.tenant_id = threats.tenant_id
					  AND ti.threat_id = threats.id
				), 0) AS indicator_count,
				affected_asset_count, alert_count, first_seen_at, last_seen_at,
				contained_at, tags, metadata, created_by, created_at, updated_at, deleted_at
			FROM threats
			WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
			tenantID, threatID,
		)
		threat, err := scanThreat(row)
		if err != nil {
			if err == pgx.ErrNoRows {
				return ErrNotFound
			}
			return fmt.Errorf("get threat: %w", err)
		}
		item = threat
		return nil
	})
	return item, err
}

// Update edits a threat record without changing its lifecycle status.
func (r *ThreatRepository) Update(ctx context.Context, threat *model.Threat) (*model.Threat, error) {
	if err := runWithTenantWrite(ctx, r.db, threat.TenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `
			UPDATE threats
			SET
				name = $3,
				description = $4,
				type = $5,
				severity = $6,
				threat_actor = $7,
				campaign = $8,
				mitre_tactic_ids = $9,
				mitre_technique_ids = $10,
				tags = $11,
				metadata = $12,
				updated_at = now()
			WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
			threat.TenantID,
			threat.ID,
			threat.Name,
			threat.Description,
			threat.Type,
			threat.Severity,
			threat.ThreatActor,
			threat.Campaign,
			threat.MITRETacticIDs,
			threat.MITRETechniqueIDs,
			threat.Tags,
			ensureRawMessage(threat.Metadata, "{}"),
		)
		if err != nil {
			return fmt.Errorf("update threat: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, threat.TenantID, threat.ID)
}

// Create inserts a threat record.
func (r *ThreatRepository) Create(ctx context.Context, threat *model.Threat) (*model.Threat, error) {
	if threat.ID == uuid.Nil {
		threat.ID = uuid.New()
	}
	if threat.MITRETacticIDs == nil {
		threat.MITRETacticIDs = []string{}
	}
	if threat.MITRETechniqueIDs == nil {
		threat.MITRETechniqueIDs = []string{}
	}
	if threat.Tags == nil {
		threat.Tags = []string{}
	}
	now := time.Now().UTC()
	if threat.FirstSeenAt.IsZero() {
		threat.FirstSeenAt = now
	}
	if threat.LastSeenAt.IsZero() {
		threat.LastSeenAt = now
	}
	if err := runWithTenantWrite(ctx, r.db, threat.TenantID, func(db dbtx) error {
		_, err := db.Exec(ctx, `
			INSERT INTO threats (
				id, tenant_id, name, description, type, severity, status,
				threat_actor, campaign, mitre_tactic_ids, mitre_technique_ids,
				affected_asset_count, alert_count, first_seen_at, last_seen_at,
				contained_at, tags, metadata, created_by, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7,
				$8, $9, $10, $11,
				$12, $13, $14, $15,
				$16, $17, $18, $19, now(), now()
			)`,
			threat.ID, threat.TenantID, threat.Name, threat.Description, threat.Type, threat.Severity, threat.Status,
			threat.ThreatActor, threat.Campaign, threat.MITRETacticIDs, threat.MITRETechniqueIDs,
			threat.AffectedAssetCount, threat.AlertCount, threat.FirstSeenAt, threat.LastSeenAt,
			threat.ContainedAt, threat.Tags, ensureRawMessage(threat.Metadata, "{}"), threat.CreatedBy,
		)
		if err != nil {
			return fmt.Errorf("insert threat: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, threat.TenantID, threat.ID)
}

// UpdateStatus updates a threat status.
func (r *ThreatRepository) UpdateStatus(ctx context.Context, tenantID, threatID uuid.UUID, status model.ThreatStatus) (*model.Threat, error) {
	if err := runWithTenantWrite(ctx, r.db, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `
			UPDATE threats
			SET
				status = $3,
				contained_at = CASE
					WHEN $3 = 'contained' AND contained_at IS NULL THEN now()
					WHEN $3 <> 'contained' AND status = 'contained' THEN contained_at
					ELSE contained_at
				END,
				updated_at = now()
			WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
			tenantID, threatID, status,
		)
		if err != nil {
			return fmt.Errorf("update threat status: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, tenantID, threatID)
}

// Delete soft-deletes a threat record.
func (r *ThreatRepository) Delete(ctx context.Context, tenantID, threatID uuid.UUID) error {
	return runWithTenantWrite(ctx, r.db, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `
			UPDATE threats
			SET deleted_at = now(), updated_at = now()
			WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
			tenantID, threatID,
		)
		if err != nil {
			return fmt.Errorf("delete threat: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// UpsertSyntheticThreat ensures there is a threat record associated with an indicator-driven detection.
func (r *ThreatRepository) UpsertSyntheticThreat(ctx context.Context, tenantID uuid.UUID, name, description string, threatType model.ThreatType, severity model.Severity, tags []string) (*model.Threat, error) {
	var item *model.Threat
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		row := db.QueryRow(ctx, `
			SELECT
				id, tenant_id, name, description, type, severity, status,
				threat_actor, campaign, mitre_tactic_ids, mitre_technique_ids,
				COALESCE((
					SELECT COUNT(*)
					FROM threat_indicators ti
					WHERE ti.tenant_id = threats.tenant_id
					  AND ti.threat_id = threats.id
				), 0) AS indicator_count,
				affected_asset_count, alert_count, first_seen_at, last_seen_at,
				contained_at, tags, metadata, created_by, created_at, updated_at, deleted_at
			FROM threats
			WHERE tenant_id = $1 AND name = $2 AND deleted_at IS NULL`,
			tenantID, name,
		)
		threat, err := scanThreat(row)
		if err != nil {
			if err == pgx.ErrNoRows {
				return nil
			}
			return fmt.Errorf("lookup synthetic threat: %w", err)
		}
		item = threat
		return nil
	})
	if err != nil {
		return nil, err
	}
	if item != nil {
		return item, nil
	}
	return r.Create(ctx, &model.Threat{
		TenantID:          tenantID,
		Name:              name,
		Description:       description,
		Type:              threatType,
		Severity:          severity,
		Status:            model.ThreatStatusActive,
		Tags:              tags,
		MITRETacticIDs:    []string{},
		MITRETechniqueIDs: []string{},
	})
}

// RecordObservation updates threat counters based on a new detection.
func (r *ThreatRepository) RecordObservation(ctx context.Context, tenantID, threatID uuid.UUID, assetIDs []uuid.UUID) error {
	return runWithTenantWrite(ctx, r.db, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `
			UPDATE threats
			SET
				alert_count = alert_count + 1,
				affected_asset_count = GREATEST(affected_asset_count, $3),
				last_seen_at = now(),
				updated_at = now()
			WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
			tenantID, threatID, len(assetIDs),
		)
		if err != nil {
			return fmt.Errorf("record threat observation: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// Stats returns aggregated threat counts.
func (r *ThreatRepository) Stats(ctx context.Context, tenantID uuid.UUID) (*model.ThreatStats, error) {
	stats := &model.ThreatStats{}
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		if err := db.QueryRow(ctx, `
			SELECT
				COUNT(*) FILTER (WHERE deleted_at IS NULL),
				COUNT(*) FILTER (WHERE deleted_at IS NULL AND status = 'active'),
				COUNT(*) FILTER (
					WHERE deleted_at IS NULL
					  AND contained_at >= date_trunc('month', now())
				)
			FROM threats
			WHERE tenant_id = $1`,
			tenantID,
		).Scan(&stats.Total, &stats.Active, &stats.ContainedThisMonth); err != nil {
			return fmt.Errorf("threat totals: %w", err)
		}
		if err := db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM threat_indicators
			WHERE tenant_id = $1`,
			tenantID,
		).Scan(&stats.IndicatorsTotal); err != nil {
			return fmt.Errorf("threat indicator totals: %w", err)
		}

		queryCounts := func(column string) ([]model.NamedCount, error) {
			rows, err := db.Query(ctx, fmt.Sprintf(`
				SELECT %s::text, COUNT(*)
				FROM threats
				WHERE tenant_id = $1 AND deleted_at IS NULL
				GROUP BY %s
				ORDER BY COUNT(*) DESC, %s ASC`, column, column, column), tenantID)
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			out := make([]model.NamedCount, 0)
			for rows.Next() {
				var item model.NamedCount
				if err := rows.Scan(&item.Name, &item.Count); err != nil {
					return nil, err
				}
				out = append(out, item)
			}
			return out, rows.Err()
		}

		var err error
		if stats.ByType, err = queryCounts("type"); err != nil {
			return fmt.Errorf("threat stats by type: %w", err)
		}
		if stats.ByStatus, err = queryCounts("status"); err != nil {
			return fmt.Errorf("threat stats by status: %w", err)
		}
		if stats.BySeverity, err = queryCounts("severity"); err != nil {
			return fmt.Errorf("threat stats by severity: %w", err)
		}
		return nil
	})
	return stats, err
}

// Trend returns daily threat metrics for dashboard trend views.
func (r *ThreatRepository) Trend(ctx context.Context, tenantID uuid.UUID, days int) ([]dto.ThreatTrendPoint, error) {
	if days <= 0 {
		days = 30
	}
	start := time.Now().UTC().AddDate(0, 0, -(days - 1)).Truncate(24 * time.Hour)
	series := make([]dto.ThreatTrendPoint, 0, days)
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, `
			WITH buckets AS (
				SELECT generate_series($2::timestamptz, date_trunc('day', now()), interval '1 day') AS bucket
			)
			SELECT
				b.bucket,
				COALESCE(created.total_count, 0) AS total_count,
				COALESCE(active.active_count, 0) AS active_count,
				COALESCE(contained.contained_count, 0) AS contained_count
			FROM buckets b
			LEFT JOIN (
				SELECT date_trunc('day', created_at) AS bucket, COUNT(*)::int AS total_count
				FROM threats
				WHERE tenant_id = $1
				  AND deleted_at IS NULL
				  AND created_at >= $2
				GROUP BY 1
			) created ON created.bucket = b.bucket
			LEFT JOIN LATERAL (
				SELECT COUNT(*)::int AS active_count
				FROM threats
				WHERE tenant_id = $1
				  AND deleted_at IS NULL
				  AND first_seen_at < (b.bucket + interval '1 day')
				  AND (contained_at IS NULL OR contained_at >= b.bucket)
			) active ON true
			LEFT JOIN (
				SELECT date_trunc('day', contained_at) AS bucket, COUNT(*)::int AS contained_count
				FROM threats
				WHERE tenant_id = $1
				  AND deleted_at IS NULL
				  AND contained_at IS NOT NULL
				  AND contained_at >= $2
				GROUP BY 1
			) contained ON contained.bucket = b.bucket
			ORDER BY b.bucket ASC`,
			tenantID, start,
		)
		if err != nil {
			return fmt.Errorf("threat trend: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var point dto.ThreatTrendPoint
			if err := rows.Scan(&point.Date, &point.Total, &point.Active, &point.Contained); err != nil {
				return err
			}
			series = append(series, point)
		}
		return rows.Err()
	})
	return series, err
}
