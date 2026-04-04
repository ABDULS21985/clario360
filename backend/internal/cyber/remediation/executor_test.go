package remediation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/remediation/strategy"
)

// ---------------------------------------------------------------------------
// Compile-time interface assertions
// ---------------------------------------------------------------------------

var _ strategy.RemediationStrategy = (*fakeStrategy)(nil)
var _ remediationRepo = (*fakeRemRepo)(nil)
var _ alertRepo = (*fakeAlertRepo)(nil)
var _ vulnerabilityRepo = (*fakeVulnRepo)(nil)
var _ auditRecorder = (*fakeAudit)(nil)

// ---------------------------------------------------------------------------
// fakeStrategy — configurable test double for strategy.RemediationStrategy
// ---------------------------------------------------------------------------

type fakeStrategy struct {
	typeFn         func() model.RemediationType
	dryRunFn       func(ctx context.Context, action *model.RemediationAction) (*model.DryRunResult, error)
	executeFn      func(ctx context.Context, action *model.RemediationAction) (*model.ExecutionResult, error)
	verifyFn       func(ctx context.Context, action *model.RemediationAction) (*model.VerificationResult, error)
	rollbackFn     func(ctx context.Context, action *model.RemediationAction) error
	captureStateFn func(ctx context.Context, action *model.RemediationAction) (json.RawMessage, error)
}

func (f *fakeStrategy) Type() model.RemediationType {
	if f.typeFn != nil {
		return f.typeFn()
	}
	return model.RemediationTypeCustom
}

func (f *fakeStrategy) DryRun(ctx context.Context, action *model.RemediationAction) (*model.DryRunResult, error) {
	if f.dryRunFn != nil {
		return f.dryRunFn(ctx, action)
	}
	return &model.DryRunResult{Success: true}, nil
}

func (f *fakeStrategy) Execute(ctx context.Context, action *model.RemediationAction) (*model.ExecutionResult, error) {
	if f.executeFn != nil {
		return f.executeFn(ctx, action)
	}
	return &model.ExecutionResult{Success: true, StepsExecuted: 2, StepsTotal: 2}, nil
}

func (f *fakeStrategy) Verify(ctx context.Context, action *model.RemediationAction) (*model.VerificationResult, error) {
	if f.verifyFn != nil {
		return f.verifyFn(ctx, action)
	}
	return &model.VerificationResult{Verified: true}, nil
}

func (f *fakeStrategy) Rollback(ctx context.Context, action *model.RemediationAction) error {
	if f.rollbackFn != nil {
		return f.rollbackFn(ctx, action)
	}
	return nil
}

func (f *fakeStrategy) CaptureState(ctx context.Context, action *model.RemediationAction) (json.RawMessage, error) {
	if f.captureStateFn != nil {
		return f.captureStateFn(ctx, action)
	}
	return json.RawMessage(`{"captured":"ok"}`), nil
}

// ---------------------------------------------------------------------------
// fakeRemRepo — implements remediationRepo, records all UpdateStatus calls
// ---------------------------------------------------------------------------

type statusUpdate struct {
	TenantID uuid.UUID
	ID       uuid.UUID
	Status   model.RemediationStatus
	Fields   map[string]interface{}
}

type fakeRemRepo struct {
	statuses    []statusUpdate
	updateError error
}

func (f *fakeRemRepo) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status model.RemediationStatus, fields map[string]interface{}) error {
	f.statuses = append(f.statuses, statusUpdate{
		TenantID: tenantID,
		ID:       id,
		Status:   status,
		Fields:   fields,
	})
	return f.updateError
}

// ---------------------------------------------------------------------------
// fakeAlertRepo — implements alertRepo
// ---------------------------------------------------------------------------

type fakeAlertRepo struct {
	getByIDFn      func(ctx context.Context, tenantID, alertID uuid.UUID) (*model.Alert, error)
	updateStatusFn func(ctx context.Context, tenantID, alertID uuid.UUID, status model.AlertStatus, notes, reason *string) (*model.Alert, error)
	createFn       func(ctx context.Context, alert *model.Alert) (*model.Alert, error)
	createdAlerts  []*model.Alert
}

func (f *fakeAlertRepo) GetByID(ctx context.Context, tenantID, alertID uuid.UUID) (*model.Alert, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, tenantID, alertID)
	}
	return &model.Alert{ID: alertID, TenantID: tenantID, Status: model.AlertStatusNew}, nil
}

func (f *fakeAlertRepo) UpdateStatus(ctx context.Context, tenantID, alertID uuid.UUID, status model.AlertStatus, notes, reason *string) (*model.Alert, error) {
	if f.updateStatusFn != nil {
		return f.updateStatusFn(ctx, tenantID, alertID, status, notes, reason)
	}
	return &model.Alert{ID: alertID, TenantID: tenantID, Status: status}, nil
}

func (f *fakeAlertRepo) Create(ctx context.Context, alert *model.Alert) (*model.Alert, error) {
	f.createdAlerts = append(f.createdAlerts, alert)
	if f.createFn != nil {
		return f.createFn(ctx, alert)
	}
	alert.ID = uuid.New()
	return alert, nil
}

// ---------------------------------------------------------------------------
// fakeVulnRepo — implements vulnerabilityRepo
// ---------------------------------------------------------------------------

type vulnStatusUpdate struct {
	TenantID uuid.UUID
	VulnID   uuid.UUID
	Status   string
	Notes    *string
}

type fakeVulnRepo struct {
	getByIDFn            func(ctx context.Context, tenantID, vulnID uuid.UUID) (*model.Vulnerability, error)
	updateStatusGlobalFn func(ctx context.Context, tenantID, vulnID uuid.UUID, status string, notes *string) (*model.Vulnerability, error)
	statusUpdates        []vulnStatusUpdate
}

func (f *fakeVulnRepo) GetByID(ctx context.Context, tenantID, vulnID uuid.UUID) (*model.Vulnerability, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, tenantID, vulnID)
	}
	return &model.Vulnerability{ID: vulnID, TenantID: tenantID, Status: "open"}, nil
}

