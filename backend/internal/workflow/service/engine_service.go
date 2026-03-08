package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/workflow/dto"
	"github.com/clario360/platform/internal/workflow/executor"
	"github.com/clario360/platform/internal/workflow/expression"
	"github.com/clario360/platform/internal/workflow/model"
)

// instanceRepo defines the persistence operations for workflow instances.
type instanceRepo interface {
	Create(ctx context.Context, inst *model.WorkflowInstance) error
	GetByID(ctx context.Context, tenantID, id string) (*model.WorkflowInstance, error)
	UpdateWithLock(ctx context.Context, inst *model.WorkflowInstance) error
	List(ctx context.Context, tenantID, status, definitionID, startedBy string, dateFrom, dateTo *time.Time, limit, offset int) ([]*model.WorkflowInstance, int, error)
	ListRunning(ctx context.Context, limit, offset int) ([]*model.WorkflowInstance, error)
	CreateStepExecution(ctx context.Context, exec *model.StepExecution) error
	UpdateStepExecution(ctx context.Context, exec *model.StepExecution) error
	GetStepExecutions(ctx context.Context, instanceID string) ([]*model.StepExecution, error)
	GetLastFailedStep(ctx context.Context, instanceID string) (*model.StepExecution, error)
}

// taskRepo defines the persistence operations for human tasks.
type taskRepo interface {
	Create(ctx context.Context, task *model.HumanTask) error
	GetByID(ctx context.Context, tenantID, id string) (*model.HumanTask, error)
	ListForUser(ctx context.Context, tenantID, userID string, roles []string, statuses []string, limit, offset int) ([]*model.HumanTask, int, error)
	ClaimTask(ctx context.Context, tenantID, taskID, userID string) error
	CompleteTask(ctx context.Context, tenantID, taskID string, formData map[string]interface{}) error
	DelegateTask(ctx context.Context, tenantID, taskID, fromUserID, toUserID string) error
	RejectTask(ctx context.Context, tenantID, taskID, userID, reason string) error
	CountByStatus(ctx context.Context, tenantID, userID string, roles []string) (map[string]int, error)
	GetOverdueTasks(ctx context.Context, limit int) ([]*model.HumanTask, error)
	MarkSLABreached(ctx context.Context, taskID string) error
	EscalateTask(ctx context.Context, taskID, escalationRole string) error
	CancelByInstance(ctx context.Context, instanceID string) error
}

// executorRegistry dispatches step execution to the appropriate executor.
type executorRegistry interface {
	Execute(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition, exec *model.StepExecution) (*executor.ExecutionResult, error)
}

// eventPublisher defines the interface for publishing events.
type eventPublisher interface {
	Publish(ctx context.Context, topic string, event *events.Event) error
}

// EngineService is the core workflow engine that orchestrates instance lifecycle,
// step execution, transition evaluation, and state management.
type EngineService struct {
	instanceRepo instanceRepo
	defRepo      definitionRepo
	taskRepo     taskRepo
	executors    executorRegistry
	evaluator    *expression.Evaluator
	resolver     *expression.VariableResolver
	producer     eventPublisher
	logger       zerolog.Logger
}

// NewEngineService creates a new EngineService with all required dependencies.
func NewEngineService(
	instanceRepo instanceRepo,
	defRepo definitionRepo,
	taskRepo taskRepo,
	executors executorRegistry,
	producer eventPublisher,
	logger zerolog.Logger,
) *EngineService {
	return &EngineService{
		instanceRepo: instanceRepo,
		defRepo:      defRepo,
		taskRepo:     taskRepo,
		executors:    executors,
		evaluator:    expression.NewEvaluator(),
		resolver:     expression.NewVariableResolver(),
		producer:     producer,
		logger:       logger.With().Str("service", "workflow-engine").Logger(),
	}
}

