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
