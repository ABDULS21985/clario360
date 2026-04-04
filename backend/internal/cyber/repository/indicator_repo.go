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

// IndicatorRepository handles threat indicator persistence.
type IndicatorRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

const indicatorSelectWithThreat = `
	SELECT
		a.id, a.tenant_id, a.threat_id, t.name AS threat_name, t.type AS threat_type, t.status AS threat_status,
		a.type, a.value, a.description, a.severity, a.source,
		a.confidence, a.active, a.first_seen_at, a.last_seen_at, a.expires_at,
		a.tags, a.metadata, a.created_by, a.created_at, a.updated_at
	FROM threat_indicators a
	LEFT JOIN threats t ON t.id = a.threat_id AND t.tenant_id = a.tenant_id AND t.deleted_at IS NULL`

// NewIndicatorRepository creates a new IndicatorRepository.
func NewIndicatorRepository(db *pgxpool.Pool, logger zerolog.Logger) *IndicatorRepository {
	return &IndicatorRepository{db: db, logger: logger}
}

// Create inserts or updates an indicator.
func (r *IndicatorRepository) Create(ctx context.Context, indicator *model.ThreatIndicator) (*model.ThreatIndicator, error) {
	if indicator.ID == uuid.Nil {
		indicator.ID = uuid.New()
	}
	now := time.Now().UTC()
	if err := runWithTenantWrite(ctx, r.db, indicator.TenantID, func(db dbtx) error {
		_, err := db.Exec(ctx, `
			INSERT INTO threat_indicators (
				id, tenant_id, threat_id, type, value, description, severity, source,
				confidence, active, first_seen_at, last_seen_at, expires_at,
				tags, metadata, created_by, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8,
				$9, $10, $11, $12, $13,
				$14, $15, $16, $17, $17
			)
			ON CONFLICT (tenant_id, type, value)
			DO UPDATE SET
				threat_id = COALESCE(EXCLUDED.threat_id, threat_indicators.threat_id),
				description = EXCLUDED.description,
				severity = EXCLUDED.severity,
				source = EXCLUDED.source,
				confidence = EXCLUDED.confidence,
				active = EXCLUDED.active,
				last_seen_at = GREATEST(threat_indicators.last_seen_at, EXCLUDED.last_seen_at),
				expires_at = EXCLUDED.expires_at,
				tags = EXCLUDED.tags,
				metadata = EXCLUDED.metadata,
				updated_at = now()`,
			indicator.ID, indicator.TenantID, indicator.ThreatID, indicator.Type, indicator.Value,
			indicator.Description, indicator.Severity, indicator.Source, indicator.Confidence,
			indicator.Active, coalesceTime(indicator.FirstSeenAt, now), coalesceTime(indicator.LastSeenAt, now),
			indicator.ExpiresAt, indicator.Tags, ensureRawMessage(indicator.Metadata, "{}"),
			indicator.CreatedBy, now,
		)
		if err != nil {
			return fmt.Errorf("upsert threat indicator: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return r.GetByTypeValue(ctx, indicator.TenantID, indicator.Type, indicator.Value)
}

// GetByTypeValue fetches an indicator by unique key.
func (r *IndicatorRepository) GetByTypeValue(ctx context.Context, tenantID uuid.UUID, indicatorType model.IndicatorType, value string) (*model.ThreatIndicator, error) {
	var indicator *model.ThreatIndicator
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		row := db.QueryRow(ctx, `
			SELECT
				id, tenant_id, threat_id, type, value, description, severity, source,
				confidence, active, first_seen_at, last_seen_at, expires_at,
				tags, metadata, created_by, created_at, updated_at
			FROM threat_indicators
			WHERE tenant_id = $1 AND type = $2 AND value = $3`,
			tenantID, indicatorType, value,
		)
		item, err := scanIndicator(row)
		if err != nil {
			if err == pgx.ErrNoRows {
				return ErrNotFound
			}
			return fmt.Errorf("get threat indicator: %w", err)
		}
		indicator = item
		return nil
	})
	return indicator, err
}

// GetByID fetches an indicator by primary key.
func (r *IndicatorRepository) GetByID(ctx context.Context, tenantID, indicatorID uuid.UUID) (*model.ThreatIndicator, error) {
	var indicator *model.ThreatIndicator
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		row := db.QueryRow(ctx, indicatorSelectWithThreat+`
			WHERE a.tenant_id = $1 AND a.id = $2`,
			tenantID, indicatorID,
		)
		item, err := scanIndicatorWithThreat(row)
		if err != nil {
			if err == pgx.ErrNoRows {
				return ErrNotFound
			}
			return fmt.Errorf("get threat indicator by id: %w", err)
		}
		indicator = item
		return nil
	})
	return indicator, err
}