// StartInstance creates and begins executing a new workflow instance.
func (s *EngineService) StartInstance(ctx context.Context, tenantID, userID string, req dto.StartInstanceRequest) (*model.WorkflowInstance, error) {
	// 1. Load definition (must be active).
	def, err := s.defRepo.GetActiveByID(ctx, tenantID, req.DefinitionID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, fmt.Errorf("workflow definition %s not found or not active", req.DefinitionID)
		}
		return nil, fmt.Errorf("loading workflow definition: %w", err)
	}

	if def.Status != model.DefinitionStatusActive {
		return nil, fmt.Errorf("workflow definition %s is not active (status: %s)", def.ID, def.Status)
	}

	// 2. Create instance.
	now := time.Now().UTC()
	firstStepID := ""
	if len(def.Steps) > 0 {
		firstStepID = def.Steps[0].ID
	}

	inst := &model.WorkflowInstance{
		ID:            generateUUID(),
		TenantID:      tenantID,
		DefinitionID:  def.ID,
		DefinitionVer: def.Version,
		Status:        model.InstanceStatusRunning,
		CurrentStepID: &firstStepID,
		Variables:     make(map[string]interface{}),
		StepOutputs:   make(map[string]interface{}),
		TriggerData:   req.TriggerData,
		StartedBy:     &userID,
		StartedAt:     now,
		UpdatedAt:     now,
		LockVersion:   0,
	}

	// 3. Resolve initial variables from trigger data and defaults.
	inst.Variables = s.resolveInitialVariables(def.Variables, req.InputVariables, req.TriggerData)

	if err := s.instanceRepo.Create(ctx, inst); err != nil {
		return nil, fmt.Errorf("creating workflow instance: %w", err)
	}

	s.logger.Info().
		Str("instance_id", inst.ID).
		Str("definition_id", def.ID).
		Str("tenant_id", tenantID).
		Str("started_by", userID).
		Msg("workflow instance started")

	// 4. Publish workflow.instance.started event.
	s.publishEvent(ctx, "workflow.instance.started", tenantID, map[string]interface{}{
		"instance_id":   inst.ID,
		"definition_id": def.ID,
		"started_by":    userID,
	})

	// 5. Advance workflow from the first step.
	if firstStepID != "" {
		if err := s.executeStep(ctx, inst, def, firstStepID); err != nil {
			s.logger.Error().Err(err).
				Str("instance_id", inst.ID).
				Str("step_id", firstStepID).
				Msg("failed to execute first step")
			// Do not fail the whole start; the instance is created and can be retried.
		}
	}

	return inst, nil
}

// AdvanceWorkflow moves a workflow instance forward from a given step.
func (s *EngineService) AdvanceWorkflow(ctx context.Context, instanceID, fromStepID string) error {
	// Load instance without tenant filter (called internally).
	inst, err := s.instanceRepo.GetByID(ctx, "", instanceID)
	if err != nil {
		return fmt.Errorf("loading instance for advance: %w", err)
	}

	// If status is not running, do nothing.
	if !inst.IsRunnable() {
		s.logger.Debug().
			Str("instance_id", instanceID).
			Str("status", inst.Status).
			Msg("instance is not runnable, skipping advance")
		return nil
	}

	// Load definition.
	def, err := s.defRepo.GetByID(ctx, inst.TenantID, inst.DefinitionID)
	if err != nil {
		return fmt.Errorf("loading definition for advance: %w", err)
	}

	// Determine next step by evaluating transitions from fromStepID.
	nextStepID, err := s.evaluateTransitions(def.Steps, fromStepID, inst)
	if err != nil {
		return fmt.Errorf("evaluating transitions from step %s: %w", fromStepID, err)
	}

	if nextStepID == "" {
		// No transitions found; this shouldn't happen in a well-formed workflow.
		s.logger.Warn().
			Str("instance_id", instanceID).
			Str("from_step", fromStepID).
			Msg("no matching transition found")
		return nil
	}

	// Check if the next step is "end".
	nextStep := findStep(def.Steps, nextStepID)
	if nextStep == nil {
		return fmt.Errorf("transition target step %s not found in definition", nextStepID)
	}

	if nextStep.Type == model.StepTypeEnd {
		return s.completeInstance(ctx, inst, nextStepID)
	}

	// Execute the next step.
	return s.executeStep(ctx, inst, def, nextStepID)
}

