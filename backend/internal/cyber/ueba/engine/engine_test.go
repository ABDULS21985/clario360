package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmiddleware "github.com/clario360/platform/internal/aigovernance/middleware"
)

type fakeGovernedPredictor struct {
	result *aigovernance.PredictionResult
	err    error
	calls  int
}

func (f *fakeGovernedPredictor) Predict(_ context.Context, _ aigovernance.PredictParams) (*aigovernance.PredictionResult, error) {
	f.calls++
	return f.result, f.err
}

func TestRunGovernedCycleFallsBackWhenGovernanceUnavailable(t *testing.T) {
	t.Helper()

	tenantID := uuid.New()
	expected := &cycleResult{
		CollectedEvents: 11,
		ProfilesUpdated: 4,
		SignalsCreated:  2,
		AlertsCreated:   1,
		EntitiesScored:  3,
		Confidence:      0.84,
	}
	predictor := &fakeGovernedPredictor{err: aigovmiddleware.ErrGovernanceUnavailable}
	runCalls := 0

	engine := &UEBAEngine{
		predLogger: predictor,
		runCycle: func(_ context.Context, gotTenantID uuid.UUID, _ UEBAConfig) (*cycleResult, error) {
			runCalls++
			if gotTenantID != tenantID {
				t.Fatalf("runCycle tenantID = %s, want %s", gotTenantID, tenantID)
			}
			return expected, nil
		},
		logger: zerolog.Nop(),
	}

	result, err := engine.runGovernedCycle(context.Background(), tenantID, UEBAConfig{})
	if err != nil {
		t.Fatalf("runGovernedCycle() error = %v", err)
	}
	if result != expected {
		t.Fatalf("runGovernedCycle() result = %#v, want %#v", result, expected)
	}
	if predictor.calls != 1 {
		t.Fatalf("Predict() calls = %d, want 1", predictor.calls)
	}
	if runCalls != 1 {
		t.Fatalf("runCycle calls = %d, want 1", runCalls)
	}
}

func TestRunGovernedCyclePropagatesNonGovernanceErrors(t *testing.T) {
	tenantID := uuid.New()
	expectedErr := errors.New("explanation failed")
	predictor := &fakeGovernedPredictor{err: expectedErr}
	runCalls := 0

	engine := &UEBAEngine{
		predLogger: predictor,
		runCycle: func(context.Context, uuid.UUID, UEBAConfig) (*cycleResult, error) {
			runCalls++
			return &cycleResult{}, nil
		},
		logger: zerolog.Nop(),
	}

	result, err := engine.runGovernedCycle(context.Background(), tenantID, UEBAConfig{})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("runGovernedCycle() error = %v, want %v", err, expectedErr)
	}
	if result != nil {
		t.Fatalf("runGovernedCycle() result = %#v, want nil", result)
	}
	if predictor.calls != 1 {
		t.Fatalf("Predict() calls = %d, want 1", predictor.calls)
	}
	if runCalls != 0 {
		t.Fatalf("runCycle calls = %d, want 0", runCalls)
	}
}

func TestRunGovernedCycleMapsPredictionOutput(t *testing.T) {
	tenantID := uuid.New()
	predictor := &fakeGovernedPredictor{
		result: &aigovernance.PredictionResult{
			Output: map[string]any{
				"events_processed":  9,
				"profiles_updated":  5,
				"signals_generated": 3,
				"alerts_created":    2,
				"entities_scored":   7,
			},
			Confidence: 0.91,
		},
	}
	engine := &UEBAEngine{
		predLogger: predictor,
		runCycle: func(context.Context, uuid.UUID, UEBAConfig) (*cycleResult, error) {
			t.Fatal("runCycle should not be called when prediction logging succeeds")
			return nil, nil
		},
		logger: zerolog.Nop(),
	}

	result, err := engine.runGovernedCycle(context.Background(), tenantID, UEBAConfig{})
	if err != nil {
		t.Fatalf("runGovernedCycle() error = %v", err)
	}
	if result == nil {
		t.Fatal("runGovernedCycle() returned nil result")
	}
	if result.CollectedEvents != 9 || result.ProfilesUpdated != 5 || result.SignalsCreated != 3 || result.AlertsCreated != 2 || result.EntitiesScored != 7 {
		t.Fatalf("runGovernedCycle() result = %#v, want mapped prediction output", result)
	}
	if result.Confidence != 0.91 {
		t.Fatalf("runGovernedCycle() confidence = %.2f, want 0.91", result.Confidence)
	}
}
