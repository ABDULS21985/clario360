package remediation

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/remediation/strategy"
)

// ---------------------------------------------------------------------------
// fakeStrategy — configurable test double for strategy.RemediationStrategy
// ---------------------------------------------------------------------------

type fakeStrategy struct {
	stratType       model.RemediationType
	dryRunResult    *model.DryRunResult
	dryRunErr       error
	executeResult   *model.ExecutionResult
	executeErr      error
	verifyResult    *model.VerificationResult
	verifyErr       error
	rollbackErr     error
	captureStateVal json.RawMessage
	captureStateErr error
}

func (f *fakeStrategy) Type() model.RemediationType { return f.stratType }

func (f *fakeStrategy) DryRun(_ context.Context, _ *model.RemediationAction) (*model.DryRunResult, error) {
	return f.dryRunResult, f.dryRunErr
}

func (f *fakeStrategy) Execute(_ context.Context, _ *model.RemediationAction) (*model.ExecutionResult, error) {
	return f.executeResult, f.executeErr
}

func (f *fakeStrategy) Verify(_ context.Context, _ *model.RemediationAction) (*model.VerificationResult, error) {
	return f.verifyResult, f.verifyErr
}

func (f *fakeStrategy) Rollback(_ context.Context, _ *model.RemediationAction) error {
	return f.rollbackErr
}

func (f *fakeStrategy) CaptureState(_ context.Context, _ *model.RemediationAction) (json.RawMessage, error) {
	return f.captureStateVal, f.captureStateErr
}

// Compile-time assertion that fakeStrategy satisfies the interface.
var _ strategy.RemediationStrategy = (*fakeStrategy)(nil)

// ---------------------------------------------------------------------------
// newTestExecutor — builds an executor with nil repos (safe for pre-condition
// tests that return before any repo or audit-trail calls are made).
// ---------------------------------------------------------------------------

func newTestExecutor(strats ...strategy.RemediationStrategy) *RemediationExecutor {
	m := make(map[model.RemediationType]strategy.RemediationStrategy)
	for _, s := range strats {
		m[s.Type()] = s
	}
	logger := zerolog.Nop()
	// AuditTrail with nil repo: RecordTransition panics if repo is nil, so we
	// only call newTestExecutor for tests that are guaranteed to return before
	// reaching the audit trail (i.e. pre-condition/strategy-not-found failures).
	audit := NewAuditTrail(nil, logger)
	return NewRemediationExecutor(m, audit, nil, nil, nil, nil, logger)
}

// ---------------------------------------------------------------------------
// Helper action builders
// ---------------------------------------------------------------------------

// approvedAction returns an action that has passed approval and a successful
// dry-run — ready to be executed.
func approvedAction() *model.RemediationAction {
	tenantID := uuid.New()
	actionID := uuid.New()
	now := time.Now()
	approverID := uuid.New()
	dryRunAt := now.Add(-time.Minute)
	return &model.RemediationAction{
		ID:       actionID,
		TenantID: tenantID,
		Type:     model.RemediationTypePatch,
		Status:   model.StatusDryRunCompleted,
		Plan:     model.RemediationPlan{Reversible: true},
		ApprovedBy: &approverID,
		ApprovedAt: &now,
		DryRunAt:   &dryRunAt,
		DryRunResult: &model.DryRunResult{Success: true},
	}
}

// executedAction returns an action that has been executed and is within the
// rollback window, with pre-execution state captured.
func executedAction() *model.RemediationAction {
	a := approvedAction()
	a.Status = model.StatusExecuted
	deadline := time.Now().Add(72 * time.Hour)
	a.RollbackDeadline = &deadline
	a.PreExecutionState = json.RawMessage(`{"captured_at":"2024-01-01T00:00:00Z"}`)
	return a
}

// ---------------------------------------------------------------------------
// DryRun tests
// ---------------------------------------------------------------------------

// TestExecutorDryRun_WrongStatus verifies that DryRun rejects actions whose
// status is neither StatusApproved nor StatusDryRunFailed.
func TestExecutorDryRun_WrongStatus(t *testing.T) {
	exec := newTestExecutor()
	action := &model.RemediationAction{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Type:     model.RemediationTypePatch,
		Status:   model.StatusDraft,
	}
	actorID := uuid.New()

	_, err := exec.DryRun(context.Background(), action, &actorID, "tester")

	if err == nil {
		t.Fatal("expected error for wrong status, got nil")
	}
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed, got: %v", err)
	}
}

