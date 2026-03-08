package handler

import (
	"crypto/hmac"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	intmodel "github.com/clario360/platform/internal/integration/model"
	intsvc "github.com/clario360/platform/internal/integration/service"
	snsvc "github.com/clario360/platform/internal/integration/service/servicenow"
)

type ServiceNowHandler struct {
	service   *intsvc.IntegrationService
	snService *snsvc.Service
	producer  *events.Producer
	logger    zerolog.Logger
}

func NewServiceNowHandler(
	service *intsvc.IntegrationService,
	snService *snsvc.Service,
	producer *events.Producer,
	logger zerolog.Logger,
) *ServiceNowHandler {
	return &ServiceNowHandler{
		service:   service,
		snService: snService,
		producer:  producer,
		logger:    logger.With().Str("component", "integration_servicenow_handler").Logger(),
	}
}

func (h *ServiceNowHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	body, err := readBodyAndRestore(r, 1<<20)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_BODY", "failed to read servicenow webhook body")
		return
	}
	token := strings.TrimSpace(r.Header.Get("X-ServiceNow-Token"))
	if token == "" {
		token = strings.TrimSpace(r.Header.Get("X-Clario-Webhook-Secret"))
	}
	if token == "" {
		writeError(w, r, http.StatusUnauthorized, "UNVERIFIED_WEBHOOK", "missing servicenow webhook token")
		return
	}

	integration, cfg, err := h.findWebhookIntegration(r.Context(), token)
	if err != nil {
		h.logger.Warn().Err(err).Msg("rejected servicenow webhook with invalid token")
		writeError(w, r, http.StatusUnauthorized, "UNVERIFIED_WEBHOOK", "invalid servicenow webhook token")
		return
	}

	event, err := snsvc.ParseWebhookEvent(body)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_BODY", "invalid servicenow webhook payload")
		return
	}
	result := event.Result
	externalID := stringValue(result["sys_id"])
	externalStatus := firstNonEmpty(stringValue(result["state"]), stringValue(result["status"]), stringValue(result["incident_state"]))
	if externalID == "" || externalStatus == "" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	if _, err := h.snService.SyncWebhookStatus(r.Context(), integration, cfg, externalID, externalStatus); err != nil {
		writeError(w, r, http.StatusBadGateway, "SERVICENOW_SYNC_FAILED", err.Error())
		return
	}
	publishAuditEvent(r.Context(), h.producer, integration.TenantID, "integration.servicenow.webhook.synced", nil, map[string]any{
		"external_system": "servicenow",
		"external_id":     externalID,
		"external_status": externalStatus,
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *ServiceNowHandler) CreateIncident(w http.ResponseWriter, r *http.Request) {
	_, tenantID := requireAuth(r)
	if tenantID == "" {
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

	link, err := h.service.CreateServiceNowIncident(r.Context(), tenantID, req.IntegrationID, req.EntityType, req.EntityID, actorFromRequest(r))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "SERVICENOW_INCIDENT_CREATE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": link})
}

func (h *ServiceNowHandler) findWebhookIntegration(ctx context.Context, token string) (*intmodel.Integration, intmodel.ServiceNowConfig, error) {
	integration, configMap, err := h.service.FindActiveByType(ctx, intmodel.IntegrationTypeServiceNow, func(_ *intmodel.Integration, config map[string]any) bool {
		secret := strings.TrimSpace(stringValue(config["webhook_secret"]))
		return secret != "" && hmac.Equal([]byte(secret), []byte(token))
	})
	if err != nil {
		return nil, intmodel.ServiceNowConfig{}, err
	}
	var cfg intmodel.ServiceNowConfig
	if err := intsvc.DecodeInto(configMap, &cfg); err != nil {
		return nil, intmodel.ServiceNowConfig{}, err
	}
	return integration, cfg, nil
}

