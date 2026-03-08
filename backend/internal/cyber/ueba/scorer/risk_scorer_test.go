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
