package aggregation

import (
	"testing"
	"time"
)

func TestClassifyTrend(t *testing.T) {
	tests := []struct {
		current  int64
		previous int64
		wantDir  string
	}{
		{100, 50, "increasing"},
		{50, 100, "decreasing"},
		{100, 95, "stable"},
		{0, 0, "stable"},
		{10, 0, "increasing"},
		{0, 10, "decreasing"},
	}
	for _, tt := range tests {
		dir, _ := classifyTrend(tt.current, tt.previous)
		if dir != tt.wantDir {
			t.Errorf("classifyTrend(%d, %d) = %q, want %q", tt.current, tt.previous, dir, tt.wantDir)
		}
	}
}

func TestComputeRiskScore(t *testing.T) {
	// All maxed out → 100
	score := computeRiskScore(200, 20, 10, 10)
	if score != 100 {
		t.Errorf("max risk score: want 100, got %f", score)
	}

	// All zero → 0
	score = computeRiskScore(0, 0, 0, 0)
	if score != 0 {
		t.Errorf("zero risk score: want 0, got %f", score)
	}

	// Moderate values
	score = computeRiskScore(50, 5, 2, 2)
	if score < 30 || score > 70 {
		t.Errorf("moderate risk score out of expected range: %f", score)
	}
}

func TestDefaultPeriods(t *testing.T) {
	if len(DefaultPeriods) != 4 {
		t.Fatalf("expected 4 periods, got %d", len(DefaultPeriods))
	}
	expected := []string{"24h", "7d", "30d", "90d"}
	for i, p := range DefaultPeriods {
		if p.Label != expected[i] {
			t.Errorf("period %d: want %q, got %q", i, expected[i], p.Label)
		}
		if p.Duration <= 0 {
			t.Errorf("period %q has non-positive duration", p.Label)
		}
	}
}

func TestDefaultScheduleConfig(t *testing.T) {
	c := DefaultScheduleConfig
	if c.FullInterval != 5*time.Minute {
		t.Errorf("FullInterval: want 5m, got %v", c.FullInterval)
	}
	if c.ExecutiveInterval != 2*time.Minute {
		t.Errorf("ExecutiveInterval: want 2m, got %v", c.ExecutiveInterval)
	}
	if c.CleanupInterval != 1*time.Hour {
		t.Errorf("CleanupInterval: want 1h, got %v", c.CleanupInterval)
	}
	if c.MaxAggregationAge != 7*24*time.Hour {
		t.Errorf("MaxAggregationAge: want 7d, got %v", c.MaxAggregationAge)
	}
}