func (f *fakeVulnRepo) UpdateStatusGlobal(ctx context.Context, tenantID, vulnID uuid.UUID, status string, notes *string) (*model.Vulnerability, error) {
	f.statusUpdates = append(f.statusUpdates, vulnStatusUpdate{
		TenantID: tenantID,
		VulnID:   vulnID,
		Status:   status,
		Notes:    notes,
	})
	if f.updateStatusGlobalFn != nil {
		return f.updateStatusGlobalFn(ctx, tenantID, vulnID, status, notes)
	}
	return &model.Vulnerability{ID: vulnID, TenantID: tenantID, Status: status}, nil
}

// ---------------------------------------------------------------------------
// fakeAudit — implements auditRecorder, captures all recorded events
// ---------------------------------------------------------------------------

type auditTransition struct {
	TenantID      uuid.UUID
	RemediationID uuid.UUID
	Action        string
	ActorID       *uuid.UUID
	ActorName     string
	OldStatus     model.RemediationStatus
	NewStatus     model.RemediationStatus
	Details       map[string]interface{}
}

type auditStep struct {
	TenantID      uuid.UUID
	RemediationID uuid.UUID
	StepNum       int
	StepAction    string
	StepResult    string
	DurationMs    int64
	ErrMsg        string
	Details       map[string]interface{}
}

type auditAction struct {
	TenantID      uuid.UUID
	RemediationID uuid.UUID
	Action        string
	ActorID       *uuid.UUID
	ActorName     string
	Details       map[string]interface{}
}

type fakeAudit struct {
	transitions []auditTransition
	steps       []auditStep
	actions     []auditAction
}

func (f *fakeAudit) RecordTransition(ctx context.Context, tenantID, remediationID uuid.UUID, action string, actorID *uuid.UUID, actorName string, oldStatus, newStatus model.RemediationStatus, details map[string]interface{}) {
	f.transitions = append(f.transitions, auditTransition{
		TenantID:      tenantID,
		RemediationID: remediationID,
		Action:        action,
		ActorID:       actorID,
		ActorName:     actorName,
		OldStatus:     oldStatus,
		NewStatus:     newStatus,
		Details:       details,
	})
}

func (f *fakeAudit) RecordStep(ctx context.Context, tenantID, remediationID uuid.UUID, stepNum int, stepAction, stepResult string, durationMs int64, errMsg string, details map[string]interface{}) {
	f.steps = append(f.steps, auditStep{
		TenantID:      tenantID,
		RemediationID: remediationID,
		StepNum:       stepNum,
		StepAction:    stepAction,
		StepResult:    stepResult,
		DurationMs:    durationMs,
		ErrMsg:        errMsg,
		Details:       details,
	})
}

func (f *fakeAudit) RecordAction(ctx context.Context, tenantID, remediationID uuid.UUID, action string, actorID *uuid.UUID, actorName string, details map[string]interface{}) {
	f.actions = append(f.actions, auditAction{
		TenantID:      tenantID,
		RemediationID: remediationID,
		Action:        action,
		ActorID:       actorID,
		ActorName:     actorName,
		Details:       details,
	})
}

// ---------------------------------------------------------------------------
// Helper builders
// ---------------------------------------------------------------------------

func testAction(status model.RemediationStatus) *model.RemediationAction {
	return &model.RemediationAction{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		Type:     model.RemediationTypeCustom,
		Title:    "Test remediation",
		Status:   status,
		Plan: model.RemediationPlan{
			Steps: []model.RemediationStep{
				{Number: 1, Action: "step-one", Description: "First step"},
				{Number: 2, Action: "step-two", Description: "Second step"},
			},
			Reversible:        true,
			EstimatedDowntime: "5m",
			RiskLevel:         "medium",
		},
		AffectedAssetIDs: []uuid.UUID{uuid.New()},
		CreatedBy:        uuid.New(),
	}
}

func approvedAction() *model.RemediationAction {
	a := testAction(model.StatusApproved)
	approver := uuid.New()
	now := time.Now().UTC()
	a.ApprovedBy = &approver
	a.ApprovedAt = &now
	return a
}

func dryRunCompletedAction() *model.RemediationAction {
	a := approvedAction()
	a.Status = model.StatusDryRunCompleted
	now := time.Now().UTC()
	a.DryRunAt = &now
	a.DryRunResult = &model.DryRunResult{Success: true}
	return a
}

func executedAction() *model.RemediationAction {
	a := dryRunCompletedAction()
	a.Status = model.StatusExecuted
	a.PreExecutionState = json.RawMessage(`{"captured_at":"2025-01-01T00:00:00Z"}`)
	a.Plan.Reversible = true
	deadline := time.Now().UTC().Add(72 * time.Hour)
	a.RollbackDeadline = &deadline
	return a
}

func newTestExecutor(strat strategy.RemediationStrategy, rr *fakeRemRepo, ar *fakeAlertRepo, vr *fakeVulnRepo, audit *fakeAudit) *RemediationExecutor {
	strategies := map[model.RemediationType]strategy.RemediationStrategy{
		model.RemediationTypeCustom: strat,
	}
	return NewRemediationExecutor(strategies, audit, rr, ar, vr, nil, zerolog.Nop())
}

// ---------------------------------------------------------------------------
// DryRun Tests
// ---------------------------------------------------------------------------

