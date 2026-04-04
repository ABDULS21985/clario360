package exception

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// ExpiryChecker scans for risk exceptions that have passed their expiry date
// or are due for periodic re-review, and takes corrective action.
type ExpiryChecker struct {
	repo       ExceptionRepository
	remUpdater RemediationUpdater
	logger     zerolog.Logger
}

// NewExpiryChecker constructs an ExpiryChecker with the required dependencies.
func NewExpiryChecker(repo ExceptionRepository, remUpdater RemediationUpdater, logger zerolog.Logger) *ExpiryChecker {
	return &ExpiryChecker{
		repo:       repo,
		remUpdater: remUpdater,
		logger:     logger.With().Str("component", "expiry_checker").Logger(),
	}
}

// Run scans all exceptions for the given tenant and expires any whose ExpiresAt
// is in the past. For each expired exception:
//  1. The exception's ApprovalStatus is set to expired.
//  2. The exception's Status is set to expired.
//  3. Any linked remediation is re-opened (status → open).
//
// Returns the number of exceptions that were expired.
func (ec *ExpiryChecker) Run(ctx context.Context, tenantID uuid.UUID) (int, error) {
	exceptions, err := ec.repo.FindExpired(ctx, tenantID)
	if err != nil {
		return 0, fmt.Errorf("expiry checker: find expired: %w", err)
	}

	ec.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("expired_count", len(exceptions)).
		Msg("checking for expired exceptions")

	now := time.Now().UTC()
	expiredCount := 0

	for i := range exceptions {
		exc := &exceptions[i]

		exc.ApprovalStatus = model.ApprovalExpired
		exc.Status = model.ExceptionStatusExpired
		exc.UpdatedAt = now

		if updateErr := ec.repo.Update(ctx, exc); updateErr != nil {
			ec.logger.Error().
				Err(updateErr).
				Str("exception_id", exc.ID.String()).
				Msg("failed to expire exception")
			continue
		}

		// Re-open the linked remediation so it re-enters the active queue.
		if exc.RemediationID != nil {
			if remErr := ec.remUpdater.UpdateStatus(ctx, tenantID, *exc.RemediationID, model.StatusOpen); remErr != nil {
				ec.logger.Error().
					Err(remErr).
					Str("exception_id", exc.ID.String()).
					Str("remediation_id", exc.RemediationID.String()).
					Msg("failed to re-open linked remediation after exception expiry")
			}
		}

		expiredCount++

		ec.logger.Info().
			Str("tenant_id", tenantID.String()).
			Str("exception_id", exc.ID.String()).
			Time("expired_at", exc.ExpiresAt).
			Msg("exception expired")
	}

	ec.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("expired_count", expiredCount).
		Msg("expiry check complete")

	return expiredCount, nil
}

// CheckReviews finds all active exceptions whose NextReviewAt is at or before
// the current time, indicating they are overdue for periodic re-review.
func (ec *ExpiryChecker) CheckReviews(ctx context.Context, tenantID uuid.UUID) ([]model.RiskException, error) {
	exceptions, err := ec.repo.FindNeedingReview(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("expiry checker: find needing review: %w", err)
	}

	ec.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("needs_review", len(exceptions)).
		Msg("review check complete")

	return exceptions, nil
}
