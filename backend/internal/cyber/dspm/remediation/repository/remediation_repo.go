package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/dto"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// RemediationRepository handles persistence for DSPM remediation work items.
type RemediationRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewRemediationRepository creates a new RemediationRepository.
func NewRemediationRepository(db *pgxpool.Pool, logger zerolog.Logger) *RemediationRepository {
	return &RemediationRepository{db: db, logger: logger}
}

// Create inserts a new remediation record and returns it with server-generated fields.
func (r *RemediationRepository) Create(ctx context.Context, remediation *model.Remediation) (*model.Remediation, error) {
	if remediation.ID == uuid.Nil {
		remediation.ID = uuid.New()
	}
	if len(remediation.Steps) == 0 {
		remediation.Steps = json.RawMessage("[]")
	}
	if len(remediation.ComplianceTags) == 0 {
		remediation.ComplianceTags = json.RawMessage("[]")
	}
	if len(remediation.PreActionState) == 0 {
		remediation.PreActionState = json.RawMessage("{}")
	}

	row := r.db.QueryRow(ctx, `
		INSERT INTO dspm_remediations (
			id, tenant_id, finding_type, finding_id, data_asset_id, data_asset_name,
			identity_id, playbook_id, title, description, severity, steps,
			current_step, total_steps, assigned_to, assigned_team, sla_due_at,
			sla_breached, risk_score_before, risk_score_after, risk_reduction,
			pre_action_state, rollback_available, rolled_back, status,
			cyber_alert_id, created_by, compliance_tags,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17,
			$18, $19, $20, $21,
			$22, $23, $24, $25,
			$26, $27, $28,
			now(), now()
		)
		RETURNING id, tenant_id, finding_type, finding_id, data_asset_id, data_asset_name,
			identity_id, playbook_id, title, description, severity, steps,
			current_step, total_steps, assigned_to, assigned_team, sla_due_at,
			sla_breached, risk_score_before, risk_score_after, risk_reduction,
			pre_action_state, rollback_available, rolled_back, status,
			cyber_alert_id, created_by, created_at, updated_at, completed_at, compliance_tags`,
		remediation.ID, remediation.TenantID, remediation.FindingType, remediation.FindingID,
		remediation.DataAssetID, remediation.DataAssetName,
		remediation.IdentityID, remediation.PlaybookID, remediation.Title, remediation.Description,
		remediation.Severity, remediation.Steps,
		remediation.CurrentStep, remediation.TotalSteps, remediation.AssignedTo, remediation.AssignedTeam,
		remediation.SLADueAt,
		remediation.SLABreached, remediation.RiskScoreBefore, remediation.RiskScoreAfter,
		remediation.RiskReduction,
		remediation.PreActionState, remediation.RollbackAvailable, remediation.RolledBack,
		remediation.Status,
		remediation.CyberAlertID, remediation.CreatedBy, remediation.ComplianceTags,
	)

	result, err := scanRemediation(row)
	if err != nil {
		return nil, fmt.Errorf("create remediation: %w", err)
	}
	return result, nil
}

// GetByID fetches a single remediation by ID with tenant isolation.
func (r *RemediationRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*model.Remediation, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, finding_type, finding_id, data_asset_id, data_asset_name,
			identity_id, playbook_id, title, description, severity, steps,
			current_step, total_steps, assigned_to, assigned_team, sla_due_at,
			sla_breached, risk_score_before, risk_score_after, risk_reduction,
			pre_action_state, rollback_available, rolled_back, status,
			cyber_alert_id, created_by, created_at, updated_at, completed_at, compliance_tags
		FROM dspm_remediations
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)

	result, err := scanRemediation(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("remediation not found")
		}
		return nil, fmt.Errorf("get remediation: %w", err)
	}
	return result, nil
}

// allowedSortColumns defines the whitelist of columns that can be used for sorting.
var allowedSortColumns = map[string]bool{
	"created_at": true,
	"updated_at": true,
	"severity":   true,
	"status":     true,
	"sla_due_at": true,
}

