package detector

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

func TestNewTableAccessDetector(t *testing.T) {
	profile := &model.UEBAProfile{
		ID:              uuid.New(),
		EntityType:      model.EntityTypeUser,
		EntityID:        "user-1",
		ProfileMaturity: model.ProfileMaturityMature,
	}
	profile.EnsureDefaults()
	profile.Baseline.AccessPatterns.TablesAccessed = []model.FrequencyEntry{{Name: "public.orders"}}

	det := &newTableAccessDetector{}
	known := &model.DataAccessEvent{ID: uuid.New(), EventTimestamp: time.Now().UTC(), SchemaName: "public", TableName: "orders", TableSensitivity: "restricted"}
	if signal := det.Detect(context.Background(), known, profile); signal != nil {
		t.Fatalf("expected no signal for known table")
	}

	restricted := &model.DataAccessEvent{ID: uuid.New(), EventTimestamp: time.Now().UTC(), SchemaName: "finance", TableName: "payroll", TableSensitivity: "restricted"}
	signal := det.Detect(context.Background(), restricted, profile)
	if signal == nil || signal.Severity != "high" {
		t.Fatalf("expected high signal for restricted new table")
	}

	public := &model.DataAccessEvent{ID: uuid.New(), EventTimestamp: time.Now().UTC(), SchemaName: "public", TableName: "catalog", TableSensitivity: "public"}
	if signal := det.Detect(context.Background(), public, profile); signal != nil {
		t.Fatalf("expected no signal for public new table")
	}
}
