package exception

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

func newTestExpiryChecker(repo *mockExceptionRepo, updater *mockRemediationUpdater) *ExpiryChecker {
	logger := zerolog.Nop()
	return NewExpiryChecker(repo, updater, logger)
}

func TestRunExpiresExceptions(t *testing.T) {
	repo := newMockExceptionRepo()
	updater := newMockRemediationUpdater()
	checker := newTestExpiryChecker(repo, updater)

	tenantID := uuid.New()

	// Create expired exceptions (these are returned by FindExpired).
	exc1 := model.RiskException{
		ID:             uuid.New(),
		TenantID:       tenantID,
		ApprovalStatus: model.ApprovalApproved,
		Status:         model.ExceptionStatusActive,
		ExpiresAt:      time.Now().Add(-24 * time.Hour), // expired yesterday
		CreatedAt:      time.Now().Add(-30 * 24 * time.Hour),
		UpdatedAt:      time.Now().Add(-24 * time.Hour),
	}

	exc2 := model.RiskException{
		ID:             uuid.New(),
		TenantID:       tenantID,
		ApprovalStatus: model.ApprovalApproved,
		Status:         model.ExceptionStatusActive,
		ExpiresAt:      time.Now().Add(-48 * time.Hour), // expired 2 days ago
		CreatedAt:      time.Now().Add(-60 * 24 * time.Hour),
		UpdatedAt:      time.Now().Add(-48 * time.Hour),
	}

	repo.expired = []model.RiskException{exc1, exc2}

	count, err := checker.Run(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Verify both exceptions were updated with expired status.
	assert.Equal(t, 2, repo.updatedCount)
}

func TestRunExpiresExceptionStatusFields(t *testing.T) {
	repo := newMockExceptionRepo()
	updater := newMockRemediationUpdater()
	checker := newTestExpiryChecker(repo, updater)

	tenantID := uuid.New()

	exc := model.RiskException{
		ID:             uuid.New(),
		TenantID:       tenantID,
		ApprovalStatus: model.ApprovalApproved,
		Status:         model.ExceptionStatusActive,
		ExpiresAt:      time.Now().Add(-24 * time.Hour),
		CreatedAt:      time.Now().Add(-30 * 24 * time.Hour),
		UpdatedAt:      time.Now().Add(-24 * time.Hour),
	}

	repo.expired = []model.RiskException{exc}

	count, err := checker.Run(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// The Update call receives the modified exception pointer;
	// verify that it was called.
	assert.Equal(t, 1, repo.updatedCount)
}

func TestRunReopensLinkedRemediation(t *testing.T) {
	repo := newMockExceptionRepo()
	updater := newMockRemediationUpdater()
	checker := newTestExpiryChecker(repo, updater)

	tenantID := uuid.New()
	remediationID := uuid.New()

	exc := model.RiskException{
		ID:             uuid.New(),
		TenantID:       tenantID,
		RemediationID:  &remediationID,
		ApprovalStatus: model.ApprovalApproved,
		Status:         model.ExceptionStatusActive,
		ExpiresAt:      time.Now().Add(-24 * time.Hour),
		CreatedAt:      time.Now().Add(-30 * 24 * time.Hour),
		UpdatedAt:      time.Now().Add(-24 * time.Hour),
	}

	repo.expired = []model.RiskException{exc}

	count, err := checker.Run(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify the linked remediation was re-opened.
	assert.Equal(t, model.StatusOpen, updater.statusUpdates[remediationID])
}

func TestRunNoExpired(t *testing.T) {
	repo := newMockExceptionRepo()
	updater := newMockRemediationUpdater()
	checker := newTestExpiryChecker(repo, updater)

	repo.expired = []model.RiskException{} // no expired exceptions

	count, err := checker.Run(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Equal(t, 0, repo.updatedCount)
	assert.Empty(t, updater.statusUpdates)
}

func TestRunFindExpiredError(t *testing.T) {
	repo := newMockExceptionRepo()
	repo.findExpiredErr = assert.AnError
	updater := newMockRemediationUpdater()
	checker := newTestExpiryChecker(repo, updater)

	count, err := checker.Run(context.Background(), uuid.New())
	assert.Error(t, err)
	assert.Equal(t, 0, count)
	assert.Contains(t, err.Error(), "find expired")
}

func TestRunUpdateError(t *testing.T) {
	repo := newMockExceptionRepo()
	repo.updateErr = assert.AnError
	updater := newMockRemediationUpdater()
	checker := newTestExpiryChecker(repo, updater)

	exc := model.RiskException{
		ID:             uuid.New(),
		TenantID:       uuid.New(),
		ApprovalStatus: model.ApprovalApproved,
		Status:         model.ExceptionStatusActive,
		ExpiresAt:      time.Now().Add(-24 * time.Hour),
		CreatedAt:      time.Now().Add(-30 * 24 * time.Hour),
		UpdatedAt:      time.Now().Add(-24 * time.Hour),
	}
	repo.expired = []model.RiskException{exc}

	count, err := checker.Run(context.Background(), uuid.New())
	require.NoError(t, err)
	// Update failed, so expiredCount remains 0.
	assert.Equal(t, 0, count)
}

func TestRunNoRemediationNoReopenCall(t *testing.T) {
	repo := newMockExceptionRepo()
	updater := newMockRemediationUpdater()
	checker := newTestExpiryChecker(repo, updater)

	exc := model.RiskException{
		ID:             uuid.New(),
		TenantID:       uuid.New(),
		RemediationID:  nil, // no linked remediation
		ApprovalStatus: model.ApprovalApproved,
		Status:         model.ExceptionStatusActive,
		ExpiresAt:      time.Now().Add(-24 * time.Hour),
		CreatedAt:      time.Now().Add(-30 * 24 * time.Hour),
		UpdatedAt:      time.Now().Add(-24 * time.Hour),
	}
	repo.expired = []model.RiskException{exc}

	count, err := checker.Run(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Empty(t, updater.statusUpdates, "no remediation should be re-opened when RemediationID is nil")
}

func TestCheckReviews(t *testing.T) {
	repo := newMockExceptionRepo()
	updater := newMockRemediationUpdater()
	checker := newTestExpiryChecker(repo, updater)

	tenantID := uuid.New()
	pastReview := time.Now().Add(-24 * time.Hour)

	repo.needingReview = []model.RiskException{
		{
			ID:             uuid.New(),
			TenantID:       tenantID,
			ApprovalStatus: model.ApprovalApproved,
			Status:         model.ExceptionStatusActive,
			NextReviewAt:   &pastReview,
		},
		{
			ID:             uuid.New(),
			TenantID:       tenantID,
			ApprovalStatus: model.ApprovalApproved,
			Status:         model.ExceptionStatusActive,
			NextReviewAt:   &pastReview,
		},
	}

	exceptions, err := checker.CheckReviews(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Len(t, exceptions, 2)
}

func TestCheckReviewsNoOverdue(t *testing.T) {
	repo := newMockExceptionRepo()
	updater := newMockRemediationUpdater()
	checker := newTestExpiryChecker(repo, updater)

	repo.needingReview = []model.RiskException{}

	exceptions, err := checker.CheckReviews(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Empty(t, exceptions)
}

func TestCheckReviewsError(t *testing.T) {
	repo := newMockExceptionRepo()
	repo.findReviewErr = assert.AnError
	updater := newMockRemediationUpdater()
	checker := newTestExpiryChecker(repo, updater)

	exceptions, err := checker.CheckReviews(context.Background(), uuid.New())
	assert.Error(t, err)
	assert.Nil(t, exceptions)
	assert.Contains(t, err.Error(), "find needing review")
}
