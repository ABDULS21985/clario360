package remediation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/remediation/strategy"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/events"
)

const defaultRollbackWindowHours = 72

// RemediationExecutor orchestrates the dry-run → execute → verify → rollback lifecycle.
type RemediationExecutor struct {
	strategies  map[model.RemediationType]strategy.RemediationStrategy
	auditTrail  *AuditTrail
	remRepo     *repository.RemediationRepository
	producer    *events.Producer
	logger      zerolog.Logger
}

// NewRemediationExecutor creates a RemediationExecutor with all registered strategies.
func NewRemediationExecutor(
	strategies map[model.RemediationType]strategy.RemediationStrategy,
	auditTrail *AuditTrail,
	remRepo *repository.RemediationRepository,
	producer *events.Producer,
	logger zerolog.Logger,
) *RemediationExecutor {
	return &RemediationExecutor{
		strategies: strategies,
		auditTrail: auditTrail,
		remRepo:    remRepo,
		producer:   producer,
		logger:     logger.With().Str("component", "remediation-executor").Logger(),
	}
}

// DryRun executes a dry-run simulation for the remediation action.
func (e *RemediationExecutor) DryRun(ctx context.Context, action *model.RemediationAction, actorID *uuid.UUID, actorName string) (*model.DryRunResult, error) {
	if action.Status != model.StatusApproved {
		return nil, fmt.Errorf("%w: action must be in 'approved' state to start dry-run (current: %s)",
			ErrPreConditionFailed, action.Status)
	}

	strat, err := e.getStrategy(action.Type)
	if err != nil {
		return nil, err
	}

	// Transition to dry_run_running
	if err := e.remRepo.UpdateStatus(ctx, action.TenantID, action.ID, model.StatusDryRunRunning, map[string]interface{}{}); err != nil {
		return nil, fmt.Errorf("transition to dry_run_running: %w", err)
	}
	e.auditTrail.RecordTransition(ctx, action.TenantID, action.ID, "dry_run_started", actorID, actorName,
		model.StatusApproved, model.StatusDryRunRunning, nil)
	e.publishEvent(ctx, action.TenantID, "com.clario360.cyber.remediation.dry_run_started",
		map[string]interface{}{"id": action.ID})

	start := time.Now()
	result, dryRunErr := strat.DryRun(ctx, action)
	durationMs := time.Since(start).Milliseconds()

	if dryRunErr != nil {
		failResult := &model.DryRunResult{
			Success:  false,
			Blockers: []string{dryRunErr.Error()},
		}
		resultJSON, _ := json.Marshal(failResult)
		_ = e.remRepo.UpdateStatus(ctx, action.TenantID, action.ID, model.StatusDryRunFailed, map[string]interface{}{
			"dry_run_result":    resultJSON,
			"dry_run_at":        time.Now().UTC(),
			"dry_run_duration_ms": durationMs,
		})
		e.auditTrail.RecordTransition(ctx, action.TenantID, action.ID, "dry_run_failed", nil, "system",
			model.StatusDryRunRunning, model.StatusDryRunFailed, map[string]interface{}{"error": dryRunErr.Error()})
		e.publishEvent(ctx, action.TenantID, "com.clario360.cyber.remediation.dry_run_failed",
			map[string]interface{}{"id": action.ID, "error": dryRunErr.Error()})
		return nil, dryRunErr
	}

	resultJSON, _ := json.Marshal(result)
	newStatus := model.StatusDryRunCompleted
	if !result.Success {
		newStatus = model.StatusDryRunFailed
	}
	_ = e.remRepo.UpdateStatus(ctx, action.TenantID, action.ID, newStatus, map[string]interface{}{
		"dry_run_result":    resultJSON,
		"dry_run_at":        time.Now().UTC(),
		"dry_run_duration_ms": durationMs,
	})

	transitionAction := "dry_run_completed"
	if !result.Success {
		transitionAction = "dry_run_failed"
	}
	e.auditTrail.RecordTransition(ctx, action.TenantID, action.ID, transitionAction, nil, "system",
		model.StatusDryRunRunning, newStatus, map[string]interface{}{
			"success":        result.Success,
			"warnings_count": len(result.Warnings),
			"blockers_count": len(result.Blockers),
		})
	e.publishEvent(ctx, action.TenantID, "com.clario360.cyber.remediation.dry_run_completed",
		map[string]interface{}{"id": action.ID, "success": result.Success, "warnings_count": len(result.Warnings)})

	return result, nil
}

