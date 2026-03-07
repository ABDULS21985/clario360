package remediation

import (
	"errors"
	"fmt"
	"time"

	"github.com/clario360/platform/internal/cyber/model"
)

// Governance errors — used as sentinel values for HTTP response mapping.
var (
	ErrInvalidTransition    = errors.New("invalid state transition")
	ErrInsufficientPermission = errors.New("insufficient permission for this action")
	ErrPreConditionFailed   = errors.New("pre-condition check failed")
)

// roleLevel maps role names to numeric levels for comparison.
var roleLevel = map[string]int{
	"analyst":          1,
	"security_analyst": 1,
	"security_manager": 2,
	"ciso":             3,
	"tenant_admin":     3,
	"admin":            4,
}

func hasMinRole(role, minimum string) bool {
	return roleLevel[role] >= roleLevel[minimum]
}

// transitionDef defines one valid state transition.
type transitionDef struct {
	From        model.RemediationStatus
	To          model.RemediationStatus
	Action      string
	MinRole     string
	PreCondition func(a *model.RemediationAction) error
}

// validTransitions is the complete, authoritative state machine.
var validTransitions = []transitionDef{
	{model.StatusDraft, model.StatusPendingApproval, "submit", "analyst", nil},
	{model.StatusPendingApproval, model.StatusApproved, "approve", "security_manager", nil},
	{model.StatusPendingApproval, model.StatusRejected, "reject", "security_manager", nil},
	{model.StatusPendingApproval, model.StatusRevisionRequested, "request_revision", "security_manager", nil},
	{model.StatusRejected, model.StatusDraft, "revise", "analyst", nil},
	{model.StatusRevisionRequested, model.StatusPendingApproval, "resubmit", "analyst", nil},
	{model.StatusApproved, model.StatusDryRunRunning, "start_dry_run", "analyst", nil},
	{model.StatusDryRunRunning, model.StatusDryRunCompleted, "dry_run_done", "system", nil},
	{model.StatusDryRunRunning, model.StatusDryRunFailed, "dry_run_error", "system", nil},
	{model.StatusDryRunFailed, model.StatusDryRunRunning, "retry_dry_run", "analyst", nil},
	{model.StatusDryRunCompleted, model.StatusExecutionPending, "queue_execution", "analyst", checkExecutePreConditions},
	{model.StatusExecutionPending, model.StatusExecuting, "start_execution", "system", nil},
	{model.StatusExecuting, model.StatusExecuted, "execution_done", "system", nil},
	{model.StatusExecuting, model.StatusExecutionFailed, "execution_error", "system", nil},
	{model.StatusExecutionFailed, model.StatusRollingBack, "auto_rollback", "system", nil},
	{model.StatusExecutionFailed, model.StatusDraft, "revise", "analyst", nil},
	{model.StatusExecuted, model.StatusVerificationPending, "start_verify", "analyst", nil},
	{model.StatusVerificationPending, model.StatusVerified, "verify_success", "system", nil},
	{model.StatusVerificationPending, model.StatusVerificationFailed, "verify_failure", "system", nil},
	{model.StatusExecuted, model.StatusRollbackPending, "request_rollback", "analyst", checkRollbackPreConditions},
	{model.StatusVerified, model.StatusRollbackPending, "request_rollback", "analyst", checkRollbackPreConditions},
	{model.StatusVerificationFailed, model.StatusRollbackPending, "request_rollback", "analyst", checkRollbackPreConditions},
	{model.StatusRollbackPending, model.StatusRollingBack, "approve_rollback", "security_manager", nil},
	{model.StatusRollingBack, model.StatusRolledBack, "rollback_done", "system", nil},
	{model.StatusRollingBack, model.StatusRollbackFailed, "rollback_error", "system", nil},
	{model.StatusVerified, model.StatusClosed, "close", "analyst", nil},
	{model.StatusRolledBack, model.StatusClosed, "close", "analyst", nil},
}

func checkExecutePreConditions(a *model.RemediationAction) error {
	if a.DryRunResult == nil || a.DryRunAt == nil {
		return fmt.Errorf("%w: dry-run must be completed before execution", ErrPreConditionFailed)
	}
	if !a.DryRunResult.Success {
		return fmt.Errorf("%w: cannot execute — dry-run reported failures, fix issues and re-run dry-run", ErrPreConditionFailed)
	}
	if a.ApprovedBy == nil || a.ApprovedAt == nil {
		return fmt.Errorf("%w: approval is required before execution", ErrPreConditionFailed)
	}
	return nil
}

func checkRollbackPreConditions(a *model.RemediationAction) error {
	if a.RollbackDeadline != nil && time.Now().After(*a.RollbackDeadline) {
		return fmt.Errorf("%w: rollback window has expired (%s), manual intervention required",
			ErrPreConditionFailed, a.RollbackDeadline.Format(time.RFC3339))
	}
	if a.PreExecutionState == nil {
		return fmt.Errorf("%w: no pre-execution state captured, rollback not possible", ErrPreConditionFailed)
	}
	if !a.Plan.Reversible {
		return fmt.Errorf("%w: this remediation type is not reversible", ErrPreConditionFailed)
	}
	return nil
}

// ValidateTransition checks whether the transition is allowed for the given action and role.
// Returns nil if valid, or a typed error (ErrInvalidTransition, ErrInsufficientPermission, ErrPreConditionFailed).
func ValidateTransition(action *model.RemediationAction, target model.RemediationStatus, actorRole string) error {
	for _, t := range validTransitions {
		if t.From == action.Status && t.To == target {
			// Check role
			if !hasMinRole(actorRole, t.MinRole) {
				return fmt.Errorf("%w: role '%s' cannot perform action '%s' (requires '%s')",
					ErrInsufficientPermission, actorRole, t.Action, t.MinRole)
			}
			// Check pre-conditions
			if t.PreCondition != nil {
				if err := t.PreCondition(action); err != nil {
					return err
				}
			}
			return nil
		}
	}
	return fmt.Errorf("%w: cannot transition from '%s' to '%s'",
		ErrInvalidTransition, action.Status, target)
}

// IsTerminalStatus returns true if the status is a final state.
func IsTerminalStatus(s model.RemediationStatus) bool {
	switch s {
	case model.StatusClosed, model.StatusRejected, model.StatusRollbackFailed:
		return true
	}
	return false
}

// IsPreExecutionStatus returns true if the action has not yet begun execution.
func IsPreExecutionStatus(s model.RemediationStatus) bool {
	switch s {
	case model.StatusDraft, model.StatusPendingApproval, model.StatusApproved,
		model.StatusRevisionRequested, model.StatusRejected,
		model.StatusDryRunRunning, model.StatusDryRunCompleted, model.StatusDryRunFailed,
		model.StatusExecutionPending:
		return true
	}
	return false
}
