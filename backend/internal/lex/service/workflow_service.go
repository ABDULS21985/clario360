package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/lex/dto"
	"github.com/clario360/platform/internal/lex/metrics"
	"github.com/clario360/platform/internal/lex/model"
	"github.com/clario360/platform/internal/lex/repository"
	workflowmodel "github.com/clario360/platform/internal/workflow/model"
	workflowrepo "github.com/clario360/platform/internal/workflow/repository"
)

const legalReviewWorkflowName = "Lex Contract Review"

type WorkflowService struct {
	db          *pgxpool.Pool
	defRepo     *workflowrepo.DefinitionRepository
	instRepo    *workflowrepo.InstanceRepository
	taskRepo    *workflowrepo.TaskRepository
	contracts   *repository.ContractRepository
	publisher   Publisher
	metrics     *metrics.Metrics
	topic       string
	logger      zerolog.Logger
	now         func() time.Time
}

func NewWorkflowService(db *pgxpool.Pool, defRepo *workflowrepo.DefinitionRepository, instRepo *workflowrepo.InstanceRepository, taskRepo *workflowrepo.TaskRepository, contracts *repository.ContractRepository, publisher Publisher, appMetrics *metrics.Metrics, topic string, logger zerolog.Logger) *WorkflowService {
	return &WorkflowService{
		db:        db,
		defRepo:   defRepo,
		instRepo:  instRepo,
		taskRepo:  taskRepo,
		contracts: contracts,
		publisher: publisherOrNoop(publisher),
		metrics:   appMetrics,
		topic:     topic,
		logger:    logger.With().Str("service", "lex-workflows").Logger(),
		now:       time.Now,
	}
}

