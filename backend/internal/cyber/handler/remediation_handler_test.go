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
	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/remediation"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/cyber/service"
)

// ---------------------------------------------------------------------------
// mock + helpers
// ---------------------------------------------------------------------------

type mockRemediationService struct {
	createFn          func(ctx context.Context, tenantID, userID uuid.UUID, actor *service.Actor, req *dto.CreateRemediationRequest) (*model.RemediationAction, error)
	listFn            func(ctx context.Context, tenantID uuid.UUID, params *dto.RemediationListParams) (*dto.RemediationListResponse, error)
	getFn             func(ctx context.Context, tenantID, remediationID uuid.UUID) (*model.RemediationAction, error)
	updateFn          func(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.UpdateRemediationRequest) (*model.RemediationAction, error)
	deleteFn          func(ctx context.Context, tenantID, remediationID uuid.UUID, actor *service.Actor) error
	submitFn          func(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string) (*model.RemediationAction, error)
	approveFn         func(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.ApproveRemediationRequest) (*model.RemediationAction, error)
	rejectFn          func(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.RejectRemediationRequest) (*model.RemediationAction, error)
	requestRevisionFn func(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.RequestRevisionRequest) (*model.RemediationAction, error)
	dryRunFn          func(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string) (*model.DryRunResult, error)
	getDryRunFn       func(ctx context.Context, tenantID, remediationID uuid.UUID) (*model.DryRunResult, error)
	executeFn         func(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.ExecuteRemediationRequest) (*model.RemediationAction, error)
	verifyFn          func(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.VerifyRemediationRequest) (*model.RemediationAction, error)
	rollbackFn        func(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.RollbackRequest) (*model.RemediationAction, error)
	closeFn           func(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string) (*model.RemediationAction, error)
	auditTrailFn      func(ctx context.Context, tenantID, remediationID uuid.UUID) ([]model.RemediationAuditEntry, error)
	statsFn           func(ctx context.Context, tenantID uuid.UUID) (*model.RemediationStats, error)
}

func (m *mockRemediationService) Create(ctx context.Context, tenantID, userID uuid.UUID, actor *service.Actor, req *dto.CreateRemediationRequest) (*model.RemediationAction, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, userID, actor, req)
	}
	return nil, nil
}

func (m *mockRemediationService) List(ctx context.Context, tenantID uuid.UUID, params *dto.RemediationListParams) (*dto.RemediationListResponse, error) {
	if m.listFn != nil {
		return m.listFn(ctx, tenantID, params)
	}
	return nil, nil
}

func (m *mockRemediationService) Get(ctx context.Context, tenantID, remediationID uuid.UUID) (*model.RemediationAction, error) {
	if m.getFn != nil {
		return m.getFn(ctx, tenantID, remediationID)
	}
	return nil, nil
}

func (m *mockRemediationService) Update(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.UpdateRemediationRequest) (*model.RemediationAction, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, tenantID, remediationID, actorID, actorName, actorRole, req)
	}
	return nil, nil
}

func (m *mockRemediationService) Delete(ctx context.Context, tenantID, remediationID uuid.UUID, actor *service.Actor) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, tenantID, remediationID, actor)
	}
	return nil
}

func (m *mockRemediationService) Submit(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string) (*model.RemediationAction, error) {
	if m.submitFn != nil {
		return m.submitFn(ctx, tenantID, remediationID, actorID, actorName, actorRole)
	}
	return nil, nil
}

func (m *mockRemediationService) Approve(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.ApproveRemediationRequest) (*model.RemediationAction, error) {
	if m.approveFn != nil {
		return m.approveFn(ctx, tenantID, remediationID, actorID, actorName, actorRole, req)
	}
	return nil, nil
}

func (m *mockRemediationService) Reject(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.RejectRemediationRequest) (*model.RemediationAction, error) {
	if m.rejectFn != nil {
		return m.rejectFn(ctx, tenantID, remediationID, actorID, actorName, actorRole, req)
	}
	return nil, nil
}

func (m *mockRemediationService) RequestRevision(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.RequestRevisionRequest) (*model.RemediationAction, error) {
	if m.requestRevisionFn != nil {
		return m.requestRevisionFn(ctx, tenantID, remediationID, actorID, actorName, actorRole, req)
	}
	return nil, nil
}

func (m *mockRemediationService) DryRun(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string) (*model.DryRunResult, error) {
	if m.dryRunFn != nil {
		return m.dryRunFn(ctx, tenantID, remediationID, actorID, actorName, actorRole)
	}
	return nil, nil
}

func (m *mockRemediationService) GetDryRun(ctx context.Context, tenantID, remediationID uuid.UUID) (*model.DryRunResult, error) {
	if m.getDryRunFn != nil {
		return m.getDryRunFn(ctx, tenantID, remediationID)
	}
	return nil, nil
}

