package remediation

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
)

// AuditTrail manages the immutable audit log for remediation actions.
type AuditTrail struct {
	repo   *repository.RemediationAuditRepository
	logger zerolog.Logger
}

// NewAuditTrail creates a new AuditTrail.
func NewAuditTrail(repo *repository.RemediationAuditRepository, logger zerolog.Logger) *AuditTrail {
	return &AuditTrail{repo: repo, logger: logger}
}

// RecordTransition records a state transition event.
func (a *AuditTrail) RecordTransition(ctx context.Context, tenantID, remediationID uuid.UUID, action string, actorID *uuid.UUID, actorName string, oldStatus, newStatus model.RemediationStatus, details map[string]interface{}) {
	if details == nil {
		details = map[string]interface{}{}
	}
	entry := &model.RemediationAuditEntry{
		TenantID:      tenantID,
		RemediationID: remediationID,
		Action:        action,
		ActorID:       actorID,
		ActorName:     actorName,
		OldStatus:     string(oldStatus),
		NewStatus:     string(newStatus),
		Details:       details,
		CreatedAt:     time.Now().UTC(),
	}
	if err := a.repo.RecordEntry(ctx, entry); err != nil {
		a.logger.Error().Err(err).Str("remediation_id", remediationID.String()).Msg("failed to record audit transition")
	}
}

// RecordStep records an individual execution step outcome.
func (a *AuditTrail) RecordStep(ctx context.Context, tenantID, remediationID uuid.UUID, stepNum int, stepAction, stepResult string, durationMs int64, errMsg string, details map[string]interface{}) {
	if details == nil {
		details = map[string]interface{}{}
	}
	dur := durationMs
	entry := &model.RemediationAuditEntry{
		TenantID:      tenantID,
		RemediationID: remediationID,
		Action:        "step_execution",
		StepNumber:    &stepNum,
		StepAction:    stepAction,
		StepResult:    stepResult,
		DurationMs:    &dur,
		ErrorMessage:  errMsg,
		Details:       details,
		CreatedAt:     time.Now().UTC(),
	}
	if err := a.repo.RecordEntry(ctx, entry); err != nil {
		a.logger.Error().Err(err).Str("remediation_id", remediationID.String()).Int("step", stepNum).Msg("failed to record audit step")
	}
}

// RecordAction records a generic action event with optional actor.
func (a *AuditTrail) RecordAction(ctx context.Context, tenantID, remediationID uuid.UUID, action string, actorID *uuid.UUID, actorName string, details map[string]interface{}) {
	if details == nil {
		details = map[string]interface{}{}
	}
	entry := &model.RemediationAuditEntry{
		TenantID:      tenantID,
		RemediationID: remediationID,
		Action:        action,
		ActorID:       actorID,
		ActorName:     actorName,
		Details:       details,
		CreatedAt:     time.Now().UTC(),
	}
	if err := a.repo.RecordEntry(ctx, entry); err != nil {
		a.logger.Error().Err(err).Str("remediation_id", remediationID.String()).Msg("failed to record audit action")
	}
}
