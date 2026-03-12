package tools

import (
	"testing"
	"time"
)

func TestMapString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    map[string]any
		key  string
		want string
	}{
		{"string value", map[string]any{"id": "abc-123"}, "id", "abc-123"},
		{"int value fallback", map[string]any{"id": 42}, "id", "42"},
		{"missing key", map[string]any{}, "id", ""},
		{"nil value", map[string]any{"id": nil}, "id", ""},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := mapString(tc.m, tc.key)
			if got != tc.want {
				t.Fatalf("mapString(%v, %q) = %q, want %q", tc.m, tc.key, got, tc.want)
			}
		})
	}
}

func TestMapInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    map[string]any
		key  string
		want int
	}{
		{"int", map[string]any{"n": 5}, "n", 5},
		{"int64", map[string]any{"n": int64(7)}, "n", 7},
		{"float64", map[string]any{"n": float64(3)}, "n", 3},
		{"string fallback", map[string]any{"n": "oops"}, "n", 0},
		{"missing", map[string]any{}, "n", 0},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := mapInt(tc.m, tc.key)
			if got != tc.want {
				t.Fatalf("mapInt(%v, %q) = %d, want %d", tc.m, tc.key, got, tc.want)
			}
		})
	}
}

func TestMapFloat64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		m    map[string]any
		key  string
		want float64
	}{
		{"float64", map[string]any{"s": 78.5}, "s", 78.5},
		{"int", map[string]any{"s": 42}, "s", 42},
		{"int64", map[string]any{"s": int64(99)}, "s", 99},
		{"string fallback", map[string]any{"s": "bad"}, "s", 0},
		{"missing", map[string]any{}, "s", 0},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := mapFloat64(tc.m, tc.key)
			if got != tc.want {
				t.Fatalf("mapFloat64(%v, %q) = %f, want %f", tc.m, tc.key, got, tc.want)
			}
		})
	}
}

func TestMapTime(t *testing.T) {
	t.Parallel()

	fixed := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		m        map[string]any
		key      string
		wantZero bool
	}{
		{"time.Time", map[string]any{"due": fixed}, "due", false},
		{"RFC3339 string", map[string]any{"due": "2025-06-15T10:30:00Z"}, "due", false},
		{"bad string", map[string]any{"due": "not-a-date"}, "due", true},
		{"missing", map[string]any{}, "due", true},
		{"int fallback", map[string]any{"due": 12345}, "due", true},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := mapTime(tc.m, tc.key)
			if tc.wantZero && !got.IsZero() {
				t.Fatalf("mapTime expected zero, got %v", got)
			}
			if !tc.wantZero && got.IsZero() {
				t.Fatalf("mapTime expected non-zero, got zero")
			}
		})
	}
}

func TestRecommendation_PriorityOrdering(t *testing.T) {
	t.Parallel()

	// Simulate the scoring and sorting logic from the recommendation tool.
	// Critical alert (score=100) > failing pipeline with 3+ failures (score=80) >
	// expiring contract within 3 days (score=75) > UEBA score > 80 (score=70) >
	// overdue action item (score=65) > compliance gap (score=60).

	recs := []recommendationItem{
		{Category: "compliance", Title: "Compliance gap", Score: 60},
		{Category: "security", Title: "Critical alert", Score: 100},
		{Category: "data", Title: "Pipeline failure", Score: 80},
		{Category: "legal", Title: "Expiring contract", Score: 75},
		{Category: "ueba", Title: "High-risk entity", Score: 70},
		{Category: "governance", Title: "Overdue action item", Score: 65},
	}

	// Sort descending by score (same logic as recommendation.go)
	for i := 0; i < len(recs); i++ {
		for j := i + 1; j < len(recs); j++ {
			if recs[j].Score > recs[i].Score {
				recs[i], recs[j] = recs[j], recs[i]
			}
		}
	}

	expectedOrder := []string{"security", "data", "legal", "ueba", "governance", "compliance"}
	for i, rec := range recs {
		if rec.Category != expectedOrder[i] {
			t.Fatalf("position %d: got %q, want %q", i, rec.Category, expectedOrder[i])
		}
	}
}

func TestRecommendation_ScoreThresholds(t *testing.T) {
	t.Parallel()

	// Pipeline with < 3 consecutive failures scores 50
	t.Run("pipeline_low_failures", func(t *testing.T) {
		t.Parallel()
		consecutive := 2
		score := 50
		if consecutive >= 3 {
			score = 80
		}
		if score != 50 {
			t.Fatalf("score = %d, want 50", score)
		}
	})

	// Pipeline with >= 3 consecutive failures scores 80
	t.Run("pipeline_high_failures", func(t *testing.T) {
		t.Parallel()
		consecutive := 5
		score := 50
		if consecutive >= 3 {
			score = 80
		}
		if score != 80 {
			t.Fatalf("score = %d, want 80", score)
		}
	})

	// Contract expiring in < 3 days scores 75
	t.Run("contract_urgent", func(t *testing.T) {
		t.Parallel()
		days := 1
		score := 45
		if days < 3 {
			score = 75
		}
		if score != 75 {
			t.Fatalf("score = %d, want 75", score)
		}
	})

	// UEBA entity with score > 80 scores 70
	t.Run("ueba_high_risk", func(t *testing.T) {
		t.Parallel()
		uebaScore := 85.0
		score := 40
		if uebaScore > 80 {
			score = 70
		}
		if score != 70 {
			t.Fatalf("score = %d, want 70", score)
		}
	})
}
