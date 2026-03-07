package aggregator

import "testing"

func TestJSONPath_SimpleKey(t *testing.T) {
	value, err := ExtractValue(map[string]any{"score": 42}, "$.score")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != 42 {
		t.Fatalf("expected 42, got %v", value)
	}
}

func TestJSONPath_Nested(t *testing.T) {
	value, err := ExtractValue(map[string]any{"data": map[string]any{"risk_score": 88.5}}, "$.data.risk_score")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != 88.5 {
		t.Fatalf("expected 88.5, got %v", value)
	}
}

func TestJSONPath_DeepNested(t *testing.T) {
	value, err := ExtractValue(map[string]any{"data": map[string]any{"kpis": map[string]any{"mttr_hours": 3.5}}}, "$.data.kpis.mttr_hours")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != 3.5 {
		t.Fatalf("expected 3.5, got %v", value)
	}
}

func TestJSONPath_IntToFloat(t *testing.T) {
	value, err := ExtractValue(map[string]any{"count": 7}, "$.count")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != 7 {
		t.Fatalf("expected 7, got %v", value)
	}
}

func TestJSONPath_NotFound(t *testing.T) {
	if _, err := ExtractValue(map[string]any{"score": 1}, "$.missing"); err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestJSONPath_NotNumeric(t *testing.T) {
	if _, err := ExtractValue(map[string]any{"score": "bad"}, "$.score"); err == nil {
		t.Fatal("expected error for non-numeric value")
	}
}
