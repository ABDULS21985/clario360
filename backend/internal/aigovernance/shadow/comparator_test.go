package shadow

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
)

func TestCompareFullAgreement(t *testing.T) {
	confidence := 0.92
	production := &aigovmodel.PredictionLog{
		ID:         uuid.New(),
		InputHash:  "abc",
		UseCase:    "risk_scoring",
		Prediction: json.RawMessage(`{"score":80}`),
		Confidence: &confidence,
	}
	shadow := &aigovmodel.PredictionLog{
		ID:         uuid.New(),
		InputHash:  "abc",
		UseCase:    "risk_scoring",
		Prediction: json.RawMessage(`{"score":80}`),
		Confidence: &confidence,
	}

	agree, divergence := ComparePredictionLogs(production, shadow)
	if !agree {
		t.Fatal("expected full agreement")
	}
	if divergence != nil {
		t.Fatalf("expected no divergence, got %#v", divergence)
	}
}

func TestComparePartialAgreement(t *testing.T) {
	left := 0.6
	right := 0.9
	production := &aigovmodel.PredictionLog{
		ID:         uuid.New(),
		InputHash:  "abc",
		UseCase:    "anomaly_detection",
		Prediction: json.RawMessage(`{"anomaly":false}`),
		Confidence: &left,
	}
	shadow := &aigovmodel.PredictionLog{
		ID:         uuid.New(),
		InputHash:  "abc",
		UseCase:    "anomaly_detection",
		Prediction: json.RawMessage(`{"anomaly":true}`),
		Confidence: &right,
	}

	agree, divergence := ComparePredictionLogs(production, shadow)
	if agree {
		t.Fatal("expected disagreement")
	}
	if divergence == nil {
		t.Fatal("expected divergence details")
	}
}

func TestRecommendPromote(t *testing.T) {
	recommendation, _, _ := Recommend(
		0.97,
		MetricsSummary{Accuracy: 0.91, AvgConfidence: 0.8, AvgLatencyMS: 15},
		MetricsSummary{Accuracy: 0.95, AvgConfidence: 0.86, AvgLatencyMS: 12},
	)
	if recommendation != aigovmodel.ShadowRecommendationPromote {
		t.Fatalf("recommendation = %s, want promote", recommendation)
	}
}

func TestRecommendReject(t *testing.T) {
	recommendation, _, _ := Recommend(
		0.72,
		MetricsSummary{Accuracy: 0.91, AvgConfidence: 0.8, AvgLatencyMS: 15},
		MetricsSummary{Accuracy: 0.95, AvgConfidence: 0.86, AvgLatencyMS: 12},
	)
	if recommendation != aigovmodel.ShadowRecommendationReject {
		t.Fatalf("recommendation = %s, want reject", recommendation)
	}
}

func TestRecommendKeepShadow(t *testing.T) {
	recommendation, _, _ := Recommend(
		0.92,
		MetricsSummary{Accuracy: 0.91, AvgConfidence: 0.8, AvgLatencyMS: 15},
		MetricsSummary{Accuracy: 0.9, AvgConfidence: 0.82, AvgLatencyMS: 15},
	)
	if recommendation != aigovmodel.ShadowRecommendationKeepShadow {
		t.Fatalf("recommendation = %s, want keep_shadow", recommendation)
	}
}
