package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	intdto "github.com/clario360/platform/internal/integration/dto"
	intmodel "github.com/clario360/platform/internal/integration/model"
	intsvc "github.com/clario360/platform/internal/integration/service"
	jirasvc "github.com/clario360/platform/internal/integration/service/jira"
)

type jiraOAuthState struct {
	TenantID   string `json:"tenant_id"`
	UserID     string `json:"user_id"`
	Name       string `json:"name,omitempty"`
	ProjectKey string `json:"project_key,omitempty"`
}

type jiraOAuthSessionRequest struct {
	Name       string `json:"name"`
	ProjectKey string `json:"project_key"`
}

type createTicketRequest struct {
	IntegrationID string `json:"integration_id"`
	EntityType    string `json:"entity_type"`
	EntityID      string `json:"entity_id"`
}

type JiraHandler struct {
	service      *intsvc.IntegrationService
	jiraService  *jirasvc.Service
	producer     *events.Producer
	redis        *redis.Client
	oauthCfg     jirasvc.OAuthConfig
	publicAppURL string
	stateTTL     time.Duration
	logger       zerolog.Logger
}

func NewJiraHandler(
	service *intsvc.IntegrationService,
	jiraService *jirasvc.Service,
	producer *events.Producer,
	redis *redis.Client,
	oauthCfg jirasvc.OAuthConfig,
	publicAppURL string,
	stateTTL time.Duration,
	logger zerolog.Logger,
) *JiraHandler {
	if stateTTL <= 0 {
		stateTTL = 15 * time.Minute
	}
	return &JiraHandler{
		service:      service,
		jiraService:  jiraService,
		producer:     producer,
		redis:        redis,
		oauthCfg:     oauthCfg,
		publicAppURL: strings.TrimRight(publicAppURL, "/"),
		stateTTL:     stateTTL,
		logger:       logger.With().Str("component", "integration_jira_handler").Logger(),
	}
}

func (h *JiraHandler) OAuthStart(w http.ResponseWriter, r *http.Request) {
	if missing := h.oauthMissingConfig(); len(missing) > 0 {
		writeError(w, r, http.StatusServiceUnavailable, "JIRA_OAUTH_NOT_CONFIGURED", "jira oauth is not configured: "+strings.Join(missing, ", "))
		return
	}
	if h.redis == nil {
		writeError(w, r, http.StatusServiceUnavailable, "STATE_STORE_UNAVAILABLE", "oauth state store is unavailable")
		return
	}
	stateID := strings.TrimSpace(r.URL.Query().Get("state"))
	if stateID == "" {
		user, tenantID := requireAuth(r)
		if user == nil {
			writeError(w, r, http.StatusBadRequest, "INVALID_OAUTH_STATE", "missing oauth state")
			return
		}
		var err error
		stateID, err = h.prepareOAuthState(r.Context(), tenantID, user.ID, strings.TrimSpace(r.URL.Query().Get("name")), strings.TrimSpace(r.URL.Query().Get("project_key")))
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "STATE_STORE_FAILED", "failed to persist oauth state")
			return
		}
	} else {
		exists, err := h.redis.Exists(r.Context(), "integration:jira:oauth:"+stateID).Result()
		if err != nil {
			writeError(w, r, http.StatusInternalServerError, "STATE_STORE_FAILED", "failed to load oauth state")
			return
		}
		if exists == 0 {
			writeError(w, r, http.StatusBadRequest, "INVALID_OAUTH_STATE", "oauth state is missing or expired")
			return
		}
	}
	http.Redirect(w, r, jirasvc.BuildOAuthURL(h.oauthCfg, stateID), http.StatusFound)
}

func (h *JiraHandler) CreateOAuthSession(w http.ResponseWriter, r *http.Request) {
	user, tenantID := requireAuth(r)
	if user == nil {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	if missing := h.oauthMissingConfig(); len(missing) > 0 {
		writeError(w, r, http.StatusServiceUnavailable, "JIRA_OAUTH_NOT_CONFIGURED", "jira oauth is not configured: "+strings.Join(missing, ", "))
		return
	}
	if h.redis == nil {
		writeError(w, r, http.StatusServiceUnavailable, "STATE_STORE_UNAVAILABLE", "oauth state store is unavailable")
		return
	}

	var req jiraOAuthSessionRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, "INVALID_BODY", "invalid jira oauth session payload")
			return
		}
	}

	stateID, err := h.prepareOAuthState(r.Context(), tenantID, user.ID, req.Name, req.ProjectKey)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "STATE_STORE_FAILED", "failed to persist oauth state")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"url": jirasvc.BuildOAuthURL(h.oauthCfg, stateID),
		},
	})
}

