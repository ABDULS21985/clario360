package strategy

import (
	"context"
	"encoding/json"
	"time"

	"github.com/clario360/platform/internal/cyber/model"
)

// CustomStrategy handles manual/custom remediations (tracking only).
type CustomStrategy struct{}

// NewCustomStrategy creates a CustomStrategy.
func NewCustomStrategy() *CustomStrategy { return &CustomStrategy{} }

func (s *CustomStrategy) Type() model.RemediationType { return model.RemediationTypeCustom }

func (s *CustomStrategy) DryRun(_ context.Context, action *model.RemediationAction) (*model.DryRunResult, error) {
	return &model.DryRunResult{
		Success:          true,
		SimulatedChanges: []model.SimulatedChange{},
		Warnings:         []string{"Manual execution required — no automated dry-run for custom remediations"},
		Blockers:         []string{},
		AffectedServices: []string{},
		EstimatedImpact: model.ImpactEstimate{
			Downtime:        action.Plan.EstimatedDowntime,
			RiskLevel:       action.Plan.RiskLevel,
			RecommendWindow: "per operator judgement",
		},
		DurationMs: 0,
	}, nil
}

func (s *CustomStrategy) Execute(_ context.Context, action *model.RemediationAction) (*model.ExecutionResult, error) {
	steps := make([]model.StepResult, 0, len(action.Plan.Steps))
	for _, step := range action.Plan.Steps {
		steps = append(steps, model.StepResult{
			StepNumber: step.Number,
			Action:     step.Action,
			Status:     "success",
			Output:     "Manual step confirmed by operator",
		})
	}
	return &model.ExecutionResult{
		Success:        true,
		StepsExecuted:  len(action.Plan.Steps),
		StepsTotal:     len(action.Plan.Steps),
		StepResults:    steps,
		ChangesApplied: []model.AppliedChange{{ChangeType: "manual", Description: "Manual remediation confirmed by operator"}},
		DurationMs:     0,
	}, nil
}

func (s *CustomStrategy) Verify(_ context.Context, _ *model.RemediationAction) (*model.VerificationResult, error) {
	return &model.VerificationResult{
		Verified: true,
		Checks: []model.VerificationCheck{{
			Name:     "Manual verification",
			Passed:   true,
			Expected: "operator confirmation",
			Actual:   "operator confirmed",
		}},
		DurationMs: 0,
	}, nil
}

func (s *CustomStrategy) Rollback(_ context.Context, _ *model.RemediationAction) error {
	// Manual rollback — tracking only, the operator performs the actual rollback
	return nil
}

func (s *CustomStrategy) CaptureState(_ context.Context, _ *model.RemediationAction) (json.RawMessage, error) {
	return json.Marshal(map[string]interface{}{
		"note":         "manual remediation — no automated state capture",
		"captured_at":  time.Now().UTC(),
	})
}
