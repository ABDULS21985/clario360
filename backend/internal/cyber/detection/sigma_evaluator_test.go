package detection

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

func TestSigmaEvaluatorSimpleMatch(t *testing.T) {
	evaluator := &SigmaEvaluator{}
	compiled, err := evaluator.Compile([]byte(`{
		"detection": {
			"selection": {"source":"firewall","dest_port|in":[4444,5555],"protocol":"tcp"},
			"condition": "selection"
		}
	}`))
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	port := 4444
	proto := "TCP"
	event := model.SecurityEvent{
		ID:        uuid.New(),
		Timestamp: time.Now().UTC(),
		Source:    "firewall",
		DestPort:  &port,
		Protocol:  &proto,
	}
	matches := evaluator.Evaluate(compiled, []model.SecurityEvent{event})
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
}

func TestSigmaEvaluatorFilterAndTimeframe(t *testing.T) {
	evaluator := &SigmaEvaluator{}
	compiled, err := evaluator.Compile([]byte(`{
		"detection": {
			"selection": {"type":"file_rename","file_path|re":"(?i)\\.encrypted$"},
			"filter": {"source_ip|startswith":"10."},
			"condition": "selection and not filter"
		},
		"timeframe": "5m",
		"threshold": 3
	}`))
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	ip := "8.8.8.8"
	path := "/tmp/a.encrypted"
	events := make([]model.SecurityEvent, 0, 3)
	base := time.Now().UTC()
	for i := 0; i < 3; i++ {
		events = append(events, model.SecurityEvent{
			ID:        uuid.New(),
			Timestamp: base.Add(time.Duration(i) * time.Minute),
			Type:      "file_rename",
			SourceIP:  &ip,
			FilePath:  &path,
		})
	}
	matches := evaluator.Evaluate(compiled, events)
	if len(matches) != 1 {
		t.Fatalf("expected 1 timeframe match, got %d", len(matches))
	}
}
