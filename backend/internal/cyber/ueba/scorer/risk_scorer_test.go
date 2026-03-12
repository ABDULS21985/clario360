package scorer

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

type fakeProfileRepo struct {
	profile *model.UEBAProfile
}

func (f *fakeProfileRepo) GetByEntity(context.Context, uuid.UUID, string) (*model.UEBAProfile, error) {
	return f.profile, nil
}

func (f *fakeProfileRepo) UpdateRisk(_ context.Context, profile *model.UEBAProfile) error {
	f.profile = profile
	return nil
}

func (f *fakeProfileRepo) DecayRiskScores(context.Context, uuid.UUID, float64, time.Time) (int64, error) {
	return 1, nil
}

type fakeAlertRepo struct {
	alerts []*model.UEBAAlert
}

func (f *fakeAlertRepo) ListByEntitySince(context.Context, uuid.UUID, string, time.Time) ([]*model.UEBAAlert, error) {
	return f.alerts, nil
}

func (f *fakeAlertRepo) UpdateRiskImpact(context.Context, uuid.UUID, uuid.UUID, float64, float64) error {
	return nil
}

func TestRiskScorerSingleCritical(t *testing.T) {
	profile := &model.UEBAProfile{ID: uuid.New(), EntityID: "user-1", RiskLevel: model.RiskLevelLow}
	profile.EnsureDefaults()
	alert := &model.UEBAAlert{
		ID:         uuid.New(),
		EntityID:   "user-1",
		Severity:   "critical",
		Confidence: 0.85,
		CreatedAt:  time.Now().UTC(),
	}
	scorer := New(&fakeProfileRepo{profile: profile}, &fakeAlertRepo{alerts: []*model.UEBAAlert{alert}}, 0.10, zerolog.Nop())
	if err := scorer.UpdateRiskScore(context.Background(), uuid.New(), "user-1", []model.UEBAAlert{*alert}); err != nil {
		t.Fatalf("UpdateRiskScore() error = %v", err)
	}
	if profile.RiskScore < 25 || profile.RiskScore > 26 {
		t.Fatalf("risk score = %v, want approximately 25.5", profile.RiskScore)
	}
}

func TestRiskMultipleAlerts(t *testing.T) {
	now := time.Now().UTC()
	profile := &model.UEBAProfile{ID: uuid.New(), EntityID: "user-multi", RiskLevel: model.RiskLevelLow}
	profile.EnsureDefaults()

	alerts := []*model.UEBAAlert{
		{ID: uuid.New(), EntityID: "user-multi", Severity: "high", Confidence: 0.8, CreatedAt: now},
		{ID: uuid.New(), EntityID: "user-multi", Severity: "high", Confidence: 0.7, CreatedAt: now.Add(-time.Hour)},
		{ID: uuid.New(), EntityID: "user-multi", Severity: "medium", Confidence: 0.6, CreatedAt: now.Add(-2 * time.Hour)},
	}

	// Expected: all within 24h → recency=1.0
	// high(0.8): 20 * 1.0 * 0.8 = 16.0
	// high(0.7): 20 * 1.0 * 0.7 = 14.0
	// medium(0.6): 10 * 1.0 * 0.6 = 6.0
	// Total = 36.0
	scorer := New(&fakeProfileRepo{profile: profile}, &fakeAlertRepo{alerts: alerts}, 0.10, zerolog.Nop())
	newAlerts := make([]model.UEBAAlert, len(alerts))
	for i, a := range alerts {
		newAlerts[i] = *a
	}
	if err := scorer.UpdateRiskScore(context.Background(), uuid.New(), "user-multi", newAlerts); err != nil {
		t.Fatalf("UpdateRiskScore() error = %v", err)
	}
	if profile.RiskScore < 35 || profile.RiskScore > 37 {
		t.Fatalf("risk score = %v, want ≈ 36.0", profile.RiskScore)
	}
	if len(profile.RiskFactors) != 3 {
		t.Fatalf("risk factors count = %d, want 3", len(profile.RiskFactors))
	}
}

func TestRiskRecencyWeight(t *testing.T) {
	now := time.Now().UTC()
	profile := &model.UEBAProfile{ID: uuid.New(), EntityID: "user-recency", RiskLevel: model.RiskLevelLow}
	profile.EnsureDefaults()

	// Alert from 10 days ago → recency weight = 0.3 (7-14d bracket)
	// critical(0.85): 30 * 0.3 * 0.85 = 7.65
	alert := &model.UEBAAlert{
		ID:         uuid.New(),
		EntityID:   "user-recency",
		Severity:   "critical",
		Confidence: 0.85,
		CreatedAt:  now.Add(-10 * 24 * time.Hour),
	}

	scorer := New(&fakeProfileRepo{profile: profile}, &fakeAlertRepo{alerts: []*model.UEBAAlert{alert}}, 0.10, zerolog.Nop())
	if err := scorer.UpdateRiskScore(context.Background(), uuid.New(), "user-recency", []model.UEBAAlert{*alert}); err != nil {
		t.Fatalf("UpdateRiskScore() error = %v", err)
	}
	// 30 * 0.3 * 0.85 = 7.65
	if profile.RiskScore < 7 || profile.RiskScore > 8.5 {
		t.Fatalf("risk score = %v, want ≈ 7.65 (10-day-old alert)", profile.RiskScore)
	}
}

func TestRiskCapped(t *testing.T) {
	now := time.Now().UTC()
	profile := &model.UEBAProfile{ID: uuid.New(), EntityID: "user-cap", RiskLevel: model.RiskLevelLow}
	profile.EnsureDefaults()

	// 10 critical alerts at max confidence, all recent → sum well over 100.
	alerts := make([]*model.UEBAAlert, 10)
	newAlerts := make([]model.UEBAAlert, 10)
	for i := 0; i < 10; i++ {
		a := &model.UEBAAlert{
			ID:         uuid.New(),
			EntityID:   "user-cap",
			Severity:   "critical",
			Confidence: 0.95,
			CreatedAt:  now.Add(-time.Duration(i) * time.Hour),
		}
		alerts[i] = a
		newAlerts[i] = *a
	}

	// Sum = 10 * (30 * 1.0 * 0.95) = 285 → capped at 100
	scorer := New(&fakeProfileRepo{profile: profile}, &fakeAlertRepo{alerts: alerts}, 0.10, zerolog.Nop())
	if err := scorer.UpdateRiskScore(context.Background(), uuid.New(), "user-cap", newAlerts); err != nil {
		t.Fatalf("UpdateRiskScore() error = %v", err)
	}
	if profile.RiskScore != 100 {
		t.Fatalf("risk score = %v, want 100 (capped)", profile.RiskScore)
	}
}

func TestRiskLevel(t *testing.T) {
	tests := []struct {
		name      string
		score     float64
		wantLevel model.RiskLevel
	}{
		{"low", 10, model.RiskLevelLow},
		{"medium", 30, model.RiskLevelMedium},
		{"high", 60, model.RiskLevelHigh},
		{"critical", 80, model.RiskLevelCritical},
		{"boundary_25", 25, model.RiskLevelMedium},
		{"boundary_50", 50, model.RiskLevelHigh},
		{"boundary_75", 75, model.RiskLevelCritical},
		{"zero", 0, model.RiskLevelLow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := riskLevelForScore(tt.score)
			if got != tt.wantLevel {
				t.Fatalf("riskLevelForScore(%v) = %s, want %s", tt.score, got, tt.wantLevel)
			}
		})
	}
}
