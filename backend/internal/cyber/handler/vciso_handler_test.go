package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/cyber/service"
)

// ---- mock ----------------------------------------------------------------

type mockVCISOService struct {
	generateBriefingFn func(ctx context.Context, tenantID, userID uuid.UUID, periodDays int, actor *service.Actor) (*model.ExecutiveBriefing, error)
	listBriefingsFn    func(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOBriefingHistoryParams) (*dto.VCISOBriefingHistoryResponse, error)
	recommendationsFn  func(ctx context.Context, tenantID uuid.UUID) ([]model.RiskRecommendation, error)
	generateReportFn   func(ctx context.Context, tenantID, userID uuid.UUID, req *dto.VCISOReportRequest, actor *service.Actor) (*dto.VCISOReportResponse, error)
	postureSummaryFn   func(ctx context.Context, tenantID uuid.UUID) (*model.PostureSummary, error)
}

func (m *mockVCISOService) GenerateBriefing(ctx context.Context, tenantID, userID uuid.UUID, periodDays int, actor *service.Actor) (*model.ExecutiveBriefing, error) {
	if m.generateBriefingFn != nil {
		return m.generateBriefingFn(ctx, tenantID, userID, periodDays, actor)
	}
	return nil, nil
}

func (m *mockVCISOService) ListBriefings(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOBriefingHistoryParams) (*dto.VCISOBriefingHistoryResponse, error) {
	if m.listBriefingsFn != nil {
		return m.listBriefingsFn(ctx, tenantID, params)
	}
	return nil, nil
}