// ListByThreat returns indicators attached to a threat.
func (r *IndicatorRepository) ListByThreat(ctx context.Context, tenantID, threatID uuid.UUID) ([]*model.ThreatIndicator, error) {
	indicators := make([]*model.ThreatIndicator, 0)
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, indicatorSelectWithThreat+`
			WHERE a.tenant_id = $1 AND a.threat_id = $2
			ORDER BY a.value ASC`,
			tenantID, threatID,
		)
		if err != nil {
			return fmt.Errorf("list threat indicators: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			indicator, err := scanIndicatorWithThreat(rows)
			if err != nil {
				return err
			}
			indicators = append(indicators, indicator)
		}
		return rows.Err()
	})
	return indicators, err
}

// UpdateActive toggles the active state for an indicator.
func (r *IndicatorRepository) UpdateActive(ctx context.Context, tenantID, indicatorID uuid.UUID, active bool) (*model.ThreatIndicator, error) {
	if err := runWithTenantWrite(ctx, r.db, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `
			UPDATE threat_indicators
			SET active = $3, updated_at = now()
			WHERE tenant_id = $1 AND id = $2`,
			tenantID, indicatorID, active,
		)
		if err != nil {
			return fmt.Errorf("update indicator active: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, tenantID, indicatorID)
}

// Update edits an indicator's mutable fields.
func (r *IndicatorRepository) Update(ctx context.Context, indicator *model.ThreatIndicator) (*model.ThreatIndicator, error) {
	if err := runWithTenantWrite(ctx, r.db, indicator.TenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `
			UPDATE threat_indicators
			SET
				threat_id = $3,
				type = $4,
				value = $5,
				description = $6,
				severity = $7,
				source = $8,
				confidence = $9,
				active = $10,
				expires_at = $11,
				tags = $12,
				metadata = $13,
				updated_at = now()
			WHERE tenant_id = $1 AND id = $2`,
			indicator.TenantID,
			indicator.ID,
			indicator.ThreatID,
			indicator.Type,
			indicator.Value,
			indicator.Description,
			indicator.Severity,
			indicator.Source,
			indicator.Confidence,
			indicator.Active,
			indicator.ExpiresAt,
			indicator.Tags,
			ensureRawMessage(indicator.Metadata, "{}"),
		)
		if err != nil {
			return fmt.Errorf("update indicator: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, indicator.TenantID, indicator.ID)
}

// Delete removes an indicator permanently.
func (r *IndicatorRepository) Delete(ctx context.Context, tenantID, indicatorID uuid.UUID) error {
	return runWithTenantWrite(ctx, r.db, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `
			DELETE FROM threat_indicators
			WHERE tenant_id = $1 AND id = $2`,
			tenantID, indicatorID,
		)
		if err != nil {
			return fmt.Errorf("delete indicator: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// ListActiveByTenant loads active, non-expired indicators for matcher refreshes.
func (r *IndicatorRepository) ListActiveByTenant(ctx context.Context, tenantID uuid.UUID) ([]*model.ThreatIndicator, error) {
	items := make([]*model.ThreatIndicator, 0)
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, `
			SELECT
				id, tenant_id, threat_id, type, value, description, severity, source,
				confidence, active, first_seen_at, last_seen_at, expires_at,
				tags, metadata, created_by, created_at, updated_at
			FROM threat_indicators
			WHERE tenant_id = $1
			  AND active = true
			  AND (expires_at IS NULL OR expires_at > now())
			ORDER BY updated_at DESC`,
			tenantID,
		)
		if err != nil {
			return fmt.Errorf("list active indicators: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			indicator, err := scanIndicator(rows)
			if err != nil {
				return err
			}
			items = append(items, indicator)
		}
		return rows.Err()
	})
	return items, err
}

// List returns a paginated list of indicators.
func (r *IndicatorRepository) List(ctx context.Context, tenantID uuid.UUID, params *dto.IndicatorListParams) ([]*model.ThreatIndicator, int, error) {
	qb := database.NewQueryBuilder(indicatorSelectWithThreat)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.WhereIn("a.type", params.Types)
	qb.WhereIn("a.source", params.Sources)
	qb.WhereIn("a.severity", params.Severities)
	if params.Linked != nil {
		if *params.Linked {
			qb.Where("a.threat_id IS NOT NULL")
		} else {
			qb.Where("a.threat_id IS NULL")
		}
	}
	if params.ThreatID != nil {
		qb.Where("a.threat_id = ?", *params.ThreatID)
	}
	if params.Active != nil {
		qb.Where("a.active = ?", *params.Active)
	}
	if params.MinConfidence != nil {
		qb.Where("a.confidence >= ?", *params.MinConfidence)
	}
	if params.MaxConfidence != nil {
		qb.Where("a.confidence <= ?", *params.MaxConfidence)
	}
	if params.Search != nil && strings.TrimSpace(*params.Search) != "" {
		search := "%" + strings.TrimSpace(*params.Search) + "%"
		qb.Where("(a.value ILIKE ? OR a.description ILIKE ? OR array_to_string(a.tags, ' ') ILIKE ?)", search, search, search)
	}
	qb.OrderBy(params.Sort, params.Order, []string{"created_at", "updated_at", "first_seen_at", "last_seen_at", "expires_at", "value", "source", "severity", "confidence", "type"})
	qb.Paginate(params.Page, params.PerPage)

	var total int
	items := make([]*model.ThreatIndicator, 0)
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		countSQL, countArgs := qb.BuildCount()
		if err := db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
			return fmt.Errorf("count indicators: %w", err)
		}

		sql, args := qb.Build()
		rows, err := db.Query(ctx, sql, args...)
		if err != nil {
			return fmt.Errorf("list indicators: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			indicator, err := scanIndicatorWithThreat(rows)
			if err != nil {
				return err
			}
			items = append(items, indicator)
		}
		return rows.Err()
	})
	return items, total, err
}

// Stats returns aggregate IOC metrics for dashboards.
func (r *IndicatorRepository) Stats(ctx context.Context, tenantID uuid.UUID) (*model.IndicatorStats, error) {
	stats := &model.IndicatorStats{}
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		if err := db.QueryRow(ctx, `
			SELECT
				COUNT(*),
				COUNT(*) FILTER (WHERE active = true),
				COUNT(*) FILTER (
					WHERE active = true
					  AND expires_at IS NOT NULL
					  AND expires_at <= now() + INTERVAL '7 days'
				)
			FROM threat_indicators
			WHERE tenant_id = $1`,
			tenantID,
		).Scan(&stats.Total, &stats.Active, &stats.ExpiringSoon); err != nil {
			return fmt.Errorf("indicator stats: %w", err)
		}

		rows, err := db.Query(ctx, `
			SELECT source, COUNT(*)
			FROM threat_indicators
			WHERE tenant_id = $1
			GROUP BY source
			ORDER BY COUNT(*) DESC, source ASC`,
			tenantID,
		)
		if err != nil {
			return fmt.Errorf("indicator stats by source: %w", err)
		}
		defer rows.Close()

		stats.BySource = make([]model.NamedCount, 0)
		for rows.Next() {
			var item model.NamedCount
			if err := rows.Scan(&item.Name, &item.Count); err != nil {
				return err
			}
			stats.BySource = append(stats.BySource, item)
		}
		return rows.Err()
	})
	return stats, err
}

// CheckValues matches arbitrary values against stored indicators.
func (r *IndicatorRepository) CheckValues(ctx context.Context, tenantID uuid.UUID, values []string) (map[string][]*model.ThreatIndicator, error) {
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{})
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		normalized = append(normalized, lower)
	}
	if len(normalized) == 0 {
		return map[string][]*model.ThreatIndicator{}, nil
	}
	results := make(map[string][]*model.ThreatIndicator)
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, `
			SELECT
				id, tenant_id, threat_id, type, value, description, severity, source,
				confidence, active, first_seen_at, last_seen_at, expires_at,
				tags, metadata, created_by, created_at, updated_at
			FROM threat_indicators
			WHERE tenant_id = $1
			  AND LOWER(value) = ANY($2)
			  AND active = true
			  AND (expires_at IS NULL OR expires_at > now())`,
			tenantID, normalized,
		)
		if err != nil {
			return fmt.Errorf("check indicator values: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			indicator, err := scanIndicator(rows)
			if err != nil {
				return err
			}
			key := strings.ToLower(indicator.Value)
			results[key] = append(results[key], indicator)
		}
		return rows.Err()
	})
	return results, err
}

