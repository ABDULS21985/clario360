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

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/cyber/dspm/shadow"
	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/cyber/service"
)

// ---- mock ----------------------------------------------------------------

type mockDSPMService struct {
	listDataAssetsFn        func(ctx context.Context, tenantID uuid.UUID, params *dto.DSPMAssetListParams) (*dto.DSPMAssetListResponse, error)
	getDataAssetFn          func(ctx context.Context, tenantID, assetID uuid.UUID) (*model.DSPMDataAsset, error)
	triggerScanFn           func(ctx context.Context, tenantID, userID uuid.UUID, actor *service.Actor, req *dto.DSPMScanTriggerRequest) (*model.DSPMScan, error)
	listScansFn             func(ctx context.Context, tenantID uuid.UUID, params *dto.DSPMScanListParams) (*dto.DSPMScanListResponse, error)
	getScanFn               func(ctx context.Context, tenantID, scanID uuid.UUID) (*model.DSPMScanResult, error)
	classificationSummaryFn func(ctx context.Context, tenantID uuid.UUID) (*model.DSPMClassificationSummary, error)
	exposureAnalysisFn      func(ctx context.Context, tenantID uuid.UUID) (*model.DSPMExposureAnalysis, error)
	dependenciesFn          func(ctx context.Context, tenantID uuid.UUID) ([]model.DSPMDependencyNode, error)
	dashboardFn             func(ctx context.Context, tenantID uuid.UUID) (*model.DSPMDashboard, error)
	detectShadowCopiesFn    func(ctx context.Context, tenantID uuid.UUID) (*shadow.DetectionResult, error)
}

func (m *mockDSPMService) ListDataAssets(ctx context.Context, tenantID uuid.UUID, params *dto.DSPMAssetListParams) (*dto.DSPMAssetListResponse, error) {
	if m.listDataAssetsFn != nil {
		return m.listDataAssetsFn(ctx, tenantID, params)
	}
	return nil, nil
}

func (m *mockDSPMService) GetDataAsset(ctx context.Context, tenantID, assetID uuid.UUID) (*model.DSPMDataAsset, error) {
	if m.getDataAssetFn != nil {
		return m.getDataAssetFn(ctx, tenantID, assetID)
	}
	return nil, nil
}

func (m *mockDSPMService) TriggerScan(ctx context.Context, tenantID, userID uuid.UUID, actor *service.Actor, req *dto.DSPMScanTriggerRequest) (*model.DSPMScan, error) {
	if m.triggerScanFn != nil {
		return m.triggerScanFn(ctx, tenantID, userID, actor, req)
	}
	return nil, nil
}

func (m *mockDSPMService) ListScans(ctx context.Context, tenantID uuid.UUID, params *dto.DSPMScanListParams) (*dto.DSPMScanListResponse, error) {
	if m.listScansFn != nil {
		return m.listScansFn(ctx, tenantID, params)
	}
	return nil, nil
}

func (m *mockDSPMService) GetScan(ctx context.Context, tenantID, scanID uuid.UUID) (*model.DSPMScanResult, error) {
	if m.getScanFn != nil {
		return m.getScanFn(ctx, tenantID, scanID)
	}
	return nil, nil
}

