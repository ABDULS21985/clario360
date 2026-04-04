package exception

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/dto"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// mockExceptionRepo implements ExceptionRepository for tests.
type mockExceptionRepo struct {
	created       *model.RiskException
	exceptions    map[uuid.UUID]*model.RiskException
	updatedCount  int
	createErr     error
	getErr        error
	updateErr     error
	expired       []model.RiskException
	findExpiredErr error
	needingReview []model.RiskException
	findReviewErr error
}

func newMockExceptionRepo() *mockExceptionRepo {
	return &mockExceptionRepo{
		exceptions: make(map[uuid.UUID]*model.RiskException),
	}
}

func (m *mockExceptionRepo) Create(_ context.Context, exception *model.RiskException) (*model.RiskException, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.created = exception
	m.exceptions[exception.ID] = exception
	return exception, nil
}

func (m *mockExceptionRepo) GetByID(_ context.Context, _ uuid.UUID, exceptionID uuid.UUID) (*model.RiskException, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	exc, ok := m.exceptions[exceptionID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return exc, nil
}

func (m *mockExceptionRepo) Update(_ context.Context, exception *model.RiskException) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.exceptions[exception.ID] = exception
	m.updatedCount++
	return nil
}

func (m *mockExceptionRepo) ListByTenant(_ context.Context, _ uuid.UUID) ([]model.RiskException, error) {
	var result []model.RiskException
	for _, exc := range m.exceptions {
		result = append(result, *exc)
	}
	return result, nil
}

func (m *mockExceptionRepo) FindExpired(_ context.Context, _ uuid.UUID) ([]model.RiskException, error) {
	if m.findExpiredErr != nil {
		return nil, m.findExpiredErr
	}
	return m.expired, nil
}

func (m *mockExceptionRepo) FindNeedingReview(_ context.Context, _ uuid.UUID) ([]model.RiskException, error) {
	if m.findReviewErr != nil {
		return nil, m.findReviewErr
	}
	return m.needingReview, nil
}

// mockRemediationUpdater implements RemediationUpdater for tests.
type mockRemediationUpdater struct {
	statusUpdates map[uuid.UUID]model.RemediationStatus
	updateErr     error
}

func newMockRemediationUpdater() *mockRemediationUpdater {
	return &mockRemediationUpdater{
		statusUpdates: make(map[uuid.UUID]model.RemediationStatus),
	}
}

func (m *mockRemediationUpdater) UpdateStatus(_ context.Context, _ uuid.UUID, remediationID uuid.UUID, status model.RemediationStatus) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.statusUpdates[remediationID] = status
	return nil
}

func newTestExceptionManager(repo *mockExceptionRepo, updater *mockRemediationUpdater) *ExceptionManager {
	logger := zerolog.Nop()
	return NewExceptionManager(repo, updater, logger)
}

func validCreateExceptionRequest() *dto.CreateExceptionRequest {
	return &dto.CreateExceptionRequest{
		ExceptionType:        "posture_finding",
		Justification:        "Business critical system requires temporary exception",
		BusinessReason:       "System migration in progress",
		CompensatingControls: "Additional monitoring and alerting configured",
		RiskScore:            35.0,
		RiskLevel:            "medium",
		ExpiresAt:            time.Now().Add(30 * 24 * time.Hour),
		ReviewIntervalDays:   90,
	}
}

func TestRequestException(t *testing.T) {
	repo := newMockExceptionRepo()
	updater := newMockRemediationUpdater()
	mgr := newTestExceptionManager(repo, updater)

	tenantID := uuid.New()
	requestedBy := uuid.New()
	req := validCreateExceptionRequest()
	assetID := uuid.New()
	req.DataAssetID = &assetID

	exception, err := mgr.Request(context.Background(), tenantID, requestedBy, req)
	require.NoError(t, err)
	require.NotNil(t, exception)

	assert.NotEqual(t, uuid.Nil, exception.ID)
	assert.Equal(t, tenantID, exception.TenantID)
	assert.Equal(t, model.ExceptionPostureFinding, exception.ExceptionType)
	assert.Equal(t, requestedBy, exception.RequestedBy)
	assert.Equal(t, model.ApprovalPending, exception.ApprovalStatus)
	assert.Equal(t, model.ExceptionStatusActive, exception.Status)
	assert.Equal(t, req.Justification, exception.Justification)
	assert.Equal(t, req.BusinessReason, exception.BusinessReason)
	assert.Equal(t, req.CompensatingControls, exception.CompensatingControls)
	assert.Equal(t, req.RiskScore, exception.RiskScore)
	assert.Equal(t, req.RiskLevel, exception.RiskLevel)
	assert.Equal(t, 90, exception.ReviewIntervalDays)
	assert.Equal(t, 0, exception.ReviewCount)
	assert.NotNil(t, exception.NextReviewAt)
	assert.Equal(t, assetID, *exception.DataAssetID)
}