func TestDryRun_HappyPath(t *testing.T) {
	strat := &fakeStrategy{
		dryRunFn: func(ctx context.Context, action *model.RemediationAction) (*model.DryRunResult, error) {
			return &model.DryRunResult{
				Success:  true,
				Warnings: []string{"minor warning"},
			}, nil
		},
	}
	rr := &fakeRemRepo{}
	audit := &fakeAudit{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, audit)

	action := approvedAction()
	actorID := uuid.New()

	result, err := exec.DryRun(context.Background(), action, &actorID, "tester")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatal("expected Success=true")
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != "minor warning" {
		t.Fatalf("unexpected warnings: %v", result.Warnings)
	}

	// Verify remRepo received: dry_run_running, then dry_run_completed
	if len(rr.statuses) != 2 {
		t.Fatalf("expected 2 status updates, got %d", len(rr.statuses))
	}
	if rr.statuses[0].Status != model.StatusDryRunRunning {
		t.Errorf("first update should be dry_run_running, got %s", rr.statuses[0].Status)
	}
	if rr.statuses[1].Status != model.StatusDryRunCompleted {
		t.Errorf("second update should be dry_run_completed, got %s", rr.statuses[1].Status)
	}

	// Verify both updates target the correct action
	if rr.statuses[0].ID != action.ID {
		t.Errorf("first update ID mismatch: got %s, want %s", rr.statuses[0].ID, action.ID)
	}
	if rr.statuses[0].TenantID != action.TenantID {
		t.Errorf("first update TenantID mismatch: got %s, want %s", rr.statuses[0].TenantID, action.TenantID)
	}

	// Verify audit transitions recorded
	if len(audit.transitions) < 2 {
		t.Fatalf("expected at least 2 audit transitions, got %d", len(audit.transitions))
	}
	if audit.transitions[0].Action != "dry_run_started" {
		t.Errorf("first transition should be dry_run_started, got %s", audit.transitions[0].Action)
	}
	if audit.transitions[0].OldStatus != model.StatusApproved {
		t.Errorf("first transition old status should be approved, got %s", audit.transitions[0].OldStatus)
	}
	if audit.transitions[0].NewStatus != model.StatusDryRunRunning {
		t.Errorf("first transition new status should be dry_run_running, got %s", audit.transitions[0].NewStatus)
	}
	if audit.transitions[1].Action != "dry_run_completed" {
		t.Errorf("second transition should be dry_run_completed, got %s", audit.transitions[1].Action)
	}
	if audit.transitions[1].NewStatus != model.StatusDryRunCompleted {
		t.Errorf("second transition new status should be dry_run_completed, got %s", audit.transitions[1].NewStatus)
	}
}

func TestDryRun_StrategyError(t *testing.T) {
	stratErr := errors.New("strategy dry-run boom")
	strat := &fakeStrategy{
		dryRunFn: func(ctx context.Context, action *model.RemediationAction) (*model.DryRunResult, error) {
			return nil, stratErr
		},
	}
	rr := &fakeRemRepo{}
	audit := &fakeAudit{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, audit)

	action := approvedAction()

	result, err := exec.DryRun(context.Background(), action, nil, "system")
	if result != nil {
		t.Fatal("expected nil result on strategy error")
	}
	if !errors.Is(err, stratErr) {
		t.Fatalf("expected strategy error, got: %v", err)
	}

	// Should have: dry_run_running, then dry_run_failed
	if len(rr.statuses) != 2 {
		t.Fatalf("expected 2 status updates, got %d", len(rr.statuses))
	}
	if rr.statuses[0].Status != model.StatusDryRunRunning {
		t.Errorf("first update should be dry_run_running, got %s", rr.statuses[0].Status)
	}
	if rr.statuses[1].Status != model.StatusDryRunFailed {
		t.Errorf("second update should be dry_run_failed, got %s", rr.statuses[1].Status)
	}

	// Audit should record the failure transition
	if len(audit.transitions) < 2 {
		t.Fatalf("expected at least 2 audit transitions, got %d", len(audit.transitions))
	}
	if audit.transitions[1].Action != "dry_run_failed" {
		t.Errorf("second transition action should be dry_run_failed, got %s", audit.transitions[1].Action)
	}
	if audit.transitions[1].NewStatus != model.StatusDryRunFailed {
		t.Errorf("second transition new status should be dry_run_failed, got %s", audit.transitions[1].NewStatus)
	}
}

func TestDryRun_ResultNotSuccessful(t *testing.T) {
	strat := &fakeStrategy{
		dryRunFn: func(ctx context.Context, action *model.RemediationAction) (*model.DryRunResult, error) {
			return &model.DryRunResult{
				Success:  false,
				Blockers: []string{"critical blocker found"},
			}, nil
		},
	}
	rr := &fakeRemRepo{}
	audit := &fakeAudit{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, audit)

	action := approvedAction()

	result, err := exec.DryRun(context.Background(), action, nil, "system")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Fatal("expected Success=false")
	}
	if len(result.Blockers) != 1 || result.Blockers[0] != "critical blocker found" {
		t.Fatalf("unexpected blockers: %v", result.Blockers)
	}

	// Should have: dry_run_running, then dry_run_failed
	if len(rr.statuses) != 2 {
		t.Fatalf("expected 2 status updates, got %d", len(rr.statuses))
	}
	if rr.statuses[0].Status != model.StatusDryRunRunning {
		t.Errorf("first update should be dry_run_running, got %s", rr.statuses[0].Status)
	}
	if rr.statuses[1].Status != model.StatusDryRunFailed {
		t.Errorf("second update should be dry_run_failed, got %s", rr.statuses[1].Status)
	}

	// Audit should record failure transition
	found := false
	for _, tr := range audit.transitions {
		if tr.Action == "dry_run_failed" && tr.NewStatus == model.StatusDryRunFailed {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected audit transition with action=dry_run_failed and new status=dry_run_failed")
	}
}

func TestDryRun_WrongStatus(t *testing.T) {
	strat := &fakeStrategy{}
	rr := &fakeRemRepo{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, &fakeAudit{})

	action := testAction(model.StatusDraft)

	_, err := exec.DryRun(context.Background(), action, nil, "system")
	if err == nil {
		t.Fatal("expected error for wrong status")
	}
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed, got: %v", err)
	}

	// No status updates should have happened
	if len(rr.statuses) != 0 {
		t.Errorf("expected 0 status updates, got %d", len(rr.statuses))
	}
}

func TestDryRun_UnknownStrategy(t *testing.T) {
	strat := &fakeStrategy{}
	rr := &fakeRemRepo{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, &fakeAudit{})

	action := approvedAction()
	action.Type = "unknown_type"

	_, err := exec.DryRun(context.Background(), action, nil, "system")
	if err == nil {
		t.Fatal("expected error for unknown strategy type")
	}
	expected := "no strategy registered for type 'unknown_type'"
	if err.Error() != expected {
		t.Fatalf("expected error %q, got %q", expected, err.Error())
	}

	// No status updates since the error occurs before any repo call
	if len(rr.statuses) != 0 {
		t.Errorf("expected 0 status updates, got %d", len(rr.statuses))
	}
}

