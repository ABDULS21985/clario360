package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ValidationDatasetType string

const (
	ValidationDatasetHistorical ValidationDatasetType = "historical"
	ValidationDatasetCustom     ValidationDatasetType = "custom"
	ValidationDatasetLiveReplay ValidationDatasetType = "live_replay"
)

type ValidationRecommendation string

const (
	ValidationRecommendationPromote     ValidationRecommendation = "promote"
	ValidationRecommendationKeepTesting ValidationRecommendation = "keep_testing"
	ValidationRecommendationReject      ValidationRecommendation = "reject"
)

type ValidationLabel string

const (
	ValidationLabelThreat ValidationLabel = "threat"
	ValidationLabelBenign ValidationLabel = "benign"
)

type ROCPoint struct {
	Threshold float64 `json:"threshold"`
	FPR       float64 `json:"fpr"`
	TPR       float64 `json:"tpr"`
}

type MetricsSummary struct {
	DatasetSize       int     `json:"dataset_size"`
	PositiveCount     int     `json:"positive_count"`
	NegativeCount     int     `json:"negative_count"`
	TruePositives     int     `json:"true_positives"`
	FalsePositives    int     `json:"false_positives"`
	TrueNegatives     int     `json:"true_negatives"`
	FalseNegatives    int     `json:"false_negatives"`
	Precision         float64 `json:"precision"`
	Recall            float64 `json:"recall"`
	F1Score           float64 `json:"f1_score"`
	FalsePositiveRate float64 `json:"false_positive_rate"`
	FalseNegativeRate float64 `json:"false_negative_rate,omitempty"`
	Accuracy          float64 `json:"accuracy"`
	AUC               float64 `json:"auc,omitempty"`
}

type PredictionSample struct {
	PredictionID    *uuid.UUID      `json:"prediction_id,omitempty"`
	InputHash       string          `json:"input_hash"`
	InputSummary    json.RawMessage `json:"input_summary"`
	PredictedOutput json.RawMessage `json:"predicted_output"`
	PredictedLabel  ValidationLabel `json:"predicted_label"`
	ExpectedLabel   ValidationLabel `json:"expected_label"`
	Confidence      float64         `json:"confidence"`
	Severity        string          `json:"severity,omitempty"`
	RuleType        string          `json:"rule_type,omitempty"`
	Explanation     string          `json:"explanation"`
}

type ValidationResult struct {
	ID                   uuid.UUID                 `json:"id"`
	TenantID             uuid.UUID                 `json:"tenant_id,omitempty"`
	ModelID              uuid.UUID                 `json:"model_id,omitempty"`
	VersionID            uuid.UUID                 `json:"version_id"`
	DatasetType          ValidationDatasetType     `json:"dataset_type"`
	DatasetSize          int                       `json:"dataset_size"`
	PositiveCount        int                       `json:"positive_count"`
	NegativeCount        int                       `json:"negative_count"`
	TruePositives        int                       `json:"true_positives"`
	FalsePositives       int                       `json:"false_positives"`
	TrueNegatives        int                       `json:"true_negatives"`
	FalseNegatives       int                       `json:"false_negatives"`
	Precision            float64                   `json:"precision"`
	Recall               float64                   `json:"recall"`
	F1Score              float64                   `json:"f1_score"`
	FalsePositiveRate    float64                   `json:"false_positive_rate"`
	Accuracy             float64                   `json:"accuracy"`
	AUC                  float64                   `json:"auc"`
	ROCCurve             []ROCPoint                `json:"roc_curve"`
	ProductionMetrics    *MetricsSummary           `json:"production_metrics,omitempty"`
	Deltas               map[string]float64        `json:"deltas,omitempty"`
	BySeverity           map[string]MetricsSummary `json:"by_severity"`
	ByRuleType           map[string]MetricsSummary `json:"by_rule_type,omitempty"`
	FPSamples            []PredictionSample        `json:"false_positive_samples"`
	FNSamples            []PredictionSample        `json:"false_negative_samples"`
	Recommendation       ValidationRecommendation  `json:"recommendation"`
	RecommendationReason string                    `json:"recommendation_reason"`
	Warnings             []string                  `json:"warnings"`
	ValidatedAt          time.Time                 `json:"validated_at"`
	DurationMs           int                       `json:"duration_ms"`
}
