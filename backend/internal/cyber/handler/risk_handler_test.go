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
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/service"
)

// ---------------------------------------------------------------------------
// mock
// ---------------------------------------------------------------------------

type mockRiskService struct {
	getCurrentScoreFn func(ctx context.Context, tenantID uuid.UUID) (*model.OrganizationRiskScore, error)
	recalculateFn     func(ctx context.Context, tenantID uuid.UUID, actor *service.Actor) (*model.OrganizationRiskScore, error)
	trendFn           func(ctx context.Context, tenantID uuid.UUID, days int) ([]model.RiskTrendPoint, error)
	heatmapFn         func(ctx context.Context, tenantID uuid.UUID) (*model.RiskHeatmap, error)
	topRisksFn        func(ctx context.Context, tenantID uuid.UUID) ([]model.RiskContributor, error)
	recommendationsFn func(ctx context.Context, tenantID uuid.UUID) ([]model.RiskRecommendation, error)
}

func (m *mockRiskService) GetCurrentScore(ctx context.Context, tenantID uuid.UUID) (*model.OrganizationRiskScore, error) {
	if m.getCurrentScoreFn != nil {
		return m.getCurrentScoreFn(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockRiskService) Recalculate(ctx context.Context, tenantID uuid.UUID, actor *service.Actor) (*model.OrganizationRiskScore, error) {
	if m.recalculateFn != nil {
		return m.recalculateFn(ctx, tenantID, actor)
	}
	return nil, nil
}

func (m *mockRiskService) Trend(ctx context.Context, tenantID uuid.UUID, days int) ([]model.RiskTrendPoint, error) {
	if m.trendFn != nil {
		return m.trendFn(ctx, tenantID, days)
	}
	return nil, nil
}

func (m *mockRiskService) Heatmap(ctx context.Context, tenantID uuid.UUID) (*model.RiskHeatmap, error) {
	if m.heatmapFn != nil {
		return m.heatmapFn(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockRiskService) TopRisks(ctx context.Context, tenantID uuid.UUID) ([]model.RiskContributor, error) {
	if m.topRisksFn != nil {
		return m.topRisksFn(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockRiskService) Recommendations(ctx context.Context, tenantID uuid.UUID) ([]model.RiskRecommendation, error) {
	if m.recommendationsFn != nil {
		return m.recommendationsFn(ctx, tenantID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func riskAuthRequest(method, path string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	ctx := r.Context()
	ctx = auth.WithTenantID(ctx, tenantID.String())
	ctx = auth.WithUser(ctx, &auth.ContextUser{
		ID:       userID.String(),
		TenantID: tenantID.String(),
		Email:    "analyst@example.com",
		Roles:    []string{"security_analyst"},
	})
	return r.WithContext(ctx)
}

func riskAuthRequestWithPermission(method, path string, roles []string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	ctx := r.Context()
	ctx = auth.WithTenantID(ctx, tenantID.String())
	ctx = auth.WithUser(ctx, &auth.ContextUser{
		ID:       userID.String(),
		TenantID: tenantID.String(),
		Email:    "admin@example.com",
		Roles:    roles,
	})
	return r.WithContext(ctx)
}

// ---------------------------------------------------------------------------
// GetHeatmap tests
// ---------------------------------------------------------------------------

func TestGetHeatmap_Success(t *testing.T) {
	mock := &mockRiskService{
		heatmapFn: func(_ context.Context, _ uuid.UUID) (*model.RiskHeatmap, error) {
			return &model.RiskHeatmap{
				Rows: []model.HeatmapRow{
					{
						AssetType:  "server",
						AssetCount: 10,
						Cells: map[string]model.HeatmapCell{
							"critical": {VulnCount: 5, AffectedAssets: 3},
							"high":     {VulnCount: 12, AffectedAssets: 7},
							"medium":   {VulnCount: 8, AffectedAssets: 4},
							"low":      {VulnCount: 2, AffectedAssets: 1},
						},
						TotalVulns: 27,
					},
					{
						AssetType:  "endpoint",
						AssetCount: 20,
						Cells: map[string]model.HeatmapCell{
							"high": {VulnCount: 3, AffectedAssets: 2},
						},
						TotalVulns: 3,
					},
				},
				MaxValue: 12,
			}, nil
		},
	}

	h := NewRiskHandler(mock)
	w := httptest.NewRecorder()
	r := riskAuthRequest("GET", "/api/v1/cyber/risk/heatmap")
	h.GetHeatmap(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}

	dataRaw, ok := resp["data"]
	if !ok {
		t.Fatal("missing 'data' key in response envelope")
	}

	var data struct {
		Cells                []struct {
			AssetType          string `json:"asset_type"`
			Severity           string `json:"severity"`
			Count              int    `json:"count"`
			AffectedAssetCount int    `json:"affected_asset_count"`
			TotalAssetsOfType  int    `json:"total_assets_of_type"`
		} `json:"cells"`
		AssetTypes           []string `json:"asset_types"`
		TotalVulnerabilities int      `json:"total_vulnerabilities"`
		GeneratedAt          string   `json:"generated_at"`
	}
	if err := json.Unmarshal(dataRaw, &data); err != nil {
		t.Fatalf("failed to unmarshal data: %v", err)
	}

	// Verify flat cell structure — 2 asset types × 4 severities = 8 cells
	if len(data.Cells) != 8 {
		t.Errorf("expected 8 cells, got %d", len(data.Cells))
	}

	// Verify total = 5+12+8+2+0+3+0+0 = 30
	if data.TotalVulnerabilities != 30 {
		t.Errorf("total_vulnerabilities = %d, want 30", data.TotalVulnerabilities)
	}

	// Verify asset_types list
	if len(data.AssetTypes) != 2 {
		t.Errorf("expected 2 asset_types, got %d", len(data.AssetTypes))
	}

	// Verify generated_at is present
	if data.GeneratedAt == "" {
		t.Error("generated_at is empty")
	}

	// Verify a specific cell's field mapping
	var serverCrit *struct {
		AssetType          string `json:"asset_type"`
		Severity           string `json:"severity"`
		Count              int    `json:"count"`
		AffectedAssetCount int    `json:"affected_asset_count"`
		TotalAssetsOfType  int    `json:"total_assets_of_type"`
	}
	for i := range data.Cells {
		if data.Cells[i].AssetType == "server" && data.Cells[i].Severity == "critical" {
			serverCrit = &data.Cells[i]
			break
		}
	}
	if serverCrit == nil {
		t.Fatal("missing cell: server/critical")
	}
	if serverCrit.Count != 5 {
		t.Errorf("server/critical count = %d, want 5", serverCrit.Count)
	}
	if serverCrit.AffectedAssetCount != 3 {
		t.Errorf("server/critical affected_asset_count = %d, want 3", serverCrit.AffectedAssetCount)
	}
	if serverCrit.TotalAssetsOfType != 10 {
		t.Errorf("server/critical total_assets_of_type = %d, want 10", serverCrit.TotalAssetsOfType)
	}
}

func TestGetHeatmap_EmptyRows(t *testing.T) {
	mock := &mockRiskService{
		heatmapFn: func(_ context.Context, _ uuid.UUID) (*model.RiskHeatmap, error) {
			return &model.RiskHeatmap{Rows: []model.HeatmapRow{}}, nil
		},
	}

	h := NewRiskHandler(mock)
	w := httptest.NewRecorder()
	r := riskAuthRequest("GET", "/api/v1/cyber/risk/heatmap")
	h.GetHeatmap(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	var data struct {
		Cells                []json.RawMessage `json:"cells"`
		TotalVulnerabilities int               `json:"total_vulnerabilities"`
	}
	if err := json.Unmarshal(resp["data"], &data); err != nil {
		t.Fatalf("failed to unmarshal data: %v", err)
	}
	if len(data.Cells) != 0 {
		t.Errorf("expected 0 cells, got %d", len(data.Cells))
	}
	if data.TotalVulnerabilities != 0 {
		t.Errorf("total_vulnerabilities = %d, want 0", data.TotalVulnerabilities)
	}
}

func TestGetHeatmap_ServiceError(t *testing.T) {
	mock := &mockRiskService{
		heatmapFn: func(_ context.Context, _ uuid.UUID) (*model.RiskHeatmap, error) {
			return nil, fmt.Errorf("database connection failed")
		},
	}

	h := NewRiskHandler(mock)
	w := httptest.NewRecorder()
	r := riskAuthRequest("GET", "/api/v1/cyber/risk/heatmap")
	h.GetHeatmap(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestGetHeatmap_NoAuth(t *testing.T) {
	h := NewRiskHandler(&mockRiskService{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/cyber/risk/heatmap", nil)
	h.GetHeatmap(w, r)

	if w.Code < 400 {
		t.Errorf("expected 4xx without auth, got %d", w.Code)
	}
}

func TestGetHeatmap_ZeroCountCells(t *testing.T) {
	// When a severity has no vulnerabilities, the map won't have that key.
	// The DTO should still produce a zero-count cell for all 4 severities.
	mock := &mockRiskService{
		heatmapFn: func(_ context.Context, _ uuid.UUID) (*model.RiskHeatmap, error) {
			return &model.RiskHeatmap{
				Rows: []model.HeatmapRow{
					{
						AssetType:  "database",
						AssetCount: 5,
						Cells: map[string]model.HeatmapCell{
							"critical": {VulnCount: 1, AffectedAssets: 1},
							// high, medium, low missing → should be zero
						},
						TotalVulns: 1,
					},
				},
			}, nil
		},
	}

	h := NewRiskHandler(mock)
	w := httptest.NewRecorder()
	r := riskAuthRequest("GET", "/api/v1/cyber/risk/heatmap")
	h.GetHeatmap(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]json.RawMessage
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	var data struct {
		Cells []struct {
			Severity string `json:"severity"`
			Count    int    `json:"count"`
		} `json:"cells"`
	}
	_ = json.Unmarshal(resp["data"], &data)

	if len(data.Cells) != 4 {
		t.Fatalf("expected 4 cells for 1 asset type, got %d", len(data.Cells))
	}

	sevCounts := map[string]int{}
	for _, c := range data.Cells {
		sevCounts[c.Severity] = c.Count
	}
	if sevCounts["critical"] != 1 {
		t.Errorf("critical count = %d, want 1", sevCounts["critical"])
	}
	if sevCounts["high"] != 0 {
		t.Errorf("high count = %d, want 0", sevCounts["high"])
	}
	if sevCounts["medium"] != 0 {
		t.Errorf("medium count = %d, want 0", sevCounts["medium"])
	}
	if sevCounts["low"] != 0 {
		t.Errorf("low count = %d, want 0", sevCounts["low"])
	}
}

// ---------------------------------------------------------------------------
// Other RiskHandler endpoints — auth & error coverage
// ---------------------------------------------------------------------------

func TestRiskHandler_NoAuth(t *testing.T) {
	h := NewRiskHandler(&mockRiskService{})

	cases := []struct {
		name   string
		invoke func(w http.ResponseWriter, r *http.Request)
	}{
		{"GetScore", h.GetScore},
		{"GetTrend", h.GetTrend},
		{"GetHeatmap", h.GetHeatmap},
		{"GetTopRisks", h.GetTopRisks},
		{"GetRecommendations", h.GetRecommendations},
		{"Recalculate", h.Recalculate},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)
			tc.invoke(w, r)
			if w.Code < 400 {
				t.Errorf("%s: expected 4xx without auth, got %d", tc.name, w.Code)
			}
		})
	}
}

func TestRecalculate_Forbidden(t *testing.T) {
	h := NewRiskHandler(&mockRiskService{})
	w := httptest.NewRecorder()
	// User with only read permission, not cyber:write or admin:*
	r := riskAuthRequestWithPermission("POST", "/api/v1/cyber/risk/score/recalculate", []string{"cyber:read"})
	h.Recalculate(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestGetScore_Success(t *testing.T) {
	mock := &mockRiskService{
		getCurrentScoreFn: func(_ context.Context, _ uuid.UUID) (*model.OrganizationRiskScore, error) {
			return &model.OrganizationRiskScore{
				OverallScore: 42.5,
				Grade:        "C",
				Trend:        "stable",
			}, nil
		},
	}

	h := NewRiskHandler(mock)
	w := httptest.NewRecorder()
	r := riskAuthRequest("GET", "/api/v1/cyber/risk/score")
	h.GetScore(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]json.RawMessage
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	var data struct {
		OverallScore float64 `json:"overall_score"`
		Grade        string  `json:"grade"`
	}
	_ = json.Unmarshal(resp["data"], &data)
	if data.OverallScore != 42.5 {
		t.Errorf("overall_score = %f, want 42.5", data.OverallScore)
	}
	if data.Grade != "C" {
		t.Errorf("grade = %q, want C", data.Grade)
	}
}

func TestGetTopRisks_Success(t *testing.T) {
	mock := &mockRiskService{
		topRisksFn: func(_ context.Context, _ uuid.UUID) ([]model.RiskContributor, error) {
			return []model.RiskContributor{
				{Type: "vulnerability", Title: "CVE-2024-1234", Score: 25.3},
			}, nil
		},
	}

	h := NewRiskHandler(mock)
	w := httptest.NewRecorder()
	r := riskAuthRequest("GET", "/api/v1/cyber/risk/top-risks")
	h.GetTopRisks(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetRecommendations_Success(t *testing.T) {
	mock := &mockRiskService{
		recommendationsFn: func(_ context.Context, _ uuid.UUID) ([]model.RiskRecommendation, error) {
			return []model.RiskRecommendation{
				{Priority: 1, Title: "Patch critical vulns", Component: "vulnerability_risk"},
			}, nil
		},
	}

	h := NewRiskHandler(mock)
	w := httptest.NewRecorder()
	r := riskAuthRequest("GET", "/api/v1/cyber/risk/recommendations")
	h.GetRecommendations(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
