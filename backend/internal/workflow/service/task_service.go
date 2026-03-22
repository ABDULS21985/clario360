package service

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/workflow/model"
)

// TaskService manages human task operations including listing, claiming,
// completing, delegating, and rejecting tasks.
type TaskService struct {
	taskRepo taskRepo
	engine   *EngineService
	logger   zerolog.Logger
}

// NewTaskService creates a new TaskService.
func NewTaskService(taskRepo taskRepo, engine *EngineService, logger zerolog.Logger) *TaskService {
	return &TaskService{
		taskRepo: taskRepo,
		engine:   engine,
		logger:   logger.With().Str("service", "workflow-task").Logger(),
	}
}

// ListTasks returns a paginated list of tasks visible to the specified user,
// filtered by role assignments and optional status filter.
func (s *TaskService) ListTasks(ctx context.Context, tenantID, userID string, roles []string, statuses []string, sortBy, sortOrder string, page, pageSize int) ([]*model.HumanTask, int, error) {
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

	tasks, total, err := s.taskRepo.ListForUser(ctx, tenantID, userID, roles, statuses, sortBy, sortOrder, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("listing tasks: %w", err)
	}
	return tasks, total, nil
}

// GetTask retrieves a single human task by ID.
func (s *TaskService) GetTask(ctx context.Context, tenantID, taskID string) (*model.HumanTask, error) {
	task, err := s.taskRepo.GetByID(ctx, tenantID, taskID)
	if err != nil {
		return nil, fmt.Errorf("getting task: %w", err)
	}
	return task, nil
}

// ClaimTask assigns a task to the requesting user. The task must be in pending
// status and the user must be eligible based on role assignment rules.
func (s *TaskService) ClaimTask(ctx context.Context, tenantID, taskID, userID string) error {
	// Load the task to validate assignment rules.
	task, err := s.taskRepo.GetByID(ctx, tenantID, taskID)
	if err != nil {
		return fmt.Errorf("loading task for claim: %w", err)
	}

	// Task must be claimable (pending status).
	if !task.IsClaimable() {
		return fmt.Errorf("task %s is not in a claimable state (status: %s)", taskID, task.Status)
	}

	// If the task is assigned to a specific user, only that user can claim it.
	if task.AssigneeID != nil && *task.AssigneeID != "" {
		if *task.AssigneeID != userID {
			return fmt.Errorf("task %s is assigned to a specific user and cannot be claimed by %s", taskID, userID)
		}
	}

	if err := s.taskRepo.ClaimTask(ctx, tenantID, taskID, userID); err != nil {
		return fmt.Errorf("claiming task: %w", err)
	}

	s.logger.Info().
		Str("task_id", taskID).
		Str("claimed_by", userID).
		Str("tenant_id", tenantID).
		Msg("task claimed")

	return nil
}

// CompleteTask marks a task as completed with the provided form data and
// resumes the workflow from the task's step.
func (s *TaskService) CompleteTask(ctx context.Context, tenantID, taskID, userID string, formData map[string]interface{}) error {
	// 1. Load and validate the task.
	task, err := s.taskRepo.GetByID(ctx, tenantID, taskID)
	if err != nil {
		return fmt.Errorf("loading task for completion: %w", err)
	}

	// Task must be completable (claimed status).
	if !task.IsCompletable() {
		return fmt.Errorf("task %s is not in a completable state (status: %s)", taskID, task.Status)
	}

	// Verify the claiming user is the one completing it.
	if task.ClaimedBy == nil || *task.ClaimedBy != userID {
		return fmt.Errorf("task %s is not claimed by user %s", taskID, userID)
	}

	// 2. Validate form data against form schema.
	if err := s.validateFormData(task.FormSchema, formData); err != nil {
		return fmt.Errorf("form validation failed: %w", err)
	}

	// 3. Complete the task in the repository.
	if err := s.taskRepo.CompleteTask(ctx, tenantID, taskID, formData); err != nil {
		return fmt.Errorf("completing task: %w", err)
	}

	s.logger.Info().
		Str("task_id", taskID).
		Str("completed_by", userID).
		Str("tenant_id", tenantID).
		Str("instance_id", task.InstanceID).
		Msg("task completed")

	// 4. Resume the workflow engine.
	// Reload the task to get the updated form_data.
	task.FormData = formData
	task.Status = model.TaskStatusCompleted

	if err := s.engine.ResumeFromTask(ctx, task); err != nil {
		s.logger.Error().Err(err).
			Str("task_id", taskID).
			Str("instance_id", task.InstanceID).
			Msg("failed to resume workflow from task")
		return fmt.Errorf("resuming workflow from task: %w", err)
	}

	return nil
}