// List returns paginated remediations with filtering and sorting.
func (r *RemediationRepository) List(ctx context.Context, tenantID uuid.UUID, params *dto.RemediationListParams) ([]model.Remediation, int, error) {
	params.SetDefaults()

	var (
		conditions []string
		args       []interface{}
		argIdx     int
	)

	nextArg := func(val interface{}) string {
		argIdx++
		args = append(args, val)
		return fmt.Sprintf("$%d", argIdx)
	}

	// Tenant isolation is mandatory.
	conditions = append(conditions, "tenant_id = "+nextArg(tenantID))

	if len(params.Status) > 0 {
		placeholders := make([]string, len(params.Status))
		for i, s := range params.Status {
			placeholders[i] = nextArg(s)
		}
		conditions = append(conditions, "status IN ("+strings.Join(placeholders, ", ")+")")
	}

	if len(params.Severity) > 0 {
		placeholders := make([]string, len(params.Severity))
		for i, s := range params.Severity {
			placeholders[i] = nextArg(s)
		}
		conditions = append(conditions, "severity IN ("+strings.Join(placeholders, ", ")+")")
	}

	if len(params.FindingType) > 0 {
		placeholders := make([]string, len(params.FindingType))
		for i, ft := range params.FindingType {
			placeholders[i] = nextArg(ft)
		}
		conditions = append(conditions, "finding_type IN ("+strings.Join(placeholders, ", ")+")")
	}

	if params.AssignedTo != nil {
		conditions = append(conditions, "assigned_to = "+nextArg(*params.AssignedTo))
	}

	if params.AssetID != nil {
		conditions = append(conditions, "data_asset_id = "+nextArg(*params.AssetID))
	}

	if params.SLABreached != nil {
		conditions = append(conditions, "sla_breached = "+nextArg(*params.SLABreached))
	}

	if strings.TrimSpace(params.Search) != "" {
		search := "%" + strings.TrimSpace(params.Search) + "%"
		conditions = append(conditions, "title ILIKE "+nextArg(search))
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count query.
	var countQuery strings.Builder
	countQuery.WriteString("SELECT COUNT(*) FROM dspm_remediations WHERE ")
	countQuery.WriteString(whereClause)

	var total int
	if err := r.db.QueryRow(ctx, countQuery.String(), args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count remediations: %w", err)
	}

	// Data query with sorting and pagination.
	sortCol := params.Sort
	if !allowedSortColumns[sortCol] {
		sortCol = "created_at"
	}
	order := strings.ToUpper(params.Order)
	if order != "ASC" && order != "DESC" {
		order = "DESC"
	}

	offset := (params.Page - 1) * params.PerPage

	var dataQuery strings.Builder
	dataQuery.WriteString(`SELECT id, tenant_id, finding_type, finding_id, data_asset_id, data_asset_name,
		identity_id, playbook_id, title, description, severity, steps,
		current_step, total_steps, assigned_to, assigned_team, sla_due_at,
		sla_breached, risk_score_before, risk_score_after, risk_reduction,
		pre_action_state, rollback_available, rolled_back, status,
		cyber_alert_id, created_by, created_at, updated_at, completed_at, compliance_tags
		FROM dspm_remediations WHERE `)
	dataQuery.WriteString(whereClause)
	dataQuery.WriteString(fmt.Sprintf(" ORDER BY %s %s", sortCol, order))
	dataQuery.WriteString(fmt.Sprintf(" LIMIT %s OFFSET %s", nextArg(params.PerPage), nextArg(offset)))

	rows, err := r.db.Query(ctx, dataQuery.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list remediations: %w", err)
	}
	defer rows.Close()

	items := make([]model.Remediation, 0)
	for rows.Next() {
		item, err := scanRemediation(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan remediation row: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate remediations: %w", err)
	}

	return items, total, nil
}

// Update updates all mutable fields of a remediation record.
func (r *RemediationRepository) Update(ctx context.Context, remediation *model.Remediation) error {
	if len(remediation.Steps) == 0 {
		remediation.Steps = json.RawMessage("[]")
	}
	if len(remediation.ComplianceTags) == 0 {
		remediation.ComplianceTags = json.RawMessage("[]")
	}
	if len(remediation.PreActionState) == 0 {
		remediation.PreActionState = json.RawMessage("{}")
	}

	tag, err := r.db.Exec(ctx, `
		UPDATE dspm_remediations
		SET
			finding_type = $3,
			finding_id = $4,
			data_asset_id = $5,
			data_asset_name = $6,
			identity_id = $7,
			playbook_id = $8,
			title = $9,
			description = $10,
			severity = $11,
			steps = $12,
			current_step = $13,
			total_steps = $14,
			assigned_to = $15,
			assigned_team = $16,
			sla_due_at = $17,
			sla_breached = $18,
			risk_score_before = $19,
			risk_score_after = $20,
			risk_reduction = $21,
			pre_action_state = $22,
			rollback_available = $23,
			rolled_back = $24,
			status = $25,
			cyber_alert_id = $26,
			compliance_tags = $27,
			completed_at = $28,
			updated_at = now()
		WHERE tenant_id = $1 AND id = $2`,
		remediation.TenantID, remediation.ID,
		remediation.FindingType, remediation.FindingID,
		remediation.DataAssetID, remediation.DataAssetName,
		remediation.IdentityID, remediation.PlaybookID,
		remediation.Title, remediation.Description,
		remediation.Severity, remediation.Steps,
		remediation.CurrentStep, remediation.TotalSteps,
		remediation.AssignedTo, remediation.AssignedTeam,
		remediation.SLADueAt, remediation.SLABreached,
		remediation.RiskScoreBefore, remediation.RiskScoreAfter,
		remediation.RiskReduction, remediation.PreActionState,
		remediation.RollbackAvailable, remediation.RolledBack,
		remediation.Status, remediation.CyberAlertID,
		remediation.ComplianceTags, remediation.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("update remediation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("remediation not found")
	}
	return nil
}

// terminalStatuses lists statuses that represent a final state.
var terminalStatuses = []model.RemediationStatus{
	model.StatusCompleted, model.StatusCancelled,
	model.StatusRolledBack, model.StatusExceptionGranted,
}

// isTerminalStatus returns true if the status is a terminal state.
func isTerminalStatus(s model.RemediationStatus) bool {
	for _, ts := range terminalStatuses {
		if s == ts {
			return true
		}
	}
	return false
}

// UpdateStatus updates the status field and sets completed_at for terminal statuses.
func (r *RemediationRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status model.RemediationStatus) error {
	var query string
	if isTerminalStatus(status) {
		query = `
			UPDATE dspm_remediations
			SET status = $3, completed_at = now(), updated_at = now()
			WHERE tenant_id = $1 AND id = $2`
	} else {
		query = `
			UPDATE dspm_remediations
			SET status = $3, updated_at = now()
			WHERE tenant_id = $1 AND id = $2`
	}

	tag, err := r.db.Exec(ctx, query, tenantID, id, status)
	if err != nil {
		return fmt.Errorf("update remediation status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("remediation not found")
	}
	return nil
}

// UpdateSteps updates the steps JSONB and current_step for a remediation.
func (r *RemediationRepository) UpdateSteps(ctx context.Context, tenantID, id uuid.UUID, steps json.RawMessage, currentStep int) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE dspm_remediations
		SET steps = $3, current_step = $4, updated_at = now()
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id, steps, currentStep,
	)
	if err != nil {
		return fmt.Errorf("update remediation steps: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("remediation not found")
	}
	return nil
}

// Stats returns aggregated remediation statistics for the dashboard.
func (r *RemediationRepository) Stats(ctx context.Context, tenantID uuid.UUID) (*model.RemediationStats, error) {
	stats := &model.RemediationStats{
		ByStatus:      make(map[string]int),
		BySeverity:    make(map[string]int),
		ByFindingType: make(map[string]int),
	}

	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status IN ('open', 'in_progress', 'awaiting_approval')) AS total_open,
			COUNT(*) FILTER (WHERE status IN ('open', 'in_progress', 'awaiting_approval') AND severity = 'critical') AS total_critical_open,
			COUNT(*) FILTER (WHERE status = 'in_progress') AS total_in_progress,
			COUNT(*) FILTER (WHERE status = 'completed' AND completed_at >= now() - interval '7 days') AS completed_last_7_days,
			COUNT(*) FILTER (WHERE sla_breached = true AND status IN ('open', 'in_progress', 'awaiting_approval')) AS sla_breaches,
			COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - created_at)) / 3600.0) FILTER (WHERE status = 'completed' AND completed_at IS NOT NULL), 0) AS avg_resolution_hours,
			COALESCE(SUM(risk_reduction) FILTER (WHERE status = 'completed'), 0) AS total_risk_reduction
		FROM dspm_remediations
		WHERE tenant_id = $1`,
		tenantID,
	).Scan(
		&stats.TotalOpen,
		&stats.TotalCriticalOpen,
		&stats.TotalInProgress,
		&stats.CompletedLast7Days,
		&stats.SLABreaches,
		&stats.AvgResolutionHours,
		&stats.TotalRiskReduction,
	)
	if err != nil {
		return nil, fmt.Errorf("remediation stats aggregates: %w", err)
	}

	// By status breakdown.
	statusRows, err := r.db.Query(ctx, `
		SELECT status::text, COUNT(*)
		FROM dspm_remediations
		WHERE tenant_id = $1
		GROUP BY status
		ORDER BY COUNT(*) DESC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("remediation stats by status: %w", err)
	}
	defer statusRows.Close()
	for statusRows.Next() {
		var name string
		var count int
		if err := statusRows.Scan(&name, &count); err != nil {
			return nil, fmt.Errorf("scan status stat: %w", err)
		}
		stats.ByStatus[name] = count
	}
	if err := statusRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate status stats: %w", err)
	}

	// By severity breakdown.
	sevRows, err := r.db.Query(ctx, `
		SELECT severity::text, COUNT(*)
		FROM dspm_remediations
		WHERE tenant_id = $1
		GROUP BY severity
		ORDER BY COUNT(*) DESC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("remediation stats by severity: %w", err)
	}
	defer sevRows.Close()
	for sevRows.Next() {
		var name string
		var count int
		if err := sevRows.Scan(&name, &count); err != nil {
			return nil, fmt.Errorf("scan severity stat: %w", err)
		}
		stats.BySeverity[name] = count
	}
	if err := sevRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate severity stats: %w", err)
	}

	// By finding type breakdown.
	ftRows, err := r.db.Query(ctx, `
		SELECT finding_type::text, COUNT(*)
		FROM dspm_remediations
		WHERE tenant_id = $1
		GROUP BY finding_type
		ORDER BY COUNT(*) DESC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("remediation stats by finding type: %w", err)
	}
	defer ftRows.Close()
	for ftRows.Next() {
		var name string
		var count int
		if err := ftRows.Scan(&name, &count); err != nil {
			return nil, fmt.Errorf("scan finding type stat: %w", err)
		}
		stats.ByFindingType[name] = count
	}
	if err := ftRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate finding type stats: %w", err)
	}

	return stats, nil
}