// ---------------------------------------------------------------------------
// Execute Tests
// ---------------------------------------------------------------------------

func TestExecute_HappyPath(t *testing.T) {
	strat := &fakeStrategy{
		executeFn: func(ctx context.Context, action *model.RemediationAction) (*model.ExecutionResult, error) {
			return &model.ExecutionResult{
				Success:       true,
				StepsExecuted: 2,
				StepsTotal:    2,
				StepResults: []model.StepResult{
					{StepNumber: 1, Action: "step-one", Status: "success", DurationMs: 100, Output: "ok"},
					{StepNumber: 2, Action: "step-two", Status: "success", DurationMs: 200, Output: "done"},
				},
			}, nil
		},
		captureStateFn: func(ctx context.Context, action *model.RemediationAction) (json.RawMessage, error) {
			return json.RawMessage(`{"state":"captured"}`), nil
		},
	}
	rr := &fakeRemRepo{}
	audit := &fakeAudit{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, audit)

	action := dryRunCompletedAction()
	executedBy := uuid.New()

	result, err := exec.Execute(context.Background(), action, executedBy, "executor-user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatal("expected Success=true")
	}
	if result.StepsExecuted != 2 {
		t.Errorf("expected 2 steps executed, got %d", result.StepsExecuted)
	}

	// Should have: executing, then executed
	if len(rr.statuses) != 2 {
		t.Fatalf("expected 2 status updates, got %d", len(rr.statuses))
	}
	if rr.statuses[0].Status != model.StatusExecuting {
		t.Errorf("first update should be executing, got %s", rr.statuses[0].Status)
	}
	if rr.statuses[1].Status != model.StatusExecuted {
		t.Errorf("second update should be executed, got %s", rr.statuses[1].Status)
	}

	// Verify pre_execution_state and executed_by were passed to the first update
	if _, ok := rr.statuses[0].Fields["pre_execution_state"]; !ok {
		t.Error("expected pre_execution_state in first status update fields")
	}
	if eb, ok := rr.statuses[0].Fields["executed_by"]; !ok || eb != executedBy {
		t.Error("expected executed_by in first status update fields matching the given ID")
	}

	// Verify step audit entries recorded for each step
	if len(audit.steps) != 2 {
		t.Fatalf("expected 2 step audit entries, got %d", len(audit.steps))
	}
	if audit.steps[0].StepAction != "step-one" {
		t.Errorf("first step audit action should be step-one, got %s", audit.steps[0].StepAction)
	}
	if audit.steps[0].StepResult != "success" {
		t.Errorf("first step audit result should be success, got %s", audit.steps[0].StepResult)
	}
	if audit.steps[1].StepAction != "step-two" {
		t.Errorf("second step audit action should be step-two, got %s", audit.steps[1].StepAction)
	}

	// Verify transitions: execution_started, then execution_done
	if len(audit.transitions) < 2 {
		t.Fatalf("expected at least 2 transitions, got %d", len(audit.transitions))
	}
	if audit.transitions[0].Action != "execution_started" {
		t.Errorf("first transition should be execution_started, got %s", audit.transitions[0].Action)
	}
	if audit.transitions[1].Action != "execution_done" {
		t.Errorf("second transition should be execution_done, got %s", audit.transitions[1].Action)
	}
}

func TestExecute_StrategyFailure_Reversible(t *testing.T) {
	rollbackCalled := false
	strat := &fakeStrategy{
		executeFn: func(ctx context.Context, action *model.RemediationAction) (*model.ExecutionResult, error) {
			return nil, errors.New("execution boom")
		},
		rollbackFn: func(ctx context.Context, action *model.RemediationAction) error {
			rollbackCalled = true
			return nil
		},
		captureStateFn: func(ctx context.Context, action *model.RemediationAction) (json.RawMessage, error) {
			return json.RawMessage(`{"state":"captured"}`), nil
		},
	}
	rr := &fakeRemRepo{}
	audit := &fakeAudit{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, audit)

	action := dryRunCompletedAction()
	action.Plan.Reversible = true

	_, err := exec.Execute(context.Background(), action, uuid.New(), "executor")
	if err == nil {
		t.Fatal("expected error")
	}

	// Auto-rollback should have been called for reversible plans
	if !rollbackCalled {
		t.Error("expected auto-rollback to be called for reversible plan")
	}

	// Status should include execution_failed
	foundFailed := false
	for _, su := range rr.statuses {
		if su.Status == model.StatusExecutionFailed {
			foundFailed = true
			break
		}
	}
	if !foundFailed {
		t.Error("expected execution_failed status update")
	}

	// Audit should have auto_rollback_completed action (successful rollback)
	foundAutoRollback := false
	for _, a := range audit.actions {
		if a.Action == "auto_rollback_completed" {
			foundAutoRollback = true
			break
		}
	}
	if !foundAutoRollback {
		t.Error("expected auto_rollback_completed audit action")
	}
}

func TestExecute_StrategyFailure_NonReversible(t *testing.T) {
	rollbackCalled := false
	strat := &fakeStrategy{
		executeFn: func(ctx context.Context, action *model.RemediationAction) (*model.ExecutionResult, error) {
			return nil, errors.New("execution boom")
		},
		rollbackFn: func(ctx context.Context, action *model.RemediationAction) error {
			rollbackCalled = true
			return nil
		},
		captureStateFn: func(ctx context.Context, action *model.RemediationAction) (json.RawMessage, error) {
			return json.RawMessage(`{"state":"captured"}`), nil
		},
	}
	rr := &fakeRemRepo{}
	audit := &fakeAudit{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, audit)

	action := dryRunCompletedAction()
	action.Plan.Reversible = false

	_, err := exec.Execute(context.Background(), action, uuid.New(), "executor")
	if err == nil {
		t.Fatal("expected error")
	}

	// Auto-rollback should NOT have been called for non-reversible plans
	if rollbackCalled {
		t.Error("expected auto-rollback NOT to be called for non-reversible plan")
	}

	// Status should include execution_failed
	foundFailed := false
	for _, su := range rr.statuses {
		if su.Status == model.StatusExecutionFailed {
			foundFailed = true
			break
		}
	}
	if !foundFailed {
		t.Error("expected execution_failed status update")
	}

	// No auto-rollback audit actions should exist
	for _, a := range audit.actions {
		if a.Action == "auto_rollback_completed" || a.Action == "auto_rollback_failed" {
			t.Errorf("unexpected auto-rollback audit action: %s", a.Action)
		}
	}
}

