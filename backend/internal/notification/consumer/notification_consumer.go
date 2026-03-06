package consumer

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/notification/metrics"
	"github.com/clario360/platform/internal/notification/service"
)

// NotificationConsumer consumes domain events from Kafka and creates notifications.
type NotificationConsumer struct {
	consumer          *events.Consumer
	ruleEngine        *RuleEngine
	recipientResolver *RecipientResolver
	notifSvc          *service.NotificationService
	logger            zerolog.Logger
}

// NewNotificationConsumer creates a new NotificationConsumer.
func NewNotificationConsumer(
	consumer *events.Consumer,
	notifSvc *service.NotificationService,
	recipientResolver *RecipientResolver,
	logger zerolog.Logger,
) *NotificationConsumer {
	return &NotificationConsumer{
		consumer:          consumer,
		ruleEngine:        NewRuleEngine(),
		recipientResolver: recipientResolver,
		notifSvc:          notifSvc,
		logger:            logger.With().Str("component", "notification_consumer").Logger(),
	}
}

// Start subscribes to all relevant topics and begins processing events.
func (c *NotificationConsumer) Start(ctx context.Context) error {
	topics := ExtractEventTopics()

	handler := events.EventHandlerFunc(func(ctx context.Context, event *events.Event) error {
		return c.handleEvent(ctx, event)
	})

	for _, topic := range topics {
		c.consumer.Subscribe(topic, handler)
	}

	c.logger.Info().Strs("topics", topics).Msg("notification consumer starting")
	return c.consumer.Start(ctx)
}

// Stop gracefully shuts down the consumer.
func (c *NotificationConsumer) Stop() error {
	return c.consumer.Stop()
}

func (c *NotificationConsumer) handleEvent(ctx context.Context, event *events.Event) error {
	matched := c.ruleEngine.Match(event)
	if len(matched) == 0 {
		return nil
	}

	for _, m := range matched {
		if err := c.processMatch(ctx, event, m); err != nil {
			c.logger.Warn().Err(err).
				Str("event_type", event.Type).
				Str("notif_type", string(m.Rule.NotifType)).
				Msg("failed to process notification rule match")
			metrics.ConsumerEventsProcessed.WithLabelValues(event.Type, "error").Inc()
		} else {
			metrics.ConsumerEventsProcessed.WithLabelValues(event.Type, "success").Inc()
		}
	}

	return nil
}

func (c *NotificationConsumer) processMatch(ctx context.Context, event *events.Event, match MatchedRule) error {
	rule := match.Rule
	data := match.Data

	priority := ResolvePriority(rule, data)

	// Resolve recipients.
	var userIDs []string

	switch rule.RecipientMode {
	case RecipientDirect:
		userIDs = ResolveDirectUserIDs(rule, data)
	case RecipientRoleBased:
		roles := ResolveRoles(rule, data)
		if len(roles) > 0 {
			resolved, err := c.recipientResolver.ResolveByRoles(ctx, event.TenantID, roles)
			if err != nil {
				c.logger.Warn().Err(err).Strs("roles", roles).Msg("failed to resolve role-based recipients")
			}
			userIDs = resolved
		}
	case RecipientTenantBroadcast:
		// For broadcast, we use the event's user ID as a placeholder;
		// the actual broadcast happens via WebSocket hub at dispatch time.
		if event.UserID != "" {
			userIDs = []string{event.UserID}
		}
	}

	if len(userIDs) == 0 {
		c.logger.Debug().
			Str("event_type", event.Type).
			Str("notif_type", string(rule.NotifType)).
			Msg("no recipients resolved")
		return nil
	}

	for _, userID := range userIDs {
		req := service.CreateNotificationRequest{
			TenantID:      event.TenantID,
			UserID:        userID,
			Type:          rule.NotifType,
			Category:      rule.Category,
			Priority:      priority,
			Title:         rule.TitleTemplate,
			Body:          rule.BodyTemplate,
			ActionURL:     rule.ActionURLTmpl,
			SourceEventID: event.ID,
			Data:          data,
		}

		if err := c.notifSvc.CreateNotification(ctx, req); err != nil {
			c.logger.Warn().Err(err).
				Str("user_id", userID).
				Str("notif_type", string(rule.NotifType)).
				Msg("failed to create notification for user")
		}
	}

	return nil
}
