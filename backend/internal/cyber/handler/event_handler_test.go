package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
)

// ---------------------------------------------------------------------------
// mock
// ---------------------------------------------------------------------------

type mockEventRepository struct {
	querySecurityEventsFn  func(ctx context.Context, tenantID uuid.UUID, params *model.EventQueryParams) ([]model.SecurityEvent, int, error)
	getSecurityEventFn     func(ctx context.Context, tenantID, eventID uuid.UUID) (*model.SecurityEvent, error)
	getEventStatsFn        func(ctx context.Context, tenantID uuid.UUID, from, to *time.Time) (*model.EventStats, error)
	insertSecurityEventsFn func(ctx context.Context, events []model.SecurityEvent) error
}

func (m *mockEventRepository) QuerySecurityEvents(ctx context.Context, tenantID uuid.UUID, params *model.EventQueryParams) ([]model.SecurityEvent, int, error) {
	if m.querySecurityEventsFn != nil {
		return m.querySecurityEventsFn(ctx, tenantID, params)
	}
	return []model.SecurityEvent{}, 0, nil
}

func (m *mockEventRepository) GetSecurityEvent(ctx context.Context, tenantID, eventID uuid.UUID) (*model.SecurityEvent, error) {
	if m.getSecurityEventFn != nil {
		return m.getSecurityEventFn(ctx, tenantID, eventID)
	}
	return nil, nil
}

func (m *mockEventRepository) GetSecurityEventStats(ctx context.Context, tenantID uuid.UUID, from, to *time.Time) (*model.EventStats, error) {
	if m.getEventStatsFn != nil {
		return m.getEventStatsFn(ctx, tenantID, from, to)
	}
	return &model.EventStats{}, nil
}

