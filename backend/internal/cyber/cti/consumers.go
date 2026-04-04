package cti

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
)

// ---------------------------------------------------------------------------
// AggregationTriggerConsumer refreshes dashboard summary tables when triggered.
// ---------------------------------------------------------------------------

type AggregationTriggerConsumer struct {
	repo        Repository
	logger      zerolog.Logger
	lastRefresh sync.Map // tenantID → time.Time
}

func NewAggregationTriggerConsumer(repo Repository, logger zerolog.Logger) *AggregationTriggerConsumer {
	return &AggregationTriggerConsumer{
		repo:   repo,
		logger: logger.With().Str("component", "cti-aggregation-consumer").Logger(),
	}
}

func (c *AggregationTriggerConsumer) Handle(ctx context.Context, event *events.Event) error {
	tenantID, err := uuid.Parse(event.TenantID)
	if err != nil {
		return nil // skip malformed
	}

	// Debounce: skip if refreshed within 30 seconds
	if last, ok := c.lastRefresh.Load(tenantID.String()); ok {
		if time.Since(last.(time.Time)) < 30*time.Second {
			return nil
		}
	}

	now := time.Now().UTC()
	periods := []struct{ start, end time.Time }{
		{now.Add(-24 * time.Hour), now},
		{now.Add(-7 * 24 * time.Hour), now},
		{now.Add(-30 * 24 * time.Hour), now},
	}

	for _, p := range periods {
		if err := c.repo.RefreshGeoThreatSummary(ctx, tenantID, p.start, p.end); err != nil {
			c.logger.Warn().Err(err).Msg("refresh geo summary")
		}
		if err := c.repo.RefreshSectorThreatSummary(ctx, tenantID, p.start, p.end); err != nil {
			c.logger.Warn().Err(err).Msg("refresh sector summary")
		}
	}
	if err := c.repo.RefreshExecutiveSnapshot(ctx, tenantID); err != nil {
		c.logger.Warn().Err(err).Msg("refresh executive snapshot")
	}

	c.lastRefresh.Store(tenantID.String(), now)
	c.logger.Debug().Str("tenant_id", tenantID.String()).Msg("aggregation refresh completed")
	return nil
}

// ---------------------------------------------------------------------------
// WebSocketBroadcastConsumer bridges CTI Kafka events to connected WS clients.
// ---------------------------------------------------------------------------

type WebSocketBroadcastConsumer struct {
	hub    *WSHub
	logger zerolog.Logger
}

func NewWebSocketBroadcastConsumer(hub *WSHub, logger zerolog.Logger) *WebSocketBroadcastConsumer {
	return &WebSocketBroadcastConsumer{
		hub:    hub,
		logger: logger.With().Str("component", "cti-ws-broadcast").Logger(),
	}
}

func (c *WebSocketBroadcastConsumer) Handle(ctx context.Context, event *events.Event) error {
	if c.hub == nil {
		return nil
	}
	tenantID := event.TenantID
	if tenantID == "" {
		return nil
	}

	// Parse the raw data for the broadcast payload
	var data json.RawMessage
	if event.Data != nil {
		data = event.Data
	} else {
		data = json.RawMessage("{}")
	}

	c.hub.Broadcast(tenantID, event.Type, data)
	return nil
}
