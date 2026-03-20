package models

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/google/uuid"
)

type AssetRiskSample struct {
	AssetID              uuid.UUID `json:"asset_id"`
	AssetName            string    `json:"asset_name"`
	AssetType            string    `json:"asset_type"`
	CriticalityScore     float64   `json:"criticality_score"`
	OpenCritical         float64   `json:"open_critical"`
	OpenHigh             float64   `json:"open_high"`
	PatchAgeDays         float64   `json:"patch_age_days"`
	InternetFacing       float64   `json:"internet_facing"`
	HistoricalAlerts     float64   `json:"historical_alerts"`
	UserAccessCount      float64   `json:"user_access_count"`
	DataSensitivity      float64   `json:"data_sensitivity"`
	IndustrySignal       float64   `json:"industry_signal"`
	TechniqueCoverageGap float64   `json:"technique_coverage_gap"`
	TargetedLabel        float64   `json:"targeted_label"`
}

type AssetRiskPredictor struct {
	ModelVersion string             `json:"model_version"`
	Intercept    float64            `json:"intercept"`
	Weights      map[string]float64 `json:"weights"`
	Baseline     map[string]float64 `json:"baseline"`
	Residuals    []float64          `json:"residuals"`
}

func NewAssetRiskPredictor(version string) *AssetRiskPredictor {
	if version == "" {
		version = "asset-risk-v1"
	}
	return &AssetRiskPredictor{
		ModelVersion: version,
		Intercept:    -1.75,
		Weights: map[string]float64{
			"criticality_score":      0.45,
			"open_critical":          0.60,
			"open_high":              0.35,
			"patch_age_days":         0.02,
			"internet_facing":        0.75,
			"historical_alerts":      0.08,
			"user_access_count":      0.01,
			"data_sensitivity":       0.30,
			"industry_signal":        0.25,
			"technique_coverage_gap": 0.40,
		},
		Baseline: map[string]float64{},
	}
}

func (m *AssetRiskPredictor) Train(samples []AssetRiskSample) error {
	if len(samples) < 5 {
		return fmt.Errorf("at least 5 asset samples are required")
	}
	features := []string{
		"criticality_score", "open_critical", "open_high", "patch_age_days", "internet_facing",
		"historical_alerts", "user_access_count", "data_sensitivity", "industry_signal", "technique_coverage_gap",
	}
	for _, feature := range features {
		values := make([]float64, 0, len(samples))
		positive := make([]float64, 0, len(samples))
		negative := make([]float64, 0, len(samples))
		for _, sample := range samples {
			value := sampleValue(sample, feature)
			values = append(values, value)
			if sample.TargetedLabel >= 0.5 {
				positive = append(positive, value)
			} else {
				negative = append(negative, value)
			}
		}
		m.Baseline[feature] = mean(values)
		if len(positive) > 0 && len(negative) > 0 {
			diff := mean(positive) - mean(negative)
			if diff != 0 {
				sign := diff / math.Abs(diff)
				m.Weights[feature] = sign * math.Max(math.Abs(m.Weights[feature]), math.Abs(diff)/(math.Abs(m.Baseline[feature])+1))
			}
		}
	}
	rate := 0.0
	for _, sample := range samples {
		rate += sample.TargetedLabel
	}
	rate = clamp(rate/float64(len(samples)), 0.05, 0.95)
	m.Intercept = math.Log(rate / (1 - rate))
	m.Residuals = m.Residuals[:0]
	for _, sample := range samples {
		score := m.Predict(sample)
		m.Residuals = append(m.Residuals, sample.TargetedLabel-score)
	}
	return nil
}

func (m *AssetRiskPredictor) Predict(sample AssetRiskSample) float64 {
	score := m.Intercept
	for feature, weight := range m.Weights {
		baseline := m.Baseline[feature]
		score += (sampleValue(sample, feature) - baseline) * weight
	}
	return clamp(logistic(score), 0.01, 0.99)
}

func (m *AssetRiskPredictor) Serialize() ([]byte, error) {
	return json.Marshal(m)
}

func (m *AssetRiskPredictor) Deserialize(payload []byte) error {
	return json.Unmarshal(payload, m)
}

func sampleValue(sample AssetRiskSample, feature string) float64 {
	switch feature {
	case "criticality_score":
		return sample.CriticalityScore
	case "open_critical":
		return sample.OpenCritical
	case "open_high":
		return sample.OpenHigh
	case "patch_age_days":
		return sample.PatchAgeDays
	case "internet_facing":
		return sample.InternetFacing
	case "historical_alerts":
		return sample.HistoricalAlerts
	case "user_access_count":
		return sample.UserAccessCount
	case "data_sensitivity":
		return sample.DataSensitivity
	case "industry_signal":
		return sample.IndustrySignal
	case "technique_coverage_gap":
		return sample.TechniqueCoverageGap
	default:
		return 0
	}
}
