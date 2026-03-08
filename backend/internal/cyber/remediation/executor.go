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
	"github.com/clario360/platform/internal/events"
)

const defaultRollbackWindowHours = 72

// RemediationExecutor orchestrates the dry-run → execute → verify → rollback lifecycle.
type RemediationExecutor struct {
	strategies map[model.RemediationType]strategy.RemediationStrategy
	auditTrail auditRecorder
	remRepo    remediationRepo
	alertRepo  alertRepo
	vulnRepo   vulnerabilityRepo
	producer   *events.Producer
	logger     zerolog.Logger
}

// NewRemediationExecutor creates a RemediationExecutor with all registered strategies.
func NewRemediationExecutor(
	strategies map[model.RemediationType]strategy.RemediationStrategy,
	auditTrail auditRecorder,
	remRepo remediationRepo,
	alertRepo alertRepo,
	vulnRepo vulnerabilityRepo,
	producer *events.Producer,
	logger zerolog.Logger,
) *RemediationExecutor {
	return &RemediationExecutor{
		strategies: strategies,
		auditTrail: auditTrail,
		remRepo:    remRepo,
		alertRepo:  alertRepo,
		vulnRepo:   vulnRepo,
		producer:   producer,
		logger:     logger.With().Str("component", "remediation-executor").Logger(),
	}
}

