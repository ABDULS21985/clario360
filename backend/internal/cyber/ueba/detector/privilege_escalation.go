package detector

import (
	"context"
	"fmt"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

type privilegeEscalationDetector struct {
	config Config
}

func (d *privilegeEscalationDetector) Name() model.SignalType {
	return model.SignalTypePrivilegeEscalation
}

func (d *privilegeEscalationDetector) Detect(_ context.Context, event *model.DataAccessEvent, profile *model.UEBAProfile) *model.AnomalySignal {
	if event == nil || profile == nil {
		return nil
	}
	if normalizeQueryType(event.Action) != "ddl" {
		return nil
	}
	ddlShare := profile.Baseline.AccessPatterns.QueryTypes["ddl"]
	if ddlShare >= d.config.DDLUnusualThreshold {
		return nil
	}
	confidence := clampConfidence(0.75 + (d.config.DDLUnusualThreshold-ddlShare)*4)
	return &model.AnomalySignal{
		SignalType:     d.Name(),
		Title:          "Unexpected DDL activity",
		Description:    "The entity executed schema-changing activity despite a historically DML-dominant profile.",
		Severity:       "high",
		Confidence:     confidence,
		ExpectedValue:  fmt.Sprintf("DDL share >= %.2f%% is normal", d.config.DDLUnusualThreshold*100),
		ActualValue:    fmt.Sprintf("historical DDL share %.2f%% and current action %s", ddlShare*100, event.Action),
		EventID:        event.ID,
		MITRETechnique: "T1068",
		MITRETactic:    "TA0004",
	}
}
