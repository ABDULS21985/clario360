package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/notification/dto"
	"github.com/clario360/platform/internal/notification/model"
	"github.com/clario360/platform/internal/notification/repository"
	"github.com/clario360/platform/internal/notification/service"
)

// PreferenceHandler handles preference and webhook REST endpoints.
type PreferenceHandler struct {
	prefSvc     *service.PreferenceService
	webhookRepo *repository.WebhookRepository
	logger      zerolog.Logger
}

// NewPreferenceHandler creates a new PreferenceHandler.
func NewPreferenceHandler(prefSvc *service.PreferenceService, webhookRepo *repository.WebhookRepository, logger zerolog.Logger) *PreferenceHandler {
	return &PreferenceHandler{
		prefSvc:     prefSvc,
		webhookRepo: webhookRepo,
		logger:      logger.With().Str("component", "preference_handler").Logger(),
	}
}

// GetPreferences handles GET /api/v1/notifications/preferences.
func (h *PreferenceHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	tenantID := auth.TenantFromContext(r.Context())

	pref, err := h.prefSvc.Get(r.Context(), user.ID, tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get preferences")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get preferences", r)
		return
	}

	writeJSON(w, http.StatusOK, pref)
}

// UpdatePreferences handles PUT /api/v1/notifications/preferences.
func (h *PreferenceHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	tenantID := auth.TenantFromContext(r.Context())

	var req dto.PreferenceUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON body", r)
		return
	}

	if err := req.Validate(); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), r)
		return
	}

	if err := h.prefSvc.Update(r.Context(), user.ID, tenantID, req.GlobalPrefs, req.PerTypePrefs, req.QuietHours, req.DigestConfig); err != nil {
		h.logger.Error().Err(err).Msg("failed to update preferences")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update preferences", r)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ListWebhooks handles GET /api/v1/notifications/webhooks.
func (h *PreferenceHandler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.TenantFromContext(r.Context())

	webhooks, err := h.webhookRepo.ListByTenant(r.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list webhooks")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list webhooks", r)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"webhooks": webhooks})
}

// CreateWebhook handles POST /api/v1/notifications/webhooks.
func (h *PreferenceHandler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	tenantID := auth.TenantFromContext(r.Context())

	var req dto.WebhookCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON body", r)
		return
	}
	if err := req.Validate(); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), r)
		return
	}

	wh := &model.Webhook{
		TenantID:   tenantID,
		Name:       req.Name,
		URL:        req.URL,
		EventTypes: req.EventTypes,
		Active:     true,
		CreatedBy:  user.ID,
	}
	if req.Secret != "" {
		wh.Secret = &req.Secret
	}

	id, err := h.webhookRepo.Insert(r.Context(), wh)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to create webhook")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create webhook", r)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

// UpdateWebhook handles PUT /api/v1/notifications/webhooks/{id}.
func (h *PreferenceHandler) UpdateWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.TenantFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req dto.WebhookUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON body", r)
		return
	}

	if err := h.webhookRepo.Update(r.Context(), tenantID, id, req.Name, req.URL, req.Secret, req.EventTypes, req.Active); err != nil {
		writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", err.Error(), r)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// DeleteWebhook handles DELETE /api/v1/notifications/webhooks/{id}.
func (h *PreferenceHandler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.TenantFromContext(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.webhookRepo.Deactivate(r.Context(), tenantID, id); err != nil {
		writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", err.Error(), r)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
