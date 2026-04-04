package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	intmodel "github.com/clario360/platform/internal/integration/model"
	intrepo "github.com/clario360/platform/internal/integration/repository"
	intsvc "github.com/clario360/platform/internal/integration/service"
)

type IntegrationConsumer struct {
	consumer      *events.Consumer
	repo          *intrepo.IntegrationRepository
	delivery      *intsvc.DeliveryService
	redis         *redis.Client
	cacheTTL      time.Duration
	logger        zerolog.Logger
}

func NewIntegrationConsumer(
	consumer *events.Consumer,
	repo *intrepo.IntegrationRepository,
	delivery *intsvc.DeliveryService,
	redis *redis.Client,
	cacheTTL time.Duration,
	logger zerolog.Logger,
) *IntegrationConsumer {
	if cacheTTL <= 0 {
		cacheTTL = time.Minute
	}
	return &IntegrationConsumer{
		consumer: consumer,
		repo:     repo,
		delivery: delivery,
		redis:    redis,
		cacheTTL: cacheTTL,
		logger:   logger.With().Str("component", "integration_consumer").Logger(),
	}
}

func (c *IntegrationConsumer) Start(ctx context.Context) error {
	topics := make([]string, 0, len(events.AllTopics()))
	for _, topic := range events.AllTopics() {
		if topic == events.Topics.DeadLetter {
			continue
		}
		topics = append(topics, topic)
	}

	handler := events.EventHandlerFunc(func(ctx context.Context, event *events.Event) error {
		return c.handleEvent(ctx, event)
	})
	for _, topic := range topics {
		c.consumer.Subscribe(topic, handler)
	}
	c.logger.Info().Strs("topics", topics).Msg("integration consumer starting")
	return c.consumer.Start(ctx)
}

func (c *IntegrationConsumer) Stop() error {
	if c == nil || c.consumer == nil {
		return nil
	}
	return c.consumer.Stop()
}

func (c *IntegrationConsumer) handleEvent(ctx context.Context, event *events.Event) error {
	if event == nil || event.TenantID == "" {
		return nil
	}
	active, err := c.loadActiveIntegrations(ctx, event.TenantID)
	if err != nil {
		return err
	}
	for idx := range active {
		if !intsvc.MatchesEventFilters(event, active[idx].EventFilters) {
			continue
		}
		if _, err := c.delivery.QueueEvent(ctx, &active[idx], event); err != nil {
			c.logger.Warn().
				Err(err).
				Str("event_id", event.ID).
				Str("integration_id", active[idx].ID).
				Msg("failed to queue integration delivery")
		}
	}
	return nil
}

func (c *IntegrationConsumer) loadActiveIntegrations(ctx context.Context, tenantID string) ([]intmodel.Integration, error) {
	cacheKey := "active_integrations:" + tenantID
	if c.redis != nil {
		if raw, err := c.redis.Get(ctx, cacheKey).Bytes(); err == nil && len(raw) > 0 {
			var cached []intmodel.Integration
			if err := json.Unmarshal(raw, &cached); err == nil {
				return cached, nil
			}
		}
	}

	items, err := c.repo.ListActiveByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("load active integrations: %w", err)
	}
	if c.redis != nil {
		if raw, err := json.Marshal(items); err == nil {
			_ = c.redis.Set(ctx, cacheKey, raw, c.cacheTTL).Err()
		}
	}
	return items, nil
}
