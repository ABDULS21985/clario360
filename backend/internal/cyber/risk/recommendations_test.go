package risk

import "testing"

func TestEstimateReduction(t *testing.T) {
	t.Parallel()

	if got := estimateReduction(18, 5, 10); got != 9 {
		t.Fatalf("estimateReduction = %v, want 9", got)
	}
	if got := estimateReduction(18, 20, 10); got != 18 {
		t.Fatalf("estimateReduction should clamp to weighted score, got %v", got)
	}
}

func TestEffortFromCount(t *testing.T) {
	t.Parallel()

	if effort := effortFromCount(3); effort != "low" {
		t.Fatalf("expected low effort, got %q", effort)
	}
	if effort := effortFromCount(10); effort != "medium" {
		t.Fatalf("expected medium effort, got %q", effort)
	}
	if effort := effortFromCount(25); effort != "high" {
		t.Fatalf("expected high effort, got %q", effort)
	}
}

func TestStringSliceFromDetails(t *testing.T) {
	t.Parallel()

	values := stringSliceFromDetails(map[string]interface{}{
		"items": []interface{}{"T1059", "T1078"},
	}, "items")
	if len(values) != 2 || values[0] != "T1059" || values[1] != "T1078" {
		t.Fatalf("unexpected slice conversion: %#v", values)
	}
}

func TestIntFromDetails(t *testing.T) {
	t.Parallel()

	if got := intFromDetails(map[string]interface{}{"count": float64(7)}, "count"); got != 7 {
		t.Fatalf("expected 7, got %d", got)
	}
}
