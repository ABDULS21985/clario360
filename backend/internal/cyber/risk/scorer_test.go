package risk

import (
	"testing"

	"github.com/clario360/platform/internal/cyber/model"
)

func TestGradeForScoreBoundaries(t *testing.T) {
	t.Parallel()

	cases := []struct {
		score float64
		want  string
	}{
		{0, "A"},
		{20, "A"},
		{21, "B"},
		{40, "B"},
		{41, "C"},
		{60, "C"},
		{61, "D"},
		{80, "D"},
		{81, "F"},
		{100, "F"},
	}

	for _, tc := range cases {
		if got := gradeForScore(tc.score); got != tc.want {
			t.Fatalf("gradeForScore(%v) = %q, want %q", tc.score, got, tc.want)
		}
	}
}

func TestTrendForScore(t *testing.T) {
	t.Parallel()

	if trend, delta := trendForScore(35, 40); trend != "improving" || delta != -5 {
		t.Fatalf("expected improving/-5, got %s/%v", trend, delta)
	}
	if trend, delta := trendForScore(72, 68); trend != "worsening" || delta != 4 {
		t.Fatalf("expected worsening/4, got %s/%v", trend, delta)
	}
	if trend, delta := trendForScore(50, 49); trend != "stable" || delta != 1 {
		t.Fatalf("expected stable/1, got %s/%v", trend, delta)
	}
}

func TestComponentScoreFromResultWeighted(t *testing.T) {
	t.Parallel()

	result := &model.RiskComponentResult{
		Score:       80,
		Description: "test component",
		Details:     map[string]interface{}{"count": 3},
	}
	score := componentScoreFromResult(result, 0.30, 60)
	if score.Score != 80 {
		t.Fatalf("expected score 80, got %v", score.Score)
	}
	if score.Weighted != 24 {
		t.Fatalf("expected weighted 24, got %v", score.Weighted)
	}
	if score.Trend != "worsening" || score.TrendDelta != 20 {
		t.Fatalf("expected worsening/20, got %s/%v", score.Trend, score.TrendDelta)
	}
}
