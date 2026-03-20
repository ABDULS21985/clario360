package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
)

// ---- mock ----------------------------------------------------------------

type mockDashboardService struct {
	getSOCDashboardFn        func(ctx context.Context, tenantID uuid.UUID) (*model.SOCDashboard, error)
	getKPIsFn                func(ctx context.Context, tenantID uuid.UUID) (model.KPICards, error)
	getAlertTimelineFn       func(ctx context.Context, tenantID uuid.UUID) (model.AlertTimelineData, error)
	getSeverityDistFn        func(ctx context.Context, tenantID uuid.UUID) (model.SeverityDistribution, error)
	getMTTRFn                func(ctx context.Context, tenantID uuid.UUID) (*model.MTTRReport, error)
	getAnalystWorkloadFn     func(ctx context.Context, tenantID uuid.UUID) ([]model.AnalystWorkloadEntry, error)
	getTopAttackedAssetsFn   func(ctx context.Context, tenantID uuid.UUID) ([]model.AssetAlertSummary, error)
	getMITREHeatmapFn        func(ctx context.Context, tenantID uuid.UUID) (model.MITREHeatmapData, error)
	getMetricsFn             func(ctx context.Context, tenantID uuid.UUID) (*dto.DashboardMetricsResponse, error)
	getTrendsFn              func(ctx context.Context, tenantID uuid.UUID, days int) (*dto.DashboardTrendsResponse, error)
}

