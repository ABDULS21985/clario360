package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/notification/dto"
	"github.com/clario360/platform/internal/notification/repository"
	"github.com/clario360/platform/internal/notification/service"
)

// NotificationHandler handles notification REST endpoints.
type NotificationHandler struct {
	notifSvc  *service.NotificationService
	notifRepo *repository.NotificationRepository
	logger    zerolog.Logger
}

// NewNotificationHandler creates a new NotificationHandler.
func NewNotificationHandler(notifSvc *service.NotificationService, notifRepo *repository.NotificationRepository, logger zerolog.Logger) *NotificationHandler {
	return &NotificationHandler{
		notifSvc:  notifSvc,
		notifRepo: notifRepo,
		logger:    logger.With().Str("component", "notification_handler").Logger(),
	}
}

// ListNotifications handles GET /api/v1/notifications.
func (h *NotificationHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	tenantID := auth.TenantFromContext(r.Context())

	params, err := dto.ParseQueryParams(r)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_PARAMS", err.Error(), r)
		return
	}

	params.TenantID = tenantID
	params.UserID = user.ID

	notifications, total, err := h.notifRepo.Query(r.Context(), params)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to query notifications")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to query notifications", r)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": notifications,
		"meta": dto.NewPagination(params.Page, params.PerPage, total),
	})
}

// UnreadCount handles GET /api/v1/notifications/unread-count.
func (h *NotificationHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	tenantID := auth.TenantFromContext(r.Context())

	count, err := h.notifSvc.UnreadCount(r.Context(), tenantID, user.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get unread count")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get unread count", r)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"count": count})
}

// GetNotification handles GET /api/v1/notifications/{id}.
func (h *NotificationHandler) GetNotification(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	tenantID := auth.TenantFromContext(r.Context())
	id := chi.URLParam(r, "id")

	notif, err := h.notifSvc.GetByID(r.Context(), tenantID, user.ID, id)
	if err != nil {
		h.logger.Error().Err(err).Str("id", id).Msg("failed to get notification")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get notification", r)
		return
	}
	if notif == nil {
		writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", "notification not found", r)
		return
	}

	writeJSON(w, http.StatusOK, notif)
}

// MarkRead handles PUT /api/v1/notifications/{id}/read.
func (h *NotificationHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	tenantID := auth.TenantFromContext(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.notifSvc.MarkRead(r.Context(), tenantID, user.ID, id); err != nil {
		writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", err.Error(), r)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// MarkAllRead handles PUT /api/v1/notifications/read-all.
func (h *NotificationHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	tenantID := auth.TenantFromContext(r.Context())

	count, err := h.notifSvc.MarkAllRead(r.Context(), tenantID, user.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to mark all read")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to mark all as read", r)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"marked": count})
}

// DeleteNotification handles DELETE /api/v1/notifications/{id}.
func (h *NotificationHandler) DeleteNotification(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	tenantID := auth.TenantFromContext(r.Context())
	id := chi.URLParam(r, "id")

	if err := h.notifSvc.Delete(r.Context(), tenantID, user.ID, id); err != nil {
		writeErrorResponse(w, http.StatusNotFound, "NOT_FOUND", err.Error(), r)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeErrorResponse(w http.ResponseWriter, status int, code, message string, r *http.Request) {
	resp := map[string]interface{}{
		"code":    code,
		"message": message,
	}
	if reqID := r.Context().Value("request_id"); reqID != nil {
		resp["request_id"] = reqID
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}
