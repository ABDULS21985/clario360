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
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/cyber/service"
)

// ---------------------------------------------------------------------------
// mock
// ---------------------------------------------------------------------------

type mockAlertService struct {
	listAlertsFn          func(ctx context.Context, tenantID uuid.UUID, params *dto.AlertListParams, actor *service.Actor) (*dto.AlertListResponse, error)
	getAlertFn            func(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor) (*model.Alert, error)
	updateStatusFn        func(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor, req *dto.AlertStatusUpdateRequest) (*model.Alert, error)
	assignFn              func(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor, assignedTo uuid.UUID) (*model.Alert, error)
	escalateFn            func(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor, escalatedTo uuid.UUID, reason string) (*model.Alert, error)
	markFalsePositiveFn   func(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor, reason string) (*model.Alert, error)
	addCommentFn          func(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor, req *dto.AlertCommentRequest) (*model.AlertComment, error)
	listCommentsFn        func(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor) ([]*model.AlertComment, error)
	listTimelineFn        func(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor) ([]*model.AlertTimelineEntry, error)
	mergeFn               func(ctx context.Context, tenantID, primaryAlertID uuid.UUID, mergeIDs []uuid.UUID, actor *service.Actor) (*model.Alert, error)
	relatedFn             func(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor) ([]*model.Alert, error)
	statsFn               func(ctx context.Context, tenantID uuid.UUID, actor *service.Actor) (*model.AlertStats, error)
	countFn               func(ctx context.Context, tenantID uuid.UUID, params *dto.AlertListParams, actor *service.Actor) (int, error)
	bulkUpdateStatusFn    func(ctx context.Context, tenantID uuid.UUID, actor *service.Actor, req *dto.BulkAlertStatusRequest) (*dto.BulkOperationResult, error)
	bulkAssignFn          func(ctx context.Context, tenantID uuid.UUID, actor *service.Actor, req *dto.BulkAlertAssignRequest) (*dto.BulkOperationResult, error)
	bulkMarkFPFn          func(ctx context.Context, tenantID uuid.UUID, actor *service.Actor, req *dto.BulkAlertFalsePositiveRequest) (*dto.BulkOperationResult, error)
	countWithHistoryFn    func(ctx context.Context, tenantID uuid.UUID, params *dto.AlertListParams, actor *service.Actor) (*dto.AlertCountResponse, error)
}

func (m *mockAlertService) ListAlerts(ctx context.Context, tenantID uuid.UUID, params *dto.AlertListParams, actor *service.Actor) (*dto.AlertListResponse, error) {
	if m.listAlertsFn != nil {
		return m.listAlertsFn(ctx, tenantID, params, actor)
	}
	return &dto.AlertListResponse{Data: []*model.Alert{}, Meta: dto.PaginationMeta{Page: 1, PerPage: 25, Total: 0, TotalPages: 1}}, nil
}

func (m *mockAlertService) GetAlert(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor) (*model.Alert, error) {
	if m.getAlertFn != nil {
		return m.getAlertFn(ctx, tenantID, alertID, actor)
	}
	return nil, nil
}

func (m *mockAlertService) UpdateStatus(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor, req *dto.AlertStatusUpdateRequest) (*model.Alert, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, tenantID, alertID, actor, req)
	}
	return nil, nil
}

func (m *mockAlertService) Assign(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor, assignedTo uuid.UUID) (*model.Alert, error) {
	if m.assignFn != nil {
		return m.assignFn(ctx, tenantID, alertID, actor, assignedTo)
	}
	return nil, nil
}

func (m *mockAlertService) Escalate(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor, escalatedTo uuid.UUID, reason string) (*model.Alert, error) {
	if m.escalateFn != nil {
		return m.escalateFn(ctx, tenantID, alertID, actor, escalatedTo, reason)
	}
	return nil, nil
}

func (m *mockAlertService) MarkFalsePositive(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor, reason string) (*model.Alert, error) {
	if m.markFalsePositiveFn != nil {
		return m.markFalsePositiveFn(ctx, tenantID, alertID, actor, reason)
	}
	return nil, nil
}

func (m *mockAlertService) AddComment(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor, req *dto.AlertCommentRequest) (*model.AlertComment, error) {
	if m.addCommentFn != nil {
		return m.addCommentFn(ctx, tenantID, alertID, actor, req)
	}
	return nil, nil
}