func (m *mockVCISOService) Recommendations(ctx context.Context, tenantID uuid.UUID) ([]model.RiskRecommendation, error) {
	if m.recommendationsFn != nil {
		return m.recommendationsFn(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockVCISOService) GenerateReport(ctx context.Context, tenantID, userID uuid.UUID, req *dto.VCISOReportRequest, actor *service.Actor) (*dto.VCISOReportResponse, error) {
	if m.generateReportFn != nil {
		return m.generateReportFn(ctx, tenantID, userID, req, actor)
	}
	return nil, nil
}

func (m *mockVCISOService) PostureSummary(ctx context.Context, tenantID uuid.UUID) (*model.PostureSummary, error) {
	if m.postureSummaryFn != nil {
		return m.postureSummaryFn(ctx, tenantID)
	}
	return nil, nil
}

// ---- helpers -------------------------------------------------------------

func vcisoAuthCtx() context.Context {
	tenantID := uuid.New()
	userID := uuid.New()
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

func vcisoAuthCtxWithIDs(tenantID, userID uuid.UUID) context.Context {
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

// ---- auth-free 401/403 tests -------------------------------------------

func TestVCISOHandler_NoAuth(t *testing.T) {
	h := NewVCISOHandler(nil)

	cases := []struct {
		name   string
		method string
		invoke func(w http.ResponseWriter, r *http.Request)
		body   []byte
	}{
		{"Briefing", "GET", h.Briefing, nil},
		{"BriefingHistory", "GET", h.BriefingHistory, nil},
		{"Recommendations", "GET", h.Recommendations, nil},
		{"PostureSummary", "GET", h.PostureSummary, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var body *bytes.Buffer
			if tc.body != nil {
				body = bytes.NewBuffer(tc.body)
			} else {
				body = &bytes.Buffer{}
			}
			r := httptest.NewRequest(tc.method, "/cyber/vciso", body)
			w := httptest.NewRecorder()
			tc.invoke(w, r)
			// Without auth context requireTenantAndUser writes 403 (no tenant)
			// or 401 (no user). Either way >= 400.
			if w.Code < 400 {
				t.Errorf("%s: expected 4xx without auth, got %d", tc.name, w.Code)
			}
		})
	}
}

// TestVCISOHandler_Report_NoAuth verifies the POST /vciso/report endpoint
// also rejects unauthenticated requests before it attempts JSON decode.
func TestVCISOHandler_Report_NoAuth(t *testing.T) {
	h := NewVCISOHandler(nil)
	body, _ := json.Marshal(dto.VCISOReportRequest{Type: "executive", PeriodDays: 30})
	r := httptest.NewRequest("POST", "/cyber/vciso/report", bytes.NewBuffer(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Report(w, r)
	if w.Code < 400 {
		t.Errorf("Report: expected 4xx without auth, got %d", w.Code)
	}
}

// ---- writeError mapping -------------------------------------------------

func TestVCISOWriteError(t *testing.T) {
	h := &VCISOHandler{}

	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			name:       "ErrNotFound maps to 404",
			err:        repository.ErrNotFound,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "wrapped ErrNotFound maps to 404",
			err:        fmt.Errorf("briefing record not found: %w", repository.ErrNotFound),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "generic error maps to 500",
			err:        fmt.Errorf("LLM generation timeout"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "wrapped generic error maps to 500",
			err:        fmt.Errorf("report: %w", fmt.Errorf("PDF render failed")),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			h.writeError(w, tc.err)
			if w.Code != tc.wantStatus {
				t.Errorf("writeError(%v): got status %d, want %d", tc.err, w.Code, tc.wantStatus)
			}
			// Ensure the response body is valid JSON.
			var body map[string]any
			if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
				t.Errorf("response body is not valid JSON: %v", err)
			}
		})
	}
}

// ---- VCISOBriefingParams (SetDefaults) ----------------------------------

func TestVCISOBriefingParams_Defaults(t *testing.T) {
	p := &dto.VCISOBriefingParams{}
	p.SetDefaults()
	if p.PeriodDays != 30 {
		t.Errorf("expected default PeriodDays=30, got %d", p.PeriodDays)
	}
}

func TestVCISOBriefingParams_CapsAt365(t *testing.T) {
	p := &dto.VCISOBriefingParams{PeriodDays: 400}
	p.SetDefaults()
	if p.PeriodDays != 365 {
		t.Errorf("expected PeriodDays capped at 365, got %d", p.PeriodDays)
	}
}

// ---- VCISOBriefingHistoryParams validation ------------------------------

func TestVCISOBriefingHistoryParams_ValidType(t *testing.T) {
	validTypes := []string{"executive", "technical", "compliance", "custom"}
	for _, typ := range validTypes {
		t.Run(typ, func(t *testing.T) {
			typCopy := typ
			p := &dto.VCISOBriefingHistoryParams{Type: &typCopy}
			p.SetDefaults()
			if err := p.Validate(); err != nil {
				t.Errorf("valid type %q should not produce error, got: %v", typ, err)
			}
		})
	}
}

func TestVCISOBriefingHistoryParams_InvalidType(t *testing.T) {
	bad := "quarterly"
	p := &dto.VCISOBriefingHistoryParams{Type: &bad}
	p.SetDefaults()
	if err := p.Validate(); err == nil {
		t.Errorf("invalid type %q should produce validation error, got nil", bad)
	}
}

func TestVCISOBriefingHistoryParams_DefaultPaging(t *testing.T) {
	p := &dto.VCISOBriefingHistoryParams{}
	p.SetDefaults()
	if p.Page != 1 {
		t.Errorf("expected default page=1, got %d", p.Page)
	}
	if p.PerPage != 20 {
		t.Errorf("expected default per_page=20, got %d", p.PerPage)
	}
}

// ---- VCISOReportRequest validation --------------------------------------

func TestVCISOReportRequest_ValidTypes(t *testing.T) {
	validTypes := []string{"executive", "technical", "compliance", "custom"}
	for _, typ := range validTypes {
		t.Run(typ, func(t *testing.T) {
			req := &dto.VCISOReportRequest{Type: typ, PeriodDays: 30}
			if err := req.Validate(); err != nil {
				t.Errorf("valid type %q should not produce error: %v", typ, err)
			}
		})
	}
}

func TestVCISOReportRequest_InvalidType(t *testing.T) {
	req := &dto.VCISOReportRequest{Type: "monthly", PeriodDays: 30}
	if err := req.Validate(); err == nil {
		t.Error("invalid report type should produce validation error, got nil")
	}
}

func TestVCISOReportRequest_PeriodDaysExceedsMax(t *testing.T) {
	req := &dto.VCISOReportRequest{Type: "executive", PeriodDays: 400}
	if err := req.Validate(); err == nil {
		t.Error("period_days > 365 should produce validation error, got nil")
	}
}

func TestVCISOReportRequest_ZeroPeriodDaysDefaultsTo30(t *testing.T) {
	req := &dto.VCISOReportRequest{Type: "executive", PeriodDays: 0}
	if err := req.Validate(); err != nil {
		t.Errorf("zero period_days should be valid (defaulted to 30 by Validate), got: %v", err)
	}
	if req.PeriodDays != 30 {
		t.Errorf("expected PeriodDays set to 30 by Validate, got %d", req.PeriodDays)
	}
}

// ---- BriefingHistory handler validates query params before auth --------

func TestVCISOHandler_BriefingHistory_InvalidType_NoAuth(t *testing.T) {
	h := NewVCISOHandler(nil)
	r := httptest.NewRequest("GET", "/cyber/vciso/briefing/history?type=quarterly", nil)
	w := httptest.NewRecorder()
	h.BriefingHistory(w, r)
	if w.Code < 400 {
		t.Errorf("expected 4xx, got %d", w.Code)
	}
}

// ---- Briefing handler tests ----------------------------------------------

func TestBriefing_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	mock := &mockVCISOService{
		generateBriefingFn: func(ctx context.Context, tid, uid uuid.UUID, periodDays int, actor *service.Actor) (*model.ExecutiveBriefing, error) {
			if tid != tenantID {
				t.Errorf("expected tenantID %s, got %s", tenantID, tid)
			}
			if uid != userID {
				t.Errorf("expected userID %s, got %s", userID, uid)
			}
			if periodDays != 30 {
				t.Errorf("expected default periodDays=30, got %d", periodDays)
			}
			return &model.ExecutiveBriefing{
				GeneratedAt: now,
				Period: model.DateRange{
					Start: now.AddDate(0, 0, -30),
					End:   now,
					Days:  30,
				},
				CriticalIssues:  []model.CriticalIssue{},
				Recommendations: []model.RiskRecommendation{},
			}, nil
		},
	}
	h := NewVCISOHandler(mock)

	ctx := vcisoAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/vciso/briefing", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.Briefing(w, r)

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

func TestBriefing_CustomPeriod(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	mock := &mockVCISOService{
		generateBriefingFn: func(ctx context.Context, tid, uid uuid.UUID, periodDays int, actor *service.Actor) (*model.ExecutiveBriefing, error) {
			if periodDays != 7 {
				t.Errorf("expected periodDays=7, got %d", periodDays)
			}
			return &model.ExecutiveBriefing{
				GeneratedAt: now,
				Period: model.DateRange{
					Start: now.AddDate(0, 0, -7),
					End:   now,
					Days:  7,
				},
				CriticalIssues:  []model.CriticalIssue{},
				Recommendations: []model.RiskRecommendation{},
			}, nil
		},
	}
	h := NewVCISOHandler(mock)

	ctx := vcisoAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/vciso/briefing?period_days=7", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.Briefing(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

func TestBriefing_ServiceError(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockVCISOService{
		generateBriefingFn: func(ctx context.Context, tid, uid uuid.UUID, periodDays int, actor *service.Actor) (*model.ExecutiveBriefing, error) {
			return nil, fmt.Errorf("LLM generation timeout")
		},
	}
	h := NewVCISOHandler(mock)

	ctx := vcisoAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/vciso/briefing", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.Briefing(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

// ---- BriefingHistory handler tests ---------------------------------------

func TestBriefingHistory_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockVCISOService{
		listBriefingsFn: func(ctx context.Context, tid uuid.UUID, params *dto.VCISOBriefingHistoryParams) (*dto.VCISOBriefingHistoryResponse, error) {
			if tid != tenantID {
				t.Errorf("expected tenantID %s, got %s", tenantID, tid)
			}
			return &dto.VCISOBriefingHistoryResponse{
				Data: []*model.VCISOBriefingRecord{},
				Meta: dto.PaginationMeta{
					Total:      0,
					Page:       1,
					PerPage:    20,
					TotalPages: 0,
				},
			}, nil
		},
	}
	h := NewVCISOHandler(mock)

	ctx := vcisoAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/vciso/briefing/history", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.BriefingHistory(w, r)

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

func TestBriefingHistory_WithTypeFilter(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockVCISOService{
		listBriefingsFn: func(ctx context.Context, tid uuid.UUID, params *dto.VCISOBriefingHistoryParams) (*dto.VCISOBriefingHistoryResponse, error) {
			if params.Type == nil || *params.Type != "executive" {
				t.Errorf("expected type filter 'executive', got %v", params.Type)
			}
			return &dto.VCISOBriefingHistoryResponse{
				Data: []*model.VCISOBriefingRecord{},
				Meta: dto.PaginationMeta{
					Total:      0,
					Page:       1,
					PerPage:    20,
					TotalPages: 0,
				},
			}, nil
		},
	}
	h := NewVCISOHandler(mock)

	ctx := vcisoAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/vciso/briefing/history?type=executive", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.BriefingHistory(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

func TestBriefingHistory_InvalidType(t *testing.T) {
	mock := &mockVCISOService{}
	h := NewVCISOHandler(mock)

	ctx := vcisoAuthCtx()
	r := httptest.NewRequest("GET", "/cyber/vciso/briefing/history?type=badtype", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.BriefingHistory(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid type, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

func TestBriefingHistory_Pagination(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockVCISOService{
		listBriefingsFn: func(ctx context.Context, tid uuid.UUID, params *dto.VCISOBriefingHistoryParams) (*dto.VCISOBriefingHistoryResponse, error) {
			if params.Page != 2 {
				t.Errorf("expected page=2, got %d", params.Page)
			}
			if params.PerPage != 5 {
				t.Errorf("expected per_page=5, got %d", params.PerPage)
			}
			return &dto.VCISOBriefingHistoryResponse{
				Data: []*model.VCISOBriefingRecord{},
				Meta: dto.PaginationMeta{
					Total:      10,
					Page:       2,
					PerPage:    5,
					TotalPages: 2,
				},
			}, nil
		},
	}
	h := NewVCISOHandler(mock)

	ctx := vcisoAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/vciso/briefing/history?page=2&per_page=5", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.BriefingHistory(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

// ---- Recommendations handler tests --------------------------------------

func TestRecommendations_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockVCISOService{
		recommendationsFn: func(ctx context.Context, tid uuid.UUID) ([]model.RiskRecommendation, error) {
			if tid != tenantID {
				t.Errorf("expected tenantID %s, got %s", tenantID, tid)
			}
			return []model.RiskRecommendation{
				{
					Priority:        1,
					Title:           "Enable MFA for all admin accounts",
					Description:     "Multi-factor authentication is not enabled for 3 admin accounts.",
					Component:       "iam",
					EstimatedImpact: 5.0,
					Effort:          "low",
					Category:        "identity",
					Actions:         []string{"Enable MFA", "Enforce MFA policy"},
				},
			}, nil
		},
	}
	h := NewVCISOHandler(mock)

	ctx := vcisoAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/vciso/recommendations", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.Recommendations(w, r)

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

func TestRecommendations_ServiceError(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockVCISOService{
		recommendationsFn: func(ctx context.Context, tid uuid.UUID) ([]model.RiskRecommendation, error) {
			return nil, fmt.Errorf("risk engine unavailable")
		},
	}
	h := NewVCISOHandler(mock)

	ctx := vcisoAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/vciso/recommendations", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.Recommendations(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

// ---- Report handler tests ------------------------------------------------

func TestReport_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockVCISOService{
		generateReportFn: func(ctx context.Context, tid, uid uuid.UUID, req *dto.VCISOReportRequest, actor *service.Actor) (*dto.VCISOReportResponse, error) {
			if tid != tenantID {
				t.Errorf("expected tenantID %s, got %s", tenantID, tid)
			}
			if uid != userID {
				t.Errorf("expected userID %s, got %s", userID, uid)
			}
			if req.Type != "executive" {
				t.Errorf("expected type=executive, got %s", req.Type)
			}
			return &dto.VCISOReportResponse{
				JobID:  uuid.New().String(),
				Status: "queued",
			}, nil
		},
	}
	h := NewVCISOHandler(mock)

	body, _ := json.Marshal(dto.VCISOReportRequest{Type: "executive", PeriodDays: 30})
	ctx := vcisoAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("POST", "/cyber/vciso/report", bytes.NewBuffer(body)).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Report(w, r)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if resp["data"] == nil {
		t.Error("expected 'data' key in response")
	}
}

func TestReport_InvalidJSON(t *testing.T) {
	mock := &mockVCISOService{}
	h := NewVCISOHandler(mock)

	ctx := vcisoAuthCtx()
	r := httptest.NewRequest("POST", "/cyber/vciso/report", bytes.NewBufferString("{invalid json}")).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Report(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid JSON, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

func TestReport_ValidationError_BadType(t *testing.T) {
	mock := &mockVCISOService{}
	h := NewVCISOHandler(mock)

	body, _ := json.Marshal(dto.VCISOReportRequest{Type: "invalid_type", PeriodDays: 30})
	ctx := vcisoAuthCtx()
	r := httptest.NewRequest("POST", "/cyber/vciso/report", bytes.NewBuffer(body)).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Report(w, r)

	// req.Validate() returns error for invalid type before service is called
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid report type, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

func TestReport_ValidationError_PeriodTooLarge(t *testing.T) {
	mock := &mockVCISOService{}
	h := NewVCISOHandler(mock)

	body, _ := json.Marshal(dto.VCISOReportRequest{Type: "executive", PeriodDays: 400})
	ctx := vcisoAuthCtx()
	r := httptest.NewRequest("POST", "/cyber/vciso/report", bytes.NewBuffer(body)).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Report(w, r)

	// req.Validate() returns error for period_days > 365 before service is called
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for period_days > 365, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

func TestReport_ServiceError(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockVCISOService{
		generateReportFn: func(ctx context.Context, tid, uid uuid.UUID, req *dto.VCISOReportRequest, actor *service.Actor) (*dto.VCISOReportResponse, error) {
			return nil, fmt.Errorf("report generation queue full")
		},
	}
	h := NewVCISOHandler(mock)

	body, _ := json.Marshal(dto.VCISOReportRequest{Type: "executive", PeriodDays: 30})
	ctx := vcisoAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("POST", "/cyber/vciso/report", bytes.NewBuffer(body)).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Report(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

func TestReport_NotFound(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockVCISOService{
		generateReportFn: func(ctx context.Context, tid, uid uuid.UUID, req *dto.VCISOReportRequest, actor *service.Actor) (*dto.VCISOReportResponse, error) {
			return nil, repository.ErrNotFound
		},
	}
	h := NewVCISOHandler(mock)

	body, _ := json.Marshal(dto.VCISOReportRequest{Type: "compliance", PeriodDays: 90})
	ctx := vcisoAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("POST", "/cyber/vciso/report", bytes.NewBuffer(body)).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Report(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

// ---- PostureSummary handler tests ----------------------------------------

func TestPostureSummary_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockVCISOService{
		postureSummaryFn: func(ctx context.Context, tid uuid.UUID) (*model.PostureSummary, error) {
			if tid != tenantID {
				t.Errorf("expected tenantID %s, got %s", tenantID, tid)
			}
			return &model.PostureSummary{
				RiskScore:              62.5,
				Grade:                  "C",
				Trend:                  "improving",
				TrendDelta:             -3.2,
				TopIssues:              []string{"Unpatched critical vulns", "Exposed data assets"},
				OpenCriticalAlerts:     2,
				UnpatchedCriticalVulns: 5,
				ActiveThreats:          1,
				DSPMScore:              78.0,
			}, nil
		},
	}
	h := NewVCISOHandler(mock)

	ctx := vcisoAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/vciso/posture", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.PostureSummary(w, r)

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

func TestPostureSummary_ServiceError(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockVCISOService{
		postureSummaryFn: func(ctx context.Context, tid uuid.UUID) (*model.PostureSummary, error) {
			return nil, fmt.Errorf("posture aggregation failed")
		},
	}
	h := NewVCISOHandler(mock)

	ctx := vcisoAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/vciso/posture", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.PostureSummary(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

// ---- BriefingHistory service error test ----------------------------------

func TestBriefingHistory_ServiceError(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockVCISOService{
		listBriefingsFn: func(ctx context.Context, tid uuid.UUID, params *dto.VCISOBriefingHistoryParams) (*dto.VCISOBriefingHistoryResponse, error) {
			return nil, fmt.Errorf("database timeout")
		},
	}
	h := NewVCISOHandler(mock)

	ctx := vcisoAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/vciso/briefing/history", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.BriefingHistory(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}