func TestExecute_AutoRollback_AlsoFails(t *testing.T) {
	strat := &fakeStrategy{
		executeFn: func(ctx context.Context, action *model.RemediationAction) (*model.ExecutionResult, error) {
			return nil, errors.New("execution boom")
		},
		rollbackFn: func(ctx context.Context, action *model.RemediationAction) error {
			return errors.New("rollback also boom")
		},
		captureStateFn: func(ctx context.Context, action *model.RemediationAction) (json.RawMessage, error) {
			return json.RawMessage(`{"state":"captured"}`), nil
		},
	}
	rr := &fakeRemRepo{}
	ar := &fakeAlertRepo{}
	audit := &fakeAudit{}
	exec := newTestExecutor(strat, rr, ar, &fakeVulnRepo{}, audit)

	action := dryRunCompletedAction()
	action.Plan.Reversible = true

	_, err := exec.Execute(context.Background(), action, uuid.New(), "executor")
	if err == nil {
		t.Fatal("expected error")
	}

	// An alert should have been created for the rollback failure
	if len(ar.createdAlerts) != 1 {
		t.Fatalf("expected 1 alert created for rollback failure, got %d", len(ar.createdAlerts))
	}
	createdAlert := ar.createdAlerts[0]
	if createdAlert.Severity != model.SeverityCritical {
		t.Errorf("expected critical severity, got %s", createdAlert.Severity)
	}
	if createdAlert.Status != model.AlertStatusNew {
		t.Errorf("expected alert status=new, got %s", createdAlert.Status)
	}
	if createdAlert.Source != "remediation" {
		t.Errorf("expected source=remediation, got %s", createdAlert.Source)
	}
	if createdAlert.TenantID != action.TenantID {
		t.Errorf("expected alert tenant ID to match action tenant ID")
	}

	// Audit should have auto_rollback_failed action
	foundFailedRollback := false
	for _, a := range audit.actions {
		if a.Action == "auto_rollback_failed" {
			foundFailedRollback = true
			break
		}
	}
	if !foundFailedRollback {
		t.Error("expected auto_rollback_failed audit action")
	}
}

func TestExecute_ResultNotSuccessful(t *testing.T) {
	strat := &fakeStrategy{
		executeFn: func(ctx context.Context, action *model.RemediationAction) (*model.ExecutionResult, error) {
			return &model.ExecutionResult{
				Success:       false,
				StepsExecuted: 1,
				StepsTotal:    2,
				StepResults: []model.StepResult{
					{StepNumber: 1, Action: "step-one", Status: "success", DurationMs: 100},
					{StepNumber: 2, Action: "step-two", Status: "failure", DurationMs: 50, Error: "step two failed"},
				},
			}, nil
		},
		captureStateFn: func(ctx context.Context, action *model.RemediationAction) (json.RawMessage, error) {
			return json.RawMessage(`{"state":"captured"}`), nil
		},
	}
	rr := &fakeRemRepo{}
	audit := &fakeAudit{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, audit)

	action := dryRunCompletedAction()

	_, err := exec.Execute(context.Background(), action, uuid.New(), "executor")
	if err == nil {
		t.Fatal("expected error for unsuccessful result")
	}
	if got := err.Error(); got != "execution failed: step two failed" {
		t.Errorf("unexpected error message: %s", got)
	}

	// Status should go to execution_failed
	foundFailed := false
	for _, su := range rr.statuses {
		if su.Status == model.StatusExecutionFailed {
			foundFailed = true
			break
		}
	}
	if !foundFailed {
		t.Error("expected execution_failed status update")
	}

	// Step audit entries should have been recorded for each step result
	if len(audit.steps) != 2 {
		t.Fatalf("expected 2 step audit entries, got %d", len(audit.steps))
	}
	if audit.steps[1].StepResult != "failure" {
		t.Errorf("second step audit result should be failure, got %s", audit.steps[1].StepResult)
	}
	if audit.steps[1].ErrMsg != "step two failed" {
		t.Errorf("second step error should be 'step two failed', got %s", audit.steps[1].ErrMsg)
	}

	// Transition should include execution_failed
	foundTransition := false
	for _, tr := range audit.transitions {
		if tr.Action == "execution_failed" {
			foundTransition = true
			break
		}
	}
	if !foundTransition {
		t.Error("expected audit transition with action=execution_failed")
	}
}

func TestExecute_PreConditionFailure_NoApproval(t *testing.T) {
	strat := &fakeStrategy{}
	rr := &fakeRemRepo{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, &fakeAudit{})

	action := dryRunCompletedAction()
	action.ApprovedBy = nil
	action.ApprovedAt = nil

	_, err := exec.Execute(context.Background(), action, uuid.New(), "executor")
	if err == nil {
		t.Fatal("expected error for no approval")
	}
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed, got: %v", err)
	}

	// No status updates should have occurred
	if len(rr.statuses) != 0 {
		t.Errorf("expected 0 status updates, got %d", len(rr.statuses))
	}
}

func TestExecute_PreConditionFailure_NoDryRun(t *testing.T) {
	strat := &fakeStrategy{}
	rr := &fakeRemRepo{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, &fakeAudit{})

	action := dryRunCompletedAction()
	action.DryRunResult = nil
	action.DryRunAt = nil

	_, err := exec.Execute(context.Background(), action, uuid.New(), "executor")
	if err == nil {
		t.Fatal("expected error for no dry-run")
	}
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed, got: %v", err)
	}

	if len(rr.statuses) != 0 {
		t.Errorf("expected 0 status updates, got %d", len(rr.statuses))
	}
}

