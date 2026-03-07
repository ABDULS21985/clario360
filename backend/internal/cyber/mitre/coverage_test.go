package mitre

import (
	"testing"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

func TestCoverageBuildAndTactics(t *testing.T) {
	if len(AllTactics()) != 14 {
		t.Fatalf("expected 14 tactics, got %d", len(AllTactics()))
	}
	rules := []*model.DetectionRule{
		{ID: uuid.New(), Name: "Exploit Public-Facing App", MITRETechniqueIDs: []string{"T1190"}},
	}
	coverage := BuildCoverage(rules)
	var found bool
	for _, item := range coverage {
		if item.Technique.ID == "T1190" {
			found = true
			if !item.HasDetection || item.RuleCount != 1 {
				t.Fatalf("expected coverage for T1190, got %+v", item)
			}
		}
	}
	if !found {
		t.Fatal("expected T1190 to exist in coverage data")
	}
}
