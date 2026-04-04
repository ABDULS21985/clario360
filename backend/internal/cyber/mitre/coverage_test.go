package mitre

import (
	"testing"
	"time"

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

func TestFrameworkMeta(t *testing.T) {
	meta := FrameworkMeta()
	if meta.Version != FrameworkVersion {
		t.Errorf("version: expected %q, got %q", FrameworkVersion, meta.Version)
	}
	if meta.UpdatedAt != FrameworkUpdatedAt {
		t.Errorf("updated_at: expected %q, got %q", FrameworkUpdatedAt, meta.UpdatedAt)
	}
	if meta.TacticCount != 14 {
		t.Errorf("tactic_count: expected 14, got %d", meta.TacticCount)
	}
	if meta.TechniqueCount < 50 {
		t.Errorf("technique_count: expected ≥50, got %d", meta.TechniqueCount)
	}

	// StaleDays should be positive (catalog date is in the past)
	parsed, _ := time.Parse("2006-01-02", FrameworkUpdatedAt)
	expectedDays := int(time.Since(parsed).Hours() / 24)
	if meta.StaleDays < expectedDays-1 || meta.StaleDays > expectedDays+1 {
		t.Errorf("stale_days: expected ~%d, got %d", expectedDays, meta.StaleDays)
	}
}
