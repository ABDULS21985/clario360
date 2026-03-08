package detector

import (
	"context"
	"fmt"
	"strings"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

type bulkDataAccessDetector struct {
	config Config
}

func (d *bulkDataAccessDetector) Name() model.SignalType {
	return model.SignalTypeBulkDataAccess
}

func (d *bulkDataAccessDetector) Detect(_ context.Context, event *model.DataAccessEvent, profile *model.UEBAProfile) *model.AnomalySignal {
	meanRows := profile.Baseline.DataVolume.DailyRowsMean
	if meanRows <= 0 || event.RowsAccessed <= 0 {
		return nil
	}

	severity := ""
	switch {
	case float64(event.RowsAccessed) > d.config.BulkRowsHighMultiplier*meanRows:
		severity = "high"
	case float64(event.RowsAccessed) > d.config.BulkRowsMediumMultiplier*meanRows:
		severity = "medium"
	default:
		return nil
	}
	if hasNoWhereClause(event) {
		severity = escalateSeverity(severity, 1)
	}
	return &model.AnomalySignal{
		SignalType:     d.Name(),
		Title:          "Bulk data access",
		Description:    "The entity read far more rows than its daily baseline predicts for a single access event.",
		Severity:       severity,
		Confidence:     clampConfidence(0.55 + (float64(event.RowsAccessed) / (meanRows * 20))),
		ExpectedValue:  fmt.Sprintf("single event row count <= %.0f", d.config.BulkRowsMediumMultiplier*meanRows),
		ActualValue:    fmt.Sprintf("single event row count %d", event.RowsAccessed),
		EventID:        event.ID,
		MITRETechnique: "T1530",
		MITRETactic:    "TA0010",
	}
}

func hasNoWhereClause(event *model.DataAccessEvent) bool {
	if event == nil || event.Action != "select" {
		return false
	}
	query := strings.ToLower(strings.TrimSpace(event.QueryPreview))
	if query == "" {
		return false
	}
	return strings.HasPrefix(query, "select") && !strings.Contains(query, " where ")
}
