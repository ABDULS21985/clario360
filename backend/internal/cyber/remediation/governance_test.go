package remediation

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

func draftAction() *model.RemediationAction {
	return &model.RemediationAction{
		Status: model.StatusDraft,
		Plan:   model.RemediationPlan{Reversible: true},
	}
}

// TestValidTransitionHappyPaths verifies that well-known legal transitions succeed.
func TestValidTransitionHappyPaths(t *testing.T) {
	cases := []struct {
		name   string
		from   model.RemediationStatus
		to     model.RemediationStatus
		role   string
		setup  func(*model.RemediationAction)
	}{
		{
			name: "analyst submits draft",
			from: model.StatusDraft,
			to:   model.StatusPendingApproval,
			role: "analyst",
		},
		{
			name: "security_manager approves",
			from: model.StatusPendingApproval,
			to:   model.StatusApproved,
			role: "security_manager",
		},
		{
			name: "security_manager rejects",
			from: model.StatusPendingApproval,
			to:   model.StatusRejected,
			role: "security_manager",
		},
		{
			name: "analyst starts dry-run",
			from: model.StatusApproved,
			to:   model.StatusDryRunRunning,
			role: "analyst",
		},
		{
			name: "system completes dry-run",
			from: model.StatusDryRunRunning,
			to:   model.StatusDryRunCompleted,
			role: "system",
		},
		{
			name: "system marks dry-run failed",
			from: model.StatusDryRunRunning,
			to:   model.StatusDryRunFailed,
			role: "system",
		},
		{
			name: "analyst queues execution after successful dry-run + approval",
			from: model.StatusDryRunCompleted,
			to:   model.StatusExecutionPending,
			role: "analyst",
			setup: func(a *model.RemediationAction) {
				now := time.Now()
				approverID := uuid.New()
				a.ApprovedBy = &approverID
				a.ApprovedAt = &now
				dryRunAt := now.Add(-time.Minute)
				a.DryRunAt = &dryRunAt
				a.DryRunResult = &model.DryRunResult{Success: true}
			},
		},
		{
			name: "analyst requests rollback from executed",
			from: model.StatusExecuted,
			to:   model.StatusRollbackPending,
			role: "analyst",
			setup: func(a *model.RemediationAction) {
				raw, _ := json.Marshal(map[string]string{"key": "val"})
				a.PreExecutionState = raw
			},
		},
		{
			name: "analyst closes verified action",
			from: model.StatusVerified,
			to:   model.StatusClosed,
			role: "analyst",
		},
		{
			name: "admin closes verified action (higher role)",
			from: model.StatusVerified,
			to:   model.StatusClosed,
			role: "admin",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := draftAction()
			a.Status = tc.from
			if tc.setup != nil {
				tc.setup(a)
			}
			if err := ValidateTransition(a, tc.to, tc.role); err != nil {
				t.Fatalf("expected valid transition, got: %v", err)
			}
		})
	}
}

// TestInvalidTransitionRole checks that insufficient role is rejected.
func TestInvalidTransitionRole(t *testing.T) {
	a := draftAction()
	a.Status = model.StatusPendingApproval
	err := ValidateTransition(a, model.StatusApproved, "analyst")
	if !errors.Is(err, ErrInsufficientPermission) {
		t.Fatalf("expected ErrInsufficientPermission, got %v", err)
	}
}

// TestInvalidTransitionUnknown checks that unknown transitions return ErrInvalidTransition.
func TestInvalidTransitionUnknown(t *testing.T) {
	a := draftAction()
	a.Status = model.StatusDraft
	err := ValidateTransition(a, model.StatusVerified, "admin")
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}
}

// TestPreConditionExecuteNoDryRun checks that execution cannot be queued without a completed dry-run.
func TestPreConditionExecuteNoDryRun(t *testing.T) {
	a := draftAction()
	a.Status = model.StatusDryRunCompleted
	now := time.Now()
	approverID := uuid.New()
	a.ApprovedBy = &approverID
	a.ApprovedAt = &now
	// DryRunResult intentionally nil
	err := ValidateTransition(a, model.StatusExecutionPending, "analyst")
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed, got %v", err)
	}
}

// TestPreConditionExecuteNoApproval checks that execution cannot be queued without approval.
func TestPreConditionExecuteNoApproval(t *testing.T) {
	a := draftAction()
	a.Status = model.StatusDryRunCompleted
	now := time.Now()
	a.DryRunAt = &now
	a.DryRunResult = &model.DryRunResult{Success: true}
	// No approval
	err := ValidateTransition(a, model.StatusExecutionPending, "analyst")
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed, got %v", err)
	}
}

// TestPreConditionRollbackNoState checks that rollback without pre-execution state fails.
func TestPreConditionRollbackNoState(t *testing.T) {
	a := draftAction()
	a.Status = model.StatusExecuted
	// PreExecutionState intentionally nil
	err := ValidateTransition(a, model.StatusRollbackPending, "analyst")
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed, got %v", err)
	}
}

// TestPreConditionRollbackNotReversible checks that non-reversible plans block rollback.
func TestPreConditionRollbackNotReversible(t *testing.T) {
	a := draftAction()
	a.Status = model.StatusExecuted
	a.Plan.Reversible = false
	raw, _ := json.Marshal(map[string]string{"k": "v"})
	a.PreExecutionState = raw
	err := ValidateTransition(a, model.StatusRollbackPending, "analyst")
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed (not reversible), got %v", err)
	}
}

// TestPreConditionRollbackExpiredWindow checks that an expired rollback window blocks rollback.
func TestPreConditionRollbackExpiredWindow(t *testing.T) {
	a := draftAction()
	a.Status = model.StatusExecuted
	raw, _ := json.Marshal(map[string]string{"k": "v"})
	a.PreExecutionState = raw
	expired := time.Now().Add(-time.Hour)
	a.RollbackDeadline = &expired
	err := ValidateTransition(a, model.StatusRollbackPending, "analyst")
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed (expired window), got %v", err)
	}
}

// TestIsTerminalStatus checks terminal state classification.
func TestIsTerminalStatus(t *testing.T) {
	terminals := []model.RemediationStatus{
		model.StatusClosed, model.StatusRejected, model.StatusRollbackFailed,
	}
	for _, s := range terminals {
		if !IsTerminalStatus(s) {
			t.Errorf("expected %s to be terminal", s)
		}
	}
	nonTerminals := []model.RemediationStatus{
		model.StatusDraft, model.StatusExecuting, model.StatusVerified,
	}
	for _, s := range nonTerminals {
		if IsTerminalStatus(s) {
			t.Errorf("expected %s to NOT be terminal", s)
		}
	}
}

// TestIsPreExecutionStatus ensures pre-execution states are classified correctly.
func TestIsPreExecutionStatus(t *testing.T) {
	pre := []model.RemediationStatus{
		model.StatusDraft, model.StatusPendingApproval, model.StatusApproved,
		model.StatusRevisionRequested, model.StatusRejected,
		model.StatusDryRunRunning, model.StatusDryRunCompleted, model.StatusDryRunFailed,
		model.StatusExecutionPending,
	}
	for _, s := range pre {
		if !IsPreExecutionStatus(s) {
			t.Errorf("expected %s to be pre-execution", s)
		}
	}
	post := []model.RemediationStatus{
		model.StatusExecuting, model.StatusExecuted, model.StatusVerified, model.StatusClosed,
	}
	for _, s := range post {
		if IsPreExecutionStatus(s) {
			t.Errorf("expected %s to NOT be pre-execution", s)
		}
	}
}