func (m *mockEventRepository) InsertSecurityEvents(ctx context.Context, events []model.SecurityEvent) error {
	if m.insertSecurityEventsFn != nil {
		return m.insertSecurityEventsFn(ctx, events)
	}
	return nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

var (
	eventTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	eventUserID   = uuid.MustParse("00000000-0000-0000-0000-000000000002")
)

func eventAuthRequest(method, path string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	ctx := r.Context()
	ctx = auth.WithTenantID(ctx, eventTenantID.String())
	ctx = auth.WithUser(ctx, &auth.ContextUser{
		ID:       eventUserID.String(),
		TenantID: eventTenantID.String(),
		Email:    "analyst@example.com",
		Roles:    []string{"security_analyst"},
	})
	return r.WithContext(ctx)
}

func eventAuthRequestWithID(method, path string, id uuid.UUID) *http.Request {
	r := eventAuthRequest(method, path)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	return r.WithContext(ctx)
}

func decodeEventBody(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body
}

func sampleEvent() model.SecurityEvent {
	now := time.Now().UTC().Truncate(time.Second)
	srcIP := "192.168.1.10"
	destIP := "10.0.0.5"
	destPort := 443
	proto := "TCP"
	user := "jsmith"
	proc := "curl"
	ruleID := uuid.MustParse("00000000-0000-0000-0000-aaaaaaaaaaaa")
	return model.SecurityEvent{
		ID:           uuid.MustParse("00000000-0000-0000-0000-000000000010"),
		TenantID:     eventTenantID,
		Timestamp:    now,
		Source:       "firewall",
		Type:         "connection_attempt",
		Severity:     model.SeverityHigh,
		SourceIP:     &srcIP,
		DestIP:       &destIP,
		DestPort:     &destPort,
		Protocol:     &proto,
		Username:     &user,
		Process:      &proc,
		RawEvent:     json.RawMessage(`{"action":"allow"}`),
		MatchedRules: []uuid.UUID{ruleID},
		ProcessedAt:  now,
	}
}

func newEventHandler(repo *mockEventRepository) *EventHandler {
	return NewEventHandler(repo, zerolog.Nop())
}

// ---------------------------------------------------------------------------
// TestEventHandler_NoAuth — all endpoints reject unauthenticated requests
// ---------------------------------------------------------------------------

func TestEventHandler_NoAuth(t *testing.T) {
	h := newEventHandler(&mockEventRepository{})

	tests := []struct {
		name    string
		handler http.HandlerFunc
		method  string
		path    string
	}{
		{"ListEvents", h.ListEvents, "GET", "/events"},
		{"GetEvent", h.GetEvent, "GET", "/events/00000000-0000-0000-0000-000000000010"},
		{"GetEventStats", h.GetEventStats, "GET", "/events/stats"},
		{"ExportEvents", h.ExportEvents, "GET", "/events/export?from=2024-01-01"},
		{"IngestEvents", h.IngestEvents, "POST", "/events"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			tt.handler(rr, req)
			if rr.Code != http.StatusForbidden && rr.Code != http.StatusUnauthorized {
				t.Errorf("expected 401 or 403, got %d", rr.Code)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ListEvents
// ---------------------------------------------------------------------------

func TestListEvents_Success(t *testing.T) {
	evt := sampleEvent()
	repo := &mockEventRepository{
		querySecurityEventsFn: func(_ context.Context, tenantID uuid.UUID, params *model.EventQueryParams) ([]model.SecurityEvent, int, error) {
			if tenantID != eventTenantID {
				t.Errorf("unexpected tenant %v", tenantID)
			}
			return []model.SecurityEvent{evt}, 1, nil
		},
	}
	h := newEventHandler(repo)

	rr := httptest.NewRecorder()
	h.ListEvents(rr, eventAuthRequest("GET", "/events"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	body := decodeEventBody(t, rr)
	data := body["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("expected 1 event, got %d", len(data))
	}
	first := data[0].(map[string]any)
	if first["source"] != "firewall" {
		t.Errorf("expected source=firewall, got %v", first["source"])
	}
	meta := body["meta"].(map[string]any)
	if int(meta["total"].(float64)) != 1 {
		t.Errorf("expected total=1, got %v", meta["total"])
	}
	if int(meta["total_pages"].(float64)) != 1 {
		t.Errorf("expected total_pages=1, got %v", meta["total_pages"])
	}
}

func TestListEvents_WithFilters(t *testing.T) {
	repo := &mockEventRepository{
		querySecurityEventsFn: func(_ context.Context, _ uuid.UUID, params *model.EventQueryParams) ([]model.SecurityEvent, int, error) {
			if params.Source != "endpoint" {
				t.Errorf("expected source=endpoint, got %q", params.Source)
			}
			if params.SourceIP != "10.0.0.1" {
				t.Errorf("expected source_ip=10.0.0.1, got %q", params.SourceIP)
			}
			if len(params.Severities) != 2 || params.Severities[0] != "critical" || params.Severities[1] != "high" {
				t.Errorf("expected severities=[critical,high], got %v", params.Severities)
			}
			if params.Page != 2 {
				t.Errorf("expected page=2, got %d", params.Page)
			}
			if params.PerPage != 10 {
				t.Errorf("expected per_page=10, got %d", params.PerPage)
			}
			if params.Sort != "source" {
				t.Errorf("expected sort=source, got %q", params.Sort)
			}
			if params.Order != "asc" {
				t.Errorf("expected order=asc, got %q", params.Order)
			}
			return []model.SecurityEvent{}, 0, nil
		},
	}
	h := newEventHandler(repo)

	rr := httptest.NewRecorder()
	h.ListEvents(rr, eventAuthRequest("GET",
		"/events?source=endpoint&source_ip=10.0.0.1&severity=critical&severity=high&page=2&per_page=10&sort=source&order=asc"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestListEvents_WithTimeRange(t *testing.T) {
	repo := &mockEventRepository{
		querySecurityEventsFn: func(_ context.Context, _ uuid.UUID, params *model.EventQueryParams) ([]model.SecurityEvent, int, error) {
			if params.From == nil {
				t.Fatal("expected from to be set")
			}
			if params.To == nil {
				t.Fatal("expected to to be set")
			}
			if params.From.Year() != 2024 || params.From.Month() != 1 || params.From.Day() != 1 {
				t.Errorf("expected from=2024-01-01, got %v", params.From)
			}
			return []model.SecurityEvent{}, 0, nil
		},
	}
	h := newEventHandler(repo)

	rr := httptest.NewRecorder()
	h.ListEvents(rr, eventAuthRequest("GET", "/events?from=2024-01-01&to=2024-01-31"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestListEvents_InvalidTimeRange(t *testing.T) {
	h := newEventHandler(&mockEventRepository{})

	rr := httptest.NewRecorder()
	h.ListEvents(rr, eventAuthRequest("GET", "/events?from=not-a-date"))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestListEvents_WithTextFilters(t *testing.T) {
	repo := &mockEventRepository{
		querySecurityEventsFn: func(_ context.Context, _ uuid.UUID, params *model.EventQueryParams) ([]model.SecurityEvent, int, error) {
			if params.Username != "admin" {
				t.Errorf("expected username=admin, got %q", params.Username)
			}
			if params.Process != "powershell" {
				t.Errorf("expected process=powershell, got %q", params.Process)
			}
			if params.CmdContains != "invoke" {
				t.Errorf("expected cmd_contains=invoke, got %q", params.CmdContains)
			}
			if params.FileHash != "abc123" {
				t.Errorf("expected file_hash=abc123, got %q", params.FileHash)
			}
			if params.Search != "suspicious" {
				t.Errorf("expected search=suspicious, got %q", params.Search)
			}
			return []model.SecurityEvent{}, 0, nil
		},
	}
	h := newEventHandler(repo)

	rr := httptest.NewRecorder()
	h.ListEvents(rr, eventAuthRequest("GET",
		"/events?username=admin&process=powershell&cmd_contains=invoke&file_hash=abc123&search=suspicious"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestListEvents_WithProtocolFilter(t *testing.T) {
	repo := &mockEventRepository{
		querySecurityEventsFn: func(_ context.Context, _ uuid.UUID, params *model.EventQueryParams) ([]model.SecurityEvent, int, error) {
			if len(params.Protocols) != 2 || params.Protocols[0] != "TCP" || params.Protocols[1] != "UDP" {
				t.Errorf("expected protocols=[TCP,UDP], got %v", params.Protocols)
			}
			return []model.SecurityEvent{}, 0, nil
		},
	}
	h := newEventHandler(repo)

	rr := httptest.NewRecorder()
	h.ListEvents(rr, eventAuthRequest("GET", "/events?protocol=TCP&protocol=UDP"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestListEvents_ServiceError(t *testing.T) {
	repo := &mockEventRepository{
		querySecurityEventsFn: func(_ context.Context, _ uuid.UUID, _ *model.EventQueryParams) ([]model.SecurityEvent, int, error) {
			return nil, 0, fmt.Errorf("db error")
		},
	}
	h := newEventHandler(repo)

	rr := httptest.NewRecorder()
	h.ListEvents(rr, eventAuthRequest("GET", "/events"))

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestListEvents_EmptyResult(t *testing.T) {
	repo := &mockEventRepository{
		querySecurityEventsFn: func(_ context.Context, _ uuid.UUID, _ *model.EventQueryParams) ([]model.SecurityEvent, int, error) {
			return []model.SecurityEvent{}, 0, nil
		},
	}
	h := newEventHandler(repo)

	rr := httptest.NewRecorder()
	h.ListEvents(rr, eventAuthRequest("GET", "/events"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := decodeEventBody(t, rr)
	data := body["data"].([]any)
	if len(data) != 0 {
		t.Errorf("expected empty data array, got %d items", len(data))
	}
}

func TestListEvents_Pagination(t *testing.T) {
	repo := &mockEventRepository{
		querySecurityEventsFn: func(_ context.Context, _ uuid.UUID, _ *model.EventQueryParams) ([]model.SecurityEvent, int, error) {
			return []model.SecurityEvent{sampleEvent()}, 150, nil
		},
	}
	h := newEventHandler(repo)

	rr := httptest.NewRecorder()
	h.ListEvents(rr, eventAuthRequest("GET", "/events?per_page=50"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := decodeEventBody(t, rr)
	meta := body["meta"].(map[string]any)
	totalPages := int(meta["total_pages"].(float64))
	if totalPages != 3 {
		t.Errorf("expected 3 total_pages for 150 total / 50 per_page, got %d", totalPages)
	}
}

func TestListEvents_DefaultPagination(t *testing.T) {
	repo := &mockEventRepository{
		querySecurityEventsFn: func(_ context.Context, _ uuid.UUID, params *model.EventQueryParams) ([]model.SecurityEvent, int, error) {
			if params.Page != 1 {
				t.Errorf("expected default page=1, got %d", params.Page)
			}
			if params.PerPage != 50 {
				t.Errorf("expected default per_page=50, got %d", params.PerPage)
			}
			if params.Sort != "timestamp" {
				t.Errorf("expected default sort=timestamp, got %q", params.Sort)
			}
			if params.Order != "desc" {
				t.Errorf("expected default order=desc, got %q", params.Order)
			}
			return []model.SecurityEvent{}, 0, nil
		},
	}
	h := newEventHandler(repo)

	rr := httptest.NewRecorder()
	h.ListEvents(rr, eventAuthRequest("GET", "/events"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestListEvents_MatchedRuleFilter(t *testing.T) {
	repo := &mockEventRepository{
		querySecurityEventsFn: func(_ context.Context, _ uuid.UUID, params *model.EventQueryParams) ([]model.SecurityEvent, int, error) {
			if params.MatchedRule != "00000000-0000-0000-0000-aaaaaaaaaaaa" {
				t.Errorf("expected matched_rule UUID, got %q", params.MatchedRule)
			}
			return []model.SecurityEvent{}, 0, nil
		},
	}
	h := newEventHandler(repo)

	rr := httptest.NewRecorder()
	h.ListEvents(rr, eventAuthRequest("GET", "/events?matched_rule=00000000-0000-0000-0000-aaaaaaaaaaaa"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// GetEvent
// ---------------------------------------------------------------------------

func TestGetEvent_Success(t *testing.T) {
	evt := sampleEvent()
	repo := &mockEventRepository{
		getSecurityEventFn: func(_ context.Context, tenantID, eventID uuid.UUID) (*model.SecurityEvent, error) {
			if tenantID != eventTenantID {
				t.Errorf("unexpected tenant %v", tenantID)
			}
			if eventID != evt.ID {
				t.Errorf("unexpected event ID %v", eventID)
			}
			return &evt, nil
		},
	}
	h := newEventHandler(repo)

	rr := httptest.NewRecorder()
	h.GetEvent(rr, eventAuthRequestWithID("GET", "/events/"+evt.ID.String(), evt.ID))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := decodeEventBody(t, rr)
	data := body["data"].(map[string]any)
	if data["source"] != "firewall" {
		t.Errorf("expected source=firewall, got %v", data["source"])
	}
	if data["severity"] != "high" {
		t.Errorf("expected severity=high, got %v", data["severity"])
	}
}

func TestGetEvent_NotFound(t *testing.T) {
	repo := &mockEventRepository{
		getSecurityEventFn: func(_ context.Context, _, _ uuid.UUID) (*model.SecurityEvent, error) {
			return nil, repository.ErrNotFound
		},
	}
	h := newEventHandler(repo)

	id := uuid.New()
	rr := httptest.NewRecorder()
	h.GetEvent(rr, eventAuthRequestWithID("GET", "/events/"+id.String(), id))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestGetEvent_InvalidUUID(t *testing.T) {
	h := newEventHandler(&mockEventRepository{})

	rr := httptest.NewRecorder()
	// Set chi route context with invalid UUID
	r := eventAuthRequest("GET", "/events/not-a-uuid")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "not-a-uuid")
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	h.GetEvent(rr, r.WithContext(ctx))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestGetEvent_VerifyResponseEnvelope(t *testing.T) {
	evt := sampleEvent()
	repo := &mockEventRepository{
		getSecurityEventFn: func(_ context.Context, _, _ uuid.UUID) (*model.SecurityEvent, error) {
			return &evt, nil
		},
	}
	h := newEventHandler(repo)

	rr := httptest.NewRecorder()
	h.GetEvent(rr, eventAuthRequestWithID("GET", "/events/"+evt.ID.String(), evt.ID))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := decodeEventBody(t, rr)
	// Verify envelope has "data" key wrapping the event
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data to be a map, got %T", body["data"])
	}
	// Verify key fields are present
	if data["id"] == nil {
		t.Error("expected id in response")
	}
	if data["timestamp"] == nil {
		t.Error("expected timestamp in response")
	}
	if data["raw_event"] == nil {
		t.Error("expected raw_event in response")
	}
	if data["matched_rules"] == nil {
		t.Error("expected matched_rules in response")
	}
	if data["processed_at"] == nil {
		t.Error("expected processed_at in response")
	}
	// Verify optional fields are present when set
	if data["source_ip"] == nil {
		t.Error("expected source_ip in response")
	}
	if data["dest_port"] == nil {
		t.Error("expected dest_port in response")
	}
}

// ---------------------------------------------------------------------------
// GetEventStats
// ---------------------------------------------------------------------------

func TestGetEventStats_Success(t *testing.T) {
	repo := &mockEventRepository{
		getEventStatsFn: func(_ context.Context, tenantID uuid.UUID, from, to *time.Time) (*model.EventStats, error) {
			if tenantID != eventTenantID {
				t.Errorf("unexpected tenant %v", tenantID)
			}
			return &model.EventStats{
				Total: 500,
				BySource: []model.NamedCount{
					{Name: "firewall", Count: 300},
					{Name: "endpoint", Count: 200},
				},
				ByType: []model.NamedCount{
					{Name: "connection_attempt", Count: 400},
					{Name: "process_execution", Count: 100},
				},
				BySeverity: []model.NamedCount{
					{Name: "critical", Count: 50},
					{Name: "high", Count: 150},
					{Name: "medium", Count: 200},
					{Name: "low", Count: 80},
					{Name: "info", Count: 20},
				},
			}, nil
		},
	}
	h := newEventHandler(repo)

	rr := httptest.NewRecorder()
	h.GetEventStats(rr, eventAuthRequest("GET", "/events/stats"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := decodeEventBody(t, rr)
	data := body["data"].(map[string]any)
	if int(data["total"].(float64)) != 500 {
		t.Errorf("expected total=500, got %v", data["total"])
	}
	bySrc := data["by_source"].([]any)
	if len(bySrc) != 2 {
		t.Errorf("expected 2 by_source entries, got %d", len(bySrc))
	}
	bySev := data["by_severity"].([]any)
	if len(bySev) != 5 {
		t.Errorf("expected 5 by_severity entries, got %d", len(bySev))
	}
}

func TestGetEventStats_WithTimeRange(t *testing.T) {
	repo := &mockEventRepository{
		getEventStatsFn: func(_ context.Context, _ uuid.UUID, from, to *time.Time) (*model.EventStats, error) {
			if from == nil {
				t.Fatal("expected from to be set")
			}
			if to == nil {
				t.Fatal("expected to to be set")
			}
			if from.Year() != 2024 || from.Month() != 3 || from.Day() != 1 {
				t.Errorf("expected from=2024-03-01, got %v", from)
			}
			return &model.EventStats{Total: 42}, nil
		},
	}
	h := newEventHandler(repo)

	rr := httptest.NewRecorder()
	h.GetEventStats(rr, eventAuthRequest("GET", "/events/stats?from=2024-03-01&to=2024-03-31"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestGetEventStats_WithRFC3339Time(t *testing.T) {
	repo := &mockEventRepository{
		getEventStatsFn: func(_ context.Context, _ uuid.UUID, from, to *time.Time) (*model.EventStats, error) {
			if from == nil {
				t.Fatal("expected from to be set")
			}
			return &model.EventStats{Total: 10}, nil
		},
	}
	h := newEventHandler(repo)

	rr := httptest.NewRecorder()
	h.GetEventStats(rr, eventAuthRequest("GET", "/events/stats?from=2024-03-01T08:00:00Z"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestGetEventStats_InvalidFrom(t *testing.T) {
	h := newEventHandler(&mockEventRepository{})

	rr := httptest.NewRecorder()
	h.GetEventStats(rr, eventAuthRequest("GET", "/events/stats?from=bad-date"))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestGetEventStats_InvalidTo(t *testing.T) {
	h := newEventHandler(&mockEventRepository{})

	rr := httptest.NewRecorder()
	h.GetEventStats(rr, eventAuthRequest("GET", "/events/stats?to=bad-date"))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestGetEventStats_ServiceError(t *testing.T) {
	repo := &mockEventRepository{
		getEventStatsFn: func(_ context.Context, _ uuid.UUID, _, _ *time.Time) (*model.EventStats, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	h := newEventHandler(repo)

	rr := httptest.NewRecorder()
	h.GetEventStats(rr, eventAuthRequest("GET", "/events/stats"))

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestGetEventStats_NoTimeRange(t *testing.T) {
	repo := &mockEventRepository{
		getEventStatsFn: func(_ context.Context, _ uuid.UUID, from, to *time.Time) (*model.EventStats, error) {
			if from != nil {
				t.Errorf("expected from=nil, got %v", from)
			}
			if to != nil {
				t.Errorf("expected to=nil, got %v", to)
			}
			return &model.EventStats{Total: 1000}, nil
		},
	}
	h := newEventHandler(repo)

	rr := httptest.NewRecorder()
	h.GetEventStats(rr, eventAuthRequest("GET", "/events/stats"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Response contract verification
// ---------------------------------------------------------------------------

func TestListEvents_ResponseContract(t *testing.T) {
	evt := sampleEvent()
	repo := &mockEventRepository{
		querySecurityEventsFn: func(_ context.Context, _ uuid.UUID, _ *model.EventQueryParams) ([]model.SecurityEvent, int, error) {
			return []model.SecurityEvent{evt}, 1, nil
		},
	}
	h := newEventHandler(repo)

	rr := httptest.NewRecorder()
	h.ListEvents(rr, eventAuthRequest("GET", "/events"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	// Verify Content-Type
	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type=application/json, got %q", ct)
	}

	body := decodeEventBody(t, rr)

	// Verify top-level keys
	if _, ok := body["data"]; !ok {
		t.Fatal("response missing 'data' key")
	}
	if _, ok := body["meta"]; !ok {
		t.Fatal("response missing 'meta' key")
	}

	// Verify meta shape
	meta := body["meta"].(map[string]any)
	for _, key := range []string{"page", "per_page", "total", "total_pages"} {
		if _, ok := meta[key]; !ok {
			t.Errorf("meta missing key %q", key)
		}
	}

	// Verify event fields
	data := body["data"].([]any)
	event := data[0].(map[string]any)
	expectedFields := []string{
		"id", "tenant_id", "timestamp", "source", "type", "severity",
		"source_ip", "dest_ip", "dest_port", "protocol", "username", "process",
		"raw_event", "matched_rules", "processed_at",
	}
	for _, field := range expectedFields {
		if _, ok := event[field]; !ok {
			t.Errorf("event missing field %q", field)
		}
	}
}

// ---------------------------------------------------------------------------
// ExportEvents
// ---------------------------------------------------------------------------

func TestExportEvents_CSV(t *testing.T) {
	evt := sampleEvent()
	repo := &mockEventRepository{
		querySecurityEventsFn: func(_ context.Context, _ uuid.UUID, params *model.EventQueryParams) ([]model.SecurityEvent, int, error) {
			if params.PerPage != 10000 {
				t.Errorf("expected export per_page=10000, got %d", params.PerPage)
			}
			if params.Page != 1 {
				t.Errorf("expected export page=1, got %d", params.Page)
			}
			return []model.SecurityEvent{evt}, 1, nil
		},
	}
	h := newEventHandler(repo)

	rr := httptest.NewRecorder()
	h.ExportEvents(rr, eventAuthRequest("GET", "/events/export?from=2024-01-01"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if ct != "text/csv" {
		t.Errorf("expected Content-Type=text/csv, got %q", ct)
	}
	disp := rr.Header().Get("Content-Disposition")
	if !strings.Contains(disp, "events_export_") {
		t.Errorf("expected Content-Disposition with events_export_, got %q", disp)
	}
	if rr.Header().Get("X-Export-Total") != "1" {
		t.Errorf("expected X-Export-Total=1, got %q", rr.Header().Get("X-Export-Total"))
	}
	// Verify CSV has header + 1 data row
	lines := strings.Split(strings.TrimSpace(rr.Body.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 CSV lines (header + 1 row), got %d", len(lines))
	}
}

func TestExportEvents_NDJSON(t *testing.T) {
	evt := sampleEvent()
	repo := &mockEventRepository{
		querySecurityEventsFn: func(_ context.Context, _ uuid.UUID, _ *model.EventQueryParams) ([]model.SecurityEvent, int, error) {
			return []model.SecurityEvent{evt}, 1, nil
		},
	}
	h := newEventHandler(repo)

	rr := httptest.NewRecorder()
	h.ExportEvents(rr, eventAuthRequest("GET", "/events/export?from=2024-01-01&format=ndjson"))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if ct != "application/x-ndjson" {
		t.Errorf("expected Content-Type=application/x-ndjson, got %q", ct)
	}
	// Verify each line is valid JSON
	lines := strings.Split(strings.TrimSpace(rr.Body.String()), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 NDJSON line, got %d", len(lines))
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &parsed); err != nil {
		t.Errorf("invalid JSON in NDJSON line: %v", err)
	}
}

func TestExportEvents_RequiresFrom(t *testing.T) {
	h := newEventHandler(&mockEventRepository{})

	rr := httptest.NewRecorder()
	h.ExportEvents(rr, eventAuthRequest("GET", "/events/export"))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when from is missing, got %d", rr.Code)
	}
}

func TestExportEvents_ServiceError(t *testing.T) {
	repo := &mockEventRepository{
		querySecurityEventsFn: func(_ context.Context, _ uuid.UUID, _ *model.EventQueryParams) ([]model.SecurityEvent, int, error) {
			return nil, 0, fmt.Errorf("db error")
		},
	}
	h := newEventHandler(repo)

	rr := httptest.NewRecorder()
	h.ExportEvents(rr, eventAuthRequest("GET", "/events/export?from=2024-01-01"))

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// IngestEvents
// ---------------------------------------------------------------------------

func TestIngestEvents_Success(t *testing.T) {
	var receivedEvents []model.SecurityEvent
	repo := &mockEventRepository{
		insertSecurityEventsFn: func(_ context.Context, events []model.SecurityEvent) error {
			receivedEvents = events
			return nil
		},
	}
	h := newEventHandler(repo)

	body := `[{"source":"firewall","type":"connection_attempt"},{"source":"endpoint","type":"process_execution"}]`
	rr := httptest.NewRecorder()
	req := eventAuthRequest("POST", "/events")
	req.Body = io.NopCloser(strings.NewReader(body))
	h.IngestEvents(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d; body: %s", rr.Code, rr.Body.String())
	}

	resp := decodeEventBody(t, rr)
	if int(resp["inserted"].(float64)) != 2 {
		t.Errorf("expected inserted=2, got %v", resp["inserted"])
	}

	// Verify defaults were applied
	if len(receivedEvents) != 2 {
		t.Fatalf("expected 2 events passed to repo, got %d", len(receivedEvents))
	}
	for i, evt := range receivedEvents {
		if evt.TenantID != eventTenantID {
			t.Errorf("event[%d] tenant mismatch: %v", i, evt.TenantID)
		}
		if evt.ID == uuid.Nil {
			t.Errorf("event[%d] should have generated ID", i)
		}
		if evt.Severity == "" {
			t.Errorf("event[%d] should have default severity", i)
		}
	}
}

func TestIngestEvents_EmptyArray(t *testing.T) {
	h := newEventHandler(&mockEventRepository{})

	req := eventAuthRequest("POST", "/events")
	req.Body = io.NopCloser(strings.NewReader("[]"))
	rr := httptest.NewRecorder()
	h.IngestEvents(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty array, got %d", rr.Code)
	}
}

func TestIngestEvents_InvalidJSON(t *testing.T) {
	h := newEventHandler(&mockEventRepository{})

	req := eventAuthRequest("POST", "/events")
	req.Body = io.NopCloser(strings.NewReader("{not json"))
	rr := httptest.NewRecorder()
	h.IngestEvents(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}

func TestIngestEvents_BatchTooLarge(t *testing.T) {
	h := newEventHandler(&mockEventRepository{})

	// Build a JSON array with 1001 minimal events
	var sb strings.Builder
	sb.WriteString("[")
	for i := 0; i < 1001; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`{"source":"test","type":"test"}`)
	}
	sb.WriteString("]")

	req := eventAuthRequest("POST", "/events")
	req.Body = io.NopCloser(strings.NewReader(sb.String()))
	rr := httptest.NewRecorder()
	h.IngestEvents(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for batch > 1000, got %d", rr.Code)
	}
}

func TestIngestEvents_ServiceError(t *testing.T) {
	repo := &mockEventRepository{
		insertSecurityEventsFn: func(_ context.Context, _ []model.SecurityEvent) error {
			return fmt.Errorf("db error")
		},
	}
	h := newEventHandler(repo)

	req := eventAuthRequest("POST", "/events")
	req.Body = io.NopCloser(strings.NewReader(`[{"source":"test","type":"test"}]`))
	rr := httptest.NewRecorder()
	h.IngestEvents(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}
