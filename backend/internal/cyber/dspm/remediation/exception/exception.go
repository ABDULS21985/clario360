package exception

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/dto"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// ExceptionRepository abstracts persistence for risk exceptions.
type ExceptionRepository interface {
	Create(ctx context.Context, exception *model.RiskException) (*model.RiskException, error)
	GetByID(ctx context.Context, tenantID, exceptionID uuid.UUID) (*model.RiskException, error)
	Update(ctx context.Context, exception *model.RiskException) error
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]model.RiskException, error)
	FindExpired(ctx context.Context, tenantID uuid.UUID) ([]model.RiskException, error)
	FindNeedingReview(ctx context.Context, tenantID uuid.UUID) ([]model.RiskException, error)
}

// RemediationUpdater abstracts updating the status of linked remediation items
// when an exception is approved or expires.
type RemediationUpdater interface {
	UpdateStatus(ctx context.Context, tenantID, remediationID uuid.UUID, status model.RemediationStatus) error
}

// ExceptionManager orchestrates the lifecycle of risk exceptions including
// creation, approval, and rejection workflows.
type ExceptionManager struct {
	repo       ExceptionRepository
	remUpdater RemediationUpdater
	logger     zerolog.Logger
}

// NewExceptionManager constructs an ExceptionManager with the required dependencies.
func NewExceptionManager(repo ExceptionRepository, remUpdater RemediationUpdater, logger zerolog.Logger) *ExceptionManager {
	return &ExceptionManager{
		repo:       repo,
		remUpdater: remUpdater,
		logger:     logger.With().Str("component", "exception_manager").Logger(),
	}
}

// Request validates and creates a new risk exception in pending state.
// The exception must be approved by an authorised reviewer before it takes effect.
func (em *ExceptionManager) Request(ctx context.Context, tenantID, requestedBy uuid.UUID, req *dto.CreateExceptionRequest) (*model.RiskException, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("exception request: validation failed: %w", err)
	}

	now := time.Now().UTC()
	reviewInterval := req.ReviewIntervalDays
	if reviewInterval <= 0 {
		reviewInterval = 90
	}

	nextReview := now.AddDate(0, 0, reviewInterval)

	exception := &model.RiskException{
		ID:                   uuid.New(),
		TenantID:             tenantID,
		ExceptionType:        model.ExceptionType(req.ExceptionType),
		RemediationID:        req.RemediationID,
		DataAssetID:          req.DataAssetID,
		PolicyID:             req.PolicyID,
		Justification:        req.Justification,
		BusinessReason:       req.BusinessReason,
		CompensatingControls: req.CompensatingControls,
		RiskScore:            req.RiskScore,
		RiskLevel:            req.RiskLevel,
		RequestedBy:          requestedBy,
		ApprovalStatus:       model.ApprovalPending,
		ExpiresAt:            req.ExpiresAt,
		ReviewIntervalDays:   reviewInterval,
		NextReviewAt:         &nextReview,
		ReviewCount:          0,
		Status:               model.ExceptionStatusActive,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	created, err := em.repo.Create(ctx, exception)
	if err != nil {
		return nil, fmt.Errorf("exception request: create: %w", err)
	}
	exception = created

	em.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("exception_id", exception.ID.String()).
		Str("exception_type", string(exception.ExceptionType)).
		Str("requested_by", requestedBy.String()).
		Msg("risk exception requested")

	return exception, nil
}

// Approve marks a pending exception as approved, sets the next review date,
// and updates any linked remediation to exception_granted status.
func (em *ExceptionManager) Approve(ctx context.Context, tenantID, exceptionID, approverID uuid.UUID) (*model.RiskException, error) {
	exception, err := em.repo.GetByID(ctx, tenantID, exceptionID)
	if err != nil {
		return nil, fmt.Errorf("exception approve: get: %w", err)
	}

	if exception.ApprovalStatus != model.ApprovalPending {
		return nil, fmt.Errorf("exception approve: exception is not in pending state (current: %s)", exception.ApprovalStatus)
	}

	if approverID == exception.RequestedBy {
		return nil, fmt.Errorf("exception approve: approver cannot be the same as the requester")
	}

	now := time.Now().UTC()
	nextReview := now.AddDate(0, 0, exception.ReviewIntervalDays)

	exception.ApprovalStatus = model.ApprovalApproved
	exception.ApprovedBy = &approverID
	exception.ApprovedAt = &now
	exception.NextReviewAt = &nextReview
	exception.LastReviewedAt = &now
	exception.ReviewCount++
	exception.UpdatedAt = now

	if err := em.repo.Update(ctx, exception); err != nil {
		return nil, fmt.Errorf("exception approve: update: %w", err)
	}

	// If there is a linked remediation, move it to exception_granted status.
	if exception.RemediationID != nil {
		if updateErr := em.remUpdater.UpdateStatus(ctx, tenantID, *exception.RemediationID, model.StatusExceptionGranted); updateErr != nil {
			em.logger.Error().
				Err(updateErr).
				Str("remediation_id", exception.RemediationID.String()).
				Msg("failed to update linked remediation status after exception approval")
			// Do not fail the approval — the exception itself was successfully updated.
		}
	}

	em.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("exception_id", exceptionID.String()).
		Str("approved_by", approverID.String()).
		Time("next_review_at", nextReview).
		Msg("risk exception approved")

	return exception, nil
}

// Reject marks a pending exception as rejected with the given reason.
// Linked remediations remain in their current state.
func (em *ExceptionManager) Reject(ctx context.Context, tenantID, exceptionID, rejecterID uuid.UUID, reason string) (*model.RiskException, error) {
	if reason == "" {
		return nil, fmt.Errorf("exception reject: rejection reason is required")
	}

	exception, err := em.repo.GetByID(ctx, tenantID, exceptionID)
	if err != nil {
		return nil, fmt.Errorf("exception reject: get: %w", err)
	}

	if exception.ApprovalStatus != model.ApprovalPending {
		return nil, fmt.Errorf("exception reject: exception is not in pending state (current: %s)", exception.ApprovalStatus)
	}

	now := time.Now().UTC()

	exception.ApprovalStatus = model.ApprovalRejected
	exception.ApprovedBy = &rejecterID
	exception.RejectionReason = reason
	exception.UpdatedAt = now

	if err := em.repo.Update(ctx, exception); err != nil {
		return nil, fmt.Errorf("exception reject: update: %w", err)
	}

	em.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("exception_id", exceptionID.String()).
		Str("rejected_by", rejecterID.String()).
		Str("reason", reason).
		Msg("risk exception rejected")

	return exception, nil
}
