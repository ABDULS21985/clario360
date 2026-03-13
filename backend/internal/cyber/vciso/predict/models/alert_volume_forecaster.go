package models

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	predictmodel "github.com/clario360/platform/internal/cyber/vciso/predict/model"
)

type AlertVolumeSample struct {
	Timestamp         time.Time `json:"timestamp"`
	AlertCount        float64   `json:"alert_count"`
	ThreatActivity    float64   `json:"threat_activity"`
	AssetOnboarding   float64   `json:"asset_onboarding"`
	DetectionChanges  float64   `json:"detection_changes"`
	MaintenanceWindow float64   `json:"maintenance_window"`
	Holiday           float64   `json:"holiday"`
}

type AlertVolumeForecaster struct {
	ModelVersion   string             `json:"model_version"`
	BaseLevel      float64            `json:"base_level"`
	Trend          float64            `json:"trend"`
	WeekdayFactor  map[int]float64    `json:"weekday_factor"`
	FeatureWeights map[string]float64 `json:"feature_weights"`
	Residuals      []float64          `json:"residuals"`
	LastTimestamp  time.Time          `json:"last_timestamp"`
	LastObserved   float64            `json:"last_observed"`
}

func NewAlertVolumeForecaster(version string) *AlertVolumeForecaster {
	if version == "" {
		version = "alert-volume-v1"
	}
	return &AlertVolumeForecaster{
		ModelVersion: version,
		WeekdayFactor: map[int]float64{
			0: 0.90,
			1: 1.05,
			2: 1.10,
			3: 1.05,
			4: 1.00,
			5: 0.85,
			6: 0.75,
		},
		FeatureWeights: map[string]float64{
			"threat_activity":    0.30,
			"asset_onboarding":   0.15,
			"detection_changes":  0.25,
			"maintenance_window": -0.20,
			"holiday":            -0.10,
		},
	}
}

func (m *AlertVolumeForecaster) Train(samples []AlertVolumeSample) error {
	if len(samples) < 7 {
		return fmt.Errorf("at least 7 alert samples are required")
	}
	counts := make([]float64, 0, len(samples))
	weekdayBuckets := map[int][]float64{}
	for _, sample := range samples {
		counts = append(counts, sample.AlertCount)
		weekdayBuckets[int(sample.Timestamp.Weekday())] = append(weekdayBuckets[int(sample.Timestamp.Weekday())], sample.AlertCount)
		if sample.Timestamp.After(m.LastTimestamp) {
			m.LastTimestamp = sample.Timestamp
			m.LastObserved = sample.AlertCount
		}
	}
	m.BaseLevel = math.Max(mean(counts), 1)
	m.Trend = slope(counts)
	for day := 0; day < 7; day++ {
		if values := weekdayBuckets[day]; len(values) > 0 {
			m.WeekdayFactor[day] = clamp(mean(values)/m.BaseLevel, 0.50, 1.75)
		}
	}
	m.Residuals = m.Residuals[:0]
	for idx, sample := range samples {
		estimate := m.BaseLevel*m.WeekdayFactor[int(sample.Timestamp.Weekday())] + m.Trend*float64(idx)
		estimate += sample.ThreatActivity * m.FeatureWeights["threat_activity"]
		estimate += sample.AssetOnboarding * m.FeatureWeights["asset_onboarding"]
		estimate += sample.DetectionChanges * m.FeatureWeights["detection_changes"]
		estimate += sample.MaintenanceWindow * m.FeatureWeights["maintenance_window"]
		estimate += sample.Holiday * m.FeatureWeights["holiday"]
		m.Residuals = append(m.Residuals, sample.AlertCount-estimate)
	}
	return nil
}

func (m *AlertVolumeForecaster) Forecast(horizonDays int, future []AlertVolumeSample) (predictmodel.AlertVolumeForecast, map[string]float64) {
	if horizonDays <= 0 {
		horizonDays = 7
	}
	points := make([]predictmodel.ForecastPoint, 0, horizonDays)
	featureTotals := map[string]float64{
		"base_level":         m.BaseLevel,
		"trend":              m.Trend,
		"threat_activity":    0,
		"asset_onboarding":   0,
		"detection_changes":  0,
		"maintenance_window": 0,
		"holiday":            0,
	}
	spread := math.Max(percentile(absAll(m.Residuals), 0.90), 1)
	start := m.LastTimestamp
	if start.IsZero() {
		start = time.Now().UTC()
	}
	for day := 0; day < horizonDays; day++ {
		pointTime := start.AddDate(0, 0, day+1)
		sample := AlertVolumeSample{Timestamp: pointTime}
		if day < len(future) {
			sample = future[day]
			if sample.Timestamp.IsZero() {
				sample.Timestamp = pointTime
			}
		}
		estimate := m.BaseLevel*m.WeekdayFactor[int(sample.Timestamp.Weekday())] + m.Trend*float64(day+1)
		estimate += sample.ThreatActivity * m.FeatureWeights["threat_activity"]
		estimate += sample.AssetOnboarding * m.FeatureWeights["asset_onboarding"]
		estimate += sample.DetectionChanges * m.FeatureWeights["detection_changes"]
		estimate += sample.MaintenanceWindow * m.FeatureWeights["maintenance_window"]
		estimate += sample.Holiday * m.FeatureWeights["holiday"]
		if estimate < 0 {
			estimate = 0
		}
		featureTotals["threat_activity"] += sample.ThreatActivity * m.FeatureWeights["threat_activity"]
		featureTotals["asset_onboarding"] += sample.AssetOnboarding * m.FeatureWeights["asset_onboarding"]
		featureTotals["detection_changes"] += sample.DetectionChanges * m.FeatureWeights["detection_changes"]
		featureTotals["maintenance_window"] += sample.MaintenanceWindow * m.FeatureWeights["maintenance_window"]
		featureTotals["holiday"] += sample.Holiday * m.FeatureWeights["holiday"]
		points = append(points, predictmodel.ForecastPoint{
			Timestamp: sample.Timestamp,
			Value:     math.Round(estimate*100) / 100,
			Bounds: predictmodel.ConfidenceInterval{
				P10: math.Max(0, estimate-spread),
				P50: estimate,
				P90: estimate + spread,
			},
		})
	}
	return predictmodel.AlertVolumeForecast{
		HorizonDays: horizonDays,
		Points:      points,
		AnomalyFlag: m.LastObserved > 0 && (m.LastObserved < points[0].Bounds.P10 || m.LastObserved > points[0].Bounds.P90),
		Summary: map[string]any{
			"predicted_total":      sumForecast(points),
			"average_daily_alerts": averageForecast(points),
		},
	}, featureTotals
}

func (m *AlertVolumeForecaster) Serialize() ([]byte, error) {
	return json.Marshal(m)
}

func (m *AlertVolumeForecaster) Deserialize(payload []byte) error {
	return json.Unmarshal(payload, m)
}

func absAll(values []float64) []float64 {
	out := make([]float64, 0, len(values))
	for _, value := range values {
		out = append(out, math.Abs(value))
	}
	return out
}

func sumForecast(points []predictmodel.ForecastPoint) float64 {
	total := 0.0
	for _, point := range points {
		total += point.Value
	}
	return math.Round(total*100) / 100
}

func averageForecast(points []predictmodel.ForecastPoint) float64 {
	if len(points) == 0 {
		return 0
	}
	return math.Round((sumForecast(points)/float64(len(points)))*100) / 100
}
