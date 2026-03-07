package contradiction

import (
	"testing"
	"time"

	cruntime "github.com/clario360/platform/internal/data/contradiction/runtime"
	"github.com/clario360/platform/internal/data/model"
)

func TestComputeConfidenceLogicalHigh(t *testing.T) {
	now := time.Now().UTC()
	rowCount := int64(1000)
	raw := cruntime.RawContradiction{
		Type:            model.ContradictionTypeLogical,
		Column:          "email",
		AffectedRecords: 200,
	}
	modelA := &model.DataModel{PIIColumns: []string{"email"}}
	modelB := &model.DataModel{}
	sourceA := &model.DataSource{Name: "A", Status: model.DataSourceStatusActive, LastSyncedAt: &now, TotalRowCount: &rowCount}
	sourceB := &model.DataSource{Name: "B", Status: model.DataSourceStatusActive, LastSyncedAt: &now, TotalRowCount: &rowCount}

	got := ComputeConfidence(raw, modelA, modelB, sourceA, sourceB)
	if got <= 0.9 {
		t.Fatalf("ComputeConfidence() = %v, want > 0.9", got)
	}
}

func TestComputeConfidenceTimingReduction(t *testing.T) {
	now := time.Now().UTC()
	old := now.Add(-48 * time.Hour)
	raw := cruntime.RawContradiction{Type: model.ContradictionTypeLogical, Column: "status", AffectedRecords: 10}
	modelA := &model.DataModel{}
	modelB := &model.DataModel{}
	sourceA := &model.DataSource{Name: "A", Status: model.DataSourceStatusActive, LastSyncedAt: &now}
	sourceB := &model.DataSource{Name: "B", Status: model.DataSourceStatusActive, LastSyncedAt: &old}

	got := ComputeConfidence(raw, modelA, modelB, sourceA, sourceB)
	if got >= 0.85 {
		t.Fatalf("ComputeConfidence() = %v, want reduced score", got)
	}
}

func TestComputeConfidenceNumericRoundingReduction(t *testing.T) {
	raw := cruntime.RawContradiction{
		Type:            model.ContradictionTypeAnalytical,
		Column:          "total_amount",
		AffectedRecords: 50,
		NumericDeltaPct: 0.5,
	}
	got := ComputeConfidence(raw, &model.DataModel{}, &model.DataModel{}, &model.DataSource{}, &model.DataSource{})
	if got >= 0.75 {
		t.Fatalf("ComputeConfidence() = %v, want reduced score", got)
	}
}

func TestComputeConfidenceClamped(t *testing.T) {
	now := time.Now().UTC()
	raw := cruntime.RawContradiction{
		Type:            model.ContradictionTypeLogical,
		Column:          "email",
		AffectedRecords: 10000,
	}
	modelA := &model.DataModel{PIIColumns: []string{"email"}}
	modelB := &model.DataModel{}
	sourceA := &model.DataSource{Status: model.DataSourceStatusActive, LastSyncedAt: &now}
	sourceB := &model.DataSource{Status: model.DataSourceStatusActive, LastSyncedAt: &now}

	got := ComputeConfidence(raw, modelA, modelB, sourceA, sourceB)
	if got != 0.99 {
		t.Fatalf("ComputeConfidence() = %v, want 0.99", got)
	}
}
