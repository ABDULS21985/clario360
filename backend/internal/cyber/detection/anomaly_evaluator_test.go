package detection

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

func TestAnomalyEvaluatorAboveThreshold(t *testing.T) {
	store := NewBaselineStore(nil, zerolog.New(io.Discard))
	evaluator := NewAnomalyEvaluator(store)
	compiled, err := evaluator.Compile([]byte(`{
		"metric":"bytes_transferred",
		"group_by":"asset_id",
		"window":"1h",
		"z_score_threshold":3.0,
		"min_baseline_samples":5,
		"direction":"above"
	}`))
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}
	cfg := compiled.(*compiledAnomalyRule)
	cfg.RuleID = uuid.New()
	cfg.TenantID = uuid.New()
	group := uuid.New()
	for _, value := range []float64{50, 49, 51, 52, 48, 50, 49} {
		if _, err := store.UpdateBaseline(context.Background(), cfg.TenantID, cfg.RuleID, group.String(), value); err != nil {
			t.Fatalf("UpdateBaseline returned error: %v", err)
		}
	}
	events := []model.SecurityEvent{{
		ID:        uuid.New(),
		Timestamp: time.Now().UTC(),
		AssetID:   &group,
		RawEvent:  []byte(`{"bytes_transferred":85}`),
	}}
	matches := evaluator.Evaluate(compiled, events)
	if len(matches) == 0 {
		t.Fatal("expected anomaly match for large bytes-transferred deviation")
	}
}

func TestBaselineStoreUpdateBaseline(t *testing.T) {
	store := NewBaselineStore(nil, zerolog.New(io.Discard))
	tenantID := uuid.New()
	ruleID := uuid.New()
	group := "asset-1"
	for _, value := range []float64{10, 20, 30} {
		if _, err := store.UpdateBaseline(context.Background(), tenantID, ruleID, group, value); err != nil {
			t.Fatalf("UpdateBaseline returned error: %v", err)
		}
	}
	baseline, err := store.GetBaseline(context.Background(), tenantID, ruleID, group)
	if err != nil {
		t.Fatalf("GetBaseline returned error: %v", err)
	}
	if baseline.Count != 3 || baseline.Mean != 20 {
		t.Fatalf("unexpected baseline after Welford updates: %+v", baseline)
	}
}
