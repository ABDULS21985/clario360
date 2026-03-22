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
// mock + helpers
// ---------------------------------------------------------------------------

type mockThreatFeedService struct {
	listFn    func(ctx context.Context, tenantID uuid.UUID, page, perPage int, search, sort, order string, actor *service.Actor) (*dto.ThreatFeedListResponse, error)
	getFn     func(ctx context.Context, tenantID, feedID uuid.UUID, actor *service.Actor) (*model.ThreatFeedConfig, error)
	createFn  func(ctx context.Context, tenantID, userID uuid.UUID, actor *service.Actor, req *dto.ThreatFeedConfigRequest) (*model.ThreatFeedConfig, error)
	updateFn  func(ctx context.Context, tenantID, feedID uuid.UUID, actor *service.Actor, req *dto.ThreatFeedConfigRequest) (*model.ThreatFeedConfig, error)
	deleteFn  func(ctx context.Context, tenantID, feedID uuid.UUID, actor *service.Actor) error
	syncFn    func(ctx context.Context, tenantID, feedID uuid.UUID, actor *service.Actor) (map[string]interface{}, error)
	historyFn func(ctx context.Context, tenantID, feedID uuid.UUID, actor *service.Actor) ([]*model.ThreatFeedSyncHistory, error)
}

func (m *mockThreatFeedService) ListFeeds(ctx context.Context, tenantID uuid.UUID, page, perPage int, search, sort, order string, actor *service.Actor) (*dto.ThreatFeedListResponse, error) {
	if m.listFn != nil {
		return m.listFn(ctx, tenantID, page, perPage, search, sort, order, actor)
	}
	return &dto.ThreatFeedListResponse{Data: []*model.ThreatFeedConfig{}, Meta: dto.NewPaginationMeta(1, 25, 0)}, nil
}

func (m *mockThreatFeedService) GetFeed(ctx context.Context, tenantID, feedID uuid.UUID, actor *service.Actor) (*model.ThreatFeedConfig, error) {
	if m.getFn != nil {
		return m.getFn(ctx, tenantID, feedID, actor)
	}
	return nil, repository.ErrNotFound
}

func (m *mockThreatFeedService) CreateFeed(ctx context.Context, tenantID, userID uuid.UUID, actor *service.Actor, req *dto.ThreatFeedConfigRequest) (*model.ThreatFeedConfig, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, userID, actor, req)
	}
	return nil, nil
}

func (m *mockThreatFeedService) UpdateFeed(ctx context.Context, tenantID, feedID uuid.UUID, actor *service.Actor, req *dto.ThreatFeedConfigRequest) (*model.ThreatFeedConfig, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, tenantID, feedID, actor, req)
	}
	return nil, nil
}

func (m *mockThreatFeedService) DeleteFeed(ctx context.Context, tenantID, feedID uuid.UUID, actor *service.Actor) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, tenantID, feedID, actor)
	}
	return nil
}

func (m *mockThreatFeedService) SyncFeed(ctx context.Context, tenantID, feedID uuid.UUID, actor *service.Actor) (map[string]interface{}, error) {
	if m.syncFn != nil {
		return m.syncFn(ctx, tenantID, feedID, actor)
	}
	return map[string]interface{}{"indicators_imported": 0}, nil
}

func (m *mockThreatFeedService) ListHistory(ctx context.Context, tenantID, feedID uuid.UUID, actor *service.Actor) ([]*model.ThreatFeedSyncHistory, error) {
	if m.historyFn != nil {
		return m.historyFn(ctx, tenantID, feedID, actor)
	}
	return []*model.ThreatFeedSyncHistory{}, nil
}

// feedAuthRequest creates an *http.Request with tenant + user auth context.
func feedAuthRequest(method, path string, body []byte) *http.Request {
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
		Email:    "admin@example.com",
		Roles:    []string{"security_admin"},
	})
	return r.WithContext(ctx)
}

