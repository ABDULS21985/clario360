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
		FROM alerts a
		LEFT JOIN detection_rules dr ON dr.tenant_id = a.tenant_id AND dr.id = a.rule_id`
	qb := database.NewQueryBuilder(baseSelect)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.Where("a.deleted_at IS NULL")
	if params.Search != nil && strings.TrimSpace(*params.Search) != "" {
		search := strings.TrimSpace(*params.Search)
		qb.Where(
			"(to_tsvector('english', coalesce(a.title, '') || ' ' || coalesce(a.description, '')) @@ plainto_tsquery('english', ?) OR a.title ILIKE ? OR a.description ILIKE ?)",
			search,
			"%"+search+"%",
			"%"+search+"%",
		)
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
	if params.RuleType != nil {
		qb.Where("dr.rule_type = ?", *params.RuleType)
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
	if params.MaxConfidence != nil {
		qb.Where("a.confidence_score <= ?", *params.MaxConfidence)
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

	var total int
	alerts := make([]*model.Alert, 0)
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		countSQL, countArgs := qb.BuildCount()
		if err := db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
			return fmt.Errorf("count alerts: %w", err)
		}
		sql, args := qb.Build()
		rows, err := db.Query(ctx, sql, args...)
		if err != nil {
			return fmt.Errorf("list alerts: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			alert, err := scanAlert(rows)
			if err != nil {
				return err
			}
			alerts = append(alerts, alert)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, err
	}
	if err := r.enrichAlerts(ctx, tenantID, alerts); err != nil {
		return nil, 0, err
	}
	return alerts, total, nil
}

// Count returns a simple count with the provided filters.
func (r *AlertRepository) Count(ctx context.Context, tenantID uuid.UUID, params *dto.AlertListParams) (int, error) {
	_, total, err := r.List(ctx, tenantID, params)
	return total, err
}

// GetByID fetches a single alert.
func (r *AlertRepository) GetByID(ctx context.Context, tenantID, alertID uuid.UUID) (*model.Alert, error) {
	var alert *model.Alert
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		row := db.QueryRow(ctx, `
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
		item, err := scanAlert(row)
		if err != nil {
			if err == pgx.ErrNoRows {
				return ErrNotFound
			}
			return fmt.Errorf("get alert: %w", err)
		}
		alert = item
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := r.enrichAlerts(ctx, tenantID, []*model.Alert{alert}); err != nil {
		return nil, err
	}
	return alert, nil
}

