package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

type CTEMRemediationGroupRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewCTEMRemediationGroupRepository(db *pgxpool.Pool, logger zerolog.Logger) *CTEMRemediationGroupRepository {
	return &CTEMRemediationGroupRepository{db: db, logger: logger}
}

func (r *CTEMRemediationGroupRepository) ReplaceForAssessment(ctx context.Context, tenantID, assessmentID uuid.UUID, groups []*model.CTEMRemediationGroup) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `DELETE FROM ctem_remediation_groups WHERE tenant_id = $1 AND assessment_id = $2`, tenantID, assessmentID); err != nil {
		return err
	}
	if len(groups) == 0 {
		return tx.Commit(ctx)
	}

	rows := make([][]any, 0, len(groups))
	for _, group := range groups {
		rows = append(rows, []any{
			group.ID, group.TenantID, group.AssessmentID, group.Title, group.Description, string(group.Type),
			group.FindingCount, group.AffectedAssetCount, group.CVEIDs, group.MaxPriorityScore, group.PriorityGroup,
			string(group.Effort), group.EstimatedDays, group.ScoreReduction, string(group.Status),
			group.WorkflowInstanceID, group.TargetDate, group.StartedAt, group.CompletedAt, group.CreatedAt, group.UpdatedAt,
		})
	}
	_, err = tx.CopyFrom(ctx, pgx.Identifier{"ctem_remediation_groups"},
		[]string{
			"id", "tenant_id", "assessment_id", "title", "description", "type",
			"finding_count", "affected_asset_count", "cve_ids", "max_priority_score", "priority_group",
			"effort", "estimated_days", "score_reduction", "status", "workflow_instance_id",
			"target_date", "started_at", "completed_at", "created_at", "updated_at",
		},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("replace remediation groups: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *CTEMRemediationGroupRepository) ListByAssessment(ctx context.Context, tenantID, assessmentID uuid.UUID) ([]*model.CTEMRemediationGroup, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, assessment_id, title, description, type, finding_count,
		       affected_asset_count, cve_ids, max_priority_score, priority_group, effort,
		       estimated_days, score_reduction, status, workflow_instance_id, target_date,
		       started_at, completed_at, created_at, updated_at
		FROM ctem_remediation_groups
		WHERE tenant_id = $1 AND assessment_id = $2
		ORDER BY max_priority_score DESC, created_at ASC`,
		tenantID, assessmentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*model.CTEMRemediationGroup, 0)
	for rows.Next() {
		item, err := scanCTEMRemediationGroup(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *CTEMRemediationGroupRepository) GetByID(ctx context.Context, tenantID, groupID uuid.UUID) (*model.CTEMRemediationGroup, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, assessment_id, title, description, type, finding_count,
		       affected_asset_count, cve_ids, max_priority_score, priority_group, effort,
		       estimated_days, score_reduction, status, workflow_instance_id, target_date,
		       started_at, completed_at, created_at, updated_at
		FROM ctem_remediation_groups
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, groupID,
	)
	item, err := scanCTEMRemediationGroup(row)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return item, err
}

func (r *CTEMRemediationGroupRepository) UpdateStatus(ctx context.Context, tenantID, groupID uuid.UUID, status model.CTEMRemediationGroupStatus) (*model.CTEMRemediationGroup, error) {
	now := time.Now().UTC()
	tag, err := r.db.Exec(ctx, `
		UPDATE ctem_remediation_groups
		SET status = $3,
		    started_at = CASE WHEN $3 = 'in_progress' AND started_at IS NULL THEN $4 ELSE started_at END,
		    completed_at = CASE WHEN $3 = 'completed' THEN $4 ELSE completed_at END,
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, groupID, string(status), now,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, tenantID, groupID)
}

func (r *CTEMRemediationGroupRepository) UpdateWorkflowInstance(ctx context.Context, tenantID, groupID uuid.UUID, workflowInstanceID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE ctem_remediation_groups
		SET workflow_instance_id = $3, updated_at = now()
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, groupID, workflowInstanceID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanCTEMRemediationGroup(row interface{ Scan(dest ...any) error }) (*model.CTEMRemediationGroup, error) {
	var (
		item       model.CTEMRemediationGroup
		groupType  string
		effort     string
		status     string
		targetDate *time.Time
	)
	err := row.Scan(
		&item.ID, &item.TenantID, &item.AssessmentID, &item.Title, &item.Description, &groupType, &item.FindingCount,
		&item.AffectedAssetCount, &item.CVEIDs, &item.MaxPriorityScore, &item.PriorityGroup, &effort,
		&item.EstimatedDays, &item.ScoreReduction, &status, &item.WorkflowInstanceID, &targetDate,
		&item.StartedAt, &item.CompletedAt, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	item.Type = model.CTEMRemediationType(groupType)
	item.Effort = model.CTEMRemediationEffort(effort)
	item.Status = model.CTEMRemediationGroupStatus(status)
	item.TargetDate = targetDate
	if item.CVEIDs == nil {
		item.CVEIDs = []string{}
	}
	return &item, nil
}
