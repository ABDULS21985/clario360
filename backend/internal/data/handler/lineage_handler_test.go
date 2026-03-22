package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/service"
)

// ---------------------------------------------------------------------------
// mock lineageService
// ---------------------------------------------------------------------------

type mockLineageService struct {
	fullGraphFn   func(ctx context.Context, tenantID uuid.UUID) (*model.LineageGraph, error)
	entityGraphFn func(ctx context.Context, tenantID uuid.UUID, entityType model.LineageEntityType, entityID uuid.UUID, depth int) (*model.LineageGraph, error)
	upstreamFn    func(ctx context.Context, tenantID uuid.UUID, entityType model.LineageEntityType, entityID uuid.UUID, depth int) (*model.LineageGraph, error)
	downstreamFn  func(ctx context.Context, tenantID uuid.UUID, entityType model.LineageEntityType, entityID uuid.UUID, depth int) (*model.LineageGraph, error)
	impactFn      func(ctx context.Context, tenantID uuid.UUID, entityType model.LineageEntityType, entityID uuid.UUID) (*model.ImpactAnalysis, error)
	recordFn      func(ctx context.Context, tenantID uuid.UUID, req dto.RecordLineageEdgeRequest) (*model.LineageEdgeRecord, error)
	deleteEdgeFn  func(ctx context.Context, tenantID, edgeID uuid.UUID) error
	searchFn      func(ctx context.Context, tenantID uuid.UUID, params dto.SearchLineageParams) ([]model.LineageSearchResult, error)
	statsFn       func(ctx context.Context, tenantID uuid.UUID) (*model.LineageStatsSummary, error)
}

