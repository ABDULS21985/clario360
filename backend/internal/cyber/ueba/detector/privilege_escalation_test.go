package detector

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

func TestPrivilegeEscalationDetector(t *testing.T) {
	det := &privilegeEscalationDetector{config: DefaultConfig()}

	dmlProfile := &model.UEBAProfile{ID: uuid.New(), EntityID: "user-1", ProfileMaturity: model.ProfileMaturityMature}
	dmlProfile.EnsureDefaults()
	dmlProfile.Baseline.AccessPatterns.QueryTypes["ddl"] = 0.01

	ddlEvent := &model.DataAccessEvent{ID: uuid.New(), EventTimestamp: time.Now().UTC(), Action: "create"}
	signal := det.Detect(context.Background(), ddlEvent, dmlProfile)
	if signal == nil || signal.SignalType != model.SignalTypePrivilegeEscalation {
		t.Fatalf("expected privilege escalation signal for DML-only user")
	}

	dbaProfile := &model.UEBAProfile{ID: uuid.New(), EntityID: "dba-1", ProfileMaturity: model.ProfileMaturityMature}
	dbaProfile.EnsureDefaults()
	dbaProfile.Baseline.AccessPatterns.QueryTypes["ddl"] = 0.30
	if signal := det.Detect(context.Background(), ddlEvent, dbaProfile); signal != nil {
		t.Fatalf("expected no signal for DBA-like profile")
	}
}
