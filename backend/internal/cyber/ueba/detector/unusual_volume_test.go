package detector

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

func TestUnusualVolumeDetector(t *testing.T) {
	profile := &model.UEBAProfile{
		ID:               uuid.New(),
		EntityType:       model.EntityTypeUser,
		EntityID:         "user-1",
		ProfileMaturity:  model.ProfileMaturityMature,
		ObservationCount: 200,
	}
	profile.EnsureDefaults()
	profile.Baseline.DataVolume.DailyBytesMean = 1000
	profile.Baseline.DataVolume.DailyBytesStddev = 1000

	det := &unusualVolumeDetector{config: DefaultConfig()}
	normal := &model.DataAccessEvent{ID: uuid.New(), EventTimestamp: time.Now().UTC(), BytesAccessed: 2500}
	if signal := det.Detect(context.Background(), normal, profile); signal != nil {
		t.Fatalf("expected no signal for normal volume")
	}

	medium := &model.DataAccessEvent{ID: uuid.New(), EventTimestamp: time.Now().UTC(), BytesAccessed: 4200}
	signal := det.Detect(context.Background(), medium, profile)
	if signal == nil || signal.Severity != "medium" {
		t.Fatalf("expected medium unusual volume signal")
	}

	critical := &model.DataAccessEvent{ID: uuid.New(), EventTimestamp: time.Now().UTC(), BytesAccessed: 6500}
	signal = det.Detect(context.Background(), critical, profile)
	if signal == nil || signal.Severity != "critical" {
		t.Fatalf("expected critical unusual volume signal")
	}
}

func TestVolume_LowVariance(t *testing.T) {
	profile := &model.UEBAProfile{
		ID:               uuid.New(),
		EntityType:       model.EntityTypeUser,
		EntityID:         "user-2",
		ProfileMaturity:  model.ProfileMaturityMature,
		ObservationCount: 200,
	}
	profile.EnsureDefaults()
	// Set stddev below the UnusualVolumeStddevMin guard (default 1000).
	profile.Baseline.DataVolume.DailyBytesMean = 500
	profile.Baseline.DataVolume.DailyBytesStddev = 100
	profile.Baseline.DataVolume.DailyRowsStddev = 50

	det := &unusualVolumeDetector{config: DefaultConfig()}
	// Even with a huge value, stddev is too low to be meaningful — should skip.
	event := &model.DataAccessEvent{ID: uuid.New(), EventTimestamp: time.Now().UTC(), BytesAccessed: 100000}
	signal := det.Detect(context.Background(), event, profile)
	if signal != nil {
		t.Fatalf("expected no signal when stddev < min (%v), got %v", DefaultConfig().UnusualVolumeStddevMin, signal.Severity)
	}
}

func TestVolume_Confidence(t *testing.T) {
	profile := &model.UEBAProfile{
		ID:               uuid.New(),
		EntityType:       model.EntityTypeUser,
		EntityID:         "user-3",
		ProfileMaturity:  model.ProfileMaturityMature,
		ObservationCount: 200,
	}
	profile.EnsureDefaults()
	profile.Baseline.DataVolume.DailyBytesMean = 1000
	profile.Baseline.DataVolume.DailyBytesStddev = 1000

	det := &unusualVolumeDetector{config: DefaultConfig()}

	// z≈3.1 → confidence ≈ 0.51 (just above the threshold; z=3.0 exactly is not strictly greater)
	// (4100-1000)/1000 = 3.1
	eventZ3 := &model.DataAccessEvent{ID: uuid.New(), EventTimestamp: time.Now().UTC(), BytesAccessed: 4100}
	signalZ3 := det.Detect(context.Background(), eventZ3, profile)
	if signalZ3 == nil {
		t.Fatalf("expected signal for z≈3.1")
	}
	if signalZ3.Confidence < 0.45 || signalZ3.Confidence > 0.55 {
		t.Fatalf("z≈3.1 confidence = %v, want ≈ 0.51", signalZ3.Confidence)
	}

	// z=8 → confidence should be capped at 0.95
	eventZ8 := &model.DataAccessEvent{ID: uuid.New(), EventTimestamp: time.Now().UTC(), BytesAccessed: 9000}
	signalZ8 := det.Detect(context.Background(), eventZ8, profile)
	if signalZ8 == nil {
		t.Fatalf("expected signal for z≈8")
	}
	if signalZ8.Confidence < 0.90 || signalZ8.Confidence > 0.96 {
		t.Fatalf("z≈8 confidence = %v, want ≈ 0.95", signalZ8.Confidence)
	}
}
