package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	nbdto "github.com/clario360/platform/internal/notebook/dto"
	nbmodel "github.com/clario360/platform/internal/notebook/model"
	"github.com/clario360/platform/internal/security"
)

const (
	defaultServerID       = "default"
	notebookEventSource   = "iam-service"
	notebookTopic         = "platform.notebook.events"
	jupyterAuthHeaderFmt  = "token %s"
	defaultNotebookReason = "user"
)

type NotebookService struct {
	jupyterhubAPIURL   string
	jupyterhubBaseURL  string
	jupyterhubToken    string
	httpClient         *http.Client
	producer           *events.Producer
	metrics            *security.NotebookMetrics
	logger             zerolog.Logger
	profiles           map[string]nbmodel.NotebookProfile
	templates          map[string]nbmodel.NotebookTemplate
	orderedProfiles    []nbmodel.NotebookProfile
	orderedTemplates   []nbmodel.NotebookTemplate
}

func NewNotebookService(jupyterhubAPIURL, jupyterhubBaseURL, jupyterhubToken string, httpClient *http.Client, producer *events.Producer, metrics *security.NotebookMetrics, logger zerolog.Logger) *NotebookService {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	profiles := nbmodel.DefaultProfiles()
	profileMap := make(map[string]nbmodel.NotebookProfile, len(profiles))
	for _, profile := range profiles {
		profileMap[profile.Slug] = profile
	}

	templates := nbmodel.DefaultTemplates()
	templateMap := make(map[string]nbmodel.NotebookTemplate, len(templates))
	for _, template := range templates {
		templateMap[template.ID] = template
	}

	return &NotebookService{
		jupyterhubAPIURL:  strings.TrimRight(jupyterhubAPIURL, "/"),
		jupyterhubBaseURL: strings.TrimRight(jupyterhubBaseURL, "/"),
		jupyterhubToken:   strings.TrimSpace(jupyterhubToken),
		httpClient:        httpClient,
		producer:          producer,
		metrics:           metrics,
		logger:            logger,
		profiles:          profileMap,
		templates:         templateMap,
		orderedProfiles:   profiles,
		orderedTemplates:  templates,
	}
}

func (s *NotebookService) ListProfiles(actor nbmodel.Actor) []nbmodel.NotebookProfile {
	profiles := make([]nbmodel.NotebookProfile, 0, len(s.orderedProfiles))
	for _, profile := range s.orderedProfiles {
		if s.profileAllowed(actor, profile) {
			profiles = append(profiles, profile)
		}
	}
	return profiles
}

func (s *NotebookService) ListTemplates() []nbmodel.NotebookTemplate {
	out := make([]nbmodel.NotebookTemplate, len(s.orderedTemplates))
	copy(out, s.orderedTemplates)
	return out
}

func (s *NotebookService) ListServers(ctx context.Context, actor nbmodel.Actor) ([]nbmodel.NotebookServer, error) {
	user, err := s.getHubUser(ctx, actor.Email)
	if err != nil {
		if isHubNotFound(err) {
			return []nbmodel.NotebookServer{}, nil
		}
		if isHubUnavailable(err) {
			s.logger.Warn().Err(err).Str("user_email", actor.Email).Msg("jupyterhub unavailable; returning empty server list")
			return []nbmodel.NotebookServer{}, nil
		}
		return nil, err
	}

	servers := mapHubUserToServers(user, s.jupyterhubBaseURL, actor.Email)
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].ID < servers[j].ID
	})
	return servers, nil
}

func (s *NotebookService) StartServer(ctx context.Context, actor nbmodel.Actor, profileSlug string) (*nbmodel.NotebookServer, error) {
	profile, ok := s.profiles[profileSlug]
	if !ok {
		return nil, nbmodel.ErrInvalidProfile
	}
	if !s.profileAllowed(actor, profile) {
		return nil, nbmodel.ErrProfileForbidden
	}

	existing, err := s.ListServers(ctx, actor)
	if err != nil {
		return nil, err
	}
	for _, server := range existing {
		if server.Status == "running" || server.Status == "starting" {
			return nil, nbmodel.ErrServerRunning
		}
	}

	start := time.Now()
	body := map[string]any{"profile": profileSlug}
	reqURL := fmt.Sprintf("%s/users/%s/server", s.jupyterhubAPIURL, url.PathEscape(actor.Email))
	if err := s.doJSON(ctx, http.MethodPost, reqURL, body, http.StatusCreated, http.StatusAccepted); err != nil {
		return nil, err
	}

	server := &nbmodel.NotebookServer{
		ID:      defaultServerID,
		Profile: profileSlug,
		Status:  "starting",
		URL:     s.userLabURL(actor.Email, ""),
	}
	if s.metrics != nil {
		s.metrics.ServerStartTotal.WithLabelValues(profileSlug).Inc()
		s.metrics.ServerStartDuration.WithLabelValues(profileSlug).Observe(time.Since(start).Seconds())
		s.metrics.ServersActive.WithLabelValues(profileSlug).Inc()
	}

	s.publishNotebookEvent(ctx, actor, "notebook.server.started", map[string]any{
		"server_id": defaultServerID,
		"profile":   profileSlug,
		"url":       server.URL,
		"status":    server.Status,
	})

	return server, nil
}