func TestExecute_WrongStatus(t *testing.T) {
	strat := &fakeStrategy{}
	rr := &fakeRemRepo{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, &fakeAudit{})

	action := testAction(model.StatusDraft)

	_, err := exec.Execute(context.Background(), action, uuid.New(), "executor")
	if err == nil {
		t.Fatal("expected error for wrong status")
	}
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed, got: %v", err)
	}

	if len(rr.statuses) != 0 {
		t.Errorf("expected 0 status updates, got %d", len(rr.statuses))
	}
}

func TestExecute_CaptureState_Failure_Reversible(t *testing.T) {
	strat := &fakeStrategy{
		captureStateFn: func(ctx context.Context, action *model.RemediationAction) (json.RawMessage, error) {
			return nil, errors.New("capture state failed")
		},
	}
	rr := &fakeRemRepo{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, &fakeAudit{})

	action := dryRunCompletedAction()
	action.Plan.Reversible = true

	_, err := exec.Execute(context.Background(), action, uuid.New(), "executor")
	if err == nil {
		t.Fatal("expected error when capture state fails for reversible plan")
	}
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed, got: %v", err)
	}

	// Should not have progressed to executing status
	for _, su := range rr.statuses {
		if su.Status == model.StatusExecuting {
			t.Error("should not have transitioned to executing when state capture failed for reversible plan")
		}
	}
}

func TestExecute_CaptureState_Failure_NonReversible(t *testing.T) {
	executeCalled := false
	strat := &fakeStrategy{
		captureStateFn: func(ctx context.Context, action *model.RemediationAction) (json.RawMessage, error) {
			return nil, errors.New("capture state failed")
		},
		executeFn: func(ctx context.Context, action *model.RemediationAction) (*model.ExecutionResult, error) {
			executeCalled = true
			return &model.ExecutionResult{
				Success:       true,
				StepsExecuted: 2,
				StepsTotal:    2,
			}, nil
		},
	}
	rr := &fakeRemRepo{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, &fakeAudit{})

	action := dryRunCompletedAction()
	action.Plan.Reversible = false

	result, err := exec.Execute(context.Background(), action, uuid.New(), "executor")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !executeCalled {
		t.Error("expected execution to proceed for non-reversible plan despite capture failure")
	}
	if !result.Success {
		t.Error("expected successful execution")
	}

	// Should have transitioned to executing with fallback state
	if len(rr.statuses) < 1 {
		t.Fatal("expected at least 1 status update")
	}
	if rr.statuses[0].Status != model.StatusExecuting {
		t.Errorf("first update should be executing, got %s", rr.statuses[0].Status)
	}

	// The pre_execution_state should contain the fallback capture_error
	preState, ok := rr.statuses[0].Fields["pre_execution_state"]
	if !ok {
		t.Fatal("expected pre_execution_state field in status update")
	}
	rawState, ok := preState.(json.RawMessage)
	if !ok {
		t.Fatalf("expected json.RawMessage for pre_execution_state, got %T", preState)
	}
	var stateMap map[string]interface{}
	if err := json.Unmarshal(rawState, &stateMap); err != nil {
		t.Fatalf("failed to unmarshal pre_execution_state: %v", err)
	}
	if stateMap["capture_error"] != "state capture failed" {
		t.Errorf("expected capture_error='state capture failed' in fallback state, got: %v", stateMap)
	}
}

// ---------------------------------------------------------------------------
// Verify Tests
// ---------------------------------------------------------------------------

func TestVerify_HappyPath_Verified(t *testing.T) {
	strat := &fakeStrategy{
		verifyFn: func(ctx context.Context, action *model.RemediationAction) (*model.VerificationResult, error) {
			return &model.VerificationResult{
				Verified: true,
				Checks: []model.VerificationCheck{
					{Name: "port-closed", Passed: true, Expected: "closed", Actual: "closed"},
				},
			}, nil
		},
	}
	rr := &fakeRemRepo{}
	audit := &fakeAudit{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, audit)

	action := executedAction()
	actorID := uuid.New()

	result, err := exec.Verify(context.Background(), action, &actorID, "verifier")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Verified {
		t.Fatal("expected Verified=true")
	}
	if len(result.Checks) != 1 || result.Checks[0].Name != "port-closed" {
		t.Fatalf("unexpected checks: %v", result.Checks)
	}

	// Should have: verification_pending, then verified
	if len(rr.statuses) != 2 {
		t.Fatalf("expected 2 status updates, got %d", len(rr.statuses))
	}
	if rr.statuses[0].Status != model.StatusVerificationPending {
		t.Errorf("first update should be verification_pending, got %s", rr.statuses[0].Status)
	}
	if rr.statuses[1].Status != model.StatusVerified {
		t.Errorf("second update should be verified, got %s", rr.statuses[1].Status)
	}

	// Verify audit transitions
	if len(audit.transitions) < 2 {
		t.Fatalf("expected at least 2 transitions, got %d", len(audit.transitions))
	}
	if audit.transitions[0].Action != "verification_started" {
		t.Errorf("first transition should be verification_started, got %s", audit.transitions[0].Action)
	}
	if audit.transitions[0].OldStatus != model.StatusExecuted {
		t.Errorf("first transition old status should be executed, got %s", audit.transitions[0].OldStatus)
	}
	if audit.transitions[0].NewStatus != model.StatusVerificationPending {
		t.Errorf("first transition new status should be verification_pending, got %s", audit.transitions[0].NewStatus)
	}
	if audit.transitions[1].Action != "verify_success" {
		t.Errorf("second transition should be verify_success, got %s", audit.transitions[1].Action)
	}
	if audit.transitions[1].NewStatus != model.StatusVerified {
		t.Errorf("second transition new status should be verified, got %s", audit.transitions[1].NewStatus)
	}
}

