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
	cyberrepo "github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/cyber/ueba/dto"
	"github.com/clario360/platform/internal/cyber/ueba/model"
)

// ---- mock service ---------------------------------------------------------

type mockUEBAService struct {
	listProfilesFn        func(ctx context.Context, tenantID uuid.UUID, params *dto.ProfileListParams) (*dto.ProfileListResponse, error)
	getProfileFn          func(ctx context.Context, tenantID uuid.UUID, entityID string) (*dto.ProfileDetailResponse, error)
	getTimelineFn         func(ctx context.Context, tenantID uuid.UUID, entityID string, page, perPage int) (*dto.TimelineResponse, error)
	getHeatmapFn          func(ctx context.Context, tenantID uuid.UUID, entityID string, days int) (*dto.HeatmapResponse, error)
	updateProfileStatusFn func(ctx context.Context, tenantID uuid.UUID, entityID string, req *dto.ProfileStatusUpdateRequest) (*model.UEBAProfile, error)
	listAlertsFn          func(ctx context.Context, tenantID uuid.UUID, params *dto.AlertListParams) (*dto.AlertListResponse, error)
	getAlertFn            func(ctx context.Context, tenantID, alertID uuid.UUID) (*model.UEBAAlert, error)
	updateAlertStatusFn   func(ctx context.Context, tenantID, alertID uuid.UUID, actorID *uuid.UUID, req *dto.AlertStatusUpdateRequest) (*model.UEBAAlert, error)
	markFalsePositiveFn   func(ctx context.Context, tenantID, alertID uuid.UUID, actorID *uuid.UUID, req *dto.FalsePositiveRequest) (*model.UEBAAlert, error)
	getDashboardFn        func(ctx context.Context, tenantID uuid.UUID) (*dto.DashboardResponse, error)
	getRiskRankingFn      func(ctx context.Context, tenantID uuid.UUID, limit int) ([]dto.RiskRankingItem, error)
	getConfigFn           func() dto.UEBAConfigDTO
	updateConfigFn        func(ctx context.Context, req dto.UEBAConfigDTO) (dto.UEBAConfigDTO, error)
}

func (m *mockUEBAService) ListProfiles(ctx context.Context, tenantID uuid.UUID, params *dto.ProfileListParams) (*dto.ProfileListResponse, error) {
	if m.listProfilesFn != nil {
		return m.listProfilesFn(ctx, tenantID, params)
	}
	return &dto.ProfileListResponse{Data: []*model.UEBAProfile{}, Meta: dto.NewPaginationMeta(1, 25, 0)}, nil
}

func (m *mockUEBAService) GetProfile(ctx context.Context, tenantID uuid.UUID, entityID string) (*dto.ProfileDetailResponse, error) {
	if m.getProfileFn != nil {
		return m.getProfileFn(ctx, tenantID, entityID)
	}
	return nil, cyberrepo.ErrNotFound
}

func (m *mockUEBAService) GetTimeline(ctx context.Context, tenantID uuid.UUID, entityID string, page, perPage int) (*dto.TimelineResponse, error) {
	if m.getTimelineFn != nil {
		return m.getTimelineFn(ctx, tenantID, entityID, page, perPage)
	}
	return &dto.TimelineResponse{Data: []*model.DataAccessEvent{}, Meta: dto.NewPaginationMeta(1, 50, 0)}, nil
}

func (m *mockUEBAService) GetHeatmap(ctx context.Context, tenantID uuid.UUID, entityID string, days int) (*dto.HeatmapResponse, error) {
	if m.getHeatmapFn != nil {
		return m.getHeatmapFn(ctx, tenantID, entityID, days)
	}
	return &dto.HeatmapResponse{EntityID: entityID, Days: days}, nil
}