// DelegateTask transfers a task from one user to another.
func (s *TaskService) DelegateTask(ctx context.Context, tenantID, taskID, fromUserID, toUserID, reason string) error {
	// Validate the task exists and the delegating user owns it.
	task, err := s.taskRepo.GetByID(ctx, tenantID, taskID)
	if err != nil {
		return fmt.Errorf("loading task for delegation: %w", err)
	}

	// Task must be in pending or claimed status to delegate.
	if task.Status != model.TaskStatusPending && task.Status != model.TaskStatusClaimed {
		return fmt.Errorf("task %s cannot be delegated in status: %s", taskID, task.Status)
	}

	// If claimed, only the claiming user can delegate.
	if task.Status == model.TaskStatusClaimed {
		if task.ClaimedBy == nil || *task.ClaimedBy != fromUserID {
			return fmt.Errorf("task %s is claimed by another user and cannot be delegated by %s", taskID, fromUserID)
		}
	}

	if fromUserID == toUserID {
		return fmt.Errorf("cannot delegate task to the same user")
	}

	if err := s.taskRepo.DelegateTask(ctx, tenantID, taskID, fromUserID, toUserID, reason); err != nil {
		return fmt.Errorf("delegating task: %w", err)
	}

	s.logger.Info().
		Str("task_id", taskID).
		Str("from_user", fromUserID).
		Str("to_user", toUserID).
		Str("tenant_id", tenantID).
		Msg("task delegated")

	return nil
}

// RejectTask rejects a task with a reason. The task must be claimed by the rejecting user.
func (s *TaskService) RejectTask(ctx context.Context, tenantID, taskID, userID, reason string) error {
	task, err := s.taskRepo.GetByID(ctx, tenantID, taskID)
	if err != nil {
		return fmt.Errorf("loading task for rejection: %w", err)
	}

	if task.Status != model.TaskStatusClaimed {
		return fmt.Errorf("only claimed tasks can be rejected, current status: %s", task.Status)
	}

	if task.ClaimedBy == nil || *task.ClaimedBy != userID {
		return fmt.Errorf("task %s is not claimed by user %s", taskID, userID)
	}

	if reason == "" {
		return fmt.Errorf("rejection reason is required")
	}

	if err := s.taskRepo.RejectTask(ctx, tenantID, taskID, userID, reason); err != nil {
		return fmt.Errorf("rejecting task: %w", err)
	}

	s.logger.Info().
		Str("task_id", taskID).
		Str("rejected_by", userID).
		Str("reason", reason).
		Str("tenant_id", tenantID).
		Msg("task rejected")

	return nil
}

// UpdateMetadata persists updated metadata for a task.
func (s *TaskService) UpdateMetadata(ctx context.Context, tenantID, taskID string, metadata map[string]interface{}) error {
	if err := s.taskRepo.UpdateMetadata(ctx, tenantID, taskID, metadata); err != nil {
		return fmt.Errorf("updating task metadata: %w", err)
	}
	return nil
}

// CountTasks returns task counts bucketed by status for the user's dashboard.
func (s *TaskService) CountTasks(ctx context.Context, tenantID, userID string, roles []string) (map[string]int, error) {
	counts, err := s.taskRepo.CountByStatus(ctx, tenantID, userID, roles)
	if err != nil {
		return nil, fmt.Errorf("counting tasks by status: %w", err)
	}
	return counts, nil
}

// DailyCreatedCounts returns a zero-filled daily count of tasks created over
// the last N days, suitable for KPI sparklines.
func (s *TaskService) DailyCreatedCounts(ctx context.Context, tenantID string, days int) ([]int, error) {
	return s.taskRepo.DailyCreatedCounts(ctx, tenantID, days)
}

// validateFormData validates the submitted form data against the task's form schema.
// It checks that all required fields are present and that field types match.
func (s *TaskService) validateFormData(schema []model.FormField, formData map[string]interface{}) error {
	if len(schema) == 0 {
		return nil
	}

	for _, field := range schema {
		val, exists := formData[field.Name]

		// Check required fields.
		if field.Required && (!exists || val == nil) {
			return fmt.Errorf("required field '%s' is missing", field.Name)
		}

		if !exists || val == nil {
			continue
		}

		// Validate field types.
		if err := validateFieldType(field.Name, field.Type, val); err != nil {
			return err
		}

		// Validate select options if applicable.
		if field.Type == "select" && len(field.Options) > 0 {
			strVal, ok := val.(string)
			if ok {
				valid := false
				for _, opt := range field.Options {
					if opt == strVal {
						valid = true
						break
					}
				}
				if !valid {
					return fmt.Errorf("field '%s' value '%s' is not a valid option", field.Name, strVal)
				}
			}
		}
	}

	return nil
}

// validateFieldType checks that a form field value matches its declared type.
func validateFieldType(fieldName, fieldType string, value interface{}) error {
	switch fieldType {
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field '%s' must be a boolean", fieldName)
		}
	case "text", "textarea", "date":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' must be a string", fieldName)
		}
	case "select":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' must be a string", fieldName)
		}
	case "number":
		switch value.(type) {
		case float64, int, int64, float32:
			// Valid number types.
		default:
			return fmt.Errorf("field '%s' must be a number", fieldName)
		}
	}
	return nil
}
