package detector

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

func TestUnusualTimeDetector(t *testing.T) {
	baseProfile := &model.UEBAProfile{
		ID:              uuid.New(),
		EntityType:      model.EntityTypeUser,
		EntityID:        "user-1",
		ProfileMaturity: model.ProfileMaturityMature,
	}
	baseProfile.EnsureDefaults()
	for hour := 9; hour <= 17; hour++ {
		baseProfile.Baseline.AccessTimes.HourlyDistribution[hour] = 1.0 / 9.0
	}

	det := New(DefaultConfig(), nil)
	normalEvent := &model.DataAccessEvent{ID: uuid.New(), EventTimestamp: time.Date(2026, 3, 8, 14, 0, 0, 0, time.UTC)}
	if signals := det.DetectAnomalies(context.Background(), normalEvent, baseProfile); len(signals) != 0 {
		t.Fatalf("expected no signal for normal hour, got %d", len(signals))
	}

	rareEvent := &model.DataAccessEvent{ID: uuid.New(), EventTimestamp: time.Date(2026, 3, 8, 3, 0, 0, 0, time.UTC)}
	signals := det.DetectAnomalies(context.Background(), rareEvent, baseProfile)
	if len(signals) == 0 || signals[0].SignalType != model.SignalTypeUnusualTime {
		t.Fatalf("expected unusual_time signal for rare hour")
	}

	learning := *baseProfile
	learning.ProfileMaturity = model.ProfileMaturityLearning
	if signals := det.DetectAnomalies(context.Background(), rareEvent, &learning); len(signals) != 0 {
		t.Fatalf("expected no signals for learning profile")
	}

	serviceAccount := *baseProfile
	serviceAccount.EntityType = model.EntityTypeServiceAccount
	if signals := det.DetectAnomalies(context.Background(), rareEvent, &serviceAccount); len(signals) != 0 {
		t.Fatalf("expected no signals for service account")
	}
}