// feedAuthRequestWithID creates a request with auth context AND chi URL param "feedId".
func feedAuthRequestWithID(method, path string, id uuid.UUID, body []byte) *http.Request {
	r := feedAuthRequest(method, path, body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("feedId", id.String())
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	return r.WithContext(ctx)
}

func sampleFeedConfig() *model.ThreatFeedConfig {
	now := time.Now().UTC()
	feedURL := "https://intel.example.com/feed.json"
	return &model.ThreatFeedConfig{
		ID:                uuid.MustParse("00000000-0000-0000-0000-000000000010"),
		TenantID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Name:              "STIX Community Feed",
		Type:              model.ThreatFeedTypeSTIX,
		URL:               &feedURL,
		AuthType:          model.ThreatFeedAuthNone,
		AuthConfig:        json.RawMessage(`{}`),
		SyncInterval:      model.ThreatFeedIntervalDaily,
		DefaultSeverity:   model.SeverityMedium,
		DefaultConfidence: 0.75,
		DefaultTags:       []string{"community"},
		IndicatorTypes:    []string{},
		Enabled:           true,
		Status:            model.ThreatFeedStatusActive,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// ---------------------------------------------------------------------------
// 1. Auth enforcement
// ---------------------------------------------------------------------------

func TestThreatFeedHandler_NoAuth(t *testing.T) {
	h := NewThreatFeedHandler(&mockThreatFeedService{})
	cases := []struct {
		name   string
		method string
		invoke func(w http.ResponseWriter, r *http.Request)
	}{
		{"List", "GET", h.List},
		{"Get", "GET", h.Get},
		{"Create", "POST", h.Create},
		{"Update", "PUT", h.Update},
		{"Delete", "DELETE", h.Delete},
		{"Sync", "POST", h.Sync},
		{"History", "GET", h.History},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, "/cyber/threat-feeds", nil)
			w := httptest.NewRecorder()
			tc.invoke(w, r)
			if w.Code < 400 {
				t.Errorf("%s: expected 4xx without auth, got %d", tc.name, w.Code)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 2. List
// ---------------------------------------------------------------------------

func TestThreatFeedHandler_List_Success(t *testing.T) {
	sample := sampleFeedConfig()
	h := NewThreatFeedHandler(&mockThreatFeedService{
		listFn: func(_ context.Context, _ uuid.UUID, page, perPage int, search, sort, order string, _ *service.Actor) (*dto.ThreatFeedListResponse, error) {
			if page != 2 {
				t.Errorf("expected page=2, got %d", page)
			}
			if perPage != 10 {
				t.Errorf("expected per_page=10, got %d", perPage)
			}
			if search != "stix" {
				t.Errorf("expected search=stix, got %q", search)
			}
			return &dto.ThreatFeedListResponse{
				Data: []*model.ThreatFeedConfig{sample},
				Meta: dto.NewPaginationMeta(2, 10, 1),
			}, nil
		},
	})

	r := feedAuthRequest("GET", "/cyber/threat-feeds?page=2&per_page=10&search=stix", nil)
	w := httptest.NewRecorder()
	h.List(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestThreatFeedHandler_List_ServiceError(t *testing.T) {
	h := NewThreatFeedHandler(&mockThreatFeedService{
		listFn: func(_ context.Context, _ uuid.UUID, _, _ int, _, _, _ string, _ *service.Actor) (*dto.ThreatFeedListResponse, error) {
			return nil, fmt.Errorf("db down")
		},
	})
	r := feedAuthRequest("GET", "/cyber/threat-feeds", nil)
	w := httptest.NewRecorder()
	h.List(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// 3. Get
// ---------------------------------------------------------------------------

func TestThreatFeedHandler_Get_Success(t *testing.T) {
	sample := sampleFeedConfig()
	h := NewThreatFeedHandler(&mockThreatFeedService{
		getFn: func(_ context.Context, _, feedID uuid.UUID, _ *service.Actor) (*model.ThreatFeedConfig, error) {
			if feedID != sample.ID {
				t.Errorf("wrong feedID: got %v", feedID)
			}
			return sample, nil
		},
	})

	r := feedAuthRequestWithID("GET", "/cyber/threat-feeds/"+sample.ID.String(), sample.ID, nil)
	w := httptest.NewRecorder()
	h.Get(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Error("response missing 'data' envelope")
	}
}

func TestThreatFeedHandler_Get_NotFound(t *testing.T) {
	h := NewThreatFeedHandler(&mockThreatFeedService{
		getFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor) (*model.ThreatFeedConfig, error) {
			return nil, repository.ErrNotFound
		},
	})

	id := uuid.New()
	r := feedAuthRequestWithID("GET", "/cyber/threat-feeds/"+id.String(), id, nil)
	w := httptest.NewRecorder()
	h.Get(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestThreatFeedHandler_Get_InvalidUUID(t *testing.T) {
	h := NewThreatFeedHandler(&mockThreatFeedService{})
	r := feedAuthRequest("GET", "/cyber/threat-feeds/not-a-uuid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("feedId", "not-a-uuid")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h.Get(w, r)
	if w.Code < 400 {
		t.Errorf("expected 4xx for invalid UUID, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// 4. Create
// ---------------------------------------------------------------------------

func TestThreatFeedHandler_Create_Success(t *testing.T) {
	sample := sampleFeedConfig()
	h := NewThreatFeedHandler(&mockThreatFeedService{
		createFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor, req *dto.ThreatFeedConfigRequest) (*model.ThreatFeedConfig, error) {
			if req.Name != "My Feed" {
				t.Errorf("expected name=My Feed, got %q", req.Name)
			}
			return sample, nil
		},
	})

	body, _ := json.Marshal(dto.ThreatFeedConfigRequest{
		Name:              "My Feed",
		Type:              model.ThreatFeedTypeSTIX,
		URL:               "https://example.com/feed.json",
		AuthType:          model.ThreatFeedAuthNone,
		SyncInterval:      model.ThreatFeedIntervalDaily,
		DefaultSeverity:   model.SeverityMedium,
		DefaultConfidence: 0.75,
		Enabled:           true,
	})
	r := feedAuthRequest("POST", "/cyber/threat-feeds", body)
	w := httptest.NewRecorder()
	h.Create(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestThreatFeedHandler_Create_ServiceError(t *testing.T) {
	h := NewThreatFeedHandler(&mockThreatFeedService{
		createFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor, _ *dto.ThreatFeedConfigRequest) (*model.ThreatFeedConfig, error) {
			return nil, fmt.Errorf("feed URL is required")
		},
	})
	body, _ := json.Marshal(dto.ThreatFeedConfigRequest{
		Name:            "Broken",
		Type:            model.ThreatFeedTypeSTIX,
		AuthType:        model.ThreatFeedAuthNone,
		SyncInterval:    model.ThreatFeedIntervalDaily,
		DefaultSeverity: model.SeverityMedium,
		Enabled:         true,
	})
	r := feedAuthRequest("POST", "/cyber/threat-feeds", body)
	w := httptest.NewRecorder()
	h.Create(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// 5. Update
// ---------------------------------------------------------------------------

func TestThreatFeedHandler_Update_Success(t *testing.T) {
	sample := sampleFeedConfig()
	h := NewThreatFeedHandler(&mockThreatFeedService{
		updateFn: func(_ context.Context, _, feedID uuid.UUID, _ *service.Actor, req *dto.ThreatFeedConfigRequest) (*model.ThreatFeedConfig, error) {
			if feedID != sample.ID {
				t.Errorf("wrong feedID: got %v", feedID)
			}
			return sample, nil
		},
	})
	body, _ := json.Marshal(dto.ThreatFeedConfigRequest{
		Name:              "Updated Feed",
		Type:              model.ThreatFeedTypeSTIX,
		URL:               "https://example.com/feed.json",
		AuthType:          model.ThreatFeedAuthNone,
		SyncInterval:      model.ThreatFeedIntervalHourly,
		DefaultSeverity:   model.SeverityHigh,
		DefaultConfidence: 0.9,
		Enabled:           true,
	})
	r := feedAuthRequestWithID("PUT", "/cyber/threat-feeds/"+sample.ID.String(), sample.ID, body)
	w := httptest.NewRecorder()
	h.Update(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// 6. Delete
// ---------------------------------------------------------------------------

func TestThreatFeedHandler_Delete_Success(t *testing.T) {
	var deletedID uuid.UUID
	h := NewThreatFeedHandler(&mockThreatFeedService{
		deleteFn: func(_ context.Context, _, feedID uuid.UUID, _ *service.Actor) error {
			deletedID = feedID
			return nil
		},
	})

	id := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	r := feedAuthRequestWithID("DELETE", "/cyber/threat-feeds/"+id.String(), id, nil)
	w := httptest.NewRecorder()
	h.Delete(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if deletedID != id {
		t.Errorf("service received wrong feed ID: got %v, want %v", deletedID, id)
	}
}

func TestThreatFeedHandler_Delete_NotFound(t *testing.T) {
	h := NewThreatFeedHandler(&mockThreatFeedService{
		deleteFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor) error {
			return repository.ErrNotFound
		},
	})

	id := uuid.New()
	r := feedAuthRequestWithID("DELETE", "/cyber/threat-feeds/"+id.String(), id, nil)
	w := httptest.NewRecorder()
	h.Delete(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestThreatFeedHandler_Delete_InternalError(t *testing.T) {
	h := NewThreatFeedHandler(&mockThreatFeedService{
		deleteFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor) error {
			return fmt.Errorf("database connection refused")
		},
	})

	id := uuid.New()
	r := feedAuthRequestWithID("DELETE", "/cyber/threat-feeds/"+id.String(), id, nil)
	w := httptest.NewRecorder()
	h.Delete(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestThreatFeedHandler_Delete_InvalidUUID(t *testing.T) {
	h := NewThreatFeedHandler(&mockThreatFeedService{})
	r := feedAuthRequest("DELETE", "/cyber/threat-feeds/not-a-uuid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("feedId", "not-a-uuid")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	if w.Code < 400 {
		t.Errorf("expected 4xx for invalid UUID, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// 7. Sync
// ---------------------------------------------------------------------------

func TestThreatFeedHandler_Sync_Success(t *testing.T) {
	h := NewThreatFeedHandler(&mockThreatFeedService{
		syncFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor) (map[string]interface{}, error) {
			return map[string]interface{}{
				"feed_id":             "feed-1",
				"indicators_imported": 42,
				"indicators_parsed":   50,
			}, nil
		},
	})

	id := uuid.New()
	r := feedAuthRequestWithID("POST", "/cyber/threat-feeds/"+id.String()+"/sync", id, nil)
	w := httptest.NewRecorder()
	h.Sync(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("response missing 'data' envelope")
	}
	if data["indicators_imported"].(float64) != 42 {
		t.Errorf("expected 42 imported, got %v", data["indicators_imported"])
	}
}

func TestThreatFeedHandler_Sync_NotFound(t *testing.T) {
	h := NewThreatFeedHandler(&mockThreatFeedService{
		syncFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor) (map[string]interface{}, error) {
			return nil, &service.SyncError{Kind: service.SyncErrNotFound, Cause: fmt.Errorf("not found")}
		},
	})
	id := uuid.New()
	r := feedAuthRequestWithID("POST", "/cyber/threat-feeds/"+id.String()+"/sync", id, nil)
	w := httptest.NewRecorder()
	h.Sync(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestThreatFeedHandler_Sync_UpstreamError(t *testing.T) {
	h := NewThreatFeedHandler(&mockThreatFeedService{
		syncFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor) (map[string]interface{}, error) {
			return nil, &service.SyncError{Kind: service.SyncErrUpstream, Cause: fmt.Errorf("feed returned 503")}
		},
	})
	id := uuid.New()
	r := feedAuthRequestWithID("POST", "/cyber/threat-feeds/"+id.String()+"/sync", id, nil)
	w := httptest.NewRecorder()
	h.Sync(w, r)
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestThreatFeedHandler_Sync_ParseError(t *testing.T) {
	h := NewThreatFeedHandler(&mockThreatFeedService{
		syncFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor) (map[string]interface{}, error) {
			return nil, &service.SyncError{Kind: service.SyncErrParse, Cause: fmt.Errorf("invalid JSON")}
		},
	})
	id := uuid.New()
	r := feedAuthRequestWithID("POST", "/cyber/threat-feeds/"+id.String()+"/sync", id, nil)
	w := httptest.NewRecorder()
	h.Sync(w, r)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// 8. History
// ---------------------------------------------------------------------------

func TestThreatFeedHandler_History_Success(t *testing.T) {
	now := time.Now().UTC()
	h := NewThreatFeedHandler(&mockThreatFeedService{
		historyFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor) ([]*model.ThreatFeedSyncHistory, error) {
			return []*model.ThreatFeedSyncHistory{
				{
					ID:                 uuid.New(),
					TenantID:           uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					FeedID:             uuid.MustParse("00000000-0000-0000-0000-000000000010"),
					Status:             "completed",
					IndicatorsImported: 15,
					DurationMs:         1200,
					Metadata:           json.RawMessage(`{}`),
					StartedAt:          now,
				},
			}, nil
		},
	})

	id := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	r := feedAuthRequestWithID("GET", "/cyber/threat-feeds/"+id.String()+"/history", id, nil)
	w := httptest.NewRecorder()
	h.History(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := resp["data"]; !ok {
		t.Error("response missing 'data' envelope")
	}
}

func TestThreatFeedHandler_History_ServiceError(t *testing.T) {
	h := NewThreatFeedHandler(&mockThreatFeedService{
		historyFn: func(_ context.Context, _, _ uuid.UUID, _ *service.Actor) ([]*model.ThreatFeedSyncHistory, error) {
			return nil, fmt.Errorf("db error")
		},
	})
	id := uuid.New()
	r := feedAuthRequestWithID("GET", "/cyber/threat-feeds/"+id.String()+"/history", id, nil)
	w := httptest.NewRecorder()
	h.History(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
