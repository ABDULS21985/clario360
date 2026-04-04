package darkdata

import (
	"testing"
	"time"

	"github.com/clario360/platform/internal/data/model"
)

func TestRiskScoreRestrictedStaleUnmanaged(t *testing.T) {
	classification := model.DataClassificationRestricted
	asset := &model.DarkDataAsset{
		ContainsPII:            true,
		InferredClassification: &classification,
		GovernanceStatus:       model.DarkDataGovernanceUnmanaged,
	}
	scorer := NewRiskScorer()
	score, factors := scorer.ScoreRisk(asset)
	if score < 90 {
		t.Fatalf("Score = %f, want near 100", score)
	}
	if len(factors) != 3 {
		t.Fatalf("factors = %d, want 3", len(factors))
	}
}

func TestRiskScoreLowRiskGoverned(t *testing.T) {
	classification := model.DataClassificationInternal
	lastAccessed := time.Now().Add(-24 * time.Hour)
	rows := int64(500)
	asset := &model.DarkDataAsset{
		ContainsPII:            false,
		InferredClassification: &classification,
		LastAccessedAt:         &lastAccessed,
		GovernanceStatus:       model.DarkDataGovernanceGoverned,
		EstimatedRowCount:      &rows,
	}
	scorer := NewRiskScorer()
	score, _ := scorer.ScoreRisk(asset)
	if score >= 30 {
		t.Fatalf("Score = %f, want low risk", score)
	}
}
