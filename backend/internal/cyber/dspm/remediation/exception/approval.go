package exception

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// ApprovalWorkflow provides read-only queries for the exception approval
// pipeline, allowing reviewers to discover pending approvals and check
// the current approval state of individual exceptions.
type ApprovalWorkflow struct {
	repo   ExceptionRepository
	logger zerolog.Logger
}

// NewApprovalWorkflow constructs an ApprovalWorkflow with the required dependencies.
func NewApprovalWorkflow(repo ExceptionRepository, logger zerolog.Logger) *ApprovalWorkflow {
	return &ApprovalWorkflow{
		repo:   repo,
		logger: logger.With().Str("component", "approval_workflow").Logger(),
	}
}

// PendingApprovals returns all risk exceptions for the given tenant that are
// awaiting approval.
func (aw *ApprovalWorkflow) PendingApprovals(ctx context.Context, tenantID uuid.UUID) ([]model.RiskException, error) {
	all, err := aw.repo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("approval workflow: list pending: %w", err)
	}

	var pending []model.RiskException
	for _, exc := range all {
		if exc.ApprovalStatus == model.ApprovalPending {
			pending = append(pending, exc)
		}
	}

	aw.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("pending_count", len(pending)).
		Msg("retrieved pending approvals")

	return pending, nil
}

// IsApproved checks whether a specific exception has been approved.
func (aw *ApprovalWorkflow) IsApproved(ctx context.Context, tenantID, exceptionID uuid.UUID) (bool, error) {
	exception, err := aw.repo.GetByID(ctx, tenantID, exceptionID)
	if err != nil {
		return false, fmt.Errorf("approval workflow: get by id: %w", err)
	}

	return exception.ApprovalStatus == model.ApprovalApproved, nil
}