// executeStep creates a step execution record and dispatches to the executor.
func (s *EngineService) executeStep(ctx context.Context, inst *model.WorkflowInstance, def *model.WorkflowDefinition, stepID string) error {
	step := findStep(def.Steps, stepID)
	if step == nil {
		return fmt.Errorf("step %s not found in definition %s", stepID, def.ID)
	}

	// Create step execution record.
	now := time.Now().UTC()
	stepExec := &model.StepExecution{
		ID:         generateUUID(),
		InstanceID: inst.ID,
		StepID:     stepID,
		StepType:   step.Type,
		Status:     model.StepStatusRunning,
		Attempt:    1,
		StartedAt:  &now,
		CreatedAt:  now,
	}

	// Serialize the step config as input data.
	inputData, _ := json.Marshal(step.Config)
	stepExec.InputData = inputData

	if err := s.instanceRepo.CreateStepExecution(ctx, stepExec); err != nil {
		return fmt.Errorf("creating step execution: %w", err)
	}

	// Update instance current step.
	inst.CurrentStepID = &stepID
	inst.UpdatedAt = time.Now().UTC()
	if err := s.instanceRepo.UpdateWithLock(ctx, inst); err != nil {
		s.logger.Error().Err(err).
			Str("instance_id", inst.ID).
			Str("step_id", stepID).
			Msg("failed to update instance current step")
	}

	// Dispatch to executor.
	result, err := s.executors.Execute(ctx, inst, step, stepExec)
	if err != nil {
		if errors.Is(err, executor.ErrParked) {
			// Step is parked (waiting for external signal).
			stepExec.Status = model.StepStatusRunning
			completedAt := time.Now().UTC()
			stepExec.CompletedAt = nil
			_ = s.instanceRepo.UpdateStepExecution(ctx, stepExec)

			s.logger.Info().
				Str("instance_id", inst.ID).
				Str("step_id", stepID).
				Str("step_type", step.Type).
				Msg("step parked, waiting for external completion")

			// Store partial output if the result contains data.
			if result != nil && result.Output != nil {
				s.storeStepOutput(ctx, inst, stepID, result.Output)
			}
			_ = completedAt // suppress unused
			return nil
		}

		// Execution failed.
		return s.handleStepFailure(ctx, inst, def, step, stepExec, err)
	}

	// Success: store output and advance.
	completedAt := time.Now().UTC()
	stepExec.Status = model.StepStatusCompleted
	stepExec.CompletedAt = &completedAt

	if result != nil && result.Output != nil {
		outputData, _ := json.Marshal(result.Output)
		stepExec.OutputData = outputData
		s.storeStepOutput(ctx, inst, stepID, result.Output)
	}

	if err := s.instanceRepo.UpdateStepExecution(ctx, stepExec); err != nil {
		s.logger.Error().Err(err).
			Str("instance_id", inst.ID).
			Str("step_id", stepID).
			Msg("failed to update step execution on completion")
	}

	s.logger.Info().
		Str("instance_id", inst.ID).
		Str("step_id", stepID).
		Str("step_type", step.Type).
		Msg("step completed successfully")

	// Check if result indicates parked (executor returned success but with Parked flag).
	if result != nil && result.Parked {
		s.logger.Info().
			Str("instance_id", inst.ID).
			Str("step_id", stepID).
			Msg("step returned parked status, waiting for external signal")
		return nil
	}

	// Advance to next step.
	return s.AdvanceWorkflow(ctx, inst.ID, stepID)
}