func (m *mockDSPMService) ClassificationSummary(ctx context.Context, tenantID uuid.UUID) (*model.DSPMClassificationSummary, error) {
	if m.classificationSummaryFn != nil {
		return m.classificationSummaryFn(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockDSPMService) ExposureAnalysis(ctx context.Context, tenantID uuid.UUID) (*model.DSPMExposureAnalysis, error) {
	if m.exposureAnalysisFn != nil {
		return m.exposureAnalysisFn(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockDSPMService) Dependencies(ctx context.Context, tenantID uuid.UUID) ([]model.DSPMDependencyNode, error) {
	if m.dependenciesFn != nil {
		return m.dependenciesFn(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockDSPMService) Dashboard(ctx context.Context, tenantID uuid.UUID) (*model.DSPMDashboard, error) {
	if m.dashboardFn != nil {
		return m.dashboardFn(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockDSPMService) DetectShadowCopies(ctx context.Context, tenantID uuid.UUID) (*shadow.DetectionResult, error) {
	if m.detectShadowCopiesFn != nil {
		return m.detectShadowCopiesFn(ctx, tenantID)
	}
	return nil, nil
}

// ---- helpers -------------------------------------------------------------

func dspmAuthCtx() context.Context {
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

func dspmAuthCtxWithIDs(tenantID, userID uuid.UUID) context.Context {
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

func dspmRequestWithID(method, url string, body []byte, ctx context.Context, id string) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		r = httptest.NewRequest(method, url, nil)
	}
	r = r.WithContext(ctx)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	return r
}

// ---- auth-free 401/403 tests -------------------------------------------

func TestDSPMHandler_NoAuth(t *testing.T) {
	h := NewDSPMHandler(nil)

	cases := []struct {
		name   string
		method string
		invoke func(w http.ResponseWriter, r *http.Request)
		body   []byte
	}{
		{"ListDataAssets", "GET", h.ListDataAssets, nil},
		{"TriggerScan", "POST", h.TriggerScan, nil},
		{"ListScans", "GET", h.ListScans, nil},
		{"Classification", "GET", h.Classification, nil},
		{"Exposure", "GET", h.Exposure, nil},
		{"Dependencies", "GET", h.Dependencies, nil},
		{"Dashboard", "GET", h.Dashboard, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var body *bytes.Buffer
			if tc.body != nil {
				body = bytes.NewBuffer(tc.body)
			} else {
				body = &bytes.Buffer{}
			}
			r := httptest.NewRequest(tc.method, "/cyber/dspm", body)
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

// ---- writeError mapping -------------------------------------------------

func TestDSPMWriteError(t *testing.T) {
	h := &DSPMHandler{}

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
			err:        fmt.Errorf("asset not found: %w", repository.ErrNotFound),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "generic error maps to 500",
			err:        fmt.Errorf("scan engine unavailable"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "wrapped generic error maps to 500",
			err:        fmt.Errorf("classify: %w", fmt.Errorf("NLP model failed")),
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

// ---- parseDSPMAssetListParams -------------------------------------------

func TestParseDSPMAssetListParams_Defaults(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/dspm/data-assets", nil)
	params, err := parseDSPMAssetListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Page != 1 {
		t.Errorf("default Page: got %d, want 1", params.Page)
	}
	if params.PerPage != 50 {
		t.Errorf("default PerPage: got %d, want 50", params.PerPage)
	}
	if params.Sort != "risk_score" {
		t.Errorf("default Sort: got %q, want %q", params.Sort, "risk_score")
	}
	if params.Order != "desc" {
		t.Errorf("default Order: got %q, want %q", params.Order, "desc")
	}
}

func TestParseDSPMAssetListParams_ClassificationFilter(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/dspm/data-assets?classification=confidential", nil)
	params, err := parseDSPMAssetListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Classification == nil || *params.Classification != "confidential" {
		t.Errorf("expected classification=%q, got %v", "confidential", params.Classification)
	}
}

func TestParseDSPMAssetListParams_InvalidSort(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/dspm/data-assets?sort=badfield", nil)
	_, err := parseDSPMAssetListParams(r)
	if err == nil {
		t.Error("expected error for invalid sort field, got nil")
	}
}

func TestParseDSPMAssetListParams_SearchParam(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/dspm/data-assets?search=customer+data", nil)
	params, err := parseDSPMAssetListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Search == nil || *params.Search != "customer data" {
		t.Errorf("expected search=%q, got %v", "customer data", params.Search)
	}
}

func TestParseDSPMAssetListParams_ContainsPII(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/dspm/data-assets?contains_pii=true", nil)
	params, err := parseDSPMAssetListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.ContainsPII == nil || !*params.ContainsPII {
		t.Errorf("expected contains_pii=true, got %v", params.ContainsPII)
	}
}

// ---- ListDataAssets handler tests ----------------------------------------

func TestListDataAssets_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	assetID := uuid.New()

	mock := &mockDSPMService{
		listDataAssetsFn: func(ctx context.Context, tid uuid.UUID, params *dto.DSPMAssetListParams) (*dto.DSPMAssetListResponse, error) {
			if tid != tenantID {
				t.Errorf("expected tenantID %s, got %s", tenantID, tid)
			}
			return &dto.DSPMAssetListResponse{
				Data: []*model.DSPMDataAsset{
					{ID: assetID, TenantID: tenantID, DataClassification: "internal"},
				},
				Total:      1,
				Page:       1,
				PerPage:    50,
				TotalPages: 1,
			}, nil
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dspm/data-assets", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.ListDataAssets(w, r)

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

func TestListDataAssets_WithFilters(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockDSPMService{
		listDataAssetsFn: func(ctx context.Context, tid uuid.UUID, params *dto.DSPMAssetListParams) (*dto.DSPMAssetListResponse, error) {
			if params.Classification == nil || *params.Classification != "confidential" {
				t.Errorf("expected classification=confidential, got %v", params.Classification)
			}
			if params.ContainsPII == nil || !*params.ContainsPII {
				t.Errorf("expected contains_pii=true, got %v", params.ContainsPII)
			}
			if params.MinRiskScore == nil || *params.MinRiskScore != 75.0 {
				t.Errorf("expected min_risk_score=75.0, got %v", params.MinRiskScore)
			}
			if params.Sort != "sensitivity_score" {
				t.Errorf("expected sort=sensitivity_score, got %s", params.Sort)
			}
			if params.Order != "asc" {
				t.Errorf("expected order=asc, got %s", params.Order)
			}
			return &dto.DSPMAssetListResponse{
				Data:       []*model.DSPMDataAsset{},
				Total:      0,
				Page:       1,
				PerPage:    50,
				TotalPages: 0,
			}, nil
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dspm/data-assets?classification=confidential&contains_pii=true&min_risk_score=75&sort=sensitivity_score&order=asc", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.ListDataAssets(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

func TestListDataAssets_InvalidParams(t *testing.T) {
	mock := &mockDSPMService{}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtx()
	r := httptest.NewRequest("GET", "/cyber/dspm/data-assets?sort=hacker_score", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.ListDataAssets(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

// ---- GetDataAsset handler tests ------------------------------------------

func TestGetDataAsset_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	assetID := uuid.New()

	mock := &mockDSPMService{
		getDataAssetFn: func(ctx context.Context, tid, aid uuid.UUID) (*model.DSPMDataAsset, error) {
			if tid != tenantID {
				t.Errorf("expected tenantID %s, got %s", tenantID, tid)
			}
			if aid != assetID {
				t.Errorf("expected assetID %s, got %s", assetID, aid)
			}
			return &model.DSPMDataAsset{
				ID:                 assetID,
				TenantID:           tenantID,
				DataClassification: "confidential",
				RiskScore:          85.5,
			}, nil
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := dspmRequestWithID("GET", "/cyber/dspm/data-assets/"+assetID.String(), nil, ctx, assetID.String())
	w := httptest.NewRecorder()

	h.GetDataAsset(w, r)

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

func TestGetDataAsset_NotFound(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	assetID := uuid.New()

	mock := &mockDSPMService{
		getDataAssetFn: func(ctx context.Context, tid, aid uuid.UUID) (*model.DSPMDataAsset, error) {
			return nil, repository.ErrNotFound
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := dspmRequestWithID("GET", "/cyber/dspm/data-assets/"+assetID.String(), nil, ctx, assetID.String())
	w := httptest.NewRecorder()

	h.GetDataAsset(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

func TestGetDataAsset_InvalidID(t *testing.T) {
	mock := &mockDSPMService{}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtx()
	r := dspmRequestWithID("GET", "/cyber/dspm/data-assets/not-a-uuid", nil, ctx, "not-a-uuid")
	w := httptest.NewRecorder()

	h.GetDataAsset(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

// ---- TriggerScan handler tests -------------------------------------------

func TestTriggerScan_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	scanID := uuid.New()
	now := time.Now()

	mock := &mockDSPMService{
		triggerScanFn: func(ctx context.Context, tid, uid uuid.UUID, actor *service.Actor, req *dto.DSPMScanTriggerRequest) (*model.DSPMScan, error) {
			if tid != tenantID {
				t.Errorf("expected tenantID %s, got %s", tenantID, tid)
			}
			if uid != userID {
				t.Errorf("expected userID %s, got %s", userID, uid)
			}
			return &model.DSPMScan{
				ID:        scanID,
				TenantID:  tenantID,
				Status:    "running",
				StartedAt: now,
				CreatedBy: userID,
				CreatedAt: now,
			}, nil
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("POST", "/cyber/dspm/scans/trigger", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.TriggerScan(w, r)

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

func TestTriggerScan_ServiceError(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockDSPMService{
		triggerScanFn: func(ctx context.Context, tid, uid uuid.UUID, actor *service.Actor, req *dto.DSPMScanTriggerRequest) (*model.DSPMScan, error) {
			return nil, fmt.Errorf("scan engine unavailable")
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("POST", "/cyber/dspm/scans/trigger", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.TriggerScan(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

// ---- ListScans handler tests ---------------------------------------------

func TestListScans_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	scanID := uuid.New()
	now := time.Now()

	mock := &mockDSPMService{
		listScansFn: func(ctx context.Context, tid uuid.UUID, params *dto.DSPMScanListParams) (*dto.DSPMScanListResponse, error) {
			if tid != tenantID {
				t.Errorf("expected tenantID %s, got %s", tenantID, tid)
			}
			return &dto.DSPMScanListResponse{
				Data: []*model.DSPMScan{
					{ID: scanID, TenantID: tenantID, Status: "completed", StartedAt: now, CreatedBy: userID, CreatedAt: now},
				},
				Total:      1,
				Page:       1,
				PerPage:    20,
				TotalPages: 1,
			}, nil
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dspm/scans", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.ListScans(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

func TestListScans_WithStatusFilter(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockDSPMService{
		listScansFn: func(ctx context.Context, tid uuid.UUID, params *dto.DSPMScanListParams) (*dto.DSPMScanListResponse, error) {
			if params.Status == nil || *params.Status != "running" {
				t.Errorf("expected status filter 'running', got %v", params.Status)
			}
			return &dto.DSPMScanListResponse{
				Data:       []*model.DSPMScan{},
				Total:      0,
				Page:       1,
				PerPage:    20,
				TotalPages: 0,
			}, nil
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dspm/scans?status=running", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.ListScans(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

func TestListScans_InvalidStatus(t *testing.T) {
	mock := &mockDSPMService{}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtx()
	r := httptest.NewRequest("GET", "/cyber/dspm/scans?status=invalid_status", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.ListScans(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid status filter, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

// ---- GetScan handler tests -----------------------------------------------

func TestGetScan_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	scanID := uuid.New()

	mock := &mockDSPMService{
		getScanFn: func(ctx context.Context, tid, sid uuid.UUID) (*model.DSPMScanResult, error) {
			if tid != tenantID {
				t.Errorf("expected tenantID %s, got %s", tenantID, tid)
			}
			if sid != scanID {
				t.Errorf("expected scanID %s, got %s", scanID, sid)
			}
			return &model.DSPMScanResult{
				Scan: &model.DSPMScan{
					ID:       scanID,
					TenantID: tenantID,
					Status:   "completed",
				},
				AssetsScanned:  10,
				PIIAssetsFound: 3,
				HighRiskFound:  1,
				FindingsCount:  7,
			}, nil
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := dspmRequestWithID("GET", "/cyber/dspm/scans/"+scanID.String(), nil, ctx, scanID.String())
	w := httptest.NewRecorder()

	h.GetScan(w, r)

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

func TestGetScan_NotFound(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	scanID := uuid.New()

	mock := &mockDSPMService{
		getScanFn: func(ctx context.Context, tid, sid uuid.UUID) (*model.DSPMScanResult, error) {
			return nil, repository.ErrNotFound
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := dspmRequestWithID("GET", "/cyber/dspm/scans/"+scanID.String(), nil, ctx, scanID.String())
	w := httptest.NewRecorder()

	h.GetScan(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

func TestGetScan_InvalidID(t *testing.T) {
	mock := &mockDSPMService{}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtx()
	r := dspmRequestWithID("GET", "/cyber/dspm/scans/not-a-uuid", nil, ctx, "not-a-uuid")
	w := httptest.NewRecorder()

	h.GetScan(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

// ---- Classification handler tests ----------------------------------------

func TestClassification_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockDSPMService{
		classificationSummaryFn: func(ctx context.Context, tid uuid.UUID) (*model.DSPMClassificationSummary, error) {
			if tid != tenantID {
				t.Errorf("expected tenantID %s, got %s", tenantID, tid)
			}
			return &model.DSPMClassificationSummary{
				Public:       5,
				Internal:     10,
				Confidential: 3,
				Restricted:   2,
				Total:        20,
			}, nil
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dspm/classification", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.Classification(w, r)

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

func TestClassification_ServiceError(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockDSPMService{
		classificationSummaryFn: func(ctx context.Context, tid uuid.UUID) (*model.DSPMClassificationSummary, error) {
			return nil, fmt.Errorf("database connection refused")
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dspm/classification", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.Classification(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

// ---- Exposure handler tests ----------------------------------------------

func TestExposure_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockDSPMService{
		exposureAnalysisFn: func(ctx context.Context, tid uuid.UUID) (*model.DSPMExposureAnalysis, error) {
			if tid != tenantID {
				t.Errorf("expected tenantID %s, got %s", tenantID, tid)
			}
			return &model.DSPMExposureAnalysis{
				InternalOnly:      12,
				VPNAccessible:     5,
				InternetFacing:    3,
				Unknown:           1,
				CriticalExposures: []model.DSPMDataAsset{},
			}, nil
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dspm/exposure", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.Exposure(w, r)

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

// ---- Dependencies handler tests ------------------------------------------

func TestDependencies_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	assetID := uuid.New()

	mock := &mockDSPMService{
		dependenciesFn: func(ctx context.Context, tid uuid.UUID) ([]model.DSPMDependencyNode, error) {
			if tid != tenantID {
				t.Errorf("expected tenantID %s, got %s", tenantID, tid)
			}
			return []model.DSPMDependencyNode{
				{
					AssetID:        assetID,
					AssetName:      "customer-db",
					AssetType:      "database",
					Classification: "confidential",
					RiskScore:      78.5,
					ConsumerCount:  3,
					ProducerCount:  1,
					Dependencies:   []model.DSPMDependencyEdge{},
				},
			}, nil
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dspm/dependencies", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.Dependencies(w, r)

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

// ---- Dashboard handler tests ---------------------------------------------

func TestDashboard_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockDSPMService{
		dashboardFn: func(ctx context.Context, tid uuid.UUID) (*model.DSPMDashboard, error) {
			if tid != tenantID {
				t.Errorf("expected tenantID %s, got %s", tenantID, tid)
			}
			return &model.DSPMDashboard{
				TotalDataAssets:         25,
				PIIAssetsCount:          8,
				HighRiskAssetsCount:     4,
				AvgPostureScore:         72.5,
				AvgRiskScore:            65.0,
				ClassificationBreakdown: map[string]int{"confidential": 10, "internal": 15},
				ExposureBreakdown:       map[string]int{"internal_only": 20, "internet_facing": 5},
				TopRiskyAssets:          []model.DSPMDataAsset{},
				RecentScans:             []model.DSPMScan{},
				PIITypeFrequency:        map[string]int{"email": 5, "ssn": 3},
			}, nil
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dspm/dashboard", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.Dashboard(w, r)

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

func TestDashboard_ServiceError(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockDSPMService{
		dashboardFn: func(ctx context.Context, tid uuid.UUID) (*model.DSPMDashboard, error) {
			return nil, fmt.Errorf("aggregation pipeline failed")
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dspm/dashboard", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.Dashboard(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

// ---- Additional parseDSPMAssetListParams edge-case tests ------------------

func TestParseDSPMAssetListParams_InvalidContainsPII(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/dspm/data-assets?contains_pii=maybe", nil)
	_, err := parseDSPMAssetListParams(r)
	if err == nil {
		t.Error("expected error for invalid contains_pii value, got nil")
	}
}

func TestParseDSPMAssetListParams_MinRiskScore(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/dspm/data-assets?min_risk_score=75.5", nil)
	params, err := parseDSPMAssetListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.MinRiskScore == nil || *params.MinRiskScore != 75.5 {
		t.Errorf("expected min_risk_score=75.5, got %v", params.MinRiskScore)
	}
}

func TestParseDSPMAssetListParams_InvalidMinRiskScore(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/dspm/data-assets?min_risk_score=high", nil)
	_, err := parseDSPMAssetListParams(r)
	if err == nil {
		t.Error("expected error for non-numeric min_risk_score, got nil")
	}
}

func TestParseDSPMAssetListParams_InvalidAssetID(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/dspm/data-assets?asset_id=not-a-uuid", nil)
	_, err := parseDSPMAssetListParams(r)
	if err == nil {
		t.Error("expected error for invalid asset_id UUID, got nil")
	}
}

func TestParseDSPMAssetListParams_ExplicitPageAndPerPage(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/dspm/data-assets?page=2&per_page=25", nil)
	params, err := parseDSPMAssetListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Page != 2 {
		t.Errorf("expected page=2, got %d", params.Page)
	}
	if params.PerPage != 25 {
		t.Errorf("expected per_page=25, got %d", params.PerPage)
	}
}

func TestParseDSPMAssetListParams_NetworkExposure(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/dspm/data-assets?network_exposure=public", nil)
	params, err := parseDSPMAssetListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.NetworkExposure == nil || *params.NetworkExposure != "public" {
		t.Errorf("expected network_exposure=%q, got %v", "public", params.NetworkExposure)
	}
}

// ---- ListDataAssets service error test -----------------------------------

func TestListDataAssets_ServiceError(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockDSPMService{
		listDataAssetsFn: func(ctx context.Context, tid uuid.UUID, params *dto.DSPMAssetListParams) (*dto.DSPMAssetListResponse, error) {
			return nil, fmt.Errorf("database timeout")
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dspm/data-assets", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.ListDataAssets(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

// ---- GetDataAsset service error test ------------------------------------

func TestGetDataAsset_ServiceError(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	assetID := uuid.New()

	mock := &mockDSPMService{
		getDataAssetFn: func(ctx context.Context, tid, aid uuid.UUID) (*model.DSPMDataAsset, error) {
			return nil, fmt.Errorf("unexpected db error")
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := dspmRequestWithID("GET", "/cyber/dspm/data-assets/"+assetID.String(), nil, ctx, assetID.String())
	w := httptest.NewRecorder()

	h.GetDataAsset(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

// ---- Exposure service error test ----------------------------------------

func TestExposure_ServiceError(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockDSPMService{
		exposureAnalysisFn: func(ctx context.Context, tid uuid.UUID) (*model.DSPMExposureAnalysis, error) {
			return nil, fmt.Errorf("analysis engine down")
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dspm/exposure", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.Exposure(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

// ---- Dependencies service error test ------------------------------------

func TestDependencies_ServiceError(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockDSPMService{
		dependenciesFn: func(ctx context.Context, tid uuid.UUID) ([]model.DSPMDependencyNode, error) {
			return nil, fmt.Errorf("graph query timeout")
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dspm/dependencies", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.Dependencies(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

// ---- ListScans service error test ----------------------------------------

func TestListScans_ServiceError(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockDSPMService{
		listScansFn: func(ctx context.Context, tid uuid.UUID, params *dto.DSPMScanListParams) (*dto.DSPMScanListResponse, error) {
			return nil, fmt.Errorf("scan repository unavailable")
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := httptest.NewRequest("GET", "/cyber/dspm/scans", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	h.ListScans(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}

// ---- GetScan service error test ------------------------------------------

func TestGetScan_ServiceError(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	scanID := uuid.New()

	mock := &mockDSPMService{
		getScanFn: func(ctx context.Context, tid, sid uuid.UUID) (*model.DSPMScanResult, error) {
			return nil, fmt.Errorf("unexpected error")
		},
	}
	h := NewDSPMHandler(mock)

	ctx := dspmAuthCtxWithIDs(tenantID, userID)
	r := dspmRequestWithID("GET", "/cyber/dspm/scans/"+scanID.String(), nil, ctx, scanID.String())
	w := httptest.NewRecorder()

	h.GetScan(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
}
