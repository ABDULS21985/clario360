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
	"github.com/clario360/platform/internal/database"
)

type CTEMAssessmentRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewCTEMAssessmentRepository(db *pgxpool.Pool, logger zerolog.Logger) *CTEMAssessmentRepository {
	return &CTEMAssessmentRepository{db: db, logger: logger}
}

func (r *CTEMAssessmentRepository) Create(ctx context.Context, tenantID, userID uuid.UUID, req *dto.CreateCTEMAssessmentRequest) (*model.CTEMAssessment, error) {
	id := uuid.New()
	now := time.Now().UTC()
	scopeJSON, err := json.Marshal(req.Scope)
	if err != nil {
		return nil, fmt.Errorf("marshal scope: %w", err)
	}
	phasesJSON, err := json.Marshal(defaultPhaseProgress())
	if err != nil {
		return nil, fmt.Errorf("marshal phases: %w", err)
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO ctem_assessments (
			id, tenant_id, name, description, status, scope, phases,
			scheduled, schedule_cron, tags, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13
		)`,
		id, tenantID, req.Name, req.Description, string(model.CTEMAssessmentStatusCreated),
		scopeJSON, phasesJSON, req.Scheduled, req.ScheduleCron, req.Tags, userID, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create assessment: %w", err)
	}
	return r.GetByID(ctx, tenantID, id)
}

func (r *CTEMAssessmentRepository) GetByID(ctx context.Context, tenantID, assessmentID uuid.UUID) (*model.CTEMAssessment, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, description, status, scope, resolved_asset_ids,
		       resolved_asset_count, phases, current_phase, exposure_score, score_breakdown,
		       findings_summary, started_at, completed_at, duration_ms, error_message,
		       error_phase, scheduled, schedule_cron, parent_assessment_id, tags,
		       created_by, created_at, updated_at, deleted_at
		FROM ctem_assessments
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, assessmentID,
	)
	assessment, err := scanCTEMAssessment(row)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get assessment: %w", err)
	}
	return assessment, nil
}

func (r *CTEMAssessmentRepository) List(ctx context.Context, tenantID uuid.UUID, params *dto.CTEMAssessmentListParams) ([]*model.CTEMAssessment, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT id, tenant_id, name, description, status, scope, resolved_asset_ids,
		       resolved_asset_count, phases, current_phase, exposure_score, score_breakdown,
		       findings_summary, started_at, completed_at, duration_ms, error_message,
		       error_phase, scheduled, schedule_cron, parent_assessment_id, tags,
		       created_by, created_at, updated_at, deleted_at
		FROM ctem_assessments a`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.Where("a.deleted_at IS NULL")
	if params.Status != nil {
		qb.Where("a.status = ?", *params.Status)
	}
	if params.Scheduled != nil {
		qb.Where("a.scheduled = ?", *params.Scheduled)
	}
	if params.Search != nil && *params.Search != "" {
		qb.Where("(a.name ILIKE ? OR a.description ILIKE ?)", "%"+*params.Search+"%", "%"+*params.Search+"%")
	}
	if params.Tag != nil && *params.Tag != "" {
		qb.WhereArrayContains("a.tags", *params.Tag)
	}
	qb.OrderBy(params.Sort, params.Order, []string{"created_at", "updated_at", "started_at", "completed_at", "exposure_score", "name"})
	qb.Paginate(params.Page, params.PerPage)

	countSQL, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count assessments: %w", err)
	}

	sql, args := qb.Build()
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list assessments: %w", err)
	}
	defer rows.Close()

	items := make([]*model.CTEMAssessment, 0)
	for rows.Next() {
		item, err := scanCTEMAssessment(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan assessment: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *CTEMAssessmentRepository) UpdateDefinition(ctx context.Context, tenantID, assessmentID uuid.UUID, req *dto.UpdateCTEMAssessmentRequest) (*model.CTEMAssessment, error) {
	sets := []string{"updated_at = now()"}
	args := make([]any, 0, 8)
	index := 1

	add := func(clause string, value any) {
		sets = append(sets, fmt.Sprintf("%s = $%d", clause, index))
		args = append(args, value)
		index++
	}

	if req.Name != nil {
		add("name", *req.Name)
	}
	if req.Description != nil {
		add("description", *req.Description)
	}
	if req.Scope != nil {
		scopeJSON, err := json.Marshal(req.Scope)
		if err != nil {
			return nil, fmt.Errorf("marshal scope: %w", err)
		}
		add("scope", scopeJSON)
	}
	if req.Scheduled != nil {
		add("scheduled", *req.Scheduled)
	}
	if req.ScheduleCron != nil {
		add("schedule_cron", *req.ScheduleCron)
	}
	if req.Tags != nil {
		add("tags", *req.Tags)
	}

	args = append(args, tenantID, assessmentID, string(model.CTEMAssessmentStatusCreated))
	sql := fmt.Sprintf(`
		UPDATE ctem_assessments
		SET %s
		WHERE tenant_id = $%d AND id = $%d AND status = $%d AND deleted_at IS NULL`,
		strings.Join(sets, ", "), index, index+1, index+2,
	)

	tag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("update assessment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrConflict
	}
	return r.GetByID(ctx, tenantID, assessmentID)
}

func (r *CTEMAssessmentRepository) SaveState(ctx context.Context, assessment *model.CTEMAssessment) error {
	scopeJSON, err := json.Marshal(assessment.Scope)
	if err != nil {
		return fmt.Errorf("marshal scope: %w", err)
	}
	phasesJSON, err := json.Marshal(assessment.Phases)
	if err != nil {
		return fmt.Errorf("marshal phases: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		UPDATE ctem_assessments
		SET name = $3,
		    description = $4,
		    status = $5,
		    scope = $6,
		    resolved_asset_ids = $7,
		    resolved_asset_count = $8,
		    phases = $9,
		    current_phase = $10,
		    exposure_score = $11,
		    score_breakdown = $12,
		    findings_summary = $13,
		    started_at = $14,
		    completed_at = $15,
		    duration_ms = $16,
		    error_message = $17,
		    error_phase = $18,
		    scheduled = $19,
		    schedule_cron = $20,
		    parent_assessment_id = $21,
		    tags = $22,
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		assessment.TenantID, assessment.ID, assessment.Name, assessment.Description,
		string(assessment.Status), scopeJSON, assessment.ResolvedAssetIDs, assessment.ResolvedAssetCount,
		phasesJSON, assessment.CurrentPhase, assessment.ExposureScore, jsonOrNil(assessment.ScoreBreakdown),
		jsonOrNil(assessment.FindingsSummary), assessment.StartedAt, assessment.CompletedAt,
		assessment.DurationMs, assessment.ErrorMessage, assessment.ErrorPhase, assessment.Scheduled,
		assessment.ScheduleCron, assessment.ParentAssessmentID, assessment.Tags,
	)
	if err != nil {
		return fmt.Errorf("save assessment state: %w", err)
	}
	return nil
}

func (r *CTEMAssessmentRepository) SoftDelete(ctx context.Context, tenantID, assessmentID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE ctem_assessments
		SET deleted_at = now(), updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, assessmentID,
	)
	if err != nil {
		return fmt.Errorf("soft delete assessment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CTEMAssessmentRepository) ListActive(ctx context.Context, tenantID uuid.UUID) ([]*model.CTEMAssessment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, name, description, status, scope, resolved_asset_ids,
		       resolved_asset_count, phases, current_phase, exposure_score, score_breakdown,
		       findings_summary, started_at, completed_at, duration_ms, error_message,
		       error_phase, scheduled, schedule_cron, parent_assessment_id, tags,
		       created_by, created_at, updated_at, deleted_at
		FROM ctem_assessments
		WHERE tenant_id = $1 AND status IN ('scoping','discovery','prioritizing','validating','mobilizing')
		  AND deleted_at IS NULL
		ORDER BY created_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*model.CTEMAssessment, 0)
	for rows.Next() {
		item, err := scanCTEMAssessment(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *CTEMAssessmentRepository) LatestCompleted(ctx context.Context, tenantID uuid.UUID) (*model.CTEMAssessment, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, description, status, scope, resolved_asset_ids,
		       resolved_asset_count, phases, current_phase, exposure_score, score_breakdown,
		       findings_summary, started_at, completed_at, duration_ms, error_message,
		       error_phase, scheduled, schedule_cron, parent_assessment_id, tags,
		       created_by, created_at, updated_at, deleted_at
		FROM ctem_assessments
		WHERE tenant_id = $1 AND status = 'completed' AND deleted_at IS NULL
		ORDER BY completed_at DESC NULLS LAST, created_at DESC
		LIMIT 1`,
		tenantID,
	)
	item, err := scanCTEMAssessment(row)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return item, err
}

func defaultPhaseProgress() map[string]model.PhaseProgress {
	return map[string]model.PhaseProgress{
		"scoping":       {Status: model.CTEMPhaseStatusPending},
		"discovery":     {Status: model.CTEMPhaseStatusPending},
		"prioritizing":  {Status: model.CTEMPhaseStatusPending},
		"validating":    {Status: model.CTEMPhaseStatusPending},
		"mobilizing":    {Status: model.CTEMPhaseStatusPending},
	}
}

func scanCTEMAssessment(row interface{ Scan(dest ...any) error }) (*model.CTEMAssessment, error) {
	var (
		item              model.CTEMAssessment
		scopeJSON         []byte
		phasesJSON        []byte
		scoreBreakdown    []byte
		findingsSummary   []byte
	)
	err := row.Scan(
		&item.ID, &item.TenantID, &item.Name, &item.Description, &item.Status, &scopeJSON, &item.ResolvedAssetIDs,
		&item.ResolvedAssetCount, &phasesJSON, &item.CurrentPhase, &item.ExposureScore, &scoreBreakdown,
		&findingsSummary, &item.StartedAt, &item.CompletedAt, &item.DurationMs, &item.ErrorMessage,
		&item.ErrorPhase, &item.Scheduled, &item.ScheduleCron, &item.ParentAssessmentID, &item.Tags,
		&item.CreatedBy, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(scopeJSON) > 0 {
		if err := json.Unmarshal(scopeJSON, &item.Scope); err != nil {
			return nil, fmt.Errorf("unmarshal scope: %w", err)
		}
	}
	if len(phasesJSON) > 0 {
		if err := json.Unmarshal(phasesJSON, &item.Phases); err != nil {
			return nil, fmt.Errorf("unmarshal phases: %w", err)
		}
	}
	if len(scoreBreakdown) > 0 {
		item.ScoreBreakdown = scoreBreakdown
	}
	if len(findingsSummary) > 0 {
		item.FindingsSummary = findingsSummary
	}
	if item.Phases == nil {
		item.Phases = defaultPhaseProgress()
	}
	if item.Tags == nil {
		item.Tags = []string{}
	}
	return &item, nil
}

func jsonOrNil(in json.RawMessage) any {
	if len(in) == 0 {
		return nil
	}
	return in
}
