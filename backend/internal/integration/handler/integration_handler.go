package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	intdto "github.com/clario360/platform/internal/integration/dto"
	intmodel "github.com/clario360/platform/internal/integration/model"
	intsvc "github.com/clario360/platform/internal/integration/service"
)

type IntegrationHandler struct {
	service   *intsvc.IntegrationService
	providers []ProviderStatus
	logger    zerolog.Logger
}

type ProviderStatus struct {
	Type            intmodel.IntegrationType `json:"type"`
	Name            string                   `json:"name"`
	Description     string                   `json:"description"`
	SetupMode       string                   `json:"setup_mode"`
	Configured      bool                     `json:"configured"`
	OAuthEnabled    bool                     `json:"oauth_enabled"`
	OAuthStartURL   string                   `json:"oauth_start_url,omitempty"`
	MissingConfig   []string                 `json:"missing_config,omitempty"`
	SupportsInbound bool                     `json:"supports_inbound"`
	SupportsOutbound bool                    `json:"supports_outbound"`
}

func NewIntegrationHandler(service *intsvc.IntegrationService, providers []ProviderStatus, logger zerolog.Logger) *IntegrationHandler {
	return &IntegrationHandler{
		service:   service,
		providers: providers,
		logger:    logger.With().Str("component", "integration_handler").Logger(),
	}
}

func (h *IntegrationHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	user, _ := requireAuth(r)
	if user == nil {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": h.providers})
}

func (h *IntegrationHandler) List(w http.ResponseWriter, r *http.Request) {
	user, tenantID := requireAuth(r)
	if user == nil {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	query, err := intdto.ParseListQuery(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	items, total, err := h.service.List(r.Context(), tenantID, query)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "INTEGRATION_LIST_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": intdto.NewPagination(query.Page, query.PerPage, total),
	})
}

func (h *IntegrationHandler) Get(w http.ResponseWriter, r *http.Request) {
	user, tenantID := requireAuth(r)
	if user == nil {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	item, err := h.service.Get(r.Context(), tenantID, chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "NOT_FOUND", "integration not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": item})
}

func (h *IntegrationHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, tenantID := requireAuth(r)
	if user == nil {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	if !auth.HasPermission(user.Roles, "tenant:write") {
		writeError(w, r, http.StatusForbidden, "FORBIDDEN", "tenant:write permission is required")
		return
	}
	var req intdto.CreateIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_BODY", "invalid json body")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	item, err := h.service.Create(r.Context(), tenantID, user.ID, &req, actorFromRequest(r))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INTEGRATION_CREATE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": item})
}

func (h *IntegrationHandler) Update(w http.ResponseWriter, r *http.Request) {
	user, tenantID := requireAuth(r)
	if user == nil {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	if !auth.HasPermission(user.Roles, "tenant:write") {
		writeError(w, r, http.StatusForbidden, "FORBIDDEN", "tenant:write permission is required")
		return
	}
	var req intdto.UpdateIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_BODY", "invalid json body")
		return
	}
	item, err := h.service.Update(r.Context(), tenantID, chi.URLParam(r, "id"), &req, actorFromRequest(r))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INTEGRATION_UPDATE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": item})
}

func (h *IntegrationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user, tenantID := requireAuth(r)
	if user == nil {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	if !auth.HasPermission(user.Roles, "tenant:write") {
		writeError(w, r, http.StatusForbidden, "FORBIDDEN", "tenant:write permission is required")
		return
	}
	if err := h.service.Delete(r.Context(), tenantID, chi.URLParam(r, "id"), actorFromRequest(r)); err != nil {
		writeError(w, r, http.StatusBadRequest, "INTEGRATION_DELETE_FAILED", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *IntegrationHandler) Test(w http.ResponseWriter, r *http.Request) {
	user, tenantID := requireAuth(r)
	if user == nil {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	code, body, err := h.service.Test(r.Context(), tenantID, chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INTEGRATION_TEST_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"success":       err == nil,
			"response_code": code,
			"response_body": body,
		},
	})
}

func (h *IntegrationHandler) Deliveries(w http.ResponseWriter, r *http.Request) {
	user, tenantID := requireAuth(r)
	if user == nil {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	query, err := intdto.ParseDeliveryQuery(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_PARAMS", err.Error())
		return
	}
	items, total, err := h.service.ListDeliveries(r.Context(), tenantID, chi.URLParam(r, "id"), query)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "DELIVERY_LIST_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": intdto.NewPagination(query.Page, query.PerPage, total),
	})
}

func (h *IntegrationHandler) RetryFailed(w http.ResponseWriter, r *http.Request) {
	user, tenantID := requireAuth(r)
	if user == nil {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	count, err := h.service.RetryFailed(r.Context(), tenantID, chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "RETRY_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]int{"retried_count": count}})
}

func (h *IntegrationHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	user, tenantID := requireAuth(r)
	if user == nil {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	var req intdto.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_BODY", "invalid json body")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	if err := h.service.UpdateStatus(r.Context(), tenantID, chi.URLParam(r, "id"), req.Status, actorFromRequest(r)); err != nil {
		writeError(w, r, http.StatusBadRequest, "STATUS_UPDATE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"status": string(req.Status)}})
}

func (h *IntegrationHandler) ListTicketLinks(w http.ResponseWriter, r *http.Request) {
	user, tenantID := requireAuth(r)
	if user == nil {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	query := &intdto.TicketLinkQuery{
		IntegrationID: r.URL.Query().Get("integration_id"),
		EntityType:     r.URL.Query().Get("entity_type"),
		EntityID:       r.URL.Query().Get("entity_id"),
		ExternalSystem: r.URL.Query().Get("external_system"),
	}
	items, err := h.service.ListTicketLinks(r.Context(), tenantID, query)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "TICKET_LINK_LIST_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (h *IntegrationHandler) GetTicketLink(w http.ResponseWriter, r *http.Request) {
	user, tenantID := requireAuth(r)
	if user == nil {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	item, err := h.service.GetTicketLink(r.Context(), tenantID, chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "NOT_FOUND", "ticket link not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": item})
}

func (h *IntegrationHandler) SyncTicketLink(w http.ResponseWriter, r *http.Request) {
	user, tenantID := requireAuth(r)
	if user == nil {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	if err := h.service.ForceSync(r.Context(), tenantID, chi.URLParam(r, "id")); err != nil {
		writeError(w, r, http.StatusBadRequest, "TICKET_SYNC_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"status": "synced"}})
}

func (h *IntegrationHandler) allowedTypes() []intmodel.IntegrationType {
	return []intmodel.IntegrationType{
		intmodel.IntegrationTypeSlack,
		intmodel.IntegrationTypeTeams,
		intmodel.IntegrationTypeJira,
		intmodel.IntegrationTypeServiceNow,
		intmodel.IntegrationTypeWebhook,
	}
}
