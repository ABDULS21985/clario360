package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	aigovdto "github.com/clario360/platform/internal/aigovernance/dto"
	aigovmetrics "github.com/clario360/platform/internal/aigovernance/metrics"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type fakeValidationRegistryRepo struct {
	version           *aigovmodel.ModelVersion
	productionVersion *aigovmodel.ModelVersion
}

func (f *fakeValidationRegistryRepo) GetVersion(_ context.Context, _, _, _ uuid.UUID) (*aigovmodel.ModelVersion, error) {
	if f.version == nil {
		return nil, errors.New("missing version")
	}
	return f.version, nil
}

func (f *fakeValidationRegistryRepo) GetCurrentProductionVersion(_ context.Context, _, _ uuid.UUID) (*aigovmodel.ModelVersion, error) {
	if f.productionVersion == nil {
		return nil, errors.New("missing production version")
	}
	return f.productionVersion, nil
}

func (f *fakeValidationRegistryRepo) UpdateVersionValidationMetrics(context.Context, uuid.UUID, uuid.UUID, json.RawMessage, float64, float64, float64) error {
	return nil
}

type fakeValidationPredictionRepo struct {
	windowLogs []aigovmodel.PredictionLog
	customLogs []aigovmodel.PredictionLog
}

func (f *fakeValidationPredictionRepo) ListByVersionAndWindow(context.Context, uuid.UUID, uuid.UUID, time.Time, time.Time, *bool) ([]aigovmodel.PredictionLog, error) {
	return f.windowLogs, nil
}

func (f *fakeValidationPredictionRepo) ListLatestByVersionAndInputHashes(context.Context, uuid.UUID, uuid.UUID, []string) ([]aigovmodel.PredictionLog, error) {
	return f.customLogs, nil
}

type fakeValidationResultRepo struct {
	created *aigovmodel.ValidationResult
	latest  map[uuid.UUID]*aigovmodel.ValidationResult
}

func (f *fakeValidationResultRepo) Create(_ context.Context, item *aigovmodel.ValidationResult) error {
	f.created = item
	return nil
}

func (f *fakeValidationResultRepo) LatestByVersion(_ context.Context, _, versionID uuid.UUID) (*aigovmodel.ValidationResult, error) {
	if item, ok := f.latest[versionID]; ok {
		return item, nil
	}
	return nil, errors.New("not found")
}

func (f *fakeValidationResultRepo) HistoryByVersion(context.Context, uuid.UUID, uuid.UUID, int) ([]aigovmodel.ValidationResult, error) {
	return nil, nil
}

func TestValidation_InsufficientData(t *testing.T) {
	version := fakeVersion()
	service := NewValidationService(
		&fakeValidationRegistryRepo{version: version},
		&fakeValidationPredictionRepo{windowLogs: repeatedLogs(30)},
		&fakeValidationResultRepo{latest: map[uuid.UUID]*aigovmodel.ValidationResult{}},
		nil,
		nil,
		nil,
		zerolog.Nop(),
	)

	_, err := service.Validate(context.Background(), version.TenantID, version.ModelID, version.ID, aigovdto.ValidateRequest{
		DatasetType: aigovmodel.ValidationDatasetHistorical,
		TimeRange:   "30d",
	})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "need at least 50") {
		t.Fatalf("expected insufficient data error, got %v", err)
	}
}

func TestValidation_Warning(t *testing.T) {
	version := fakeVersion()
	results := &fakeValidationResultRepo{latest: map[uuid.UUID]*aigovmodel.ValidationResult{}}
	service := NewValidationService(
		&fakeValidationRegistryRepo{version: version},
		&fakeValidationPredictionRepo{windowLogs: repeatedLogs(100)},
		results,
		nil,
		nil,
		nil,
		zerolog.Nop(),
	)

	result, err := service.Validate(context.Background(), version.TenantID, version.ModelID, version.ID, aigovdto.ValidateRequest{
		DatasetType: aigovmodel.ValidationDatasetHistorical,
		TimeRange:   "30d",
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if result.DatasetSize != 100 {
		t.Fatalf("dataset_size = %d, want 100", result.DatasetSize)
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], "< 200") {
		t.Fatalf("warnings = %v, want statistical warning", result.Warnings)
	}
	if results.created == nil {
		t.Fatal("expected validation result to be persisted")
	}
}

func TestRecommendation_Promote(t *testing.T) {
	recommendation, reason := recommend(aigovmodel.MetricsSummary{
		Precision:         0.90,
		Recall:            0.85,
		FalsePositiveRate: 0.03,
		AUC:               0.95,
	})
	if recommendation != aigovmodel.ValidationRecommendationPromote {
		t.Fatalf("recommendation = %s, want promote", recommendation)
	}
	if !strings.Contains(reason, "thresholds") {
		t.Fatalf("reason = %q", reason)
	}
}

func TestRecommendation_Reject(t *testing.T) {
	recommendation, reason := recommend(aigovmodel.MetricsSummary{
		Precision:         0.60,
		Recall:            0.85,
		FalsePositiveRate: 0.03,
		AUC:               0.95,
	})
	if recommendation != aigovmodel.ValidationRecommendationReject {
		t.Fatalf("recommendation = %s, want reject", recommendation)
	}
	if !strings.Contains(reason, "precision") {
		t.Fatalf("reason = %q", reason)
	}
}

func TestComparisonDelta(t *testing.T) {
	candidate := aigovmodel.MetricsSummary{Precision: 0.92}
	production := aigovmodel.MetricsSummary{Precision: 0.88}
	deltas := aigovmetrics.CompareMetrics(candidate, production)
	if got := deltas["precision"]; math.Abs(got-0.04) > 0.0001 {
		t.Fatalf("precision delta = %.2f, want 0.04", got)
	}
}

func fakeVersion() *aigovmodel.ModelVersion {
	return &aigovmodel.ModelVersion{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		ModelID:         uuid.New(),
		ModelSlug:       "cyber-sigma-evaluator",
		VersionNumber:   2,
		TrainingMetrics: json.RawMessage(`{}`),
		CreatedAt:       time.Now().UTC().Add(-24 * time.Hour),
	}
}

func repeatedLogs(count int) []aigovmodel.PredictionLog {
	logs := make([]aigovmodel.PredictionLog, 0, count)
	for idx := 0; idx < count; idx++ {
		predictedPositive := idx%2 == 0
		feedbackCorrect := true
		prediction := map[string]any{
			"matched":   predictedPositive,
			"severity":  "high",
			"rule_type": "sigma",
		}
		payload, _ := json.Marshal(prediction)
		inputSummary, _ := json.Marshal(map[string]any{
			"severity":  "high",
			"rule_type": "sigma",
		})
		confidence := 0.9
		if !predictedPositive {
			confidence = 0.2
		}
		logs = append(logs, aigovmodel.PredictionLog{
			ID:                      uuid.New(),
			InputHash:               uuid.NewString(),
			InputSummary:            inputSummary,
			Prediction:              payload,
			Confidence:              &confidence,
			ExplanationText:         "unit test explanation",
			FeedbackCorrect:         &feedbackCorrect,
			FeedbackCorrectedOutput: json.RawMessage(`null`),
			CreatedAt:               time.Now().UTC(),
		})
	}
	return logs
}
