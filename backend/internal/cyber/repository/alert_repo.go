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

// AlertRepository handles alert storage and lifecycle persistence.
type AlertRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewAlertRepository creates a new AlertRepository.
func NewAlertRepository(db *pgxpool.Pool, logger zerolog.Logger) *AlertRepository {
	return &AlertRepository{db: db, logger: logger}
}

// List returns a paginated list of alerts.
func (r *AlertRepository) List(ctx context.Context, tenantID uuid.UUID, params *dto.AlertListParams) ([]*model.Alert, int, error) {
	baseSelect := `
		SELECT
			a.id, a.tenant_id, a.title, a.description, a.severity, a.status,
			a.source, a.rule_id, a.asset_id, a.asset_ids, a.assigned_to, a.assigned_at,
			a.escalated_to, a.escalated_at, a.explanation, a.confidence_score,
			a.mitre_tactic_id, a.mitre_tactic_name, a.mitre_technique_id, a.mitre_technique_name,
			a.event_count, a.first_event_at, a.last_event_at, a.resolved_at,
			a.resolution_notes, a.false_positive_reason, a.tags, a.metadata,
			a.created_at, a.updated_at, a.deleted_at
		FROM alerts a`
	qb := database.NewQueryBuilder(baseSelect)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.Where("a.deleted_at IS NULL")
	if params.Search != nil && strings.TrimSpace(*params.Search) != "" {
		qb.WhereFTS([]string{"a.title", "a.description"}, strings.TrimSpace(*params.Search))
	}
	if len(params.Severities) > 0 {
		qb.WhereIn("a.severity", params.Severities)
	}
	if len(params.Statuses) > 0 {
		qb.WhereIn("a.status", params.Statuses)
	}
	if params.AssignedTo != nil {
		qb.Where("a.assigned_to = ?", *params.AssignedTo)
	}
	if params.Unassigned != nil && *params.Unassigned {
		qb.Where("a.assigned_to IS NULL")
	}
	if params.AssetID != nil {
		qb.Where("(a.asset_id = ? OR ? = ANY(a.asset_ids))", *params.AssetID, *params.AssetID)
	}
	if params.RuleID != nil {
		qb.Where("a.rule_id = ?", *params.RuleID)
	}
	if params.MITRETechniqueID != nil {
		qb.Where("a.mitre_technique_id = ?", *params.MITRETechniqueID)
	}
	if params.MITRETacticID != nil {
		qb.Where("a.mitre_tactic_id = ?", *params.MITRETacticID)
	}
	if params.MinConfidence != nil {
		qb.Where("a.confidence_score >= ?", *params.MinConfidence)
	}
	if len(params.Tags) > 0 {
		qb.WhereArrayContainsAll("a.tags", params.Tags)
	}
	if params.DateFrom != nil {
		qb.Where("a.created_at >= ?", *params.DateFrom)
	}
	if params.DateTo != nil {
		qb.Where("a.created_at <= ?", *params.DateTo)
	}
	qb.OrderBy(params.Sort, params.Order, []string{"severity", "confidence_score", "created_at", "event_count", "status"})
	qb.Paginate(params.Page, params.PerPage)

	countSQL, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count alerts: %w", err)
	}
	sql, args := qb.Build()
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()

	alerts := make([]*model.Alert, 0)
	for rows.Next() {
		alert, err := scanAlert(rows)
		if err != nil {
			return nil, 0, err
		}
		alerts = append(alerts, alert)
	}
	return alerts, total, rows.Err()
}

// Count returns a simple count with the provided filters.
func (r *AlertRepository) Count(ctx context.Context, tenantID uuid.UUID, params *dto.AlertListParams) (int, error) {
	_, total, err := r.List(ctx, tenantID, params)
	return total, err
}

// GetByID fetches a single alert.
func (r *AlertRepository) GetByID(ctx context.Context, tenantID, alertID uuid.UUID) (*model.Alert, error) {
	row := r.db.QueryRow(ctx, `
		SELECT
			id, tenant_id, title, description, severity, status,
			source, rule_id, asset_id, asset_ids, assigned_to, assigned_at,
			escalated_to, escalated_at, explanation, confidence_score,
			mitre_tactic_id, mitre_tactic_name, mitre_technique_id, mitre_technique_name,
			event_count, first_event_at, last_event_at, resolved_at,
			resolution_notes, false_positive_reason, tags, metadata,
			created_at, updated_at, deleted_at
		FROM alerts
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, alertID,
	)
	alert, err := scanAlert(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get alert: %w", err)
	}
	return alert, nil
}

// GetByIDs fetches multiple alerts for merge operations.
func (r *AlertRepository) GetByIDs(ctx context.Context, tenantID uuid.UUID, alertIDs []uuid.UUID) ([]*model.Alert, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			id, tenant_id, title, description, severity, status,
			source, rule_id, asset_id, asset_ids, assigned_to, assigned_at,
			escalated_to, escalated_at, explanation, confidence_score,
			mitre_tactic_id, mitre_tactic_name, mitre_technique_id, mitre_technique_name,
			event_count, first_event_at, last_event_at, resolved_at,
			resolution_notes, false_positive_reason, tags, metadata,
			created_at, updated_at, deleted_at
		FROM alerts
		WHERE tenant_id = $1 AND id = ANY($2) AND deleted_at IS NULL`,
		tenantID, alertIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("get alerts by ids: %w", err)
	}
	defer rows.Close()
	alerts := make([]*model.Alert, 0)
	for rows.Next() {
		alert, err := scanAlert(rows)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}
	return alerts, rows.Err()
}

