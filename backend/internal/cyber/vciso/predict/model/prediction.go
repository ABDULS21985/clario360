package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type PredictionType string

const (
	PredictionTypeAlertVolumeForecast     PredictionType = "alert_volume_forecast"
	PredictionTypeAssetRisk               PredictionType = "asset_risk_prediction"
	PredictionTypeVulnerabilityExploit    PredictionType = "vulnerability_exploit_prediction"
	PredictionTypeAttackTechniqueTrend    PredictionType = "attack_technique_trend"
	PredictionTypeInsiderThreatTrajectory PredictionType = "insider_threat_trajectory"
	PredictionTypeCampaignDetection       PredictionType = "campaign_detection"
)

type ModelFramework string

const (
	FrameworkStatistical ModelFramework = "statistical"
	FrameworkXGBoost     ModelFramework = "xgboost"
	FrameworkGBM         ModelFramework = "gbm"
	FrameworkProphet     ModelFramework = "prophet"
	FrameworkRegression  ModelFramework = "regression"
	FrameworkLSTM        ModelFramework = "lstm"
	FrameworkDBSCAN      ModelFramework = "dbscan"
)

type PredictionModelStatus string

const (
	PredictionModelStatusTraining   PredictionModelStatus = "training"
	PredictionModelStatusValidating PredictionModelStatus = "validating"
	PredictionModelStatusActive     PredictionModelStatus = "active"
	PredictionModelStatusDeprecated PredictionModelStatus = "deprecated"
	PredictionModelStatusFailed     PredictionModelStatus = "failed"
)

type ConfidenceInterval struct {
	P10 float64 `json:"p10"`
	P50 float64 `json:"p50"`
	P90 float64 `json:"p90"`
}

type FeatureContribution struct {
	Feature   string  `json:"feature"`
	SHAPValue float64 `json:"shap_value"`
	Direction string  `json:"direction"`
	Value     any     `json:"value,omitempty"`
}

type StoredPrediction struct {
	ID                 uuid.UUID             `json:"id" db:"id"`
	TenantID           uuid.UUID             `json:"tenant_id" db:"tenant_id"`
	PredictionType     PredictionType        `json:"prediction_type" db:"prediction_type"`
	ModelVersion       string                `json:"model_version" db:"model_version"`
	PredictionJSON     json.RawMessage       `json:"prediction_json" db:"prediction_json"`
	ConfidenceScore    float64               `json:"confidence_score" db:"confidence_score"`
	ConfidenceInterval ConfidenceInterval    `json:"confidence_interval" db:"-"`
	TopFeatures        []FeatureContribution `json:"top_features" db:"-"`
	ExplanationText    string                `json:"explanation_text" db:"explanation_text"`
	TargetEntityType   *string               `json:"target_entity_type,omitempty" db:"target_entity_type"`
	TargetEntityID     *string               `json:"target_entity_id,omitempty" db:"target_entity_id"`
	ForecastStart      time.Time             `json:"forecast_start" db:"forecast_start"`
	ForecastEnd        time.Time             `json:"forecast_end" db:"forecast_end"`
	OutcomeObserved    *bool                 `json:"outcome_observed,omitempty" db:"outcome_observed"`
	OutcomeValue       json.RawMessage       `json:"outcome_value,omitempty" db:"outcome_value"`
	AccuracyScore      *float64              `json:"accuracy_score,omitempty" db:"accuracy_score"`
	PredictionLogID    *uuid.UUID            `json:"prediction_log_id,omitempty" db:"prediction_log_id"`
	CreatedAt          time.Time             `json:"created_at" db:"created_at"`
	EvaluatedAt        *time.Time            `json:"evaluated_at,omitempty" db:"evaluated_at"`
}

type PredictionModel struct {
	ID                      uuid.UUID             `json:"id" db:"id"`
	ModelType               PredictionType        `json:"model_type" db:"model_type"`
	Version                 string                `json:"version" db:"version"`
	ModelArtifactPath       string                `json:"model_artifact_path" db:"model_artifact_path"`
	ModelFramework          ModelFramework        `json:"model_framework" db:"model_framework"`
	BacktestAccuracy        *float64              `json:"backtest_accuracy,omitempty" db:"backtest_accuracy"`
	BacktestPrecision       *float64              `json:"backtest_precision,omitempty" db:"backtest_precision"`
	BacktestRecall          *float64              `json:"backtest_recall,omitempty" db:"backtest_recall"`
	BacktestF1              *float64              `json:"backtest_f1,omitempty" db:"backtest_f1"`
	BacktestMAPE            *float64              `json:"backtest_mape,omitempty" db:"backtest_mape"`
	FeatureCount            int                   `json:"feature_count" db:"feature_count"`
	TrainingSamples         int                   `json:"training_samples" db:"training_samples"`
	TrainingDurationSeconds int                   `json:"training_duration_seconds" db:"training_duration_seconds"`
	Status                  PredictionModelStatus `json:"status" db:"status"`
	Active                  bool                  `json:"active" db:"active"`
	LastDriftCheck          *time.Time            `json:"last_drift_check,omitempty" db:"last_drift_check"`
	DriftScore              *float64              `json:"drift_score,omitempty" db:"drift_score"`
	CreatedAt               time.Time             `json:"created_at" db:"created_at"`
	ActivatedAt             *time.Time            `json:"activated_at,omitempty" db:"activated_at"`
	DeprecatedAt            *time.Time            `json:"deprecated_at,omitempty" db:"deprecated_at"`
}

