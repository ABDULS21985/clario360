package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/notification/channel"
	"github.com/clario360/platform/internal/notification/dto"
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

// notifTypeFromShort maps short frontend type names to backend NotificationType constants.
func notifTypeFromShort(short string) model.NotificationType {
	switch short {
	case "alert":
		return model.NotifAlertCreated
	case "task":
		return model.NotifTaskAssigned
	case "approval":
		return model.NotifRemediationApproval
	case "system":
		return model.NotifSystemMaintenance
	case "mention":
		return model.NotifActionItemAssigned
	case "deadline":
		return model.NotifTaskOverdue
	case "completion":
		return model.NotifWorkflowCompleted
	case "error":
		return model.NotifPipelineFailed
	case "report":
		return model.NotifAnalysisReady
	default:
		return model.NotifSystemMaintenance
	}
}

// categoryFromType returns the category for a notification type.
func categoryFromType(t model.NotificationType) string {
	switch t {
	case model.NotifAlertCreated, model.NotifAlertEscalated, model.NotifSecurityIncident, model.NotifLoginAnomaly, model.NotifPasswordExpiring, model.NotifMalwareDetected:
		return model.CategorySecurity
	case model.NotifPipelineFailed, model.NotifPipelineCompleted, model.NotifQualityIssue, model.NotifContradictionFound:
		return model.CategoryData
	case model.NotifContractExpiring, model.NotifContractCreated, model.NotifClauseRiskFlagged, model.NotifAnalysisReady:
		return model.CategoryLegal
	case model.NotifMeetingScheduled, model.NotifMeetingReminder, model.NotifActionItemAssigned, model.NotifActionItemOverdue, model.NotifMinutesApproved, model.NotifKPIThreshold:
		return model.CategoryGovernance
	case model.NotifTaskAssigned, model.NotifTaskOverdue, model.NotifTaskEscalated, model.NotifRemediationApproval, model.NotifRemediationCompleted, model.NotifRemediationFailed, model.NotifWorkflowFailed, model.NotifWorkflowCompleted:
		return model.CategoryWorkflow
	default:
		return model.CategorySystem
	}
}

// SendTestNotification handles POST /api/v1/notifications/test.
func (h *AdminHandler) SendTestNotification(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	tenantID := auth.TenantFromContext(r.Context())

	var reqBody dto.TestNotificationRequest
	_ = json.NewDecoder(r.Body).Decode(&reqBody)

	notifType := model.NotifSystemMaintenance
	if reqBody.Type != "" {
		notifType = notifTypeFromShort(reqBody.Type)
	}

	req := service.CreateNotificationRequest{
		TenantID:  tenantID,
		UserID:    user.ID,
		Type:      notifType,
		Category:  categoryFromType(notifType),
		Priority:  model.PriorityLow,
		Title:     "Test Notification",
		Body:      "This is a test notification sent at " + time.Now().UTC().Format(time.RFC3339),
		ActionURL: "",
		Data:      map[string]interface{}{"test": true, "email": user.Email, "type": reqBody.Type, "channel": reqBody.Channel},
	}

	if err := h.notifSvc.CreateNotification(r.Context(), req); err != nil {
		h.logger.Error().Err(err).Msg("failed to create test notification")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to send test notification", r)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Test notification sent successfully",
	})
}

// parsePeriod converts a period string to a time.Duration and label.
func parsePeriod(p string) (time.Duration, string) {
	switch p {
	case "30d":
		return 30 * 24 * time.Hour, "30d"
	case "90d":
		return 90 * 24 * time.Hour, "90d"
	default:
		return 7 * 24 * time.Hour, "7d"
	}
}

// GetDeliveryStats handles GET /api/v1/notifications/delivery-stats.
func (h *AdminHandler) GetDeliveryStats(w http.ResponseWriter, r *http.Request) {
	tenantID := auth.TenantFromContext(r.Context())

	periodParam := r.URL.Query().Get("period")
	channelParam := r.URL.Query().Get("channel")
	duration, periodLabel := parsePeriod(periodParam)
	since := time.Now().UTC().Add(-duration)

	stats, err := h.deliveryRepo.GetRichDeliveryStats(r.Context(), tenantID, since, periodLabel, channelParam)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get delivery stats")
		writeErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get delivery stats", r)
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// RetryFailed handles POST /api/v1/notifications/retry-failed.
func (h *AdminHandler) RetryFailed(w http.ResponseWriter, r *http.Request) {
	var reqBody dto.RetryFailedRequest
	_ = json.NewDecoder(r.Body).Decode(&reqBody)

	since := time.Now().UTC().Add(-24 * time.Hour)
	if reqBody.Since != "" {
		if parsed, err := time.Parse(time.RFC3339, reqBody.Since); err == nil {
			since = parsed
		}
	}

	failed, err := h.deliveryRepo.GetFailedRecentFiltered(r.Context(), reqBody.Channel, since)
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
		"retried": retried,
		"message": "Retry completed",
	})
}
