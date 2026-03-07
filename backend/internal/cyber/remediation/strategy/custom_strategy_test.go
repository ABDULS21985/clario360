package strategy

import (
	"context"
	"testing"

	"github.com/clario360/platform/internal/cyber/model"
)

func TestCustomStrategy_Type(t *testing.T) {
	s := NewCustomStrategy()
	if s.Type() != model.RemediationTypeCustom {
		t.Errorf("expected RemediationTypeCustom, got %s", s.Type())
	}
}

func TestCustomStrategy_DryRun(t *testing.T) {
	s := NewCustomStrategy()
	action := &model.RemediationAction{
		Plan: model.RemediationPlan{
			EstimatedDowntime: "30m",
			RiskLevel:         "medium",
		},
	}
	result, err := s.DryRun(context.Background(), action)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected dry-run success=true for custom strategy")
	}
	if len(result.Warnings) == 0 {
		t.Error("expected at least one warning about manual execution")
	}
	if len(result.Blockers) != 0 {
		t.Errorf("expected no blockers, got %v", result.Blockers)
	}
	if result.EstimatedImpact.Downtime != "30m" {
		t.Errorf("expected downtime 30m, got %s", result.EstimatedImpact.Downtime)
	}
}

func TestCustomStrategy_Execute(t *testing.T) {
	s := NewCustomStrategy()
	action := &model.RemediationAction{
		Plan: model.RemediationPlan{
			Steps: []model.RemediationStep{
				{Number: 1, Action: "investigate_alert"},
				{Number: 2, Action: "confirm_scope"},
			},
		},
	}
	result, err := s.Execute(context.Background(), action)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("expected execute success=true")
	}
	if result.StepsExecuted != 2 {
		t.Errorf("expected 2 steps executed, got %d", result.StepsExecuted)
	}
	if result.StepsTotal != 2 {
		t.Errorf("expected 2 steps total, got %d", result.StepsTotal)
	}
	if len(result.StepResults) != 2 {
		t.Errorf("expected 2 step results, got %d", len(result.StepResults))
	}
	for i, sr := range result.StepResults {
		if sr.Status != "success" {
			t.Errorf("step %d: expected status success, got %s", i, sr.Status)
		}
	}
}

func TestCustomStrategy_Verify(t *testing.T) {
	s := NewCustomStrategy()
	result, err := s.Verify(context.Background(), &model.RemediationAction{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Verified {
		t.Error("expected Verified=true for custom strategy")
	}
	if len(result.Checks) == 0 {
		t.Error("expected at least one verification check")
	}
	if !result.Checks[0].Passed {
		t.Error("expected first check to pass")
	}
}

func TestCustomStrategy_Rollback(t *testing.T) {
	s := NewCustomStrategy()
	err := s.Rollback(context.Background(), &model.RemediationAction{})
	if err != nil {
		t.Errorf("expected no error on rollback, got %v", err)
	}
}

func TestCustomStrategy_CaptureState(t *testing.T) {
	s := NewCustomStrategy()
	raw, err := s.CaptureState(context.Background(), &model.RemediationAction{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(raw) == 0 {
		t.Error("expected non-empty state JSON")
	}
}
