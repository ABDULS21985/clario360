package remediation

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
)

// RollbackRequest carries the parameters for a rollback operation.
type RollbackRequest struct {
	Reason        string
	RequestedBy   uuid.UUID
	RequesterName string
	Role          string
}

// RollbackOrchestrator coordinates the two-phase rollback governance flow:
// Phase 1 — RequestRollback: analyst requests rollback → transitions to rollback_pending
// Phase 2 — ApproveAndExecute: security_manager approves → executor rolls back
//
// It enforces the rollback window, validates pre-conditions, and coordinates
// with the RemediationExecutor for the actual rollback execution.
type RollbackOrchestrator struct {
	executor *RemediationExecutor
	remRepo  *repository.RemediationRepository
	audit    *AuditTrail
	logger   zerolog.Logger
}

// NewRollbackOrchestrator creates a RollbackOrchestrator.
func NewRollbackOrchestrator(
	executor *RemediationExecutor,
	remRepo *repository.RemediationRepository,
	audit *AuditTrail,
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

	if err := o.executor.Rollback(ctx, action, reason, approverID, approverName); err != nil {
		return fmt.Errorf("rollback execution: %w", err)
	}

	return nil
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
