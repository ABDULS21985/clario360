package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/notification/channel"
	"github.com/clario360/platform/internal/notification/metrics"
	"github.com/clario360/platform/internal/notification/model"
	"github.com/clario360/platform/internal/notification/repository"
)

// CreateNotificationRequest is the input for creating a notification.
type CreateNotificationRequest struct {
	TenantID      string
	UserID        string
	Type          model.NotificationType
	Category      string
	Priority      string
	Title         string
	Body          string
	ActionURL     string
	SourceEventID string
	Data          map[string]interface{}
	Channels      []string
}

// NotificationService is the core orchestration service for creating and dispatching notifications.
type NotificationService struct {
	notifRepo  *repository.NotificationRepository
	prefSvc    *PreferenceService
	dispatcher *DispatcherService
	tmplSvc    *TemplateService
	producer   *events.Producer
	rdb        *redis.Client
	logger     zerolog.Logger
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(
	notifRepo *repository.NotificationRepository,
	prefSvc *PreferenceService,
	dispatcher *DispatcherService,
	tmplSvc *TemplateService,
	producer *events.Producer,
	rdb *redis.Client,
	logger zerolog.Logger,
) *NotificationService {
	return &NotificationService{
		notifRepo:  notifRepo,
		prefSvc:    prefSvc,
		dispatcher: dispatcher,
		tmplSvc:    tmplSvc,
		producer:   producer,
		rdb:        rdb,
		logger:     logger.With().Str("component", "notification_service").Logger(),
	}
}

// CreateNotification creates, persists, and dispatches a notification.
func (s *NotificationService) CreateNotification(ctx context.Context, req CreateNotificationRequest) error {
	// Render templates.
	title := req.Title
	body := req.Body
	actionURL := req.ActionURL

	if req.Data != nil {
		if rendered, err := s.tmplSvc.RenderText(title, req.Data); err == nil {
			title = rendered
		}
		if rendered, err := s.tmplSvc.RenderText(body, req.Data); err == nil {
			body = rendered
		}
		if rendered, err := s.tmplSvc.RenderText(actionURL, req.Data); err == nil {
			actionURL = rendered
		}
	}

	dataBytes, _ := json.Marshal(req.Data)

	notif := &model.Notification{
		TenantID:  req.TenantID,
		UserID:    req.UserID,
		Type:      req.Type,
		Category:  req.Category,
		Priority:  req.Priority,
		Title:     title,
		Body:      body,
		Data:      dataBytes,
		ActionURL: actionURL,
	}

	// Deduplication.
	var id string
	var err error
	if req.SourceEventID != "" {
		notif.SourceEventID = &req.SourceEventID
		id, err = s.notifRepo.InsertWithDedup(ctx, notif)
		if err != nil {
			return fmt.Errorf("insert notification: %w", err)
		}
		if id == "" {
			// Duplicate — skip silently.
			s.logger.Debug().Str("source_event_id", req.SourceEventID).Msg("duplicate notification skipped")
			metrics.DuplicatesSkipped.Inc()
			return nil
		}
	} else {
		id, err = s.notifRepo.Insert(ctx, notif)
		if err != nil {
			return fmt.Errorf("insert notification: %w", err)
		}
	}

	notif.ID = id
	metrics.NotificationsCreated.WithLabelValues(string(req.Type), req.Category).Inc()

	// Resolve preferences.
	chanPrefs, err := s.prefSvc.ResolveChannels(ctx, req.UserID, req.TenantID, req.Type)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to resolve preferences, using defaults")
		chanPrefs = model.DefaultPreferences
	}

	// Quiet hours check.
	inQuietHours := false
	if req.Priority != model.PriorityCritical {
		if qh, err := s.prefSvc.IsInQuietHours(ctx, req.UserID, req.TenantID); err == nil {
			inQuietHours = qh
		}
	}

	// Build channel deliveries.
	allowedChannels := normalizeRequestedChannels(req.Channels)
	var deliveries []channel.ChannelDelivery
	if chanPrefs.InApp && channelRequested(allowedChannels, model.ChannelInApp) {
		deliveries = append(deliveries, channel.ChannelDelivery{Channel: model.ChannelInApp})
	}
	if chanPrefs.WebSocket && channelRequested(allowedChannels, model.ChannelWebSocket) {
		deliveries = append(deliveries, channel.ChannelDelivery{Channel: model.ChannelWebSocket})
	}
	if chanPrefs.Email && channelRequested(allowedChannels, model.ChannelEmail) {
		deliveries = append(deliveries, channel.ChannelDelivery{
			Channel:  model.ChannelEmail,
			Deferred: inQuietHours,
		})
	}
	if chanPrefs.Webhook && channelRequested(allowedChannels, model.ChannelWebhook) {
		deliveries = append(deliveries, channel.ChannelDelivery{Channel: model.ChannelWebhook})
	}

	// Dispatch.
	if len(deliveries) > 0 {
		s.dispatcher.Dispatch(ctx, notif, deliveries)
	}

	// Publish event.
	if s.producer != nil {
		evt, evtErr := events.NewEvent(
			"com.clario360.notification.created",
			"clario360/notification-service",
			req.TenantID,
			map[string]interface{}{
				"notification_id": id,
				"user_id":         req.UserID,
				"type":            string(req.Type),
				"priority":        req.Priority,
			},
		)
		if evtErr == nil {
			if pubErr := s.producer.Publish(ctx, events.Topics.NotificationEvents, evt); pubErr != nil {
				s.logger.Warn().Err(pubErr).Msg("failed to publish notification.created event")
			}
		}
	}

	return nil
}

