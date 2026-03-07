package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/service"
)

type VisusConsumer struct {
	alerts *service.AlertService
	logger zerolog.Logger
}

func NewVisusConsumer(logger zerolog.Logger) *VisusConsumer {
	return &VisusConsumer{logger: logger.With().Str("component", "visus_consumer").Logger()}
}

func (c *VisusConsumer) WithAlertService(alerts *service.AlertService) *VisusConsumer {
	c.alerts = alerts
	return c
}

func (c *VisusConsumer) Register(consumer *events.Consumer) {
	if consumer == nil {
		return
	}
	consumer.Subscribe(events.Topics.AlertEvents, events.EventHandlerFunc(c.Handle))
	consumer.Subscribe(events.Topics.PipelineEvents, events.EventHandlerFunc(c.Handle))
	consumer.Subscribe(events.Topics.QualityEvents, events.EventHandlerFunc(c.Handle))
	consumer.Subscribe(events.Topics.ActaEvents, events.EventHandlerFunc(c.Handle))
	consumer.Subscribe(events.Topics.LexEvents, events.EventHandlerFunc(c.Handle))
}

func (c *VisusConsumer) Handle(ctx context.Context, event *events.Event) error {
	if c.alerts == nil || event == nil {
		return nil
	}
	tenantID, err := uuid.Parse(event.TenantID)
	if err != nil {
		return err
	}
	payload := map[string]any{}
	_ = json.Unmarshal(event.Data, &payload)

	switch {
	case strings.HasPrefix(event.Type, "com.clario360.cyber") && strings.Contains(strings.ToLower(event.Type), "alert"):
		return c.create(ctx, tenantID, "Critical security event", "A critical cybersecurity event was propagated to the executive console.", model.AlertCategoryRisk, model.AlertSeverityCritical, "cyber", "event", payload)
	case strings.HasPrefix(event.Type, "com.clario360.data") && strings.Contains(strings.ToLower(event.Type), "pipeline"):
		return c.create(ctx, tenantID, "Pipeline failure event", "A data pipeline failure was propagated to the executive console.", model.AlertCategoryOperational, model.AlertSeverityHigh, "data", "event", payload)
	case strings.HasPrefix(event.Type, "com.clario360.data") && strings.Contains(strings.ToLower(event.Type), "quality"):
		return c.create(ctx, tenantID, "Data quality event", "A data quality degradation event was propagated to the executive console.", model.AlertCategoryDataQuality, model.AlertSeverityHigh, "data", "event", payload)
	case strings.HasPrefix(event.Type, "com.clario360.enterprise.acta") || strings.Contains(event.Source, "acta"):
		return c.create(ctx, tenantID, "Governance event", "A governance event was propagated to the executive console.", model.AlertCategoryGovernance, model.AlertSeverityHigh, "acta", "event", payload)
	case strings.HasPrefix(event.Type, "com.clario360.enterprise.lex") || strings.Contains(event.Source, "lex"):
		return c.create(ctx, tenantID, "Legal event", "A legal/compliance event was propagated to the executive console.", model.AlertCategoryLegal, model.AlertSeverityHigh, "lex", "event", payload)
	default:
		return nil
	}
}

func (c *VisusConsumer) create(ctx context.Context, tenantID uuid.UUID, title, description string, category model.AlertCategory, severity model.AlertSeverity, sourceSuite, sourceType string, payload map[string]any) error {
	dedupKey := fmt.Sprintf("%s:%s:%s", sourceSuite, sourceType, time.Now().UTC().Format("2006010215"))
	_, err := c.alerts.Create(ctx, &model.ExecutiveAlert{
		TenantID:        tenantID,
		Title:           title,
		Description:     description,
		Category:        category,
		Severity:        severity,
		SourceSuite:     sourceSuite,
		SourceType:      sourceType,
		DedupKey:        &dedupKey,
		OccurrenceCount: 1,
		FirstSeenAt:     time.Now().UTC(),
		LastSeenAt:      time.Now().UTC(),
		Metadata:        payload,
	})
	return err
}
