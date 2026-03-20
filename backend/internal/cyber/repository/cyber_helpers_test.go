package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

type stubScanner struct {
	values []any
	err    error
}

func (s stubScanner) Scan(dest ...any) error {
	if s.err != nil {
		return s.err
	}
	if len(dest) != len(s.values) {
		return fmt.Errorf("scan arg count mismatch: got %d want %d", len(dest), len(s.values))
	}
	for i := range dest {
		if err := assignValue(dest[i], s.values[i]); err != nil {
			return fmt.Errorf("assign value %d: %w", i, err)
		}
	}
	return nil
}

func assignValue(dest any, value any) error {
	target := reflect.ValueOf(dest)
	if target.Kind() != reflect.Pointer || target.IsNil() {
		return fmt.Errorf("destination must be a non-nil pointer")
	}

	elem := target.Elem()
	if value == nil {
		elem.Set(reflect.Zero(elem.Type()))
		return nil
	}

	source := reflect.ValueOf(value)
	if source.Type().AssignableTo(elem.Type()) {
		elem.Set(source)
		return nil
	}
	if source.Type().ConvertibleTo(elem.Type()) {
		elem.Set(source.Convert(elem.Type()))
		return nil
	}
	return fmt.Errorf("cannot assign %T to %s", value, elem.Type())
}

func TestMarshalJSONHandlesNilValueAndEncodeErrors(t *testing.T) {
	payload, err := marshalJSON(nil)
	if err != nil {
		t.Fatalf("marshalJSON(nil) returned error: %v", err)
	}
	if string(payload) != "{}" {
		t.Fatalf("marshalJSON(nil) = %s, want {}", payload)
	}

	type sample struct {
		Name string `json:"name"`
	}
	payload, err = marshalJSON(sample{Name: "asset"})
	if err != nil {
		t.Fatalf("marshalJSON(sample) returned error: %v", err)
	}

	var decoded map[string]string
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal encoded payload: %v", err)
	}
	if decoded["name"] != "asset" {
		t.Fatalf("encoded payload = %#v, want name=asset", decoded)
	}

	if _, err := marshalJSON(make(chan int)); err == nil {
		t.Fatal("marshalJSON(channel) expected error")
	}
}

func TestEnsureRawMessageUsesFallbackForEmptyPayload(t *testing.T) {
	if got := ensureRawMessage(nil, "{}"); string(got) != "{}" {
		t.Fatalf("ensureRawMessage(nil) = %s, want {}", got)
	}

	if got := ensureRawMessage(json.RawMessage(`{"ok":true}`), "{}"); string(got) != `{"ok":true}` {
		t.Fatalf("ensureRawMessage(non-empty) = %s, want original payload", got)
	}
}

func TestScanAlertDecodesExplanationAndDefaultsCollections(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	alertID := uuid.New()
	tenantID := uuid.New()
	explanation := []byte(`{"summary":"Suspicious login","reason":"Multiple detections"}`)

	alert, err := scanAlert(stubScanner{
		values: []any{
			alertID,
			tenantID,
			"Suspicious login",
			"Investigate impossible travel",
			model.SeverityHigh,
			model.AlertStatusInvestigating,
			"detection-engine",
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			explanation,
			0.91,
			nil,
			nil,
			nil,
			nil,
			3,
			now,
			now,
			nil,
			nil,
			nil,
			nil,
			nil,
			now,
			now,
			nil,
		},
	})
	if err != nil {
		t.Fatalf("scanAlert returned error: %v", err)
	}

	if alert.ID != alertID || alert.TenantID != tenantID {
		t.Fatalf("scanAlert returned wrong IDs: %+v", alert)
	}
	if alert.Explanation.Summary != "Suspicious login" {
		t.Fatalf("alert explanation summary = %q, want %q", alert.Explanation.Summary, "Suspicious login")
	}
	if string(alert.Metadata) != "{}" {
		t.Fatalf("alert metadata = %s, want {}", alert.Metadata)
	}
	if alert.Tags == nil || len(alert.Tags) != 0 {
		t.Fatalf("alert tags = %#v, want empty slice", alert.Tags)
	}
	if alert.AssetIDs == nil || len(alert.AssetIDs) != 0 {
		t.Fatalf("alert asset_ids = %#v, want empty slice", alert.AssetIDs)
	}
}

