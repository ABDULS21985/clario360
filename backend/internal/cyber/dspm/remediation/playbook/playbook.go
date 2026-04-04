package playbook

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// PlaybookExecutor orchestrates the step-by-step execution of a remediation playbook.
// Each step is dispatched to the appropriate StepExecutor based on its action type.
type PlaybookExecutor struct {
	registry  *Registry
	executors map[model.StepAction]StepExecutor
	logger    zerolog.Logger
}

// NewPlaybookExecutor creates a PlaybookExecutor wired with all step executors from the registry.
func NewPlaybookExecutor(registry *Registry, logger zerolog.Logger) *PlaybookExecutor {
	l := logger.With().Str("component", "playbook_executor").Logger()
	return &PlaybookExecutor{
		registry:  registry,
		logger:    l,
		executors: buildExecutorMap(l),
	}
}

// buildExecutorMap constructs the mapping of every StepAction to its concrete executor.
func buildExecutorMap(logger zerolog.Logger) map[model.StepAction]StepExecutor {
	return map[model.StepAction]StepExecutor{
		model.StepActionEncryptAtRest:    NewEncryptAtRestExecutor(logger),
		model.StepActionEncryptInTransit: NewEncryptInTransitExecutor(logger),
		model.StepActionRevokeAccess:     NewRevokeAccessExecutor(logger),
		model.StepActionDowngradeAccess:  NewDowngradeAccessExecutor(logger),
		model.StepActionRestrictNetwork:  NewRestrictNetworkExecutor(logger),
		model.StepActionEnableAuditLog:   NewEnableAuditLogExecutor(logger),
		model.StepActionConfigureBackup:  NewConfigureBackupExecutor(logger),
		model.StepActionCreateTicket:     NewCreateTicketExecutor(logger),
		model.StepActionNotifyOwner:      NewNotifyOwnerExecutor(logger),
		model.StepActionQuarantine:       NewQuarantineExecutor(logger),
		model.StepActionReclassify:       NewReclassifyExecutor(logger),
		model.StepActionScheduleReview:   NewScheduleReviewExecutor(logger),
		model.StepActionArchiveData:      NewArchiveDataExecutor(logger),
		model.StepActionDeleteData:       NewDeleteDataExecutor(logger),
	}
}

// Execute runs a single step within a playbook, identified by stepIndex (0-based).
// It respects the step's configured timeout via context deadline and returns a StepResult
// with timing information regardless of success or failure.
func (pe *PlaybookExecutor) Execute(ctx context.Context, playbook *model.Playbook, stepIndex int) (*model.StepResult, error) {
	if playbook == nil {
		return nil, fmt.Errorf("playbook is nil")
	}
	if stepIndex < 0 || stepIndex >= len(playbook.Steps) {
		return nil, fmt.Errorf("step index %d out of range [0, %d)", stepIndex, len(playbook.Steps))
	}

	step := &playbook.Steps[stepIndex]

	pe.logger.Info().
		Str("playbook_id", playbook.ID).
		Str("step_id", step.ID).
		Int("step_order", step.Order).
		Str("action", string(step.Action)).
		Msg("executing playbook step")

	executor, ok := pe.executors[step.Action]
	if !ok {
		start := time.Now()
		now := time.Now()
		return &model.StepResult{
			StepID:      step.ID,
			Action:      string(step.Action),
			Status:      model.StepStatusFailed,
			StartedAt:   start,
			CompletedAt: &now,
			DurationMs:  0,
			Error:       fmt.Sprintf("no executor registered for action %q", step.Action),
		}, nil
	}

	// Apply step-level timeout if configured and no tighter deadline exists on the context.
	stepCtx := ctx
	if step.Timeout > 0 {
		var cancel context.CancelFunc
		stepCtx, cancel = context.WithTimeout(ctx, step.Timeout)
		defer cancel()
	}

	result, err := executor.Execute(stepCtx, step)
	if err != nil {
		pe.logger.Error().
			Err(err).
			Str("playbook_id", playbook.ID).
			Str("step_id", step.ID).
			Str("action", string(step.Action)).
			Msg("step execution returned error")

		start := time.Now()
		now := time.Now()
		return &model.StepResult{
			StepID:      step.ID,
			Action:      string(step.Action),
			Status:      model.StepStatusFailed,
			StartedAt:   start,
			CompletedAt: &now,
			DurationMs:  0,
			Error:       err.Error(),
		}, nil
	}

	pe.logger.Info().
		Str("playbook_id", playbook.ID).
		Str("step_id", step.ID).
		Str("action", string(step.Action)).
		Str("status", string(result.Status)).
		Int64("duration_ms", result.DurationMs).
		Msg("step execution completed")

	return result, nil
}