func (m *mockAlertService) ListComments(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor) ([]*model.AlertComment, error) {
	if m.listCommentsFn != nil {
		return m.listCommentsFn(ctx, tenantID, alertID, actor)
	}
	return nil, nil
}

func (m *mockAlertService) ListTimeline(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor) ([]*model.AlertTimelineEntry, error) {
	if m.listTimelineFn != nil {
		return m.listTimelineFn(ctx, tenantID, alertID, actor)
	}
	return nil, nil
}

func (m *mockAlertService) Merge(ctx context.Context, tenantID, primaryAlertID uuid.UUID, mergeIDs []uuid.UUID, actor *service.Actor) (*model.Alert, error) {
	if m.mergeFn != nil {
		return m.mergeFn(ctx, tenantID, primaryAlertID, mergeIDs, actor)
	}
	return nil, nil
}

func (m *mockAlertService) Related(ctx context.Context, tenantID, alertID uuid.UUID, actor *service.Actor) ([]*model.Alert, error) {
	if m.relatedFn != nil {
		return m.relatedFn(ctx, tenantID, alertID, actor)
	}
	return nil, nil
}

func (m *mockAlertService) Stats(ctx context.Context, tenantID uuid.UUID, actor *service.Actor) (*model.AlertStats, error) {
	if m.statsFn != nil {
		return m.statsFn(ctx, tenantID, actor)
	}
	return nil, nil
}

func (m *mockAlertService) Count(ctx context.Context, tenantID uuid.UUID, params *dto.AlertListParams, actor *service.Actor) (int, error) {
	if m.countFn != nil {
		return m.countFn(ctx, tenantID, params, actor)
	}
	return 0, nil
}

func (m *mockAlertService) CountWithHistory(ctx context.Context, tenantID uuid.UUID, params *dto.AlertListParams, actor *service.Actor) (*dto.AlertCountResponse, error) {
	if m.countWithHistoryFn != nil {
		return m.countWithHistoryFn(ctx, tenantID, params, actor)
	}
	return &dto.AlertCountResponse{Count: 0}, nil
}

func (m *mockAlertService) BulkUpdateStatus(ctx context.Context, tenantID uuid.UUID, actor *service.Actor, req *dto.BulkAlertStatusRequest) (*dto.BulkOperationResult, error) {
	if m.bulkUpdateStatusFn != nil {
		return m.bulkUpdateStatusFn(ctx, tenantID, actor, req)
	}
	return nil, nil
}

func (m *mockAlertService) BulkAssign(ctx context.Context, tenantID uuid.UUID, actor *service.Actor, req *dto.BulkAlertAssignRequest) (*dto.BulkOperationResult, error) {
	if m.bulkAssignFn != nil {
		return m.bulkAssignFn(ctx, tenantID, actor, req)
	}
	return nil, nil
}