func (h *JiraHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	if missing := h.oauthMissingConfig(); len(missing) > 0 {
		writeError(w, r, http.StatusServiceUnavailable, "JIRA_OAUTH_NOT_CONFIGURED", "jira oauth is not configured: "+strings.Join(missing, ", "))
		return
	}
	if h.redis == nil {
		writeError(w, r, http.StatusServiceUnavailable, "STATE_STORE_UNAVAILABLE", "oauth state store is unavailable")
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	stateID := strings.TrimSpace(r.URL.Query().Get("state"))
	if code == "" || stateID == "" {
		writeError(w, r, http.StatusBadRequest, "INVALID_OAUTH_CALLBACK", "missing code or state")
		return
	}

	stateRaw, err := h.redis.GetDel(r.Context(), "integration:jira:oauth:"+stateID).Bytes()
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_OAUTH_STATE", "oauth state is missing or expired")
		return
	}
	var state jiraOAuthState
	if err := json.Unmarshal(stateRaw, &state); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_OAUTH_STATE", "oauth state is invalid")
		return
	}

	tokenPayload, err := jirasvc.ExchangeCode(r.Context(), h.oauthCfg, code)
	if err != nil {
		writeError(w, r, http.StatusBadGateway, "JIRA_OAUTH_FAILED", err.Error())
		return
	}
	accessToken := stringValue(tokenPayload["access_token"])
	refreshToken := stringValue(tokenPayload["refresh_token"])
	if accessToken == "" {
		writeError(w, r, http.StatusBadGateway, "JIRA_OAUTH_FAILED", "jira oauth did not return an access token")
		return
	}

	resource, err := fetchAccessibleResource(r.Context(), accessToken)
	if err != nil {
		writeError(w, r, http.StatusBadGateway, "JIRA_OAUTH_FAILED", err.Error())
		return
	}
	config := map[string]any{
		"base_url":      stringValue(resource["url"]),
		"cloud_id":      stringValue(resource["id"]),
		"auth_token":    accessToken,
		"refresh_token": refreshToken,
	}
	if state.ProjectKey != "" {
		config["project_key"] = state.ProjectKey
	}
	name := firstNonEmpty(state.Name, "Jira "+firstNonEmpty(stringValue(resource["name"]), "Cloud"))

	var integration *intmodel.Integration
	if state.ProjectKey != "" {
		req := &intdto.CreateIntegrationRequest{
			Type:   intmodel.IntegrationTypeJira,
			Name:   name,
			Config: config,
		}
		integration, err = h.service.Create(r.Context(), state.TenantID, state.UserID, req, &intsvc.AuditActor{UserID: state.UserID})
	} else {
		integration, err = h.service.CreateSetupPending(r.Context(), state.TenantID, state.UserID, intmodel.IntegrationTypeJira, name, "Jira integration awaiting project configuration", config, &intsvc.AuditActor{UserID: state.UserID})
	}
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "JIRA_INSTALL_FAILED", err.Error())
		return
	}

	http.Redirect(w, r, h.publicAppURL+"/admin/integrations/"+integration.ID, http.StatusFound)
}

