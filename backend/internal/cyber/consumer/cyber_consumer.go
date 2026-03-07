package consumer

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/service"
	"github.com/clario360/platform/internal/events"
)

// CyberConsumer handles Kafka events relevant to the cyber service.
type CyberConsumer struct {
	assetSvc *service.AssetService
	consumer *events.Consumer
	logger   zerolog.Logger
}

// NewCyberConsumer creates a CyberConsumer and registers its event handlers.
func NewCyberConsumer(assetSvc *service.AssetService, consumer *events.Consumer, logger zerolog.Logger) *CyberConsumer {
	c := &CyberConsumer{assetSvc: assetSvc, consumer: consumer, logger: logger}

	// Register handler for scan-triggered asset enrichment
	consumer.Subscribe(events.Topics.AssetEvents, events.EventHandlerFunc(c.handleAssetEvent))

	return c
}

// Start begins consuming from subscribed topics.
func (c *CyberConsumer) Start(ctx context.Context) error {
	return c.consumer.Start(ctx)
}

// Stop gracefully stops the consumer.
func (c *CyberConsumer) Stop() error {
	return c.consumer.Stop()
}

// handleAssetEvent dispatches asset events to the appropriate handler.
func (c *CyberConsumer) handleAssetEvent(ctx context.Context, event *events.Event) error {
	c.logger.Debug().Str("type", event.Type).Str("id", event.ID).Msg("received asset event")

	switch event.Type {
	case "cyber.asset.bulk_created":
		return c.handleBulkCreated(ctx, event)
	case "cyber.asset.created":
		return c.handleAssetCreated(ctx, event)
	default:
		return nil // ignore unknown event types
	}
}

func (c *CyberConsumer) handleAssetCreated(ctx context.Context, event *events.Event) error {
	var data struct {
		AssetID  string `json:"asset_id"`
		TenantID string `json:"tenant_id"`
	}
	if err := event.Unmarshal(&data); err != nil {
		return err
	}
	// Enrichment already triggered inline in CreateAsset; this is a no-op safety net.
	c.logger.Debug().Str("asset_id", data.AssetID).Msg("asset.created event processed")
	return nil
}

func (c *CyberConsumer) handleBulkCreated(ctx context.Context, event *events.Event) error {
	var data struct {
		IDs []string `json:"ids"`
	}
	if err := event.Unmarshal(&data); err != nil {
		return err
	}

	tenantID, err := uuid.Parse(event.TenantID)
	if err != nil {
		return err
	}

	ids := make([]uuid.UUID, 0, len(data.IDs))
	for _, idStr := range data.IDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.logger.Warn().Str("id", idStr).Msg("invalid UUID in bulk_created event")
			continue
		}
		ids = append(ids, id)
	}

	c.logger.Info().Int("count", len(ids)).Msg("processing bulk enrichment from event")

	// EnrichBatch is safe to call from a consumer goroutine
	go func() {
		c.assetSvc.EnrichBatch(context.Background(), tenantID, ids)
	}()
	return nil
}