// Execute performs the remediation execution after governance checks.
func (e *RemediationExecutor) Execute(ctx context.Context, action *model.RemediationAction, executedBy uuid.UUID, executorName string) (*model.ExecutionResult, error) {
	// Governance validation: must have approval and completed successful dry-run
	if err := checkExecutePreConditions(action); err != nil {
		return nil, err
	}
	if action.Status != model.StatusDryRunCompleted && action.Status != model.StatusExecutionPending {
		return nil, fmt.Errorf("%w: status must be 'dry_run_completed' or 'execution_pending' (current: %s)",
			ErrPreConditionFailed, action.Status)
	}

	strat, err := e.getStrategy(action.Type)
	if err != nil {
		return nil, err
	}

	// CAPTURE PRE-EXECUTION STATE (CRITICAL for rollback)
	preState, err := strat.CaptureState(ctx, action)
	if err != nil {
		e.logger.Warn().Err(err).Str("id", action.ID.String()).Msg("failed to capture pre-execution state")
		preState = json.RawMessage(`{"error": "state capture failed"}`)
	}

	rollbackDeadline := time.Now().UTC().Add(defaultRollbackWindowHours * time.Hour)
	now := time.Now().UTC()
	_ = e.remRepo.UpdateStatus(ctx, action.TenantID, action.ID, model.StatusExecuting, map[string]interface{}{
		"pre_execution_state":  preState,
		"executed_by":          executedBy,
		"execution_started_at": now,
		"rollback_deadline":    rollbackDeadline,
	})
	e.auditTrail.RecordTransition(ctx, action.TenantID, action.ID, "execution_started", &executedBy, executorName,
		action.Status, model.StatusExecuting, map[string]interface{}{"steps_total": len(action.Plan.Steps)})
	e.publishEvent(ctx, action.TenantID, "com.clario360.cyber.remediation.execution_started",
		map[string]interface{}{"id": action.ID, "executed_by": executedBy, "steps_total": len(action.Plan.Steps)})

	// Reload with pre_execution_state set
	action.PreExecutionState = preState

	start := time.Now()
	result, execErr := strat.Execute(ctx, action)
	durationMs := time.Since(start).Milliseconds()

	if execErr != nil || (result != nil && !result.Success) {
		errMsg := ""
		if execErr != nil {
			errMsg = execErr.Error()
		} else if result != nil {
			for _, sr := range result.StepResults {
				if sr.Status == "failure" {
					errMsg = sr.Error
					break
				}
			}
		}

		var resultJSON []byte
		if result != nil {
			resultJSON, _ = json.Marshal(result)
		}
		_ = e.remRepo.UpdateStatus(ctx, action.TenantID, action.ID, model.StatusExecutionFailed, map[string]interface{}{
			"execution_result":       resultJSON,
			"execution_completed_at": time.Now().UTC(),
			"execution_duration_ms":  durationMs,
		})
		e.auditTrail.RecordTransition(ctx, action.TenantID, action.ID, "execution_failed", nil, "system",
			model.StatusExecuting, model.StatusExecutionFailed, map[string]interface{}{"error": errMsg})
		e.publishEvent(ctx, action.TenantID, "com.clario360.cyber.remediation.execution_failed",
			map[string]interface{}{"id": action.ID, "error": errMsg})

		// Auto-rollback if plan is reversible
		if action.Plan.Reversible {
			_ = e.remRepo.UpdateStatus(ctx, action.TenantID, action.ID, model.StatusRollingBack, map[string]interface{}{})
			_ = strat.Rollback(ctx, action)
		}

		if execErr != nil {
			return result, execErr
		}
		return result, fmt.Errorf("execution failed: %s", errMsg)
	}

	resultJSON, _ := json.Marshal(result)
	_ = e.remRepo.UpdateStatus(ctx, action.TenantID, action.ID, model.StatusExecuted, map[string]interface{}{
		"execution_result":       resultJSON,
		"execution_completed_at": time.Now().UTC(),
		"execution_duration_ms":  durationMs,
	})
	e.auditTrail.RecordTransition(ctx, action.TenantID, action.ID, "execution_done", nil, "system",
		model.StatusExecuting, model.StatusExecuted, map[string]interface{}{
			"steps_executed": result.StepsExecuted,
			"duration_ms":    durationMs,
		})
	e.publishEvent(ctx, action.TenantID, "com.clario360.cyber.remediation.executed",
		map[string]interface{}{"id": action.ID, "steps_executed": result.StepsExecuted, "duration_ms": durationMs})

	return result, nil
}

// Verify runs post-execution verification.
func (e *RemediationExecutor) Verify(ctx context.Context, action *model.RemediationAction, actorID *uuid.UUID, actorName string) (*model.VerificationResult, error) {
	if action.Status != model.StatusExecuted && action.Status != model.StatusVerificationPending {
		return nil, fmt.Errorf("%w: status must be 'executed' or 'verification_pending' (current: %s)",
			ErrPreConditionFailed, action.Status)
	}

	strat, err := e.getStrategy(action.Type)
	if err != nil {
		return nil, err
	}

	_ = e.remRepo.UpdateStatus(ctx, action.TenantID, action.ID, model.StatusVerificationPending, map[string]interface{}{})
	e.auditTrail.RecordTransition(ctx, action.TenantID, action.ID, "verification_started", actorID, actorName,
		action.Status, model.StatusVerificationPending, nil)

	start := time.Now()
	result, verErr := strat.Verify(ctx, action)
	durationMs := time.Since(start).Milliseconds()

	if verErr != nil {
		return nil, verErr
	}

	resultJSON, _ := json.Marshal(result)
	newStatus := model.StatusVerified
	if !result.Verified {
		newStatus = model.StatusVerificationFailed
	}

	_ = e.remRepo.UpdateStatus(ctx, action.TenantID, action.ID, newStatus, map[string]interface{}{
		"verification_result": resultJSON,
		"verified_at":         time.Now().UTC(),
	})

	transitionAction := "verify_success"
	eventType := "com.clario360.cyber.remediation.verified"
	if !result.Verified {
		transitionAction = "verify_failure"
		eventType = "com.clario360.cyber.remediation.verification_failed"
	}
	e.auditTrail.RecordTransition(ctx, action.TenantID, action.ID, transitionAction, nil, "system",
		model.StatusVerificationPending, newStatus, map[string]interface{}{
			"verified":       result.Verified,
			"duration_ms":    durationMs,
			"failure_reason": result.FailureReason,
		})
	e.publishEvent(ctx, action.TenantID, eventType,
		map[string]interface{}{"id": action.ID, "verified": result.Verified})

	return result, nil
}

