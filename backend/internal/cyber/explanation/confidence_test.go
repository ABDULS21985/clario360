package explanation

import (
	"testing"
	"time"

	"github.com/clario360/platform/internal/cyber/model"
)

func TestComputeConfidenceFactorsAndClamp(t *testing.T) {
	rule := &model.DetectionRule{
		BaseConfidence:     0.95,
		FalsePositiveCount: 60,
		TruePositiveCount:  40,
	}
	asset := &model.Asset{Criticality: model.CriticalityCritical}
	match := model.RuleMatch{
		Events: make([]model.SecurityEvent, 11),
		MatchDetails: map[string]interface{}{
			"matched_condition_count": 5,
			"indicator_age_hours":     2.0,
			"correlated_recent":       true,
			"maintenance_window":      true,
			"service_account":         "svc-monitor",
		},
		Timestamp: time.Now().UTC(),
	}
	score, factors := ComputeConfidence(rule, match, asset)
	if score > 0.99 {
		t.Fatalf("expected score clamp at 0.99, got %.2f", score)
	}
	if len(factors) < 4 {
		t.Fatalf("expected multiple confidence factors, got %d", len(factors))
	}
}
