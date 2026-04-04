package alert

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/repository"
)

type Publisher interface {
	Publish(ctx context.Context, topic string, event *events.Event) error
}

type Generator struct {
	alerts       *repository.AlertRepository
	deduplicator *Deduplicator
	publisher    Publisher
	logger       zerolog.Logger
}

func NewGenerator(alerts *repository.AlertRepository, deduplicator *Deduplicator, publisher Publisher, logger zerolog.Logger) *Generator {
	return &Generator{
		alerts:       alerts,
		deduplicator: deduplicator,
		publisher:    publisher,
		logger:       logger.With().Str("component", "visus_alert_generator").Logger(),
	}
}

func (g *Generator) GenerateFromKPI(ctx context.Context, tenantID uuid.UUID, kpi *model.KPIDefinition, value float64, status model.KPIStatus) error {
	if kpi == nil {
		return nil
	}
	dedupKey := fmt.Sprintf("kpi_breach:%s", kpi.ID.String())
	if _, matched, err := g.deduplicator.CheckAndUpdate(ctx, tenantID, dedupKey, time.Hour); err != nil {
		return err
	} else if matched {
		return nil
	}

	threshold := 0.0
	if status == model.KPIStatusCritical && kpi.CriticalThreshold != nil {
		threshold = *kpi.CriticalThreshold
	}
	if status == model.KPIStatusWarning && kpi.WarningThreshold != nil {
		threshold = *kpi.WarningThreshold
	}
	alert := &model.ExecutiveAlert{
		TenantID:        tenantID,
		Title:           fmt.Sprintf("%s is %s", kpi.Name, status),
		Description:     fmt.Sprintf("%s reached %.2f %s, exceeding the %s threshold of %.2f.", kpi.Name, value, kpi.Unit, status, threshold),
		Category:        categoryFromSuite(kpi.Suite),
		Severity:        severityFromStatus(status),
		SourceSuite:     string(kpi.Suite),
		SourceType:      "kpi_breach",
		SourceEntityID:  &kpi.ID,
		Status:          model.AlertStatusNew,
		DedupKey:        &dedupKey,
		OccurrenceCount: 1,
		FirstSeenAt:     time.Now().UTC(),
		LastSeenAt:      time.Now().UTC(),
		LinkedKPIID:     &kpi.ID,
		Metadata: map[string]any{
			"kpi_name":  kpi.Name,
			"kpi_value": value,
			"status":    status,
			"suite":     kpi.Suite,
		},
	}
	created, err := g.alerts.Create(ctx, alert)
	if err != nil {
		return err
	}
	return g.publishAlert(ctx, created)
}

func (g *Generator) CreateAlert(ctx context.Context, alert *model.ExecutiveAlert) (*model.ExecutiveAlert, error) {
	if alert == nil {
		return nil, repository.ErrValidation
	}
	if alert.Status == "" {
		alert.Status = model.AlertStatusNew
	}
	if alert.FirstSeenAt.IsZero() {
		alert.FirstSeenAt = time.Now().UTC()
	}
	if alert.LastSeenAt.IsZero() {
		alert.LastSeenAt = alert.FirstSeenAt
	}
	if alert.OccurrenceCount == 0 {
		alert.OccurrenceCount = 1
	}
	if alert.Metadata == nil {
		alert.Metadata = map[string]any{}
	}
	if alert.DedupKey != nil {
		if _, matched, err := g.deduplicator.CheckAndUpdate(ctx, alert.TenantID, *alert.DedupKey, time.Hour); err != nil {
			return nil, err
		} else if matched {
			return alert, nil
		}
	}
	created, err := g.alerts.Create(ctx, alert)
	if err != nil {
		return nil, err
	}
	return created, g.publishAlert(ctx, created)
}

func (g *Generator) publishAlert(ctx context.Context, alert *model.ExecutiveAlert) error {
	if g.publisher == nil || alert == nil {
		return nil
	}
	event, err := events.NewEvent("visus.alert.created", "visus-service", alert.TenantID.String(), map[string]any{
		"id":           alert.ID,
		"title":        alert.Title,
		"category":     alert.Category,
		"severity":     alert.Severity,
		"source_suite": alert.SourceSuite,
	})
	if err != nil {
		return err
	}
	return g.publisher.Publish(ctx, events.Topics.VisusEvents, event)
}

func categoryFromSuite(suite model.KPISuite) model.AlertCategory {
	switch suite {
	case model.KPISuiteCyber:
		return model.AlertCategoryRisk
	case model.KPISuiteData:
		return model.AlertCategoryDataQuality
	case model.KPISuiteActa:
		return model.AlertCategoryGovernance
	case model.KPISuiteLex:
		return model.AlertCategoryLegal
	default:
		return model.AlertCategoryOperational
	}
}

func severityFromStatus(status model.KPIStatus) model.AlertSeverity {
	switch status {
	case model.KPIStatusCritical:
		return model.AlertSeverityCritical
	case model.KPIStatusWarning:
		return model.AlertSeverityHigh
	default:
		return model.AlertSeverityInfo
	}
}