func (m *mockRemediationService) Execute(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.ExecuteRemediationRequest) (*model.RemediationAction, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, tenantID, remediationID, actorID, actorName, actorRole, req)
	}
	return nil, nil
}

func (m *mockRemediationService) Verify(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.VerifyRemediationRequest) (*model.RemediationAction, error) {
	if m.verifyFn != nil {
		return m.verifyFn(ctx, tenantID, remediationID, actorID, actorName, actorRole, req)
	}
	return nil, nil
}

func (m *mockRemediationService) Rollback(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.RollbackRequest) (*model.RemediationAction, error) {
	if m.rollbackFn != nil {
		return m.rollbackFn(ctx, tenantID, remediationID, actorID, actorName, actorRole, req)
	}
	return nil, nil
}

func (m *mockRemediationService) Close(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string) (*model.RemediationAction, error) {
	if m.closeFn != nil {
		return m.closeFn(ctx, tenantID, remediationID, actorID, actorName, actorRole)
	}
	return nil, nil
}

func (m *mockRemediationService) AuditTrail(ctx context.Context, tenantID, remediationID uuid.UUID) ([]model.RemediationAuditEntry, error) {
	if m.auditTrailFn != nil {
		return m.auditTrailFn(ctx, tenantID, remediationID)
	}
	return nil, nil
}

func (m *mockRemediationService) Stats(ctx context.Context, tenantID uuid.UUID) (*model.RemediationStats, error) {
	if m.statsFn != nil {
		return m.statsFn(ctx, tenantID)
	}
	return nil, nil
}