func TestVerify_HappyPath_Failed(t *testing.T) {
	strat := &fakeStrategy{
		verifyFn: func(ctx context.Context, action *model.RemediationAction) (*model.VerificationResult, error) {
			return &model.VerificationResult{
				Verified:      false,
				FailureReason: "port still open",
				Checks: []model.VerificationCheck{
					{Name: "port-closed", Passed: false, Expected: "closed", Actual: "open"},
				},
			}, nil
		},
	}
	rr := &fakeRemRepo{}
	audit := &fakeAudit{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, audit)

	action := executedAction()

	result, err := exec.Verify(context.Background(), action, nil, "system")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Verified {
		t.Fatal("expected Verified=false")
	}
	if result.FailureReason != "port still open" {
		t.Errorf("expected failure reason 'port still open', got %s", result.FailureReason)
	}

	// Should have: verification_pending, then verification_failed
	if len(rr.statuses) != 2 {
		t.Fatalf("expected 2 status updates, got %d", len(rr.statuses))
	}
	if rr.statuses[0].Status != model.StatusVerificationPending {
		t.Errorf("first update should be verification_pending, got %s", rr.statuses[0].Status)
	}
	if rr.statuses[1].Status != model.StatusVerificationFailed {
		t.Errorf("second update should be verification_failed, got %s", rr.statuses[1].Status)
	}

	// Audit transition should be verify_failure
	foundVerifyFailure := false
	for _, tr := range audit.transitions {
		if tr.Action == "verify_failure" && tr.NewStatus == model.StatusVerificationFailed {
			foundVerifyFailure = true
			break
		}
	}
	if !foundVerifyFailure {
		t.Error("expected audit transition with action=verify_failure")
	}
}

func TestVerify_ResolvesLinkedVulnerability(t *testing.T) {
	strat := &fakeStrategy{
		verifyFn: func(ctx context.Context, action *model.RemediationAction) (*model.VerificationResult, error) {
			return &model.VerificationResult{Verified: true}, nil
		},
	}
	rr := &fakeRemRepo{}
	vr := &fakeVulnRepo{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, vr, &fakeAudit{})

	action := executedAction()
	vulnID := uuid.New()
	action.VulnerabilityID = &vulnID

	result, err := exec.Verify(context.Background(), action, nil, "system")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Verified {
		t.Fatal("expected Verified=true")
	}

	// Vulnerability should have been resolved
	if len(vr.statusUpdates) != 1 {
		t.Fatalf("expected 1 vulnerability status update, got %d", len(vr.statusUpdates))
	}
	if vr.statusUpdates[0].VulnID != vulnID {
		t.Errorf("expected vuln ID %s, got %s", vulnID, vr.statusUpdates[0].VulnID)
	}
	if vr.statusUpdates[0].TenantID != action.TenantID {
		t.Errorf("expected tenant ID %s, got %s", action.TenantID, vr.statusUpdates[0].TenantID)
	}
	if vr.statusUpdates[0].Status != "resolved" {
		t.Errorf("expected status=resolved, got %s", vr.statusUpdates[0].Status)
	}
	if vr.statusUpdates[0].Notes != nil {
		t.Errorf("expected nil notes, got %v", vr.statusUpdates[0].Notes)
	}
}

func TestVerify_ResolvesLinkedAlert(t *testing.T) {
	var updatedAlertID uuid.UUID
	var updatedStatus model.AlertStatus
	var updatedNotes *string
	ar := &fakeAlertRepo{
		updateStatusFn: func(ctx context.Context, tenantID, alertID uuid.UUID, status model.AlertStatus, notes, reason *string) (*model.Alert, error) {
			updatedAlertID = alertID
			updatedStatus = status
			updatedNotes = notes
			return &model.Alert{ID: alertID, TenantID: tenantID, Status: status}, nil
		},
	}
	strat := &fakeStrategy{
		verifyFn: func(ctx context.Context, action *model.RemediationAction) (*model.VerificationResult, error) {
			return &model.VerificationResult{Verified: true}, nil
		},
	}
	rr := &fakeRemRepo{}
	exec := newTestExecutor(strat, rr, ar, &fakeVulnRepo{}, &fakeAudit{})

	action := executedAction()
	alertID := uuid.New()
	action.AlertID = &alertID

	result, err := exec.Verify(context.Background(), action, nil, "system")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Verified {
		t.Fatal("expected Verified=true")
	}

	if updatedAlertID != alertID {
		t.Errorf("expected alert ID %s to be updated, got %s", alertID, updatedAlertID)
	}
	if updatedStatus != model.AlertStatusResolved {
		t.Errorf("expected alert status=resolved, got %s", updatedStatus)
	}
	if updatedNotes == nil || *updatedNotes != "Resolved by verified remediation" {
		t.Errorf("expected resolution notes, got %v", updatedNotes)
	}
}

func TestVerify_WrongStatus(t *testing.T) {
	strat := &fakeStrategy{}
	rr := &fakeRemRepo{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, &fakeAudit{})

	action := testAction(model.StatusDraft)

	_, err := exec.Verify(context.Background(), action, nil, "system")
	if err == nil {
		t.Fatal("expected error for wrong status")
	}
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed, got: %v", err)
	}

	if len(rr.statuses) != 0 {
		t.Errorf("expected 0 status updates, got %d", len(rr.statuses))
	}
}

// ---------------------------------------------------------------------------
// Rollback Tests
// ---------------------------------------------------------------------------

