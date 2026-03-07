package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
)

// RemediationRepository handles remediation_actions table operations.
type RemediationRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewRemediationRepository creates a new RemediationRepository.
func NewRemediationRepository(db *pgxpool.Pool, logger zerolog.Logger) *RemediationRepository {
	return &RemediationRepository{db: db, logger: logger}
}

// Create inserts a new remediation action.
func (r *RemediationRepository) Create(ctx context.Context, tenantID, createdBy uuid.UUID, req *dto.CreateRemediationRequest) (*model.RemediationAction, error) {
	id := uuid.New()
	now := time.Now().UTC()

	planJSON, err := json.Marshal(req.Plan)
	if err != nil {
		return nil, fmt.Errorf("marshal plan: %w", err)
	}

	severity := req.Severity
	if severity == "" {
		severity = "medium"
	}
	executionMode := req.ExecutionMode
	if executionMode == "" {
		executionMode = "manual"
	}
	requiresApprovalFrom := req.RequiresApprovalFrom
	if requiresApprovalFrom == "" {
		requiresApprovalFrom = "security_manager"
	}

	assetIDs := req.AffectedAssetIDs
	if assetIDs == nil {
		assetIDs = []uuid.UUID{}
	}
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}
	metadata := req.Metadata
	if metadata == nil {
		metadata = map[string]interface{}{}
	}
	metaJSON, _ := json.Marshal(metadata)

	_, err = r.db.Exec(ctx, `
		INSERT INTO remediation_actions (
			id, tenant_id, alert_id, vulnerability_id, assessment_id, ctem_finding_id, remediation_group_id,
			type, severity, title, description, plan, affected_asset_ids, affected_asset_count,
			execution_mode, status, requires_approval_from, tags, metadata, created_by, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,'draft',$16,$17,$18,$19,$20,$20
		)`,
		id, tenantID, req.AlertID, req.VulnerabilityID, req.AssessmentID, req.CTEMFindingID, req.RemediationGroupID,
		req.Type, severity, req.Title, req.Description, planJSON, assetIDs, len(assetIDs),
		executionMode, requiresApprovalFrom, tags, metaJSON, createdBy, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert remediation: %w", err)
	}
	return r.GetByID(ctx, tenantID, id)
}

// GetByID retrieves a remediation action by ID with tenant isolation.
func (r *RemediationRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*model.RemediationAction, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, alert_id, vulnerability_id, assessment_id, ctem_finding_id, remediation_group_id,
		       type, severity, title, description, plan, affected_asset_ids, affected_asset_count,
		       execution_mode, status, submitted_by, submitted_at, approved_by, approved_at,
		       rejected_by, rejected_at, rejection_reason, approval_notes, requires_approval_from,
		       dry_run_result, dry_run_at, dry_run_duration_ms, pre_execution_state,
		       execution_result, executed_by, execution_started_at, execution_completed_at, execution_duration_ms,
		       verification_result, verified_by, verified_at,
		       rollback_result, rollback_reason, rollback_approved_by, rolled_back_at, rollback_deadline,
		       workflow_instance_id, tags, metadata, created_by, created_at, updated_at
		FROM remediation_actions
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		id, tenantID,
	)
	return scanRemediation(row)
}

// Update updates fields on a draft or revision_requested remediation.
func (r *RemediationRepository) Update(ctx context.Context, tenantID, id uuid.UUID, req *dto.UpdateRemediationRequest) (*model.RemediationAction, error) {
	action, err := r.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		action.Title = *req.Title
	}
	if req.Description != nil {
		action.Description = *req.Description
	}
	if req.Plan != nil {
		action.Plan = *req.Plan
	}
	if req.AffectedAssetIDs != nil {
		action.AffectedAssetIDs = req.AffectedAssetIDs
		action.AffectedAssetCount = len(req.AffectedAssetIDs)
	}
	if req.Severity != nil {
		action.Severity = *req.Severity
	}
	if req.Tags != nil {
		action.Tags = req.Tags
	}
	if req.Metadata != nil {
		action.Metadata = req.Metadata
	}

	planJSON, err := json.Marshal(action.Plan)
	if err != nil {
		return nil, fmt.Errorf("marshal plan: %w", err)
	}
	metaJSON, _ := json.Marshal(action.Metadata)

	_, err = r.db.Exec(ctx, `
		UPDATE remediation_actions
		SET title=$1, description=$2, plan=$3, affected_asset_ids=$4, affected_asset_count=$5,
		    severity=$6, tags=$7, metadata=$8, updated_at=$9
		WHERE id=$10 AND tenant_id=$11 AND deleted_at IS NULL`,
		action.Title, action.Description, planJSON, action.AffectedAssetIDs, action.AffectedAssetCount,
		action.Severity, action.Tags, metaJSON, time.Now().UTC(), id, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("update remediation: %w", err)
	}
	return r.GetByID(ctx, tenantID, id)
}

