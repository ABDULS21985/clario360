package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	nbmodel "github.com/clario360/platform/internal/notebook/model"
	"github.com/clario360/platform/internal/security"
)

func TestListServers_ReturnsUserServers(t *testing.T) {
	hub := newNotebookHubMock()
	hub.users["analyst@example.com"] = hubUser{
		Name: "analyst@example.com",
		Servers: map[string]hubServer{
			"default": {
				Ready:        true,
				Started:      time.Now().Add(-2 * time.Hour),
				LastActivity: time.Now().Add(-5 * time.Minute),
				URL:          "/user/analyst@example.com/lab",
				UserOptions:  map[string]any{"profile": "soc-analyst"},
				State: map[string]any{
					"cpu_percent":     45.0,
					"memory_mb":       1228,
					"memory_limit_mb": 4096,
				},
			},
		},
	}
	server := hub.server(t)
	defer server.Close()

	svc := newNotebookServiceForTest(server)
	servers, err := svc.ListServers(context.Background(), analystActor())
	if err != nil {
		t.Fatalf("ListServers failed: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if servers[0].Profile != "soc-analyst" {
		t.Fatalf("expected soc-analyst profile, got %s", servers[0].Profile)
	}
	if servers[0].Status != "running" {
		t.Fatalf("expected running status, got %s", servers[0].Status)
	}
}

func TestStartServer_ValidProfile(t *testing.T) {
	hub := newNotebookHubMock()
	server := hub.server(t)
	defer server.Close()

	svc := newNotebookServiceForTest(server)
	got, err := svc.StartServer(context.Background(), analystActor(), "soc-analyst")
	if err != nil {
		t.Fatalf("StartServer failed: %v", err)
	}
	if got.Status != "starting" {
		t.Fatalf("expected starting status, got %s", got.Status)
	}
	if got.Profile != "soc-analyst" {
		t.Fatalf("expected soc-analyst profile, got %s", got.Profile)
	}
}

func TestStartServer_AdminProfile_RequiresRole(t *testing.T) {
	hub := newNotebookHubMock()
	server := hub.server(t)
	defer server.Close()

	svc := newNotebookServiceForTest(server)
	_, err := svc.StartServer(context.Background(), analystActor(), "admin")
	if err != nbmodel.ErrProfileForbidden {
		t.Fatalf("expected ErrProfileForbidden, got %v", err)
	}
}

func TestStartServer_AdminProfile_AdminAllowed(t *testing.T) {
	hub := newNotebookHubMock()
	server := hub.server(t)
	defer server.Close()

	svc := newNotebookServiceForTest(server)
	got, err := svc.StartServer(context.Background(), adminActor(), "admin")
	if err != nil {
		t.Fatalf("StartServer failed: %v", err)
	}
	if got.Profile != "admin" {
		t.Fatalf("expected admin profile, got %s", got.Profile)
	}
}

func TestStartServer_AlreadyRunning(t *testing.T) {
	hub := newNotebookHubMock()
	hub.users["analyst@example.com"] = hubUser{
		Name: "analyst@example.com",
		Servers: map[string]hubServer{
			"default": {
				Ready:       true,
				UserOptions: map[string]any{"profile": "soc-analyst"},
			},
		},
	}
	server := hub.server(t)
	defer server.Close()

	svc := newNotebookServiceForTest(server)
	_, err := svc.StartServer(context.Background(), analystActor(), "soc-analyst")
	if err != nbmodel.ErrServerRunning {
		t.Fatalf("expected ErrServerRunning, got %v", err)
	}
}

func TestStopServer_Running(t *testing.T) {
	hub := newNotebookHubMock()
	hub.users["analyst@example.com"] = hubUser{
		Name: "analyst@example.com",
		Servers: map[string]hubServer{
			"default": {
				Ready:       true,
				Started:     time.Now().Add(-1 * time.Hour),
				UserOptions: map[string]any{"profile": "soc-analyst"},
			},
		},
	}
	server := hub.server(t)
	defer server.Close()

	svc := newNotebookServiceForTest(server)
	if err := svc.StopServer(context.Background(), analystActor(), "default", "user"); err != nil {
		t.Fatalf("StopServer failed: %v", err)
	}
	if len(hub.users["analyst@example.com"].Servers) != 0 {
		t.Fatalf("expected server to be deleted")
	}
}

func TestStopServer_AlreadyStopped(t *testing.T) {
	hub := newNotebookHubMock()
	server := hub.server(t)
	defer server.Close()

	svc := newNotebookServiceForTest(server)
	if err := svc.StopServer(context.Background(), analystActor(), "default", "user"); err != nbmodel.ErrServerNotFound {
		t.Fatalf("expected ErrServerNotFound, got %v", err)
	}
}

func TestGetStatus_Running(t *testing.T) {
	hub := newNotebookHubMock()
	hub.users["analyst@example.com"] = hubUser{
		Name: "analyst@example.com",
		Servers: map[string]hubServer{
			"default": {
				Ready:        true,
				Started:      time.Now().Add(-30 * time.Minute),
				LastActivity: time.Now().Add(-1 * time.Minute),
				UserOptions:  map[string]any{"profile": "soc-analyst"},
				State: map[string]any{
					"cpu_percent":     12.5,
					"memory_mb":       1536,
					"memory_limit_mb": 4096,
				},
			},
		},
	}
	server := hub.server(t)
	defer server.Close()

	svc := newNotebookServiceForTest(server)
	status, err := svc.GetServerStatus(context.Background(), analystActor(), "default")
	if err != nil {
		t.Fatalf("GetServerStatus failed: %v", err)
	}
	if status.CPUPercent != 12.5 {
		t.Fatalf("expected CPU percent 12.5, got %v", status.CPUPercent)
	}
	if status.MemoryMB != 1536 {
		t.Fatalf("expected memory 1536, got %d", status.MemoryMB)
	}
	if status.UptimeSeconds <= 0 {
		t.Fatalf("expected positive uptime, got %d", status.UptimeSeconds)
	}
}

func TestListTemplates(t *testing.T) {
	hub := newNotebookHubMock()
	server := hub.server(t)
	defer server.Close()

	svc := newNotebookServiceForTest(server)
	templates := svc.ListTemplates()
	if len(templates) != 10 {
		t.Fatalf("expected 10 templates, got %d", len(templates))
	}
	if templates[0].ID == "" || templates[0].Filename == "" {
		t.Fatalf("expected template metadata to be populated")
	}
}

func newNotebookServiceForTest(server *httptest.Server) *NotebookService {
	metrics := security.NewNotebookMetrics(prometheus.NewRegistry())
	return NewNotebookService(server.URL+"/hub/api", server.URL, "test-token", server.Client(), nil, metrics, zerolog.New(ioDiscard{}))
}

func analystActor() nbmodel.Actor {
	return nbmodel.Actor{
		UserID:   "user-1",
		TenantID: "tenant-1",
		Email:    "analyst@example.com",
		Roles:    []string{"security-analyst"},
	}
}

func adminActor() nbmodel.Actor {
	return nbmodel.Actor{
		UserID:   "user-2",
		TenantID: "tenant-1",
		Email:    "admin@example.com",
		Roles:    []string{"tenant-admin"},
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

type notebookHubMock struct {
	mu    sync.Mutex
	users map[string]hubUser
	files map[string]map[string]any
}

func newNotebookHubMock() *notebookHubMock {
	return &notebookHubMock{
		users: map[string]hubUser{},
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

func (m *notebookHubMock) server(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()

		switch {
		case strings.HasPrefix(r.URL.Path, "/hub/api/users/"):
			m.handleHubUsers(w, r)
		case strings.HasPrefix(r.URL.Path, "/user/"):
			m.handleUserContents(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
}

func (m *notebookHubMock) handleHubUsers(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/hub/api/users/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	email, _ := url.PathUnescape(parts[0])
	user := m.users[email]
	if user.Name == "" {
		user.Name = email
		user.Servers = map[string]hubServer{}
	}

	if len(parts) == 1 && r.Method == http.MethodGet {
		if _, ok := m.users[email]; !ok {
			http.NotFound(w, r)
			return
		}
		writeTestJSON(w, http.StatusOK, user)
		return
	}

	if len(parts) == 2 && parts[1] == "server" && r.Method == http.MethodPost {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		user.Servers["default"] = hubServer{
			Ready:       false,
			Pending:     "spawn",
			URL:         "/user/" + email + "/lab",
			UserOptions: req,
		}
		m.users[email] = user
		w.WriteHeader(http.StatusCreated)
		return
	}

	if len(parts) == 2 && parts[1] == "server" && r.Method == http.MethodDelete {
		if _, ok := user.Servers["default"]; !ok {
			http.NotFound(w, r)
			return
		}
		delete(user.Servers, "default")
		m.users[email] = user
		w.WriteHeader(http.StatusNoContent)
		return
	}

	http.NotFound(w, r)
}

func (m *notebookHubMock) handleUserContents(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/user/")
	parts := strings.SplitN(trimmed, "/", 2)
	email, _ := url.PathUnescape(parts[0])
	if len(parts) != 2 || !strings.HasPrefix(parts[1], "api/contents/") {
		http.NotFound(w, r)
		return
	}
	contentPath, _ := url.PathUnescape(strings.TrimPrefix(parts[1], "api/contents/"))
	key := email + "/" + contentPath

	switch r.Method {
	case http.MethodGet:
		model, ok := m.files[key]
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeTestJSON(w, http.StatusOK, model)
	case http.MethodPut:
		var model map[string]any
		if err := json.NewDecoder(r.Body).Decode(&model); err != nil {
			tWriteError(w, err)
			return
		}
		m.files[key] = model
		writeTestJSON(w, http.StatusCreated, model)
	default:
		http.NotFound(w, r)
	}
}

func writeTestJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func tWriteError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusBadRequest)
}
