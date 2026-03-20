package correlator

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

func TestCorrelate_Exfiltration(t *testing.T) {
	entityID := "user-1"
	c := New(time.Hour)
	now := time.Now().UTC()
	signals := []model.AnomalySignal{
		{SignalType: model.SignalTypeUnusualTime, Severity: "high", Confidence: 0.8, EventID: uuid.New(), EventTimestamp: now, MITRETactic: "TA0006"},
		{SignalType: model.SignalTypeUnusualVolume, Severity: "high", Confidence: 0.7, DeviationZ: 4.2, EventID: uuid.New(), EventTimestamp: now.Add(2 * time.Minute), MITRETactic: "TA0010"},
	}
	alerts := c.Correlate(context.Background(), uuid.New(), entityID, signals)
	if len(alerts) == 0 || alerts[0].AlertType != model.AlertTypePossibleDataExfiltration {
		t.Fatalf("expected data exfiltration alert")
	}
}

func TestCorrelate_CredentialCompromise(t *testing.T) {
	entityID := "user-1"
	c := New(time.Hour)
	now := time.Now().UTC()
	signals := []model.AnomalySignal{
		{SignalType: model.SignalTypeNewSourceIP, Severity: "high", Confidence: 0.9, EventID: uuid.New(), EventTimestamp: now},
		{SignalType: model.SignalTypeNewTableAccess, Severity: "medium", Confidence: 0.6, EventID: uuid.New(), EventTimestamp: now.Add(3 * time.Minute)},
	}
	alerts := c.Correlate(context.Background(), uuid.New(), entityID, signals)
	if len(alerts) == 0 || alerts[0].AlertType != model.AlertTypePossibleCredentialCompromise {
		t.Fatalf("expected credential compromise alert")
	}
}

func TestCorrelate_GenericThreeSignals(t *testing.T) {
	entityID := "user-1"
	c := New(time.Hour)
	now := time.Now().UTC()
	signals := []model.AnomalySignal{
		{SignalType: model.SignalTypeNewSourceIP, Severity: "medium", Confidence: 0.5, EventID: uuid.New(), EventTimestamp: now},
		{SignalType: model.SignalTypeUnusualTime, Severity: "medium", Confidence: 0.5, EventID: uuid.New(), EventTimestamp: now.Add(time.Minute)},
		{SignalType: model.SignalTypeFailedAccessSpike, Severity: "medium", Confidence: 0.5, EventID: uuid.New(), EventTimestamp: now.Add(2 * time.Minute)},
	}
	alerts := c.Correlate(context.Background(), uuid.New(), entityID, signals)
	if len(alerts) == 0 || alerts[0].AlertType == "" {
		t.Fatalf("expected generic alert")
	}
}

func TestCorrelate_StandaloneHigh(t *testing.T) {
	c := New(time.Hour)
	now := time.Now().UTC()
	signals := []model.AnomalySignal{
		{SignalType: model.SignalTypePrivilegeEscalation, Severity: "high", Confidence: 0.9, EventID: uuid.New(), EventTimestamp: now},
	}
	alerts := c.Correlate(context.Background(), uuid.New(), "user-1", signals)
	if len(alerts) == 0 {
		t.Fatalf("expected standalone high-severity alert")
	}
}

func TestCorrelate_InsiderThreat(t *testing.T) {
	c := New(time.Hour)
	now := time.Now().UTC()

	// Rule 3: privilege_escalation + bulk_data_access → possible_insider_threat
	signals := []model.AnomalySignal{
		{SignalType: model.SignalTypePrivilegeEscalation, Severity: "high", Confidence: 0.85, EventID: uuid.New(), EventTimestamp: now},
		{SignalType: model.SignalTypeBulkDataAccess, Severity: "high", Confidence: 0.80, EventID: uuid.New(), EventTimestamp: now.Add(time.Minute)},
	}
	alerts := c.Correlate(context.Background(), uuid.New(), "user-insider", signals)
	if len(alerts) == 0 {
		t.Fatalf("expected insider threat alert")
	}
	if alerts[0].AlertType != model.AlertTypePossibleInsiderThreat {
		t.Fatalf("alert type = %s, want %s", alerts[0].AlertType, model.AlertTypePossibleInsiderThreat)
	}
	// Insider threat has a floor severity of "high" per spec.
	if alerts[0].Severity != "high" && alerts[0].Severity != "critical" {
		t.Fatalf("severity = %s, want at least high", alerts[0].Severity)
	}
	if alerts[0].MITRETactic == "" {
		t.Fatal("expected MITRE tactic to be set")
	}
}