func (h *JiraHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	body, err := readBodyAndRestore(r, 1<<20)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_BODY", "failed to read jira webhook body")
		return
	}
	signature := strings.TrimSpace(r.Header.Get("X-Hub-Signature"))
	if signature == "" {
		signature = strings.TrimSpace(r.Header.Get("X-Atlassian-Webhook-Signature"))
	}
	if signature == "" {
		writeError(w, r, http.StatusUnauthorized, "UNVERIFIED_WEBHOOK", "missing jira signature")
		return
	}

	integration, cfg, err := h.findWebhookIntegration(r.Context(), signature, body)
	if err != nil {
		h.logger.Warn().Err(err).Msg("rejected jira webhook with invalid signature")
		writeError(w, r, http.StatusUnauthorized, "UNVERIFIED_WEBHOOK", "invalid jira signature")
		return
	}

	event, err := jirasvc.ParseWebhookEvent(body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_BODY", "invalid jira webhook payload")
		return
	}
	if event.WebhookEvent != "jira:issue_updated" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}

	newStatus := ""
	for _, item := range event.Changelog.Items {
		if strings.EqualFold(item.Field, "status") {
			newStatus = firstNonEmpty(item.ToString, event.Issue.Fields.Status.Name)
			break
		}
	}
	if newStatus == "" {
		newStatus = event.Issue.Fields.Status.Name
	}
	if newStatus == "" || event.Issue.ID == "" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	if _, err := h.jiraService.SyncWebhookStatus(r.Context(), integration, cfg, event.Issue.ID, newStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.logger.Debug().Str("external_id", event.Issue.ID).Msg("jira webhook did not match a linked ticket")
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}
		writeError(w, r, http.StatusBadGateway, "JIRA_SYNC_FAILED", err.Error())
		return
	}

	publishAuditEvent(r.Context(), h.producer, integration.TenantID, "integration.jira.webhook.synced", nil, map[string]any{
		"external_system": "jira",
		"external_id":     event.Issue.ID,
		"external_key":    event.Issue.Key,
		"new_status":      newStatus,
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *JiraHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	user, tenantID := requireAuth(r)
	if user == nil {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	var req createTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_BODY", "invalid json body")
		return
	}
	if strings.TrimSpace(req.IntegrationID) == "" || strings.TrimSpace(req.EntityType) == "" || strings.TrimSpace(req.EntityID) == "" {
		writeError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "integration_id, entity_type, and entity_id are required")
		return
	}

	link, err := h.service.CreateJiraTicket(r.Context(), tenantID, req.IntegrationID, req.EntityType, req.EntityID, actorFromRequest(r))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "JIRA_TICKET_CREATE_FAILED", err.Error())
		return
	}
	publishAuditEvent(r.Context(), h.producer, tenantID, "integration.jira.ticket.created", &intsvc.AuditActor{UserID: user.ID, UserEmail: user.Email}, map[string]any{
		"integration_id": req.IntegrationID,
		"entity_type":    req.EntityType,
		"entity_id":      req.EntityID,
		"external_key":   link.ExternalKey,
	})
	writeJSON(w, http.StatusCreated, map[string]any{"data": link})
}

func (h *JiraHandler) oauthMissingConfig() []string {
	missing := make([]string, 0, 2)
	if strings.TrimSpace(h.oauthCfg.ClientID) == "" {
		missing = append(missing, "NOTIF_ATLASSIAN_CLIENT_ID")
	}
	if strings.TrimSpace(h.oauthCfg.ClientSecret) == "" {
		missing = append(missing, "NOTIF_ATLASSIAN_CLIENT_SECRET")
	}
	return missing
}

func (h *JiraHandler) prepareOAuthState(ctx context.Context, tenantID, userID, name, projectKey string) (string, error) {
	stateID := events.GenerateUUID()
	state := jiraOAuthState{
		TenantID:   tenantID,
		UserID:     userID,
		Name:       strings.TrimSpace(name),
		ProjectKey: strings.TrimSpace(projectKey),
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	if err := h.redis.Set(ctx, "integration:jira:oauth:"+stateID, raw, h.stateTTL).Err(); err != nil {
		return "", err
	}
	return stateID, nil
}

func (h *JiraHandler) findWebhookIntegration(ctx context.Context, signature string, body []byte) (*intmodel.Integration, intmodel.JiraConfig, error) {
	integration, configMap, err := h.service.FindActiveByType(ctx, intmodel.IntegrationTypeJira, func(_ *intmodel.Integration, config map[string]any) bool {
		secret := strings.TrimSpace(stringValue(config["webhook_secret"]))
		return secret != "" && jirasvc.VerifyJiraSignatureValues(signature, body, secret) == nil
	})
	if err != nil {
		return nil, intmodel.JiraConfig{}, err
	}
	var cfg intmodel.JiraConfig
	if err := intsvc.DecodeInto(configMap, &cfg); err != nil {
		return nil, intmodel.JiraConfig{}, err
	}
	return integration, cfg, nil
}

func fetchAccessibleResource(ctx context.Context, accessToken string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.atlassian.com/oauth/token/accessible-resources", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("jira accessible-resources returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var resources []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&resources); err != nil {
		return nil, err
	}
	if len(resources) == 0 {
		return nil, fmt.Errorf("jira accessible-resources returned no sites")
	}
	return resources[0], nil
}
