package handler

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/notification/channel"
	"github.com/clario360/platform/internal/notification/model"
	"github.com/clario360/platform/internal/notification/repository"
	"github.com/clario360/platform/internal/notification/service"
)

// AdminHandler handles operational endpoints.
type AdminHandler struct {
	notifSvc     *service.NotificationService
	deliveryRepo *repository.DeliveryRepository
	dispatcher   *service.DispatcherService
	logger       zerolog.Logger
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(
	notifSvc *service.NotificationService,
	deliveryRepo *repository.DeliveryRepository,
	dispatcher *service.DispatcherService,
	logger zerolog.Logger,
) *AdminHandler {
	return &AdminHandler{
		notifSvc:     notifSvc,
		deliveryRepo: deliveryRepo,
		dispatcher:   dispatcher,
		logger:       logger.With().Str("component", "admin_handler").Logger(),
	}
}

// SendTestNotification handles POST /api/v1/notifications/test.
func (h *AdminHandler) SendTestNotification(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	tenantID := auth.TenantFromContext(r.Context())

	req := service.CreateNotificationRequest{
		TenantID:  tenantID,
		UserID:    user.ID,
		Type:      model.NotifSystemMaintenance,
		Category:  model.CategorySystem,
		Priority:  model.PriorityLow,
		Title:     "Test Notification",
		Body:      "This is a test notification sent at " + time.Now().UTC().Format(time.RFC3339),
		ActionURL: "",
		Data:      map[string]interface{}{"test": true, "email": user.Email},
	}

	if err := h.notifSvc.CreateNotification(r.Context(), req); err != nil {
		h.logger.Error().Err(err).Msg("failed to create test notification")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to send test notification", r)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

// GetDeliveryStats handles GET /api/v1/notifications/delivery-stats.
func (h *AdminHandler) GetDeliveryStats(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.TenantFromContext(r.Context())
	since := time.Now().UTC().Add(-24 * time.Hour)

	stats, err := h.deliveryRepo.GetDeliveryStats(r.Context(), tenantID, since)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get delivery stats")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get delivery stats", r)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"stats":  stats,
		"period": "last_24h",
	})
}

// RetryFailed handles POST /api/v1/notifications/retry-failed.
func (h *AdminHandler) RetryFailed(w http.ResponseWriter, r *http.Request) {
	failed, err := h.deliveryRepo.GetFailedRecent(r.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get failed deliveries")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get failed deliveries", r)
		return
	}

	retried := 0
	for _, rec := range failed {
		notif, err := h.deliveryRepo.GetNotificationByID(r.Context(), rec.NotificationID)
		if err != nil {
			h.logger.Warn().Err(err).Str("notification_id", rec.NotificationID).Msg("failed to load notification for retry")
			continue
		}

		// Re-dispatch for the failed channel.
		results := h.dispatcher.Dispatch(r.Context(), notif, []channel.ChannelDelivery{
			{Channel: rec.Channel},
		})

		for _, result := range results {
			if result.Success {
				retried++
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total_failed": len(failed),
		"retried":      retried,
	})
}