func (m *mockAlertService) BulkMarkFalsePositive(ctx context.Context, tenantID uuid.UUID, actor *service.Actor, req *dto.BulkAlertFalsePositiveRequest) (*dto.BulkOperationResult, error) {
	if m.bulkMarkFPFn != nil {
		return m.bulkMarkFPFn(ctx, tenantID, actor, req)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func alertAuthRequest(method, path string, body []byte) *http.Request {
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

func alertAuthRequestWithID(method, path string, id uuid.UUID, body []byte) *http.Request {
	r := alertAuthRequest(method, path, body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	return r.WithContext(ctx)
}

func sampleAlert() *model.Alert {
	now := time.Now()
	return &model.Alert{
		ID:              uuid.MustParse("00000000-0000-0000-0000-000000000010"),
		TenantID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Title:           "Suspicious SSH login",
		Description:     "Multiple failed SSH login attempts detected",
		Severity:        "high",
		Status:          model.AlertStatusNew,
		ConfidenceScore: 0.85,
		Source:          "detection_engine",
		EventCount:      3,
		AssetIDs:        []uuid.UUID{},
		Tags:            []string{"ssh", "brute-force"},
		FirstEventAt:    now.Add(-time.Hour),
		LastEventAt:     now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func decodeResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	return body
}

// ---------------------------------------------------------------------------
// 1. Auth enforcement
// ---------------------------------------------------------------------------

func TestAlertHandler_ListAlerts_NoAuth(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{})
	r := httptest.NewRequest("GET", "/cyber/alerts", nil)
	w := httptest.NewRecorder()
	h.ListAlerts(w, r)
	if w.Code < 400 {
		t.Errorf("expected 4xx without auth, got %d", w.Code)
	}
}

func TestAlertHandler_GetAlert_NoAuth(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{})
	r := httptest.NewRequest("GET", "/cyber/alerts/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	h.GetAlert(w, r)
	if w.Code < 400 {
		t.Errorf("expected 4xx without auth, got %d", w.Code)
	}
}

func TestAlertHandler_UpdateStatus_NoAuth(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{})
	body := []byte(`{"status":"acknowledged"}`)
	r := httptest.NewRequest("PUT", "/cyber/alerts/"+uuid.New().String()+"/status", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	if w.Code < 400 {
		t.Errorf("expected 4xx without auth, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// 2. Error mapping
// ---------------------------------------------------------------------------

func TestAlertHandler_GetAlert_NotFound(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{
		getAlertFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor) (*model.Alert, error) {
			return nil, repository.ErrNotFound
		},
	})
	id := uuid.New()
	r := alertAuthRequestWithID("GET", "/cyber/alerts/"+id.String(), id, nil)
	w := httptest.NewRecorder()
	h.GetAlert(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAlertHandler_GetAlert_InternalError(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{
		getAlertFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor) (*model.Alert, error) {
			return nil, fmt.Errorf("database connection refused")
		},
	})
	id := uuid.New()
	r := alertAuthRequestWithID("GET", "/cyber/alerts/"+id.String(), id, nil)
	w := httptest.NewRecorder()
	h.GetAlert(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestAlertHandler_UpdateStatus_ServiceError(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{
		updateStatusFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor, _ *dto.AlertStatusUpdateRequest) (*model.Alert, error) {
			return nil, fmt.Errorf("invalid transition")
		},
	})
	id := uuid.New()
	body := []byte(`{"status":"acknowledged"}`)
	r := alertAuthRequestWithID("PUT", "/cyber/alerts/"+id.String()+"/status", id, body)
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAlertHandler_ListAlerts_ParseError(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{})
	r := alertAuthRequest("GET", "/cyber/alerts?assigned_to=not-a-uuid", nil)
	w := httptest.NewRecorder()
	h.ListAlerts(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// 3. Parameter parsing
// ---------------------------------------------------------------------------

func TestAlertHandler_ListAlerts_DefaultParams(t *testing.T) {
	var captured *dto.AlertListParams
	h := NewAlertHandler(&mockAlertService{
		listAlertsFn: func(_ context.Context, _ uuid.UUID, params *dto.AlertListParams, _ *service.Actor) (*dto.AlertListResponse, error) {
			captured = params
			return &dto.AlertListResponse{Data: []*model.Alert{}, Meta: dto.PaginationMeta{Page: 1, PerPage: 25, Total: 0, TotalPages: 1}}, nil
		},
	})
	r := alertAuthRequest("GET", "/cyber/alerts", nil)
	w := httptest.NewRecorder()
	h.ListAlerts(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if captured == nil {
		t.Fatal("params not captured")
	}
	if captured.Page != 0 && captured.Page != 1 {
		t.Errorf("expected default page 0 or 1, got %d", captured.Page)
	}
}

func TestAlertHandler_ListAlerts_WithFilters(t *testing.T) {
	var captured *dto.AlertListParams
	h := NewAlertHandler(&mockAlertService{
		listAlertsFn: func(_ context.Context, _ uuid.UUID, params *dto.AlertListParams, _ *service.Actor) (*dto.AlertListResponse, error) {
			captured = params
			return &dto.AlertListResponse{Data: []*model.Alert{}, Meta: dto.PaginationMeta{Page: 1, PerPage: 10, Total: 0, TotalPages: 1}}, nil
		},
	})
	r := alertAuthRequest("GET", "/cyber/alerts?severity=high&status=new&page=2&per_page=10&sort=severity&order=asc", nil)
	w := httptest.NewRecorder()
	h.ListAlerts(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if len(captured.Severities) != 1 || captured.Severities[0] != "high" {
		t.Errorf("expected severity=[high], got %v", captured.Severities)
	}
	if len(captured.Statuses) != 1 || captured.Statuses[0] != "new" {
		t.Errorf("expected status=[new], got %v", captured.Statuses)
	}
	if captured.Page != 2 {
		t.Errorf("expected page=2, got %d", captured.Page)
	}
	if captured.PerPage != 10 {
		t.Errorf("expected per_page=10, got %d", captured.PerPage)
	}
	if captured.Sort != "severity" {
		t.Errorf("expected sort=severity, got %q", captured.Sort)
	}
	if captured.Order != "asc" {
		t.Errorf("expected order=asc, got %q", captured.Order)
	}
}

func TestAlertHandler_ListAlerts_InvalidSortReject(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{
		listAlertsFn: func(_ context.Context, _ uuid.UUID, params *dto.AlertListParams, _ *service.Actor) (*dto.AlertListResponse, error) {
			params.SetDefaults()
			if err := params.Validate(); err != nil {
				return nil, err
			}
			return &dto.AlertListResponse{Data: []*model.Alert{}, Meta: dto.PaginationMeta{Page: 1, PerPage: 25, Total: 0, TotalPages: 1}}, nil
		},
	})
	r := alertAuthRequest("GET", "/cyber/alerts?sort=invalid_field", nil)
	w := httptest.NewRecorder()
	h.ListAlerts(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid sort, got %d", w.Code)
	}
}

func TestAlertHandler_ListAlerts_SortByTitle(t *testing.T) {
	var captured *dto.AlertListParams
	h := NewAlertHandler(&mockAlertService{
		listAlertsFn: func(_ context.Context, _ uuid.UUID, params *dto.AlertListParams, _ *service.Actor) (*dto.AlertListResponse, error) {
			captured = params
			return &dto.AlertListResponse{Data: []*model.Alert{}, Meta: dto.PaginationMeta{Page: 1, PerPage: 25, Total: 0, TotalPages: 1}}, nil
		},
	})
	r := alertAuthRequest("GET", "/cyber/alerts?sort=title", nil)
	w := httptest.NewRecorder()
	h.ListAlerts(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for sort=title, got %d", w.Code)
	}
	if captured.Sort != "title" {
		t.Errorf("expected sort=title, got %q", captured.Sort)
	}
}

func TestAlertHandler_Count_PassesFilters(t *testing.T) {
	var captured *dto.AlertListParams
	h := NewAlertHandler(&mockAlertService{
		countWithHistoryFn: func(_ context.Context, _ uuid.UUID, params *dto.AlertListParams, _ *service.Actor) (*dto.AlertCountResponse, error) {
			captured = params
			return &dto.AlertCountResponse{Count: 42, History: []int{1, 2, 3}}, nil
		},
	})
	r := alertAuthRequest("GET", "/cyber/alerts/count?severity=critical&status=new", nil)
	w := httptest.NewRecorder()
	h.Count(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if len(captured.Severities) != 1 || captured.Severities[0] != "critical" {
		t.Errorf("expected severity=[critical], got %v", captured.Severities)
	}
}

// ---------------------------------------------------------------------------
// 4. Validation
// ---------------------------------------------------------------------------

func TestAlertHandler_UpdateStatus_InvalidJSON(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{})
	id := uuid.New()
	r := alertAuthRequestWithID("PUT", "/cyber/alerts/"+id.String()+"/status", id, []byte(`{invalid`))
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestAlertHandler_UpdateStatus_MissingStatus(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{})
	id := uuid.New()
	r := alertAuthRequestWithID("PUT", "/cyber/alerts/"+id.String()+"/status", id, []byte(`{}`))
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing status, got %d", w.Code)
	}
}

func TestAlertHandler_Escalate_MissingReason(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{})
	id := uuid.New()
	body := []byte(`{"escalated_to":"00000000-0000-0000-0000-000000000003"}`)
	r := alertAuthRequestWithID("POST", "/cyber/alerts/"+id.String()+"/escalate", id, body)
	w := httptest.NewRecorder()
	h.Escalate(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing reason, got %d", w.Code)
	}
}

func TestAlertHandler_Merge_EmptyMergeIDs(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{
		mergeFn: func(_ context.Context, _, _ uuid.UUID, ids []uuid.UUID, _ *service.Actor) (*model.Alert, error) {
			if len(ids) == 0 {
				return nil, fmt.Errorf("merge_ids must not be empty")
			}
			return nil, nil
		},
	})
	id := uuid.New()
	body := []byte(`{"merge_ids":[]}`)
	r := alertAuthRequestWithID("POST", "/cyber/alerts/"+id.String()+"/merge", id, body)
	w := httptest.NewRecorder()
	h.Merge(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty merge_ids, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// 5. Happy paths
// ---------------------------------------------------------------------------

func TestAlertHandler_ListAlerts_Success(t *testing.T) {
	alert := sampleAlert()
	h := NewAlertHandler(&mockAlertService{
		listAlertsFn: func(_ context.Context, _ uuid.UUID, _ *dto.AlertListParams, _ *service.Actor) (*dto.AlertListResponse, error) {
			return &dto.AlertListResponse{
				Data: []*model.Alert{alert},
				Meta: dto.PaginationMeta{Page: 1, PerPage: 25, Total: 1, TotalPages: 1},
			}, nil
		},
	})
	r := alertAuthRequest("GET", "/cyber/alerts", nil)
	w := httptest.NewRecorder()
	h.ListAlerts(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := decodeResponse(t, w)
	if _, ok := body["data"]; !ok {
		t.Error("expected data key in response")
	}
}

func TestAlertHandler_GetAlert_Success(t *testing.T) {
	alert := sampleAlert()
	h := NewAlertHandler(&mockAlertService{
		getAlertFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor) (*model.Alert, error) {
			return alert, nil
		},
	})
	r := alertAuthRequestWithID("GET", "/cyber/alerts/"+alert.ID.String(), alert.ID, nil)
	w := httptest.NewRecorder()
	h.GetAlert(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := decodeResponse(t, w)
	data, _ := body["data"].(map[string]any)
	if data["title"] != "Suspicious SSH login" {
		t.Errorf("expected alert title, got %v", data["title"])
	}
}

func TestAlertHandler_UpdateStatus_Success(t *testing.T) {
	alert := sampleAlert()
	alert.Status = model.AlertStatusAcknowledged
	h := NewAlertHandler(&mockAlertService{
		updateStatusFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor, _ *dto.AlertStatusUpdateRequest) (*model.Alert, error) {
			return alert, nil
		},
	})
	id := uuid.New()
	body := []byte(`{"status":"acknowledged"}`)
	r := alertAuthRequestWithID("PUT", "/cyber/alerts/"+id.String()+"/status", id, body)
	w := httptest.NewRecorder()
	h.UpdateStatus(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAlertHandler_Assign_Success(t *testing.T) {
	alert := sampleAlert()
	h := NewAlertHandler(&mockAlertService{
		assignFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor, _ uuid.UUID) (*model.Alert, error) {
			return alert, nil
		},
	})
	id := uuid.New()
	body := []byte(`{"assigned_to":"00000000-0000-0000-0000-000000000003"}`)
	r := alertAuthRequestWithID("PUT", "/cyber/alerts/"+id.String()+"/assign", id, body)
	w := httptest.NewRecorder()
	h.Assign(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAlertHandler_Escalate_Success(t *testing.T) {
	alert := sampleAlert()
	h := NewAlertHandler(&mockAlertService{
		escalateFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor, _ uuid.UUID, _ string) (*model.Alert, error) {
			return alert, nil
		},
	})
	id := uuid.New()
	body := []byte(`{"escalated_to":"00000000-0000-0000-0000-000000000003","reason":"Potential APT activity requiring senior analysis"}`)
	r := alertAuthRequestWithID("POST", "/cyber/alerts/"+id.String()+"/escalate", id, body)
	w := httptest.NewRecorder()
	h.Escalate(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAlertHandler_MarkFalsePositive_Success(t *testing.T) {
	alert := sampleAlert()
	h := NewAlertHandler(&mockAlertService{
		markFalsePositiveFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor, _ string) (*model.Alert, error) {
			return alert, nil
		},
	})
	id := uuid.New()
	body := []byte(`{"reason":"Known scanner traffic from security team"}`)
	r := alertAuthRequestWithID("PUT", "/cyber/alerts/"+id.String()+"/false-positive", id, body)
	w := httptest.NewRecorder()
	h.MarkFalsePositive(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAlertHandler_AddComment_Success(t *testing.T) {
	comment := &model.AlertComment{
		ID:       uuid.New(),
		TenantID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		AlertID:  uuid.New(),
		Content:  "Investigation notes",
	}
	h := NewAlertHandler(&mockAlertService{
		addCommentFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor, _ *dto.AlertCommentRequest) (*model.AlertComment, error) {
			return comment, nil
		},
	})
	id := uuid.New()
	body := []byte(`{"content":"Investigation notes"}`)
	r := alertAuthRequestWithID("POST", "/cyber/alerts/"+id.String()+"/comment", id, body)
	w := httptest.NewRecorder()
	h.AddComment(w, r)
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
}

func TestAlertHandler_ListComments_Success(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{
		listCommentsFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor) ([]*model.AlertComment, error) {
			return []*model.AlertComment{}, nil
		},
	})
	id := uuid.New()
	r := alertAuthRequestWithID("GET", "/cyber/alerts/"+id.String()+"/comments", id, nil)
	w := httptest.NewRecorder()
	h.ListComments(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAlertHandler_ListTimeline_Success(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{
		listTimelineFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor) ([]*model.AlertTimelineEntry, error) {
			return []*model.AlertTimelineEntry{}, nil
		},
	})
	id := uuid.New()
	r := alertAuthRequestWithID("GET", "/cyber/alerts/"+id.String()+"/timeline", id, nil)
	w := httptest.NewRecorder()
	h.ListTimeline(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAlertHandler_Merge_Success(t *testing.T) {
	alert := sampleAlert()
	h := NewAlertHandler(&mockAlertService{
		mergeFn: func(_ context.Context, _, _ uuid.UUID, _ []uuid.UUID, _ *service.Actor) (*model.Alert, error) {
			return alert, nil
		},
	})
	id := uuid.New()
	mergeID := uuid.New()
	body, _ := json.Marshal(map[string]any{"merge_ids": []string{mergeID.String()}})
	r := alertAuthRequestWithID("POST", "/cyber/alerts/"+id.String()+"/merge", id, body)
	w := httptest.NewRecorder()
	h.Merge(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAlertHandler_Related_Success(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{
		relatedFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor) ([]*model.Alert, error) {
			return []*model.Alert{}, nil
		},
	})
	id := uuid.New()
	r := alertAuthRequestWithID("GET", "/cyber/alerts/"+id.String()+"/related", id, nil)
	w := httptest.NewRecorder()
	h.Related(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAlertHandler_Stats_Success(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{
		statsFn: func(_ context.Context, _ uuid.UUID, _ *service.Actor) (*model.AlertStats, error) {
			return &model.AlertStats{Total: 42}, nil
		},
	})
	r := alertAuthRequest("GET", "/cyber/alerts/stats", nil)
	w := httptest.NewRecorder()
	h.Stats(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAlertHandler_Count_Success(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{
		countWithHistoryFn: func(_ context.Context, _ uuid.UUID, _ *dto.AlertListParams, _ *service.Actor) (*dto.AlertCountResponse, error) {
			trend := 2
			return &dto.AlertCountResponse{Count: 7, Trend: &trend, History: []int{3, 5}}, nil
		},
	})
	r := alertAuthRequest("GET", "/cyber/alerts/count", nil)
	w := httptest.NewRecorder()
	h.Count(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := decodeResponse(t, w)
	data, _ := body["data"].(map[string]any)
	if data["count"] != float64(7) {
		t.Errorf("expected count=7, got %v", data["count"])
	}
}

// ---------------------------------------------------------------------------
// 6. UUID parsing
// ---------------------------------------------------------------------------

func TestAlertHandler_GetAlert_InvalidUUID(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{})
	r := alertAuthRequest("GET", "/cyber/alerts/not-a-uuid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "not-a-uuid")
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	h.GetAlert(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid UUID, got %d", w.Code)
	}
}

func TestAlertHandler_Assign_InvalidUUID(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{})
	body := []byte(`{"assigned_to":"00000000-0000-0000-0000-000000000003"}`)
	r := alertAuthRequest("PUT", "/cyber/alerts/bad-uuid/assign", body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "bad-uuid")
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()
	h.Assign(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid UUID, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// 7. Bulk endpoints
// ---------------------------------------------------------------------------

func TestAlertHandler_BulkUpdateStatus_Success(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{
		bulkUpdateStatusFn: func(_ context.Context, _ uuid.UUID, _ *service.Actor, _ *dto.BulkAlertStatusRequest) (*dto.BulkOperationResult, error) {
			return &dto.BulkOperationResult{Processed: 2, Successful: 2}, nil
		},
	})
	ids := []string{uuid.New().String(), uuid.New().String()}
	body, _ := json.Marshal(map[string]any{"alert_ids": ids, "status": "acknowledged"})
	r := alertAuthRequest("PUT", "/cyber/alerts/bulk/status", body)
	w := httptest.NewRecorder()
	h.BulkUpdateStatus(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	resp := decodeResponse(t, w)
	data, _ := resp["data"].(map[string]any)
	if data["successful"] != float64(2) {
		t.Errorf("expected successful=2, got %v", data["successful"])
	}
}

func TestAlertHandler_BulkUpdateStatus_EmptyIDs(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{})
	body := []byte(`{"alert_ids":[],"status":"acknowledged"}`)
	r := alertAuthRequest("PUT", "/cyber/alerts/bulk/status", body)
	w := httptest.NewRecorder()
	h.BulkUpdateStatus(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty alert_ids, got %d", w.Code)
	}
}

func TestAlertHandler_BulkAssign_Success(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{
		bulkAssignFn: func(_ context.Context, _ uuid.UUID, _ *service.Actor, _ *dto.BulkAlertAssignRequest) (*dto.BulkOperationResult, error) {
			return &dto.BulkOperationResult{Processed: 3, Successful: 3}, nil
		},
	})
	ids := []string{uuid.New().String(), uuid.New().String(), uuid.New().String()}
	body, _ := json.Marshal(map[string]any{"alert_ids": ids, "assigned_to": uuid.New().String()})
	r := alertAuthRequest("PUT", "/cyber/alerts/bulk/assign", body)
	w := httptest.NewRecorder()
	h.BulkAssign(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAlertHandler_BulkAssign_NoAuth(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{})
	body := []byte(`{"alert_ids":["` + uuid.New().String() + `"],"assigned_to":"` + uuid.New().String() + `"}`)
	r := httptest.NewRequest("PUT", "/cyber/alerts/bulk/assign", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	h.BulkAssign(w, r)
	if w.Code < 400 {
		t.Errorf("expected 4xx without auth, got %d", w.Code)
	}
}

func TestAlertHandler_BulkMarkFalsePositive_Success(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{
		bulkMarkFPFn: func(_ context.Context, _ uuid.UUID, _ *service.Actor, _ *dto.BulkAlertFalsePositiveRequest) (*dto.BulkOperationResult, error) {
			return &dto.BulkOperationResult{Processed: 2, Successful: 1, Failed: 1, Errors: []dto.BulkError{{AlertID: "x", Error: "not found"}}}, nil
		},
	})
	ids := []string{uuid.New().String(), uuid.New().String()}
	body, _ := json.Marshal(map[string]any{"alert_ids": ids, "reason": "Known scanner traffic"})
	r := alertAuthRequest("PUT", "/cyber/alerts/bulk/false-positive", body)
	w := httptest.NewRecorder()
	h.BulkMarkFalsePositive(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	resp := decodeResponse(t, w)
	data, _ := resp["data"].(map[string]any)
	if data["failed"] != float64(1) {
		t.Errorf("expected failed=1, got %v", data["failed"])
	}
}

func TestAlertHandler_BulkMarkFalsePositive_MissingReason(t *testing.T) {
	h := NewAlertHandler(&mockAlertService{})
	ids := []string{uuid.New().String()}
	body, _ := json.Marshal(map[string]any{"alert_ids": ids, "reason": ""})
	r := alertAuthRequest("PUT", "/cyber/alerts/bulk/false-positive", body)
	w := httptest.NewRecorder()
	h.BulkMarkFalsePositive(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing reason, got %d", w.Code)
	}
}