// authRequest creates an *http.Request with tenantID + userID injected into the context.
func authRequest(method, path string, body []byte) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewBuffer(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
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

// authRequestWithID creates an *http.Request with auth context AND chi URL param "id".
func authRequestWithID(method, path string, id uuid.UUID, body []byte) *http.Request {
	r := authRequest(method, path, body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	return r.WithContext(ctx)
}

// sampleAction returns a *model.RemediationAction with reasonable test data.
func sampleAction() *model.RemediationAction {
	now := time.Now()
	creatorName := "admin@clario.dev"
	return &model.RemediationAction{
		ID:          uuid.MustParse("00000000-0000-0000-0000-000000000010"),
		TenantID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Type:        model.RemediationTypePatch,
		Severity:    "high",
		Title:       "Patch OpenSSL CVE-2024-1234",
		Description: "Apply the latest OpenSSL security patch to all affected servers",
		Plan: model.RemediationPlan{
			Steps: []model.RemediationStep{
				{Number: 1, Action: "download", Description: "Download patch from vendor"},
				{Number: 2, Action: "apply", Description: "Apply patch to servers"},
			},
			Reversible: true,
		},
		AffectedAssetIDs:   []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-000000000020")},
		AffectedAssetCount: 1,
		ExecutionMode:      "automatic",
		Status:             model.StatusDraft,
		CreatedBy:          uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		CreatedByName:      &creatorName,
		CreatedAt:          now,
		UpdatedAt:          now,
		Tags:               []string{"openssl", "cve"},
	}
}

// ---------------------------------------------------------------------------
// 1. Auth enforcement (3 tests)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// 2. writeError mapping (6 tests)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// 3. parseRemediationListParams (9 tests)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// 4. remediationRoleFromRequest (1 test)
// ---------------------------------------------------------------------------

func TestRemediationRoleFromRequest_NoContext(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	// No auth context -> user is nil -> should return "viewer".
	role := remediationRoleFromRequest(r)
	if role != "viewer" {
		t.Errorf("expected role %q without auth context, got %q", "viewer", role)
	}
}

// ---------------------------------------------------------------------------
// 5. Happy-path handler tests (35 tests)
// ---------------------------------------------------------------------------

// ---- Create ---------------------------------------------------------------

func TestCreate_Success(t *testing.T) {
	action := sampleAction()
	mock := &mockRemediationService{
		createFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor, _ *dto.CreateRemediationRequest) (*model.RemediationAction, error) {
			return action, nil
		},
	}
	h := NewRemediationHandler(mock)

	body, _ := json.Marshal(dto.CreateRemediationRequest{
		Type:     "patch",
		Severity: "high",
		Title:    "Patch OpenSSL",
		Plan: model.RemediationPlan{
			Steps: []model.RemediationStep{{Number: 1, Action: "apply", Description: "apply patch"}},
		},
	})

	r := authRequest("POST", "/cyber/remediation", body)
	w := httptest.NewRecorder()
	h.Create(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Error("expected response to contain 'data' key")
	}
}

func TestCreate_InvalidJSON(t *testing.T) {
	h := NewRemediationHandler(&mockRemediationService{})
	r := authRequest("POST", "/cyber/remediation", []byte(`{not json`))
	w := httptest.NewRecorder()
	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_ServiceError(t *testing.T) {
	mock := &mockRemediationService{
		createFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor, _ *dto.CreateRemediationRequest) (*model.RemediationAction, error) {
			return nil, fmt.Errorf("database unavailable")
		},
	}
	h := NewRemediationHandler(mock)

	body, _ := json.Marshal(dto.CreateRemediationRequest{
		Type:  "patch",
		Title: "x",
		Plan: model.RemediationPlan{
			Steps: []model.RemediationStep{{Number: 1, Action: "a", Description: "d"}},
		},
	})
	r := authRequest("POST", "/cyber/remediation", body)
	w := httptest.NewRecorder()
	h.Create(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// ---- List -----------------------------------------------------------------

func TestList_Success(t *testing.T) {
	mock := &mockRemediationService{
		listFn: func(_ context.Context, _ uuid.UUID, _ *dto.RemediationListParams) (*dto.RemediationListResponse, error) {
			return &dto.RemediationListResponse{
				Data:       []*model.RemediationAction{sampleAction()},
				Total:      1,
				Page:       1,
				PerPage:    50,
				TotalPages: 1,
			}, nil
		},
	}
	h := NewRemediationHandler(mock)

	r := authRequest("GET", "/cyber/remediation", nil)
	w := httptest.NewRecorder()
	h.List(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Error("expected response to contain 'data' key")
	}
}

func TestList_WithFilters(t *testing.T) {
	var capturedParams *dto.RemediationListParams
	mock := &mockRemediationService{
		listFn: func(_ context.Context, _ uuid.UUID, params *dto.RemediationListParams) (*dto.RemediationListResponse, error) {
			capturedParams = params
			return &dto.RemediationListResponse{
				Data:       []*model.RemediationAction{},
				Total:      0,
				Page:       1,
				PerPage:    50,
				TotalPages: 0,
			}, nil
		},
	}
	h := NewRemediationHandler(mock)

	r := authRequest("GET", "/cyber/remediation?status=draft&severity=high&search=openssl", nil)
	w := httptest.NewRecorder()
	h.List(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	if capturedParams == nil {
		t.Fatal("service.List was not called")
	}
	if len(capturedParams.Statuses) != 1 || capturedParams.Statuses[0] != "draft" {
		t.Errorf("expected statuses=[draft], got %v", capturedParams.Statuses)
	}
	if len(capturedParams.Severities) != 1 || capturedParams.Severities[0] != "high" {
		t.Errorf("expected severities=[high], got %v", capturedParams.Severities)
	}
	if capturedParams.Search == nil || *capturedParams.Search != "openssl" {
		t.Errorf("expected search=openssl, got %v", capturedParams.Search)
	}
}

func TestList_InvalidParams(t *testing.T) {
	h := NewRemediationHandler(&mockRemediationService{})

	r := authRequest("GET", "/cyber/remediation?sort=bad", nil)
	w := httptest.NewRecorder()
	h.List(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid sort, got %d", w.Code)
	}
}

// ---- Get ------------------------------------------------------------------

func TestGet_Success(t *testing.T) {
	action := sampleAction()
	mock := &mockRemediationService{
		getFn: func(_ context.Context, _, _ uuid.UUID) (*model.RemediationAction, error) {
			return action, nil
		},
	}
	h := NewRemediationHandler(mock)

	r := authRequestWithID("GET", "/cyber/remediation/"+action.ID.String(), action.ID, nil)
	w := httptest.NewRecorder()
	h.Get(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Error("expected response to contain 'data' key")
	}
}

func TestGet_NotFound(t *testing.T) {
	mock := &mockRemediationService{
		getFn: func(_ context.Context, _, _ uuid.UUID) (*model.RemediationAction, error) {
			return nil, repository.ErrNotFound
		},
	}
	h := NewRemediationHandler(mock)
	id := uuid.New()
	r := authRequestWithID("GET", "/cyber/remediation/"+id.String(), id, nil)
	w := httptest.NewRecorder()
	h.Get(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGet_InvalidID(t *testing.T) {
	h := NewRemediationHandler(&mockRemediationService{})

	r := authRequest("GET", "/cyber/remediation/not-a-uuid", nil)
	// Set up chi route context with invalid UUID.
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "not-a-uuid")
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	h.Get(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid UUID, got %d", w.Code)
	}
}

// ---- Update ---------------------------------------------------------------

func TestUpdate_Success(t *testing.T) {
	action := sampleAction()
	mock := &mockRemediationService{
		updateFn: func(_ context.Context, _, _, _ uuid.UUID, _, _ string, _ *dto.UpdateRemediationRequest) (*model.RemediationAction, error) {
			return action, nil
		},
	}
	h := NewRemediationHandler(mock)

	newTitle := "Updated Title"
	body, _ := json.Marshal(dto.UpdateRemediationRequest{Title: &newTitle})
	r := authRequestWithID("PUT", "/cyber/remediation/"+action.ID.String(), action.ID, body)
	w := httptest.NewRecorder()
	h.Update(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Error("expected response to contain 'data' key")
	}
}

func TestUpdate_NotFound(t *testing.T) {
	mock := &mockRemediationService{
		updateFn: func(_ context.Context, _, _, _ uuid.UUID, _, _ string, _ *dto.UpdateRemediationRequest) (*model.RemediationAction, error) {
			return nil, repository.ErrNotFound
		},
	}
	h := NewRemediationHandler(mock)
	id := uuid.New()
	body, _ := json.Marshal(dto.UpdateRemediationRequest{})
	r := authRequestWithID("PUT", "/cyber/remediation/"+id.String(), id, body)
	w := httptest.NewRecorder()
	h.Update(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestUpdate_InvalidJSON(t *testing.T) {
	h := NewRemediationHandler(&mockRemediationService{})
	id := uuid.New()
	r := authRequestWithID("PUT", "/cyber/remediation/"+id.String(), id, []byte(`{broken`))
	w := httptest.NewRecorder()
	h.Update(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ---- Delete ---------------------------------------------------------------

func TestDelete_Success(t *testing.T) {
	mock := &mockRemediationService{
		deleteFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor) error {
			return nil
		},
	}
	h := NewRemediationHandler(mock)
	id := uuid.New()
	r := authRequestWithID("DELETE", "/cyber/remediation/"+id.String(), id, nil)
	w := httptest.NewRecorder()
	h.Delete(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected response 'data' to be a JSON object")
	}
	if deleted, ok := data["deleted"].(bool); !ok || !deleted {
		t.Errorf("expected {\"deleted\":true}, got %v", data)
	}
}

func TestDelete_NotFound(t *testing.T) {
	mock := &mockRemediationService{
		deleteFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor) error {
			return repository.ErrNotFound
		},
	}
	h := NewRemediationHandler(mock)
	id := uuid.New()
	r := authRequestWithID("DELETE", "/cyber/remediation/"+id.String(), id, nil)
	w := httptest.NewRecorder()
	h.Delete(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ---- Submit ---------------------------------------------------------------

func TestSubmit_Success(t *testing.T) {
	action := sampleAction()
	action.Status = model.StatusPendingApproval
	mock := &mockRemediationService{
		submitFn: func(_ context.Context, _, _, _ uuid.UUID, _, _ string) (*model.RemediationAction, error) {
			return action, nil
		},
	}
	h := NewRemediationHandler(mock)
	r := authRequestWithID("POST", "/cyber/remediation/"+action.ID.String()+"/submit", action.ID, nil)
	w := httptest.NewRecorder()
	h.Submit(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Error("expected response to contain 'data' key")
	}
}

func TestSubmit_InvalidTransition(t *testing.T) {
	mock := &mockRemediationService{
		submitFn: func(_ context.Context, _, _, _ uuid.UUID, _, _ string) (*model.RemediationAction, error) {
			return nil, fmt.Errorf("cannot submit: %w", remediation.ErrInvalidTransition)
		},
	}
	h := NewRemediationHandler(mock)
	id := uuid.New()
	r := authRequestWithID("POST", "/cyber/remediation/"+id.String()+"/submit", id, nil)
	w := httptest.NewRecorder()
	h.Submit(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ---- Approve --------------------------------------------------------------

func TestApprove_Success(t *testing.T) {
	action := sampleAction()
	action.Status = model.StatusApproved
	mock := &mockRemediationService{
		approveFn: func(_ context.Context, _, _, _ uuid.UUID, _, _ string, _ *dto.ApproveRemediationRequest) (*model.RemediationAction, error) {
			return action, nil
		},
	}
	h := NewRemediationHandler(mock)

	body, _ := json.Marshal(dto.ApproveRemediationRequest{Notes: "LGTM"})
	r := authRequestWithID("POST", "/cyber/remediation/"+action.ID.String()+"/approve", action.ID, body)
	w := httptest.NewRecorder()
	h.Approve(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Error("expected response to contain 'data' key")
	}
}

func TestApprove_InsufficientRole(t *testing.T) {
	mock := &mockRemediationService{
		approveFn: func(_ context.Context, _, _, _ uuid.UUID, _, _ string, _ *dto.ApproveRemediationRequest) (*model.RemediationAction, error) {
			return nil, fmt.Errorf("role viewer cannot approve: %w", remediation.ErrInsufficientPermission)
		},
	}
	h := NewRemediationHandler(mock)
	id := uuid.New()
	body, _ := json.Marshal(dto.ApproveRemediationRequest{Notes: "ok"})
	r := authRequestWithID("POST", "/cyber/remediation/"+id.String()+"/approve", id, body)
	w := httptest.NewRecorder()
	h.Approve(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// ---- Reject ---------------------------------------------------------------

func TestReject_Success(t *testing.T) {
	action := sampleAction()
	action.Status = model.StatusRejected
	mock := &mockRemediationService{
		rejectFn: func(_ context.Context, _, _, _ uuid.UUID, _, _ string, _ *dto.RejectRemediationRequest) (*model.RemediationAction, error) {
			return action, nil
		},
	}
	h := NewRemediationHandler(mock)

	body, _ := json.Marshal(dto.RejectRemediationRequest{Reason: "plan incomplete"})
	r := authRequestWithID("POST", "/cyber/remediation/"+action.ID.String()+"/reject", action.ID, body)
	w := httptest.NewRecorder()
	h.Reject(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Error("expected response to contain 'data' key")
	}
}

func TestReject_MissingReason(t *testing.T) {
	mock := &mockRemediationService{
		rejectFn: func(_ context.Context, _, _, _ uuid.UUID, _, _ string, _ *dto.RejectRemediationRequest) (*model.RemediationAction, error) {
			return nil, fmt.Errorf("rejection reason is required")
		},
	}
	h := NewRemediationHandler(mock)
	id := uuid.New()
	// Send empty reason -- the handler does not validate, it passes to service.
	body, _ := json.Marshal(dto.RejectRemediationRequest{Reason: ""})
	r := authRequestWithID("POST", "/cyber/remediation/"+id.String()+"/reject", id, body)
	w := httptest.NewRecorder()
	h.Reject(w, r)

	// The handler maps generic errors to 500.
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// ---- RequestRevision ------------------------------------------------------

func TestRequestRevision_Success(t *testing.T) {
	action := sampleAction()
	action.Status = model.StatusRevisionRequested
	mock := &mockRemediationService{
		requestRevisionFn: func(_ context.Context, _, _, _ uuid.UUID, _, _ string, _ *dto.RequestRevisionRequest) (*model.RemediationAction, error) {
			return action, nil
		},
	}
	h := NewRemediationHandler(mock)

	body, _ := json.Marshal(dto.RequestRevisionRequest{Notes: "add rollback steps"})
	r := authRequestWithID("POST", "/cyber/remediation/"+action.ID.String()+"/request-revision", action.ID, body)
	w := httptest.NewRecorder()
	h.RequestRevision(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Error("expected response to contain 'data' key")
	}
}

func TestRequestRevision_ServiceError(t *testing.T) {
	mock := &mockRemediationService{
		requestRevisionFn: func(_ context.Context, _, _, _ uuid.UUID, _, _ string, _ *dto.RequestRevisionRequest) (*model.RemediationAction, error) {
			return nil, fmt.Errorf("unexpected DB error")
		},
	}
	h := NewRemediationHandler(mock)
	id := uuid.New()
	body, _ := json.Marshal(dto.RequestRevisionRequest{Notes: "fix it"})
	r := authRequestWithID("POST", "/cyber/remediation/"+id.String()+"/request-revision", id, body)
	w := httptest.NewRecorder()
	h.RequestRevision(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// ---- DryRun ---------------------------------------------------------------

func TestDryRun_Success(t *testing.T) {
	dryResult := &model.DryRunResult{
		Success:  true,
		Warnings: []string{"service window recommended"},
		Blockers: nil,
	}
	mock := &mockRemediationService{
		dryRunFn: func(_ context.Context, _, _, _ uuid.UUID, _, _ string) (*model.DryRunResult, error) {
			return dryResult, nil
		},
	}
	h := NewRemediationHandler(mock)
	id := uuid.New()
	r := authRequestWithID("POST", "/cyber/remediation/"+id.String()+"/dry-run", id, nil)
	w := httptest.NewRecorder()
	h.DryRun(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Error("expected response to contain 'data' key")
	}
}

func TestDryRun_PreConditionFailed(t *testing.T) {
	mock := &mockRemediationService{
		dryRunFn: func(_ context.Context, _, _, _ uuid.UUID, _, _ string) (*model.DryRunResult, error) {
			return nil, fmt.Errorf("approval required: %w", remediation.ErrPreConditionFailed)
		},
	}
	h := NewRemediationHandler(mock)
	id := uuid.New()
	r := authRequestWithID("POST", "/cyber/remediation/"+id.String()+"/dry-run", id, nil)
	w := httptest.NewRecorder()
	h.DryRun(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ---- GetDryRun ------------------------------------------------------------

func TestGetDryRun_Success(t *testing.T) {
	dryResult := &model.DryRunResult{
		Success:  true,
		Warnings: nil,
		Blockers: nil,
	}
	mock := &mockRemediationService{
		getDryRunFn: func(_ context.Context, _, _ uuid.UUID) (*model.DryRunResult, error) {
			return dryResult, nil
		},
	}
	h := NewRemediationHandler(mock)
	id := uuid.New()
	r := authRequestWithID("GET", "/cyber/remediation/"+id.String()+"/dry-run", id, nil)
	w := httptest.NewRecorder()
	h.GetDryRun(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Error("expected response to contain 'data' key")
	}
}

func TestGetDryRun_NotFound(t *testing.T) {
	mock := &mockRemediationService{
		getDryRunFn: func(_ context.Context, _, _ uuid.UUID) (*model.DryRunResult, error) {
			return nil, repository.ErrNotFound
		},
	}
	h := NewRemediationHandler(mock)
	id := uuid.New()
	r := authRequestWithID("GET", "/cyber/remediation/"+id.String()+"/dry-run", id, nil)
	w := httptest.NewRecorder()
	h.GetDryRun(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ---- Execute --------------------------------------------------------------

func TestExecute_Success(t *testing.T) {
	action := sampleAction()
	action.Status = model.StatusExecuting
	mock := &mockRemediationService{
		executeFn: func(_ context.Context, _, _, _ uuid.UUID, _, _ string, _ *dto.ExecuteRemediationRequest) (*model.RemediationAction, error) {
			return action, nil
		},
	}
	h := NewRemediationHandler(mock)

	// Execute can accept an empty body (ContentLength == 0).
	r := authRequestWithID("POST", "/cyber/remediation/"+action.ID.String()+"/execute", action.ID, nil)
	w := httptest.NewRecorder()
	h.Execute(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Error("expected response to contain 'data' key")
	}
}

func TestExecute_PreConditionFailed(t *testing.T) {
	mock := &mockRemediationService{
		executeFn: func(_ context.Context, _, _, _ uuid.UUID, _, _ string, _ *dto.ExecuteRemediationRequest) (*model.RemediationAction, error) {
			return nil, fmt.Errorf("dry-run required: %w", remediation.ErrPreConditionFailed)
		},
	}
	h := NewRemediationHandler(mock)
	id := uuid.New()
	r := authRequestWithID("POST", "/cyber/remediation/"+id.String()+"/execute", id, nil)
	w := httptest.NewRecorder()
	h.Execute(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ---- Verify ---------------------------------------------------------------

func TestVerify_Success(t *testing.T) {
	action := sampleAction()
	action.Status = model.StatusVerified
	mock := &mockRemediationService{
		verifyFn: func(_ context.Context, _, _, _ uuid.UUID, _, _ string, _ *dto.VerifyRemediationRequest) (*model.RemediationAction, error) {
			return action, nil
		},
	}
	h := NewRemediationHandler(mock)

	// Verify can accept an empty body (ContentLength == 0).
	r := authRequestWithID("POST", "/cyber/remediation/"+action.ID.String()+"/verify", action.ID, nil)
	w := httptest.NewRecorder()
	h.Verify(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Error("expected response to contain 'data' key")
	}
}

// ---- Rollback -------------------------------------------------------------

func TestRollback_Success(t *testing.T) {
	action := sampleAction()
	action.Status = model.StatusRollingBack
	mock := &mockRemediationService{
		rollbackFn: func(_ context.Context, _, _, _ uuid.UUID, _, _ string, _ *dto.RollbackRequest) (*model.RemediationAction, error) {
			return action, nil
		},
	}
	h := NewRemediationHandler(mock)

	body, _ := json.Marshal(dto.RollbackRequest{Reason: "verification failed"})
	r := authRequestWithID("POST", "/cyber/remediation/"+action.ID.String()+"/rollback", action.ID, body)
	w := httptest.NewRecorder()
	h.Rollback(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Error("expected response to contain 'data' key")
	}
}

func TestRollback_InvalidJSON(t *testing.T) {
	h := NewRemediationHandler(&mockRemediationService{})
	id := uuid.New()
	r := authRequestWithID("POST", "/cyber/remediation/"+id.String()+"/rollback", id, []byte(`{not json}`))
	w := httptest.NewRecorder()
	h.Rollback(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ---- Close ----------------------------------------------------------------

func TestClose_Success(t *testing.T) {
	action := sampleAction()
	action.Status = model.StatusClosed
	mock := &mockRemediationService{
		closeFn: func(_ context.Context, _, _, _ uuid.UUID, _, _ string) (*model.RemediationAction, error) {
			return action, nil
		},
	}
	h := NewRemediationHandler(mock)

	r := authRequestWithID("POST", "/cyber/remediation/"+action.ID.String()+"/close", action.ID, nil)
	w := httptest.NewRecorder()
	h.Close(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Error("expected response to contain 'data' key")
	}
}

// ---- AuditTrail -----------------------------------------------------------

func TestAuditTrail_Success(t *testing.T) {
	entries := []model.RemediationAuditEntry{
		{
			ID:            uuid.New(),
			TenantID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			RemediationID: uuid.MustParse("00000000-0000-0000-0000-000000000010"),
			Action:        "created",
			ActorName:     "analyst@example.com",
			OldStatus:     "",
			NewStatus:     "draft",
			CreatedAt:     time.Now(),
		},
		{
			ID:            uuid.New(),
			TenantID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			RemediationID: uuid.MustParse("00000000-0000-0000-0000-000000000010"),
			Action:        "submitted",
			ActorName:     "analyst@example.com",
			OldStatus:     "draft",
			NewStatus:     "pending_approval",
			CreatedAt:     time.Now(),
		},
	}
	mock := &mockRemediationService{
		auditTrailFn: func(_ context.Context, _, _ uuid.UUID) ([]model.RemediationAuditEntry, error) {
			return entries, nil
		},
	}
	h := NewRemediationHandler(mock)
	id := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	r := authRequestWithID("GET", "/cyber/remediation/"+id.String()+"/audit-trail", id, nil)
	w := httptest.NewRecorder()
	h.AuditTrail(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Error("expected response to contain 'data' key")
	}
	// Verify the data is an array.
	var dataArr []json.RawMessage
	if err := json.Unmarshal(resp["data"], &dataArr); err != nil {
		t.Fatalf("expected 'data' to be a JSON array: %v", err)
	}
	if len(dataArr) != 2 {
		t.Errorf("expected 2 audit entries, got %d", len(dataArr))
	}
}

func TestAuditTrail_NotFound(t *testing.T) {
	mock := &mockRemediationService{
		auditTrailFn: func(_ context.Context, _, _ uuid.UUID) ([]model.RemediationAuditEntry, error) {
			return nil, repository.ErrNotFound
		},
	}
	h := NewRemediationHandler(mock)
	id := uuid.New()
	r := authRequestWithID("GET", "/cyber/remediation/"+id.String()+"/audit-trail", id, nil)
	w := httptest.NewRecorder()
	h.AuditTrail(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ---- Stats ----------------------------------------------------------------

func TestStats_Success(t *testing.T) {
	stats := &model.RemediationStats{
		Total:                   42,
		Draft:                   5,
		PendingApproval:         3,
		Approved:                10,
		DryRunCompleted:         4,
		Executing:               2,
		Executed:                8,
		Verified:                6,
		VerificationFailed:      1,
		RolledBack:              1,
		Failed:                  0,
		Closed:                  2,
		AvgExecutionHours:       1.5,
		VerificationSuccessRate: 0.86,
		RollbackRate:            0.05,
	}
	mock := &mockRemediationService{
		statsFn: func(_ context.Context, _ uuid.UUID) (*model.RemediationStats, error) {
			return stats, nil
		},
	}
	h := NewRemediationHandler(mock)

	r := authRequest("GET", "/cyber/remediation/stats", nil)
	w := httptest.NewRecorder()
	h.Stats(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Error("expected response to contain 'data' key")
	}
	// Verify some stats fields are present in the data.
	var statsResp model.RemediationStats
	if err := json.Unmarshal(resp["data"], &statsResp); err != nil {
		t.Fatalf("failed to unmarshal stats data: %v", err)
	}
	if statsResp.Total != 42 {
		t.Errorf("expected total=42, got %d", statsResp.Total)
	}
	if statsResp.Draft != 5 {
		t.Errorf("expected draft=5, got %d", statsResp.Draft)
	}
}

// ---------------------------------------------------------------------------
// Reject endpoint — verify the JSON contract (reason field, not notes)
// ---------------------------------------------------------------------------

func TestRemediationHandler_Reject_Success(t *testing.T) {
	action := sampleAction()
	action.Status = model.StatusRejected

	var capturedReq *dto.RejectRemediationRequest
	mock := &mockRemediationService{
		rejectFn: func(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.RejectRemediationRequest) (*model.RemediationAction, error) {
			capturedReq = req
			return action, nil
		},
	}
	h := NewRemediationHandler(mock)

	// Send the payload with "reason" field (matching backend DTO contract)
	body := `{"reason":"does not meet compliance requirements"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/cyber/remediation/00000000-0000-0000-0000-000000000010/reject", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", action.ID.String())
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	r = r.WithContext(auth.WithTenantID(r.Context(), action.TenantID.String()))
	r = r.WithContext(auth.WithUser(r.Context(), &auth.ContextUser{
		ID:    uuid.MustParse("00000000-0000-0000-0000-000000000002").String(),
		Email: "admin@clario.dev",
		Roles: []string{"admin"},
	}))

	w := httptest.NewRecorder()
	h.Reject(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	if capturedReq == nil {
		t.Fatal("reject service method was never called")
	}
	if capturedReq.Reason != "does not meet compliance requirements" {
		t.Errorf("expected reason=%q, got %q", "does not meet compliance requirements", capturedReq.Reason)
	}
}

func TestRemediationHandler_Reject_EmptyReasonFails(t *testing.T) {
	mock := &mockRemediationService{
		rejectFn: func(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string, req *dto.RejectRemediationRequest) (*model.RemediationAction, error) {
			// Validate should fail before reaching here
			if err := req.Validate(); err != nil {
				return nil, err
			}
			return sampleAction(), nil
		},
	}
	h := NewRemediationHandler(mock)

	body := `{"reason":""}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/cyber/remediation/00000000-0000-0000-0000-000000000010/reject", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "00000000-0000-0000-0000-000000000010")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	r = r.WithContext(auth.WithTenantID(r.Context(), "00000000-0000-0000-0000-000000000001"))
	r = r.WithContext(auth.WithUser(r.Context(), &auth.ContextUser{
		ID:    "00000000-0000-0000-0000-000000000002",
		Email: "admin@clario.dev",
		Roles: []string{"admin"},
	}))

	w := httptest.NewRecorder()
	h.Reject(w, r)

	// The handler will forward the validation error from the service layer (500)
	// because RejectRemediationRequest.Validate() returns error for empty reason
	if w.Code == http.StatusOK {
		t.Error("expected non-200 for empty rejection reason")
	}
}

// ---------------------------------------------------------------------------
// Submit endpoint
// ---------------------------------------------------------------------------

func TestRemediationHandler_Submit_Success(t *testing.T) {
	action := sampleAction()
	action.Status = model.StatusPendingApproval

	mock := &mockRemediationService{
		submitFn: func(ctx context.Context, tenantID, remediationID, actorID uuid.UUID, actorName, actorRole string) (*model.RemediationAction, error) {
			return action, nil
		},
	}
	h := NewRemediationHandler(mock)

	r := httptest.NewRequest(http.MethodPost, "/api/v1/cyber/remediation/00000000-0000-0000-0000-000000000010/submit", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", action.ID.String())
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	r = r.WithContext(auth.WithTenantID(r.Context(), action.TenantID.String()))
	r = r.WithContext(auth.WithUser(r.Context(), &auth.ContextUser{
		ID:    "00000000-0000-0000-0000-000000000002",
		Email: "admin@clario.dev",
		Roles: []string{"admin"},
	}))

	w := httptest.NewRecorder()
	h.Submit(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response not valid JSON: %v", err)
	}
	var data model.RemediationAction
	if err := json.Unmarshal(resp["data"], &data); err != nil {
		t.Fatalf("failed to unmarshal data: %v", err)
	}
	if data.Status != model.StatusPendingApproval {
		t.Errorf("expected status=%s, got %s", model.StatusPendingApproval, data.Status)
	}
}

// ---------------------------------------------------------------------------
// created_by_name JSON contract
// ---------------------------------------------------------------------------

func TestRemediationHandler_Get_CreatedByNameInJSON(t *testing.T) {
	action := sampleAction()
	mock := &mockRemediationService{
		getFn: func(ctx context.Context, tenantID, remediationID uuid.UUID) (*model.RemediationAction, error) {
			return action, nil
		},
	}
	h := NewRemediationHandler(mock)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/remediation/00000000-0000-0000-0000-000000000010", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", action.ID.String())
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	r = r.WithContext(auth.WithTenantID(r.Context(), action.TenantID.String()))
	r = r.WithContext(auth.WithUser(r.Context(), &auth.ContextUser{
		ID:    "00000000-0000-0000-0000-000000000002",
		Email: "admin@clario.dev",
		Roles: []string{"admin"},
	}))

	w := httptest.NewRecorder()
	h.Get(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var envelope map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&envelope); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(envelope["data"], &raw); err != nil {
		t.Fatalf("failed to unmarshal data: %v", err)
	}
	if _, ok := raw["created_by_name"]; !ok {
		t.Error("missing created_by_name in JSON response")
	}
	var name string
	if err := json.Unmarshal(raw["created_by_name"], &name); err != nil {
		t.Fatalf("created_by_name is not a string: %v", err)
	}
	if name != "admin@clario.dev" {
		t.Errorf("created_by_name = %q, want %q", name, "admin@clario.dev")
	}
}
