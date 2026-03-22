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

	"github.com/clario360/platform/internal/data/connector"
	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/service"
	"github.com/jackc/pgx/v5"
)

// ---------------------------------------------------------------------------
// mock sourceService
// ---------------------------------------------------------------------------

type mockSourceService struct {
	createFn           func(ctx context.Context, tenantID, userID uuid.UUID, req dto.CreateSourceRequest) (*model.DataSource, error)
	listFn             func(ctx context.Context, tenantID uuid.UUID, params dto.ListSourcesParams) ([]*model.DataSource, int, error)
	getFn              func(ctx context.Context, tenantID, id uuid.UUID) (*model.DataSource, error)
	updateFn           func(ctx context.Context, tenantID, userID, id uuid.UUID, req dto.UpdateSourceRequest) (*model.DataSource, error)
	deleteFn           func(ctx context.Context, tenantID, id uuid.UUID) error
	changeStatusFn     func(ctx context.Context, tenantID, id uuid.UUID, status model.DataSourceStatus) (*model.DataSource, error)
	testConnectionFn   func(ctx context.Context, tenantID, id uuid.UUID) (*connector.ConnectionTestResult, error)
	testConfigFn       func(ctx context.Context, tenantID uuid.UUID, req dto.TestSourceConfigRequest) (*connector.ConnectionTestResult, error)
	discoverSchemaFn   func(ctx context.Context, tenantID, id uuid.UUID) (*model.DiscoveredSchema, error)
	getSchemaFn        func(ctx context.Context, tenantID, id uuid.UUID) (*model.DiscoveredSchema, error)
	triggerSyncFn      func(ctx context.Context, tenantID, id uuid.UUID, syncType model.SyncType, userID *uuid.UUID) (*model.SyncHistory, error)
	listSyncHistoryFn  func(ctx context.Context, tenantID, id uuid.UUID, limit int) ([]*model.SyncHistory, error)
	getStatsFn         func(ctx context.Context, tenantID, id uuid.UUID) (*dto.SourceStatsResponse, error)
	aggregateStatsFn   func(ctx context.Context, tenantID uuid.UUID) (*dto.AggregateSourceStatsResponse, error)
}