// DryRun executes a dry-run simulation for the remediation action.
func (e *RemediationExecutor) DryRun(ctx context.Context, action *model.RemediationAction, actorID *uuid.UUID, actorName string) (*model.DryRunResult, error) {
	if action.Status != model.StatusApproved && action.Status != model.StatusDryRunFailed {
		return nil, fmt.Errorf("%w: action must be in 'approved' or 'dry_run_failed' state to start dry-run (current: %s)",
			ErrPreConditionFailed, action.Status)
	}

	strat, err := e.getStrategy(action.Type)
	if err != nil {
		return nil, err
	}

	previousStatus := action.Status
	if err := e.remRepo.UpdateStatus(ctx, action.TenantID, action.ID, model.StatusDryRunRunning, map[string]interface{}{}); err != nil {
		return nil, fmt.Errorf("transition to dry_run_running: %w", err)
	}
	e.auditTrail.RecordTransition(ctx, action.TenantID, action.ID, "dry_run_started", actorID, actorName,
		previousStatus, model.StatusDryRunRunning, nil)
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
			"dry_run_result":      resultJSON,
			"dry_run_at":          time.Now().UTC(),
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
		"dry_run_result":      resultJSON,
		"dry_run_at":          time.Now().UTC(),
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
	eventType := "com.clario360.cyber.remediation.dry_run_completed"
	if !result.Success {
		eventType = "com.clario360.cyber.remediation.dry_run_failed"
	}
	e.publishEvent(ctx, action.TenantID, eventType,
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

	preState, err := e.capturePreExecutionState(ctx, strat, action)
	if err != nil {
		if action.Plan.Reversible {
			return nil, fmt.Errorf("%w: failed to capture pre-execution state: %v", ErrPreConditionFailed, err)
		}
		e.logger.Warn().Err(err).Str("id", action.ID.String()).Msg("failed to capture pre-execution state for non-reversible action")
		preState = json.RawMessage(`{"capture_error":"state capture failed"}`)
	}

	rollbackDeadline := time.Now().UTC().Add(defaultRollbackWindowHours * time.Hour)
	now := time.Now().UTC()
	if err := e.remRepo.UpdateStatus(ctx, action.TenantID, action.ID, model.StatusExecuting, map[string]interface{}{
		"pre_execution_state":  preState,
		"executed_by":          executedBy,
		"execution_started_at": now,
		"rollback_deadline":    rollbackDeadline,
	}); err != nil {
		return nil, fmt.Errorf("transition to executing: %w", err)
	}
	e.auditTrail.RecordTransition(ctx, action.TenantID, action.ID, "execution_started", &executedBy, executorName,
		action.Status, model.StatusExecuting, map[string]interface{}{"steps_total": len(action.Plan.Steps)})
	e.publishEvent(ctx, action.TenantID, "com.clario360.cyber.remediation.execution_started",
		map[string]interface{}{"id": action.ID, "executed_by": executedBy, "steps_total": len(action.Plan.Steps)})

	// Reload with pre_execution_state set
	action.PreExecutionState = preState

	start := time.Now()
	result, execErr := strat.Execute(ctx, action)
	durationMs := time.Since(start).Milliseconds()
	if result != nil {
		for _, step := range result.StepResults {
			stepDetails := map[string]interface{}{}
			if step.Output != "" {
				stepDetails["output"] = step.Output
			}
			if step.Error != "" {
				stepDetails["error"] = step.Error
			}
			e.auditTrail.RecordStep(ctx, action.TenantID, action.ID, step.StepNumber, step.Action, step.Status, step.DurationMs, step.Error, stepDetails)
		}
	}

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
			map[string]interface{}{
				"id":                 action.ID,
				"error":              errMsg,
				"created_by":         action.CreatedBy,
				"affected_asset_ids": action.AffectedAssetIDs,
			})

		// Auto-rollback executed changes for reversible plans, but keep the lifecycle in execution_failed.
		if action.Plan.Reversible {
			autoRollbackStart := time.Now()
			if rollbackErr := strat.Rollback(ctx, action); rollbackErr != nil {
				e.auditTrail.RecordAction(ctx, action.TenantID, action.ID, "auto_rollback_failed", nil, "system", map[string]interface{}{
					"error":       rollbackErr.Error(),
					"duration_ms": time.Since(autoRollbackStart).Milliseconds(),
				})
				_ = e.createRollbackFailureAlert(ctx, action, rollbackErr)
				e.logger.Error().Err(rollbackErr).Str("id", action.ID.String()).Msg("automatic rollback failed after execution failure")
			} else {
				_ = e.restoreLinkedEntityState(ctx, action)
				e.auditTrail.RecordAction(ctx, action.TenantID, action.ID, "auto_rollback_completed", nil, "system", map[string]interface{}{
					"duration_ms": time.Since(autoRollbackStart).Milliseconds(),
				})
			}
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
		map[string]interface{}{
			"id":                 action.ID,
			"steps_executed":     result.StepsExecuted,
			"duration_ms":        durationMs,
			"created_by":         action.CreatedBy,
			"affected_asset_ids": action.AffectedAssetIDs,
		})

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
		"verified_by":         actorID,
		"verified_at":         time.Now().UTC(),
	})
	if result.Verified {
		if action.VulnerabilityID != nil && e.vulnRepo != nil {
			_, _ = e.vulnRepo.UpdateStatusGlobal(ctx, action.TenantID, *action.VulnerabilityID, "resolved", nil)
		}
		if action.AlertID != nil && e.alertRepo != nil {
			note := "Resolved by verified remediation"
			_, _ = e.alertRepo.UpdateStatus(ctx, action.TenantID, *action.AlertID, model.AlertStatusResolved, &note, nil)
		}
	}

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
		_ = e.createRollbackFailureAlert(ctx, action, rollbackErr)
		e.logger.Error().Err(rollbackErr).Str("id", action.ID.String()).Msg("CRITICAL: rollback failed — manual intervention required")
		return rollbackErr
	}
	_ = e.restoreLinkedEntityState(ctx, action)

	rollbackResult := &model.RollbackResult{Success: true, DurationMs: durationMs}
	rollbackJSON, _ := json.Marshal(rollbackResult)
	now := time.Now().UTC()
	_ = e.remRepo.UpdateStatus(ctx, action.TenantID, action.ID, model.StatusRolledBack, map[string]interface{}{
		"rollback_result": rollbackJSON,
		"rolled_back_at":  now,
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

func (e *RemediationExecutor) capturePreExecutionState(ctx context.Context, strat strategy.RemediationStrategy, action *model.RemediationAction) (json.RawMessage, error) {
	strategyState, err := strat.CaptureState(ctx, action)
	if err != nil {
		return nil, err
	}
	state := map[string]interface{}{
		"captured_at": time.Now().UTC(),
	}
	if len(strategyState) > 0 {
		var decoded map[string]interface{}
		if err := json.Unmarshal(strategyState, &decoded); err == nil {
			for key, value := range decoded {
				state[key] = value
			}
		} else {
			state["strategy_state"] = json.RawMessage(strategyState)
		}
	}
	if action.VulnerabilityID != nil && e.vulnRepo != nil {
		if vuln, err := e.vulnRepo.GetByID(ctx, action.TenantID, *action.VulnerabilityID); err == nil {
			state["linked_vulnerability"] = map[string]interface{}{
				"id":     vuln.ID.String(),
				"status": vuln.Status,
			}
		}
	}
	if action.AlertID != nil && e.alertRepo != nil {
		if alert, err := e.alertRepo.GetByID(ctx, action.TenantID, *action.AlertID); err == nil {
			state["linked_alert"] = map[string]interface{}{
				"id":                    alert.ID.String(),
				"status":                string(alert.Status),
				"resolution_notes":      alert.ResolutionNotes,
				"false_positive_reason": alert.FalsePositiveReason,
			}
		}
	}
	return json.Marshal(state)
}

func (e *RemediationExecutor) restoreLinkedEntityState(ctx context.Context, action *model.RemediationAction) error {
	if len(action.PreExecutionState) == 0 {
		return nil
	}
	var state struct {
		LinkedVulnerability *struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"linked_vulnerability"`
		LinkedAlert *struct {
			ID                  string  `json:"id"`
			Status              string  `json:"status"`
			ResolutionNotes     *string `json:"resolution_notes"`
			FalsePositiveReason *string `json:"false_positive_reason"`
		} `json:"linked_alert"`
	}
	if err := json.Unmarshal(action.PreExecutionState, &state); err != nil {
		return fmt.Errorf("decode pre-execution state: %w", err)
	}
	if state.LinkedVulnerability != nil && e.vulnRepo != nil {
		vulnID, err := uuid.Parse(state.LinkedVulnerability.ID)
		if err == nil {
			_, _ = e.vulnRepo.UpdateStatusGlobal(ctx, action.TenantID, vulnID, state.LinkedVulnerability.Status, nil)
		}
	}
	if state.LinkedAlert != nil && e.alertRepo != nil {
		alertID, err := uuid.Parse(state.LinkedAlert.ID)
		if err == nil {
			var reason *string
			if state.LinkedAlert.Status == string(model.AlertStatusFalsePositive) {
				reason = state.LinkedAlert.FalsePositiveReason
			}
			_, _ = e.alertRepo.UpdateStatus(ctx, action.TenantID, alertID, model.AlertStatus(state.LinkedAlert.Status), state.LinkedAlert.ResolutionNotes, reason)
		}
	}
	return nil
}

func (e *RemediationExecutor) createRollbackFailureAlert(ctx context.Context, action *model.RemediationAction, rollbackErr error) error {
	if e.alertRepo == nil {
		return nil
	}
	now := time.Now().UTC()
	alert := &model.Alert{
		TenantID:    action.TenantID,
		Title:       fmt.Sprintf("Remediation rollback failed: %s", action.Title),
		Description: "A governed remediation rollback failed and requires immediate manual intervention.",
		Severity:    model.SeverityCritical,
		Status:      model.AlertStatusNew,
		Source:      "remediation",
		AssetIDs:    append([]uuid.UUID(nil), action.AffectedAssetIDs...),
		Explanation: model.AlertExplanation{
			Summary:            "Rollback failed",
			Reason:             rollbackErr.Error(),
			RecommendedActions: []string{"Review the remediation audit trail immediately", "Perform manual recovery on affected assets", "Escalate to the security manager and operations lead"},
		},
		ConfidenceScore: 1.0,
		EventCount:      1,
		FirstEventAt:    now,
		LastEventAt:     now,
		Metadata:        json.RawMessage(`{}`),
	}
	if len(action.AffectedAssetIDs) > 0 {
		alert.AssetID = &action.AffectedAssetIDs[0]
	}
	_, err := e.alertRepo.Create(ctx, alert)
	return err
}