// ListRecentByFeed returns recently imported indicators for one feed.
func (r *IndicatorRepository) ListRecentByFeed(ctx context.Context, tenantID, feedID uuid.UUID, limit int) ([]*model.ThreatIndicator, error) {
	if limit <= 0 {
		limit = 10
	}
	items := make([]*model.ThreatIndicator, 0, limit)
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, indicatorSelectWithThreat+`
			WHERE a.tenant_id = $1
			  AND a.metadata->>'feed_id' = $2
			ORDER BY a.created_at DESC
			LIMIT $3`,
			tenantID,
			feedID.String(),
			limit,
		)
		if err != nil {
			return fmt.Errorf("list recent indicators by feed: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			item, err := scanIndicatorWithThreat(rows)
			if err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

// ListMatches returns recent alert and security-event matches for an indicator.
func (r *IndicatorRepository) ListMatches(ctx context.Context, tenantID uuid.UUID, indicator *model.ThreatIndicator, limit int) ([]*model.IndicatorDetectionMatch, error) {
	if limit <= 0 {
		limit = 25
	}

	results := make([]*model.IndicatorDetectionMatch, 0, limit)
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		alertRows, err := db.Query(ctx, `
			SELECT
				a.id::text,
				'alert',
				a.title,
				a.description,
				a.severity::text,
				a.status::text,
				a.asset_id::text,
				COALESCE(assets.name, assets.hostname, host(assets.ip_address)),
				COALESCE(elem->>'field', ''),
				COALESCE(elem->>'value', ''),
				a.created_at
			FROM alerts a
			LEFT JOIN assets ON assets.tenant_id = a.tenant_id AND assets.id = a.asset_id AND assets.deleted_at IS NULL
			CROSS JOIN LATERAL jsonb_array_elements(COALESCE(a.explanation->'indicator_matches', '[]'::jsonb)) AS elem
			WHERE a.tenant_id = $1
			  AND a.deleted_at IS NULL
			  AND LOWER(elem->>'value') = LOWER($2)
			ORDER BY a.created_at DESC
			LIMIT $3`,
			tenantID, indicator.Value, limit,
		)
		if err != nil {
			return fmt.Errorf("list indicator alert matches: %w", err)
		}
		defer alertRows.Close()

		for alertRows.Next() {
			var item model.IndicatorDetectionMatch
			var severityText string
			var statusText string
			if err := alertRows.Scan(
				&item.ID,
				&item.Kind,
				&item.Title,
				&item.Description,
				&severityText,
				&statusText,
				&item.AssetID,
				&item.AssetName,
				&item.MatchField,
				&item.MatchValue,
				&item.Timestamp,
			); err != nil {
				return err
			}
			severity := model.Severity(severityText)
			item.Severity = &severity
			item.Status = &statusText
			results = append(results, &item)
		}
		if err := alertRows.Err(); err != nil {
			return err
		}

		query, args := securityEventMatchQuery(tenantID, indicator, limit)
		if query == "" {
			return nil
		}

		eventRows, err := db.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("list indicator security-event matches: %w", err)
		}
		defer eventRows.Close()

		for eventRows.Next() {
			var item model.IndicatorDetectionMatch
			var severityText string
			if err := eventRows.Scan(
				&item.ID,
				&item.Kind,
				&item.Title,
				&item.Description,
				&severityText,
				&item.AssetID,
				&item.AssetName,
				&item.MatchField,
				&item.MatchValue,
				&item.Timestamp,
			); err != nil {
				return err
			}
			severity := model.Severity(severityText)
			item.Severity = &severity
			results = append(results, &item)
		}
		return eventRows.Err()
	})
	return results, err
}