// handleStepFailure handles a failed step execution, including retry logic.
func (s *EngineService) handleStepFailure(ctx context.Context, inst *model.WorkflowInstance, def *model.WorkflowDefinition, step *model.StepDefinition, stepExec *model.StepExecution, execErr error) error {
	completedAt := time.Now().UTC()
	errMsg := execErr.Error()
	stepExec.Status = model.StepStatusFailed
	stepExec.CompletedAt = &completedAt
	stepExec.ErrorMessage = &errMsg

	if err := s.instanceRepo.UpdateStepExecution(ctx, stepExec); err != nil {
		s.logger.Error().Err(err).
			Str("instance_id", inst.ID).
			Str("step_id", step.ID).
			Msg("failed to update step execution on failure")
	}

	// Check retry configuration.
	maxRetries := 0
	if v, ok := step.Config["max_retries"]; ok {
		switch rv := v.(type) {
		case float64:
			maxRetries = int(rv)
		case int:
			maxRetries = rv
		}
	}

	if stepExec.Attempt < maxRetries {
		// Retry the step.
		s.logger.Info().
			Str("instance_id", inst.ID).
			Str("step_id", step.ID).
			Int("attempt", stepExec.Attempt).
			Int("max_retries", maxRetries).
			Msg("retrying failed step")

		now := time.Now().UTC()
		retryExec := &model.StepExecution{
			ID:         generateUUID(),
			InstanceID: inst.ID,
			StepID:     step.ID,
			StepType:   step.Type,
			Status:     model.StepStatusRunning,
			Attempt:    stepExec.Attempt + 1,
			StartedAt:  &now,
			CreatedAt:  now,
		}

		inputData, _ := json.Marshal(step.Config)
		retryExec.InputData = inputData

		if err := s.instanceRepo.CreateStepExecution(ctx, retryExec); err != nil {
			return fmt.Errorf("creating retry step execution: %w", err)
		}

		result, err := s.executors.Execute(ctx, inst, step, retryExec)
		if err != nil {
			if errors.Is(err, executor.ErrParked) {
				if result != nil && result.Output != nil {
					s.storeStepOutput(ctx, inst, step.ID, result.Output)
				}
				return nil
			}
			// Recursive retry (with incremented attempt counter).
			return s.handleStepFailure(ctx, inst, def, step, retryExec, err)
		}

		// Retry succeeded.
		retryCompletedAt := time.Now().UTC()
		retryExec.Status = model.StepStatusCompleted
		retryExec.CompletedAt = &retryCompletedAt
		if result != nil && result.Output != nil {
			outputData, _ := json.Marshal(result.Output)
			retryExec.OutputData = outputData
			s.storeStepOutput(ctx, inst, step.ID, result.Output)
		}
		_ = s.instanceRepo.UpdateStepExecution(ctx, retryExec)

		if result != nil && result.Parked {
			return nil
		}

		return s.AdvanceWorkflow(ctx, inst.ID, step.ID)
	}

	// All retries exhausted; fail the instance.
	return s.failInstance(ctx, inst, fmt.Sprintf("step %s failed after %d attempts: %s", step.ID, stepExec.Attempt, execErr.Error()))
}

// ResumeFromTask is called when a human task is completed, storing the form data
// as step output and advancing the workflow.
func (s *EngineService) ResumeFromTask(ctx context.Context, task *model.HumanTask) error {
	// Load the instance.
	inst, err := s.instanceRepo.GetByID(ctx, task.TenantID, task.InstanceID)
	if err != nil {
		return fmt.Errorf("loading instance for task resume: %w", err)
	}

	if !inst.IsRunnable() {
		return fmt.Errorf("instance %s is not in a runnable state (status: %s)", inst.ID, inst.Status)
	}

	// Store the task form_data as step output.
	if task.FormData != nil {
		s.storeStepOutput(ctx, inst, task.StepID, task.FormData)
	}

	// Find and complete the step execution.
	executions, err := s.instanceRepo.GetStepExecutions(ctx, inst.ID)
	if err == nil {
		for _, exec := range executions {
			if exec.StepID == task.StepID && exec.Status == model.StepStatusRunning {
				completedAt := time.Now().UTC()
				exec.Status = model.StepStatusCompleted
				exec.CompletedAt = &completedAt
				if task.FormData != nil {
					outputData, _ := json.Marshal(task.FormData)
					exec.OutputData = outputData
				}
				_ = s.instanceRepo.UpdateStepExecution(ctx, exec)
				break
			}
		}
	}

	s.logger.Info().
		Str("instance_id", inst.ID).
		Str("task_id", task.ID).
		Str("step_id", task.StepID).
		Msg("resuming workflow from completed task")

	// Advance workflow from the task's step.
	return s.AdvanceWorkflow(ctx, inst.ID, task.StepID)
}

// RetryInstance retries a failed workflow instance from its last failed step.
func (s *EngineService) RetryInstance(ctx context.Context, tenantID, instanceID string) error {
	inst, err := s.instanceRepo.GetByID(ctx, tenantID, instanceID)
	if err != nil {
		return fmt.Errorf("loading instance for retry: %w", err)
	}

	if inst.Status != model.InstanceStatusFailed {
		return fmt.Errorf("only failed instances can be retried, current status: %s", inst.Status)
	}

	// Find the last failed step.
	failedStep, err := s.instanceRepo.GetLastFailedStep(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("finding last failed step: %w", err)
	}
	if failedStep == nil {
		return fmt.Errorf("no failed step found for instance %s", instanceID)
	}

	// Reset instance to running.
	inst.Status = model.InstanceStatusRunning
	inst.ErrorMessage = nil
	inst.CurrentStepID = &failedStep.StepID
	inst.UpdatedAt = time.Now().UTC()

	if err := s.instanceRepo.UpdateWithLock(ctx, inst); err != nil {
		return fmt.Errorf("resetting instance to running: %w", err)
	}

	s.logger.Info().
		Str("instance_id", instanceID).
		Str("retry_step_id", failedStep.StepID).
		Msg("retrying failed workflow instance")

	s.publishEvent(ctx, "workflow.instance.retried", tenantID, map[string]interface{}{
		"instance_id": instanceID,
		"step_id":     failedStep.StepID,
	})

	// Load definition and re-execute from failed step.
	def, err := s.defRepo.GetByID(ctx, inst.TenantID, inst.DefinitionID)
	if err != nil {
		return fmt.Errorf("loading definition for retry: %w", err)
	}

	return s.executeStep(ctx, inst, def, failedStep.StepID)
}