// FindSLABreached returns remediations whose SLA has elapsed but are not yet marked as breached.
func (r *RemediationRepository) FindSLABreached(ctx context.Context, tenantID uuid.UUID) ([]model.Remediation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, finding_type, finding_id, data_asset_id, data_asset_name,
			identity_id, playbook_id, title, description, severity, steps,
			current_step, total_steps, assigned_to, assigned_team, sla_due_at,
			sla_breached, risk_score_before, risk_score_after, risk_reduction,
			pre_action_state, rollback_available, rolled_back, status,
			cyber_alert_id, created_by, created_at, updated_at, completed_at, compliance_tags
		FROM dspm_remediations
		WHERE tenant_id = $1
			AND sla_due_at <= now()
			AND sla_breached = false
			AND status NOT IN ('completed', 'cancelled', 'rolled_back', 'exception_granted')`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("find sla breached remediations: %w", err)
	}
	defer rows.Close()

	items := make([]model.Remediation, 0)
	for rows.Next() {
		item, err := scanRemediation(rows)
		if err != nil {
			return nil, fmt.Errorf("scan sla breached remediation: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sla breached remediations: %w", err)
	}
	return items, nil
}

// MarkSLABreached sets the sla_breached flag to true for a given remediation.
func (r *RemediationRepository) MarkSLABreached(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE dspm_remediations
		SET sla_breached = true, updated_at = now()
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	if err != nil {
		return fmt.Errorf("mark sla breached: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("remediation not found")
	}
	return nil
}

// BurndownData generates open/closed remediation counts per day for the last N days.
func (r *RemediationRepository) BurndownData(ctx context.Context, tenantID uuid.UUID, days int) ([]model.BurndownDataPoint, error) {
	if days <= 0 {
		days = 30
	}

	start := time.Now().UTC().AddDate(0, 0, -(days - 1)).Truncate(24 * time.Hour)

	rows, err := r.db.Query(ctx, `
		WITH date_series AS (
			SELECT generate_series($2::timestamptz, date_trunc('day', now()), interval '1 day') AS day
		)
		SELECT
			ds.day::date::text AS date,
			COALESCE(opened.cnt, 0) AS open_count,
			COALESCE(closed.cnt, 0) AS closed_count
		FROM date_series ds
		LEFT JOIN (
			SELECT date_trunc('day', created_at)::date AS day, COUNT(*)::int AS cnt
			FROM dspm_remediations
			WHERE tenant_id = $1
				AND created_at >= $2
			GROUP BY 1
		) opened ON opened.day = ds.day::date
		LEFT JOIN (
			SELECT date_trunc('day', completed_at)::date AS day, COUNT(*)::int AS cnt
			FROM dspm_remediations
			WHERE tenant_id = $1
				AND completed_at IS NOT NULL
				AND completed_at >= $2
				AND status IN ('completed', 'cancelled', 'rolled_back', 'exception_granted')
			GROUP BY 1
		) closed ON closed.day = ds.day::date
		ORDER BY ds.day ASC`,
		tenantID, start,
	)
	if err != nil {
		return nil, fmt.Errorf("burndown data: %w", err)
	}
	defer rows.Close()

	points := make([]model.BurndownDataPoint, 0, days)
	for rows.Next() {
		var point model.BurndownDataPoint
		if err := rows.Scan(&point.Date, &point.Open, &point.Closed); err != nil {
			return nil, fmt.Errorf("scan burndown point: %w", err)
		}
		points = append(points, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate burndown data: %w", err)
	}
	return points, nil
}

// scanner is a common interface for pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// scanRemediation scans a single remediation row into a model.Remediation.
func scanRemediation(row scanner) (*model.Remediation, error) {
	var item model.Remediation
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.FindingType,
		&item.FindingID,
		&item.DataAssetID,
		&item.DataAssetName,
		&item.IdentityID,
		&item.PlaybookID,
		&item.Title,
		&item.Description,
		&item.Severity,
		&item.Steps,
		&item.CurrentStep,
		&item.TotalSteps,
		&item.AssignedTo,
		&item.AssignedTeam,
		&item.SLADueAt,
		&item.SLABreached,
		&item.RiskScoreBefore,
		&item.RiskScoreAfter,
		&item.RiskReduction,
		&item.PreActionState,
		&item.RollbackAvailable,
		&item.RolledBack,
		&item.Status,
		&item.CyberAlertID,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.CompletedAt,
		&item.ComplianceTags,
	); err != nil {
		return nil, err
	}
	if len(item.Steps) == 0 {
		item.Steps = json.RawMessage("[]")
	}
	if len(item.ComplianceTags) == 0 {
		item.ComplianceTags = json.RawMessage("[]")
	}
	if len(item.PreActionState) == 0 {
		item.PreActionState = json.RawMessage("{}")
	}
	return &item, nil
}
