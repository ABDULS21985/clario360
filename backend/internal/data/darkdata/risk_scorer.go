package darkdata

import (
	"time"

	"github.com/clario360/platform/internal/data/model"
)

type DarkDataRiskScorer struct{}

func NewRiskScorer() *DarkDataRiskScorer {
	return &DarkDataRiskScorer{}
}

func (s *DarkDataRiskScorer) ScoreRisk(asset *model.DarkDataAsset) (float64, []model.RiskFactor) {
	if asset == nil {
		return 0, nil
	}
	sensitivity := sensitivityScore(asset)
	staleness := stalenessFactor(asset)
	governance := governanceFactor(asset)
	score := sensitivity * staleness * governance
	if score > 100 {
		score = 100
	}
	factors := []model.RiskFactor{
		{Factor: "sensitivity", Value: sensitivity, Description: "Sensitivity score derived from PII and classification"},
		{Factor: "staleness", Value: staleness, Description: "Staleness multiplier based on last access"},
		{Factor: "governance", Value: governance, Description: "Governance multiplier based on current control status"},
	}
	return score, factors
}

func sensitivityScore(asset *model.DarkDataAsset) float64 {
	classification := model.DataClassificationPublic
	if asset.InferredClassification != nil {
		classification = *asset.InferredClassification
	}
	switch {
	case asset.ContainsPII && classification == model.DataClassificationRestricted:
		return 40
	case asset.ContainsPII && classification == model.DataClassificationConfidential:
		return 30
	case !asset.ContainsPII && asset.EstimatedRowCount != nil && *asset.EstimatedRowCount > 100000:
		return 20
	default:
		return 10
	}
}

func stalenessFactor(asset *model.DarkDataAsset) float64 {
	if asset.LastAccessedAt == nil {
		return 2.5
	}
	days := int(time.Since(*asset.LastAccessedAt).Hours() / 24)
	switch {
	case days > 365:
		return 2.0
	case days > 180:
		return 1.5
	case days > 90:
		return 1.2
	default:
		return 1.0
	}
}

func governanceFactor(asset *model.DarkDataAsset) float64 {
	switch asset.GovernanceStatus {
	case model.DarkDataGovernanceUnderReview:
		return 1.5
	case model.DarkDataGovernanceGoverned:
		return 1.0
	case model.DarkDataGovernanceArchived:
		return 1.2
	default:
		return 2.5
	}
}