// Create inserts an alert.
func (r *AlertRepository) Create(ctx context.Context, alert *model.Alert) (*model.Alert, error) {
	if alert.ID == uuid.Nil {
		alert.ID = uuid.New()
	}
	explanation, err := marshalJSON(alert.Explanation)
	if err != nil {
		return nil, fmt.Errorf("marshal alert explanation: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO alerts (
			id, tenant_id, title, description, severity, status, source, rule_id,
			asset_id, asset_ids, assigned_to, assigned_at, escalated_to, escalated_at,
			explanation, confidence_score, mitre_tactic_id, mitre_tactic_name,
			mitre_technique_id, mitre_technique_name, event_count, first_event_at,
			last_event_at, resolved_at, resolution_notes, false_positive_reason,
			tags, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18,
			$19, $20, $21, $22,
			$23, $24, $25, $26,
			$27, $28, now(), now()
		)`,
		alert.ID, alert.TenantID, alert.Title, alert.Description, alert.Severity, alert.Status, alert.Source, alert.RuleID,
		alert.AssetID, alert.AssetIDs, alert.AssignedTo, alert.AssignedAt, alert.EscalatedTo, alert.EscalatedAt,
		explanation, alert.ConfidenceScore, alert.MITRETacticID, alert.MITRETacticName,
		alert.MITRETechniqueID, alert.MITRETechniqueName, alert.EventCount, alert.FirstEventAt,
		alert.LastEventAt, alert.ResolvedAt, alert.ResolutionNotes, alert.FalsePositiveReason,
		alert.Tags, ensureRawMessage(alert.Metadata, "{}"),
	)
	if err != nil {
		return nil, fmt.Errorf("insert alert: %w", err)
	}
	return r.GetByID(ctx, alert.TenantID, alert.ID)
}

// UpdateStatus updates an alert status and optional resolution fields.
func (r *AlertRepository) UpdateStatus(ctx context.Context, tenantID, alertID uuid.UUID, status model.AlertStatus, notes, reason *string) (*model.Alert, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE alerts
		SET
			status = $3,
			resolution_notes = COALESCE($4, resolution_notes),
			false_positive_reason = CASE WHEN $3 = 'false_positive' THEN COALESCE($5, false_positive_reason) ELSE false_positive_reason END,
			resolved_at = CASE WHEN $3 IN ('resolved', 'closed', 'false_positive') THEN now() ELSE resolved_at END,
			updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, alertID, status, notes, reason,
	)
	if err != nil {
		return nil, fmt.Errorf("update alert status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, tenantID, alertID)
}

// Assign sets the assigned analyst for an alert.
func (r *AlertRepository) Assign(ctx context.Context, tenantID, alertID, assignedTo uuid.UUID) (*model.Alert, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE alerts
		SET assigned_to = $3, assigned_at = now(), updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, alertID, assignedTo,
	)
	if err != nil {
		return nil, fmt.Errorf("assign alert: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, tenantID, alertID)
}

// Escalate sets the escalation target for an alert.
func (r *AlertRepository) Escalate(ctx context.Context, tenantID, alertID, escalatedTo uuid.UUID) (*model.Alert, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE alerts
		SET
			status = 'escalated',
			escalated_to = $3,
			escalated_at = now(),
			updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, alertID, escalatedTo,
	)
	if err != nil {
		return nil, fmt.Errorf("escalate alert: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, tenantID, alertID)
}

// InsertTimeline adds an immutable alert timeline entry.
func (r *AlertRepository) InsertTimeline(ctx context.Context, entry *model.AlertTimelineEntry) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO alert_timeline (
			id, tenant_id, alert_id, action, actor_id, actor_name, old_value, new_value, description, metadata, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now()
		)`,
		entry.ID, entry.TenantID, entry.AlertID, entry.Action, entry.ActorID, entry.ActorName,
		entry.OldValue, entry.NewValue, entry.Description, ensureRawMessage(entry.Metadata, "{}"),
	)
	if err != nil {
		return fmt.Errorf("insert alert timeline entry: %w", err)
	}
	return nil
}

