package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/cyber/model"
	predictdto "github.com/clario360/platform/internal/cyber/vciso/predict/dto"
	predictmodel "github.com/clario360/platform/internal/cyber/vciso/predict/model"
)

// ---------------------------------------------------------------------------
// mocks
// ---------------------------------------------------------------------------

type mockForecastEngine struct {
	predictTrendsFn    func(ctx context.Context, tenantID uuid.UUID, horizonDays int) (*predictdto.TechniqueTrendResponse, error)
	forecastAlertsFn   func(ctx context.Context, tenantID uuid.UUID, horizonDays int) (*predictdto.ForecastResponse, error)
	detectCampaignsFn  func(ctx context.Context, tenantID uuid.UUID, lookbackDays int) (*predictdto.CampaignResponse, error)
}

func (m *mockForecastEngine) PredictTechniqueTrends(ctx context.Context, tenantID uuid.UUID, horizonDays int) (*predictdto.TechniqueTrendResponse, error) {
	if m.predictTrendsFn != nil {
		return m.predictTrendsFn(ctx, tenantID, horizonDays)
	}
	return &predictdto.TechniqueTrendResponse{}, nil
}

func (m *mockForecastEngine) ForecastAlertVolume(ctx context.Context, tenantID uuid.UUID, horizonDays int) (*predictdto.ForecastResponse, error) {
	if m.forecastAlertsFn != nil {
		return m.forecastAlertsFn(ctx, tenantID, horizonDays)
	}
	return &predictdto.ForecastResponse{}, nil
}

func (m *mockForecastEngine) DetectCampaigns(ctx context.Context, tenantID uuid.UUID, lookbackDays int) (*predictdto.CampaignResponse, error) {
	if m.detectCampaignsFn != nil {
		return m.detectCampaignsFn(ctx, tenantID, lookbackDays)
	}
	return &predictdto.CampaignResponse{}, nil
}

type mockThreatStats struct {
	statsFn func(ctx context.Context, tenantID uuid.UUID) (*model.ThreatStats, error)
}

func (m *mockThreatStats) Stats(ctx context.Context, tenantID uuid.UUID) (*model.ThreatStats, error) {
	if m.statsFn != nil {
		return m.statsFn(ctx, tenantID)
	}
	return &model.ThreatStats{}, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

var (
	analyticsTenantID = uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")
	analyticsUserID   = uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000001")
)

func analyticsCtx() context.Context {
	ctx := context.Background()
	ctx = auth.WithTenantID(ctx, analyticsTenantID.String())
	ctx = auth.WithUser(ctx, &auth.ContextUser{
		ID:       analyticsUserID.String(),
		TenantID: analyticsTenantID.String(),
		Email:    "test@clario.dev",
		Roles:    []string{"admin"},
	})
	return ctx
}

func newTestAnalyticsHandler(engine *mockForecastEngine, stats *mockThreatStats) *AnalyticsHandler {
	return &AnalyticsHandler{
		forecastEngine: engine,
		threatStats:    stats,
		logger:         zerolog.Nop(),
	}
}

func decodeEnvelope(t *testing.T, body []byte) map[string]json.RawMessage {
	t.Helper()
	var env map[string]json.RawMessage
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("failed to decode envelope: %v", err)
	}
	return env
}

// ---------------------------------------------------------------------------
// ThreatForecast
// ---------------------------------------------------------------------------

