package detection

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

func TestThresholdEvaluatorDistinctMetric(t *testing.T) {
	evaluator := &ThresholdEvaluator{}
	compiled, err := evaluator.Compile([]byte(`{
		"field":"source_ip",
		"condition":{"source":"firewall"},
		"threshold":3,
		"window":"1m",
		"metric":"distinct(dest_port)"
	}`))
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	sourceIP := "1.2.3.4"
	base := time.Now().UTC()
	makeEvent := func(port int) model.SecurityEvent {
		return model.SecurityEvent{
			ID:        uuid.New(),
			Timestamp: base,
			Source:    "firewall",
			SourceIP:  &sourceIP,
			DestPort:  &port,
		}
	}
	matches := evaluator.Evaluate(compiled, []model.SecurityEvent{makeEvent(80), makeEvent(443), makeEvent(8443)})
	if len(matches) != 1 {
		t.Fatalf("expected 1 threshold match, got %d", len(matches))
	}
}