func (m *mockUEBAService) UpdateProfileStatus(ctx context.Context, tenantID uuid.UUID, entityID string, req *dto.ProfileStatusUpdateRequest) (*model.UEBAProfile, error) {
	if m.updateProfileStatusFn != nil {
		return m.updateProfileStatusFn(ctx, tenantID, entityID, req)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockUEBAService) ListAlerts(ctx context.Context, tenantID uuid.UUID, params *dto.AlertListParams) (*dto.AlertListResponse, error) {
	if m.listAlertsFn != nil {
		return m.listAlertsFn(ctx, tenantID, params)
	}
	return &dto.AlertListResponse{Data: []*model.UEBAAlert{}, Meta: dto.NewPaginationMeta(1, 25, 0)}, nil
}

func (m *mockUEBAService) GetAlert(ctx context.Context, tenantID, alertID uuid.UUID) (*model.UEBAAlert, error) {
	if m.getAlertFn != nil {
		return m.getAlertFn(ctx, tenantID, alertID)
	}
	return nil, cyberrepo.ErrNotFound
}

func (m *mockUEBAService) UpdateAlertStatus(ctx context.Context, tenantID, alertID uuid.UUID, actorID *uuid.UUID, req *dto.AlertStatusUpdateRequest) (*model.UEBAAlert, error) {
	if m.updateAlertStatusFn != nil {
		return m.updateAlertStatusFn(ctx, tenantID, alertID, actorID, req)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockUEBAService) MarkFalsePositive(ctx context.Context, tenantID, alertID uuid.UUID, actorID *uuid.UUID, req *dto.FalsePositiveRequest) (*model.UEBAAlert, error) {
	if m.markFalsePositiveFn != nil {
		return m.markFalsePositiveFn(ctx, tenantID, alertID, actorID, req)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockUEBAService) GetDashboard(ctx context.Context, tenantID uuid.UUID) (*dto.DashboardResponse, error) {
	if m.getDashboardFn != nil {
		return m.getDashboardFn(ctx, tenantID)
	}
	return &dto.DashboardResponse{}, nil
}

func (m *mockUEBAService) GetRiskRanking(ctx context.Context, tenantID uuid.UUID, limit int) ([]dto.RiskRankingItem, error) {
	if m.getRiskRankingFn != nil {
		return m.getRiskRankingFn(ctx, tenantID, limit)
	}
	return nil, nil
}

func (m *mockUEBAService) GetConfig() dto.UEBAConfigDTO {
	if m.getConfigFn != nil {
		return m.getConfigFn()
	}
	return dto.UEBAConfigDTO{}
}

func (m *mockUEBAService) UpdateConfig(ctx context.Context, req dto.UEBAConfigDTO) (dto.UEBAConfigDTO, error) {
	if m.updateConfigFn != nil {
		return m.updateConfigFn(ctx, req)
	}
	return dto.UEBAConfigDTO{}, nil
}

// ---- helpers --------------------------------------------------------------

func uebaAuthCtx(tenantID, userID uuid.UUID) context.Context {
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

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	return resp
}

func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	return bytes.NewBuffer(b)
}

func chiCtx(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func sampleAlert(tenantID uuid.UUID) *model.UEBAAlert {
	return &model.UEBAAlert{
		ID:                uuid.New(),
		TenantID:          tenantID,
		EntityType:        model.EntityTypeUser,
		EntityID:          "admin@example.com",
		EntityName:        "Admin User",
		AlertType:         model.AlertTypeUnusualActivity,
		Severity:          "medium",
		Confidence:        0.85,
		RiskScoreBefore:   10,
		RiskScoreAfter:    35,
		RiskScoreDelta:    25,
		Title:             "Unusual access pattern detected",
		Description:       "Activity deviates from baseline",
		TriggeringSignals: []model.AnomalySignal{},
		TriggeringEventIDs: []uuid.UUID{uuid.New()},
		BaselineComparison: map[string]any{},
		CorrelatedSignalCount: 2,
		Status:            "new",
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
}

// ---- NoAuth tests ---------------------------------------------------------

func TestUEBAHandler_NoAuth(t *testing.T) {
	h := NewUEBAHandler(&mockUEBAService{})

	endpoints := []struct {
		name   string
		method string
		invoke func(w http.ResponseWriter, r *http.Request)
	}{
		{"ListProfiles", "GET", h.ListProfiles},
		{"GetProfile", "GET", h.GetProfile},
		{"GetTimeline", "GET", h.GetTimeline},
		{"GetHeatmap", "GET", h.GetHeatmap},
		{"UpdateProfileStatus", "PUT", h.UpdateProfileStatus},
		{"ListAlerts", "GET", h.ListAlerts},
		{"GetAlert", "GET", h.GetAlert},
		{"UpdateAlertStatus", "PUT", h.UpdateAlertStatus},
		{"MarkFalsePositive", "POST", h.MarkFalsePositive},
		{"BulkUpdateAlertStatus", "PUT", h.BulkUpdateAlertStatus},
		{"GetDashboard", "GET", h.GetDashboard},
		{"GetRiskRanking", "GET", h.GetRiskRanking},
		{"GetConfig", "GET", h.GetConfig},
		{"UpdateConfig", "PUT", h.UpdateConfig},
	}

	for _, tc := range endpoints {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, "/ueba/test", nil)
			w := httptest.NewRecorder()
			tc.invoke(w, r)
			if w.Code < 400 {
				t.Errorf("%s: expected 4xx without auth, got %d", tc.name, w.Code)
			}
		})
	}
}

// ---- ListAlerts tests -----------------------------------------------------

func TestListAlerts_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	alert := sampleAlert(tenantID)

	mock := &mockUEBAService{
		listAlertsFn: func(ctx context.Context, tid uuid.UUID, params *dto.AlertListParams) (*dto.AlertListResponse, error) {
			if tid != tenantID {
				t.Errorf("expected tenantID %s, got %s", tenantID, tid)
			}
			return &dto.AlertListResponse{
				Data: []*model.UEBAAlert{alert},
				Meta: dto.NewPaginationMeta(1, 25, 1),
			}, nil
		},
	}
	h := NewUEBAHandler(mock)

	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("GET", "/ueba/alerts", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	h.ListAlerts(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	resp := decodeBody(t, w)
	data, ok := resp["data"].([]any)
	if !ok {
		t.Fatal("expected 'data' array in response")
	}
	if len(data) != 1 {
		t.Errorf("expected 1 alert, got %d", len(data))
	}
	if resp["meta"] == nil {
		t.Error("expected 'meta' key in response")
	}
}

func TestListAlerts_WithFilters(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockUEBAService{
		listAlertsFn: func(ctx context.Context, tid uuid.UUID, params *dto.AlertListParams) (*dto.AlertListResponse, error) {
			if params.EntityID != "admin@example.com" {
				t.Errorf("expected entity_id=admin@example.com, got %s", params.EntityID)
			}
			if params.Status != "new" {
				t.Errorf("expected status=new, got %s", params.Status)
			}
			if params.Page != 2 {
				t.Errorf("expected page=2, got %d", params.Page)
			}
			if params.PerPage != 10 {
				t.Errorf("expected per_page=10, got %d", params.PerPage)
			}
			return &dto.AlertListResponse{
				Data: []*model.UEBAAlert{},
				Meta: dto.NewPaginationMeta(2, 10, 0),
			}, nil
		},
	}
	h := NewUEBAHandler(mock)

	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("GET", "/ueba/alerts?entity_id=admin@example.com&status=new&page=2&per_page=10", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	h.ListAlerts(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestListAlerts_ServiceError(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockUEBAService{
		listAlertsFn: func(ctx context.Context, tid uuid.UUID, params *dto.AlertListParams) (*dto.AlertListResponse, error) {
			return nil, fmt.Errorf("database timeout")
		},
	}
	h := NewUEBAHandler(mock)

	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("GET", "/ueba/alerts", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	h.ListAlerts(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on service error, got %d", w.Code)
	}
}

// ---- GetAlert tests -------------------------------------------------------

func TestGetAlert_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	alert := sampleAlert(tenantID)

	mock := &mockUEBAService{
		getAlertFn: func(ctx context.Context, tid, aid uuid.UUID) (*model.UEBAAlert, error) {
			if aid != alert.ID {
				t.Errorf("expected alertID %s, got %s", alert.ID, aid)
			}
			return alert, nil
		},
	}
	h := NewUEBAHandler(mock)

	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("GET", "/ueba/alerts/"+alert.ID.String(), nil).WithContext(ctx)
	r = chiCtx(r, "id", alert.ID.String())
	w := httptest.NewRecorder()
	h.GetAlert(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	resp := decodeBody(t, w)
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected 'data' object in response")
	}
	if data["title"] != alert.Title {
		t.Errorf("expected title=%q, got %v", alert.Title, data["title"])
	}
}

func TestGetAlert_NotFound(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	alertID := uuid.New()

	mock := &mockUEBAService{
		getAlertFn: func(ctx context.Context, tid, aid uuid.UUID) (*model.UEBAAlert, error) {
			return nil, cyberrepo.ErrNotFound
		},
	}
	h := NewUEBAHandler(mock)

	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("GET", "/ueba/alerts/"+alertID.String(), nil).WithContext(ctx)
	r = chiCtx(r, "id", alertID.String())
	w := httptest.NewRecorder()
	h.GetAlert(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 on not found, got %d", w.Code)
	}
}

func TestGetAlert_InvalidUUID(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	h := NewUEBAHandler(&mockUEBAService{})
	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("GET", "/ueba/alerts/not-a-uuid", nil).WithContext(ctx)
	r = chiCtx(r, "id", "not-a-uuid")
	w := httptest.NewRecorder()
	h.GetAlert(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on invalid UUID, got %d", w.Code)
	}
}

// ---- UpdateAlertStatus tests ----------------------------------------------

func TestUpdateAlertStatus_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	alert := sampleAlert(tenantID)
	alert.Status = "acknowledged"

	mock := &mockUEBAService{
		updateAlertStatusFn: func(ctx context.Context, tid, aid uuid.UUID, actorID *uuid.UUID, req *dto.AlertStatusUpdateRequest) (*model.UEBAAlert, error) {
			if aid != alert.ID {
				t.Errorf("expected alertID %s, got %s", alert.ID, aid)
			}
			if *actorID != userID {
				t.Errorf("expected actorID %s, got %s", userID, *actorID)
			}
			if req.Status != "acknowledged" {
				t.Errorf("expected status=acknowledged, got %s", req.Status)
			}
			if req.Notes != "reviewed and acknowledged" {
				t.Errorf("expected notes='reviewed and acknowledged', got %s", req.Notes)
			}
			return alert, nil
		},
	}
	h := NewUEBAHandler(mock)

	body := jsonBody(t, map[string]string{"status": "acknowledged", "notes": "reviewed and acknowledged"})
	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("PUT", "/ueba/alerts/"+alert.ID.String()+"/status", body).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "id", alert.ID.String())
	w := httptest.NewRecorder()
	h.UpdateAlertStatus(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	resp := decodeBody(t, w)
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected 'data' in response")
	}
	if data["status"] != "acknowledged" {
		t.Errorf("expected status=acknowledged in response, got %v", data["status"])
	}
}

func TestUpdateAlertStatus_ServiceError(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	alertID := uuid.New()

	mock := &mockUEBAService{
		updateAlertStatusFn: func(ctx context.Context, tid, aid uuid.UUID, actorID *uuid.UUID, req *dto.AlertStatusUpdateRequest) (*model.UEBAAlert, error) {
			return nil, fmt.Errorf("invalid status transition")
		},
	}
	h := NewUEBAHandler(mock)

	body := jsonBody(t, map[string]string{"status": "resolved"})
	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("PUT", "/ueba/alerts/"+alertID.String()+"/status", body).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "id", alertID.String())
	w := httptest.NewRecorder()
	h.UpdateAlertStatus(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on service error, got %d", w.Code)
	}
}

func TestUpdateAlertStatus_InvalidBody(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	alertID := uuid.New()

	h := NewUEBAHandler(&mockUEBAService{})
	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("PUT", "/ueba/alerts/"+alertID.String()+"/status", bytes.NewBufferString("{invalid")).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "id", alertID.String())
	w := httptest.NewRecorder()
	h.UpdateAlertStatus(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on invalid JSON, got %d", w.Code)
	}
}

// ---- MarkFalsePositive tests ----------------------------------------------

func TestMarkFalsePositive_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	alert := sampleAlert(tenantID)
	alert.Status = "false_positive"

	mock := &mockUEBAService{
		markFalsePositiveFn: func(ctx context.Context, tid, aid uuid.UUID, actorID *uuid.UUID, req *dto.FalsePositiveRequest) (*model.UEBAAlert, error) {
			if aid != alert.ID {
				t.Errorf("expected alertID %s, got %s", alert.ID, aid)
			}
			if req.Notes != "false alarm — scheduled maintenance" {
				t.Errorf("expected notes, got %q", req.Notes)
			}
			return alert, nil
		},
	}
	h := NewUEBAHandler(mock)

	body := jsonBody(t, map[string]string{"notes": "false alarm — scheduled maintenance"})
	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("POST", "/ueba/alerts/"+alert.ID.String()+"/false-positive", body).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "id", alert.ID.String())
	w := httptest.NewRecorder()
	h.MarkFalsePositive(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	resp := decodeBody(t, w)
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected 'data' in response")
	}
	if data["status"] != "false_positive" {
		t.Errorf("expected status=false_positive, got %v", data["status"])
	}
}

