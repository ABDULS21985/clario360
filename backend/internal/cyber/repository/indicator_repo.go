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
	_, err := r.db.Exec(ctx, `
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
		return nil, fmt.Errorf("upsert threat indicator: %w", err)
	}
	return r.GetByTypeValue(ctx, indicator.TenantID, indicator.Type, indicator.Value)
}

// GetByTypeValue fetches an indicator by unique key.
func (r *IndicatorRepository) GetByTypeValue(ctx context.Context, tenantID uuid.UUID, indicatorType model.IndicatorType, value string) (*model.ThreatIndicator, error) {
	row := r.db.QueryRow(ctx, `
		SELECT
			id, tenant_id, threat_id, type, value, description, severity, source,
			confidence, active, first_seen_at, last_seen_at, expires_at,
			tags, metadata, created_by, created_at, updated_at
		FROM threat_indicators
		WHERE tenant_id = $1 AND type = $2 AND value = $3`,
		tenantID, indicatorType, value,
	)
	indicator, err := scanIndicator(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get threat indicator: %w", err)
	}
	return indicator, nil
}

// ListByThreat returns indicators attached to a threat.
func (r *IndicatorRepository) ListByThreat(ctx context.Context, tenantID, threatID uuid.UUID) ([]*model.ThreatIndicator, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			id, tenant_id, threat_id, type, value, description, severity, source,
			confidence, active, first_seen_at, last_seen_at, expires_at,
			tags, metadata, created_by, created_at, updated_at
		FROM threat_indicators
		WHERE tenant_id = $1 AND threat_id = $2
		ORDER BY value ASC`,
		tenantID, threatID,
	)
	if err != nil {
		return nil, fmt.Errorf("list threat indicators: %w", err)
	}
	defer rows.Close()

	indicators := make([]*model.ThreatIndicator, 0)
	for rows.Next() {
		indicator, err := scanIndicator(rows)
		if err != nil {
			return nil, err
		}
		indicators = append(indicators, indicator)
	}
	return indicators, rows.Err()
}

// ListActiveByTenant loads active, non-expired indicators for matcher refreshes.
func (r *IndicatorRepository) ListActiveByTenant(ctx context.Context, tenantID uuid.UUID) ([]*model.ThreatIndicator, error) {
	rows, err := r.db.Query(ctx, `
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
		return nil, fmt.Errorf("list active indicators: %w", err)
	}
	defer rows.Close()

	items := make([]*model.ThreatIndicator, 0)
	for rows.Next() {
		indicator, err := scanIndicator(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, indicator)
	}
	return items, rows.Err()
}

// List returns a paginated list of indicators.
func (r *IndicatorRepository) List(ctx context.Context, tenantID uuid.UUID, params *dto.IndicatorListParams) ([]*model.ThreatIndicator, int, error) {
	baseSelect := `
		SELECT
			a.id, a.tenant_id, a.threat_id, a.type, a.value, a.description, a.severity, a.source,
			a.confidence, a.active, a.first_seen_at, a.last_seen_at, a.expires_at,
			a.tags, a.metadata, a.created_by, a.created_at, a.updated_at
		FROM threat_indicators a`
	qb := database.NewQueryBuilder(baseSelect)
	qb.Where("a.tenant_id = ?", tenantID)
	if params.Type != nil {
		qb.Where("a.type = ?", *params.Type)
	}
	if params.ThreatID != nil {
		qb.Where("a.threat_id = ?", *params.ThreatID)
	}
	if params.Active != nil {
		qb.Where("a.active = ?", *params.Active)
	}
	if params.Search != nil && strings.TrimSpace(*params.Search) != "" {
		search := "%" + strings.TrimSpace(*params.Search) + "%"
		qb.Where("(a.value ILIKE ? OR a.description ILIKE ?)", search, search)
	}
	qb.OrderBy("created_at", "desc", []string{"created_at"})
	qb.Paginate(params.Page, params.PerPage)

	countSQL, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count indicators: %w", err)
	}

	sql, args := qb.Build()
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list indicators: %w", err)
	}
	defer rows.Close()

	items := make([]*model.ThreatIndicator, 0)
	for rows.Next() {
		indicator, err := scanIndicator(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, indicator)
	}
	return items, total, rows.Err()
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
	rows, err := r.db.Query(ctx, `
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
		return nil, fmt.Errorf("check indicator values: %w", err)
	}
	defer rows.Close()

	results := make(map[string][]*model.ThreatIndicator)
	for rows.Next() {
		indicator, err := scanIndicator(rows)
		if err != nil {
			return nil, err
		}
		key := strings.ToLower(indicator.Value)
		results[key] = append(results[key], indicator)
	}
	return results, rows.Err()
}

func coalesceTime(value, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}
	return value
}