// GetByID returns a single notification.
func (s *NotificationService) GetByID(ctx context.Context, tenantID, userID, id string) (*model.Notification, error) {
	return s.notifRepo.FindByID(ctx, tenantID, userID, id)
}

// MarkRead marks a notification as read.
func (s *NotificationService) MarkRead(ctx context.Context, tenantID, userID, id string) error {
	return s.notifRepo.MarkRead(ctx, tenantID, userID, id)
}

// MarkAllRead marks all unread notifications as read.
func (s *NotificationService) MarkAllRead(ctx context.Context, tenantID, userID string) (int64, error) {
	return s.notifRepo.MarkAllRead(ctx, tenantID, userID)
}

// Delete deletes a notification.
func (s *NotificationService) Delete(ctx context.Context, tenantID, userID, id string) error {
	return s.notifRepo.Delete(ctx, tenantID, userID, id)
}

// BulkDelete deletes multiple notifications by ID.
func (s *NotificationService) BulkDelete(ctx context.Context, tenantID, userID string, ids []string) (int64, error) {
	return s.notifRepo.BulkDelete(ctx, tenantID, userID, ids)
}

// UnreadCount returns the unread notification count.
func (s *NotificationService) UnreadCount(ctx context.Context, tenantID, userID string) (int64, error) {
	return s.notifRepo.UnreadCount(ctx, tenantID, userID)
}

func normalizeRequestedChannels(channels []string) map[string]struct{} {
	if len(channels) == 0 {
		return nil
	}

	allowed := make(map[string]struct{}, len(channels))
	for _, channelName := range channels {
		switch strings.TrimSpace(strings.ToLower(channelName)) {
		case "in_app":
			allowed[model.ChannelInApp] = struct{}{}
		case "email":
			allowed[model.ChannelEmail] = struct{}{}
		case "webhook":
			allowed[model.ChannelWebhook] = struct{}{}
		case "push", "websocket":
			allowed[model.ChannelWebSocket] = struct{}{}
		}
	}
	if len(allowed) == 0 {
		return nil
	}
	return allowed
}

func channelRequested(allowed map[string]struct{}, channelName string) bool {
	if len(allowed) == 0 {
		return true
	}
	_, ok := allowed[channelName]
	return ok
}
