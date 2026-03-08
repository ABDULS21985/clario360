package remediation

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// RollbackRequest carries the parameters for a rollback operation.
type RollbackRequest struct {
	Reason        string
	RequestedBy   uuid.UUID
	RequesterName string
	Role          string
}

// RollbackEligibility describes why an action can or cannot be rolled back.
type RollbackEligibility struct {
	Eligible        bool          `json:"eligible"`
	WindowOpen      bool          `json:"window_open"`
	WindowRemaining time.Duration `json:"window_remaining"`
	Reversible      bool          `json:"reversible"`
	HasPreState     bool          `json:"has_pre_state"`
	StatusAllowed   bool          `json:"status_allowed"`
	Reason          string        `json:"reason,omitempty"`
}

// RollbackSummary provides a structured status report of an action's rollback state.
type RollbackSummary struct {
	ActionID        uuid.UUID          `json:"action_id"`
	Status          string             `json:"status"`
	Eligible        bool               `json:"eligible"`
	WindowOpen      bool               `json:"window_open"`
	WindowRemaining time.Duration      `json:"window_remaining"`
	RollbackReason  string             `json:"rollback_reason,omitempty"`
	RollbackResult  *model.RollbackResult `json:"rollback_result,omitempty"`
	RolledBackAt    *time.Time         `json:"rolled_back_at,omitempty"`
}

// BatchRollbackResult reports the outcome of a batch rollback operation.
type BatchRollbackResult struct {
	Total     int                    `json:"total"`
	Succeeded int                    `json:"succeeded"`
	Failed    int                    `json:"failed"`
	Skipped   int                    `json:"skipped"`
	Details   []BatchRollbackDetail  `json:"details"`
}

// BatchRollbackDetail describes the outcome for a single action within a batch rollback.
type BatchRollbackDetail struct {
	ActionID uuid.UUID `json:"action_id"`
	Status   string    `json:"status"` // "rolled_back", "failed", "skipped"
	Error    string    `json:"error,omitempty"`
}

// RollbackOrchestrator coordinates the two-phase rollback governance flow:
// Phase 1 — RequestRollback: analyst requests rollback → transitions to rollback_pending
// Phase 2 — ApproveAndExecute: security_manager approves → executor rolls back
//
// It enforces the rollback window, validates pre-conditions, and coordinates
// with the RemediationExecutor for the actual rollback execution.
type RollbackOrchestrator struct {
	executor *RemediationExecutor
	remRepo  remediationRepo
	audit    auditRecorder
	logger   zerolog.Logger
}

// NewRollbackOrchestrator creates a RollbackOrchestrator.
func NewRollbackOrchestrator(
	executor *RemediationExecutor,
	remRepo remediationRepo,
	audit auditRecorder,
	logger zerolog.Logger,
) *RollbackOrchestrator {
	return &RollbackOrchestrator{
		executor: executor,
		remRepo:  remRepo,
		audit:    audit,
		logger:   logger.With().Str("component", "rollback-orchestrator").Logger(),
	}
}

// RequestRollback implements Phase 1 of the two-phase rollback governance flow.
// It validates the rollback window, checks the state-machine transition, persists
// the rollback_pending status, and records an audit entry.
func (o *RollbackOrchestrator) RequestRollback(ctx context.Context, action *model.RemediationAction, req RollbackRequest) error {
	if !IsRollbackWindowOpen(action) {
		return fmt.Errorf("%w: rollback window is closed or not set for action %s",
			ErrPreConditionFailed, action.ID)
	}

	if err := ValidateTransition(action, model.StatusRollbackPending, req.Role); err != nil {
		return fmt.Errorf("rollback transition validation: %w", err)
	}

	if err := o.remRepo.UpdateStatus(ctx, action.TenantID, action.ID, model.StatusRollbackPending,
		map[string]interface{}{"rollback_reason": req.Reason}); err != nil {
		return fmt.Errorf("persist rollback_pending status: %w", err)
	}

	o.audit.RecordAction(ctx, action.TenantID, action.ID, "rollback_requested",
		&req.RequestedBy, req.RequesterName,
		map[string]interface{}{
			"reason": req.Reason,
			"role":   req.Role,
		})

	o.logger.Info().
		Str("action_id", action.ID.String()).
		Str("requester", req.RequesterName).
		Str("reason", req.Reason).
		Msg("rollback requested — awaiting security_manager approval")

	return nil
}

