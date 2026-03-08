package consumer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	visuskpi "github.com/clario360/platform/internal/visus/kpi"
	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/repository"
	"github.com/clario360/platform/internal/visus/service"
)

const visusConsumerName = "visus_consumer"

type executiveAlertCreator interface {
	Create(ctx context.Context, alert *model.ExecutiveAlert) (*model.ExecutiveAlert, error)
}

type kpiDefinitionStore interface {
	ListEnabled(ctx context.Context, tenantID uuid.UUID) ([]model.KPIDefinition, error)
	UpdateSnapshotState(ctx context.Context, tenantID, id uuid.UUID, at time.Time, value float64, status model.KPIStatus) error
}

type kpiSnapshotStore interface {
	Create(ctx context.Context, item *model.KPISnapshot) (*model.KPISnapshot, error)
	LatestByKPI(ctx context.Context, tenantID, kpiID uuid.UUID) (*model.KPISnapshot, error)
}

type VisusConsumer struct {
	alerts    executiveAlertCreator
	kpis      kpiDefinitionStore
	snapshots kpiSnapshotStore
	redis     *redis.Client
	metrics   *events.CrossSuiteMetrics
	logger    zerolog.Logger
	threshold *visuskpi.ThresholdEvaluator
	now       func() time.Time
}

func NewVisusConsumer(logger zerolog.Logger) *VisusConsumer {
	return &VisusConsumer{
		logger:    logger.With().Str("component", visusConsumerName).Logger(),
		threshold: visuskpi.NewThresholdEvaluator(),
		now:       time.Now,
	}
}

func (c *VisusConsumer) WithDependencies(alerts *service.AlertService, kpis *repository.KPIRepository, snapshots *repository.KPISnapshotRepository, redisClient *redis.Client) *VisusConsumer {
	c.alerts = alerts
	c.kpis = kpis
	c.snapshots = snapshots
	c.redis = redisClient
	return c
}

func (c *VisusConsumer) WithMetrics(metrics *events.CrossSuiteMetrics) *VisusConsumer {
	c.metrics = metrics
	return c
}

func (c *VisusConsumer) Register(consumer *events.Consumer) {
	if consumer == nil {
		return
	}
	for _, topic := range []string{
		events.Topics.AlertEvents,
		events.Topics.RiskEvents,
		events.Topics.CtemEvents,
		events.Topics.PipelineEvents,
		events.Topics.QualityEvents,
		events.Topics.ContradictionEvents,
		events.Topics.LineageEvents,
		events.Topics.ActaEvents,
		events.Topics.LexEvents,
		events.Topics.FileEvents,
	} {
		consumer.Subscribe(topic, c)
	}
}

func (c *VisusConsumer) EventTypes() []string {
	return []string{
		"com.clario360.cyber.alert.created",
		"com.clario360.cyber.risk.score.updated",
		"com.clario360.cyber.risk.score_calculated",
		"com.clario360.cyber.ctem.assessment.completed",
		"com.clario360.data.pipeline.consecutive_failures",
		"com.clario360.data.pipeline.critical_reliability",
		"com.clario360.data.quality.score_changed",
		"com.clario360.data.contradiction.detected",
		"com.clario360.data.lineage.graph_updated",
		"com.clario360.acta.compliance.checked",
		"com.clario360.acta.action_item.overdue",
		"com.clario360.lex.contract.expiring",
		"com.clario360.enterprise.lex.contract.expiring",
		"com.clario360.lex.compliance.alert.created",
		"com.clario360.lex.compliance.alert_created",
		"com.clario360.file.scan.infected",
	}
}

func (c *VisusConsumer) Handle(ctx context.Context, event *events.Event) error {
	if c.alerts == nil || event == nil {
		return nil
	}

	switch event.Type {
	case "com.clario360.cyber.alert.created":
		return c.handleAlertCreated(ctx, event)
	case "com.clario360.cyber.risk.score.updated", "com.clario360.cyber.risk.score_calculated":
		return c.handleRiskScoreUpdated(ctx, event)
	case "com.clario360.cyber.ctem.assessment.completed":
		return c.handleCTEMCompleted(ctx, event)
	case "com.clario360.data.pipeline.consecutive_failures":
		return c.handleConsecutiveFailures(ctx, event)
	case "com.clario360.data.pipeline.critical_reliability":
		return c.handleCriticalReliability(ctx, event)
	case "com.clario360.data.quality.score_changed":
		return c.handleQualityScoreChanged(ctx, event)
	case "com.clario360.data.contradiction.detected":
		return c.handleContradictionDetected(ctx, event)
	case "com.clario360.data.lineage.graph_updated":
		return c.handleLineageUpdated(ctx, event)
	case "com.clario360.acta.compliance.checked":
		return c.handleComplianceChecked(ctx, event)
	case "com.clario360.acta.action_item.overdue":
		return c.handleActionItemOverdue(ctx, event)
	case "com.clario360.lex.contract.expiring", "com.clario360.enterprise.lex.contract.expiring":
		return c.handleContractExpiring(ctx, event)
	case "com.clario360.lex.compliance.alert.created", "com.clario360.lex.compliance.alert_created":
		return c.handleComplianceAlert(ctx, event)
	case "com.clario360.file.scan.infected":
		return c.handleMalwareDetected(ctx, event)
	default:
		return nil
	}
}

func (c *VisusConsumer) tenantID(event *events.Event) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(event.TenantID))
}

