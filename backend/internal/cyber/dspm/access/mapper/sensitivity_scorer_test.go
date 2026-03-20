package mapper

import (
	"math"
	"testing"

	"github.com/clario360/platform/internal/cyber/dspm/access/model"
)

func TestSensitivityScorer_Score(t *testing.T) {
	scorer := NewSensitivityScorer()

	// Mixed assets: restricted (weight=10), confidential (weight=5), public (weight=1).
	assets := []model.AssetAccess{
		{DataClassification: "restricted", MaxPermissionLevel: "read", SensitivityWeight: 10.0},
		{DataClassification: "confidential", MaxPermissionLevel: "write", SensitivityWeight: 5.0},
		{DataClassification: "public", MaxPermissionLevel: "read", SensitivityWeight: 1.0},
	}

	// rawScore = 10*1 + 5*2 + 1*1 = 21
	// maxPossible = (10 + 5 + 1) * 5 = 80
	maxPossible := scorer.MaxPossibleScore([]float64{10, 5, 1})
	score := scorer.Score(assets, maxPossible)

	// Expected: 21/80 * 100 = 26.25
	if score < 0 || score > 100 {
		t.Errorf("score should be normalized 0-100, got %v", score)
	}
	expectedScore := 26.25
	if math.Abs(score-expectedScore) > 0.01 {
		t.Errorf("expected score ~%.2f, got %.2f", expectedScore, score)
	}
}

func TestSensitivityScorer_ScoreEmpty(t *testing.T) {
	scorer := NewSensitivityScorer()

	score := scorer.Score(nil, 100)
	if score != 0 {
		t.Errorf("empty asset list should return 0, got %v", score)
	}

	score = scorer.Score([]model.AssetAccess{}, 100)
	if score != 0 {
		t.Errorf("empty asset slice should return 0, got %v", score)
	}
}

func TestSensitivityScorer_ScoreZeroMaxPossible(t *testing.T) {
	scorer := NewSensitivityScorer()

	assets := []model.AssetAccess{
		{SensitivityWeight: 10.0, MaxPermissionLevel: "read"},
	}

	score := scorer.Score(assets, 0)
	if score != 0 {
		t.Errorf("maxPossible=0 should return 0, got %v", score)
	}

	score = scorer.Score(assets, -1)
	if score != 0 {
		t.Errorf("maxPossible<0 should return 0, got %v", score)
	}
}

func TestSensitivityScorer_MaxPossibleScore(t *testing.T) {
	scorer := NewSensitivityScorer()

	// Each weight is multiplied by 5.0 (full_control breadth).
	weights := []float64{10.0, 5.0, 2.0, 1.0}
	maxPossible := scorer.MaxPossibleScore(weights)

	expected := (10.0 + 5.0 + 2.0 + 1.0) * 5.0 // = 90.0
	if maxPossible != expected {
		t.Errorf("expected max possible score = %v, got %v", expected, maxPossible)
	}
}

func TestSensitivityScorer_MaxPossibleScoreEmpty(t *testing.T) {
	scorer := NewSensitivityScorer()

	// Empty weights should return 1 (avoid division by zero).
	maxPossible := scorer.MaxPossibleScore(nil)
	if maxPossible != 1 {
		t.Errorf("empty weights should return 1 (avoid div-by-zero), got %v", maxPossible)
	}

	maxPossible = scorer.MaxPossibleScore([]float64{})
	if maxPossible != 1 {
		t.Errorf("empty weight slice should return 1, got %v", maxPossible)
	}
}

func TestSensitivityScorer_WeightedRisk(t *testing.T) {
	scorer := NewSensitivityScorer()

	assets := []model.AssetAccess{
		{SensitivityWeight: 10.0, MaxPermissionLevel: "admin"},      // 10 * 4 = 40
		{SensitivityWeight: 5.0, MaxPermissionLevel: "write"},       // 5 * 2 = 10
		{SensitivityWeight: 1.0, MaxPermissionLevel: "read"},        // 1 * 1 = 1
		{SensitivityWeight: 2.0, MaxPermissionLevel: "full_control"}, // 2 * 5 = 10
	}

	risk := scorer.WeightedRisk(assets)
	expected := 61.0 // 40 + 10 + 1 + 10
	if risk != expected {
		t.Errorf("expected weighted risk = %v, got %v", expected, risk)
	}
}

func TestSensitivityScorer_WeightedRiskEmpty(t *testing.T) {
	scorer := NewSensitivityScorer()

	risk := scorer.WeightedRisk(nil)
	if risk != 0 {
		t.Errorf("empty assets should return 0 weighted risk, got %v", risk)
	}
}

func TestSensitivityScorer_ScoreCappedAt100(t *testing.T) {
	scorer := NewSensitivityScorer()

	// Create assets with very high raw score relative to maxPossible.
	assets := []model.AssetAccess{
		{SensitivityWeight: 100.0, MaxPermissionLevel: "full_control"}, // 100 * 5 = 500
	}

	// maxPossible = 1 (very small), so score would be 500/1*100 = 50000 without cap.
	score := scorer.Score(assets, 1)
	if score > 100 {
		t.Errorf("score should be capped at 100, got %v", score)
	}
	if score != 100 {
		t.Errorf("expected score = 100 (capped), got %v", score)
	}
}

func TestSensitivityScorer_ScoreRounding(t *testing.T) {
	scorer := NewSensitivityScorer()

	// Create a case that produces a score needing rounding to 2 decimal places.
	assets := []model.AssetAccess{
		{SensitivityWeight: 1.0, MaxPermissionLevel: "read"}, // 1 * 1 = 1
	}

	// maxPossible = 3 -> score = 1/3*100 = 33.333...
	score := scorer.Score(assets, 3)
	// Should be rounded to 2 decimal places: 33.33
	expected := 33.33
	if math.Abs(score-expected) > 0.01 {
		t.Errorf("expected rounded score = %.2f, got %.2f", expected, score)
	}
}
