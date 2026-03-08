package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/notification/metrics"
	"github.com/clario360/platform/internal/notification/service"
)

type notificationCreator interface {
	CreateNotification(ctx context.Context, req service.CreateNotificationRequest) error
}

type recipientLookup interface {
	ResolveByRoles(ctx context.Context, tenantID string, roles []string) ([]string, error)
	GetUserEmail(ctx context.Context, userID string) (string, error)
}

// NotificationConsumer consumes domain events from Kafka and creates notifications.
type NotificationConsumer struct {
	consumer          *events.Consumer
	ruleEngine        *RuleEngine
	recipientResolver recipientLookup
	notifSvc          notificationCreator
	guard             *events.IdempotencyGuard
	metrics           *events.CrossSuiteMetrics
	logger            zerolog.Logger
}

// NewNotificationConsumer creates a new NotificationConsumer.
func NewNotificationConsumer(
	consumer *events.Consumer,
	notifSvc notificationCreator,
	recipientResolver recipientLookup,
	guard *events.IdempotencyGuard,
	metrics *events.CrossSuiteMetrics,
	logger zerolog.Logger,
) *NotificationConsumer {
	return &NotificationConsumer{
		consumer:          consumer,
		ruleEngine:        NewRuleEngine(),
		recipientResolver: recipientResolver,
		notifSvc:          notifSvc,
		guard:             guard,
		metrics:           metrics,
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
	var data map[string]interface{}
	if len(event.Data) > 0 {
		if err := json.Unmarshal(event.Data, &data); err != nil {
			c.logger.Warn().
				Err(err).
				Str("event_type", event.Type).
				Str("event_id", event.ID).
				Msg("malformed event data")
			metrics.ConsumerEventsProcessed.WithLabelValues(event.Type, "skipped").Inc()
			return nil
		}
	}
	if event.TenantID == "" {
		c.logger.Warn().Str("event_type", event.Type).Str("event_id", event.ID).Msg("missing tenant id")
		metrics.ConsumerEventsProcessed.WithLabelValues(event.Type, "skipped").Inc()
		return nil
	}

	if c.guard != nil {
		processed, err := c.guard.IsProcessed(ctx, event.ID)
		if err != nil {
			return err
		}
		if processed {
			c.recordIdempotentSkip(event.Type)
			return nil
		}
	}

	matched := c.ruleEngine.MatchData(event, data)
	if len(matched) == 0 {
		if c.guard != nil {
			if releaseErr := c.guard.Release(ctx, event.ID); releaseErr != nil {
				c.logger.Warn().Err(releaseErr).Str("event_id", event.ID).Msg("failed to release notification idempotency lock")
			}
		}
		return nil
	}

	for _, match := range matched {
		if err := c.processMatch(ctx, event, match); err != nil {
			if c.guard != nil {
				_ = c.guard.Release(ctx, event.ID)
			}
			c.logger.Warn().
				Err(err).
				Str("event_type", event.Type).
				Str("notif_type", string(match.Rule.NotifType)).
				Msg("failed to process notification rule match")
			metrics.ConsumerEventsProcessed.WithLabelValues(event.Type, "error").Inc()
			return err
		}
		metrics.ConsumerEventsProcessed.WithLabelValues(event.Type, "success").Inc()
	}

	if c.guard != nil {
		if err := c.guard.MarkProcessed(ctx, event.ID); err != nil {
			return err
		}
	}
	return nil
}

func (c *NotificationConsumer) processMatch(ctx context.Context, event *events.Event, match MatchedRule) error {
	rule := match.Rule
	data := match.Data
	priority := ResolvePriority(rule, data)

	userIDs, err := c.resolveRecipients(ctx, event, rule, data)
	if err != nil {
		return err
	}
	if len(userIDs) == 0 {
		c.logger.Debug().
			Str("event_type", event.Type).
			Str("notif_type", string(rule.NotifType)).
			Msg("no recipients resolved")
		return nil
	}

	emailRequired := requiresEmail(rule.Channels)
	emailCache := make(map[string]string, len(userIDs))

	for _, userID := range userIDs {
		payload := cloneMap(data)
		if emailRequired {
			email, ok := emailCache[userID]
			if !ok {
				resolved, err := c.recipientResolver.GetUserEmail(ctx, userID)
				if err != nil {
					return fmt.Errorf("resolve user email %s: %w", userID, err)
				}
				email = resolved
				emailCache[userID] = email
			}
			payload["email"] = email
		}

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
			Data:          payload,
			Channels:      rule.Channels,
		}

		if err := c.notifSvc.CreateNotification(ctx, req); err != nil {
			return fmt.Errorf("create notification for %s: %w", userID, err)
		}
		if c.metrics != nil {
			c.metrics.NotificationsTriggeredTotal.WithLabelValues(c.consumerName(), string(rule.NotifType)).Inc()
		}
	}

	return nil
}

func (c *NotificationConsumer) resolveRecipients(ctx context.Context, event *events.Event, rule *NotificationRule, data map[string]interface{}) ([]string, error) {
	switch rule.RecipientMode {
	case RecipientDirect:
		return ResolveDirectUserIDs(rule, data), nil
	case RecipientRoleBased:
		roles := ResolveRoles(rule, data)
		if len(roles) == 0 {
			return nil, nil
		}
		return c.recipientResolver.ResolveByRoles(ctx, event.TenantID, roles)
	case RecipientMixed:
		recipients := ResolveDirectUserIDs(rule, data)
		roles := ResolveRoles(rule, data)
		if len(roles) > 0 {
			roleRecipients, err := c.recipientResolver.ResolveByRoles(ctx, event.TenantID, roles)
			if err != nil {
				return nil, err
			}
			recipients = append(recipients, roleRecipients...)
		}
		return uniqueStrings(recipients), nil
	case RecipientTenantBroadcast:
		if event.UserID == "" {
			return nil, nil
		}
		return []string{event.UserID}, nil
	default:
		return nil, nil
	}
}

func (c *NotificationConsumer) recordIdempotentSkip(eventType string) {
	if c.metrics != nil {
		c.metrics.SkippedIdempotentTotal.WithLabelValues(c.consumerName(), eventType).Inc()
	}
	metrics.ConsumerEventsProcessed.WithLabelValues(eventType, "skipped").Inc()
}

func requiresEmail(channels []string) bool {
	for _, channel := range channels {
		if channel == "email" {
			return true
		}
	}
	return false
}

func (c *NotificationConsumer) consumerName() string {
	if c != nil && c.consumer != nil {
		return c.consumer.GroupID()
	}
	return "notification-consumer"
}
