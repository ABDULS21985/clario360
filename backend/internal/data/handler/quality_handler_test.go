package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/service"
)

// ---------------------------------------------------------------------------
// mock qualityService
// ---------------------------------------------------------------------------

type mockQualityService struct {
	createRuleFn  func(ctx context.Context, tenantID, userID uuid.UUID, req dto.CreateQualityRuleRequest) (*model.QualityRule, error)
	listRulesFn   func(ctx context.Context, tenantID uuid.UUID, params dto.ListQualityRulesParams) ([]*model.QualityRule, int, error)
	getRuleFn     func(ctx context.Context, tenantID, id uuid.UUID) (*model.QualityRule, error)
	updateRuleFn  func(ctx context.Context, tenantID, id uuid.UUID, req dto.UpdateQualityRuleRequest) (*model.QualityRule, error)
	deleteRuleFn  func(ctx context.Context, tenantID, id uuid.UUID) error
	runRuleFn     func(ctx context.Context, tenantID, id uuid.UUID) (*model.QualityResult, error)
	listResultsFn func(ctx context.Context, tenantID uuid.UUID, params dto.ListQualityResultsParams) ([]*model.QualityResult, int, error)
	getResultFn   func(ctx context.Context, tenantID, id uuid.UUID) (*model.QualityResult, error)
	scoreFn       func(ctx context.Context, tenantID uuid.UUID) (*model.QualityScore, error)
	trendFn       func(ctx context.Context, tenantID uuid.UUID, days int) ([]model.QualityTrendPoint, error)
	dashboardFn   func(ctx context.Context, tenantID uuid.UUID) (*model.QualityDashboard, error)
}