func (s *NotebookService) StopServer(ctx context.Context, actor nbmodel.Actor, serverID, reason string) error {
	id := normalizeServerID(serverID)
	server, err := s.GetServerStatus(ctx, actor, id)
	if err != nil {
		return err
	}

	reqURL := fmt.Sprintf("%s/users/%s/server", s.jupyterhubAPIURL, url.PathEscape(actor.Email))
	if err := s.doJSON(ctx, http.MethodDelete, reqURL, nil, http.StatusAccepted, http.StatusNoContent); err != nil {
		return err
	}

	if reason == "" {
		reason = defaultNotebookReason
	}
	if s.metrics != nil {
		s.metrics.ServerStopTotal.WithLabelValues(server.Profile, reason).Inc()
		if server.UptimeSeconds > 0 {
			s.metrics.ServerUptimeSeconds.WithLabelValues(server.Profile).Observe(float64(server.UptimeSeconds))
		}
		s.metrics.ServersActive.WithLabelValues(server.Profile).Dec()
	}

	s.publishNotebookEvent(ctx, actor, "notebook.server.stopped", map[string]any{
		"server_id":      id,
		"profile":        server.Profile,
		"reason":         reason,
		"uptime_seconds": server.UptimeSeconds,
	})
	return nil
}

func (s *NotebookService) GetServerStatus(ctx context.Context, actor nbmodel.Actor, serverID string) (*nbmodel.NotebookServerStatus, error) {
	user, err := s.getHubUser(ctx, actor.Email)
	if err != nil {
		return nil, err
	}

	server, ok := user.Servers[normalizeServerID(serverID)]
	if !ok && normalizeServerID(serverID) == defaultServerID {
		if user.Server != nil {
			server = *user.Server
			ok = true
		}
	}
	if !ok {
		return nil, nbmodel.ErrServerNotFound
	}

	return mapHubServerToStatus(normalizeServerID(serverID), server), nil
}

func (s *NotebookService) CopyTemplate(ctx context.Context, actor nbmodel.Actor, serverID, templateID string) (*nbmodel.CopiedTemplate, error) {
	template, ok := s.templates[templateID]
	if !ok {
		return nil, nbmodel.ErrTemplateNotFound
	}

	if _, err := s.GetServerStatus(ctx, actor, serverID); err != nil {
		return nil, err
	}

	sourcePath := path.Join("examples", template.Filename)
	contentURL := fmt.Sprintf("%s/user/%s/api/contents/%s?content=1", s.jupyterhubBaseURL, url.PathEscape(actor.Email), url.PathEscape(sourcePath))
	var model map[string]any
	if err := s.doAndDecode(ctx, http.MethodGet, contentURL, nil, &model, http.StatusOK); err != nil {
		return nil, err
	}

	model["path"] = template.Filename
	model["name"] = template.Filename

	targetURL := fmt.Sprintf("%s/user/%s/api/contents/%s", s.jupyterhubBaseURL, url.PathEscape(actor.Email), url.PathEscape(template.Filename))
	if err := s.doJSON(ctx, http.MethodPut, targetURL, model, http.StatusCreated, http.StatusOK); err != nil {
		return nil, err
	}

	openURL := s.userLabURL(actor.Email, template.Filename)
	s.publishNotebookEvent(ctx, actor, "notebook.template.copied", map[string]any{
		"server_id":   normalizeServerID(serverID),
		"template_id": template.ID,
		"filename":    template.Filename,
		"open_url":    openURL,
	})

	return &nbmodel.CopiedTemplate{
		TemplateID: template.ID,
		Path:       template.Filename,
		OpenURL:    openURL,
	}, nil
}

// Ping performs a lightweight GET against the JupyterHub API root and returns
// nil if the hub responds with any HTTP status below 500 (including 401/403,
// which indicate the hub is running but the token is not accepted at that
// endpoint). A network error or a 5xx response is returned as an error.
func (s *NotebookService) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.jupyterhubAPIURL, nil)
	if err != nil {
		return fmt.Errorf("build jupyterhub ping request: %w", err)
	}
	if s.jupyterhubToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf(jupyterAuthHeaderFmt, s.jupyterhubToken))
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ping jupyterhub: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 500 {
		return fmt.Errorf("jupyterhub ping returned %d", resp.StatusCode)
	}
	return nil
}

