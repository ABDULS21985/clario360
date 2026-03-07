package ctem

import "testing"

func TestVelocityBucketMappings(t *testing.T) {
	fast := 6.0
	slow := 91.0
	if got := velocityBucket(&fast); got != 10 {
		t.Fatalf("expected fast velocity bucket 10, got %.2f", got)
	}
	if got := velocityBucket(&slow); got != 100 {
		t.Fatalf("expected slow velocity bucket 100, got %.2f", got)
	}
	if got := velocityBucket(nil); got != 50 {
		t.Fatalf("expected nil velocity bucket 50, got %.2f", got)
	}
}

func TestGradeForScore(t *testing.T) {
	cases := []struct {
		score float64
		want  string
	}{
		{15, "A"},
		{35, "B"},
		{55, "C"},
		{72, "D"},
		{92, "F"},
	}
	for _, tc := range cases {
		if got := gradeForScore(tc.score); got != tc.want {
			t.Fatalf("score %.2f: expected grade %s, got %s", tc.score, tc.want, got)
		}
	}
}

func TestTrendForDelta(t *testing.T) {
	if got := trendForDelta(-5); got != "improving" {
		t.Fatalf("expected improving trend, got %s", got)
	}
	if got := trendForDelta(5); got != "worsening" {
		t.Fatalf("expected worsening trend, got %s", got)
	}
	if got := trendForDelta(1.5); got != "stable" {
		t.Fatalf("expected stable trend, got %s", got)
	}
}

func TestSeverityWeightAndHopWeight(t *testing.T) {
	if got := severityWeight("critical"); got != 10 {
		t.Fatalf("expected critical severity weight 10, got %.2f", got)
	}
	if got := hopWeight(0); got != 1.0 {
		t.Fatalf("expected first hop weight 1.0, got %.2f", got)
	}
	if got := hopWeight(4); got != 0.2 {
		t.Fatalf("expected fifth hop weight 0.2, got %.2f", got)
	}
}

func TestRound2Deterministic(t *testing.T) {
	first := round2(12.3456)
	second := round2(12.3456)
	if first != 12.35 || second != 12.35 {
		t.Fatalf("expected deterministic rounding to 12.35, got %.2f and %.2f", first, second)
	}
}
