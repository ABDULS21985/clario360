package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"

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
	prefSvc      *service.PreferenceService
	webhookRepo  *repository.WebhookRepository
	deliveryRepo *repository.DeliveryRepository
	logger       zerolog.Logger
}

// NewPreferenceHandler creates a new PreferenceHandler.
func NewPreferenceHandler(prefSvc *service.PreferenceService, webhookRepo *repository.WebhookRepository, deliveryRepo *repository.DeliveryRepository, logger zerolog.Logger) *PreferenceHandler {
	return &PreferenceHandler{
		prefSvc:      prefSvc,
		webhookRepo:  webhookRepo,
		deliveryRepo: deliveryRepo,
		logger:       logger.With().Str("component", "preference_handler").Logger(),
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

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	search := r.URL.Query().Get("search")

	webhooks, total, err := h.webhookRepo.ListByTenantPaginated(r.Context(), tenantID, page, perPage, search)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list webhooks")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list webhooks", r)
		return
	}

	if webhooks == nil {
		webhooks = []model.Webhook{}
	}

	totalPages := int(total) / perPage
	if int(total)%perPage != 0 {
		totalPages++
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": webhooks,
		"meta": map[string]interface{}{
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// GetWebhook handles GET /api/v1/notifications/webhooks/{id}.
func (h *PreferenceHandler) GetWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.TenantFromContext(r.Context())
	id := chi.URLParam(r, "id")

	wh, err := h.webhookRepo.FindByID(r.Context(), tenantID, id)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get webhook")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get webhook", r)
		return
	}
	if wh == nil {
		writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "webhook not found", r)
		return
	}

	writeJSON(w, http.StatusOK, wh)
}

func generateSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return "whsec_" + hex.EncodeToString(b)
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

	secret := generateSecret()
	retryPolicy := model.DefaultRetryPolicy()
	if req.RetryPolicy != nil {
		retryPolicy = *req.RetryPolicy
	}
	headers := req.Headers
	if headers == nil {
		headers = map[string]string{}
	}

	wh := &model.Webhook{
		TenantID:    tenantID,
		Name:        req.Name,
		URL:         req.URL,
		Secret:      &secret,
		Events:      req.Events,
		Active:      true,
		Headers:     headers,
		RetryPolicy: retryPolicy,
		CreatedBy:   user.ID,
	}

	id, err := h.webhookRepo.Insert(r.Context(), wh)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to create webhook")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create webhook", r)
		return
	}

	// Fetch the created webhook to return full object
	created, err := h.webhookRepo.FindByID(r.Context(), tenantID, id)
	if err != nil || created == nil {
		// Fallback: return minimal response
		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"webhook": map[string]interface{}{"id": id, "name": req.Name},
			"secret":  secret,
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"webhook": created,
		"secret":  secret,
	})
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

	if err := h.webhookRepo.Update(r.Context(), tenantID, id, req.Name, req.URL, req.Secret, req.Events, req.Active, req.Headers, req.RetryPolicy); err != nil {
		writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", err.Error(), r)
		return
	}

	// Return the updated webhook so the frontend receives the full NotificationWebhook object.
	updated, err := h.webhookRepo.FindByID(r.Context(), tenantID, id)
	if err != nil || updated == nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	writeJSON(w, http.StatusOK, updated)
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

// TestWebhook handles POST /api/v1/notifications/webhooks/{id}/test.
func (h *PreferenceHandler) TestWebhook(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.TenantFromContext(r.Context())
	id := chi.URLParam(r, "id")

	wh, err := h.webhookRepo.FindByID(r.Context(), tenantID, id)
	if err != nil || wh == nil {
		writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "webhook not found", r)
		return
	}

	// Send a test HTTP POST to the webhook URL
	testPayload := map[string]interface{}{
		"event": "webhook.test",
		"data":  map[string]interface{}{"message": "This is a test delivery from Clario 360", "webhook_id": id},
	}
	payloadBytes, _ := json.Marshal(testPayload)

	result := deliverWebhookTest(wh.URL, payloadBytes, wh.Secret, wh.Headers)
	writeJSON(w, http.StatusOK, result)
}

// RotateWebhookSecret handles POST /api/v1/notifications/webhooks/{id}/rotate.
func (h *PreferenceHandler) RotateWebhookSecret(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.TenantFromContext(r.Context())
	id := chi.URLParam(r, "id")

	newSecret := generateSecret()
	if err := h.webhookRepo.RotateSecret(r.Context(), tenantID, id, newSecret); err != nil {
		writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", err.Error(), r)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"secret": newSecret})
}

// ListWebhookDeliveries handles GET /api/v1/notifications/webhooks/{id}/deliveries.
func (h *PreferenceHandler) ListWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	status := r.URL.Query().Get("status")

	deliveries, total, err := h.deliveryRepo.GetWebhookDeliveries(r.Context(), id, page, perPage, status)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list webhook deliveries")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list webhook deliveries", r)
		return
	}

	if deliveries == nil {
		deliveries = []model.WebhookDelivery{}
	}

	totalPages := int(total) / perPage
	if int(total)%perPage != 0 {
		totalPages++
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": deliveries,
		"meta": map[string]interface{}{
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// RetryWebhookDelivery handles POST /api/v1/notifications/webhooks/{id}/deliveries/{deliveryId}/retry.
func (h *PreferenceHandler) RetryWebhookDelivery(w http.ResponseWriter, r *http.Request) {
	deliveryID := chi.URLParam(r, "deliveryId")

	if err := h.deliveryRepo.RetryDelivery(r.Context(), deliveryID); err != nil {
		h.logger.Error().Err(err).Msg("failed to retry delivery")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retry delivery", r)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "queued"})
}
