package dto

import (
	"time"

	"github.com/clario360/platform/internal/cyber/vciso/predict/model"
)

type ForecastQuery struct {
	ForecastType string `json:"forecast_type"`
	TimeHorizon  string `json:"time_horizon"`
}

type AssetRiskQuery struct {
	Limit     int    `json:"limit"`
	AssetType string `json:"asset_type"`
}

type VulnerabilityPriorityQuery struct {
	Limit          int     `json:"limit"`
	MinProbability float64 `json:"min_probability"`
}

type InsiderThreatQuery struct {
	TimeHorizon string `json:"time_horizon"`
	Threshold   int    `json:"threshold"`
}

type GenericPredictionResponse struct {
	PredictionType     model.PredictionType        `json:"prediction_type"`
	ModelVersion       string                      `json:"model_version"`
	GeneratedAt        time.Time                   `json:"generated_at"`
	ConfidenceScore    float64                     `json:"confidence_score"`
	ConfidenceInterval model.ConfidenceInterval    `json:"confidence_interval"`
	TopFeatures        []model.FeatureContribution `json:"top_features"`
	ExplanationText    string                      `json:"explanation_text"`
	VerificationSteps  []string                    `json:"verification_steps"`
}

// Normalize ensures all slice fields are non-nil so they serialize as [] not null.
func (r *GenericPredictionResponse) Normalize() {
	if r.TopFeatures == nil {
		r.TopFeatures = []model.FeatureContribution{}
	}
	if r.VerificationSteps == nil {
		r.VerificationSteps = []string{}
	}
}

type ForecastResponse struct {
	GenericPredictionResponse
	Forecast model.AlertVolumeForecast `json:"forecast"`
}

// Normalize ensures all slice fields are non-nil so they serialize as [] not null.
func (r *ForecastResponse) Normalize() {
	r.GenericPredictionResponse.Normalize()
	r.Forecast.Normalize()
}

type AssetRiskResponse struct {
	GenericPredictionResponse
	Items []model.AssetRiskItem `json:"items"`
}

type VulnerabilityPriorityResponse struct {
	GenericPredictionResponse
	Items []model.VulnerabilityPriorityItem `json:"items"`
}

type TechniqueTrendResponse struct {
	GenericPredictionResponse
	Items []model.TechniqueTrendItem `json:"items"`
}

// Normalize ensures all slice fields are non-nil so they serialize as [] not null.
func (r *TechniqueTrendResponse) Normalize() {
	r.GenericPredictionResponse.Normalize()
	if r.Items == nil {
		r.Items = []model.TechniqueTrendItem{}
	}
	for i := range r.Items {
		r.Items[i].Normalize()
	}
}

type InsiderThreatResponse struct {
	GenericPredictionResponse
	Items []model.InsiderThreatTrajectoryItem `json:"items"`
}

type CampaignResponse struct {
	GenericPredictionResponse
	Items []model.CampaignCluster `json:"items"`
}

// Normalize ensures all slice fields are non-nil so they serialize as [] not null.
func (r *CampaignResponse) Normalize() {
	r.GenericPredictionResponse.Normalize()
	if r.Items == nil {
		r.Items = []model.CampaignCluster{}
	}
	for i := range r.Items {
		r.Items[i].Normalize()
	}
}

type AccuracyResponse struct {
	GeneratedAt time.Time               `json:"generated_at"`
	Dashboard   model.AccuracyDashboard `json:"dashboard"`
}

type RetrainResponse struct {
	ModelType   model.PredictionType  `json:"model_type"`
	Version     string                `json:"version"`
	Status      string                `json:"status"`
	TriggeredAt time.Time             `json:"triggered_at"`
	Trigger     string                `json:"trigger"`
	DriftScore  *float64              `json:"drift_score,omitempty"`
	Backtest    model.BacktestMetrics `json:"backtest"`
}
