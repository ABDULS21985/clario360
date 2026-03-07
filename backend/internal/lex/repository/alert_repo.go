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

	"github.com/clario360/platform/internal/lex/model"
)

type AlertRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewAlertRepository(db *pgxpool.Pool, logger zerolog.Logger) *AlertRepository {
	return &AlertRepository{db: db, logger: logger}
}

func (r *AlertRepository) Create(ctx context.Context, q Queryer, alert *model.ComplianceAlert) error {
	query := `
		INSERT INTO compliance_alerts (
			id, tenant_id, rule_id, contract_id, title, description, severity, status,
			resolved_by, resolved_at, resolution_notes, dedup_key, evidence
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING created_at, updated_at`
	return q.QueryRow(ctx, query,
		alert.ID, alert.TenantID, alert.RuleID, alert.ContractID, alert.Title, alert.Description, alert.Severity, alert.Status,
		alert.ResolvedBy, alert.ResolvedAt, alert.ResolutionNotes, alert.DedupKey, alert.Evidence,
	).Scan(&alert.CreatedAt, &alert.UpdatedAt)
}

func (r *AlertRepository) CreateOrSkipDedup(ctx context.Context, q Queryer, alert *model.ComplianceAlert) (bool, error) {
	if alert.DedupKey == nil || strings.TrimSpace(*alert.DedupKey) == "" {
		if err := r.Create(ctx, q, alert); err != nil {
			return false, err
		}
		return true, nil
	}
	query := `
		INSERT INTO compliance_alerts (
			id, tenant_id, rule_id, contract_id, title, description, severity, status,
			resolved_by, resolved_at, resolution_notes, dedup_key, evidence
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT DO NOTHING
		RETURNING created_at, updated_at`
	err := q.QueryRow(ctx, query,
		alert.ID, alert.TenantID, alert.RuleID, alert.ContractID, alert.Title, alert.Description, alert.Severity, alert.Status,
		alert.ResolvedBy, alert.ResolvedAt, alert.ResolutionNotes, alert.DedupKey, alert.Evidence,
	).Scan(&alert.CreatedAt, &alert.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *AlertRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.ComplianceAlert, error) {
	query := alertJSONSelect(`tenant_id = $1 AND id = $2`)
	return queryRowJSON[model.ComplianceAlert](ctx, r.db, query, tenantID, id)
}

func (r *AlertRepository) List(ctx context.Context, tenantID uuid.UUID, status string, severity string, page, perPage int) ([]model.ComplianceAlert, int, error) {
	args := []any{tenantID}
	arg := 2
	conditions := []string{"tenant_id = $1"}
	if status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", arg))
		args = append(args, status)
		arg++
	}
	if severity != "" {
		conditions = append(conditions, fmt.Sprintf("severity = $%d", arg))
		args = append(args, severity)
		arg++
	}
	where := strings.Join(conditions, " AND ")
	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM compliance_alerts WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []model.ComplianceAlert{}, 0, nil
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 25
	}
	query := alertJSONSelect(where) + fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", arg, arg+1)
	args = append(args, perPage, (page-1)*perPage)
	items, err := queryListJSON[model.ComplianceAlert](ctx, r.db, query, args...)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *AlertRepository) UpdateStatus(ctx context.Context, q Queryer, tenantID, id uuid.UUID, status model.ComplianceAlertStatus, resolvedBy *uuid.UUID, resolvedAt *time.Time, notes string) error {
	ct, err := q.Exec(ctx, `
		UPDATE compliance_alerts
		SET status = $3,
		    resolved_by = $4,
		    resolved_at = $5,
		    resolution_notes = $6,
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id, status, resolvedBy, resolvedAt, notes,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *AlertRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID) (map[string]int, error) {
	rows, err := r.db.Query(ctx, `SELECT status, COUNT(*) FROM compliance_alerts WHERE tenant_id = $1 GROUP BY status`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			return nil, err
		}
		out[key] = count
	}
	return out, rows.Err()
}

func (r *AlertRepository) CountBySeverity(ctx context.Context, tenantID uuid.UUID) (map[string]int, error) {
	rows, err := r.db.Query(ctx, `SELECT severity, COUNT(*) FROM compliance_alerts WHERE tenant_id = $1 GROUP BY severity`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			return nil, err
		}
		out[key] = count
	}
	return out, rows.Err()
}

func (r *AlertRepository) OpenCount(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM compliance_alerts WHERE tenant_id = $1 AND status IN ('open','acknowledged','investigating')`, tenantID).Scan(&count)
	return count, err
}

func (r *AlertRepository) ScoreComponents(ctx context.Context, tenantID uuid.UUID) (openAlerts int, resolvedAlerts int, err error) {
	if err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM compliance_alerts WHERE tenant_id = $1 AND status IN ('open','acknowledged','investigating')`, tenantID).Scan(&openAlerts); err != nil {
		return
	}
	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM compliance_alerts WHERE tenant_id = $1 AND status = 'resolved'`, tenantID).Scan(&resolvedAlerts)
	return
}

func alertJSONSelect(where string) string {
	return `
		SELECT row_to_json(t)
		FROM (
			SELECT id, tenant_id, rule_id, contract_id, title, description, severity, status,
			       resolved_by, resolved_at, resolution_notes, dedup_key, COALESCE(evidence, '{}'::jsonb) AS evidence,
			       created_at, updated_at
			FROM compliance_alerts
			WHERE ` + where + `
		) t`
}
