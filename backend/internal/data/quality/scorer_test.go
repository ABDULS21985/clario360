package quality

import (
	"testing"

	"github.com/clario360/platform/internal/data/model"
)

func TestSeverityWeight(t *testing.T) {
	if got := severityWeight(model.QualitySeverityCritical); got != 4 {
		t.Fatalf("severityWeight(critical) = %d, want 4", got)
	}
	if got := severityWeight(model.QualitySeverityHigh); got != 3 {
		t.Fatalf("severityWeight(high) = %d, want 3", got)
	}
	if got := severityWeight(model.QualitySeverityMedium); got != 2 {
		t.Fatalf("severityWeight(medium) = %d, want 2", got)
	}
	if got := severityWeight(model.QualitySeverityLow); got != 1 {
		t.Fatalf("severityWeight(low) = %d, want 1", got)
	}
}

func TestClassificationWeight(t *testing.T) {
	if got := classificationWeight(model.DataClassificationRestricted); got != 3 {
		t.Fatalf("classificationWeight(restricted) = %v, want 3", got)
	}
	if got := classificationWeight(model.DataClassificationConfidential); got != 2 {
		t.Fatalf("classificationWeight(confidential) = %v, want 2", got)
	}
	if got := classificationWeight(model.DataClassificationInternal); got != 1 {
		t.Fatalf("classificationWeight(internal) = %v, want 1", got)
	}
	if got := classificationWeight(model.DataClassificationPublic); got != 0.5 {
		t.Fatalf("classificationWeight(public) = %v, want 0.5", got)
	}
}

func TestGrade(t *testing.T) {
	cases := []struct {
		score float64
		want  string
	}{
		{85, "A"},
		{70, "B"},
		{55, "C"},
		{40, "D"},
		{39.9, "F"},
	}
	for _, tc := range cases {
		if got := grade(tc.score); got != tc.want {
			t.Fatalf("grade(%v) = %s, want %s", tc.score, got, tc.want)
		}
	}
}