func TestMarkFalsePositive_ServiceError(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	alertID := uuid.New()

	mock := &mockUEBAService{
		markFalsePositiveFn: func(ctx context.Context, tid, aid uuid.UUID, actorID *uuid.UUID, req *dto.FalsePositiveRequest) (*model.UEBAAlert, error) {
			return nil, fmt.Errorf("profile retraining failed")
		},
	}
	h := NewUEBAHandler(mock)

	body := jsonBody(t, map[string]string{"notes": "test"})
	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("POST", "/ueba/alerts/"+alertID.String()+"/false-positive", body).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	r = chiCtx(r, "id", alertID.String())
	w := httptest.NewRecorder()
	h.MarkFalsePositive(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on service error, got %d", w.Code)
	}
}

// ---- BulkUpdateAlertStatus tests ------------------------------------------

func TestBulkUpdateAlertStatus_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	id1 := uuid.New()
	id2 := uuid.New()

	var updatedIDs []uuid.UUID
	mock := &mockUEBAService{
		updateAlertStatusFn: func(ctx context.Context, tid, aid uuid.UUID, actorID *uuid.UUID, req *dto.AlertStatusUpdateRequest) (*model.UEBAAlert, error) {
			updatedIDs = append(updatedIDs, aid)
			return sampleAlert(tid), nil
		},
	}
	h := NewUEBAHandler(mock)

	body := jsonBody(t, map[string]any{
		"alert_ids": []string{id1.String(), id2.String()},
		"status":    "acknowledged",
		"notes":     "bulk ack",
	})
	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("PUT", "/ueba/alerts/bulk/status", body).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.BulkUpdateAlertStatus(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	resp := decodeBody(t, w)
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected 'data' in response")
	}
	if int(data["updated"].(float64)) != 2 {
		t.Errorf("expected updated=2, got %v", data["updated"])
	}
	if int(data["failed"].(float64)) != 0 {
		t.Errorf("expected failed=0, got %v", data["failed"])
	}
	if len(updatedIDs) != 2 {
		t.Errorf("expected 2 service calls, got %d", len(updatedIDs))
	}
}

