package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/service"
	"github.com/clario360/platform/internal/events"
)

type DataConsumer struct {
	sourceService *service.SourceService
	consumer      *events.Consumer
	logger        zerolog.Logger
}

func NewDataConsumer(sourceService *service.SourceService, consumer *events.Consumer, logger zerolog.Logger) *DataConsumer {
	handler := &DataConsumer{
		sourceService: sourceService,
		consumer:      consumer,
		logger:        logger,
	}
	consumer.Subscribe("data.source.events", handler)
	return handler
}

func (c *DataConsumer) Handle(ctx context.Context, event *events.Event) error {
	switch event.Type {
	case "com.clario360.data.source.created":
		var payload struct {
			ID uuid.UUID `json:"id"`
		}
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return fmt.Errorf("decode data source created event: %w", err)
		}
		discoveryCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		_, err := c.sourceService.DiscoverSchema(discoveryCtx, uuid.MustParse(event.TenantID), payload.ID)
		return err
	default:
		return nil
	}
}

func (c *DataConsumer) Start(ctx context.Context) error {
	return c.consumer.Start(ctx)
}

func (c *DataConsumer) Stop() error {
	return c.consumer.Stop()
}