func TestThreatForecast_Success(t *testing.T) {
	engine := &mockForecastEngine{
		predictTrendsFn: func(_ context.Context, tenantID uuid.UUID, horizon int) (*predictdto.TechniqueTrendResponse, error) {
			if tenantID != analyticsTenantID {
				t.Errorf("unexpected tenant: %s", tenantID)
			}
			if horizon != 7 {
				t.Errorf("expected horizon 7, got %d", horizon)
			}
			return &predictdto.TechniqueTrendResponse{
				GenericPredictionResponse: predictdto.GenericPredictionResponse{
					PredictionType:  predictmodel.PredictionTypeAttackTechniqueTrend,
					ModelVersion:    "trend-v1",
					GeneratedAt:     time.Now(),
					ConfidenceScore: 0.85,
				},
				Items: []predictmodel.TechniqueTrendItem{
					{
						TechniqueID:   "T1566",
						TechniqueName: "Phishing",
						Trend:         "increasing",
						GrowthRate:    0.25,
						Forecast:      predictmodel.ConfidenceInterval{P10: 10, P50: 15, P90: 20},
					},
				},
			}, nil
		},
	}
	h := newTestAnalyticsHandler(engine, &mockThreatStats{})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/threat-forecast?horizon_days=7", nil)
	r = r.WithContext(analyticsCtx())
	w := httptest.NewRecorder()

	h.ThreatForecast(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	env := decodeEnvelope(t, w.Body.Bytes())
	var resp predictdto.TechniqueTrendResponse
	if err := json.Unmarshal(env["data"], &resp); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	if resp.Items[0].TechniqueID != "T1566" {
		t.Errorf("expected T1566, got %s", resp.Items[0].TechniqueID)
	}
}

func TestThreatForecast_DefaultHorizon(t *testing.T) {
	var capturedHorizon int
	engine := &mockForecastEngine{
		predictTrendsFn: func(_ context.Context, _ uuid.UUID, horizon int) (*predictdto.TechniqueTrendResponse, error) {
			capturedHorizon = horizon
			return &predictdto.TechniqueTrendResponse{}, nil
		},
	}
	h := newTestAnalyticsHandler(engine, &mockThreatStats{})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/threat-forecast", nil)
	r = r.WithContext(analyticsCtx())
	w := httptest.NewRecorder()

	h.ThreatForecast(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if capturedHorizon != 30 {
		t.Errorf("expected default horizon 30, got %d", capturedHorizon)
	}
}

func TestThreatForecast_InvalidHorizon(t *testing.T) {
	var capturedHorizon int
	engine := &mockForecastEngine{
		predictTrendsFn: func(_ context.Context, _ uuid.UUID, horizon int) (*predictdto.TechniqueTrendResponse, error) {
			capturedHorizon = horizon
			return &predictdto.TechniqueTrendResponse{}, nil
		},
	}
	h := newTestAnalyticsHandler(engine, &mockThreatStats{})

	// horizon_days=200 exceeds max (90) — should use default 30
	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/threat-forecast?horizon_days=200", nil)
	r = r.WithContext(analyticsCtx())
	w := httptest.NewRecorder()

	h.ThreatForecast(w, r)

	if capturedHorizon != 30 {
		t.Errorf("expected default 30 for out-of-range, got %d", capturedHorizon)
	}
}

func TestThreatForecast_EngineError(t *testing.T) {
	engine := &mockForecastEngine{
		predictTrendsFn: func(_ context.Context, _ uuid.UUID, _ int) (*predictdto.TechniqueTrendResponse, error) {
			return nil, errors.New("model not ready")
		},
	}
	h := newTestAnalyticsHandler(engine, &mockThreatStats{})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/threat-forecast", nil)
	r = r.WithContext(analyticsCtx())
	w := httptest.NewRecorder()

	h.ThreatForecast(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestThreatForecast_NoAuth(t *testing.T) {
	h := newTestAnalyticsHandler(&mockForecastEngine{}, &mockThreatStats{})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/threat-forecast", nil)
	w := httptest.NewRecorder()

	h.ThreatForecast(w, r)

	if w.Code == http.StatusOK {
		t.Fatal("expected non-200 for unauthenticated request")
	}
}

// ---------------------------------------------------------------------------
// AlertForecast
// ---------------------------------------------------------------------------

func TestAlertForecast_Success(t *testing.T) {
	now := time.Now()
	engine := &mockForecastEngine{
		forecastAlertsFn: func(_ context.Context, tenantID uuid.UUID, horizon int) (*predictdto.ForecastResponse, error) {
			if horizon != 30 {
				t.Errorf("expected horizon 30, got %d", horizon)
			}
			return &predictdto.ForecastResponse{
				GenericPredictionResponse: predictdto.GenericPredictionResponse{
					PredictionType:  predictmodel.PredictionTypeAlertVolumeForecast,
					ModelVersion:    "alert-volume-v1",
					GeneratedAt:     now,
					ConfidenceScore: 0.75,
				},
				Forecast: predictmodel.AlertVolumeForecast{
					HorizonDays: 30,
					Points: []predictmodel.ForecastPoint{
						{Timestamp: now, Value: 145.3, Bounds: predictmodel.ConfidenceInterval{P10: 130, P50: 145, P90: 160}},
						{Timestamp: now.Add(24 * time.Hour), Value: 150.0, Bounds: predictmodel.ConfidenceInterval{P10: 135, P50: 150, P90: 165}},
					},
					AnomalyFlag: false,
				},
			}, nil
		},
	}
	h := newTestAnalyticsHandler(engine, &mockThreatStats{})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/alert-forecast?horizon_days=30", nil)
	r = r.WithContext(analyticsCtx())
	w := httptest.NewRecorder()

	h.AlertForecast(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	env := decodeEnvelope(t, w.Body.Bytes())
	var resp predictdto.ForecastResponse
	if err := json.Unmarshal(env["data"], &resp); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if len(resp.Forecast.Points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(resp.Forecast.Points))
	}
	if resp.Forecast.HorizonDays != 30 {
		t.Errorf("expected horizon 30, got %d", resp.Forecast.HorizonDays)
	}
}

func TestAlertForecast_Error(t *testing.T) {
	engine := &mockForecastEngine{
		forecastAlertsFn: func(_ context.Context, _ uuid.UUID, _ int) (*predictdto.ForecastResponse, error) {
			return nil, errors.New("insufficient data")
		},
	}
	h := newTestAnalyticsHandler(engine, &mockThreatStats{})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/alert-forecast", nil)
	r = r.WithContext(analyticsCtx())
	w := httptest.NewRecorder()

	h.AlertForecast(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// TechniqueTrends
// ---------------------------------------------------------------------------

func TestTechniqueTrends_Success(t *testing.T) {
	engine := &mockForecastEngine{
		predictTrendsFn: func(_ context.Context, _ uuid.UUID, horizon int) (*predictdto.TechniqueTrendResponse, error) {
			if horizon != 30 {
				t.Errorf("expected horizon 30, got %d", horizon)
			}
			return &predictdto.TechniqueTrendResponse{
				Items: []predictmodel.TechniqueTrendItem{
					{TechniqueID: "T1059", TechniqueName: "Command and Scripting Interpreter", Trend: "increasing", GrowthRate: 0.15},
					{TechniqueID: "T1078", TechniqueName: "Valid Accounts", Trend: "stable", GrowthRate: 0.0},
					{TechniqueID: "T1190", TechniqueName: "Exploit Public-Facing Application", Trend: "decreasing", GrowthRate: -0.1},
				},
			}, nil
		},
	}
	h := newTestAnalyticsHandler(engine, &mockThreatStats{})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/technique-trends?horizon_days=30", nil)
	r = r.WithContext(analyticsCtx())
	w := httptest.NewRecorder()

	h.TechniqueTrends(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	env := decodeEnvelope(t, w.Body.Bytes())
	var resp predictdto.TechniqueTrendResponse
	if err := json.Unmarshal(env["data"], &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(resp.Items))
	}
}

// ---------------------------------------------------------------------------
// Campaigns
// ---------------------------------------------------------------------------

func TestCampaigns_Success(t *testing.T) {
	now := time.Now()
	engine := &mockForecastEngine{
		detectCampaignsFn: func(_ context.Context, _ uuid.UUID, lookback int) (*predictdto.CampaignResponse, error) {
			if lookback != 30 {
				t.Errorf("expected lookback 30, got %d", lookback)
			}
			return &predictdto.CampaignResponse{
				Items: []predictmodel.CampaignCluster{
					{
						ClusterID:          "c1",
						AlertIDs:           []string{"a1", "a2"},
						AlertTitles:        []string{"Alert One", "Alert Two"},
						StartAt:            now.Add(-48 * time.Hour),
						EndAt:              now,
						Stage:              "active_attack",
						MITRETechniques:    []string{"T1566", "T1059"},
						SharedIOCs:         []string{"192.168.1.1"},
						ConfidenceInterval: predictmodel.ConfidenceInterval{P10: 0.7, P50: 0.8, P90: 0.9},
					},
				},
			}, nil
		},
	}
	h := newTestAnalyticsHandler(engine, &mockThreatStats{})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/campaigns?lookback_days=30", nil)
	r = r.WithContext(analyticsCtx())
	w := httptest.NewRecorder()

	h.Campaigns(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	env := decodeEnvelope(t, w.Body.Bytes())
	var resp predictdto.CampaignResponse
	if err := json.Unmarshal(env["data"], &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 campaign, got %d", len(resp.Items))
	}
	c := resp.Items[0]
	if c.Stage != "active_attack" {
		t.Errorf("expected stage active_attack, got %s", c.Stage)
	}
	if len(c.AlertIDs) != 2 {
		t.Errorf("expected 2 alert IDs, got %d", len(c.AlertIDs))
	}
}

func TestCampaigns_LookbackBounds(t *testing.T) {
	var captured int
	engine := &mockForecastEngine{
		detectCampaignsFn: func(_ context.Context, _ uuid.UUID, lookback int) (*predictdto.CampaignResponse, error) {
			captured = lookback
			return &predictdto.CampaignResponse{}, nil
		},
	}
	h := newTestAnalyticsHandler(engine, &mockThreatStats{})

	// lookback_days=200 exceeds max (180) — should use default 30
	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/campaigns?lookback_days=200", nil)
	r = r.WithContext(analyticsCtx())
	w := httptest.NewRecorder()
	h.Campaigns(w, r)

	if captured != 30 {
		t.Errorf("expected default 30 for out-of-range, got %d", captured)
	}
}

func TestCampaigns_Error(t *testing.T) {
	engine := &mockForecastEngine{
		detectCampaignsFn: func(_ context.Context, _ uuid.UUID, _ int) (*predictdto.CampaignResponse, error) {
			return nil, errors.New("detection failed")
		},
	}
	h := newTestAnalyticsHandler(engine, &mockThreatStats{})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/campaigns", nil)
	r = r.WithContext(analyticsCtx())
	w := httptest.NewRecorder()
	h.Campaigns(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Landscape
// ---------------------------------------------------------------------------

func TestLandscape_Success(t *testing.T) {
	stats := &mockThreatStats{
		statsFn: func(_ context.Context, tenantID uuid.UUID) (*model.ThreatStats, error) {
			if tenantID != analyticsTenantID {
				t.Errorf("unexpected tenant: %s", tenantID)
			}
			return &model.ThreatStats{
				Total:           42,
				Active:          12,
				IndicatorsTotal: 256,
				ByType: []model.NamedCount{
					{Name: "apt", Count: 8},
					{Name: "malware", Count: 4},
				},
				BySeverity: []model.NamedCount{
					{Name: "critical", Count: 5},
					{Name: "high", Count: 7},
				},
			}, nil
		},
	}
	h := newTestAnalyticsHandler(&mockForecastEngine{}, stats)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/landscape", nil)
	r = r.WithContext(analyticsCtx())
	w := httptest.NewRecorder()

	h.Landscape(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	env := decodeEnvelope(t, w.Body.Bytes())
	var landscape model.ThreatLandscape
	if err := json.Unmarshal(env["data"], &landscape); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if landscape.ActiveThreatCount != 12 {
		t.Errorf("expected active 12, got %d", landscape.ActiveThreatCount)
	}
	if landscape.TotalThreats != 42 {
		t.Errorf("expected total 42, got %d", landscape.TotalThreats)
	}
	if landscape.IndicatorsTotal != 256 {
		t.Errorf("expected indicators 256, got %d", landscape.IndicatorsTotal)
	}
	if landscape.TopThreatType != "apt" {
		t.Errorf("expected top type apt, got %s", landscape.TopThreatType)
	}
	if len(landscape.ByType) != 2 {
		t.Errorf("expected 2 types, got %d", len(landscape.ByType))
	}
	if len(landscape.BySeverity) != 2 {
		t.Errorf("expected 2 severities, got %d", len(landscape.BySeverity))
	}
}

func TestLandscape_EmptyStats(t *testing.T) {
	stats := &mockThreatStats{
		statsFn: func(_ context.Context, _ uuid.UUID) (*model.ThreatStats, error) {
			return &model.ThreatStats{}, nil
		},
	}
	h := newTestAnalyticsHandler(&mockForecastEngine{}, stats)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/landscape", nil)
	r = r.WithContext(analyticsCtx())
	w := httptest.NewRecorder()

	h.Landscape(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Verify normalize produces [] not null
	env := decodeEnvelope(t, w.Body.Bytes())
	var landscape model.ThreatLandscape
	if err := json.Unmarshal(env["data"], &landscape); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if landscape.ByType == nil {
		t.Error("expected ByType to be [] not nil after normalize")
	}
	if landscape.BySeverity == nil {
		t.Error("expected BySeverity to be [] not nil after normalize")
	}
}

func TestLandscape_Error(t *testing.T) {
	stats := &mockThreatStats{
		statsFn: func(_ context.Context, _ uuid.UUID) (*model.ThreatStats, error) {
			return nil, errors.New("db connection lost")
		},
	}
	h := newTestAnalyticsHandler(&mockForecastEngine{}, stats)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/landscape", nil)
	r = r.WithContext(analyticsCtx())
	w := httptest.NewRecorder()

	h.Landscape(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestLandscape_NoAuth(t *testing.T) {
	h := newTestAnalyticsHandler(&mockForecastEngine{}, &mockThreatStats{})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/landscape", nil)
	w := httptest.NewRecorder()

	h.Landscape(w, r)

	if w.Code == http.StatusOK {
		t.Fatal("expected non-200 for unauthenticated request")
	}
}

// ---------------------------------------------------------------------------
// Response serialization contract: verify JSON field names match frontend
// ---------------------------------------------------------------------------

func TestLandscape_JSONFieldNames(t *testing.T) {
	stats := &mockThreatStats{
		statsFn: func(_ context.Context, _ uuid.UUID) (*model.ThreatStats, error) {
			return &model.ThreatStats{
				Total: 10, Active: 3, IndicatorsTotal: 50,
				ByType:     []model.NamedCount{{Name: "apt", Count: 3}},
				BySeverity: []model.NamedCount{{Name: "critical", Count: 2}},
			}, nil
		},
	}
	h := newTestAnalyticsHandler(&mockForecastEngine{}, stats)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/landscape", nil)
	r = r.WithContext(analyticsCtx())
	w := httptest.NewRecorder()
	h.Landscape(w, r)

	// Parse as generic map to verify field names
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal(raw["data"], &data); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}

	// These field names must match the frontend AnalyticsLandscape interface
	expectedFields := []string{
		"active_threat_count",
		"total_threats",
		"indicators_total",
		"top_threat_type",
		"by_type",
		"by_severity",
	}
	for _, field := range expectedFields {
		if _, ok := data[field]; !ok {
			t.Errorf("missing expected JSON field %q in landscape response", field)
		}
	}
}

func TestCampaigns_JSONFieldNames(t *testing.T) {
	now := time.Now()
	engine := &mockForecastEngine{
		detectCampaignsFn: func(_ context.Context, _ uuid.UUID, _ int) (*predictdto.CampaignResponse, error) {
			return &predictdto.CampaignResponse{
				Items: []predictmodel.CampaignCluster{
					{
						ClusterID:          "c1",
						AlertIDs:           []string{"a1"},
						AlertTitles:        []string{"Alert"},
						StartAt:            now,
						EndAt:              now,
						Stage:              "reconnaissance",
						MITRETechniques:    []string{"T1234"},
						SharedIOCs:         []string{"1.2.3.4"},
						ConfidenceInterval: predictmodel.ConfidenceInterval{P10: 0.6, P50: 0.7, P90: 0.8},
					},
				},
			}, nil
		},
	}
	h := newTestAnalyticsHandler(engine, &mockThreatStats{})

	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/analytics/campaigns", nil)
	r = r.WithContext(analyticsCtx())
	w := httptest.NewRecorder()
	h.Campaigns(w, r)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal(raw["data"], &data); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}

	// Verify the "items" field exists at data level (frontend reads data.items)
	if _, ok := data["items"]; !ok {
		t.Error("missing 'items' field in campaign response data")
	}

	// Parse first item and verify field names match frontend CampaignCluster
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(data["items"], &items); err != nil {
		t.Fatalf("unmarshal items: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least 1 item")
	}
	expectedFields := []string{
		"cluster_id",
		"alert_ids",
		"alert_titles",
		"start_at",
		"end_at",
		"stage",
		"mitre_techniques",
		"shared_iocs",
		"confidence_interval",
	}
	for _, field := range expectedFields {
		if _, ok := items[0][field]; !ok {
			t.Errorf("missing expected JSON field %q in campaign item", field)
		}
	}
}