func (c *VisusConsumer) createExecutiveAlert(
	ctx context.Context,
	tenantID uuid.UUID,
	title string,
	description string,
	category model.AlertCategory,
	severity model.AlertSeverity,
	sourceSuite string,
	sourceType string,
	dedupKey string,
	metadata map[string]any,
) error {
	alert := &model.ExecutiveAlert{
		TenantID:        tenantID,
		Title:           title,
		Description:     description,
		Category:        category,
		Severity:        severity,
		SourceSuite:     sourceSuite,
		SourceType:      sourceType,
		Status:          model.AlertStatusNew,
		OccurrenceCount: 1,
		FirstSeenAt:     c.now().UTC(),
		LastSeenAt:      c.now().UTC(),
		Metadata:        metadata,
	}
	if strings.TrimSpace(dedupKey) != "" {
		alert.DedupKey = &dedupKey
	}
	if _, err := c.alerts.Create(ctx, alert); err != nil {
		return err
	}
	if c.metrics != nil {
		c.metrics.AlertsCreatedTotal.WithLabelValues(visusConsumerName, string(severity)).Inc()
	}
	return nil
}

func (c *VisusConsumer) updateKPIByName(ctx context.Context, tenantID uuid.UUID, name string, value float64) (*model.KPIDefinition, model.KPIStatus, error) {
	if c.kpis == nil || c.snapshots == nil {
		return nil, model.KPIStatusUnknown, nil
	}

	definition, err := c.findKPIByName(ctx, tenantID, name)
	if err != nil || definition == nil {
		return definition, model.KPIStatusUnknown, err
	}

	now := c.now().UTC()
	var previous *model.KPISnapshot
	if latest, latestErr := c.snapshots.LatestByKPI(ctx, tenantID, definition.ID); latestErr == nil {
		previous = latest
	} else if latestErr != nil && !errors.Is(latestErr, repository.ErrNotFound) {
		return nil, model.KPIStatusUnknown, latestErr
	}

	var previousValue *float64
	var delta *float64
	var deltaPercent *float64
	if previous != nil {
		p := previous.Value
		d := value - previous.Value
		previousValue = &p
		delta = &d
		if previous.Value != 0 {
			dp := (d / previous.Value) * 100
			deltaPercent = &dp
		}
	}

	status := c.threshold.Evaluate(definition, value)
	snapshot := &model.KPISnapshot{
		TenantID:      tenantID,
		KPIID:         definition.ID,
		Value:         value,
		PreviousValue: previousValue,
		Delta:         delta,
		DeltaPercent:  deltaPercent,
		Status:        status,
		PeriodStart:   now,
		PeriodEnd:     now,
		FetchSuccess:  true,
		CreatedAt:     now,
	}
	if _, err := c.snapshots.Create(ctx, snapshot); err != nil {
		return nil, model.KPIStatusUnknown, err
	}
	if err := c.kpis.UpdateSnapshotState(ctx, tenantID, definition.ID, now, value, status); err != nil {
		return nil, model.KPIStatusUnknown, err
	}
	if c.metrics != nil {
		c.metrics.KPIUpdatesTotal.WithLabelValues(visusConsumerName, name).Inc()
	}
	return definition, status, nil
}

func (c *VisusConsumer) incrementKPIByName(ctx context.Context, tenantID uuid.UUID, name string, amount float64) (*model.KPIDefinition, model.KPIStatus, float64, error) {
	definition, err := c.findKPIByName(ctx, tenantID, name)
	if err != nil || definition == nil {
		return definition, model.KPIStatusUnknown, 0, err
	}

	current := 0.0
	if c.snapshots != nil {
		if latest, latestErr := c.snapshots.LatestByKPI(ctx, tenantID, definition.ID); latestErr == nil {
			current = latest.Value
		} else if latestErr != nil && !errors.Is(latestErr, repository.ErrNotFound) {
			return nil, model.KPIStatusUnknown, 0, latestErr
		}
	}

	_, status, err := c.updateKPIByName(ctx, tenantID, name, current+amount)
	return definition, status, current + amount, err
}

func (c *VisusConsumer) findKPIByName(ctx context.Context, tenantID uuid.UUID, name string) (*model.KPIDefinition, error) {
	if c.kpis == nil {
		return nil, nil
	}
	items, err := c.kpis.ListEnabled(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	for idx := range items {
		if strings.EqualFold(items[idx].Name, name) {
			return &items[idx], nil
		}
	}
	return nil, nil
}

func (c *VisusConsumer) invalidateKeys(ctx context.Context, keys ...string) error {
	if c.redis == nil || len(keys) == 0 {
		return nil
	}
	return c.redis.Del(ctx, keys...).Err()
}

func severityFromString(raw string) model.AlertSeverity {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "critical":
		return model.AlertSeverityCritical
	case "high", "warning":
		return model.AlertSeverityHigh
	case "medium":
		return model.AlertSeverityMedium
	case "low":
		return model.AlertSeverityLow
	default:
		return model.AlertSeverityInfo
	}
}

func kpiBreachSeverity(status model.KPIStatus) model.AlertSeverity {
	switch status {
	case model.KPIStatusCritical:
		return model.AlertSeverityCritical
	case model.KPIStatusWarning:
		return model.AlertSeverityHigh
	default:
		return model.AlertSeverityInfo
	}
}

func boolValue(value bool, whenTrue, whenFalse string) string {
	if value {
		return whenTrue
	}
	return whenFalse
}

func dedupKey(prefix string, parts ...string) string {
	values := make([]string, 0, len(parts)+1)
	values = append(values, prefix)
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return strings.Join(values, ":")
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}