func TestBulkUpdateAlertStatus_FalsePositive(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	id1 := uuid.New()

	fpCalled := false
	mock := &mockUEBAService{
		markFalsePositiveFn: func(ctx context.Context, tid, aid uuid.UUID, actorID *uuid.UUID, req *dto.FalsePositiveRequest) (*model.UEBAAlert, error) {
			fpCalled = true
			if req.Notes != "bulk fp" {
				t.Errorf("expected notes='bulk fp', got %q", req.Notes)
			}
			return sampleAlert(tid), nil
		},
	}
	h := NewUEBAHandler(mock)

	body := jsonBody(t, map[string]any{
		"alert_ids":      []string{id1.String()},
		"false_positive": true,
		"notes":          "bulk fp",
	})
	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("PUT", "/ueba/alerts/bulk/status", body).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.BulkUpdateAlertStatus(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	if !fpCalled {
		t.Error("expected MarkFalsePositive to be called")
	}
}

func TestBulkUpdateAlertStatus_PartialFailure(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	goodID := uuid.New()
	badID := uuid.New()

	mock := &mockUEBAService{
		updateAlertStatusFn: func(ctx context.Context, tid, aid uuid.UUID, actorID *uuid.UUID, req *dto.AlertStatusUpdateRequest) (*model.UEBAAlert, error) {
			if aid == badID {
				return nil, fmt.Errorf("alert not found")
			}
			return sampleAlert(tid), nil
		},
	}
	h := NewUEBAHandler(mock)

	body := jsonBody(t, map[string]any{
		"alert_ids": []string{goodID.String(), badID.String()},
		"status":    "investigating",
	})
	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("PUT", "/ueba/alerts/bulk/status", body).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.BulkUpdateAlertStatus(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	resp := decodeBody(t, w)
	data := resp["data"].(map[string]any)
	if int(data["updated"].(float64)) != 1 {
		t.Errorf("expected updated=1, got %v", data["updated"])
	}
	if int(data["failed"].(float64)) != 1 {
		t.Errorf("expected failed=1, got %v", data["failed"])
	}
	errs, ok := data["errors"].([]any)
	if !ok || len(errs) != 1 {
		t.Errorf("expected 1 error entry, got %v", data["errors"])
	}
}

func TestBulkUpdateAlertStatus_EmptyIDs(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	h := NewUEBAHandler(&mockUEBAService{})
	body := jsonBody(t, map[string]any{
		"alert_ids": []string{},
		"status":    "acknowledged",
	})
	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("PUT", "/ueba/alerts/bulk/status", body).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.BulkUpdateAlertStatus(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on empty alert_ids, got %d", w.Code)
	}
}

func TestBulkUpdateAlertStatus_InvalidStatus(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	h := NewUEBAHandler(&mockUEBAService{})
	body := jsonBody(t, map[string]any{
		"alert_ids": []string{uuid.New().String()},
		"status":    "invalid_status",
	})
	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("PUT", "/ueba/alerts/bulk/status", body).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.BulkUpdateAlertStatus(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on invalid status, got %d", w.Code)
	}
}

// ---- GetDashboard tests ---------------------------------------------------

func TestGetDashboard_Success(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockUEBAService{
		getDashboardFn: func(ctx context.Context, tid uuid.UUID) (*dto.DashboardResponse, error) {
			return &dto.DashboardResponse{
				KPIs: dto.DashboardKPIs{
					ActiveProfiles:   150,
					HighRiskEntities: 5,
					Alerts7D:         12,
					AverageRiskScore: 28.7,
				},
				RiskRanking:           []dto.RiskRankingItem{},
				AlertTypeDistribution: []dto.ChartDatum{},
				AlertTrend:            []dto.TrendDatum{},
				Profiles:              []dto.RiskRankingItem{},
			}, nil
		},
	}
	h := NewUEBAHandler(mock)

	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("GET", "/ueba/dashboard", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	h.GetDashboard(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := decodeBody(t, w)
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected 'data' in response")
	}
	kpis, ok := data["kpis"].(map[string]any)
	if !ok {
		t.Fatal("expected 'kpis' in data")
	}
	if int(kpis["active_profiles"].(float64)) != 150 {
		t.Errorf("expected active_profiles=150, got %v", kpis["active_profiles"])
	}
}

func TestGetDashboard_ServiceError(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockUEBAService{
		getDashboardFn: func(ctx context.Context, tid uuid.UUID) (*dto.DashboardResponse, error) {
			return nil, fmt.Errorf("connection reset")
		},
	}
	h := NewUEBAHandler(mock)

	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("GET", "/ueba/dashboard", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	h.GetDashboard(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// ---- GetRiskRanking tests -------------------------------------------------

func TestGetRiskRanking_DefaultLimit(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockUEBAService{
		getRiskRankingFn: func(ctx context.Context, tid uuid.UUID, limit int) ([]dto.RiskRankingItem, error) {
			if limit != 20 {
				t.Errorf("expected default limit=20, got %d", limit)
			}
			return []dto.RiskRankingItem{}, nil
		},
	}
	h := NewUEBAHandler(mock)

	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("GET", "/ueba/risk-ranking", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	h.GetRiskRanking(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetRiskRanking_CustomLimit(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockUEBAService{
		getRiskRankingFn: func(ctx context.Context, tid uuid.UUID, limit int) ([]dto.RiskRankingItem, error) {
			if limit != 50 {
				t.Errorf("expected limit=50, got %d", limit)
			}
			return []dto.RiskRankingItem{}, nil
		},
	}
	h := NewUEBAHandler(mock)

	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("GET", "/ueba/risk-ranking?limit=50", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	h.GetRiskRanking(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// ---- GetHeatmap tests -----------------------------------------------------

func TestGetHeatmap_DefaultDays(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockUEBAService{
		getHeatmapFn: func(ctx context.Context, tid uuid.UUID, entityID string, days int) (*dto.HeatmapResponse, error) {
			if days != 30 {
				t.Errorf("expected default days=30, got %d", days)
			}
			return &dto.HeatmapResponse{EntityID: entityID, Days: days}, nil
		},
	}
	h := NewUEBAHandler(mock)

	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("GET", "/ueba/profiles/admin/heatmap", nil).WithContext(ctx)
	r = chiCtx(r, "entityId", "admin")
	w := httptest.NewRecorder()
	h.GetHeatmap(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetHeatmap_CustomDays(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockUEBAService{
		getHeatmapFn: func(ctx context.Context, tid uuid.UUID, entityID string, days int) (*dto.HeatmapResponse, error) {
			if days != 7 {
				t.Errorf("expected days=7, got %d", days)
			}
			return &dto.HeatmapResponse{EntityID: entityID, Days: days}, nil
		},
	}
	h := NewUEBAHandler(mock)

	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("GET", "/ueba/profiles/admin/heatmap?days=7", nil).WithContext(ctx)
	r = chiCtx(r, "entityId", "admin")
	w := httptest.NewRecorder()
	h.GetHeatmap(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// ---- GetProfile tests -----------------------------------------------------

func TestGetProfile_NotFound(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()

	mock := &mockUEBAService{
		getProfileFn: func(ctx context.Context, tid uuid.UUID, entityID string) (*dto.ProfileDetailResponse, error) {
			return nil, cyberrepo.ErrNotFound
		},
	}
	h := NewUEBAHandler(mock)

	ctx := uebaAuthCtx(tenantID, userID)
	r := httptest.NewRequest("GET", "/ueba/profiles/unknown", nil).WithContext(ctx)
	r = chiCtx(r, "entityId", "unknown")
	w := httptest.NewRecorder()
	h.GetProfile(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
