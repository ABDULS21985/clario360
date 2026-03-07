package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/visus/model"
)

type AlertListFilters struct {
	Status       []string
	Severity     []string
	Category     []string
	SourceSuites []string
	Search       string
}

type AlertRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewAlertRepository(db *pgxpool.Pool, logger zerolog.Logger) *AlertRepository {
	return &AlertRepository{db: db, logger: logger.With().Str("repo", "visus_alerts").Logger()}
}

func (r *AlertRepository) Create(ctx context.Context, item *model.ExecutiveAlert) (*model.ExecutiveAlert, error) {
	if item == nil {
		return nil, ErrValidation
	}
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	updatedAt := item.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO visus_executive_alerts (
			tenant_id, title, description, category, severity, source_suite, source_type, source_entity_id,
			source_event_type, status, viewed_at, viewed_by, actioned_at, actioned_by, action_notes, dismissed_at,
			dismissed_by, dismiss_reason, dedup_key, occurrence_count, first_seen_at, last_seen_at, linked_kpi_id,
			linked_dashboard_id, metadata, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,
			$9,$10,$11,$12,$13,$14,$15,$16,
			$17,$18,$19,$20,$21,$22,$23,
			$24,$25,$26,$27
		)
		RETURNING id`,
		item.TenantID, item.Title, item.Description, item.Category, item.Severity, item.SourceSuite, item.SourceType, item.SourceEntityID,
		item.SourceEventType, item.Status, item.ViewedAt, item.ViewedBy, item.ActionedAt, item.ActionedBy, item.ActionNotes, item.DismissedAt,
		item.DismissedBy, item.DismissReason, item.DedupKey, item.OccurrenceCount, item.FirstSeenAt, item.LastSeenAt, item.LinkedKPIID,
		item.LinkedDashboardID, marshalJSON(item.Metadata), createdAt, updatedAt,
	).Scan(&id)
	if err != nil {
		return nil, wrapErr("create alert", err)
	}
	return r.Get(ctx, item.TenantID, id)
}

func (r *AlertRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.ExecutiveAlert, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, title, description, category, severity, source_suite, source_type, source_entity_id,
		       source_event_type, status, viewed_at, viewed_by, actioned_at, actioned_by, action_notes, dismissed_at,
		       dismissed_by, dismiss_reason, dedup_key, occurrence_count, first_seen_at, last_seen_at, linked_kpi_id,
		       linked_dashboard_id, metadata, created_at, updated_at
		FROM visus_executive_alerts
		WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	item, err := scanAlert(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return item, nil
}

func (r *AlertRepository) List(ctx context.Context, tenantID uuid.UUID, filters AlertListFilters, page, perPage int) ([]model.ExecutiveAlert, int, error) {
	meta := normalizePagination(page, perPage)
	conditions := []string{"tenant_id = $1"}
	args := []any{tenantID}
	next := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if len(filters.Status) > 0 {
		conditions = append(conditions, "status = ANY("+next(filters.Status)+")")
	}
	if len(filters.Severity) > 0 {
		conditions = append(conditions, "severity = ANY("+next(filters.Severity)+")")
	}
	if len(filters.Category) > 0 {
		conditions = append(conditions, "category = ANY("+next(filters.Category)+")")
	}
	if len(filters.SourceSuites) > 0 {
		conditions = append(conditions, "source_suite = ANY("+next(filters.SourceSuites)+")")
	}
	if trimmed := strings.TrimSpace(filters.Search); trimmed != "" {
		placeholder := next(likePattern(trimmed))
		conditions = append(conditions, "(title ILIKE "+placeholder+" OR description ILIKE "+placeholder+")")
	}
	where := strings.Join(conditions, " AND ")
	listQuery := fmt.Sprintf(`
		SELECT id, tenant_id, title, description, category, severity, source_suite, source_type, source_entity_id,
		       source_event_type, status, viewed_at, viewed_by, actioned_at, actioned_by, action_notes, dismissed_at,
		       dismissed_by, dismiss_reason, dedup_key, occurrence_count, first_seen_at, last_seen_at, linked_kpi_id,
		       linked_dashboard_id, metadata, created_at, updated_at
		FROM visus_executive_alerts
		WHERE %s
		ORDER BY created_at DESC
		LIMIT %s OFFSET %s`, where, next(meta.Limit), next(meta.Offset))
	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, wrapErr("list alerts", err)
	}
	defer rows.Close()
	items := make([]model.ExecutiveAlert, 0, meta.Limit)
	for rows.Next() {
		item, err := scanAlert(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, wrapErr("iterate alerts", err)
	}

	countArgs := args[:len(args)-2]
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM visus_executive_alerts WHERE %s`, where)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, wrapErr("count alerts", err)
	}
	return items, total, nil
}