func (m *mockLineageService) FullGraph(ctx context.Context, tenantID uuid.UUID) (*model.LineageGraph, error) {
	if m.fullGraphFn != nil {
		return m.fullGraphFn(ctx, tenantID)
	}
	return nil, nil
}
func (m *mockLineageService) EntityGraph(ctx context.Context, tenantID uuid.UUID, entityType model.LineageEntityType, entityID uuid.UUID, depth int) (*model.LineageGraph, error) {
	if m.entityGraphFn != nil {
		return m.entityGraphFn(ctx, tenantID, entityType, entityID, depth)
	}
	return nil, nil
}
func (m *mockLineageService) Upstream(ctx context.Context, tenantID uuid.UUID, entityType model.LineageEntityType, entityID uuid.UUID, depth int) (*model.LineageGraph, error) {
	if m.upstreamFn != nil {
		return m.upstreamFn(ctx, tenantID, entityType, entityID, depth)
	}
	return nil, nil
}
func (m *mockLineageService) Downstream(ctx context.Context, tenantID uuid.UUID, entityType model.LineageEntityType, entityID uuid.UUID, depth int) (*model.LineageGraph, error) {
	if m.downstreamFn != nil {
		return m.downstreamFn(ctx, tenantID, entityType, entityID, depth)
	}
	return nil, nil
}
func (m *mockLineageService) Impact(ctx context.Context, tenantID uuid.UUID, entityType model.LineageEntityType, entityID uuid.UUID) (*model.ImpactAnalysis, error) {
	if m.impactFn != nil {
		return m.impactFn(ctx, tenantID, entityType, entityID)
	}
	return nil, nil
}
func (m *mockLineageService) Record(ctx context.Context, tenantID uuid.UUID, req dto.RecordLineageEdgeRequest) (*model.LineageEdgeRecord, error) {
	if m.recordFn != nil {
		return m.recordFn(ctx, tenantID, req)
	}
	return nil, nil
}
func (m *mockLineageService) DeleteEdge(ctx context.Context, tenantID, edgeID uuid.UUID) error {
	if m.deleteEdgeFn != nil {
		return m.deleteEdgeFn(ctx, tenantID, edgeID)
	}
	return nil
}
func (m *mockLineageService) Search(ctx context.Context, tenantID uuid.UUID, params dto.SearchLineageParams) ([]model.LineageSearchResult, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, tenantID, params)
	}
	return nil, nil
}
func (m *mockLineageService) Stats(ctx context.Context, tenantID uuid.UUID) (*model.LineageStatsSummary, error) {
	if m.statsFn != nil {
		return m.statsFn(ctx, tenantID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func sampleGraph() *model.LineageGraph {
	return &model.LineageGraph{
		Nodes: []model.LineageNode{{ID: uuid.New().String(), Name: "src", Type: "source"}},
		Edges: []model.LineageEdge{},
	}
}

func newLineageHandler(svc *mockLineageService) *LineageHandler {
	return NewLineageHandler(svc, testLogger)
}

// ---------------------------------------------------------------------------
// Auth enforcement
// ---------------------------------------------------------------------------

func TestLineageHandler_FullGraph_Unauthorized(t *testing.T) {
	h := newLineageHandler(&mockLineageService{})
	w := httptest.NewRecorder()
	r := unauthRequest(http.MethodGet, "/api/v1/data/lineage/graph", nil)
	h.FullGraph(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestLineageHandler_Search_Unauthorized(t *testing.T) {
	h := newLineageHandler(&mockLineageService{})
	w := httptest.NewRecorder()
	r := unauthRequest(http.MethodGet, "/api/v1/data/lineage/search?q=test", nil)
	h.Search(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Error mapping
// ---------------------------------------------------------------------------

func TestLineageHandler_ErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"validation", fmt.Errorf("bad: %w", service.ErrValidation), http.StatusBadRequest},
		{"generic", fmt.Errorf("oops"), http.StatusInternalServerError},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockLineageService{
				fullGraphFn: func(_ context.Context, _ uuid.UUID) (*model.LineageGraph, error) {
					return nil, tc.err
				},
			}
			h := newLineageHandler(svc)
			w := httptest.NewRecorder()
			r := authRequest(http.MethodGet, "/api/v1/data/lineage/graph", nil)
			h.FullGraph(w, r)
			if w.Code != tc.wantStatus {
				t.Fatalf("expected %d for %s, got %d", tc.wantStatus, tc.name, w.Code)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Happy paths
// ---------------------------------------------------------------------------

func TestLineageHandler_FullGraph_Success(t *testing.T) {
	graph := sampleGraph()
	svc := &mockLineageService{
		fullGraphFn: func(_ context.Context, _ uuid.UUID) (*model.LineageGraph, error) {
			return graph, nil
		},
	}
	h := newLineageHandler(svc)
	w := httptest.NewRecorder()
	r := authRequest(http.MethodGet, "/api/v1/data/lineage/graph", nil)
	h.FullGraph(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLineageHandler_EntityGraph_Success(t *testing.T) {
	graph := sampleGraph()
	entityID := uuid.New()
	svc := &mockLineageService{
		entityGraphFn: func(_ context.Context, _ uuid.UUID, et model.LineageEntityType, eid uuid.UUID, depth int) (*model.LineageGraph, error) {
			if et != "data_source" {
				t.Errorf("expected entityType=data_source, got %s", et)
			}
			if eid != entityID {
				t.Errorf("expected entityID=%s, got %s", entityID, eid)
			}
			if depth != 3 {
				t.Errorf("expected default depth=3, got %d", depth)
			}
			return graph, nil
		},
	}
	h := newLineageHandler(svc)
	w := httptest.NewRecorder()
	r := authRequestWithParams(http.MethodGet, "/api/v1/data/lineage/graph/data_source/"+entityID.String(), map[string]string{
		"entityType": "data_source",
		"entityId":   entityID.String(),
	}, nil)
	h.EntityGraph(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLineageHandler_Impact_Success(t *testing.T) {
	entityID := uuid.New()
	svc := &mockLineageService{
		impactFn: func(_ context.Context, _ uuid.UUID, _ model.LineageEntityType, _ uuid.UUID) (*model.ImpactAnalysis, error) {
			return &model.ImpactAnalysis{TotalAffected: 5, Severity: "high", Summary: "5 affected"}, nil
		},
	}
	h := newLineageHandler(svc)
	w := httptest.NewRecorder()
	r := authRequestWithParams(http.MethodGet, "/api/v1/data/lineage/impact/data_source/"+entityID.String(), map[string]string{
		"entityType": "data_source",
		"entityId":   entityID.String(),
	}, nil)
	h.Impact(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLineageHandler_Record_Success(t *testing.T) {
	svc := &mockLineageService{
		recordFn: func(_ context.Context, _ uuid.UUID, _ dto.RecordLineageEdgeRequest) (*model.LineageEdgeRecord, error) {
			return &model.LineageEdgeRecord{ID: uuid.New()}, nil
		},
	}
	h := newLineageHandler(svc)
	body, _ := json.Marshal(dto.RecordLineageEdgeRequest{
		SourceType: "source",
		SourceID:   uuid.New(),
		TargetType: "model",
		TargetID:   uuid.New(),
	})
	w := httptest.NewRecorder()
	r := authRequest(http.MethodPost, "/api/v1/data/lineage/record", body)
	h.Record(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLineageHandler_DeleteEdge_Success(t *testing.T) {
	svc := &mockLineageService{
		deleteEdgeFn: func(_ context.Context, _, _ uuid.UUID) error { return nil },
	}
	h := newLineageHandler(svc)
	w := httptest.NewRecorder()
	r := authRequestWithID(http.MethodDelete, "/api/v1/data/lineage/edges/x", uuid.New(), nil)
	h.DeleteEdge(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestLineageHandler_Search_Success(t *testing.T) {
	var capturedParams dto.SearchLineageParams
	svc := &mockLineageService{
		searchFn: func(_ context.Context, _ uuid.UUID, params dto.SearchLineageParams) ([]model.LineageSearchResult, error) {
			capturedParams = params
			return []model.LineageSearchResult{
				{Node: model.LineageNode{Name: "test_src", Type: "source"}, Score: 0.95},
			}, nil
		},
	}
	h := newLineageHandler(svc)
	w := httptest.NewRecorder()
	r := authRequest(http.MethodGet, "/api/v1/data/lineage/search?query=test&type=data_source&limit=10", nil)
	h.Search(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if capturedParams.Query != "test" {
		t.Errorf("expected q=test, got %q", capturedParams.Query)
	}
	if capturedParams.Type != "data_source" {
		t.Errorf("expected type=data_source, got %q", capturedParams.Type)
	}
	if capturedParams.Limit != 10 {
		t.Errorf("expected limit=10, got %d", capturedParams.Limit)
	}
}

func TestLineageHandler_Stats_Success(t *testing.T) {
	svc := &mockLineageService{
		statsFn: func(_ context.Context, _ uuid.UUID) (*model.LineageStatsSummary, error) {
			return &model.LineageStatsSummary{NodeCount: 10, EdgeCount: 15}, nil
		},
	}
	h := newLineageHandler(svc)
	w := httptest.NewRecorder()
	r := authRequest(http.MethodGet, "/api/v1/data/lineage/stats", nil)
	h.Stats(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Parameter validation
// ---------------------------------------------------------------------------

func TestLineageHandler_EntityGraph_InvalidEntityType(t *testing.T) {
	h := newLineageHandler(&mockLineageService{})
	w := httptest.NewRecorder()
	r := authRequestWithParams(http.MethodGet, "/api/v1/data/lineage/graph/invalid_type/"+uuid.New().String(), map[string]string{
		"entityType": "invalid_type",
		"entityId":   uuid.New().String(),
	}, nil)
	h.EntityGraph(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid entityType, got %d", w.Code)
	}
}

func TestLineageHandler_EntityGraph_InvalidEntityID(t *testing.T) {
	h := newLineageHandler(&mockLineageService{})
	w := httptest.NewRecorder()
	r := authRequestWithParams(http.MethodGet, "/api/v1/data/lineage/graph/source/not-a-uuid", map[string]string{
		"entityType": "source",
		"entityId":   "not-a-uuid",
	}, nil)
	h.EntityGraph(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid entityId, got %d", w.Code)
	}
}

func TestLineageHandler_Upstream_CustomDepth(t *testing.T) {
	var capturedDepth int
	svc := &mockLineageService{
		upstreamFn: func(_ context.Context, _ uuid.UUID, _ model.LineageEntityType, _ uuid.UUID, depth int) (*model.LineageGraph, error) {
			capturedDepth = depth
			return sampleGraph(), nil
		},
	}
	h := newLineageHandler(svc)
	entityID := uuid.New()
	w := httptest.NewRecorder()
	r := authRequestWithParams(http.MethodGet, "/api/v1/data/lineage/upstream/data_source/"+entityID.String()+"?depth=5", map[string]string{
		"entityType": "data_source",
		"entityId":   entityID.String(),
	}, nil)
	h.Upstream(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if capturedDepth != 5 {
		t.Errorf("expected depth=5, got %d", capturedDepth)
	}
}
