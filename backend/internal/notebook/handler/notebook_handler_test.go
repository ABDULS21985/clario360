package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	nbmodel "github.com/clario360/platform/internal/notebook/model"
	nbservice "github.com/clario360/platform/internal/notebook/service"
	"github.com/clario360/platform/internal/security"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newHandlerForTest(hubServer *httptest.Server) *NotebookHandler {
	reg := prometheus.NewRegistry()
	metrics := security.NewNotebookMetrics(reg)
	svc := nbservice.NewNotebookService(
		hubServer.URL+"/hub/api",
		hubServer.URL,
		"test-token",
		hubServer.Client(),
		nil,
		metrics,
		zerolog.Nop(),
	)
	return NewNotebookHandler(svc, zerolog.Nop())
}

func notebookAuthCtx() context.Context {
	ctx := context.Background()
	return auth.WithUser(ctx, &auth.ContextUser{
		ID:       "user-1",
		TenantID: "tenant-1",
		Email:    "analyst@example.com",
		Roles:    []string{"security-analyst"},
	})
}

func notebookAdminCtx() context.Context {
	ctx := context.Background()
	return auth.WithUser(ctx, &auth.ContextUser{
		ID:       "user-2",
		TenantID: "tenant-1",
		Email:    "admin@example.com",
		Roles:    []string{"tenant-admin"},
	})
}

func withChiID(r *http.Request, id string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func decodeJSON(t *testing.T, body *bytes.Buffer, dst any) {
	t.Helper()
	if err := json.NewDecoder(body).Decode(dst); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
}

// ---------------------------------------------------------------------------
// minimal JupyterHub mock (independent of service package internals)
// ---------------------------------------------------------------------------

type handlerHubState struct {
	mu    sync.Mutex
	users map[string]map[string]any // email → JupyterHub user object
	files map[string]map[string]any // key → content object
}

func newHandlerHubState() *handlerHubState {
	return &handlerHubState{
		users: map[string]map[string]any{},
		files: map[string]map[string]any{
			"analyst@example.com/examples/01_threat_detection_quickstart.ipynb": {
				"name":    "01_threat_detection_quickstart.ipynb",
				"path":    "examples/01_threat_detection_quickstart.ipynb",
				"type":    "notebook",
				"format":  "json",
				"content": map[string]any{"cells": []any{}},
			},
		},
	}
}

func (s *handlerHubState) buildServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.HasPrefix(r.URL.Path, "/hub/api/users/"):
			s.handleHubAPI(w, r)
		case strings.HasPrefix(r.URL.Path, "/user/"):
			s.handleContents(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
}

func (s *handlerHubState) handleHubAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/hub/api/users/")
	parts := strings.SplitN(path, "/", 2)
	email := parts[0]

	user, ok := s.users[email]

	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(user)

	case len(parts) == 2 && parts[1] == "server" && r.Method == http.MethodPost:
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		if !ok {
			user = map[string]any{"name": email, "servers": map[string]any{}}
		}
		servers, _ := user["servers"].(map[string]any)
		if servers == nil {
			servers = map[string]any{}
		}
		servers["default"] = map[string]any{
			"name":         "default",
			"ready":        false,
			"pending":      "spawn",
			"url":          "/user/" + email + "/lab",
			"user_options": req,
			"state":        map[string]any{},
			"started":      time.Now().UTC().Format(time.RFC3339),
			"last_activity": time.Now().UTC().Format(time.RFC3339),
		}
		user["servers"] = servers
		s.users[email] = user
		w.WriteHeader(http.StatusCreated)

	case len(parts) == 2 && parts[1] == "server" && r.Method == http.MethodDelete:
		if !ok {
			http.NotFound(w, r)
			return
		}
		servers, _ := user["servers"].(map[string]any)
		if _, exists := servers["default"]; !exists {
			http.NotFound(w, r)
			return
		}
		delete(servers, "default")
		user["servers"] = servers
		s.users[email] = user
		w.WriteHeader(http.StatusNoContent)

	default:
		http.NotFound(w, r)
	}
}

