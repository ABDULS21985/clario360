package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clario360/platform/internal/cyber/repository"
)

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
	r := httptest.NewRequest("GET", "/cyber/dspm/data-assets?classification=pii", nil)
	params, err := parseDSPMAssetListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Classification == nil || *params.Classification != "pii" {
		t.Errorf("expected classification=%q, got %v", "pii", params.Classification)
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

func TestParseDSPMAssetListParams_InvalidSort(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/dspm/data-assets?sort=hacker_score", nil)
	_, err := parseDSPMAssetListParams(r)
	if err == nil {
		t.Error("expected error for invalid sort field, got nil")
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
