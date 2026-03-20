package detector

import (
	"context"
	"fmt"
	"math"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

type unusualVolumeDetector struct {
	config Config
}

func (d *unusualVolumeDetector) Name() model.SignalType {
	return model.SignalTypeUnusualVolume
}

func (d *unusualVolumeDetector) Detect(_ context.Context, event *model.DataAccessEvent, profile *model.UEBAProfile) *model.AnomalySignal {
	if event == nil || profile == nil {
		return nil
	}
	if profile.ObservationCount < 50 {
		return nil
	}
	bytesStddev := profile.Baseline.DataVolume.DailyBytesStddev
	rowsStddev := profile.Baseline.DataVolume.DailyRowsStddev
	if bytesStddev < d.config.UnusualVolumeStddevMin && rowsStddev < d.config.UnusualVolumeStddevMin {
		return nil
	}

	bestZ := 0.0
	expected := ""
	actual := ""
	if bytesStddev >= d.config.UnusualVolumeStddevMin {
		z := zScore(float64(event.BytesAccessed), profile.Baseline.DataVolume.DailyBytesMean, bytesStddev)
		if z > bestZ {
			bestZ = z
			expected = fmt.Sprintf("daily bytes mean %.0f +/- %.0f", profile.Baseline.DataVolume.DailyBytesMean, bytesStddev)
			actual = fmt.Sprintf("single event bytes %d", event.BytesAccessed)
		}
	}
	if rowsStddev >= d.config.UnusualVolumeStddevMin {
		z := zScore(float64(event.RowsAccessed), profile.Baseline.DataVolume.DailyRowsMean, rowsStddev)
		if z > bestZ {
			bestZ = z
			expected = fmt.Sprintf("daily rows mean %.0f +/- %.0f", profile.Baseline.DataVolume.DailyRowsMean, rowsStddev)
			actual = fmt.Sprintf("single event rows %d", event.RowsAccessed)
		}
	}

	severity := ""
	switch {
	case bestZ > d.config.UnusualVolumeCriticalZ:
		severity = "critical"
	case bestZ > d.config.UnusualVolumeHighZ:
		severity = "high"
	case bestZ > d.config.UnusualVolumeMediumZ:
		severity = "medium"
	default:
		return nil
	}

	return &model.AnomalySignal{
		SignalType:     d.Name(),
		Title:          "Volume materially above baseline",
		Description:    "The entity accessed data at a volume that significantly exceeded its learned distribution.",
		Severity:       severity,
		Confidence:     clampConfidence(math.Min(0.95, 0.5+(bestZ-3)*0.1)),
		DeviationZ:     bestZ,
		ExpectedValue:  expected,
		ActualValue:    actual,
		EventID:        event.ID,
		MITRETechnique: "T1567",
		MITRETactic:    "TA0010",
	}
}

func zScore(actual, mean, stddev float64) float64 {
	if stddev == 0 {
		return 0
	}
	return (actual - mean) / stddev
}