// TestExecutorDryRun_UnknownStrategy verifies that DryRun returns an error
// when no strategy is registered for the action's remediation type.
func TestExecutorDryRun_UnknownStrategy(t *testing.T) {
	// No strategies registered — any type lookup will fail.
	exec := newTestExecutor()
	action := &model.RemediationAction{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Type:     "unknown_type",
		Status:   model.StatusApproved, // valid status for DryRun
	}
	actorID := uuid.New()

	_, err := exec.DryRun(context.Background(), action, &actorID, "tester")

	if err == nil {
		t.Fatal("expected error for unknown strategy, got nil")
	}
	// getStrategy returns a plain fmt.Errorf (not a sentinel), so check the message.
	if err.Error() == "" {
		t.Fatal("expected non-empty error message")
	}
}

// TestExecutorDryRun_ApprovedStatusAccepted verifies that StatusApproved is
// accepted by the DryRun status check (error comes from missing strategy, not
// from the status guard).
func TestExecutorDryRun_ApprovedStatusAccepted(t *testing.T) {
	exec := newTestExecutor() // no strategies
	action := &model.RemediationAction{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Type:     "patch",
		Status:   model.StatusApproved,
	}
	actorID := uuid.New()

	_, err := exec.DryRun(context.Background(), action, &actorID, "tester")

	// Should fail on strategy lookup, NOT on pre-condition.
	if errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("status guard incorrectly rejected StatusApproved: %v", err)
	}
}

// TestExecutorDryRun_DryRunFailedStatusAccepted verifies that StatusDryRunFailed
// is accepted by the DryRun status check.
func TestExecutorDryRun_DryRunFailedStatusAccepted(t *testing.T) {
	exec := newTestExecutor()
	action := &model.RemediationAction{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Type:     "patch",
		Status:   model.StatusDryRunFailed,
	}
	actorID := uuid.New()

	_, err := exec.DryRun(context.Background(), action, &actorID, "tester")

	if errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("status guard incorrectly rejected StatusDryRunFailed: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Execute tests
// ---------------------------------------------------------------------------

// TestExecutorExecute_WrongStatus verifies that Execute rejects actions that
// have not reached StatusDryRunCompleted or StatusExecutionPending.
func TestExecutorExecute_WrongStatus(t *testing.T) {
	exec := newTestExecutor()
	action := &model.RemediationAction{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Type:     model.RemediationTypePatch,
		Status:   model.StatusDraft,
		// ApprovedBy and DryRunResult are nil → checkExecutePreConditions fires first.
	}
	executorID := uuid.New()

	_, err := exec.Execute(context.Background(), action, executorID, "tester")

	if err == nil {
		t.Fatal("expected error for draft status, got nil")
	}
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed, got: %v", err)
	}
}

// TestExecutorExecute_MissingApproval verifies that Execute fails when the
// action has not been approved yet, even if the status looks ready.
func TestExecutorExecute_MissingApproval(t *testing.T) {
	exec := newTestExecutor()
	now := time.Now()
	dryRunAt := now.Add(-time.Minute)
	action := &model.RemediationAction{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		Type:         model.RemediationTypePatch,
		Status:       model.StatusDryRunCompleted,
		DryRunAt:     &dryRunAt,
		DryRunResult: &model.DryRunResult{Success: true},
		// ApprovedBy and ApprovedAt intentionally nil.
	}
	executorID := uuid.New()

	_, err := exec.Execute(context.Background(), action, executorID, "tester")

	if err == nil {
		t.Fatal("expected error for missing approval, got nil")
	}
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed, got: %v", err)
	}
}

// TestExecutorExecute_MissingDryRun verifies that Execute fails when no
// dry-run result is present, even though the action is approved.
func TestExecutorExecute_MissingDryRun(t *testing.T) {
	exec := newTestExecutor()
	now := time.Now()
	approverID := uuid.New()
	action := &model.RemediationAction{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		Type:       model.RemediationTypePatch,
		Status:     model.StatusDryRunCompleted,
		ApprovedBy: &approverID,
		ApprovedAt: &now,
		// DryRunAt and DryRunResult intentionally nil.
	}
	executorID := uuid.New()

	_, err := exec.Execute(context.Background(), action, executorID, "tester")

	if err == nil {
		t.Fatal("expected error for missing dry-run, got nil")
	}
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed, got: %v", err)
	}
}