func TestRequestExceptionDefaultReviewInterval(t *testing.T) {
	repo := newMockExceptionRepo()
	updater := newMockRemediationUpdater()
	mgr := newTestExceptionManager(repo, updater)

	req := validCreateExceptionRequest()
	req.ReviewIntervalDays = 0 // should default to 90

	exception, err := mgr.Request(context.Background(), uuid.New(), uuid.New(), req)
	require.NoError(t, err)
	require.NotNil(t, exception)

	assert.Equal(t, 90, exception.ReviewIntervalDays)
}

func TestRequestExceptionValidationError(t *testing.T) {
	repo := newMockExceptionRepo()
	updater := newMockRemediationUpdater()
	mgr := newTestExceptionManager(repo, updater)

	req := &dto.CreateExceptionRequest{
		ExceptionType: "invalid_type",
		Justification: "test",
		RiskLevel:     "medium",
		RiskScore:     50,
		ExpiresAt:     time.Now().Add(24 * time.Hour),
	}

	exception, err := mgr.Request(context.Background(), uuid.New(), uuid.New(), req)
	assert.Error(t, err)
	assert.Nil(t, exception)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestApproveException(t *testing.T) {
	repo := newMockExceptionRepo()
	updater := newMockRemediationUpdater()
	mgr := newTestExceptionManager(repo, updater)

	tenantID := uuid.New()
	requestedBy := uuid.New()
	approverID := uuid.New()
	remediationID := uuid.New()

	// Create a pending exception with a linked remediation.
	exc := &model.RiskException{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		ExceptionType:      model.ExceptionPostureFinding,
		RemediationID:      &remediationID,
		RequestedBy:        requestedBy,
		ApprovalStatus:     model.ApprovalPending,
		Status:             model.ExceptionStatusActive,
		ReviewIntervalDays: 90,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	repo.exceptions[exc.ID] = exc

	approved, err := mgr.Approve(context.Background(), tenantID, exc.ID, approverID)
	require.NoError(t, err)
	require.NotNil(t, approved)

	assert.Equal(t, model.ApprovalApproved, approved.ApprovalStatus)
	assert.NotNil(t, approved.ApprovedBy)
	assert.Equal(t, approverID, *approved.ApprovedBy)
	assert.NotNil(t, approved.ApprovedAt)
	assert.NotNil(t, approved.LastReviewedAt)
	assert.NotNil(t, approved.NextReviewAt)
	assert.Equal(t, 1, approved.ReviewCount)

	// Linked remediation should be updated to exception_granted.
	assert.Equal(t, model.StatusExceptionGranted, updater.statusUpdates[remediationID])
}

func TestApproveExceptionWithoutRemediation(t *testing.T) {
	repo := newMockExceptionRepo()
	updater := newMockRemediationUpdater()
	mgr := newTestExceptionManager(repo, updater)

	tenantID := uuid.New()
	requestedBy := uuid.New()
	approverID := uuid.New()

	exc := &model.RiskException{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		RemediationID:      nil, // no linked remediation
		RequestedBy:        requestedBy,
		ApprovalStatus:     model.ApprovalPending,
		Status:             model.ExceptionStatusActive,
		ReviewIntervalDays: 90,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	repo.exceptions[exc.ID] = exc

	approved, err := mgr.Approve(context.Background(), tenantID, exc.ID, approverID)
	require.NoError(t, err)
	assert.Equal(t, model.ApprovalApproved, approved.ApprovalStatus)

	// No remediation status updates should have occurred.
	assert.Empty(t, updater.statusUpdates)
}

func TestApproveExceptionNotPending(t *testing.T) {
	repo := newMockExceptionRepo()
	updater := newMockRemediationUpdater()
	mgr := newTestExceptionManager(repo, updater)

	exc := &model.RiskException{
		ID:             uuid.New(),
		TenantID:       uuid.New(),
		RequestedBy:    uuid.New(),
		ApprovalStatus: model.ApprovalApproved, // already approved
		CreatedAt:      time.Now().UTC(),
	}
	repo.exceptions[exc.ID] = exc

	result, err := mgr.Approve(context.Background(), exc.TenantID, exc.ID, uuid.New())
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not in pending state")
}

func TestRejectException(t *testing.T) {
	repo := newMockExceptionRepo()
	updater := newMockRemediationUpdater()
	mgr := newTestExceptionManager(repo, updater)

	tenantID := uuid.New()
	requestedBy := uuid.New()
	rejecterID := uuid.New()
	reason := "Insufficient compensating controls documented"

	exc := &model.RiskException{
		ID:             uuid.New(),
		TenantID:       tenantID,
		RequestedBy:    requestedBy,
		ApprovalStatus: model.ApprovalPending,
		Status:         model.ExceptionStatusActive,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	repo.exceptions[exc.ID] = exc

	rejected, err := mgr.Reject(context.Background(), tenantID, exc.ID, rejecterID, reason)
	require.NoError(t, err)
	require.NotNil(t, rejected)

	assert.Equal(t, model.ApprovalRejected, rejected.ApprovalStatus)
	assert.Equal(t, reason, rejected.RejectionReason)
	assert.NotNil(t, rejected.ApprovedBy) // stores the rejecter
	assert.Equal(t, rejecterID, *rejected.ApprovedBy)
}

func TestRejectExceptionEmptyReason(t *testing.T) {
	repo := newMockExceptionRepo()
	updater := newMockRemediationUpdater()
	mgr := newTestExceptionManager(repo, updater)

	exc := &model.RiskException{
		ID:             uuid.New(),
		TenantID:       uuid.New(),
		RequestedBy:    uuid.New(),
		ApprovalStatus: model.ApprovalPending,
	}
	repo.exceptions[exc.ID] = exc

	result, err := mgr.Reject(context.Background(), exc.TenantID, exc.ID, uuid.New(), "")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "rejection reason is required")
}

func TestRejectExceptionNotPending(t *testing.T) {
	repo := newMockExceptionRepo()
	updater := newMockRemediationUpdater()
	mgr := newTestExceptionManager(repo, updater)

	exc := &model.RiskException{
		ID:             uuid.New(),
		TenantID:       uuid.New(),
		RequestedBy:    uuid.New(),
		ApprovalStatus: model.ApprovalRejected, // already rejected
		CreatedAt:      time.Now().UTC(),
	}
	repo.exceptions[exc.ID] = exc

	result, err := mgr.Reject(context.Background(), exc.TenantID, exc.ID, uuid.New(), "some reason")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not in pending state")
}

func TestMaxDurationEnforcement(t *testing.T) {
	req := &dto.CreateExceptionRequest{
		ExceptionType: "posture_finding",
		Justification: "test",
		RiskLevel:     "medium",
		RiskScore:     50,
		ExpiresAt:     time.Now().Add(400 * 24 * time.Hour), // 400 days, exceeds 365 max
	}

	err := req.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be more than 365 days")
}

func TestMaxDurationEnforcementPastDate(t *testing.T) {
	req := &dto.CreateExceptionRequest{
		ExceptionType: "posture_finding",
		Justification: "test",
		RiskLevel:     "medium",
		RiskScore:     50,
		ExpiresAt:     time.Now().Add(-24 * time.Hour), // in the past
	}

	err := req.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be in the future")
}

func TestMaxDurationEnforcementValid(t *testing.T) {
	req := &dto.CreateExceptionRequest{
		ExceptionType: "posture_finding",
		Justification: "test",
		RiskLevel:     "medium",
		RiskScore:     50,
		ExpiresAt:     time.Now().Add(90 * 24 * time.Hour), // 90 days, within limit
	}

	err := req.Validate()
	assert.NoError(t, err)
}

func TestSelfApprovalPrevention(t *testing.T) {
	repo := newMockExceptionRepo()
	updater := newMockRemediationUpdater()
	mgr := newTestExceptionManager(repo, updater)

	tenantID := uuid.New()
	userID := uuid.New() // same user requests and attempts to approve

	exc := &model.RiskException{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		RequestedBy:        userID,
		ApprovalStatus:     model.ApprovalPending,
		Status:             model.ExceptionStatusActive,
		ReviewIntervalDays: 90,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	repo.exceptions[exc.ID] = exc

	// Attempt self-approval.
	result, err := mgr.Approve(context.Background(), tenantID, exc.ID, userID)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "approver cannot be the same as the requester")
}

func TestRequestExceptionWithRemediationID(t *testing.T) {
	repo := newMockExceptionRepo()
	updater := newMockRemediationUpdater()
	mgr := newTestExceptionManager(repo, updater)

	tenantID := uuid.New()
	requestedBy := uuid.New()
	remediationID := uuid.New()
	policyID := uuid.New()

	req := validCreateExceptionRequest()
	req.RemediationID = &remediationID
	req.PolicyID = &policyID

	exception, err := mgr.Request(context.Background(), tenantID, requestedBy, req)
	require.NoError(t, err)
	require.NotNil(t, exception)

	assert.Equal(t, &remediationID, exception.RemediationID)
	assert.Equal(t, &policyID, exception.PolicyID)
}

func TestRequestExceptionRepoError(t *testing.T) {
	repo := newMockExceptionRepo()
	repo.createErr = fmt.Errorf("database connection failed")
	updater := newMockRemediationUpdater()
	mgr := newTestExceptionManager(repo, updater)

	req := validCreateExceptionRequest()
	exception, err := mgr.Request(context.Background(), uuid.New(), uuid.New(), req)
	assert.Error(t, err)
	assert.Nil(t, exception)
	assert.Contains(t, err.Error(), "create")
}
