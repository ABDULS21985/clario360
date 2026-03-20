package contradiction

import (
	"strings"
	"testing"
	"time"

	cruntime "github.com/clario360/platform/internal/data/contradiction/runtime"
	"github.com/clario360/platform/internal/data/model"
)

func TestGenerateGuidanceLogical(t *testing.T) {
	now := time.Now().UTC()
	guidance, authoritative := GenerateGuidance(
		cruntime.RawContradiction{Type: model.ContradictionTypeLogical, Column: "email"},
		&model.DataModel{},
		&model.DataModel{},
		&model.DataSource{Name: "CRM", LastSyncedAt: &now},
		&model.DataSource{Name: "Billing", LastSyncedAt: ptrTime(now.Add(-time.Hour))},
	)
	if !strings.Contains(guidance, "Determine which source is authoritative") {
		t.Fatalf("GenerateGuidance() guidance = %q", guidance)
	}
	if authoritative == nil || *authoritative != "CRM" {
		t.Fatalf("GenerateGuidance() authoritative = %v, want CRM", authoritative)
	}
}

func TestGenerateGuidanceSemantic(t *testing.T) {
	guidance, _ := GenerateGuidance(
		cruntime.RawContradiction{Type: model.ContradictionTypeSemantic},
		&model.DataModel{},
		&model.DataModel{},
		&model.DataSource{Name: "HR"},
		&model.DataSource{Name: "Payroll"},
	)
	if !strings.Contains(guidance, "Verify the source record") {
		t.Fatalf("GenerateGuidance() guidance = %q", guidance)
	}
}

func TestGenerateGuidanceAnalytical(t *testing.T) {
	guidance, _ := GenerateGuidance(
		cruntime.RawContradiction{Type: model.ContradictionTypeAnalytical, Column: "revenue"},
		&model.DataModel{},
		&model.DataModel{},
		&model.DataSource{Name: "Warehouse"},
		&model.DataSource{Name: "Finance"},
	)
	if !strings.Contains(guidance, "Check for missing or duplicate records") {
		t.Fatalf("GenerateGuidance() guidance = %q", guidance)
	}
}

func TestDetermineAuthoritativeSourcePrefersFresher(t *testing.T) {
	now := time.Now().UTC()
	old := now.Add(-24 * time.Hour)
	got := determineAuthoritativeSource(
		&model.DataSource{Name: "Fresh", LastSyncedAt: &now},
		&model.DataSource{Name: "Old", LastSyncedAt: &old},
	)
	if got != "Fresh" {
		t.Fatalf("determineAuthoritativeSource() = %q, want Fresh", got)
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
