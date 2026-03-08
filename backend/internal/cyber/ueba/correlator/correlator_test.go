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
