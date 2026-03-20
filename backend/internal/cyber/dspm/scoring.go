package dspm

import "github.com/clario360/platform/internal/cyber/model"

// CalculateRiskScore computes the DSPM risk score and factor breakdown.
func CalculateRiskScore(sensitivityScore float64, networkExposure string, postureScore float64) (float64, []model.DSPMRiskFactor) {
	exposureFactor := 1.0
	switch networkExposure {
	case "vpn_accessible":
		exposureFactor = 1.3
	case "internet_facing":
		exposureFactor = 1.8
	}

	rawRisk := sensitivityScore * exposureFactor * (1 - (postureScore / 100))
	if rawRisk > 100 {
		rawRisk = 100
	}
	if rawRisk < 0 {
		rawRisk = 0
	}

	factors := []model.DSPMRiskFactor{
		{
			Factor:      "sensitivity",
			Description: "Intrinsic sensitivity of the data stored or processed by this asset.",
			Weight:      0.45,
			Value:       round2(sensitivityScore),
		},
		{
			Factor:      "exposure",
			Description: "How broadly reachable the data asset is from user or network entry points.",
			Weight:      0.30,
			Value:       round2(exposureFactor * 100 / 1.8),
		},
		{
			Factor:      "control_gap",
			Description: "Residual risk left after accounting for implemented posture controls.",
			Weight:      0.25,
			Value:       round2(100 - postureScore),
		},
	}

	return round2(rawRisk), factors
}