// ApproveAndExecute implements Phase 2 of the two-phase rollback governance flow.
// It verifies the action is in rollback_pending, confirms the window is still open,
// and delegates to the RemediationExecutor for the actual rollback execution.
func (o *RollbackOrchestrator) ApproveAndExecute(ctx context.Context, action *model.RemediationAction, reason string, approverID uuid.UUID, approverName string) error {
	if action.Status != model.StatusRollbackPending {
		return fmt.Errorf("%w: action must be in 'rollback_pending' to approve rollback (current: %s)",
			ErrPreConditionFailed, action.Status)
	}

	if !IsRollbackWindowOpen(action) {
		return fmt.Errorf("%w: rollback window has expired — manual intervention required for action %s",
			ErrPreConditionFailed, action.ID)
	}

	o.audit.RecordAction(ctx, action.TenantID, action.ID, "rollback_approved",
		&approverID, approverName,
		map[string]interface{}{"reason": reason})

	if err := o.executor.Rollback(ctx, action, reason, approverID, approverName); err != nil {
		return fmt.Errorf("rollback execution: %w", err)
	}

	return nil
}

// ForceRollback bypasses the two-phase approval flow and the rollback window check
// for emergency situations. Only admin-level roles are permitted.
func (o *RollbackOrchestrator) ForceRollback(ctx context.Context, action *model.RemediationAction, reason string, actorID uuid.UUID, actorName, actorRole string) error {
	if !isAdminRole(actorRole) {
		return fmt.Errorf("%w: force rollback requires admin, ciso, or tenant_admin role (got: %s)",
			ErrInsufficientPermission, actorRole)
	}

	if err := checkRollbackPreConditions(action); err != nil {
		return fmt.Errorf("force rollback pre-conditions: %w", err)
	}

	validStatuses := map[model.RemediationStatus]bool{
		model.StatusExecuted:           true,
		model.StatusVerified:           true,
		model.StatusVerificationFailed: true,
		model.StatusRollbackPending:    true,
		model.StatusRollbackFailed:     true,
	}
	if !validStatuses[action.Status] {
		return fmt.Errorf("%w: force rollback not permitted from status '%s'",
			ErrInvalidTransition, action.Status)
	}

	o.audit.RecordAction(ctx, action.TenantID, action.ID, "force_rollback_initiated",
		&actorID, actorName,
		map[string]interface{}{
			"reason":         reason,
			"role":           actorRole,
			"window_expired": !IsRollbackWindowOpen(action),
		})

	o.logger.Warn().
		Str("action_id", action.ID.String()).
		Str("actor", actorName).
		Str("reason", reason).
		Bool("window_expired", !IsRollbackWindowOpen(action)).
		Msg("force rollback initiated — bypassing normal governance")

	if err := o.executor.Rollback(ctx, action, reason, actorID, actorName); err != nil {
		return fmt.Errorf("force rollback execution: %w", err)
	}

	return nil
}

// BatchRollback coordinates rollback of multiple related actions. It attempts to roll
// back each eligible action, collecting results. Actions that are not eligible are skipped.
func (o *RollbackOrchestrator) BatchRollback(ctx context.Context, actions []*model.RemediationAction, reason string, approverID uuid.UUID, approverName string) *BatchRollbackResult {
	result := &BatchRollbackResult{
		Total:   len(actions),
		Details: make([]BatchRollbackDetail, 0, len(actions)),
	}

	for _, action := range actions {
		detail := BatchRollbackDetail{ActionID: action.ID}

		eligibility := ValidateRollbackEligibility(action)
		if !eligibility.Eligible {
			detail.Status = "skipped"
			detail.Error = eligibility.Reason
			result.Skipped++
			result.Details = append(result.Details, detail)
			continue
		}

		err := o.executor.Rollback(ctx, action, reason, approverID, approverName)
		if err != nil {
			detail.Status = "failed"
			detail.Error = err.Error()
			result.Failed++
			o.logger.Error().Err(err).
				Str("action_id", action.ID.String()).
				Msg("batch rollback: action failed")
		} else {
			detail.Status = "rolled_back"
			result.Succeeded++
		}
		result.Details = append(result.Details, detail)
	}

	o.audit.RecordAction(ctx, uuid.Nil, uuid.Nil, "batch_rollback_completed",
		&approverID, approverName,
		map[string]interface{}{
			"total":     result.Total,
			"succeeded": result.Succeeded,
			"failed":    result.Failed,
			"skipped":   result.Skipped,
			"reason":    reason,
		})

	return result
}

