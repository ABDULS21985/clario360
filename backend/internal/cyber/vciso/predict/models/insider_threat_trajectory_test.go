package models

import (
	"testing"
	"time"
)

func TestInsiderThreatTrajectoryPredictsAcceleration(t *testing.T) {
	t.Parallel()

	model := NewInsiderThreatTrajectoryModel("")
	series := []InsiderThreatSample{
		{EntityID: "u1", Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), RiskScore: 40, LoginAnomalies: 1},
		{EntityID: "u1", Timestamp: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), RiskScore: 50, LoginAnomalies: 2, DataAccessTrend: 1},
		{EntityID: "u1", Timestamp: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC), RiskScore: 62, LoginAnomalies: 3, DataAccessTrend: 2, PolicyViolations: 1},
		{EntityID: "u1", Timestamp: time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC), RiskScore: 70, LoginAnomalies: 4, DataAccessTrend: 3, PolicyViolations: 2, PeerDeviation: 1},
	}
	if err := model.Train(map[string][]InsiderThreatSample{"u1": series}); err != nil {
		t.Fatalf("train error: %v", err)
	}
	projected, accelerating, daysToThreshold := model.Predict(series, 7, 80)
	if projected <= series[len(series)-1].RiskScore {
		t.Fatalf("projected = %.2f, want > %.2f", projected, series[len(series)-1].RiskScore)
	}
	if !accelerating {
		t.Fatal("expected accelerating trajectory")
	}
	if daysToThreshold == nil {
		t.Fatal("expected threshold crossing estimate")
	}
}

func TestInsiderThreatTrajectorySerializeRoundTrip(t *testing.T) {
	t.Parallel()

	model := NewInsiderThreatTrajectoryModel("insider-vtest")
	payload, err := model.Serialize()
	if err != nil {
		t.Fatalf("serialize error: %v", err)
	}
	loaded := NewInsiderThreatTrajectoryModel("")
	if err := loaded.Deserialize(payload); err != nil {
		t.Fatalf("deserialize error: %v", err)
	}
	if loaded.ModelVersion != "insider-vtest" {
		t.Fatalf("version = %q", loaded.ModelVersion)
	}
}