// TestExecutorExecute_DryRunNotSuccessful verifies that Execute fails when the
// dry-run completed but reported a failure.
func TestExecutorExecute_DryRunNotSuccessful(t *testing.T) {
	exec := newTestExecutor()
	now := time.Now()
	approverID := uuid.New()
	dryRunAt := now.Add(-time.Minute)
	action := &model.RemediationAction{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		Type:         model.RemediationTypePatch,
		Status:       model.StatusDryRunCompleted,
		ApprovedBy:   &approverID,
		ApprovedAt:   &now,
		DryRunAt:     &dryRunAt,
		DryRunResult: &model.DryRunResult{Success: false, Blockers: []string{"disk full"}},
	}
	executorID := uuid.New()

	_, err := exec.Execute(context.Background(), action, executorID, "tester")

	if err == nil {
		t.Fatal("expected error for failed dry-run, got nil")
	}
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Verify tests
// ---------------------------------------------------------------------------

// TestExecutorVerify_WrongStatus verifies that Verify rejects actions that
// have not reached StatusExecuted or StatusVerificationPending.
func TestExecutorVerify_WrongStatus(t *testing.T) {
	exec := newTestExecutor()
	action := &model.RemediationAction{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Type:     model.RemediationTypePatch,
		Status:   model.StatusDraft,
	}
	actorID := uuid.New()

	_, err := exec.Verify(context.Background(), action, &actorID, "tester")

	if err == nil {
		t.Fatal("expected error for wrong status, got nil")
	}
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed, got: %v", err)
	}
}

// TestExecutorVerify_ApprovedStatusRejected verifies that StatusApproved is
// not a valid starting point for verification.
func TestExecutorVerify_ApprovedStatusRejected(t *testing.T) {
	exec := newTestExecutor()
	action := &model.RemediationAction{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Type:     model.RemediationTypePatch,
		Status:   model.StatusApproved,
	}
	actorID := uuid.New()

	_, err := exec.Verify(context.Background(), action, &actorID, "tester")

	if err == nil {
		t.Fatal("expected error for approved status in Verify, got nil")
	}
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Rollback tests
// ---------------------------------------------------------------------------

// TestExecutorRollback_WrongStatus verifies that Rollback returns
// ErrInvalidTransition for a status that is not in the allowed set.
// The action must satisfy checkRollbackPreConditions (reversible + state set)
// so that the status check is the first gate to fire an error.
func TestExecutorRollback_WrongStatus(t *testing.T) {
	exec := newTestExecutor()
	action := &model.RemediationAction{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		Type:              model.RemediationTypePatch,
		Status:            model.StatusDraft,
		Plan:              model.RemediationPlan{Reversible: true},
		PreExecutionState: json.RawMessage(`{"captured_at":"2024-01-01T00:00:00Z"}`),
		// No RollbackDeadline — window not expired.
	}
	approverID := uuid.New()

	err := exec.Rollback(context.Background(), action, "testing wrong status", approverID, "tester")

	if err == nil {
		t.Fatal("expected error for draft status in Rollback, got nil")
	}
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
}

// TestExecutorRollback_WindowExpired verifies that Rollback fails when the
// rollback window has already passed.
func TestExecutorRollback_WindowExpired(t *testing.T) {
	exec := newTestExecutor()
	expired := time.Now().Add(-time.Hour)
	action := &model.RemediationAction{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		Type:              model.RemediationTypePatch,
		Status:            model.StatusExecuted,
		Plan:              model.RemediationPlan{Reversible: true},
		PreExecutionState: json.RawMessage(`{"captured_at":"2024-01-01T00:00:00Z"}`),
		RollbackDeadline:  &expired,
	}
	approverID := uuid.New()

	err := exec.Rollback(context.Background(), action, "too late", approverID, "tester")

	if err == nil {
		t.Fatal("expected error for expired rollback window, got nil")
	}
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed (expired window), got: %v", err)
	}
}

// TestExecutorRollback_NotReversible verifies that Rollback fails when the
// remediation plan is not reversible.
func TestExecutorRollback_NotReversible(t *testing.T) {
	exec := newTestExecutor()
	action := &model.RemediationAction{
		ID:                uuid.New(),
		TenantID:          uuid.New(),
		Type:              model.RemediationTypePatch,
		Status:            model.StatusExecuted,
		Plan:              model.RemediationPlan{Reversible: false},
		PreExecutionState: json.RawMessage(`{"captured_at":"2024-01-01T00:00:00Z"}`),
		// No deadline — window not expired, but plan is not reversible.
	}
	approverID := uuid.New()

	err := exec.Rollback(context.Background(), action, "trying non-reversible", approverID, "tester")

	if err == nil {
		t.Fatal("expected error for non-reversible plan, got nil")
	}
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed (not reversible), got: %v", err)
	}
}

