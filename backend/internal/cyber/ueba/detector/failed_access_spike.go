package detector

import (
	"context"
	"fmt"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

type failedAccessSpikeDetector struct {
	config Config
}

func (d *failedAccessSpikeDetector) Name() model.SignalType {
	return model.SignalTypeFailedAccessSpike
}

func (d *failedAccessSpikeDetector) Detect(_ context.Context, event *model.DataAccessEvent, profile *model.UEBAProfile) *model.AnomalySignal {
	if event.Success {
		return nil
	}
	stddev := profile.Baseline.FailureRate.DailyFailureCountStddev
	if stddev < d.config.FailureStddevMin {
		return nil
	}
	currentFailures := profile.Baseline.State.CurrentDayFailures
	z := zScore(currentFailures, profile.Baseline.FailureRate.DailyFailureCountMean, stddev)
	severity := ""
	switch {
	case z > d.config.FailureSpikeCriticalZ && currentFailures > d.config.FailureCriticalCount:
		severity = "critical"
	case z > d.config.FailureSpikeHighZ:
		severity = "high"
	case z > d.config.FailureSpikeMediumZ:
		severity = "medium"
	default:
		return nil
	}
	return &model.AnomalySignal{
		SignalType:     d.Name(),
		Title:          "Failed access spike",
		Description:    "The entity accumulated failed access attempts materially above its daily baseline.",
		Severity:       severity,
		Confidence:     clampConfidence(0.45 + (z / 10)),
		DeviationZ:     z,
		ExpectedValue:  fmt.Sprintf("daily failures mean %.1f +/- %.1f", profile.Baseline.FailureRate.DailyFailureCountMean, stddev),
		ActualValue:    fmt.Sprintf("current day failures %.0f", currentFailures),
		EventID:        event.ID,
		MITRETechnique: "T1110",
		MITRETactic:    "TA0006",
	}
}