func (s *NotebookService) RecordActivity(ctx context.Context, actor nbmodel.Actor, req nbdto.ActivityRequest) error {
	occurredAt := time.Now().UTC()
	if req.OccurredAt != nil && !req.OccurredAt.IsZero() {
		occurredAt = req.OccurredAt.UTC()
	}

	switch nbmodel.ActivityKind(req.Kind) {
	case nbmodel.ActivitySDKCall:
		if req.Endpoint == "" || req.Status == "" {
			return nbmodel.ErrActivityInvalid
		}
		if s.metrics != nil {
			s.metrics.SDKAPICallsTotal.WithLabelValues(req.Endpoint, req.Status).Inc()
		}
	case nbmodel.ActivityDataQuery:
		if req.Source == "" {
			return nbmodel.ErrActivityInvalid
		}
		if s.metrics != nil {
			s.metrics.DataQueriesTotal.WithLabelValues(req.Source).Inc()
		}
	case nbmodel.ActivitySparkJob:
		status := req.Status
		if status == "" {
			status = "unknown"
		}
		if s.metrics != nil {
			s.metrics.SparkJobsTotal.WithLabelValues(status).Inc()
		}
	default:
		return nbmodel.ErrActivityInvalid
	}

	s.publishNotebookEvent(ctx, actor, "notebook.activity.recorded", map[string]any{
		"kind":        req.Kind,
		"endpoint":    req.Endpoint,
		"status":      req.Status,
		"source":      req.Source,
		"description": req.Description,
		"metadata":    req.Metadata,
		"occurred_at": occurredAt,
	})
	return nil
}

func (s *NotebookService) getHubUser(ctx context.Context, userEmail string) (*hubUser, error) {
	reqURL := fmt.Sprintf("%s/users/%s", s.jupyterhubAPIURL, url.PathEscape(userEmail))
	var user hubUser
	if err := s.doAndDecode(ctx, http.MethodGet, reqURL, nil, &user, http.StatusOK); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *NotebookService) doJSON(ctx context.Context, method, reqURL string, body any, okStatuses ...int) error {
	return s.doAndDecode(ctx, method, reqURL, body, nil, okStatuses...)
}