// GetRollbackSummary returns a structured summary of the rollback state for an action.
func GetRollbackSummary(action *model.RemediationAction) *RollbackSummary {
	summary := &RollbackSummary{
		ActionID:       action.ID,
		Status:         string(action.Status),
		WindowOpen:     IsRollbackWindowOpen(action),
		RollbackResult: action.RollbackResult,
		RolledBackAt:   action.RolledBackAt,
	}
	if action.RollbackReason != nil {
		summary.RollbackReason = *action.RollbackReason
	}
	if summary.WindowOpen {
		summary.WindowRemaining = RollbackWindowRemaining(action)
	}
	eligibility := ValidateRollbackEligibility(action)
	summary.Eligible = eligibility.Eligible
	return summary
}

// ValidateRollbackEligibility performs a comprehensive pre-flight check returning
// detailed status about whether an action can be rolled back and why not.
func ValidateRollbackEligibility(action *model.RemediationAction) *RollbackEligibility {
	e := &RollbackEligibility{
		WindowOpen:      IsRollbackWindowOpen(action),
		WindowRemaining: RollbackWindowRemaining(action),
		Reversible:      action.Plan.Reversible,
		HasPreState:     len(action.PreExecutionState) > 0,
	}

	rollbackStatuses := map[model.RemediationStatus]bool{
		model.StatusExecuted:           true,
		model.StatusVerified:           true,
		model.StatusVerificationFailed: true,
		model.StatusRollbackPending:    true,
	}
	e.StatusAllowed = rollbackStatuses[action.Status]

	if !e.Reversible {
		e.Reason = "action plan is not marked as reversible"
		return e
	}
	if !e.HasPreState {
		e.Reason = "no pre-execution state was captured"
		return e
	}
	if !e.WindowOpen {
		e.Reason = "rollback window is closed or not set"
		return e
	}
	if !e.StatusAllowed {
		e.Reason = fmt.Sprintf("rollback not permitted from status '%s'", action.Status)
		return e
	}

	e.Eligible = true
	return e
}

// IsRollbackWindowOpen reports whether the rollback deadline has not yet passed.
// Returns false if RollbackDeadline is nil or is in the past.
func IsRollbackWindowOpen(action *model.RemediationAction) bool {
	if action.RollbackDeadline == nil {
		return false
	}
	return !time.Now().After(*action.RollbackDeadline)
}

// RollbackWindowRemaining returns the time remaining in the rollback window.
// Returns 0 if the window is closed or the deadline is not set.
func RollbackWindowRemaining(action *model.RemediationAction) time.Duration {
	if !IsRollbackWindowOpen(action) {
		return 0
	}
	return time.Until(*action.RollbackDeadline)
}

// CanRollback validates whether the action is eligible for rollback by checking
// pre-conditions from governance rules and confirming the current status is one
// of the statuses from which rollback is permitted.
func CanRollback(action *model.RemediationAction) error {
	if err := checkRollbackPreConditions(action); err != nil {
		return err
	}

	switch action.Status {
	case model.StatusExecuted,
		model.StatusVerified,
		model.StatusVerificationFailed,
		model.StatusRollbackPending:
		return nil
	default:
		return fmt.Errorf("%w: rollback is not permitted from status '%s'",
			ErrPreConditionFailed, action.Status)
	}
}

// isAdminRole returns true if the role has admin-level privileges.
func isAdminRole(role string) bool {
	switch role {
	case "admin", "ciso", "tenant_admin":
		return true
	default:
		return false
	}
}