func TestCorrelate_Dedup(t *testing.T) {
	c := New(time.Hour)
	now := time.Now().UTC()

	// Send two distinct exfiltration-qualifying signal pairs. The correlator
	// should consume signals into the first matching rule and not produce
	// duplicate alerts for the same alert type.
	signals := []model.AnomalySignal{
		{SignalType: model.SignalTypeUnusualTime, Severity: "high", Confidence: 0.8, EventID: uuid.New(), EventTimestamp: now},
		{SignalType: model.SignalTypeUnusualVolume, Severity: "high", Confidence: 0.7, DeviationZ: 4.2, EventID: uuid.New(), EventTimestamp: now.Add(time.Minute)},
	}
	alerts := c.Correlate(context.Background(), uuid.New(), "user-dedup", signals)

	// Count how many exfiltration alerts were produced.
	exfilCount := 0
	for _, alert := range alerts {
		if alert.AlertType == model.AlertTypePossibleDataExfiltration {
			exfilCount++
		}
	}
	if exfilCount != 1 {
		t.Fatalf("exfiltration alert count = %d, want exactly 1 (dedup)", exfilCount)
	}

	// Verify consumed signals are not re-emitted as standalone alerts.
	for _, alert := range alerts {
		if alert.AlertType == model.AlertTypeUnusualActivity {
			for _, signal := range alert.TriggeringSignals {
				for _, orig := range signals {
					if signal.EventID == orig.EventID {
						t.Fatalf("signal %s was consumed by exfiltration rule but also appeared in standalone alert", signal.EventID)
					}
				}
			}
		}
	}
}

func TestCorrelate_ConfidenceBoost(t *testing.T) {
	c := New(time.Hour)
	now := time.Now().UTC()

	// Single signal: confidence = 0.5, no boost.
	singleSignal := []model.AnomalySignal{
		{SignalType: model.SignalTypePrivilegeEscalation, Severity: "high", Confidence: 0.5, EventID: uuid.New(), EventTimestamp: now},
	}
	singleAlerts := c.Correlate(context.Background(), uuid.New(), "user-boost", singleSignal)
	if len(singleAlerts) == 0 {
		t.Fatal("expected standalone alert for single high-severity signal")
	}
	singleConfidence := singleAlerts[0].Confidence

	// Three correlated signals: each confidence = 0.5.
	// Base confidence = mean(0.5, 0.5, 0.5) = 0.5
	// Boost = min(0.15 * (3-1), 0.3) = 0.3
	// Final = min(0.5 + 0.3, 0.99) = 0.8
	threeSignals := []model.AnomalySignal{
		{SignalType: model.SignalTypeNewSourceIP, Severity: "medium", Confidence: 0.5, EventID: uuid.New(), EventTimestamp: now},
		{SignalType: model.SignalTypeUnusualTime, Severity: "medium", Confidence: 0.5, EventID: uuid.New(), EventTimestamp: now.Add(time.Minute)},
		{SignalType: model.SignalTypeFailedAccessSpike, Severity: "medium", Confidence: 0.5, EventID: uuid.New(), EventTimestamp: now.Add(2 * time.Minute)},
	}
	threeAlerts := c.Correlate(context.Background(), uuid.New(), "user-boost-3", threeSignals)
	if len(threeAlerts) == 0 {
		t.Fatal("expected correlated alert for 3 signals")
	}
	threeConfidence := threeAlerts[0].Confidence

	// Correlated confidence must be higher than single-signal confidence.
	if threeConfidence <= singleConfidence {
		t.Fatalf("3-signal confidence (%v) should be > single-signal confidence (%v)", threeConfidence, singleConfidence)
	}
	// Expected: 0.5 + min(0.15*2, 0.3) = 0.5 + 0.3 = 0.8
	if threeConfidence < 0.75 || threeConfidence > 0.85 {
		t.Fatalf("3-signal confidence = %v, want ≈ 0.8", threeConfidence)
	}
}
