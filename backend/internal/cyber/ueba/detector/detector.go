package detector

import (
	"context"
	"strings"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

type GeoResolver interface {
	Country(ctx context.Context, ip string) (string, error)
}

type Config struct {
	MinMaturityForAlert model.ProfileMaturity

	UnusualTimeMatureHighProb   float64
	UnusualTimeMatureMediumProb float64
	UnusualTimeBaseHighProb     float64
	UnusualTimeBaseMediumProb   float64

	UnusualVolumeMediumZ   float64
	UnusualVolumeHighZ     float64
	UnusualVolumeCriticalZ float64
	UnusualVolumeStddevMin float64

	FailureSpikeMediumZ   float64
	FailureSpikeHighZ     float64
	FailureSpikeCriticalZ float64
	FailureStddevMin      float64
	FailureCriticalCount  float64

	BulkRowsMediumMultiplier float64
	BulkRowsHighMultiplier   float64
	DDLUnusualThreshold      float64
}

func DefaultConfig() Config {
	return Config{
		MinMaturityForAlert: model.ProfileMaturityBaseline,

		// These probabilities are intentionally conservative because UEBA favors
		// precision over recall. Mature profiles alert at <2% activity and only
		// escalate to high when the hour is effectively unseen (<0.5%).
		UnusualTimeMatureHighProb:   0.005,
		UnusualTimeMatureMediumProb: 0.02,
		// Baseline profiles require even rarer behavior before alerting because
		// they are still stabilizing between 30 and 90 days of activity.
		UnusualTimeBaseHighProb:   0.002,
		UnusualTimeBaseMediumProb: 0.01,

		// z > 3 corresponds to roughly 0.13% of normally distributed activity,
		// which is materially more actionable than z > 2 (~2.3% of events).
		UnusualVolumeMediumZ:   3,
		UnusualVolumeHighZ:     4,
		UnusualVolumeCriticalZ: 5,
		UnusualVolumeStddevMin: 1000,

		FailureSpikeMediumZ:   3,
		FailureSpikeHighZ:     5,
		FailureSpikeCriticalZ: 8,
		FailureStddevMin:      1,
		FailureCriticalCount:  50,

		// 5x and 10x daily row baselines intentionally bias toward large bursts
		// instead of smaller analyst-driven pulls that are common during work.
		BulkRowsMediumMultiplier: 5,
		BulkRowsHighMultiplier:   10,
		DDLUnusualThreshold:      0.05,
	}
}

type AnomalyDetector struct {
	config      Config
	geoResolver GeoResolver
	detectors   []SignalDetector
}

func New(config Config, geoResolver GeoResolver) *AnomalyDetector {
	d := &AnomalyDetector{
		config:      config,
		geoResolver: geoResolver,
	}
	d.detectors = []SignalDetector{
		&unusualTimeDetector{config: config},
		&unusualVolumeDetector{config: config},
		&newTableAccessDetector{},
		&newSourceIPDetector{resolver: geoResolver},
		&failedAccessSpikeDetector{config: config},
		&bulkDataAccessDetector{config: config},
		&privilegeEscalationDetector{config: config},
	}
	return d
}

func (d *AnomalyDetector) DetectAnomalies(ctx context.Context, event *model.DataAccessEvent, profile *model.UEBAProfile) []model.AnomalySignal {
	if event == nil || profile == nil {
		return nil
	}
	if profile.ProfileMaturity == model.ProfileMaturityLearning {
		return nil
	}
	if profile.Status == model.ProfileStatusSuppressed && profile.SuppressedUntil != nil && profile.SuppressedUntil.After(event.EventTimestamp) {
		return nil
	}
	if profile.Status == model.ProfileStatusWhitelisted || profile.Status == model.ProfileStatusInactive {
		return nil
	}

	signals := make([]model.AnomalySignal, 0, len(d.detectors))
	for _, instance := range d.detectors {
		signal := instance.Detect(ctx, event, profile)
		if signal == nil {
			continue
		}
		signal.EntityID = profile.EntityID
		signal.EventTimestamp = event.EventTimestamp
		// Detector outputs should never include empty severities or types.
		if signal.SignalType == "" || strings.TrimSpace(signal.Severity) == "" {
			continue
		}
		signals = append(signals, *signal)
	}
	return signals
}