func (s *NotebookService) doAndDecode(ctx context.Context, method, reqURL string, body any, out any, okStatuses ...int) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal jupyterhub payload: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, reader)
	if err != nil {
		return fmt.Errorf("build jupyterhub request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if s.jupyterhubToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf(jupyterAuthHeaderFmt, s.jupyterhubToken))
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call jupyterhub: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	for _, status := range okStatuses {
		if resp.StatusCode == status {
			if out != nil && len(bodyBytes) > 0 {
				if err := json.Unmarshal(bodyBytes, out); err != nil {
					return fmt.Errorf("decode jupyterhub response: %w", err)
				}
			}
			return nil
		}
	}

	if resp.StatusCode == http.StatusNotFound {
		return nbmodel.ErrServerNotFound
	}
	if resp.StatusCode == http.StatusForbidden {
		return nbmodel.ErrProfileForbidden
	}
	return fmt.Errorf("jupyterhub %s %s returned %d: %s", method, reqURL, resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
}

func (s *NotebookService) profileAllowed(actor nbmodel.Actor, profile nbmodel.NotebookProfile) bool {
	if len(profile.RequiresRole) == 0 {
		return true
	}

	roleSet := make(map[string]struct{}, len(actor.Roles))
	for _, role := range actor.Roles {
		roleSet[strings.ToLower(strings.TrimSpace(role))] = struct{}{}
	}
	for _, required := range profile.RequiresRole {
		if _, ok := roleSet[strings.ToLower(required)]; ok {
			return true
		}
	}
	return false
}

func (s *NotebookService) userLabURL(userEmail, filePath string) string {
	base := fmt.Sprintf("%s/user/%s/lab", s.jupyterhubBaseURL, url.PathEscape(userEmail))
	if filePath == "" {
		return base
	}
	return fmt.Sprintf("%s/tree/%s", base, url.PathEscape(filePath))
}

func (s *NotebookService) publishNotebookEvent(ctx context.Context, actor nbmodel.Actor, eventType string, data map[string]any) {
	if s.producer == nil {
		return
	}
	payload := map[string]any{
		"user_id":    actor.UserID,
		"user_email": actor.Email,
		"tenant_id":  actor.TenantID,
	}
	for k, v := range data {
		payload[k] = v
	}

	evt, err := events.NewEvent(eventType, notebookEventSource, actor.TenantID, payload)
	if err != nil {
		s.logger.Error().Err(err).Str("event_type", eventType).Msg("failed to create notebook event")
		return
	}
	evt.UserID = actor.UserID
	evt.Metadata = map[string]string{
		"user_email": actor.Email,
	}
	if err := s.producer.Publish(ctx, notebookTopic, evt); err != nil {
		s.logger.Error().Err(err).Str("event_type", eventType).Msg("failed to publish notebook event")
	}
}

type hubUser struct {
	Name         string               `json:"name"`
	LastActivity time.Time            `json:"last_activity"`
	Server       *hubServer           `json:"server,omitempty"`
	Servers      map[string]hubServer `json:"servers"`
}

type hubServer struct {
	Name         string         `json:"name"`
	Ready        bool           `json:"ready"`
	Pending      string         `json:"pending"`
	Started      time.Time      `json:"started"`
	LastActivity time.Time      `json:"last_activity"`
	URL          string         `json:"url"`
	UserOptions  map[string]any `json:"user_options"`
	State        map[string]any `json:"state"`
}

func mapHubUserToServers(user *hubUser, baseURL, userEmail string) []nbmodel.NotebookServer {
	if user == nil {
		return nil
	}

	servers := make([]nbmodel.NotebookServer, 0, len(user.Servers)+1)
	if len(user.Servers) == 0 && user.Server != nil {
		servers = append(servers, mapHubServer(defaultServerID, *user.Server, baseURL, userEmail))
		return servers
	}

	for id, server := range user.Servers {
		servers = append(servers, mapHubServer(id, server, baseURL, userEmail))
	}
	return servers
}

func mapHubServer(id string, server hubServer, baseURL, userEmail string) nbmodel.NotebookServer {
	status := "stopped"
	switch {
	case server.Pending == "spawn" || server.Pending == "starting":
		status = "starting"
	case server.Pending == "stop" || server.Pending == "stopping":
		status = "stopping"
	case server.Ready:
		status = "running"
	}

	profile := stringFromState(server.UserOptions, "profile")
	if profile == "" {
		profile = stringFromState(server.State, "profile")
	}

	var startedAt *time.Time
	if !server.Started.IsZero() {
		startedAt = &server.Started
	}
	var lastActivity *time.Time
	if !server.LastActivity.IsZero() {
		lastActivity = &server.LastActivity
	}

	return nbmodel.NotebookServer{
		ID:            normalizeServerID(id),
		Profile:       profile,
		Status:        status,
		URL:           notebookURL(baseURL, userEmail, server.URL),
		StartedAt:     startedAt,
		LastActivity:  lastActivity,
		CPUPercent:    floatFromState(server.State, "cpu_percent"),
		MemoryMB:      int(floatFromState(server.State, "memory_mb")),
		MemoryLimitMB: int(floatFromState(server.State, "memory_limit_mb")),
	}
}

func mapHubServerToStatus(id string, server hubServer) *nbmodel.NotebookServerStatus {
	mapped := mapHubServer(id, server, "", "")
	uptime := int64(0)
	if mapped.StartedAt != nil {
		uptime = int64(time.Since(*mapped.StartedAt).Seconds())
	}
	return &nbmodel.NotebookServerStatus{
		ID:            mapped.ID,
		Profile:       mapped.Profile,
		Status:        mapped.Status,
		CPUPercent:    mapped.CPUPercent,
		MemoryMB:      mapped.MemoryMB,
		MemoryLimitMB: mapped.MemoryLimitMB,
		UptimeSeconds: uptime,
		LastActivity:  mapped.LastActivity,
	}
}

func normalizeServerID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" || id == "/" || id == "default" {
		return defaultServerID
	}
	return id
}

func notebookURL(baseURL, userEmail, serverURL string) string {
	if baseURL == "" {
		return serverURL
	}
	if serverURL == "" {
		return fmt.Sprintf("%s/user/%s/lab", strings.TrimRight(baseURL, "/"), url.PathEscape(userEmail))
	}
	if strings.HasPrefix(serverURL, "http://") || strings.HasPrefix(serverURL, "https://") {
		return serverURL
	}
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(serverURL, "/")
}

func stringFromState(state map[string]any, key string) string {
	if state == nil {
		return ""
	}
	value, ok := state[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func floatFromState(state map[string]any, key string) float64 {
	if state == nil {
		return 0
	}
	value, ok := state[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		f, _ := typed.Float64()
		return f
	default:
		return 0
	}
}

func isHubNotFound(err error) bool {
	return errors.Is(err, nbmodel.ErrServerNotFound)
}

func isHubUnavailable(err error) bool {
	var (
		urlErr *url.Error
		opErr  *net.OpError
		dnsErr *net.DNSError
	)
	return errors.As(err, &urlErr) || errors.As(err, &opErr) || errors.As(err, &dnsErr)
}