// CancelInstance cancels a running or suspended workflow instance and all pending tasks.
func (s *EngineService) CancelInstance(ctx context.Context, tenantID, instanceID string) error {
	inst, err := s.instanceRepo.GetByID(ctx, tenantID, instanceID)
	if err != nil {
		return fmt.Errorf("loading instance for cancellation: %w", err)
	}

	if inst.IsTerminal() {
		return fmt.Errorf("cannot cancel instance in terminal state: %s", inst.Status)
	}

	now := time.Now().UTC()
	inst.Status = model.InstanceStatusCancelled
	inst.CompletedAt = &now
	inst.UpdatedAt = now

	if err := s.instanceRepo.UpdateWithLock(ctx, inst); err != nil {
		return fmt.Errorf("cancelling instance: %w", err)
	}

	// Cancel any pending tasks for this instance.
	if err := s.taskRepo.CancelByInstance(ctx, instanceID); err != nil {
		s.logger.Error().Err(err).
			Str("instance_id", instanceID).
			Msg("failed to cancel pending tasks")
	}

	s.logger.Info().
		Str("instance_id", instanceID).
		Str("tenant_id", tenantID).
		Msg("workflow instance cancelled")

	s.publishEvent(ctx, "workflow.instance.cancelled", tenantID, map[string]interface{}{
		"instance_id": instanceID,
	})

	return nil
}

// SuspendInstance suspends a running workflow instance.
func (s *EngineService) SuspendInstance(ctx context.Context, tenantID, instanceID string) error {
	inst, err := s.instanceRepo.GetByID(ctx, tenantID, instanceID)
	if err != nil {
		return fmt.Errorf("loading instance for suspension: %w", err)
	}

	if inst.Status != model.InstanceStatusRunning {
		return fmt.Errorf("only running instances can be suspended, current status: %s", inst.Status)
	}

	inst.Status = model.InstanceStatusSuspended
	inst.UpdatedAt = time.Now().UTC()

	if err := s.instanceRepo.UpdateWithLock(ctx, inst); err != nil {
		return fmt.Errorf("suspending instance: %w", err)
	}

	s.logger.Info().
		Str("instance_id", instanceID).
		Str("tenant_id", tenantID).
		Msg("workflow instance suspended")

	s.publishEvent(ctx, "workflow.instance.suspended", tenantID, map[string]interface{}{
		"instance_id": instanceID,
	})

	return nil
}

// ResumeInstance resumes a suspended workflow instance and continues execution.
func (s *EngineService) ResumeInstance(ctx context.Context, tenantID, instanceID string) error {
	inst, err := s.instanceRepo.GetByID(ctx, tenantID, instanceID)
	if err != nil {
		return fmt.Errorf("loading instance for resume: %w", err)
	}

	if inst.Status != model.InstanceStatusSuspended {
		return fmt.Errorf("only suspended instances can be resumed, current status: %s", inst.Status)
	}

	inst.Status = model.InstanceStatusRunning
	inst.UpdatedAt = time.Now().UTC()

	if err := s.instanceRepo.UpdateWithLock(ctx, inst); err != nil {
		return fmt.Errorf("resuming instance: %w", err)
	}

	s.logger.Info().
		Str("instance_id", instanceID).
		Str("tenant_id", tenantID).
		Msg("workflow instance resumed")

	s.publishEvent(ctx, "workflow.instance.resumed", tenantID, map[string]interface{}{
		"instance_id": instanceID,
	})

	// Continue from the current step if available.
	if inst.CurrentStepID != nil && *inst.CurrentStepID != "" {
		return s.AdvanceWorkflow(ctx, instanceID, *inst.CurrentStepID)
	}

	return nil
}

