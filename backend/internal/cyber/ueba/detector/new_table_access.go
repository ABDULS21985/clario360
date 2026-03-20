package detector

import (
	"context"
	"fmt"
	"strings"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

type newTableAccessDetector struct{}

func (d *newTableAccessDetector) Name() model.SignalType {
	return model.SignalTypeNewTableAccess
}

func (d *newTableAccessDetector) Detect(_ context.Context, event *model.DataAccessEvent, profile *model.UEBAProfile) *model.AnomalySignal {
	tableName := qualifiedTableName(event.SchemaName, event.TableName)
	if tableName == "" {
		return nil
	}
	for _, known := range profile.Baseline.AccessPatterns.TablesAccessed {
		if strings.EqualFold(known.Name, tableName) {
			return nil
		}
	}

	sensitivity := strings.ToLower(strings.TrimSpace(event.TableSensitivity))
	if sensitivity == "" {
		sensitivity = "internal"
	}
	if profile.ProfileMaturity == model.ProfileMaturityBaseline && sensitivity != "restricted" {
		return nil
	}

	severity := ""
	confidence := 0.0
	switch sensitivity {
	case "restricted":
		severity = "high"
		confidence = 0.8
	case "confidential":
		severity = "medium"
		confidence = 0.6
	case "internal":
		severity = "low"
		confidence = 0.3
	default:
		return nil
	}

	return &model.AnomalySignal{
		SignalType:       d.Name(),
		Title:            "First-time table access",
		Description:      "The entity accessed a table that does not appear in its learned working set.",
		Severity:         severity,
		Confidence:       confidence,
		ExpectedValue:    "previously accessed tables only",
		ActualValue:      fmt.Sprintf("new table %s (%s)", tableName, sensitivity),
		EventID:          event.ID,
		MITRETechnique:   "T1213",
		MITRETactic:      "TA0009",
		TableSensitivity: sensitivity,
	}
}
