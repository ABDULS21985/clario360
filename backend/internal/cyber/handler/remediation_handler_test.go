package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clario360/platform/internal/cyber/remediation"
	"github.com/clario360/platform/internal/cyber/repository"
)

// ---- auth-free 401/403 tests -------------------------------------------

func TestRemediationHandler_NoAuth(t *testing.T) {
	h := NewRemediationHandler(nil)

	cases := []struct {
		name   string
		method string
		invoke func(w http.ResponseWriter, r *http.Request)
		body   []byte
	}{
		{"List", "GET", h.List, nil},
		{"Create", "POST", h.Create, []byte(`{"title":"t"}`)},
		{"Stats", "GET", h.Stats, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var body *bytes.Buffer
			if tc.body != nil {
				body = bytes.NewBuffer(tc.body)
			} else {
				body = &bytes.Buffer{}
			}
			r := httptest.NewRequest(tc.method, "/cyber/remediation", body)
			w := httptest.NewRecorder()
			tc.invoke(w, r)
			// Without auth context requireTenantAndUser writes 403 (missing tenant)
			// or 401 (missing user). Either way >= 400.
			if w.Code < 400 {
				t.Errorf("%s: expected 4xx without auth, got %d", tc.name, w.Code)
			}
		})
	}
}

// ---- writeError mapping -------------------------------------------------

func TestRemediationWriteError(t *testing.T) {
	h := &RemediationHandler{}

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
			err:        fmt.Errorf("outer: %w", repository.ErrNotFound),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "ErrPreConditionFailed maps to 400",
			err:        fmt.Errorf("dry-run required: %w", remediation.ErrPreConditionFailed),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "ErrInvalidTransition maps to 400",
			err:        fmt.Errorf("cannot transition: %w", remediation.ErrInvalidTransition),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "ErrInsufficientPermission maps to 403",
			err:        fmt.Errorf("role denied: %w", remediation.ErrInsufficientPermission),
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "generic error maps to 500",
			err:        fmt.Errorf("database connection refused"),
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

// ---- parseRemediationListParams -----------------------------------------

func TestParseRemediationListParams_Defaults(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/remediation", nil)
	params, err := parseRemediationListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Page != 1 {
		t.Errorf("default Page: got %d, want 1", params.Page)
	}
	// SetDefaults caps PerPage to 50 when not provided.
	if params.PerPage != 50 {
		t.Errorf("default PerPage: got %d, want 50", params.PerPage)
	}
	if params.Sort != "created_at" {
		t.Errorf("default Sort: got %q, want %q", params.Sort, "created_at")
	}
	if params.Order != "desc" {
		t.Errorf("default Order: got %q, want %q", params.Order, "desc")
	}
}

func TestParseRemediationListParams_StatusFilter(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/remediation?status=draft&status=approved", nil)
	params, err := parseRemediationListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(params.Statuses) != 2 {
		t.Errorf("expected 2 statuses, got %d: %v", len(params.Statuses), params.Statuses)
	}
}

func TestParseRemediationListParams_SeverityFilter(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/remediation?severity=critical&severity=high", nil)
	params, err := parseRemediationListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(params.Severities) != 2 {
		t.Errorf("expected 2 severities, got %d: %v", len(params.Severities), params.Severities)
	}
}

func TestParseRemediationListParams_SearchParam(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/remediation?search=patch+openssl", nil)
	params, err := parseRemediationListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Search == nil || *params.Search != "patch openssl" {
		t.Errorf("expected search=%q, got %v", "patch openssl", params.Search)
	}
}

func TestParseRemediationListParams_InvalidAssetID(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/remediation?asset_id=not-a-uuid", nil)
	_, err := parseRemediationListParams(r)
	if err == nil {
		t.Error("expected error for invalid asset_id UUID, got nil")
	}
}

func TestParseRemediationListParams_InvalidAlertID(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/remediation?alert_id=bad", nil)
	_, err := parseRemediationListParams(r)
	if err == nil {
		t.Error("expected error for invalid alert_id UUID, got nil")
	}
}

func TestParseRemediationListParams_InvalidVulnID(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/remediation?vulnerability_id=xyz", nil)
	_, err := parseRemediationListParams(r)
	if err == nil {
		t.Error("expected error for invalid vulnerability_id UUID, got nil")
	}
}

func TestParseRemediationListParams_ExplicitPage(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/remediation?page=3&per_page=10", nil)
	params, err := parseRemediationListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Page != 3 {
		t.Errorf("expected page=3, got %d", params.Page)
	}
	if params.PerPage != 10 {
		t.Errorf("expected per_page=10, got %d", params.PerPage)
	}
}

func TestParseRemediationListParams_InvalidSort(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/remediation?sort=unknown_field", nil)
	_, err := parseRemediationListParams(r)
	if err == nil {
		t.Error("expected error for invalid sort field, got nil")
	}
}

func TestParseRemediationListParams_InvalidOrder(t *testing.T) {
	r := httptest.NewRequest("GET", "/cyber/remediation?order=sideways", nil)
	_, err := parseRemediationListParams(r)
	if err == nil {
		t.Error("expected error for invalid order value, got nil")
	}
}

// ---- remediationRoleFromRequest -----------------------------------------

func TestRemediationRoleFromRequest_NoContext(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	// No auth context → user is nil → should return "viewer".
	role := remediationRoleFromRequest(r)
	if role != "viewer" {
		t.Errorf("expected role %q without auth context, got %q", "viewer", role)
	}
}
