package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCampaignDetectorFindsCluster(t *testing.T) {
	t.Parallel()

	model := NewCampaignDetector("")
	base := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	samples := []CampaignAlertSample{
		{AlertID: uuid.New(), Title: "Phishing 1", Timestamp: base, Embedding: []float64{1, 0, 0}, IOCs: []string{"1.1.1.1"}, Techniques: []string{"T1566"}, TargetAssets: []string{"a"}},
		{AlertID: uuid.New(), Title: "Phishing 2", Timestamp: base.Add(2 * time.Hour), Embedding: []float64{0.9, 0.1, 0}, IOCs: []string{"1.1.1.1"}, Techniques: []string{"T1566"}, TargetAssets: []string{"a"}},
		{AlertID: uuid.New(), Title: "Phishing 3", Timestamp: base.Add(4 * time.Hour), Embedding: []float64{0.85, 0.15, 0}, IOCs: []string{"1.1.1.1"}, Techniques: []string{"T1566"}, TargetAssets: []string{"b"}},
		{AlertID: uuid.New(), Title: "Unrelated", Timestamp: base.Add(96 * time.Hour), Embedding: []float64{0, 1, 0}, IOCs: []string{"2.2.2.2"}, Techniques: []string{"T1110"}, TargetAssets: []string{"z"}},
	}
	if err := model.Train(samples); err != nil {
		t.Fatalf("train error: %v", err)
	}
	clusters := model.Detect(samples)
	if len(clusters) == 0 {
		t.Fatal("expected at least one cluster")
	}
	if len(clusters[0].AlertIDs) < 2 {
		t.Fatalf("cluster size = %d, want >= 2", len(clusters[0].AlertIDs))
	}
}

func TestCampaignDetectorSerializeRoundTrip(t *testing.T) {
	t.Parallel()

	model := NewCampaignDetector("campaign-vtest")
	payload, err := model.Serialize()
	if err != nil {
		t.Fatalf("serialize error: %v", err)
	}
	loaded := NewCampaignDetector("")
	if err := loaded.Deserialize(payload); err != nil {
		t.Fatalf("deserialize error: %v", err)
	}
	if loaded.ModelVersion != "campaign-vtest" {
		t.Fatalf("version = %q", loaded.ModelVersion)
	}
}
