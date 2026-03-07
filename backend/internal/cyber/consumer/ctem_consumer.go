package consumer

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/ctem"
	"github.com/clario360/platform/internal/events"
)

type CTEMConsumer struct {
	engine   *ctem.CTEMEngine
	consumer *events.Consumer
	logger   zerolog.Logger
}

func NewCTEMConsumer(engine *ctem.CTEMEngine, consumer *events.Consumer, logger zerolog.Logger) *CTEMConsumer {
	c := &CTEMConsumer{
		engine:   engine,
		consumer: consumer,
		logger:   logger.With().Str("component", "ctem-consumer").Logger(),
	}
	consumer.Subscribe(events.Topics.CtemEvents, events.EventHandlerFunc(c.handleCTEMEvent))
	return c
}

func (c *CTEMConsumer) Start(ctx context.Context) error {
	return c.consumer.Start(ctx)
}

func (c *CTEMConsumer) Stop() error {
	return c.consumer.Stop()
}

func (c *CTEMConsumer) handleCTEMEvent(ctx context.Context, event *events.Event) error {
	switch event.Type {
	case "cyber.ctem.assessment.run_requested", "com.clario360.cyber.ctem.assessment.run_requested":
		var payload struct {
			AssessmentID string `json:"assessment_id"`
		}
		if err := event.Unmarshal(&payload); err != nil {
			return err
		}
		assessmentID, err := uuid.Parse(payload.AssessmentID)
		if err != nil {
			return err
		}
		return c.engine.RunAssessment(ctx, assessmentID)
	default:
		return nil
	}
}