// UpdateStatus performs a status transition and associated field updates.
func (r *RemediationRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status model.RemediationStatus, fields map[string]interface{}) error {
	fields["status"] = string(status)
	fields["updated_at"] = time.Now().UTC()

	setClauses := make([]string, 0, len(fields))
	args := make([]interface{}, 0, len(fields)+2)
	i := 1
	for k, v := range fields {
		setClauses = append(setClauses, fmt.Sprintf("%s=$%d", k, i))
		args = append(args, v)
		i++
	}
	args = append(args, id, tenantID)

	query := fmt.Sprintf(
		"UPDATE remediation_actions SET %s WHERE id=$%d AND tenant_id=$%d AND deleted_at IS NULL",
		strings.Join(setClauses, ", "), i, i+1,
	)
	ct, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SoftDelete soft-deletes a remediation action.
func (r *RemediationRepository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	ct, err := r.db.Exec(ctx,
		"UPDATE remediation_actions SET deleted_at=$1, updated_at=$1 WHERE id=$2 AND tenant_id=$3 AND deleted_at IS NULL",
		time.Now().UTC(), id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("soft delete remediation: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// List retrieves remediations with filtering and pagination.
func (r *RemediationRepository) List(ctx context.Context, tenantID uuid.UUID, params *dto.RemediationListParams) ([]*model.RemediationAction, int, error) {
	conds := []string{"tenant_id=$1", "deleted_at IS NULL"}
	args := []interface{}{tenantID}
	i := 2

	if len(params.Statuses) > 0 {
		conds = append(conds, fmt.Sprintf("status = ANY($%d)", i))
		args = append(args, params.Statuses)
		i++
	}
	if len(params.Types) > 0 {
		conds = append(conds, fmt.Sprintf("type = ANY($%d)", i))
		args = append(args, params.Types)
		i++
	}
	if len(params.Severities) > 0 {
		conds = append(conds, fmt.Sprintf("severity = ANY($%d)", i))
		args = append(args, params.Severities)
		i++
	}
	if params.AssetID != nil {
		conds = append(conds, fmt.Sprintf("$%d = ANY(affected_asset_ids)", i))
		args = append(args, *params.AssetID)
		i++
	}
	if params.AlertID != nil {
		conds = append(conds, fmt.Sprintf("alert_id=$%d", i))
		args = append(args, *params.AlertID)
		i++
	}
	if params.VulnID != nil {
		conds = append(conds, fmt.Sprintf("vulnerability_id=$%d", i))
		args = append(args, *params.VulnID)
		i++
	}
	if params.Search != nil && *params.Search != "" {
		conds = append(conds, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", i, i))
		args = append(args, "%"+*params.Search+"%")
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM remediation_actions "+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count remediations: %w", err)
	}

	order := "created_at"
	if params.Sort != "" {
		order = params.Sort
	}
	dir := "DESC"
	if strings.ToLower(params.Order) == "asc" {
		dir = "ASC"
	}

	offset := (params.Page - 1) * params.PerPage
	query := fmt.Sprintf(
		`SELECT id, tenant_id, alert_id, vulnerability_id, assessment_id, ctem_finding_id, remediation_group_id,
		        type, severity, title, description, plan, affected_asset_ids, affected_asset_count,
		        execution_mode, status, submitted_by, submitted_at, approved_by, approved_at,
		        rejected_by, rejected_at, rejection_reason, approval_notes, requires_approval_from,
		        dry_run_result, dry_run_at, dry_run_duration_ms, pre_execution_state,
		        execution_result, executed_by, execution_started_at, execution_completed_at, execution_duration_ms,
		        verification_result, verified_by, verified_at,
		        rollback_result, rollback_reason, rollback_approved_by, rolled_back_at, rollback_deadline,
		        workflow_instance_id, tags, metadata, created_by, created_at, updated_at
		 FROM remediation_actions %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		where, order, dir, i, i+1,
	)
	args = append(args, params.PerPage, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list remediations: %w", err)
	}
	defer rows.Close()

	var actions []*model.RemediationAction
	for rows.Next() {
		a, err := scanRemediation(rows)
		if err != nil {
			return nil, 0, err
		}
		actions = append(actions, a)
	}
	return actions, total, rows.Err()
}

// Stats returns remediation statistics for a tenant.
func (r *RemediationRepository) Stats(ctx context.Context, tenantID uuid.UUID) (*model.RemediationStats, error) {
	rows, err := r.db.Query(ctx, `
		SELECT status, COUNT(*) FROM remediation_actions
		WHERE tenant_id=$1 AND deleted_at IS NULL
		GROUP BY status`, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("stats query: %w", err)
	}
	defer rows.Close()

	stats := &model.RemediationStats{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats.Total += count
		switch model.RemediationStatus(status) {
		case model.StatusDraft:
			stats.Draft = count
		case model.StatusPendingApproval:
			stats.PendingApproval = count
		case model.StatusApproved:
			stats.Approved = count
		case model.StatusDryRunCompleted:
			stats.DryRunCompleted = count
		case model.StatusExecuting:
			stats.Executing = count
		case model.StatusExecuted:
			stats.Executed = count
		case model.StatusVerified:
			stats.Verified = count
		case model.StatusVerificationFailed:
			stats.VerificationFailed = count
		case model.StatusRolledBack:
			stats.RolledBack = count
		case model.StatusExecutionFailed, model.StatusRollbackFailed:
			stats.Failed += count
		case model.StatusClosed:
			stats.Closed = count
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Compute derived metrics
	var avgMs float64
	_ = r.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(execution_duration_ms), 0)
		FROM remediation_actions
		WHERE tenant_id=$1 AND execution_duration_ms IS NOT NULL AND deleted_at IS NULL`,
		tenantID,
	).Scan(&avgMs)
	stats.AvgExecutionHours = avgMs / 3_600_000

	if executed := stats.Executed + stats.Verified + stats.VerificationFailed + stats.RolledBack; executed > 0 {
		stats.VerificationSuccessRate = float64(stats.Verified) / float64(executed) * 100
		stats.RollbackRate = float64(stats.RolledBack) / float64(executed) * 100
	}

	return stats, nil
}

// scanRemediation scans a single row into a RemediationAction.
func scanRemediation(row interface {
	Scan(...interface{}) error
}) (*model.RemediationAction, error) {
	var a model.RemediationAction
	var planJSON, metaJSON []byte
	var dryRunResultJSON, preExecJSON, execResultJSON, verResultJSON, rollbackResultJSON []byte
	var tags []string
	var assetIDs []uuid.UUID

	err := row.Scan(
		&a.ID, &a.TenantID, &a.AlertID, &a.VulnerabilityID, &a.AssessmentID, &a.CTEMFindingID, &a.RemediationGroupID,
		&a.Type, &a.Severity, &a.Title, &a.Description, &planJSON, &assetIDs, &a.AffectedAssetCount,
		&a.ExecutionMode, &a.Status, &a.SubmittedBy, &a.SubmittedAt, &a.ApprovedBy, &a.ApprovedAt,
		&a.RejectedBy, &a.RejectedAt, &a.RejectionReason, &a.ApprovalNotes, &a.RequiresApprovalFrom,
		&dryRunResultJSON, &a.DryRunAt, &a.DryRunDurationMs, &preExecJSON,
		&execResultJSON, &a.ExecutedBy, &a.ExecutionStartedAt, &a.ExecutionCompletedAt, &a.ExecutionDurationMs,
		&verResultJSON, &a.VerifiedBy, &a.VerifiedAt,
		&rollbackResultJSON, &a.RollbackReason, &a.RollbackApprovedBy, &a.RolledBackAt, &a.RollbackDeadline,
		&a.WorkflowInstanceID, &tags, &metaJSON, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan remediation: %w", err)
	}

	_ = json.Unmarshal(planJSON, &a.Plan)
	_ = json.Unmarshal(metaJSON, &a.Metadata)
	if dryRunResultJSON != nil {
		a.DryRunResult = &model.DryRunResult{}
		_ = json.Unmarshal(dryRunResultJSON, a.DryRunResult)
	}
	if preExecJSON != nil {
		a.PreExecutionState = preExecJSON
	}
	if execResultJSON != nil {
		a.ExecutionResult = &model.ExecutionResult{}
		_ = json.Unmarshal(execResultJSON, a.ExecutionResult)
	}
	if verResultJSON != nil {
		a.VerificationResult = &model.VerificationResult{}
		_ = json.Unmarshal(verResultJSON, a.VerificationResult)
	}
	if rollbackResultJSON != nil {
		a.RollbackResult = &model.RollbackResult{}
		_ = json.Unmarshal(rollbackResultJSON, a.RollbackResult)
	}

	a.AffectedAssetIDs = assetIDs
	if a.AffectedAssetIDs == nil {
		a.AffectedAssetIDs = []uuid.UUID{}
	}
	a.Tags = tags
	if a.Tags == nil {
		a.Tags = []string{}
	}
	return &a, nil
}