func (r *AlertRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status model.AlertStatus, actorID *uuid.UUID, notes, dismissReason *string) (*model.ExecutiveAlert, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE visus_executive_alerts
		SET status = $3,
		    viewed_at = CASE WHEN $3 = 'viewed' THEN now() ELSE viewed_at END,
		    viewed_by = CASE WHEN $3 = 'viewed' THEN $4 ELSE viewed_by END,
		    actioned_at = CASE WHEN $3 = 'actioned' THEN now() ELSE actioned_at END,
		    actioned_by = CASE WHEN $3 = 'actioned' THEN $4 ELSE actioned_by END,
		    action_notes = CASE WHEN $3 = 'actioned' THEN $5 ELSE action_notes END,
		    dismissed_at = CASE WHEN $3 = 'dismissed' THEN now() ELSE dismissed_at END,
		    dismissed_by = CASE WHEN $3 = 'dismissed' THEN $4 ELSE dismissed_by END,
		    dismiss_reason = CASE WHEN $3 = 'dismissed' THEN $6 ELSE dismiss_reason END,
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id, status, actorID, notes, dismissReason,
	)
	if err != nil {
		return nil, wrapErr("update alert status", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.Get(ctx, tenantID, id)
}

func (r *AlertRepository) CountUnactioned(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM visus_executive_alerts
		WHERE tenant_id = $1
		  AND status IN ('new','viewed','acknowledged','escalated')`, tenantID).Scan(&count); err != nil {
		return 0, wrapErr("count unactioned alerts", err)
	}
	return count, nil
}

func (r *AlertRepository) Stats(ctx context.Context, tenantID uuid.UUID) (*model.AlertStats, error) {
	stats := &model.AlertStats{
		ByCategory: map[string]int{},
		BySeverity: map[string]int{},
		ByStatus:   map[string]int{},
	}
	rows, err := r.db.Query(ctx, `
		SELECT category, severity, status, COUNT(*)
		FROM visus_executive_alerts
		WHERE tenant_id = $1
		GROUP BY category, severity, status`, tenantID)
	if err != nil {
		return nil, wrapErr("alert stats", err)
	}
	defer rows.Close()
	for rows.Next() {
		var category, severity, status string
		var count int
		if err := rows.Scan(&category, &severity, &status, &count); err != nil {
			return nil, wrapErr("scan alert stats", err)
		}
		stats.ByCategory[category] += count
		stats.BySeverity[severity] += count
		stats.ByStatus[status] += count
		stats.Total += count
	}
	return stats, rows.Err()
}

func (r *AlertRepository) FindDedupMatch(ctx context.Context, tenantID uuid.UUID, dedupKey string, window time.Duration) (*model.ExecutiveAlert, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, title, description, category, severity, source_suite, source_type, source_entity_id,
		       source_event_type, status, viewed_at, viewed_by, actioned_at, actioned_by, action_notes, dismissed_at,
		       dismissed_by, dismiss_reason, dedup_key, occurrence_count, first_seen_at, last_seen_at, linked_kpi_id,
		       linked_dashboard_id, metadata, created_at, updated_at
		FROM visus_executive_alerts
		WHERE tenant_id = $1
		  AND dedup_key = $2
		  AND status IN ('new','viewed','acknowledged')
		  AND last_seen_at > now() - $3::interval
		ORDER BY last_seen_at DESC
		LIMIT 1`, tenantID, dedupKey, intervalLiteral(window))
	item, err := scanAlert(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return item, nil
}

func (r *AlertRepository) IncrementOccurrence(ctx context.Context, tenantID, id uuid.UUID) (*model.ExecutiveAlert, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE visus_executive_alerts
		SET occurrence_count = occurrence_count + 1,
		    last_seen_at = now(),
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return nil, wrapErr("increment alert occurrence", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.Get(ctx, tenantID, id)
}

func (r *AlertRepository) ListRecentBySource(ctx context.Context, tenantID uuid.UUID, sourceSuite string, since time.Time, severity *string) ([]model.ExecutiveAlert, error) {
	query := `
		SELECT id, tenant_id, title, description, category, severity, source_suite, source_type, source_entity_id,
		       source_event_type, status, viewed_at, viewed_by, actioned_at, actioned_by, action_notes, dismissed_at,
		       dismissed_by, dismiss_reason, dedup_key, occurrence_count, first_seen_at, last_seen_at, linked_kpi_id,
		       linked_dashboard_id, metadata, created_at, updated_at
		FROM visus_executive_alerts
		WHERE tenant_id = $1 AND source_suite = $2 AND created_at >= $3`
	args := []any{tenantID, sourceSuite, since}
	if severity != nil {
		query += ` AND severity = $4`
		args = append(args, *severity)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, wrapErr("list recent alerts by source", err)
	}
	defer rows.Close()
	out := make([]model.ExecutiveAlert, 0)
	for rows.Next() {
		item, err := scanAlert(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (r *AlertRepository) CountCriticalSuites(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT source_suite
		FROM visus_executive_alerts
		WHERE tenant_id = $1
		  AND severity = 'critical'
		  AND status IN ('new','viewed','acknowledged','escalated')`, tenantID)
	if err != nil {
		return nil, wrapErr("count critical alert suites", err)
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var suite string
		if err := rows.Scan(&suite); err != nil {
			return nil, wrapErr("scan alert suite", err)
		}
		out = append(out, suite)
	}
	return out, rows.Err()
}

func scanAlert(row interface{ Scan(...any) error }) (*model.ExecutiveAlert, error) {
	item := &model.ExecutiveAlert{}
	var metadata []byte
	var category, severity, status string
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.Title,
		&item.Description,
		&category,
		&severity,
		&item.SourceSuite,
		&item.SourceType,
		&item.SourceEntityID,
		&item.SourceEventType,
		&status,
		&item.ViewedAt,
		&item.ViewedBy,
		&item.ActionedAt,
		&item.ActionedBy,
		&item.ActionNotes,
		&item.DismissedAt,
		&item.DismissedBy,
		&item.DismissReason,
		&item.DedupKey,
		&item.OccurrenceCount,
		&item.FirstSeenAt,
		&item.LastSeenAt,
		&item.LinkedKPIID,
		&item.LinkedDashboardID,
		&metadata,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, wrapErr("scan alert", err)
	}
	item.Category = model.AlertCategory(category)
	item.Severity = model.AlertSeverity(severity)
	item.Status = model.AlertStatus(status)
	item.Metadata = unmarshalMap(metadata)
	return item, nil
}

func intervalLiteral(duration time.Duration) string {
	seconds := int(duration.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	return fmt.Sprintf("%d seconds", seconds)
}