func (m *mockQualityService) CreateRule(ctx context.Context, tenantID, userID uuid.UUID, req dto.CreateQualityRuleRequest) (*model.QualityRule, error) {
	if m.createRuleFn != nil {
		return m.createRuleFn(ctx, tenantID, userID, req)
	}
	return nil, nil
}
func (m *mockQualityService) ListRules(ctx context.Context, tenantID uuid.UUID, params dto.ListQualityRulesParams) ([]*model.QualityRule, int, error) {
	if m.listRulesFn != nil {
		return m.listRulesFn(ctx, tenantID, params)
	}
	return nil, 0, nil
}
func (m *mockQualityService) GetRule(ctx context.Context, tenantID, id uuid.UUID) (*model.QualityRule, error) {
	if m.getRuleFn != nil {
		return m.getRuleFn(ctx, tenantID, id)
	}
	return nil, nil
}
func (m *mockQualityService) UpdateRule(ctx context.Context, tenantID, id uuid.UUID, req dto.UpdateQualityRuleRequest) (*model.QualityRule, error) {
	if m.updateRuleFn != nil {
		return m.updateRuleFn(ctx, tenantID, id, req)
	}
	return nil, nil
}
func (m *mockQualityService) DeleteRule(ctx context.Context, tenantID, id uuid.UUID) error {
	if m.deleteRuleFn != nil {
		return m.deleteRuleFn(ctx, tenantID, id)
	}
	return nil
}
func (m *mockQualityService) RunRule(ctx context.Context, tenantID, id uuid.UUID) (*model.QualityResult, error) {
	if m.runRuleFn != nil {
		return m.runRuleFn(ctx, tenantID, id)
	}
	return nil, nil
}
func (m *mockQualityService) ListResults(ctx context.Context, tenantID uuid.UUID, params dto.ListQualityResultsParams) ([]*model.QualityResult, int, error) {
	if m.listResultsFn != nil {
		return m.listResultsFn(ctx, tenantID, params)
	}
	return nil, 0, nil
}
func (m *mockQualityService) GetResult(ctx context.Context, tenantID, id uuid.UUID) (*model.QualityResult, error) {
	if m.getResultFn != nil {
		return m.getResultFn(ctx, tenantID, id)
	}
	return nil, nil
}
func (m *mockQualityService) Score(ctx context.Context, tenantID uuid.UUID) (*model.QualityScore, error) {
	if m.scoreFn != nil {
		return m.scoreFn(ctx, tenantID)
	}
	return nil, nil
}
func (m *mockQualityService) Trend(ctx context.Context, tenantID uuid.UUID, days int) ([]model.QualityTrendPoint, error) {
	if m.trendFn != nil {
		return m.trendFn(ctx, tenantID, days)
	}
	return nil, nil
}
func (m *mockQualityService) Dashboard(ctx context.Context, tenantID uuid.UUID) (*model.QualityDashboard, error) {
	if m.dashboardFn != nil {
		return m.dashboardFn(ctx, tenantID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// sample data
// ---------------------------------------------------------------------------

func sampleQualityRule() *model.QualityRule {
	now := time.Now()
	return &model.QualityRule{
		ID:        uuid.New(),
		TenantID:  testTenantID,
		Name:      "not_null_email",
		RuleType:  "not_null",
		Severity:  "high",
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func newQualityHandler(svc *mockQualityService) *QualityHandler {
	return NewQualityHandler(svc, testLogger)
}

// ---------------------------------------------------------------------------
// Auth enforcement
// ---------------------------------------------------------------------------

func TestQualityHandler_CreateRule_Unauthorized(t *testing.T) {
	h := newQualityHandler(&mockQualityService{})
	w := httptest.NewRecorder()
	r := unauthRequest(http.MethodPost, "/api/v1/data/quality/rules", []byte(`{}`))
	h.CreateRule(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestQualityHandler_ListRules_Unauthorized(t *testing.T) {
	h := newQualityHandler(&mockQualityService{})
	w := httptest.NewRecorder()
	r := unauthRequest(http.MethodGet, "/api/v1/data/quality/rules", nil)
	h.ListRules(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Error mapping
// ---------------------------------------------------------------------------

func TestQualityHandler_ErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"validation", fmt.Errorf("bad: %w", service.ErrValidation), http.StatusBadRequest},
		{"conflict", fmt.Errorf("dup: %w", service.ErrConflict), http.StatusConflict},
		{"generic", fmt.Errorf("oops"), http.StatusInternalServerError},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockQualityService{
				getRuleFn: func(_ context.Context, _, _ uuid.UUID) (*model.QualityRule, error) {
					return nil, tc.err
				},
			}
			h := newQualityHandler(svc)
			w := httptest.NewRecorder()
			r := authRequestWithID(http.MethodGet, "/api/v1/data/quality/rules/x", uuid.New(), nil)
			h.GetRule(w, r)
			if w.Code != tc.wantStatus {
				t.Fatalf("expected %d for %s, got %d", tc.wantStatus, tc.name, w.Code)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Happy paths
// ---------------------------------------------------------------------------

func TestQualityHandler_CreateRule_Success(t *testing.T) {
	rule := sampleQualityRule()
	svc := &mockQualityService{
		createRuleFn: func(_ context.Context, _, _ uuid.UUID, _ dto.CreateQualityRuleRequest) (*model.QualityRule, error) {
			return rule, nil
		},
	}
	h := newQualityHandler(svc)
	body, _ := json.Marshal(dto.CreateQualityRuleRequest{Name: "test", RuleType: "not_null", Severity: "high"})
	w := httptest.NewRecorder()
	r := authRequest(http.MethodPost, "/api/v1/data/quality/rules", body)
	h.CreateRule(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestQualityHandler_ListRules_Success(t *testing.T) {
	rule := sampleQualityRule()
	svc := &mockQualityService{
		listRulesFn: func(_ context.Context, _ uuid.UUID, _ dto.ListQualityRulesParams) ([]*model.QualityRule, int, error) {
			return []*model.QualityRule{rule}, 1, nil
		},
	}
	h := newQualityHandler(svc)
	w := httptest.NewRecorder()
	r := authRequest(http.MethodGet, "/api/v1/data/quality/rules?page=1&per_page=10", nil)
	h.ListRules(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestQualityHandler_RunRule_Success(t *testing.T) {
	result := &model.QualityResult{ID: uuid.New(), TenantID: testTenantID, Status: "running"}
	svc := &mockQualityService{
		runRuleFn: func(_ context.Context, _, _ uuid.UUID) (*model.QualityResult, error) {
			return result, nil
		},
	}
	h := newQualityHandler(svc)
	w := httptest.NewRecorder()
	r := authRequestWithID(http.MethodPost, "/api/v1/data/quality/rules/x/run", uuid.New(), nil)
	h.RunRule(w, r)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
}

func TestQualityHandler_Score_Success(t *testing.T) {
	svc := &mockQualityService{
		scoreFn: func(_ context.Context, _ uuid.UUID) (*model.QualityScore, error) {
			return &model.QualityScore{OverallScore: 85.5}, nil
		},
	}
	h := newQualityHandler(svc)
	w := httptest.NewRecorder()
	r := authRequest(http.MethodGet, "/api/v1/data/quality/score", nil)
	h.Score(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestQualityHandler_Dashboard_Success(t *testing.T) {
	svc := &mockQualityService{
		dashboardFn: func(_ context.Context, _ uuid.UUID) (*model.QualityDashboard, error) {
			return &model.QualityDashboard{}, nil
		},
	}
	h := newQualityHandler(svc)
	w := httptest.NewRecorder()
	r := authRequest(http.MethodGet, "/api/v1/data/quality/dashboard", nil)
	h.Dashboard(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Parameter parsing
// ---------------------------------------------------------------------------

func TestQualityHandler_ListRules_ParsesEnabled(t *testing.T) {
	var capturedEnabled *bool
	svc := &mockQualityService{
		listRulesFn: func(_ context.Context, _ uuid.UUID, params dto.ListQualityRulesParams) ([]*model.QualityRule, int, error) {
			capturedEnabled = params.Enabled
			return nil, 0, nil
		},
	}
	h := newQualityHandler(svc)
	w := httptest.NewRecorder()
	r := authRequest(http.MethodGet, "/api/v1/data/quality/rules?enabled=true", nil)
	h.ListRules(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if capturedEnabled == nil || !*capturedEnabled {
		t.Error("expected enabled=true to be parsed")
	}
}

func TestQualityHandler_ListRules_InvalidEnabled(t *testing.T) {
	h := newQualityHandler(&mockQualityService{})
	w := httptest.NewRecorder()
	r := authRequest(http.MethodGet, "/api/v1/data/quality/rules?enabled=notbool", nil)
	h.ListRules(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid enabled, got %d", w.Code)
	}
}

func TestQualityHandler_Trend_ParsesDays(t *testing.T) {
	var capturedDays int
	svc := &mockQualityService{
		trendFn: func(_ context.Context, _ uuid.UUID, days int) ([]model.QualityTrendPoint, error) {
			capturedDays = days
			return nil, nil
		},
	}
	h := newQualityHandler(svc)
	w := httptest.NewRecorder()
	r := authRequest(http.MethodGet, "/api/v1/data/quality/score/trend?days=7", nil)
	h.Trend(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if capturedDays != 7 {
		t.Errorf("expected days=7, got %d", capturedDays)
	}
}

func TestQualityHandler_Trend_DefaultDays(t *testing.T) {
	var capturedDays int
	svc := &mockQualityService{
		trendFn: func(_ context.Context, _ uuid.UUID, days int) ([]model.QualityTrendPoint, error) {
			capturedDays = days
			return nil, nil
		},
	}
	h := newQualityHandler(svc)
	w := httptest.NewRecorder()
	r := authRequest(http.MethodGet, "/api/v1/data/quality/score/trend", nil)
	h.Trend(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if capturedDays != 30 {
		t.Errorf("expected default days=30, got %d", capturedDays)
	}
}

func TestQualityHandler_Trend_InvalidDays(t *testing.T) {
	h := newQualityHandler(&mockQualityService{})
	w := httptest.NewRecorder()
	r := authRequest(http.MethodGet, "/api/v1/data/quality/score/trend?days=-1", nil)
	h.Trend(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for negative days, got %d", w.Code)
	}
}