func TestRollback_HappyPath(t *testing.T) {
	strat := &fakeStrategy{
		rollbackFn: func(ctx context.Context, action *model.RemediationAction) error {
			return nil
		},
	}
	rr := &fakeRemRepo{}
	audit := &fakeAudit{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, audit)

	action := executedAction()
	approvedBy := uuid.New()

	err := exec.Rollback(context.Background(), action, "security concern", approvedBy, "manager")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have: rolling_back, then rolled_back
	if len(rr.statuses) != 2 {
		t.Fatalf("expected 2 status updates, got %d", len(rr.statuses))
	}
	if rr.statuses[0].Status != model.StatusRollingBack {
		t.Errorf("first update should be rolling_back, got %s", rr.statuses[0].Status)
	}
	if rr.statuses[1].Status != model.StatusRolledBack {
		t.Errorf("second update should be rolled_back, got %s", rr.statuses[1].Status)
	}

	// Verify the rollback_reason and rollback_approved_by in the first update
	if reason, ok := rr.statuses[0].Fields["rollback_reason"]; !ok || reason != "security concern" {
		t.Errorf("expected rollback_reason='security concern', got %v", rr.statuses[0].Fields["rollback_reason"])
	}
	if ab, ok := rr.statuses[0].Fields["rollback_approved_by"]; !ok || ab != approvedBy {
		t.Errorf("expected rollback_approved_by=%s, got %v", approvedBy, rr.statuses[0].Fields["rollback_approved_by"])
	}

	// Verify the rolled_back update has rollback_result
	if _, ok := rr.statuses[1].Fields["rollback_result"]; !ok {
		t.Error("expected rollback_result in rolled_back status update")
	}
	if _, ok := rr.statuses[1].Fields["rolled_back_at"]; !ok {
		t.Error("expected rolled_back_at in rolled_back status update")
	}

	// Verify audit transitions: rollback_started, then rollback_done
	if len(audit.transitions) < 2 {
		t.Fatalf("expected at least 2 transitions, got %d", len(audit.transitions))
	}
	if audit.transitions[0].Action != "rollback_started" {
		t.Errorf("first transition should be rollback_started, got %s", audit.transitions[0].Action)
	}
	if audit.transitions[0].OldStatus != model.StatusExecuted {
		t.Errorf("first transition old status should be executed, got %s", audit.transitions[0].OldStatus)
	}
	if audit.transitions[0].NewStatus != model.StatusRollingBack {
		t.Errorf("first transition new status should be rolling_back, got %s", audit.transitions[0].NewStatus)
	}
	if audit.transitions[1].Action != "rollback_done" {
		t.Errorf("second transition should be rollback_done, got %s", audit.transitions[1].Action)
	}
	if audit.transitions[1].NewStatus != model.StatusRolledBack {
		t.Errorf("second transition new status should be rolled_back, got %s", audit.transitions[1].NewStatus)
	}
}

func TestRollback_StrategyFailure(t *testing.T) {
	rollbackErr := errors.New("rollback boom")
	strat := &fakeStrategy{
		rollbackFn: func(ctx context.Context, action *model.RemediationAction) error {
			return rollbackErr
		},
	}
	rr := &fakeRemRepo{}
	ar := &fakeAlertRepo{}
	audit := &fakeAudit{}
	exec := newTestExecutor(strat, rr, ar, &fakeVulnRepo{}, audit)

	action := executedAction()

	err := exec.Rollback(context.Background(), action, "needs rollback", uuid.New(), "manager")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("expected rollback error, got: %v", err)
	}

	// Should have: rolling_back, then rollback_failed
	if len(rr.statuses) != 2 {
		t.Fatalf("expected 2 status updates, got %d", len(rr.statuses))
	}
	if rr.statuses[0].Status != model.StatusRollingBack {
		t.Errorf("first update should be rolling_back, got %s", rr.statuses[0].Status)
	}
	if rr.statuses[1].Status != model.StatusRollbackFailed {
		t.Errorf("second update should be rollback_failed, got %s", rr.statuses[1].Status)
	}

	// A critical alert should have been created for the failure
	if len(ar.createdAlerts) != 1 {
		t.Fatalf("expected 1 alert created, got %d", len(ar.createdAlerts))
	}
	if ar.createdAlerts[0].Severity != model.SeverityCritical {
		t.Errorf("expected critical severity alert, got %s", ar.createdAlerts[0].Severity)
	}
	if ar.createdAlerts[0].Source != "remediation" {
		t.Errorf("expected source=remediation, got %s", ar.createdAlerts[0].Source)
	}

	// Audit transition should have rollback_error
	foundError := false
	for _, tr := range audit.transitions {
		if tr.Action == "rollback_error" && tr.NewStatus == model.StatusRollbackFailed {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Error("expected audit transition with action=rollback_error")
	}
}

func TestRollback_PreConditionFailure_NotReversible(t *testing.T) {
	strat := &fakeStrategy{}
	rr := &fakeRemRepo{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, &fakeAudit{})

	action := executedAction()
	action.Plan.Reversible = false

	err := exec.Rollback(context.Background(), action, "need rollback", uuid.New(), "manager")
	if err == nil {
		t.Fatal("expected error for non-reversible plan")
	}
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed, got: %v", err)
	}

	if len(rr.statuses) != 0 {
		t.Errorf("expected 0 status updates, got %d", len(rr.statuses))
	}
}

func TestRollback_PreConditionFailure_NoPreState(t *testing.T) {
	strat := &fakeStrategy{}
	rr := &fakeRemRepo{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, &fakeAudit{})

	action := executedAction()
	action.PreExecutionState = nil

	err := exec.Rollback(context.Background(), action, "need rollback", uuid.New(), "manager")
	if err == nil {
		t.Fatal("expected error for nil pre-execution state")
	}
	if !errors.Is(err, ErrPreConditionFailed) {
		t.Fatalf("expected ErrPreConditionFailed, got: %v", err)
	}

	if len(rr.statuses) != 0 {
		t.Errorf("expected 0 status updates, got %d", len(rr.statuses))
	}
}

func TestRollback_WrongStatus(t *testing.T) {
	strat := &fakeStrategy{}
	rr := &fakeRemRepo{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, &fakeAudit{})

	action := executedAction()
	action.Status = model.StatusDraft

	err := exec.Rollback(context.Background(), action, "need rollback", uuid.New(), "manager")
	if err == nil {
		t.Fatal("expected error for wrong status")
	}
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}

	if len(rr.statuses) != 0 {
		t.Errorf("expected 0 status updates, got %d", len(rr.statuses))
	}
}

// ---------------------------------------------------------------------------
// Misc Tests
// ---------------------------------------------------------------------------

func TestGetStrategy_Missing(t *testing.T) {
	strat := &fakeStrategy{}
	rr := &fakeRemRepo{}
	exec := newTestExecutor(strat, rr, &fakeAlertRepo{}, &fakeVulnRepo{}, &fakeAudit{})

	action := approvedAction()
	action.Type = model.RemediationType("nonexistent_type")

	_, err := exec.DryRun(context.Background(), action, nil, "system")
	if err == nil {
		t.Fatal("expected error for missing strategy")
	}
	expected := fmt.Sprintf("no strategy registered for type '%s'", "nonexistent_type")
	if err.Error() != expected {
		t.Fatalf("expected error %q, got %q", expected, err.Error())
	}
}