// GetHistory returns the step execution history for a workflow instance.
func (s *EngineService) GetHistory(ctx context.Context, tenantID, instanceID string) ([]*model.StepExecution, error) {
	// Verify the instance belongs to this tenant.
	_, err := s.instanceRepo.GetByID(ctx, tenantID, instanceID)
	if err != nil {
		return nil, fmt.Errorf("loading instance for history: %w", err)
	}

	executions, err := s.instanceRepo.GetStepExecutions(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("getting step executions: %w", err)
	}

	return executions, nil
}

// ListInstances returns a paginated list of workflow instances for a tenant.
func (s *EngineService) ListInstances(ctx context.Context, tenantID, status, definitionID, startedBy string, dateFrom, dateTo *time.Time, page, pageSize int) ([]*model.WorkflowInstance, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	limit := pageSize
	offset := (page - 1) * pageSize

	return s.instanceRepo.List(ctx, tenantID, status, definitionID, startedBy, dateFrom, dateTo, limit, offset)
}

// GetInstance retrieves a single workflow instance.
func (s *EngineService) GetInstance(ctx context.Context, tenantID, instanceID string) (*model.WorkflowInstance, error) {
	inst, err := s.instanceRepo.GetByID(ctx, tenantID, instanceID)
	if err != nil {
		return nil, fmt.Errorf("getting workflow instance: %w", err)
	}
	return inst, nil
}

// completeInstance marks a workflow instance as completed.
func (s *EngineService) completeInstance(ctx context.Context, inst *model.WorkflowInstance, endStepID string) error {
	now := time.Now().UTC()

	// Create step execution for the end step.
	endExec := &model.StepExecution{
		ID:          generateUUID(),
		InstanceID:  inst.ID,
		StepID:      endStepID,
		StepType:    model.StepTypeEnd,
		Status:      model.StepStatusCompleted,
		Attempt:     1,
		StartedAt:   &now,
		CompletedAt: &now,
		CreatedAt:   now,
	}
	if err := s.instanceRepo.CreateStepExecution(ctx, endExec); err != nil {
		s.logger.Error().Err(err).
			Str("instance_id", inst.ID).
			Msg("failed to create end step execution")
	}

	inst.Status = model.InstanceStatusCompleted
	inst.CurrentStepID = &endStepID
	inst.CompletedAt = &now
	inst.UpdatedAt = now

	if err := s.instanceRepo.UpdateWithLock(ctx, inst); err != nil {
		return fmt.Errorf("completing instance: %w", err)
	}

	s.logger.Info().
		Str("instance_id", inst.ID).
		Str("tenant_id", inst.TenantID).
		Str("definition_id", inst.DefinitionID).
		Msg("workflow instance completed")

	s.publishEvent(ctx, "workflow.instance.completed", inst.TenantID, map[string]interface{}{
		"instance_id":   inst.ID,
		"definition_id": inst.DefinitionID,
		"initiator_id":  inst.StartedBy,
	})

	return nil
}

// failInstance marks a workflow instance as failed with an error message.
func (s *EngineService) failInstance(ctx context.Context, inst *model.WorkflowInstance, errMsg string) error {
	now := time.Now().UTC()
	inst.Status = model.InstanceStatusFailed
	inst.ErrorMessage = &errMsg
	inst.CompletedAt = &now
	inst.UpdatedAt = now

	if err := s.instanceRepo.UpdateWithLock(ctx, inst); err != nil {
		return fmt.Errorf("failing instance: %w", err)
	}

	s.logger.Error().
		Str("instance_id", inst.ID).
		Str("tenant_id", inst.TenantID).
		Str("error", errMsg).
		Msg("workflow instance failed")

	s.publishEvent(ctx, "workflow.instance.failed", inst.TenantID, map[string]interface{}{
		"instance_id":  inst.ID,
		"error":        errMsg,
		"initiator_id": inst.StartedBy,
	})

	return nil
}

