package models

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	predictmodel "github.com/clario360/platform/internal/cyber/vciso/predict/model"
)

type TechniqueTrendSample struct {
	TechniqueID         string    `json:"technique_id"`
	TechniqueName       string    `json:"technique_name"`
	Timestamp           time.Time `json:"timestamp"`
	InternalCount       float64   `json:"internal_count"`
	IndustryCount       float64   `json:"industry_count"`
	CampaignCorrelation float64   `json:"campaign_correlation"`
	Seasonality         float64   `json:"seasonality"`
}

type TechniqueTrendState struct {
	TechniqueID   string    `json:"technique_id"`
	TechniqueName string    `json:"technique_name"`
	BaseLevel     float64   `json:"base_level"`
	GrowthRate    float64   `json:"growth_rate"`
	LastObserved  float64   `json:"last_observed"`
	LastTimestamp time.Time `json:"last_timestamp"`
}

type TechniqueTrendAnalyzer struct {
	ModelVersion string                          `json:"model_version"`
	Weights      map[string]float64              `json:"weights"`
	States       map[string]TechniqueTrendState  `json:"states"`
	Residuals    []float64                       `json:"residuals"`
	LastSamples  map[string]TechniqueTrendSample `json:"last_samples"`
}

func NewTechniqueTrendAnalyzer(version string) *TechniqueTrendAnalyzer {
	if version == "" {
		version = "technique-trend-v1"
	}
	return &TechniqueTrendAnalyzer{
		ModelVersion: version,
		Weights: map[string]float64{
			"internal_count":       0.55,
			"industry_count":       0.25,
			"campaign_correlation": 0.15,
			"seasonality":          0.05,
		},
		States:      map[string]TechniqueTrendState{},
		LastSamples: map[string]TechniqueTrendSample{},
	}
}

func (m *TechniqueTrendAnalyzer) Train(samples []TechniqueTrendSample) error {
	if len(samples) == 0 {
		return fmt.Errorf("at least 1 technique sample is required")
	}
	grouped := map[string][]TechniqueTrendSample{}
	for _, sample := range samples {
		grouped[sample.TechniqueID] = append(grouped[sample.TechniqueID], sample)
		if previous, ok := m.LastSamples[sample.TechniqueID]; !ok || sample.Timestamp.After(previous.Timestamp) {
			m.LastSamples[sample.TechniqueID] = sample
		}
	}
	m.Residuals = m.Residuals[:0]
	for techniqueID, items := range grouped {
		sort.SliceStable(items, func(i, j int) bool { return items[i].Timestamp.Before(items[j].Timestamp) })
		series := make([]float64, 0, len(items))
		for _, item := range items {
			series = append(series, combinedTechniqueSignal(item, m.Weights))
		}
		state := TechniqueTrendState{
			TechniqueID:   techniqueID,
			TechniqueName: items[len(items)-1].TechniqueName,
			BaseLevel:     rollingEMA(series, 0.35),
			GrowthRate:    slope(series),
			LastObserved:  series[len(series)-1],
			LastTimestamp: items[len(items)-1].Timestamp,
		}
		m.States[techniqueID] = state
		for idx, actual := range series {
			expected := state.BaseLevel + state.GrowthRate*float64(idx-len(series)+1)
			m.Residuals = append(m.Residuals, actual-expected)
		}
	}
	return nil
}

func (m *TechniqueTrendAnalyzer) Predict(horizonDays int) []predictmodel.TechniqueTrendItem {
	if horizonDays <= 0 {
		horizonDays = 30
	}
	items := make([]predictmodel.TechniqueTrendItem, 0, len(m.States))
	spread := math.Max(percentile(absResiduals(m.Residuals), 0.90), 0.5)
	for techniqueID, state := range m.States {
		value := state.BaseLevel + state.GrowthRate*float64(horizonDays)
		trend := "stable"
		switch {
		case state.GrowthRate > 0.2:
			trend = "increasing"
		case state.GrowthRate < -0.2:
			trend = "decreasing"
		}
		items = append(items, predictmodel.TechniqueTrendItem{
			TechniqueID:   techniqueID,
			TechniqueName: state.TechniqueName,
			Trend:         trend,
			GrowthRate:    math.Round(state.GrowthRate*1000) / 1000,
			Forecast: predictmodel.ConfidenceInterval{
				P10: math.Max(0, value-spread),
				P50: math.Max(0, value),
				P90: math.Max(0, value+spread),
			},
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].GrowthRate == items[j].GrowthRate {
			return items[i].TechniqueID < items[j].TechniqueID
		}
		return items[i].GrowthRate > items[j].GrowthRate
	})
	return items
}

func (m *TechniqueTrendAnalyzer) Serialize() ([]byte, error) {
	return json.Marshal(m)
}

func (m *TechniqueTrendAnalyzer) Deserialize(payload []byte) error {
	return json.Unmarshal(payload, m)
}

func combinedTechniqueSignal(sample TechniqueTrendSample, weights map[string]float64) float64 {
	return sample.InternalCount*weights["internal_count"] +
		sample.IndustryCount*weights["industry_count"] +
		sample.CampaignCorrelation*weights["campaign_correlation"] +
		sample.Seasonality*weights["seasonality"]
}

func absResiduals(values []float64) []float64 {
	out := make([]float64, 0, len(values))
	for _, value := range values {
		out = append(out, math.Abs(value))
	}
	return out
}