// ListTimeline returns alert timeline entries in chronological order.
func (r *AlertRepository) ListTimeline(ctx context.Context, tenantID, alertID uuid.UUID) ([]*model.AlertTimelineEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			id, tenant_id, alert_id, action, actor_id, actor_name,
			old_value, new_value, description, metadata, created_at
		FROM alert_timeline
		WHERE tenant_id = $1 AND alert_id = $2
		ORDER BY created_at ASC`,
		tenantID, alertID,
	)
	if err != nil {
		return nil, fmt.Errorf("list alert timeline: %w", err)
	}
	defer rows.Close()
	items := make([]*model.AlertTimelineEntry, 0)
	for rows.Next() {
		item, err := scanAlertTimeline(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// FindOpenByRuleAndAsset finds an existing open alert for deduplication.
func (r *AlertRepository) FindOpenByRuleAndAsset(ctx context.Context, tenantID, ruleID uuid.UUID, assetID *uuid.UUID) (*model.Alert, error) {
	if assetID == nil {
		return nil, ErrNotFound
	}
	row := r.db.QueryRow(ctx, `
		SELECT
			id, tenant_id, title, description, severity, status,
			source, rule_id, asset_id, asset_ids, assigned_to, assigned_at,
			escalated_to, escalated_at, explanation, confidence_score,
			mitre_tactic_id, mitre_tactic_name, mitre_technique_id, mitre_technique_name,
			event_count, first_event_at, last_event_at, resolved_at,
			resolution_notes, false_positive_reason, tags, metadata,
			created_at, updated_at, deleted_at
		FROM alerts
		WHERE tenant_id = $1
		  AND rule_id = $2
		  AND asset_id = $3
		  AND status IN ('new', 'acknowledged', 'investigating', 'in_progress', 'escalated')
		  AND deleted_at IS NULL
		ORDER BY created_at ASC
		LIMIT 1`,
		tenantID, ruleID, *assetID,
	)
	alert, err := scanAlert(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find open alert by rule and asset: %w", err)
	}
	return alert, nil
}

// UpdateAggregatedDetectionAlert updates an existing deduplicated alert with more events.
func (r *AlertRepository) UpdateAggregatedDetectionAlert(ctx context.Context, tenantID, alertID uuid.UUID, additionalEvents int, lastEventAt time.Time, assetIDs []uuid.UUID, explanation *model.AlertExplanation) (*model.Alert, error) {
	explanationJSON, err := marshalJSON(explanation)
	if err != nil {
		return nil, fmt.Errorf("marshal updated explanation: %w", err)
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE alerts
		SET
			event_count = event_count + $3,
			last_event_at = GREATEST(last_event_at, $4),
			asset_ids = $5,
			explanation = $6,
			updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, alertID, additionalEvents, lastEventAt, assetIDs, explanationJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("update aggregated alert: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, tenantID, alertID)
}

// FindRelated returns alerts related by asset, rule, or MITRE technique.
func (r *AlertRepository) FindRelated(ctx context.Context, tenantID, alertID uuid.UUID) ([]*model.Alert, error) {
	alert, err := r.GetByID(ctx, tenantID, alertID)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT
			id, tenant_id, title, description, severity, status,
			source, rule_id, asset_id, asset_ids, assigned_to, assigned_at,
			escalated_to, escalated_at, explanation, confidence_score,
			mitre_tactic_id, mitre_tactic_name, mitre_technique_id, mitre_technique_name,
			event_count, first_event_at, last_event_at, resolved_at,
			resolution_notes, false_positive_reason, tags, metadata,
			created_at, updated_at, deleted_at
		FROM alerts
		WHERE tenant_id = $1
		  AND id <> $2
		  AND deleted_at IS NULL
		  AND (
			(asset_id IS NOT NULL AND $3 IS NOT NULL AND asset_id = $3) OR
			(rule_id IS NOT NULL AND $4 IS NOT NULL AND rule_id = $4) OR
			(mitre_technique_id IS NOT NULL AND $5 IS NOT NULL AND mitre_technique_id = $5)
		  )
		ORDER BY created_at DESC
		LIMIT 25`,
		tenantID, alertID, alert.AssetID, alert.RuleID, alert.MITRETechniqueID,
	)
	if err != nil {
		return nil, fmt.Errorf("find related alerts: %w", err)
	}
	defer rows.Close()
	results := make([]*model.Alert, 0)
	for rows.Next() {
		related, err := scanAlert(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, related)
	}
	return results, rows.Err()
}