func securityEventMatchQuery(tenantID uuid.UUID, indicator *model.ThreatIndicator, limit int) (string, []interface{}) {
	base := `
		SELECT
			e.id::text,
			'event',
			e.type,
			e.source,
			e.severity::text,
			e.asset_id::text,
			COALESCE(assets.name, assets.hostname, host(assets.ip_address)),
			%s,
			%s,
			e.timestamp
		FROM security_events e
		LEFT JOIN assets ON assets.tenant_id = e.tenant_id AND assets.id = e.asset_id AND assets.deleted_at IS NULL
		WHERE e.tenant_id = $1 AND %s
		ORDER BY e.timestamp DESC
		LIMIT $3`

	switch indicator.Type {
	case model.IndicatorTypeIP:
		return fmt.Sprintf(base, "'source_ip_or_dest_ip'", "$2", "(e.source_ip::text = $2 OR e.dest_ip::text = $2)"), []interface{}{tenantID, indicator.Value, limit}
	case model.IndicatorTypeCIDR:
		return fmt.Sprintf(base, "'source_ip_or_dest_ip'", "$2", "(e.source_ip <<= $2::cidr OR e.dest_ip <<= $2::cidr)"), []interface{}{tenantID, indicator.Value, limit}
	case model.IndicatorTypeHashMD5, model.IndicatorTypeHashSHA1, model.IndicatorTypeHashSHA256:
		return fmt.Sprintf(base, "'file_hash'", "$2", "(LOWER(e.file_hash) = LOWER($2) OR e.raw_event::text ILIKE '%%' || $2 || '%%')"), []interface{}{tenantID, indicator.Value, limit}
	case model.IndicatorTypeUserAgent:
		return fmt.Sprintf(base, "'raw.user_agent'", "$2", "e.raw_event::text ILIKE '%%' || $2 || '%%'"), []interface{}{tenantID, indicator.Value, limit}
	default:
		return fmt.Sprintf(base, "'raw_event'", "$2", "e.raw_event::text ILIKE '%%' || $2 || '%%'"), []interface{}{tenantID, indicator.Value, limit}
	}
}

func coalesceTime(value, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}
	return value
}