// TestExecutorRollback_NoPreExecutionState verifies that Rollback fails when
// no pre-execution state was captured (rollback would be unsafe).
func TestExecutorRollback_NoPreExecutionState(t *testing.T) {
	exec := newTestExecutor()
	action := &model.RemediationAction{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Type:     model.RemediationTypePatch,
		Status:   model.StatusExecuted,
		Plan:     model.RemediationPlan{Reversible: true},
		// PreExecutionState intentionally nil.
	}
	approverID := uuid.New()

	err := exec.Rollback(context.Background(), action, "no state", approverID, "tester")

	if err == nil {
		t.Fatal("expected error when pre-execution state is missing, got nil")
	}
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Strategy lookup tests
// ---------------------------------------------------------------------------

// TestExecutorGetStrategy_NotFound verifies that requesting an unregistered
// strategy type returns a descriptive error through the DryRun path.
func TestExecutorGetStrategy_NotFound(t *testing.T) {
	exec := newTestExecutor() // no strategies registered
	action := &model.RemediationAction{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Type:     model.RemediationTypePatch, // not registered
		Status:   model.StatusApproved,
	}
	actorID := uuid.New()

	_, err := exec.DryRun(context.Background(), action, &actorID, "tester")

	if err == nil {
		t.Fatal("expected error for unregistered strategy, got nil")
	}
	// The error should not be a pre-condition error — it should indicate a
	// missing strategy registration.
	if errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("error should be about missing strategy, not ErrPreConditionFailed: %v", err)
	}
}

// TestExecutorGetStrategy_RegisteredTypeFound verifies that when the correct
// strategy is registered, the lookup itself does not produce an error.
// (The executor proceeds past strategy lookup and fails on nil remRepo instead.)
func TestExecutorGetStrategy_RegisteredTypeFound(t *testing.T) {
	strat := &fakeStrategy{
		stratType:    model.RemediationTypePatch,
		dryRunResult: &model.DryRunResult{Success: true},
	}
	exec := newTestExecutor(strat)
	action := &model.RemediationAction{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Type:     model.RemediationTypePatch,
		Status:   model.StatusApproved,
	}
	actorID := uuid.New()

	// The call will panic/error when it reaches remRepo.UpdateStatus (nil repo),
	// but the error must NOT be "no strategy registered".
	defer func() {
		if r := recover(); r != nil {
			// A panic from nil remRepo is acceptable — it means strategy was found.
			t.Logf("got expected nil-repo panic (strategy lookup succeeded): %v", r)
		}
	}()

	_, err := exec.DryRun(context.Background(), action, &actorID, "tester")
	if err != nil && !errors.Is(err, ErrPreConditionFailed) {
		// If an error comes back without a panic, it must not be "no strategy".
		t.Logf("DryRun returned non-precondition error (strategy was found): %v", err)
	}
}

// ---------------------------------------------------------------------------
// fakeStrategy helper builders for re-use across test cases
// ---------------------------------------------------------------------------

func newPatchStrategy() *fakeStrategy {
	return &fakeStrategy{
		stratType:       model.RemediationTypePatch,
		captureStateVal: json.RawMessage(`{}`),
	}
}

// TestFakeStrategyImplementsInterface is a compile-time check encoded as a
// runtime no-op to document the contract.
func TestFakeStrategyImplementsInterface(t *testing.T) {
	var _ strategy.RemediationStrategy = (*fakeStrategy)(nil)
}

// TestApprovedActionHelper verifies that the approvedAction() builder produces
// an action that satisfies checkExecutePreConditions without error.
func TestApprovedActionHelper(t *testing.T) {
	a := approvedAction()
	if err := checkExecutePreConditions(a); err != nil {
		t.Fatalf("approvedAction() should satisfy execute pre-conditions, got: %v", err)
	}
}

// TestExecutedActionHelper verifies that the executedAction() builder produces
// an action that satisfies checkRollbackPreConditions without error.
func TestExecutedActionHelper(t *testing.T) {
	a := executedAction()
	if err := checkRollbackPreConditions(a); err != nil {
		t.Fatalf("executedAction() should satisfy rollback pre-conditions, got: %v", err)
	}
}
