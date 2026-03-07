package detection

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

func TestCorrelationEvaluatorSequenceMatched(t *testing.T) {
	evaluator := &CorrelationEvaluator{}
	compiled, err := evaluator.Compile([]byte(`{
		"events":[
			{"name":"failed_login","condition":{"type":"login_failed"}},
			{"name":"success_login","condition":{"type":"login_success"}},
			{"name":"privilege_escalation","condition":{"type":"privilege_change"}}
		],
		"sequence":["failed_login","success_login","privilege_escalation"],
		"group_by":"username",
		"window":"30m",
		"min_failed_count":3
	}`))
	if err != nil {
		t.Fatalf("Compile returned error: %v", err)
	}

	user := "alice"
	base := time.Now().UTC()
	events := []model.SecurityEvent{
		{ID: uuid.New(), Timestamp: base, Username: &user, Type: "login_failed"},
		{ID: uuid.New(), Timestamp: base.Add(2 * time.Minute), Username: &user, Type: "login_failed"},
		{ID: uuid.New(), Timestamp: base.Add(4 * time.Minute), Username: &user, Type: "login_failed"},
		{ID: uuid.New(), Timestamp: base.Add(5 * time.Minute), Username: &user, Type: "login_success"},
		{ID: uuid.New(), Timestamp: base.Add(10 * time.Minute), Username: &user, Type: "privilege_change"},
	}
	matches := evaluator.Evaluate(compiled, events)
	if len(matches) != 1 {
		t.Fatalf("expected 1 correlation match, got %d", len(matches))
	}
}