// evaluateTransitions finds the first matching transition from a given step
// by evaluating each transition's condition against the instance context.
func (s *EngineService) evaluateTransitions(steps []model.StepDefinition, fromStepID string, inst *model.WorkflowInstance) (string, error) {
	step := findStep(steps, fromStepID)
	if step == nil {
		return "", fmt.Errorf("step %s not found in definition", fromStepID)
	}

	if len(step.Transitions) == 0 {
		return "", nil
	}

	exprCtx := s.buildExpressionContext(inst)

	for _, t := range step.Transitions {
		// Unconditional transition (no condition means always true).
		if t.Condition == "" {
			return t.Target, nil
		}

		// Evaluate the condition expression.
		result, err := s.evaluator.Evaluate(t.Condition, exprCtx)
		if err != nil {
			s.logger.Warn().Err(err).
				Str("instance_id", inst.ID).
				Str("step_id", fromStepID).
				Str("condition", t.Condition).
				Str("target", t.Target).
				Msg("failed to evaluate transition condition, skipping")
			continue
		}

		if result {
			return t.Target, nil
		}
	}

	// No condition matched; try to find a default (unconditional) transition.
	// This handles the case where conditional transitions are listed first,
	// followed by a fallback.
	return "", fmt.Errorf("no matching transition from step %s", fromStepID)
}

// buildExpressionContext constructs the data context map used for expression
// evaluation and variable resolution within a workflow instance.
func (s *EngineService) buildExpressionContext(inst *model.WorkflowInstance) map[string]interface{} {
	ctx := map[string]interface{}{
		"variables": inst.Variables,
		"steps":     inst.StepOutputs,
	}

	// Add trigger data.
	triggerData := make(map[string]interface{})
	if inst.TriggerData != nil {
		var td map[string]interface{}
		if err := json.Unmarshal(inst.TriggerData, &td); err == nil {
			triggerData = td
		}
	}
	ctx["trigger"] = map[string]interface{}{
		"data": triggerData,
	}

	return ctx
}

// storeStepOutput stores the output of a step execution in the instance's step outputs map.
func (s *EngineService) storeStepOutput(ctx context.Context, inst *model.WorkflowInstance, stepID string, output map[string]interface{}) {
	if inst.StepOutputs == nil {
		inst.StepOutputs = make(map[string]interface{})
	}
	inst.StepOutputs[stepID] = map[string]interface{}{
		"output": output,
	}
	inst.UpdatedAt = time.Now().UTC()

	if err := s.instanceRepo.UpdateWithLock(ctx, inst); err != nil {
		s.logger.Error().Err(err).
			Str("instance_id", inst.ID).
			Str("step_id", stepID).
			Msg("failed to store step output")
	}
}

// resolveInitialVariables builds the initial variable map for a new instance
// by combining defaults, input overrides, and trigger data.
func (s *EngineService) resolveInitialVariables(
	defs map[string]model.VariableDef,
	inputVars map[string]interface{},
	triggerData json.RawMessage,
) map[string]interface{} {
	result := make(map[string]interface{})

	// Parse trigger data.
	var td map[string]interface{}
	if triggerData != nil {
		_ = json.Unmarshal(triggerData, &td)
	}

	for name, def := range defs {
		// Start with default value.
		if def.Default != nil {
			result[name] = def.Default
		}

		// Override with trigger data if source is specified.
		if def.Source != "" && td != nil {
			if val, ok := td[def.Source]; ok {
				result[name] = val
			}
		}

		// Override with explicitly provided input variables.
		if inputVars != nil {
			if val, ok := inputVars[name]; ok {
				result[name] = val
			}
		}
	}

	// Also include any input variables not declared in the definition.
	for name, val := range inputVars {
		if _, exists := result[name]; !exists {
			result[name] = val
		}
	}

	return result
}

// publishEvent publishes a workflow event if a producer is configured.
func (s *EngineService) publishEvent(ctx context.Context, eventType, tenantID string, data interface{}) {
	if s.producer == nil {
		return
	}

	evt, err := events.NewEvent(eventType, "workflow-engine", tenantID, data)
	if err != nil {
		s.logger.Error().Err(err).
			Str("event_type", eventType).
			Str("tenant_id", tenantID).
			Msg("failed to create workflow event")
		return
	}

	if err := s.producer.Publish(ctx, events.Topics.WorkflowEvents, evt); err != nil {
		s.logger.Error().Err(err).
			Str("event_type", eventType).
			Str("tenant_id", tenantID).
			Msg("failed to publish workflow event")
	}
}

// findStep looks up a step definition by ID within a slice.
func findStep(steps []model.StepDefinition, id string) *model.StepDefinition {
	for i := range steps {
		if steps[i].ID == id {
			return &steps[i]
		}
	}
	return nil
}