// GetByIDs fetches multiple alerts for merge operations.
func (r *AlertRepository) GetByIDs(ctx context.Context, tenantID uuid.UUID, alertIDs []uuid.UUID) ([]*model.Alert, error) {
	alerts := make([]*model.Alert, 0)
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, `
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
			return fmt.Errorf("get alerts by ids: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			alert, err := scanAlert(rows)
			if err != nil {
				return err
			}
			alerts = append(alerts, alert)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	if err := r.enrichAlerts(ctx, tenantID, alerts); err != nil {
		return nil, err
	}
	return alerts, nil
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
	if err := runWithTenantWrite(ctx, r.db, alert.TenantID, func(db dbtx) error {
		_, err = db.Exec(ctx, `
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
			return fmt.Errorf("insert alert: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, alert.TenantID, alert.ID)
}

// UpdateStatus updates an alert status and optional resolution fields.
func (r *AlertRepository) UpdateStatus(ctx context.Context, tenantID, alertID uuid.UUID, status model.AlertStatus, notes, reason *string) (*model.Alert, error) {
	if err := runWithTenantWrite(ctx, r.db, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `
			UPDATE alerts
			SET
				status = $3,
				resolution_notes = COALESCE($4, resolution_notes),
				false_positive_reason = CASE WHEN $3 = 'false_positive' THEN COALESCE($5, false_positive_reason) ELSE false_positive_reason END,
				resolved_at = CASE WHEN $3 IN ('resolved', 'closed', 'false_positive') THEN now() ELSE NULL END,
				updated_at = now()
			WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
			tenantID, alertID, status, notes, reason,
		)
		if err != nil {
			return fmt.Errorf("update alert status: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, tenantID, alertID)
}

// Assign sets the assigned analyst for an alert.
func (r *AlertRepository) Assign(ctx context.Context, tenantID, alertID, assignedTo uuid.UUID) (*model.Alert, error) {
	if err := runWithTenantWrite(ctx, r.db, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `
			UPDATE alerts
			SET assigned_to = $3, assigned_at = now(), updated_at = now()
			WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
			tenantID, alertID, assignedTo,
		)
		if err != nil {
			return fmt.Errorf("assign alert: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, tenantID, alertID)
}

// Escalate sets the escalation target for an alert.
func (r *AlertRepository) Escalate(ctx context.Context, tenantID, alertID, escalatedTo uuid.UUID) (*model.Alert, error) {
	if err := runWithTenantWrite(ctx, r.db, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `
			UPDATE alerts
			SET
				status = 'escalated',
				assigned_to = $3,
				assigned_at = now(),
				escalated_to = $3,
				escalated_at = now(),
				updated_at = now()
			WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
			tenantID, alertID, escalatedTo,
		)
		if err != nil {
			return fmt.Errorf("escalate alert: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, tenantID, alertID)
}

// InsertTimeline adds an immutable alert timeline entry.
func (r *AlertRepository) InsertTimeline(ctx context.Context, entry *model.AlertTimelineEntry) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	return runWithTenantWrite(ctx, r.db, entry.TenantID, func(db dbtx) error {
		_, err := db.Exec(ctx, `
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
	})
}

// ListTimeline returns alert timeline entries in chronological order.
func (r *AlertRepository) ListTimeline(ctx context.Context, tenantID, alertID uuid.UUID) ([]*model.AlertTimelineEntry, error) {
	items := make([]*model.AlertTimelineEntry, 0)
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, `
			SELECT
				id, tenant_id, alert_id, action, actor_id, actor_name,
				old_value, new_value, description, metadata, created_at
			FROM alert_timeline
			WHERE tenant_id = $1 AND alert_id = $2
			ORDER BY created_at ASC`,
			tenantID, alertID,
		)
		if err != nil {
			return fmt.Errorf("list alert timeline: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			item, err := scanAlertTimeline(rows)
			if err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

// FindOpenByRuleAndAsset finds an existing open alert for deduplication.
func (r *AlertRepository) FindOpenByRuleAndAsset(ctx context.Context, tenantID, ruleID uuid.UUID, assetID *uuid.UUID) (*model.Alert, error) {
	if assetID == nil {
		return nil, ErrNotFound
	}
	var alert *model.Alert
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		row := db.QueryRow(ctx, `
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
		item, err := scanAlert(row)
		if err != nil {
			if err == pgx.ErrNoRows {
				return ErrNotFound
			}
			return fmt.Errorf("find open alert by rule and asset: %w", err)
		}
		alert = item
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := r.enrichAlerts(ctx, tenantID, []*model.Alert{alert}); err != nil {
		return nil, err
	}
	return alert, nil
}

// UpdateAggregatedDetectionAlert updates an existing deduplicated alert with more events.
func (r *AlertRepository) UpdateAggregatedDetectionAlert(ctx context.Context, tenantID, alertID uuid.UUID, additionalEvents int, lastEventAt time.Time, assetIDs []uuid.UUID, explanation *model.AlertExplanation) (*model.Alert, error) {
	explanationJSON, err := marshalJSON(explanation)
	if err != nil {
		return nil, fmt.Errorf("marshal updated explanation: %w", err)
	}
	if err := runWithTenantWrite(ctx, r.db, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `
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
			return fmt.Errorf("update aggregated alert: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, tenantID, alertID)
}

// FindRelated returns alerts related by asset, rule, or MITRE technique.
func (r *AlertRepository) FindRelated(ctx context.Context, tenantID, alertID uuid.UUID) ([]*model.Alert, error) {
	alert, err := r.GetByID(ctx, tenantID, alertID)
	if err != nil {
		return nil, err
	}
	indicatorValues := indicatorMatchValues(alert.Explanation.IndicatorMatches)
	results := make([]*model.Alert, 0)
	err = runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, `
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
				(asset_id IS NOT NULL AND $3::uuid IS NOT NULL AND asset_id = $3::uuid) OR
				(rule_id IS NOT NULL AND $4::uuid IS NOT NULL AND rule_id = $4::uuid) OR
				(mitre_technique_id IS NOT NULL AND $5::text IS NOT NULL AND mitre_technique_id = $5::text) OR
				(cardinality($6::text[]) > 0 AND EXISTS (
					SELECT 1
					FROM jsonb_array_elements(COALESCE(explanation->'indicator_matches', '[]'::jsonb)) AS elem
					WHERE LOWER(elem->>'value') = ANY($6)
				))
			  )
			ORDER BY created_at DESC
			LIMIT 25`,
			tenantID, alertID, alert.AssetID, alert.RuleID, alert.MITRETechniqueID, indicatorValues,
		)
		if err != nil {
			return fmt.Errorf("find related alerts: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			related, err := scanAlert(rows)
			if err != nil {
				return err
			}
			results = append(results, related)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	if err := r.enrichAlerts(ctx, tenantID, results); err != nil {
		return nil, err
	}
	return results, nil
}

// FindByThreatContext returns alerts correlated to a threat's indicators or ATT&CK techniques.
func (r *AlertRepository) FindByThreatContext(ctx context.Context, tenantID uuid.UUID, indicatorValues, techniqueIDs []string, limit int) ([]*model.Alert, error) {
	if len(indicatorValues) == 0 && len(techniqueIDs) == 0 {
		return []*model.Alert{}, nil
	}
	if limit <= 0 {
		limit = 25
	}

	items := make([]*model.Alert, 0)
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, `
			SELECT DISTINCT
				a.id, a.tenant_id, a.title, a.description, a.severity, a.status,
				a.source, a.rule_id, a.asset_id, a.asset_ids, a.assigned_to, a.assigned_at,
				a.escalated_to, a.escalated_at, a.explanation, a.confidence_score,
				a.mitre_tactic_id, a.mitre_tactic_name, a.mitre_technique_id, a.mitre_technique_name,
				a.event_count, a.first_event_at, a.last_event_at, a.resolved_at,
				a.resolution_notes, a.false_positive_reason, a.tags, a.metadata,
				a.created_at, a.updated_at, a.deleted_at
			FROM alerts a
			WHERE a.tenant_id = $1
			  AND a.deleted_at IS NULL
			  AND (
				(cardinality($2::text[]) > 0 AND a.mitre_technique_id = ANY($2)) OR
				(cardinality($3::text[]) > 0 AND EXISTS (
					SELECT 1
					FROM jsonb_array_elements(COALESCE(a.explanation->'indicator_matches', '[]'::jsonb)) AS elem
					WHERE LOWER(elem->>'value') = ANY($3)
				))
			  )
			ORDER BY a.created_at DESC
			LIMIT $4`,
			tenantID,
			techniqueIDs,
			indicatorValues,
			limit,
		)
		if err != nil {
			return fmt.Errorf("find alerts by threat context: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			item, err := scanAlert(rows)
			if err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	if err := r.enrichAlerts(ctx, tenantID, items); err != nil {
		return nil, err
	}
	return items, nil
}

// FindRecentOpenBySourceAndMetadataValue finds an open event-driven alert by source
// and a specific metadata field value within the provided time window.
func (r *AlertRepository) FindRecentOpenBySourceAndMetadataValue(ctx context.Context, tenantID uuid.UUID, source, metadataKey, metadataValue string, since time.Time) (*model.Alert, error) {
	var alert *model.Alert
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		row := db.QueryRow(ctx, `
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
			  AND source = $2
			  AND status IN ('new', 'acknowledged', 'investigating', 'in_progress', 'escalated')
			  AND deleted_at IS NULL
			  AND metadata ->> $3 = $4
			  AND last_event_at >= $5
			ORDER BY last_event_at DESC
			LIMIT 1`,
			tenantID, source, metadataKey, metadataValue, since,
		)
		item, err := scanAlert(row)
		if err != nil {
			if err == pgx.ErrNoRows {
				return ErrNotFound
			}
			return fmt.Errorf("find recent open alert by metadata: %w", err)
		}
		alert = item
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := r.enrichAlerts(ctx, tenantID, []*model.Alert{alert}); err != nil {
		return nil, err
	}
	return alert, nil
}

// UpdateEventAlert updates a custom event-driven alert in place.
func (r *AlertRepository) UpdateEventAlert(ctx context.Context, alert *model.Alert) (*model.Alert, error) {
	if alert == nil {
		return nil, ErrInvalidInput
	}

	explanation, err := marshalJSON(alert.Explanation)
	if err != nil {
		return nil, fmt.Errorf("marshal updated event alert explanation: %w", err)
	}

	if err := runWithTenantWrite(ctx, r.db, alert.TenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `
			UPDATE alerts
			SET
				title = $3,
				description = $4,
				severity = $5,
				explanation = $6,
				confidence_score = $7,
				event_count = $8,
				last_event_at = $9,
				metadata = $10,
				updated_at = now()
			WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
			alert.TenantID,
			alert.ID,
			alert.Title,
			alert.Description,
			alert.Severity,
			explanation,
			alert.ConfidenceScore,
			alert.EventCount,
			alert.LastEventAt,
			ensureRawMessage(alert.Metadata, "{}"),
		)
		if err != nil {
			return fmt.Errorf("update event alert: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, alert.TenantID, alert.ID)
}

// CloneTimeline copies timeline entries from one alert to another for merge operations.
func (r *AlertRepository) CloneTimeline(ctx context.Context, tenantID, fromAlertID, toAlertID uuid.UUID) error {
	return runWithTenantWrite(ctx, r.db, tenantID, func(db dbtx) error {
		_, err := db.Exec(ctx, `
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
	})
}

// MarkMerged marks a secondary alert as merged into a primary alert.
func (r *AlertRepository) MarkMerged(ctx context.Context, tenantID, alertID, primaryAlertID uuid.UUID) error {
	return runWithTenantWrite(ctx, r.db, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `
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
	})
}

// UpdateAfterMerge updates the primary alert after a merge operation.
func (r *AlertRepository) UpdateAfterMerge(ctx context.Context, tenantID, alertID uuid.UUID, eventCount int, assetID *uuid.UUID, assetIDs []uuid.UUID, explanation *model.AlertExplanation) (*model.Alert, error) {
	payload, err := marshalJSON(explanation)
	if err != nil {
		return nil, fmt.Errorf("marshal merged explanation: %w", err)
	}
	if err := runWithTenantWrite(ctx, r.db, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `
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
			return fmt.Errorf("update merged alert: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, tenantID, alertID)
}

// Stats returns aggregated alert statistics.
func (r *AlertRepository) Stats(ctx context.Context, tenantID uuid.UUID) (*model.AlertStats, error) {
	stats := &model.AlertStats{}
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		if err := db.QueryRow(ctx, `
			SELECT
				COUNT(*) FILTER (WHERE deleted_at IS NULL),
				COUNT(*) FILTER (WHERE deleted_at IS NULL AND status IN ('new', 'acknowledged', 'investigating', 'in_progress', 'escalated')),
				COUNT(*) FILTER (WHERE deleted_at IS NULL AND status IN ('resolved', 'closed'))
			FROM alerts
			WHERE tenant_id = $1`,
			tenantID,
		).Scan(&stats.Total, &stats.OpenCount, &stats.ResolvedCount); err != nil {
			return fmt.Errorf("alert totals: %w", err)
		}

		buildCounts := func(sql string) ([]model.NamedCount, error) {
			rows, err := db.Query(ctx, sql, tenantID)
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
			return fmt.Errorf("stats by severity: %w", err)
		}
		if stats.ByStatus, err = buildCounts(`
			SELECT status::text, COUNT(*)
			FROM alerts
			WHERE tenant_id = $1 AND deleted_at IS NULL
			GROUP BY status
			ORDER BY COUNT(*) DESC, status ASC`); err != nil {
			return fmt.Errorf("stats by status: %w", err)
		}
		if stats.ByRule, err = buildCounts(`
			SELECT COALESCE(source, 'unknown'), COUNT(*)
			FROM alerts
			WHERE tenant_id = $1 AND deleted_at IS NULL
			GROUP BY source
			ORDER BY COUNT(*) DESC, source ASC`); err != nil {
			return fmt.Errorf("stats by rule: %w", err)
		}
		if stats.ByRuleType, err = buildCounts(`
			SELECT COALESCE(dr.rule_type::text, 'unknown'), COUNT(*)
			FROM alerts a
			LEFT JOIN detection_rules dr ON dr.tenant_id = a.tenant_id AND dr.id = a.rule_id
			WHERE a.tenant_id = $1 AND a.deleted_at IS NULL
			GROUP BY COALESCE(dr.rule_type::text, 'unknown')
			ORDER BY COUNT(*) DESC, COALESCE(dr.rule_type::text, 'unknown') ASC`); err != nil {
			return fmt.Errorf("stats by rule type: %w", err)
		}
		if stats.ByTechnique, err = buildCounts(`
			SELECT COALESCE(mitre_technique_id, 'unmapped'), COUNT(*)
			FROM alerts
			WHERE tenant_id = $1 AND deleted_at IS NULL
			GROUP BY mitre_technique_id
			ORDER BY COUNT(*) DESC, mitre_technique_id ASC`); err != nil {
			return fmt.Errorf("stats by technique: %w", err)
		}
		if err := db.QueryRow(ctx, `
			SELECT COALESCE(
				AVG(EXTRACT(EPOCH FROM (resolved_at - created_at)) / 3600.0),
				0
			)
			FROM alerts
			WHERE tenant_id = $1
			  AND deleted_at IS NULL
			  AND status IN ('resolved', 'closed')
			  AND resolved_at IS NOT NULL`,
			tenantID,
		).Scan(&stats.MTTRHours); err != nil {
			return fmt.Errorf("stats mttr: %w", err)
		}
		if err := db.QueryRow(ctx, `
			SELECT COALESCE(
				AVG(EXTRACT(EPOCH FROM (timeline.ack_at - a.created_at)) / 3600.0),
				0
			)
			FROM alerts a
			JOIN LATERAL (
				SELECT MIN(created_at) AS ack_at
				FROM alert_timeline t
				WHERE t.tenant_id = a.tenant_id
				  AND t.alert_id = a.id
				  AND t.action = 'status_changed'
				  AND t.new_value IN ('acknowledged', 'investigating', 'in_progress', 'resolved', 'closed', 'false_positive', 'escalated')
			) AS timeline ON timeline.ack_at IS NOT NULL
			WHERE a.tenant_id = $1
			  AND a.deleted_at IS NULL`,
			tenantID,
		).Scan(&stats.MTTAHours); err != nil {
			return fmt.Errorf("stats mtta: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if stats.Total > 0 {
		falsePositiveCount := 0
		for _, item := range stats.ByStatus {
			if item.Name == string(model.AlertStatusFalsePositive) {
				falsePositiveCount = item.Count
				break
			}
		}
		stats.FalsePositiveRate = float64(falsePositiveCount) / float64(stats.Total)
	}
	return stats, nil
}

func (r *AlertRepository) enrichAlerts(ctx context.Context, tenantID uuid.UUID, alerts []*model.Alert) error {
	if len(alerts) == 0 {
		return nil
	}

	type assetRecord struct {
		Name        *string
		IPAddress   *string
		Hostname    *string
		OS          *string
		Owner       *string
		Criticality *model.Criticality
	}
	type ruleRecord struct {
		Name     *string
		RuleType *model.DetectionRuleType
	}
	type userRecord struct {
		Name  *string
		Email *string
	}

	assetIDs := make([]uuid.UUID, 0, len(alerts))
	ruleIDs := make([]uuid.UUID, 0, len(alerts))
	userIDs := make([]uuid.UUID, 0, len(alerts)*2)
	for _, alert := range alerts {
		if alert == nil {
			continue
		}
		if alert.AssetID != nil {
			assetIDs = append(assetIDs, *alert.AssetID)
		}
		if alert.RuleID != nil {
			ruleIDs = append(ruleIDs, *alert.RuleID)
		}
		if alert.AssignedTo != nil {
			userIDs = append(userIDs, *alert.AssignedTo)
		}
		if alert.EscalatedTo != nil {
			userIDs = append(userIDs, *alert.EscalatedTo)
		}
	}

	assetMap := make(map[uuid.UUID]assetRecord)
	ruleMap := make(map[uuid.UUID]ruleRecord)
	userMap := make(map[uuid.UUID]userRecord)

	if err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		if ids := uniqueUUIDs(assetIDs); len(ids) > 0 {
			rows, err := db.Query(ctx, `
				SELECT id, name, host(ip_address), hostname, os, owner, criticality
				FROM assets
				WHERE tenant_id = $1 AND id = ANY($2) AND deleted_at IS NULL`,
				tenantID, ids,
			)
			if err != nil {
				return fmt.Errorf("enrich alert assets: %w", err)
			}
			defer rows.Close()
			for rows.Next() {
				var (
					id          uuid.UUID
					name        *string
					ipAddress   *string
					hostname    *string
					osValue     *string
					owner       *string
					criticality *model.Criticality
				)
				if err := rows.Scan(&id, &name, &ipAddress, &hostname, &osValue, &owner, &criticality); err != nil {
					return fmt.Errorf("scan alert asset enrichment: %w", err)
				}
				assetMap[id] = assetRecord{
					Name:        name,
					IPAddress:   ipAddress,
					Hostname:    hostname,
					OS:          osValue,
					Owner:       owner,
					Criticality: criticality,
				}
			}
			if err := rows.Err(); err != nil {
				return fmt.Errorf("iterate alert asset enrichment: %w", err)
			}
		}

		if ids := uniqueUUIDs(ruleIDs); len(ids) > 0 {
			rows, err := db.Query(ctx, `
				SELECT id, name, rule_type
				FROM detection_rules
				WHERE tenant_id = $1 AND id = ANY($2) AND deleted_at IS NULL`,
				tenantID, ids,
			)
			if err != nil {
				return fmt.Errorf("enrich alert rules: %w", err)
			}
			defer rows.Close()
			for rows.Next() {
				var (
					id       uuid.UUID
					name     *string
					ruleType *model.DetectionRuleType
				)
				if err := rows.Scan(&id, &name, &ruleType); err != nil {
					return fmt.Errorf("scan alert rule enrichment: %w", err)
				}
				ruleMap[id] = ruleRecord{Name: name, RuleType: ruleType}
			}
			if err := rows.Err(); err != nil {
				return fmt.Errorf("iterate alert rule enrichment: %w", err)
			}
		}

		if ids := uniqueUUIDs(userIDs); len(ids) > 0 {
			rows, err := db.Query(ctx, `
				SELECT id, first_name, last_name, email
				FROM users
				WHERE tenant_id = $1 AND id = ANY($2) AND deleted_at IS NULL`,
				tenantID, ids,
			)
			if err != nil {
				return fmt.Errorf("enrich alert users: %w", err)
			}
			defer rows.Close()
			for rows.Next() {
				var (
					id        uuid.UUID
					firstName string
					lastName  string
					email     string
				)
				if err := rows.Scan(&id, &firstName, &lastName, &email); err != nil {
					return fmt.Errorf("scan alert user enrichment: %w", err)
				}
				displayName := strings.TrimSpace(strings.TrimSpace(firstName + " " + lastName))
				if displayName == "" {
					displayName = email
				}
				userMap[id] = userRecord{
					Name:  stringPtr(displayName),
					Email: stringPtr(email),
				}
			}
			if err := rows.Err(); err != nil {
				return fmt.Errorf("iterate alert user enrichment: %w", err)
			}
		}
		return nil
	}); err != nil {
		return err
	}

	for _, alert := range alerts {
		if alert == nil {
			continue
		}
		if alert.AssetID != nil {
			if asset, ok := assetMap[*alert.AssetID]; ok {
				alert.AssetName = asset.Name
				alert.AssetIPAddress = asset.IPAddress
				alert.AssetHostname = asset.Hostname
				alert.AssetOS = asset.OS
				alert.AssetOwner = asset.Owner
				alert.AssetCriticality = asset.Criticality
			}
		}
		if alert.RuleID != nil {
			if rule, ok := ruleMap[*alert.RuleID]; ok {
				alert.RuleName = rule.Name
				alert.RuleType = rule.RuleType
			}
		}
		if alert.RuleName == nil && strings.TrimSpace(alert.Source) != "" {
			alert.RuleName = stringPtr(alert.Source)
		}
		if alert.AssignedTo != nil {
			if user, ok := userMap[*alert.AssignedTo]; ok {
				alert.AssignedToName = user.Name
				alert.AssignedToEmail = user.Email
			}
		}
		if alert.EscalatedTo != nil {
			if user, ok := userMap[*alert.EscalatedTo]; ok {
				alert.EscalatedToName = user.Name
				alert.EscalatedToEmail = user.Email
			}
		}
	}

	return nil
}

func indicatorMatchValues(matches []model.IndicatorEvidence) []string {
	values := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		value := strings.ToLower(strings.TrimSpace(match.Value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}

func stringPtr(value string) *string {
	return &value
}
