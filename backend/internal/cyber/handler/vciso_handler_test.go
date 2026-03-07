package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/repository"
)

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
// The handler calls requireTenantAndUser first, so an invalid type in the
// query is only checked after auth. Without auth we still get a 4xx from the
// auth guard — which is the correct behaviour.
func TestVCISOHandler_BriefingHistory_InvalidType_NoAuth(t *testing.T) {
	h := NewVCISOHandler(nil)
	r := httptest.NewRequest("GET", "/cyber/vciso/briefing/history?type=quarterly", nil)
	w := httptest.NewRecorder()
	h.BriefingHistory(w, r)
	if w.Code < 400 {
		t.Errorf("expected 4xx, got %d", w.Code)
	}
}