func (m *mockSourceService) Create(ctx context.Context, tenantID, userID uuid.UUID, req dto.CreateSourceRequest) (*model.DataSource, error) {
	if m.createFn != nil {
		return m.createFn(ctx, tenantID, userID, req)
	}
	return nil, nil
}
func (m *mockSourceService) List(ctx context.Context, tenantID uuid.UUID, params dto.ListSourcesParams) ([]*model.DataSource, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, tenantID, params)
	}
	return nil, 0, nil
}
func (m *mockSourceService) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.DataSource, error) {
	if m.getFn != nil {
		return m.getFn(ctx, tenantID, id)
	}
	return nil, nil
}
func (m *mockSourceService) Update(ctx context.Context, tenantID, userID, id uuid.UUID, req dto.UpdateSourceRequest) (*model.DataSource, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, tenantID, userID, id, req)
	}
	return nil, nil
}
func (m *mockSourceService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, tenantID, id)
	}
	return nil
}
func (m *mockSourceService) ChangeStatus(ctx context.Context, tenantID, id uuid.UUID, status model.DataSourceStatus) (*model.DataSource, error) {
	if m.changeStatusFn != nil {
		return m.changeStatusFn(ctx, tenantID, id, status)
	}
	return nil, nil
}
func (m *mockSourceService) TestConnection(ctx context.Context, tenantID, id uuid.UUID) (*connector.ConnectionTestResult, error) {
	if m.testConnectionFn != nil {
		return m.testConnectionFn(ctx, tenantID, id)
	}
	return nil, nil
}
func (m *mockSourceService) TestConfig(ctx context.Context, tenantID uuid.UUID, req dto.TestSourceConfigRequest) (*connector.ConnectionTestResult, error) {
	if m.testConfigFn != nil {
		return m.testConfigFn(ctx, tenantID, req)
	}
	return nil, nil
}
func (m *mockSourceService) DiscoverSchema(ctx context.Context, tenantID, id uuid.UUID) (*model.DiscoveredSchema, error) {
	if m.discoverSchemaFn != nil {
		return m.discoverSchemaFn(ctx, tenantID, id)
	}
	return nil, nil
}
func (m *mockSourceService) GetSchema(ctx context.Context, tenantID, id uuid.UUID) (*model.DiscoveredSchema, error) {
	if m.getSchemaFn != nil {
		return m.getSchemaFn(ctx, tenantID, id)
	}
	return nil, nil
}
func (m *mockSourceService) TriggerSync(ctx context.Context, tenantID, id uuid.UUID, syncType model.SyncType, userID *uuid.UUID) (*model.SyncHistory, error) {
	if m.triggerSyncFn != nil {
		return m.triggerSyncFn(ctx, tenantID, id, syncType, userID)
	}
	return nil, nil
}
func (m *mockSourceService) ListSyncHistory(ctx context.Context, tenantID, id uuid.UUID, limit int) ([]*model.SyncHistory, error) {
	if m.listSyncHistoryFn != nil {
		return m.listSyncHistoryFn(ctx, tenantID, id, limit)
	}
	return nil, nil
}
func (m *mockSourceService) GetStats(ctx context.Context, tenantID, id uuid.UUID) (*dto.SourceStatsResponse, error) {
	if m.getStatsFn != nil {
		return m.getStatsFn(ctx, tenantID, id)
	}
	return nil, nil
}
func (m *mockSourceService) AggregateStats(ctx context.Context, tenantID uuid.UUID) (*dto.AggregateSourceStatsResponse, error) {
	if m.aggregateStatsFn != nil {
		return m.aggregateStatsFn(ctx, tenantID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// mock connectorRegistry
// ---------------------------------------------------------------------------

type mockConnectorRegistry struct {
	listTypesFn    func() []model.DataSourceType
	typeMetadataFn func(model.DataSourceType) *connector.ConnectorTypeMetadata
}

func (m *mockConnectorRegistry) ListTypes() []model.DataSourceType {
	if m.listTypesFn != nil {
		return m.listTypesFn()
	}
	return nil
}
func (m *mockConnectorRegistry) TypeMetadata(t model.DataSourceType) *connector.ConnectorTypeMetadata {
	if m.typeMetadataFn != nil {
		return m.typeMetadataFn(t)
	}
	return nil
}

// ---------------------------------------------------------------------------
// sample data
// ---------------------------------------------------------------------------

func sampleSource() *model.DataSource {
	now := time.Now()
	return &model.DataSource{
		ID:        uuid.New(),
		TenantID:  testTenantID,
		Name:      "test-pg",
		Type:      "postgresql",
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func newSourceHandler(svc *mockSourceService) *SourceHandler {
	reg := &mockConnectorRegistry{}
	return NewSourceHandler(svc, reg, testLogger)
}

// ---------------------------------------------------------------------------
// Auth enforcement
// ---------------------------------------------------------------------------

func TestSourceHandler_Create_Unauthorized(t *testing.T) {
	h := newSourceHandler(&mockSourceService{})
	w := httptest.NewRecorder()
	r := unauthRequest(http.MethodPost, "/api/v1/data/sources", []byte(`{}`))
	h.Create(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSourceHandler_List_Unauthorized(t *testing.T) {
	h := newSourceHandler(&mockSourceService{})
	w := httptest.NewRecorder()
	r := unauthRequest(http.MethodGet, "/api/v1/data/sources", nil)
	h.List(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSourceHandler_Get_Unauthorized(t *testing.T) {
	h := newSourceHandler(&mockSourceService{})
	w := httptest.NewRecorder()
	r := unauthRequest(http.MethodGet, "/api/v1/data/sources/x", nil)
	h.Get(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Error mapping
// ---------------------------------------------------------------------------

func TestSourceHandler_ErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"validation", fmt.Errorf("bad: %w", service.ErrValidation), http.StatusBadRequest},
		{"conflict", fmt.Errorf("dup: %w", service.ErrConflict), http.StatusConflict},
		{"not_found", pgx.ErrNoRows, http.StatusNotFound},
		{"too_many_requests", service.ErrTooManyRequests, http.StatusTooManyRequests},
		{"forbidden", service.ErrForbiddenOperation, http.StatusForbidden},
		{"connection_test", fmt.Errorf("conn: %w", service.ErrConnectionTestFailed), http.StatusUnprocessableEntity},
		{"timeout", service.ErrTimeout, http.StatusGatewayTimeout},
		{"generic", fmt.Errorf("something went wrong"), http.StatusInternalServerError},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockSourceService{
				getFn: func(_ context.Context, _, _ uuid.UUID) (*model.DataSource, error) {
					return nil, tc.err
				},
			}
			h := newSourceHandler(svc)
			w := httptest.NewRecorder()
			r := authRequestWithID(http.MethodGet, "/api/v1/data/sources/x", uuid.New(), nil)
			h.Get(w, r)
			if w.Code != tc.wantStatus {
				t.Fatalf("expected %d for %s, got %d", tc.wantStatus, tc.name, w.Code)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Happy paths
// ---------------------------------------------------------------------------

func TestSourceHandler_Create_Success(t *testing.T) {
	src := sampleSource()
	svc := &mockSourceService{
		createFn: func(_ context.Context, _, _ uuid.UUID, _ dto.CreateSourceRequest) (*model.DataSource, error) {
			return src, nil
		},
	}
	h := newSourceHandler(svc)
	body, _ := json.Marshal(dto.CreateSourceRequest{Name: "pg", Type: "postgresql"})
	w := httptest.NewRecorder()
	r := authRequest(http.MethodPost, "/api/v1/data/sources", body)
	h.Create(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSourceHandler_List_Success(t *testing.T) {
	src := sampleSource()
	svc := &mockSourceService{
		listFn: func(_ context.Context, _ uuid.UUID, params dto.ListSourcesParams) ([]*model.DataSource, int, error) {
			if params.Search != "test" {
				t.Errorf("expected search=test, got %q", params.Search)
			}
			return []*model.DataSource{src}, 1, nil
		},
	}
	h := newSourceHandler(svc)
	w := httptest.NewRecorder()
	r := authRequest(http.MethodGet, "/api/v1/data/sources?search=test&page=1&per_page=10", nil)
	h.List(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSourceHandler_Get_Success(t *testing.T) {
	src := sampleSource()
	svc := &mockSourceService{
		getFn: func(_ context.Context, _, id uuid.UUID) (*model.DataSource, error) {
			if id != src.ID {
				t.Errorf("expected id %s, got %s", src.ID, id)
			}
			return src, nil
		},
	}
	h := newSourceHandler(svc)
	w := httptest.NewRecorder()
	r := authRequestWithID(http.MethodGet, "/api/v1/data/sources/x", src.ID, nil)
	h.Get(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSourceHandler_Delete_Success(t *testing.T) {
	svc := &mockSourceService{
		deleteFn: func(_ context.Context, _, _ uuid.UUID) error { return nil },
	}
	h := newSourceHandler(svc)
	w := httptest.NewRecorder()
	r := authRequestWithID(http.MethodDelete, "/api/v1/data/sources/x", uuid.New(), nil)
	h.Delete(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestSourceHandler_ChangeStatus_Success(t *testing.T) {
	src := sampleSource()
	var capturedStatus model.DataSourceStatus
	svc := &mockSourceService{
		changeStatusFn: func(_ context.Context, _, _ uuid.UUID, status model.DataSourceStatus) (*model.DataSource, error) {
			capturedStatus = status
			return src, nil
		},
	}
	h := newSourceHandler(svc)
	body, _ := json.Marshal(dto.ChangeStatusRequest{Status: "inactive"})
	w := httptest.NewRecorder()
	r := authRequestWithID(http.MethodPatch, "/api/v1/data/sources/x/status", src.ID, body)
	h.ChangeStatus(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if capturedStatus != "inactive" {
		t.Fatalf("expected status inactive, got %s", capturedStatus)
	}
}

func TestSourceHandler_AggregateStats_Success(t *testing.T) {
	svc := &mockSourceService{
		aggregateStatsFn: func(_ context.Context, _ uuid.UUID) (*dto.AggregateSourceStatsResponse, error) {
			return &dto.AggregateSourceStatsResponse{}, nil
		},
	}
	h := newSourceHandler(svc)
	w := httptest.NewRecorder()
	r := authRequest(http.MethodGet, "/api/v1/data/sources/stats", nil)
	h.GetAggregateStats(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Parameter parsing
// ---------------------------------------------------------------------------

func TestSourceHandler_List_ParsesFilters(t *testing.T) {
	var captured dto.ListSourcesParams
	svc := &mockSourceService{
		listFn: func(_ context.Context, _ uuid.UUID, params dto.ListSourcesParams) ([]*model.DataSource, int, error) {
			captured = params
			return nil, 0, nil
		},
	}
	h := newSourceHandler(svc)
	w := httptest.NewRecorder()
	r := authRequest(http.MethodGet, "/api/v1/data/sources?type=postgresql,mysql&status=active&has_schema=true&sort=name&order=asc", nil)
	h.List(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if len(captured.Types) != 2 {
		t.Errorf("expected 2 types, got %d: %v", len(captured.Types), captured.Types)
	}
	if len(captured.Statuses) != 1 {
		t.Errorf("expected 1 status, got %d: %v", len(captured.Statuses), captured.Statuses)
	}
	if captured.HasSchema == nil || !*captured.HasSchema {
		t.Error("expected has_schema=true")
	}
	if captured.Sort != "name" {
		t.Errorf("expected sort=name, got %q", captured.Sort)
	}
}

func TestSourceHandler_ListSourceTypes(t *testing.T) {
	reg := &mockConnectorRegistry{
		listTypesFn: func() []model.DataSourceType { return []model.DataSourceType{"postgresql", "mysql"} },
		typeMetadataFn: func(t model.DataSourceType) *connector.ConnectorTypeMetadata {
			return &connector.ConnectorTypeMetadata{Type: t, DisplayName: string(t)}
		},
	}
	h := NewSourceHandler(&mockSourceService{}, reg, testLogger)
	w := httptest.NewRecorder()
	r := authRequest(http.MethodGet, "/api/v1/data/source-types", nil)
	h.ListSourceTypes(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