func (s *WorkflowService) StartContractReview(ctx context.Context, tenantID, userID, contractID uuid.UUID, req dto.ReviewContractRequest) (*model.LegalWorkflowSummary, error) {
	if s.defRepo == nil || s.instRepo == nil || s.taskRepo == nil {
		return nil, internalError("workflow repositories are not configured", fmt.Errorf("missing workflow dependencies"))
	}
	contract, err := s.contracts.Get(ctx, tenantID, contractID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, notFoundError("contract not found")
		}
		return nil, internalError("load contract", err)
	}
	if contract.WorkflowInstanceID != nil {
		return nil, conflictError("contract already has an active workflow instance")
	}

	definition, err := s.ensureReviewDefinition(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	instance := &workflowmodel.WorkflowInstance{
		ID:            uuid.NewString(),
		TenantID:      tenantID.String(),
		DefinitionID:  definition.ID,
		DefinitionVer: definition.Version,
		Status:        workflowmodel.InstanceStatusRunning,
		CurrentStepID: ptrString("legal_review"),
		Variables: map[string]any{
			"contract_id":    contract.ID.String(),
			"contract_title": contract.Title,
			"contract_type":  string(contract.Type),
		},
		StepOutputs: map[string]any{},
		StartedBy:   ptrString(userID.String()),
	}
	stepExec := &workflowmodel.StepExecution{
		ID:         uuid.NewString(),
		InstanceID: instance.ID,
		StepID:     "legal_review",
		StepType:   workflowmodel.StepTypeHumanTask,
		Status:     workflowmodel.StepStatusPending,
		Attempt:    1,
		CreatedAt:  now,
	}
	task := &workflowmodel.HumanTask{
		TenantID:     tenantID.String(),
		InstanceID:   instance.ID,
		StepID:       "legal_review",
		StepExecID:   stepExec.ID,
		Name:         "Review contract",
		Description:  req.Description,
		Status:       workflowmodel.TaskStatusPending,
		AssigneeRole: normalizeOptionalString(req.ApproverRole),
		FormSchema: []workflowmodel.FormField{
			{Name: "decision", Type: "select", Label: "Decision", Required: true, Options: []string{"approve", "request_changes", "reject"}},
			{Name: "notes", Type: "textarea", Label: "Review notes", Required: false},
		},
		Metadata: map[string]any{
			"contract_id": contract.ID.String(),
		},
	}
	if req.ApproverUserID != nil {
		assignee := req.ApproverUserID.String()
		task.AssigneeID = &assignee
	}
	if req.SLAHours > 0 {
		deadline := now.Add(time.Duration(req.SLAHours) * time.Hour)
		task.SLADeadline = &deadline
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, internalError("start workflow transaction", err)
	}
	defer tx.Rollback(ctx)
	if err := s.instRepo.Create(ctx, instance); err != nil {
		return nil, internalError("create workflow instance", err)
	}
	if err := s.instRepo.CreateStepExecution(ctx, stepExec); err != nil {
		return nil, internalError("create workflow step execution", err)
	}
	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, internalError("create workflow task", err)
	}
	workflowID := uuid.MustParse(instance.ID)
	if err := s.contracts.SetWorkflowInstance(ctx, tx, tenantID, contract.ID, &workflowID); err != nil {
		return nil, internalError("link workflow to contract", err)
	}
	if contract.Status == model.ContractStatusDraft {
		prev := contract.Status
		if err := s.contracts.UpdateStatus(ctx, tx, tenantID, contract.ID, &prev, model.ContractStatusInternalReview, &userID, now, nil); err != nil {
			return nil, internalError("move contract to internal review", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, internalError("commit workflow transaction", err)
	}
	if s.metrics != nil {
		s.metrics.WorkflowActive.Inc()
	}
	writeEvent(ctx, s.publisher, "lex-service", s.topic, "com.clario360.lex.contract.review_started", tenantID, &userID, map[string]any{
		"id":                   contract.ID,
		"workflow_instance_id": workflowID,
	}, s.logger)
	return &model.LegalWorkflowSummary{
		WorkflowInstanceID: workflowID,
		ContractID:         contract.ID,
		ContractTitle:      contract.Title,
		ContractStatus:     model.ContractStatusInternalReview,
		WorkflowStatus:     instance.Status,
		CurrentStepID:      instance.CurrentStepID,
		StartedAt:          now,
		TaskStatus:         &task.Status,
	}, nil
}

func (s *WorkflowService) ListActive(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]model.LegalWorkflowSummary, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 25
	}
	var total int
	if err := s.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM workflow_instances wi
		JOIN contracts c ON c.workflow_instance_id = wi.id
		WHERE wi.tenant_id = $1 AND wi.status = 'running' AND c.deleted_at IS NULL`,
		tenantID,
	).Scan(&total); err != nil {
		return nil, internalError("count workflows", err)
	}
	if total == 0 {
		return []model.LegalWorkflowSummary{}, 0, nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT wi.id, c.id, c.title, c.status, wi.status, wi.current_step_id, wi.started_at,
		       wt.assignee_id, wt.assignee_role, wt.status
		FROM workflow_instances wi
		JOIN contracts c ON c.workflow_instance_id = wi.id
		LEFT JOIN LATERAL (
			SELECT assignee_id, assignee_role, status
			FROM workflow_tasks
			WHERE instance_id = wi.id
			ORDER BY created_at DESC
			LIMIT 1
		) wt ON true
		WHERE wi.tenant_id = $1 AND wi.status = 'running' AND c.deleted_at IS NULL
		ORDER BY wi.started_at DESC
		LIMIT $2 OFFSET $3`,
		tenantID, perPage, (page-1)*perPage,
	)
	if err != nil {
		return nil, 0, internalError("list workflows", err)
	}
	defer rows.Close()
	items := make([]model.LegalWorkflowSummary, 0)
	for rows.Next() {
		var item model.LegalWorkflowSummary
		var assigneeID *string
		if err := rows.Scan(&item.WorkflowInstanceID, &item.ContractID, &item.ContractTitle, &item.ContractStatus, &item.WorkflowStatus, &item.CurrentStepID, &item.StartedAt, &assigneeID, &item.AssigneeRole, &item.TaskStatus); err != nil {
			return nil, 0, internalError("scan workflows", err)
		}
		if assigneeID != nil {
			parsed := uuid.MustParse(*assigneeID)
			item.AssigneeID = &parsed
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (s *WorkflowService) AdvanceOnWorkflowCompletion(ctx context.Context, workflowInstanceID uuid.UUID) error {
	contract, err := s.contracts.GetByWorkflowInstance(ctx, workflowInstanceID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil
		}
		return internalError("load workflow contract", err)
	}
	if contract.Status != model.ContractStatusInternalReview {
		return nil
	}
	now := s.now().UTC()
	prev := contract.Status
	if err := s.contracts.UpdateStatus(ctx, s.db, contract.TenantID, contract.ID, &prev, model.ContractStatusLegalReview, nil, now, nil); err != nil {
		return internalError("advance contract after workflow completion", err)
	}
	return nil
}

func (s *WorkflowService) ensureReviewDefinition(ctx context.Context, tenantID, userID uuid.UUID) (*workflowmodel.WorkflowDefinition, error) {
	var existing workflowmodel.WorkflowDefinition
	err := s.db.QueryRow(ctx, `
		SELECT id, version
		FROM workflow_definitions
		WHERE tenant_id = $1 AND name = $2 AND status = 'active' AND deleted_at IS NULL
		ORDER BY version DESC
		LIMIT 1`,
		tenantID, legalReviewWorkflowName,
	).Scan(&existing.ID, &existing.Version)
	if err == nil {
		existing.TenantID = tenantID.String()
		return &existing, nil
	}
	if err != pgx.ErrNoRows {
		return nil, internalError("load workflow definition", err)
	}
	definition := &workflowmodel.WorkflowDefinition{
		ID:          uuid.NewString(),
		TenantID:    tenantID.String(),
		Name:        legalReviewWorkflowName,
		Description: "Legal contract review workflow for Clario Lex.",
		Version:     1,
		Status:      workflowmodel.DefinitionStatusActive,
		TriggerConfig: workflowmodel.TriggerConfig{
			Type: workflowmodel.TriggerTypeManual,
		},
		Variables: map[string]workflowmodel.VariableDef{
			"contract_id":    {Type: "string"},
			"contract_title": {Type: "string"},
			"contract_type":  {Type: "string"},
		},
		Steps: []workflowmodel.StepDefinition{
			{ID: "legal_review", Type: workflowmodel.StepTypeHumanTask, Name: "Legal Review", Config: map[string]any{"role": "legal"}, Transitions: []workflowmodel.Transition{{Target: "end"}}},
			{ID: "end", Type: workflowmodel.StepTypeEnd, Name: "Completed", Config: map[string]any{}, Transitions: nil},
		},
		CreatedBy: userID.String(),
	}
	if err := s.defRepo.Create(ctx, definition); err != nil {
		return nil, internalError("create workflow definition", err)
	}
	return definition, nil
}

func ptrString(value string) *string {
	return &value
}