func TestScanAlertReturnsExplanationDecodeError(t *testing.T) {
	_, err := scanAlert(stubScanner{
		values: []any{
			uuid.New(),
			uuid.New(),
			"Bad explanation",
			"broken json",
			model.SeverityMedium,
			model.AlertStatusNew,
			"detection-engine",
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			[]byte("{"),
			0.4,
			nil,
			nil,
			nil,
			nil,
			1,
			time.Now(),
			time.Now(),
			nil,
			nil,
			nil,
			nil,
			nil,
			time.Now(),
			time.Now(),
			nil,
		},
	})
	if err == nil {
		t.Fatal("scanAlert expected decode error")
	}
	if !strings.Contains(err.Error(), "decode alert explanation") {
		t.Fatalf("scanAlert error = %v, want decode alert explanation", err)
	}
}

func TestScanRuleAndSecurityEventApplySafeDefaults(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	rule, err := scanRule(stubScanner{
		values: []any{
			uuid.New(),
			nil,
			"Impossible travel",
			"Detect impossible travel events",
			model.RuleTypeCorrelation,
			model.SeverityHigh,
			true,
			nil,
			nil,
			nil,
			0.7,
			2,
			10,
			nil,
			12,
			nil,
			false,
			nil,
			nil,
			now,
			now,
			nil,
		},
	})
	if err != nil {
		t.Fatalf("scanRule returned error: %v", err)
	}

	if string(rule.RuleContent) != "{}" {
		t.Fatalf("rule content = %s, want {}", rule.RuleContent)
	}
	if rule.Tags == nil || len(rule.Tags) != 0 {
		t.Fatalf("rule tags = %#v, want empty slice", rule.Tags)
	}
	if rule.MITRETacticIDs == nil || len(rule.MITRETacticIDs) != 0 {
		t.Fatalf("rule tactic IDs = %#v, want empty slice", rule.MITRETacticIDs)
	}
	if rule.MITRETechniqueIDs == nil || len(rule.MITRETechniqueIDs) != 0 {
		t.Fatalf("rule technique IDs = %#v, want empty slice", rule.MITRETechniqueIDs)
	}

	sourceIP := "10.0.0.5"
	event, err := scanSecurityEvent(stubScanner{
		values: []any{
			uuid.New(),
			uuid.New(),
			now,
			"iam",
			"login",
			model.SeverityMedium,
			&sourceIP,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			now,
		},
	})
	if err != nil {
		t.Fatalf("scanSecurityEvent returned error: %v", err)
	}

	if event.SourceIP == nil || *event.SourceIP != sourceIP {
		t.Fatalf("event source IP = %#v, want %q", event.SourceIP, sourceIP)
	}
	if string(event.RawEvent) != "{}" {
		t.Fatalf("event raw payload = %s, want {}", event.RawEvent)
	}
	if event.MatchedRules == nil || len(event.MatchedRules) != 0 {
		t.Fatalf("event matched rules = %#v, want empty slice", event.MatchedRules)
	}
}

func TestScanHelpersPropagateScannerErrors(t *testing.T) {
	expected := errors.New("boom")

	if _, err := scanAlertComment(stubScanner{err: expected}); !errors.Is(err, expected) {
		t.Fatalf("scanAlertComment error = %v, want %v", err, expected)
	}

	if _, err := scanAlertTimeline(stubScanner{err: expected}); !errors.Is(err, expected) {
		t.Fatalf("scanAlertTimeline error = %v, want %v", err, expected)
	}
}
