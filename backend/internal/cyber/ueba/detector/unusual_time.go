package detector

import (
	"context"
	"fmt"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

type unusualTimeDetector struct {
	config Config
}

func (d *unusualTimeDetector) Name() model.SignalType {
	return model.SignalTypeUnusualTime
}

func (d *unusualTimeDetector) Detect(_ context.Context, event *model.DataAccessEvent, profile *model.UEBAProfile) *model.AnomalySignal {
	if event == nil || profile == nil {
		return nil
	}
	if profile.EntityType == model.EntityTypeServiceAccount || event.Action == "login" {
		return nil
	}

	hourProb := profile.Baseline.AccessTimes.HourlyDistribution[event.EventTimestamp.Hour()]
	highProb := d.config.UnusualTimeMatureHighProb
	mediumProb := d.config.UnusualTimeMatureMediumProb
	if profile.ProfileMaturity == model.ProfileMaturityBaseline {
		highProb = d.config.UnusualTimeBaseHighProb
		mediumProb = d.config.UnusualTimeBaseMediumProb
	}

	severity := ""
	switch {
	case hourProb < highProb:
		severity = "high"
	case hourProb < mediumProb:
		severity = "medium"
	default:
		return nil
	}

	return &model.AnomalySignal{
		SignalType:     d.Name(),
		Title:          "Activity outside learned access window",
		Description:    "The entity accessed data during an hour that is materially outside its historical distribution.",
		Severity:       severity,
		Confidence:     clampConfidence(1 - hourProb),
		ExpectedValue:  fmt.Sprintf("historical probability >= %.2f%%", mediumProb*100),
		ActualValue:    fmt.Sprintf("hour %02d:00 probability %.2f%%", event.EventTimestamp.Hour(), hourProb*100),
		EventID:        event.ID,
		MITRETechnique: "T1078",
		MITRETactic:    "TA0006",
	}
}
