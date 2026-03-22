package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
	predictdto "github.com/clario360/platform/internal/cyber/vciso/predict/dto"
	predictengine "github.com/clario360/platform/internal/cyber/vciso/predict/engine"
	predictmodel "github.com/clario360/platform/internal/cyber/vciso/predict/model"
	predictmodels "github.com/clario360/platform/internal/cyber/vciso/predict/models"
)

// TestAnalyticsIntegration exercises the analytics handler through a real
// ForecastEngine constructed without a database. Models are pre-trained with
// synthetic data so the ensure* checks pass. FeatureStore has a nil DB, so
// endpoints that call the store after ensure (AlertForecast, Campaigns) return
// graceful 500 errors. Endpoints that only need the model after ensure
// (ThreatForecast, TechniqueTrends) succeed end-to-end. This verifies:
//   - handler → engine → model → explainer → calibrator → response pipeline
//   - no panics on any endpoint
//   - well-formed JSON for successful endpoints
//   - graceful error handling for DB-dependent endpoints
func TestAnalyticsIntegration(t *testing.T) {
	t.Parallel()

	// Build a real ForecastEngine with nil DB dependencies.
	artifactDir := filepath.Join(os.TempDir(), "clario-predict-test-analytics")
	defer os.RemoveAll(artifactDir)

	logger := zerolog.Nop()
	reg := prometheus.NewRegistry()
	metrics := predictengine.NewMetrics(reg)
	store := predictengine.NewFeatureStore(nil, nil, nil, nil, nil, nil, logger)
	registry := predictengine.NewModelRegistry(nil, artifactDir, logger)
	if err := registry.EnsureDefaults(context.Background()); err != nil {
		t.Fatalf("EnsureDefaults: %v", err)
	}

	// Pre-train a TechniqueTrendAnalyzer with synthetic data so
	// ensureTechniqueModel() passes (requires len(States) > 0).
	techniqueModel := predictmodels.NewTechniqueTrendAnalyzer("technique-trend-v1-test")
	techniqueSamples := []predictmodels.TechniqueTrendSample{
		{TechniqueID: "T1566", TechniqueName: "Phishing", Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), InternalCount: 1, IndustryCount: 2},
		{TechniqueID: "T1566", TechniqueName: "Phishing", Timestamp: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), InternalCount: 3, IndustryCount: 3},
		{TechniqueID: "T1566", TechniqueName: "Phishing", Timestamp: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC), InternalCount: 5, IndustryCount: 4},
		{TechniqueID: "T1059", TechniqueName: "Command and Scripting", Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), InternalCount: 2, IndustryCount: 1},
		{TechniqueID: "T1059", TechniqueName: "Command and Scripting", Timestamp: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), InternalCount: 2, IndustryCount: 2},
		{TechniqueID: "T1059", TechniqueName: "Command and Scripting", Timestamp: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC), InternalCount: 3, IndustryCount: 2},
	}
	if err := techniqueModel.Train(techniqueSamples); err != nil {
		t.Fatalf("train technique model: %v", err)
	}
	if _, err := registry.Activate(context.Background(),
		predictmodel.PredictionTypeAttackTechniqueTrend,
		predictmodel.FrameworkRegression,
		techniqueModel,
		predictmodel.BacktestMetrics{Accuracy: 0.85, Count: len(techniqueSamples)},
		len(techniqueModel.Weights),
		len(techniqueSamples),
		time.Second,
	); err != nil {
		t.Fatalf("activate technique model: %v", err)
	}

	engine := predictengine.NewForecastEngine(
		store,
		registry,
		nil, // PredictionRepository — persistence skipped
		nil, // SHAPExplainer — auto-created
		nil, // PredictionNarrator — auto-created
		nil, // ConfidenceCalibrator — auto-created
		nil, // Backtester — auto-created
		nil, // DriftDetector — auto-created
		nil, // PredictionLogger — AI governance logging skipped
		metrics,
		logger,
	)

	// Use a real ThreatStats mock since the landscape endpoint doesn't use the engine.
	stats := &mockThreatStats{
		statsFn: func(_ context.Context, _ uuid.UUID) (*model.ThreatStats, error) {
			return &model.ThreatStats{
				Total: 10, Active: 3, IndicatorsTotal: 50,
				ByType:     []model.NamedCount{{Name: "apt", Count: 3}, {Name: "malware", Count: 7}},
				BySeverity: []model.NamedCount{{Name: "critical", Count: 2}, {Name: "high", Count: 8}},
			}, nil
		},
	}

	h := NewAnalyticsHandler(engine, stats, logger)
	ctx := analyticsCtx()

	// ThreatForecast uses PredictTechniqueTrends which only calls model.Predict()
	// after ensure — no store dependency. Full pipeline test.
	t.Run("ThreatForecast", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/threat-forecast?horizon_days=7", nil)
		r = r.WithContext(ctx)
		w := httptest.NewRecorder()
		h.ThreatForecast(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var env map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		var resp predictdto.TechniqueTrendResponse
		if err := json.Unmarshal(env["data"], &resp); err != nil {
			t.Fatalf("unmarshal data: %v", err)
		}
		if resp.Items == nil {
			t.Error("Items should be [] not nil after Normalize")
		}
		if resp.TopFeatures == nil {
			t.Error("TopFeatures should be [] not nil after Normalize")
		}
		if resp.PredictionType != predictmodel.PredictionTypeAttackTechniqueTrend {
			t.Errorf("expected prediction_type %q, got %q", predictmodel.PredictionTypeAttackTechniqueTrend, resp.PredictionType)
		}
		if resp.ConfidenceScore <= 0 {
			t.Error("expected non-zero confidence score from real engine pipeline")
		}
		if len(resp.Items) == 0 {
			t.Error("expected at least 1 technique trend item from pre-trained model")
		}
		// Verify the real explainer produced an explanation
		if resp.ExplanationText == "" {
			t.Error("expected non-empty explanation from real PredictionNarrator")
		}
	})

	// AlertForecast calls store.AlertVolumeSamples() after ensure — FeatureStore
	// returns an error with nil DB. Verify graceful 500 (no panic).
	t.Run("AlertForecast_GracefulError", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/alert-forecast?horizon_days=7", nil)
		r = r.WithContext(ctx)
		w := httptest.NewRecorder()
		h.AlertForecast(w, r)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 (nil DB), got %d: %s", w.Code, w.Body.String())
		}
		// Verify the error response is valid JSON
		var errResp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
			t.Fatalf("error response should be valid JSON: %v", err)
		}
		if errResp["code"] != "INTERNAL_ERROR" {
			t.Errorf("expected error code INTERNAL_ERROR, got %v", errResp["code"])
		}
	})

	// TechniqueTrends uses PredictTechniqueTrends (same as ThreatForecast).
	// Full pipeline test with a different horizon.
	t.Run("TechniqueTrends", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/technique-trends?horizon_days=30", nil)
		r = r.WithContext(ctx)
		w := httptest.NewRecorder()
		h.TechniqueTrends(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var env map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		var resp predictdto.TechniqueTrendResponse
		if err := json.Unmarshal(env["data"], &resp); err != nil {
			t.Fatalf("unmarshal data: %v", err)
		}
		// Should have 2 techniques from our training data
		if len(resp.Items) < 2 {
			t.Errorf("expected at least 2 technique items, got %d", len(resp.Items))
		}
	})

	// Campaigns calls store.CampaignSamples() after ensure — FeatureStore
	// returns an error with nil DB. Verify graceful 500 (no panic).
	t.Run("Campaigns_GracefulError", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/campaigns?lookback_days=30", nil)
		r = r.WithContext(ctx)
		w := httptest.NewRecorder()
		h.Campaigns(w, r)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 (nil DB), got %d: %s", w.Code, w.Body.String())
		}
		var errResp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
			t.Fatalf("error response should be valid JSON: %v", err)
		}
		if errResp["code"] != "INTERNAL_ERROR" {
			t.Errorf("expected error code INTERNAL_ERROR, got %v", errResp["code"])
		}
	})

	// Landscape doesn't use the engine — uses mock stats.
	t.Run("Landscape", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/landscape", nil)
		r = r.WithContext(ctx)
		w := httptest.NewRecorder()
		h.Landscape(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var env map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		var landscape model.ThreatLandscape
		if err := json.Unmarshal(env["data"], &landscape); err != nil {
			t.Fatalf("unmarshal data: %v", err)
		}
		if landscape.TotalThreats != 10 {
			t.Errorf("expected total 10, got %d", landscape.TotalThreats)
		}
		if landscape.TopThreatType != "apt" {
			t.Errorf("expected top type apt, got %s", landscape.TopThreatType)
		}
	})
}