// CloneTimeline copies timeline entries from one alert to another for merge operations.
func (r *AlertRepository) CloneTimeline(ctx context.Context, tenantID, fromAlertID, toAlertID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO alert_timeline (
			id, tenant_id, alert_id, action, actor_id, actor_name, old_value, new_value, description, metadata, created_at
		)
		SELECT
			gen_random_uuid(), tenant_id, $3, action, actor_id, actor_name, old_value, new_value,
			description, metadata || jsonb_build_object('source_alert_id', alert_id::text), created_at
		FROM alert_timeline
		WHERE tenant_id = $1 AND alert_id = $2`,
		tenantID, fromAlertID, toAlertID,
	)
	if err != nil {
		return fmt.Errorf("clone alert timeline: %w", err)
	}
	return nil
}

// MarkMerged marks a secondary alert as merged into a primary alert.
func (r *AlertRepository) MarkMerged(ctx context.Context, tenantID, alertID, primaryAlertID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE alerts
		SET
			status = 'merged',
			metadata = metadata || jsonb_build_object('primary_alert_id', $3::text),
			updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, alertID, primaryAlertID,
	)
	if err != nil {
		return fmt.Errorf("mark alert merged: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateAfterMerge updates the primary alert after a merge operation.
func (r *AlertRepository) UpdateAfterMerge(ctx context.Context, tenantID, alertID uuid.UUID, eventCount int, assetID *uuid.UUID, assetIDs []uuid.UUID, explanation *model.AlertExplanation) (*model.Alert, error) {
	payload, err := marshalJSON(explanation)
	if err != nil {
		return nil, fmt.Errorf("marshal merged explanation: %w", err)
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE alerts
		SET
			event_count = $3,
			asset_id = COALESCE($4, asset_id),
			asset_ids = $5,
			explanation = $6,
			last_event_at = now(),
			updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, alertID, eventCount, assetID, assetIDs, payload,
	)
	if err != nil {
		return nil, fmt.Errorf("update merged alert: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, tenantID, alertID)
}

// Stats returns aggregated alert statistics.
func (r *AlertRepository) Stats(ctx context.Context, tenantID uuid.UUID) (*model.AlertStats, error) {
	stats := &model.AlertStats{}
	if err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE deleted_at IS NULL AND status IN ('new', 'acknowledged', 'investigating', 'in_progress', 'escalated')),
			COUNT(*) FILTER (WHERE deleted_at IS NULL AND status IN ('resolved', 'closed'))
		FROM alerts
		WHERE tenant_id = $1`,
		tenantID,
	).Scan(&stats.OpenCount, &stats.ResolvedCount); err != nil {
		return nil, fmt.Errorf("alert totals: %w", err)
	}

	buildCounts := func(sql string) ([]model.NamedCount, error) {
		rows, err := r.db.Query(ctx, sql, tenantID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		items := make([]model.NamedCount, 0)
		for rows.Next() {
			var item model.NamedCount
			if err := rows.Scan(&item.Name, &item.Count); err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		return items, rows.Err()
	}

	var err error
	if stats.BySeverity, err = buildCounts(`
		SELECT severity::text, COUNT(*)
		FROM alerts
		WHERE tenant_id = $1 AND deleted_at IS NULL
		GROUP BY severity
		ORDER BY COUNT(*) DESC, severity ASC`); err != nil {
		return nil, fmt.Errorf("stats by severity: %w", err)
	}
	if stats.ByStatus, err = buildCounts(`
		SELECT status::text, COUNT(*)
		FROM alerts
		WHERE tenant_id = $1 AND deleted_at IS NULL
		GROUP BY status
		ORDER BY COUNT(*) DESC, status ASC`); err != nil {
		return nil, fmt.Errorf("stats by status: %w", err)
	}
	if stats.ByRule, err = buildCounts(`
		SELECT COALESCE(source, 'unknown'), COUNT(*)
		FROM alerts
		WHERE tenant_id = $1 AND deleted_at IS NULL
		GROUP BY source
		ORDER BY COUNT(*) DESC, source ASC`); err != nil {
		return nil, fmt.Errorf("stats by rule: %w", err)
	}
	if stats.ByTechnique, err = buildCounts(`
		SELECT COALESCE(mitre_technique_id, 'unmapped'), COUNT(*)
		FROM alerts
		WHERE tenant_id = $1 AND deleted_at IS NULL
		GROUP BY mitre_technique_id
		ORDER BY COUNT(*) DESC, mitre_technique_id ASC`); err != nil {
		return nil, fmt.Errorf("stats by technique: %w", err)
	}
	return stats, nil
}