// Rollback restores pre-execution state after governance validation.
func (e *RemediationExecutor) Rollback(ctx context.Context, action *model.RemediationAction, reason string, approvedBy uuid.UUID, approverName string) error {
	// Validate rollback pre-conditions
	if err := checkRollbackPreConditions(action); err != nil {
		return err
	}

	validStatuses := map[model.RemediationStatus]bool{
		model.StatusExecuted:           true,
		model.StatusVerified:           true,
		model.StatusVerificationFailed: true,
		model.StatusRollbackPending:    true,
	}
	if !validStatuses[action.Status] {
		return fmt.Errorf("%w: cannot rollback from status '%s'", ErrInvalidTransition, action.Status)
	}

	strat, err := e.getStrategy(action.Type)
	if err != nil {
		return err
	}

	_ = e.remRepo.UpdateStatus(ctx, action.TenantID, action.ID, model.StatusRollingBack, map[string]interface{}{
		"rollback_reason":      reason,
		"rollback_approved_by": approvedBy,
	})
	e.auditTrail.RecordTransition(ctx, action.TenantID, action.ID, "rollback_started", &approvedBy, approverName,
		action.Status, model.StatusRollingBack, map[string]interface{}{"reason": reason})
	e.publishEvent(ctx, action.TenantID, "com.clario360.cyber.remediation.rollback_requested",
		map[string]interface{}{"id": action.ID, "reason": reason})

	start := time.Now()
	rollbackErr := strat.Rollback(ctx, action)
	durationMs := time.Since(start).Milliseconds()

	if rollbackErr != nil {
		_ = e.remRepo.UpdateStatus(ctx, action.TenantID, action.ID, model.StatusRollbackFailed, map[string]interface{}{})
		e.auditTrail.RecordTransition(ctx, action.TenantID, action.ID, "rollback_error", nil, "system",
			model.StatusRollingBack, model.StatusRollbackFailed,
			map[string]interface{}{"error": rollbackErr.Error(), "duration_ms": durationMs})
		e.publishEvent(ctx, action.TenantID, "com.clario360.cyber.remediation.rollback_failed",
			map[string]interface{}{"id": action.ID, "error": rollbackErr.Error()})
		e.logger.Error().Err(rollbackErr).Str("id", action.ID.String()).Msg("CRITICAL: rollback failed — manual intervention required")
		return rollbackErr
	}

	rollbackResult := &model.RollbackResult{Success: true, DurationMs: durationMs}
	rollbackJSON, _ := json.Marshal(rollbackResult)
	now := time.Now().UTC()
	_ = e.remRepo.UpdateStatus(ctx, action.TenantID, action.ID, model.StatusRolledBack, map[string]interface{}{
		"rollback_result":  rollbackJSON,
		"rolled_back_at":   now,
	})
	e.auditTrail.RecordTransition(ctx, action.TenantID, action.ID, "rollback_done", nil, "system",
		model.StatusRollingBack, model.StatusRolledBack, map[string]interface{}{"duration_ms": durationMs})
	e.publishEvent(ctx, action.TenantID, "com.clario360.cyber.remediation.rolled_back",
		map[string]interface{}{"id": action.ID, "reason": reason})

	return nil
}

func (e *RemediationExecutor) getStrategy(t model.RemediationType) (strategy.RemediationStrategy, error) {
	strat, ok := e.strategies[t]
	if !ok {
		return nil, fmt.Errorf("no strategy registered for type '%s'", t)
	}
	return strat, nil
}

func (e *RemediationExecutor) publishEvent(ctx context.Context, tenantID uuid.UUID, eventType string, payload map[string]interface{}) {
	if e.producer == nil {
		return
	}
	ev, err := events.NewEvent(eventType, "cyber-service", tenantID.String(), payload)
	if err != nil {
		e.logger.Warn().Err(err).Str("event", eventType).Msg("failed to create event")
		return
	}
	if err := e.producer.Publish(ctx, events.Topics.RemediationEvents, ev); err != nil {
		e.logger.Warn().Err(err).Str("event", eventType).Msg("failed to publish remediation event")
	}
}