func (m *mockDashboardService) GetSOCDashboard(ctx context.Context, tenantID uuid.UUID) (*model.SOCDashboard, error) {
	if m.getSOCDashboardFn != nil {
		return m.getSOCDashboardFn(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockDashboardService) GetKPIs(ctx context.Context, tenantID uuid.UUID) (model.KPICards, error) {
	if m.getKPIsFn != nil {
		return m.getKPIsFn(ctx, tenantID)
	}
	return model.KPICards{}, nil
}

func (m *mockDashboardService) GetAlertTimeline(ctx context.Context, tenantID uuid.UUID) (model.AlertTimelineData, error) {
	if m.getAlertTimelineFn != nil {
		return m.getAlertTimelineFn(ctx, tenantID)
	}
	return model.AlertTimelineData{}, nil
}

func (m *mockDashboardService) GetSeverityDistribution(ctx context.Context, tenantID uuid.UUID) (model.SeverityDistribution, error) {
	if m.getSeverityDistFn != nil {
		return m.getSeverityDistFn(ctx, tenantID)
	}
	return model.SeverityDistribution{}, nil
}

func (m *mockDashboardService) GetMTTR(ctx context.Context, tenantID uuid.UUID) (*model.MTTRReport, error) {
	if m.getMTTRFn != nil {
		return m.getMTTRFn(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockDashboardService) GetAnalystWorkload(ctx context.Context, tenantID uuid.UUID) ([]model.AnalystWorkloadEntry, error) {
	if m.getAnalystWorkloadFn != nil {
		return m.getAnalystWorkloadFn(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockDashboardService) GetTopAttackedAssets(ctx context.Context, tenantID uuid.UUID) ([]model.AssetAlertSummary, error) {
	if m.getTopAttackedAssetsFn != nil {
		return m.getTopAttackedAssetsFn(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockDashboardService) GetMITREHeatmap(ctx context.Context, tenantID uuid.UUID) (model.MITREHeatmapData, error) {
	if m.getMITREHeatmapFn != nil {
		return m.getMITREHeatmapFn(ctx, tenantID)
	}
	return model.MITREHeatmapData{}, nil
}

func (m *mockDashboardService) GetMetrics(ctx context.Context, tenantID uuid.UUID) (*dto.DashboardMetricsResponse, error) {
	if m.getMetricsFn != nil {
		return m.getMetricsFn(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockDashboardService) GetTrends(ctx context.Context, tenantID uuid.UUID, days int) (*dto.DashboardTrendsResponse, error) {
	if m.getTrendsFn != nil {
		return m.getTrendsFn(ctx, tenantID, days)
	}
	return nil, nil
}

// ---- helpers -------------------------------------------------------------

func dashAuthCtx(tenantID, userID uuid.UUID) context.Context {
	ctx := context.Background()
	ctx = auth.WithUser(ctx, &auth.ContextUser{
		ID:       userID.String(),
		TenantID: tenantID.String(),
		Email:    "analyst@example.com",
		Roles:    []string{"security_analyst"},
	})
	ctx = auth.WithTenantID(ctx, tenantID.String())
	return ctx
}

// ---- NoAuth tests --------------------------------------------------------

func TestDashboardHandler_NoAuth(t *testing.T) {
	h := NewDashboardHandler(&mockDashboardService{})

	cases := []struct {
		name   string
		invoke func(w http.ResponseWriter, r *http.Request)
	}{
		{"GetDashboard", h.GetDashboard},
		{"GetKPIs", h.GetKPIs},
		{"GetAlertsTimeline", h.GetAlertsTimeline},
		{"GetSeverityDistribution", h.GetSeverityDistribution},
		{"GetMTTR", h.GetMTTR},
		{"GetAnalystWorkload", h.GetAnalystWorkload},
		{"GetTopAttackedAssets", h.GetTopAttackedAssets},
		{"GetMITREHeatmap", h.GetMITREHeatmap},
		{"GetMetrics", h.GetMetrics},
		{"GetTrends", h.GetTrends},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/cyber/dashboard", nil)
			w := httptest.NewRecorder()
			tc.invoke(w, r)
			if w.Code < 400 {
				t.Errorf("%s: expected 4xx without auth, got %d", tc.name, w.Code)
			}
		})
	}
}

// ---- GetMetrics tests ----------------------------------------------------

func TestGetMetrics_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mttr := 45.5
	mtta := 12.3
	sla := 97.8
	incidents := 3
	users := 8
	reviews := 2

	mock := &mockDashboardService{
		getMetricsFn: func(ctx context.Context, tid uuid.UUID) (*dto.DashboardMetricsResponse, error) {
			if tid != tenantID {
				t.Errorf("expected tenantID %s, got %s", tenantID, tid)
			}
			return &dto.DashboardMetricsResponse{
				MTTRMinutes:      &mttr,
				MTTAMinutes:      &mtta,
				SLACompliancePct: &sla,
				ActiveIncidents:  &incidents,
				ActiveUsersToday: &users,
				PendingReviews:   &reviews,
			}, nil
		},
	}
	h := NewDashboardHandler(mock)

	ctx := dashAuthCtx(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dashboard/metrics", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.GetMetrics(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected 'data' key in response envelope")
	}

	checks := map[string]float64{
		"mttr_minutes":       mttr,
		"mtta_minutes":       mtta,
		"sla_compliance_pct": sla,
		"active_incidents":   float64(incidents),
		"active_users_today": float64(users),
		"pending_reviews":    float64(reviews),
	}
	for key, want := range checks {
		got, exists := data[key]
		if !exists {
			t.Errorf("missing key %q in response data", key)
			continue
		}
		gotF, ok := got.(float64)
		if !ok {
			t.Errorf("key %q: expected float64, got %T", key, got)
			continue
		}
		if gotF != want {
			t.Errorf("key %q: got %v, want %v", key, gotF, want)
		}
	}
}

func TestGetMetrics_PartialNulls(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	incidents := 5

	mock := &mockDashboardService{
		getMetricsFn: func(ctx context.Context, tid uuid.UUID) (*dto.DashboardMetricsResponse, error) {
			return &dto.DashboardMetricsResponse{
				ActiveIncidents: &incidents,
				// All other fields nil (simulates partial failure)
			}, nil
		},
	}
	h := NewDashboardHandler(mock)

	ctx := dashAuthCtx(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dashboard/metrics", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.GetMetrics(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}

	data := resp["data"].(map[string]any)

	// Active incidents should be set
	if got, _ := data["active_incidents"].(float64); int(got) != incidents {
		t.Errorf("active_incidents: got %v, want %d", got, incidents)
	}

	// Nullable fields should be null
	nullableFields := []string{"mttr_minutes", "mtta_minutes", "sla_compliance_pct", "active_users_today", "pending_reviews"}
	for _, key := range nullableFields {
		if data[key] != nil {
			t.Errorf("expected %q to be null, got %v", key, data[key])
		}
	}
}

func TestGetMetrics_ServiceError(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockDashboardService{
		getMetricsFn: func(ctx context.Context, tid uuid.UUID) (*dto.DashboardMetricsResponse, error) {
			return nil, fmt.Errorf("database connection lost")
		},
	}
	h := NewDashboardHandler(mock)

	ctx := dashAuthCtx(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dashboard/metrics", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.GetMetrics(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if resp["code"] != "INTERNAL_ERROR" {
		t.Errorf("expected error code INTERNAL_ERROR, got %v", resp["code"])
	}
}

// ---- GetDashboard tests --------------------------------------------------

func TestGetDashboard_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockDashboardService{
		getSOCDashboardFn: func(ctx context.Context, tid uuid.UUID) (*model.SOCDashboard, error) {
			if tid != tenantID {
				t.Errorf("expected tenantID %s, got %s", tenantID, tid)
			}
			return &model.SOCDashboard{}, nil
		},
	}
	h := NewDashboardHandler(mock)

	ctx := dashAuthCtx(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dashboard", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.GetDashboard(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if resp["data"] == nil {
		t.Error("expected 'data' key in response")
	}
}

func TestGetDashboard_ServiceError(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockDashboardService{
		getSOCDashboardFn: func(ctx context.Context, tid uuid.UUID) (*model.SOCDashboard, error) {
			return nil, fmt.Errorf("timeout")
		},
	}
	h := NewDashboardHandler(mock)

	ctx := dashAuthCtx(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dashboard", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.GetDashboard(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// ---- GetKPIs tests -------------------------------------------------------

func TestGetKPIs_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockDashboardService{
		getKPIsFn: func(ctx context.Context, tid uuid.UUID) (model.KPICards, error) {
			return model.KPICards{
				OpenAlerts: 42,
				RiskScore:  78.5,
			}, nil
		},
	}
	h := NewDashboardHandler(mock)

	ctx := dashAuthCtx(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dashboard/kpis", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.GetKPIs(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected 'data' key in response")
	}
	if data["open_alerts"] != float64(42) {
		t.Errorf("expected open_alerts=42, got %v", data["open_alerts"])
	}
}

// ---- GetTrends tests -----------------------------------------------------

func TestGetTrends_DefaultDays(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockDashboardService{
		getTrendsFn: func(ctx context.Context, tid uuid.UUID, days int) (*dto.DashboardTrendsResponse, error) {
			if days != 30 {
				t.Errorf("expected default days=30, got %d", days)
			}
			return &dto.DashboardTrendsResponse{Days: days}, nil
		},
	}
	h := NewDashboardHandler(mock)

	ctx := dashAuthCtx(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dashboard/trends", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.GetTrends(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestGetTrends_CustomDays(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockDashboardService{
		getTrendsFn: func(ctx context.Context, tid uuid.UUID, days int) (*dto.DashboardTrendsResponse, error) {
			if days != 7 {
				t.Errorf("expected days=7, got %d", days)
			}
			return &dto.DashboardTrendsResponse{Days: days}, nil
		},
	}
	h := NewDashboardHandler(mock)

	ctx := dashAuthCtx(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dashboard/trends?days=7", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.GetTrends(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}