func (s *handlerHubState) handleContents(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/user/")
	parts := strings.SplitN(path, "/api/contents/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	email := parts[0]
	contentPath := parts[1]
	key := email + "/" + contentPath

	switch r.Method {
	case http.MethodGet:
		model, ok := s.files[key]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(model)
	case http.MethodPut:
		var model map[string]any
		if err := json.NewDecoder(r.Body).Decode(&model); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.files[key] = model
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(model)
	default:
		http.NotFound(w, r)
	}
}

// setRunningServer configures the mock to report a running server for the given user.
func (s *handlerHubState) setRunningServer(email string) {
	s.users[email] = map[string]any{
		"name": email,
		"servers": map[string]any{
			"default": map[string]any{
				"name":          "default",
				"ready":         true,
				"pending":       "",
				"url":           "/user/" + email + "/lab",
				"user_options":  map[string]any{"profile": "soc-analyst"},
				"state":         map[string]any{"cpu_percent": 25.0, "memory_mb": 1024, "memory_limit_mb": 4096},
				"started":       time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339),
				"last_activity": time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339),
			},
		},
	}
}

// ---------------------------------------------------------------------------
// GET /profiles
// ---------------------------------------------------------------------------

func TestNotebookHandler_ListProfiles(t *testing.T) {
	hub := newHandlerHubState()
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	r := httptest.NewRequest(http.MethodGet, "/notebooks/profiles", nil)
	r = r.WithContext(notebookAuthCtx())
	w := httptest.NewRecorder()

	h.ListProfiles(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var profiles []nbmodel.NotebookProfile
	decodeJSON(t, w.Body, &profiles)

	// analyst does not have tenant-admin role → "admin" profile must be filtered out
	for _, p := range profiles {
		if p.Slug == "admin" {
			t.Fatalf("admin profile should be hidden for non-admin users")
		}
	}
	if len(profiles) == 0 {
		t.Fatal("expected at least one profile")
	}
}

func TestNotebookHandler_ListProfiles_AdminSeesAll(t *testing.T) {
	hub := newHandlerHubState()
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	r := httptest.NewRequest(http.MethodGet, "/notebooks/profiles", nil)
	r = r.WithContext(notebookAdminCtx())
	w := httptest.NewRecorder()

	h.ListProfiles(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var profiles []nbmodel.NotebookProfile
	decodeJSON(t, w.Body, &profiles)

	found := false
	for _, p := range profiles {
		if p.Slug == "admin" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("admin profile should be visible to tenant-admin")
	}
}

// ---------------------------------------------------------------------------
// GET /templates
// ---------------------------------------------------------------------------

func TestNotebookHandler_ListTemplates(t *testing.T) {
	hub := newHandlerHubState()
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	r := httptest.NewRequest(http.MethodGet, "/notebooks/templates", nil)
	r = r.WithContext(notebookAuthCtx())
	w := httptest.NewRecorder()

	h.ListTemplates(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var templates []nbmodel.NotebookTemplate
	decodeJSON(t, w.Body, &templates)

	if len(templates) != 10 {
		t.Fatalf("expected 10 templates, got %d", len(templates))
	}
	for _, tmpl := range templates {
		if tmpl.ID == "" || tmpl.Title == "" || tmpl.Filename == "" {
			t.Fatalf("template missing required fields: %+v", tmpl)
		}
	}
}

// ---------------------------------------------------------------------------
// GET /servers
// ---------------------------------------------------------------------------

func TestNotebookHandler_ListServers_Empty(t *testing.T) {
	hub := newHandlerHubState()
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	r := httptest.NewRequest(http.MethodGet, "/notebooks/servers", nil)
	r = r.WithContext(notebookAuthCtx())
	w := httptest.NewRecorder()

	h.ListServers(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var servers []nbmodel.NotebookServer
	decodeJSON(t, w.Body, &servers)
	if len(servers) != 0 {
		t.Fatalf("expected empty list, got %d", len(servers))
	}
}

func TestNotebookHandler_ListServers_WithRunning(t *testing.T) {
	hub := newHandlerHubState()
	hub.setRunningServer("analyst@example.com")
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	r := httptest.NewRequest(http.MethodGet, "/notebooks/servers", nil)
	r = r.WithContext(notebookAuthCtx())
	w := httptest.NewRecorder()

	h.ListServers(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var servers []nbmodel.NotebookServer
	decodeJSON(t, w.Body, &servers)
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if servers[0].Status != "running" {
		t.Fatalf("expected running status, got %s", servers[0].Status)
	}
	if servers[0].Profile != "soc-analyst" {
		t.Fatalf("expected soc-analyst profile, got %s", servers[0].Profile)
	}
	// Verify all required fields are present in the serialized response
	if servers[0].ID == "" {
		t.Fatal("server ID must not be empty")
	}
}

// ---------------------------------------------------------------------------
// POST /servers  (StartServer)
// ---------------------------------------------------------------------------

func TestNotebookHandler_StartServer_ValidProfile(t *testing.T) {
	hub := newHandlerHubState()
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	body, _ := json.Marshal(map[string]string{"profile": "soc-analyst"})
	r := httptest.NewRequest(http.MethodPost, "/notebooks/servers", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(notebookAuthCtx())
	w := httptest.NewRecorder()

	h.StartServer(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var server nbmodel.NotebookServer
	decodeJSON(t, w.Body, &server)
	if server.Status != "starting" {
		t.Fatalf("expected starting status, got %s", server.Status)
	}
	if server.Profile != "soc-analyst" {
		t.Fatalf("expected soc-analyst profile, got %s", server.Profile)
	}
	if server.URL == "" {
		t.Fatal("server URL must not be empty")
	}
}

func TestNotebookHandler_StartServer_InvalidProfile(t *testing.T) {
	hub := newHandlerHubState()
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	body, _ := json.Marshal(map[string]string{"profile": "nonexistent-profile"})
	r := httptest.NewRequest(http.MethodPost, "/notebooks/servers", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(notebookAuthCtx())
	w := httptest.NewRecorder()

	h.StartServer(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var errResp map[string]string
	decodeJSON(t, w.Body, &errResp)
	if errResp["code"] != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR code, got %s", errResp["code"])
	}
}

func TestNotebookHandler_StartServer_ForbiddenProfile(t *testing.T) {
	hub := newHandlerHubState()
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	body, _ := json.Marshal(map[string]string{"profile": "admin"})
	r := httptest.NewRequest(http.MethodPost, "/notebooks/servers", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(notebookAuthCtx()) // analyst, not admin
	w := httptest.NewRecorder()

	h.StartServer(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotebookHandler_StartServer_AlreadyRunning(t *testing.T) {
	hub := newHandlerHubState()
	hub.setRunningServer("analyst@example.com")
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	body, _ := json.Marshal(map[string]string{"profile": "soc-analyst"})
	r := httptest.NewRequest(http.MethodPost, "/notebooks/servers", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(notebookAuthCtx())
	w := httptest.NewRecorder()

	h.StartServer(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotebookHandler_StartServer_MissingProfile(t *testing.T) {
	hub := newHandlerHubState()
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	// Send empty profile — validate:"required" should reject it
	body, _ := json.Marshal(map[string]string{"profile": ""})
	r := httptest.NewRequest(http.MethodPost, "/notebooks/servers", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(notebookAuthCtx())
	w := httptest.NewRecorder()

	h.StartServer(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty profile, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// DELETE /servers/{id}  (StopServer)
// ---------------------------------------------------------------------------

func TestNotebookHandler_StopServer_Running(t *testing.T) {
	hub := newHandlerHubState()
	hub.setRunningServer("analyst@example.com")
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	r := httptest.NewRequest(http.MethodDelete, "/notebooks/servers/default", nil)
	r = r.WithContext(notebookAuthCtx())
	r = withChiID(r, "default")
	w := httptest.NewRecorder()

	h.StopServer(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	decodeJSON(t, w.Body, &resp)
	if resp["message"] == "" {
		t.Fatal("expected a non-empty message in response")
	}
}

func TestNotebookHandler_StopServer_NotFound(t *testing.T) {
	hub := newHandlerHubState()
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	r := httptest.NewRequest(http.MethodDelete, "/notebooks/servers/default", nil)
	r = r.WithContext(notebookAuthCtx())
	r = withChiID(r, "default")
	w := httptest.NewRecorder()

	h.StopServer(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// GET /servers/{id}/status
// ---------------------------------------------------------------------------

func TestNotebookHandler_GetServerStatus_Running(t *testing.T) {
	hub := newHandlerHubState()
	hub.setRunningServer("analyst@example.com")
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	r := httptest.NewRequest(http.MethodGet, "/notebooks/servers/default/status", nil)
	r = r.WithContext(notebookAuthCtx())

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "default")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	h.GetServerStatus(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var status nbmodel.NotebookServerStatus
	decodeJSON(t, w.Body, &status)
	if status.Status != "running" {
		t.Fatalf("expected running status, got %s", status.Status)
	}
	if status.Profile != "soc-analyst" {
		t.Fatalf("expected soc-analyst profile, got %s", status.Profile)
	}
	if status.UptimeSeconds <= 0 {
		t.Fatalf("expected positive uptime, got %d", status.UptimeSeconds)
	}
	if status.CPUPercent != 25.0 {
		t.Fatalf("expected CPU 25.0, got %v", status.CPUPercent)
	}
	if status.MemoryMB != 1024 {
		t.Fatalf("expected memory 1024, got %d", status.MemoryMB)
	}
}

func TestNotebookHandler_GetServerStatus_NotFound(t *testing.T) {
	hub := newHandlerHubState()
	// No running server — hub user doesn't exist
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	r := httptest.NewRequest(http.MethodGet, "/notebooks/servers/default/status", nil)
	r = r.WithContext(notebookAuthCtx())
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "default")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.GetServerStatus(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// POST /servers/{id}/copy-template
// ---------------------------------------------------------------------------

func TestNotebookHandler_CopyTemplate_Success(t *testing.T) {
	hub := newHandlerHubState()
	hub.setRunningServer("analyst@example.com")
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	body, _ := json.Marshal(map[string]string{"template_id": "01_threat_detection_quickstart"})
	r := httptest.NewRequest(http.MethodPost, "/notebooks/servers/default/copy-template", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(notebookAuthCtx())
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "default")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.CopyTemplate(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result nbmodel.CopiedTemplate
	decodeJSON(t, w.Body, &result)
	if result.TemplateID != "01_threat_detection_quickstart" {
		t.Fatalf("expected template_id 01_threat_detection_quickstart, got %s", result.TemplateID)
	}
	if result.Path == "" {
		t.Fatal("path must not be empty")
	}
	if result.OpenURL == "" {
		t.Fatal("open_url must not be empty")
	}
}

func TestNotebookHandler_CopyTemplate_UnknownTemplate(t *testing.T) {
	hub := newHandlerHubState()
	hub.setRunningServer("analyst@example.com")
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	body, _ := json.Marshal(map[string]string{"template_id": "nonexistent-template"})
	r := httptest.NewRequest(http.MethodPost, "/notebooks/servers/default/copy-template", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(notebookAuthCtx())
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "default")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.CopyTemplate(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotebookHandler_CopyTemplate_MissingTemplateID(t *testing.T) {
	hub := newHandlerHubState()
	hub.setRunningServer("analyst@example.com")
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	body, _ := json.Marshal(map[string]string{"template_id": ""})
	r := httptest.NewRequest(http.MethodPost, "/notebooks/servers/default/copy-template", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(notebookAuthCtx())
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "default")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	h.CopyTemplate(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing template_id, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// POST /activity  (RecordActivity)
// ---------------------------------------------------------------------------

func TestNotebookHandler_RecordActivity_SDKCall(t *testing.T) {
	hub := newHandlerHubState()
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	body, _ := json.Marshal(map[string]string{
		"kind":     "sdk_api",
		"endpoint": "/api/v1/cyber/alerts",
		"status":   "200",
	})
	r := httptest.NewRequest(http.MethodPost, "/notebooks/activity", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(notebookAuthCtx())
	w := httptest.NewRecorder()

	h.RecordActivity(w, r)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotebookHandler_RecordActivity_DataQuery(t *testing.T) {
	hub := newHandlerHubState()
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	body, _ := json.Marshal(map[string]string{
		"kind":   "data_query",
		"source": "clickhouse",
	})
	r := httptest.NewRequest(http.MethodPost, "/notebooks/activity", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(notebookAuthCtx())
	w := httptest.NewRecorder()

	h.RecordActivity(w, r)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotebookHandler_RecordActivity_SparkJob(t *testing.T) {
	hub := newHandlerHubState()
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	body, _ := json.Marshal(map[string]string{
		"kind":   "spark_job",
		"status": "succeeded",
	})
	r := httptest.NewRequest(http.MethodPost, "/notebooks/activity", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(notebookAuthCtx())
	w := httptest.NewRecorder()

	h.RecordActivity(w, r)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotebookHandler_RecordActivity_InvalidKind(t *testing.T) {
	hub := newHandlerHubState()
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	body, _ := json.Marshal(map[string]string{"kind": "unknown_kind"})
	r := httptest.NewRequest(http.MethodPost, "/notebooks/activity", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(notebookAuthCtx())
	w := httptest.NewRecorder()

	h.RecordActivity(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid kind, got %d", w.Code)
	}
}

func TestNotebookHandler_RecordActivity_SDKCall_MissingEndpoint(t *testing.T) {
	hub := newHandlerHubState()
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	// sdk_api kind requires endpoint + status; omitting them
	body, _ := json.Marshal(map[string]string{"kind": "sdk_api"})
	r := httptest.NewRequest(http.MethodPost, "/notebooks/activity", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(notebookAuthCtx())
	w := httptest.NewRecorder()

	h.RecordActivity(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for sdk_api without endpoint/status, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Error response contract verification
// ---------------------------------------------------------------------------

func TestNotebookHandler_ErrorResponseShape(t *testing.T) {
	hub := newHandlerHubState()
	hubSrv := hub.buildServer(t)
	defer hubSrv.Close()

	h := newHandlerForTest(hubSrv)
	body, _ := json.Marshal(map[string]string{"profile": "nonexistent"})
	r := httptest.NewRequest(http.MethodPost, "/notebooks/servers", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(notebookAuthCtx())
	w := httptest.NewRecorder()

	h.StartServer(w, r)

	// Error responses must have {"code": "...", "message": "..."} shape,
	// matching what the frontend buildApiError() expects.
	var errResp map[string]string
	decodeJSON(t, w.Body, &errResp)
	if _, ok := errResp["code"]; !ok {
		t.Fatal("error response must contain 'code' field")
	}
	if _, ok := errResp["message"]; !ok {
		t.Fatal("error response must contain 'message' field")
	}
}

// ---------------------------------------------------------------------------
// HealthCheck
// ---------------------------------------------------------------------------

func TestHealthCheck_HubAvailable(t *testing.T) {
	hub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hub root responds 200 — available
		w.WriteHeader(http.StatusOK)
	}))
	defer hub.Close()

	h := newHandlerForTest(hub)
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.HealthCheck(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp map[string]any
	decodeJSON(t, w.Body, &resp)
	if resp["status"] != "ok" {
		t.Errorf("want status=ok, got %v", resp["status"])
	}
	hub0, _ := resp["jupyterhub"].(map[string]any)
	if hub0["status"] != "available" {
		t.Errorf("want jupyterhub.status=available, got %v", hub0["status"])
	}
}

func TestHealthCheck_HubUnauthorized(t *testing.T) {
	// 401 from JupyterHub still means the hub is up — treat as available
	hub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer hub.Close()

	h := newHandlerForTest(hub)
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.HealthCheck(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp map[string]any
	decodeJSON(t, w.Body, &resp)
	if resp["status"] != "ok" {
		t.Errorf("want status=ok for 401 response, got %v", resp["status"])
	}
}

func TestHealthCheck_HubServerError(t *testing.T) {
	hub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer hub.Close()

	h := newHandlerForTest(hub)
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.HealthCheck(w, r)

	// Handler always returns HTTP 200 — client reads body status
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp map[string]any
	decodeJSON(t, w.Body, &resp)
	if resp["status"] != "degraded" {
		t.Errorf("want status=degraded for 5xx response, got %v", resp["status"])
	}
	hub0, _ := resp["jupyterhub"].(map[string]any)
	if hub0["status"] != "unavailable" {
		t.Errorf("want jupyterhub.status=unavailable, got %v", hub0["status"])
	}
	if hub0["error"] == nil || hub0["error"] == "" {
		t.Error("want non-empty jupyterhub.error")
	}
}

func TestHealthCheck_HubDown(t *testing.T) {
	// Point to a server that is immediately closed — simulates hub unreachable
	hub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	hub.Close() // close before the request

	h := newHandlerForTest(hub)
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.HealthCheck(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var resp map[string]any
	decodeJSON(t, w.Body, &resp)
	if resp["status"] != "degraded" {
		t.Errorf("want status=degraded when hub is unreachable, got %v", resp["status"])
	}
}