type ForecastPoint struct {
	Timestamp time.Time          `json:"timestamp"`
	Value     float64            `json:"value"`
	Bounds    ConfidenceInterval `json:"bounds"`
}

type AlertVolumeForecast struct {
	HorizonDays int             `json:"horizon_days"`
	Points      []ForecastPoint `json:"points"`
	AnomalyFlag bool            `json:"anomaly_flag"`
	Summary     map[string]any  `json:"summary,omitempty"`
}

type AssetRiskItem struct {
	AssetID     uuid.UUID             `json:"asset_id"`
	AssetName   string                `json:"asset_name"`
	AssetType   string                `json:"asset_type"`
	Probability float64               `json:"probability"`
	Confidence  ConfidenceInterval    `json:"confidence_interval"`
	CurrentRisk float64               `json:"current_risk"`
	TopFeatures []FeatureContribution `json:"top_features"`
}

type VulnerabilityPriorityItem struct {
	VulnerabilityID uuid.UUID             `json:"vulnerability_id"`
	AssetID         uuid.UUID             `json:"asset_id"`
	AssetName       string                `json:"asset_name"`
	CVEID           string                `json:"cve_id,omitempty"`
	Severity        string                `json:"severity"`
	Probability     float64               `json:"probability"`
	Confidence      ConfidenceInterval    `json:"confidence_interval"`
	TopFeatures     []FeatureContribution `json:"top_features"`
}

type TechniqueTrendItem struct {
	TechniqueID   string                `json:"technique_id"`
	TechniqueName string                `json:"technique_name,omitempty"`
	Trend         string                `json:"trend"`
	GrowthRate    float64               `json:"growth_rate"`
	Forecast      ConfidenceInterval    `json:"forecast"`
	TopFeatures   []FeatureContribution `json:"top_features"`
}

type InsiderThreatTrajectoryItem struct {
	EntityID         string                `json:"entity_id"`
	EntityName       string                `json:"entity_name,omitempty"`
	CurrentRisk      float64               `json:"current_risk"`
	ProjectedRisk    float64               `json:"projected_risk"`
	Confidence       ConfidenceInterval    `json:"confidence_interval"`
	DaysToThreshold  *int                  `json:"days_to_threshold,omitempty"`
	Accelerating     bool                  `json:"accelerating"`
	TopFeatures      []FeatureContribution `json:"top_features"`
	VerificationStep string                `json:"verification_step,omitempty"`
}

type CampaignCluster struct {
	ClusterID       string                `json:"cluster_id"`
	AlertIDs        []uuid.UUID           `json:"alert_ids"`
	AlertTitles     []string              `json:"alert_titles,omitempty"`
	StartAt         time.Time             `json:"start_at"`
	EndAt           time.Time             `json:"end_at"`
	Stage           string                `json:"stage"`
	MITRETechniques []string              `json:"mitre_techniques,omitempty"`
	SharedIOCs      []string              `json:"shared_iocs,omitempty"`
	Confidence      ConfidenceInterval    `json:"confidence_interval"`
	TopFeatures     []FeatureContribution `json:"top_features"`
}

type AccuracyDashboard struct {
	ByType         map[PredictionType]float64 `json:"by_type"`
	RecentFailures []StoredPrediction         `json:"recent_failures,omitempty"`
	DriftAlerts    []ModelDriftAlert          `json:"drift_alerts,omitempty"`
}

type ModelDriftAlert struct {
	ModelType      PredictionType `json:"model_type"`
	Severity       string         `json:"severity"`
	DriftScore     float64        `json:"drift_score"`
	ObservedAt     time.Time      `json:"observed_at"`
	Recommendation string         `json:"recommendation"`
}

type BacktestMetrics struct {
	Accuracy  float64 `json:"accuracy"`
	Precision float64 `json:"precision,omitempty"`
	Recall    float64 `json:"recall,omitempty"`
	F1        float64 `json:"f1,omitempty"`
	MAPE      float64 `json:"mape,omitempty"`
	Count     int     `json:"count"`
}
