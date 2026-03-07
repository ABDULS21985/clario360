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
	qb.OrderBy("created_at", "desc", []string{"created_at"})
	qb.Paginate(params.Page, params.PerPage)

	countSQL, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count threats: %w", err)
	}

	sql, args := qb.Build()
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list threats: %w", err)
	}
	defer rows.Close()

	items := make([]*model.Threat, 0)
	for rows.Next() {
		threat, err := scanThreat(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, threat)
	}
	return items, total, rows.Err()
}

// GetByID fetches a single threat.
func (r *ThreatRepository) GetByID(ctx context.Context, tenantID, threatID uuid.UUID) (*model.Threat, error) {
	row := r.db.QueryRow(ctx, `
		SELECT
			id, tenant_id, name, description, type, severity, status,
			threat_actor, campaign, mitre_tactic_ids, mitre_technique_ids,
			affected_asset_count, alert_count, first_seen_at, last_seen_at,
			contained_at, tags, metadata, created_by, created_at, updated_at, deleted_at
		FROM threats
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, threatID,
	)
	item, err := scanThreat(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get threat: %w", err)
	}
	return item, nil
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
	_, err := r.db.Exec(ctx, `
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
		return nil, fmt.Errorf("insert threat: %w", err)
	}
	return r.GetByID(ctx, threat.TenantID, threat.ID)
}

// UpdateStatus updates a threat status.
func (r *ThreatRepository) UpdateStatus(ctx context.Context, tenantID, threatID uuid.UUID, status model.ThreatStatus) (*model.Threat, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE threats
		SET
			status = $3,
			contained_at = CASE WHEN $3 = 'contained' THEN now() ELSE contained_at END,
			updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, threatID, status,
	)
	if err != nil {
		return nil, fmt.Errorf("update threat status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, tenantID, threatID)
}

// UpsertSyntheticThreat ensures there is a threat record associated with an indicator-driven detection.
func (r *ThreatRepository) UpsertSyntheticThreat(ctx context.Context, tenantID uuid.UUID, name, description string, threatType model.ThreatType, severity model.Severity, tags []string) (*model.Threat, error) {
	row := r.db.QueryRow(ctx, `
		SELECT
			id, tenant_id, name, description, type, severity, status,
			threat_actor, campaign, mitre_tactic_ids, mitre_technique_ids,
			affected_asset_count, alert_count, first_seen_at, last_seen_at,
			contained_at, tags, metadata, created_by, created_at, updated_at, deleted_at
		FROM threats
		WHERE tenant_id = $1 AND name = $2 AND deleted_at IS NULL`,
		tenantID, name,
	)
	item, err := scanThreat(row)
	if err == nil {
		return item, nil
	}
	if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("lookup synthetic threat: %w", err)
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
	tag, err := r.db.Exec(ctx, `
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
}

// Stats returns aggregated threat counts.
func (r *ThreatRepository) Stats(ctx context.Context, tenantID uuid.UUID) (*model.ThreatStats, error) {
	stats := &model.ThreatStats{}
	if err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE deleted_at IS NULL),
			COUNT(*) FILTER (WHERE deleted_at IS NULL AND status = 'active')
		FROM threats
		WHERE tenant_id = $1`,
		tenantID,
	).Scan(&stats.Total, &stats.Active); err != nil {
		return nil, fmt.Errorf("threat totals: %w", err)
	}

	queryCounts := func(column string) ([]model.NamedCount, error) {
		rows, err := r.db.Query(ctx, fmt.Sprintf(`
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
		return nil, fmt.Errorf("threat stats by type: %w", err)
	}
	if stats.ByStatus, err = queryCounts("status"); err != nil {
		return nil, fmt.Errorf("threat stats by status: %w", err)
	}
	if stats.BySeverity, err = queryCounts("severity"); err != nil {
		return nil, fmt.Errorf("threat stats by severity: %w", err)
	}
	return stats, nil
}
