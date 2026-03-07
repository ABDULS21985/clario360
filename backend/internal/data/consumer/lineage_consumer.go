package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	datadashboard "github.com/clario360/platform/internal/data/dashboard"
	"github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/data/service"
	"github.com/clario360/platform/internal/events"
)

type LineageConsumer struct {
	lineageService *service.LineageService
	pipelineRepo   *repository.PipelineRepository
	runRepo        *repository.PipelineRunRepository
	cache          *datadashboard.Cache
	consumer       *events.Consumer
	logger         zerolog.Logger
}

func NewLineageConsumer(lineageService *service.LineageService, pipelineRepo *repository.PipelineRepository, runRepo *repository.PipelineRunRepository, cache *datadashboard.Cache, consumer *events.Consumer, logger zerolog.Logger) *LineageConsumer {
	handler := &LineageConsumer{
		lineageService: lineageService,
		pipelineRepo:   pipelineRepo,
		runRepo:        runRepo,
		cache:          cache,
		consumer:       consumer,
		logger:         logger,
	}
	consumer.Subscribe("data.pipeline.events", handler)
	consumer.Subscribe("data.source.events", handler)
	consumer.Subscribe("data.quality.events", handler)
	consumer.Subscribe("data.contradiction.events", handler)
	consumer.Subscribe("data.darkdata.events", handler)
	consumer.Subscribe("data.lineage.events", handler)
	consumer.Subscribe("data.analytics.events", handler)
	return handler
}

func (c *LineageConsumer) Start(ctx context.Context) error {
	return c.consumer.Start(ctx)
}

func (c *LineageConsumer) Stop() error {
	return c.consumer.Stop()
}

func (c *LineageConsumer) Handle(ctx context.Context, event *events.Event) error {
	tenantID, err := uuid.Parse(event.TenantID)
	if err != nil {
		return fmt.Errorf("parse lineage consumer tenant: %w", err)
	}
	defer func() {
		_ = c.cache.Invalidate(ctx, tenantID)
	}()

	switch event.Type {
	case "com.clario360.data.pipeline.run.completed":
		var payload struct {
			ID         uuid.UUID `json:"id"`
			PipelineID uuid.UUID `json:"pipeline_id"`
		}
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return fmt.Errorf("decode pipeline lineage payload: %w", err)
		}
		pipelineItem, err := c.pipelineRepo.Get(ctx, tenantID, payload.PipelineID)
		if err != nil {
			return err
		}
		run, err := c.runRepo.Get(ctx, tenantID, payload.PipelineID, payload.ID)
		if err != nil {
			return err
		}
		return c.lineageService.RecordPipelineRun(ctx, pipelineItem, run)
	default:
		return nil
	}
}
