package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/cyber/service"
	"github.com/clario360/platform/internal/events"
)

const (
	dataConsumerName             = "cyber_data_consumer"
	dataConnectionFailureSource  = "data_source_connection_failed"
	dspmSyncDebounceWindow       = 10 * time.Minute
	dataConnectionDedupWindow    = 30 * time.Minute
	cyberDataSystemActorEmail    = "cyber-data-consumer@system.local"
	cyberDataSystemActorName     = "cyber-data-consumer"
	cyberDataSystemActorUserUUID = "22222222-2222-4222-8222-222222222222"
)

type dspmTriggerService interface {
	TriggerScan(ctx context.Context, tenantID, userID uuid.UUID, actor *service.Actor) (*model.DSPMScan, error)
}

type DataEventConsumer struct {
	alertService alertEventService
	dspmService  dspmTriggerService
	redis        *redis.Client
	guard        *events.IdempotencyGuard
	producer     *events.Producer
	logger       zerolog.Logger
	metrics      *events.CrossSuiteMetrics
	now          func() time.Time
}

func NewDataEventConsumer(alertService alertEventService, dspmService dspmTriggerService, redisClient *redis.Client, guard *events.IdempotencyGuard, producer *events.Producer, logger zerolog.Logger, metrics *events.CrossSuiteMetrics) *DataEventConsumer {
	return &DataEventConsumer{
		alertService: alertService,
		dspmService:  dspmService,
		redis:        redisClient,
		guard:        guard,
		producer:     producer,
		logger:       logger.With().Str("component", dataConsumerName).Logger(),
		metrics:      metrics,
		now:          time.Now,
	}
}

func (c *DataEventConsumer) EventTypes() []string {
	return []string{
		"com.clario360.data.source.connection_tested",
		"com.clario360.data.darkdata.scan_completed",
	}
}

func (c *DataEventConsumer) Handle(ctx context.Context, event *events.Event) error {
	switch event.Type {
	case "com.clario360.data.source.connection_tested":
		return c.handleConnectionTested(ctx, event)
	case "com.clario360.data.darkdata.scan_completed":
		return c.handleDarkDataScanCompleted(ctx, event)
	default:
		return nil
	}
}

type dataSourceConnectionEvent struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Success   bool   `json:"success"`
	LatencyMS int64  `json:"latency_ms"`
	Error     string `json:"error"`
	Message   string `json:"message"`
}

func (c *DataEventConsumer) handleConnectionTested(ctx context.Context, event *events.Event) error {
	var payload dataSourceConnectionEvent
	if err := event.Unmarshal(&payload); err != nil {
		c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("malformed event data")
		return nil
	}
	if strings.TrimSpace(payload.ID) == "" {
		c.logger.Warn().Str("event_id", event.ID).Msg("missing required field: id")
		return nil
	}

	tenantID, err := uuid.Parse(strings.TrimSpace(event.TenantID))
	if err != nil {
		c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("invalid tenant id")
		return nil
	}

	processed, err := c.guard.IsProcessed(ctx, event.ID)
	if err != nil {
		return err
	}
	if processed {
		c.recordIdempotentSkip(event.Type)
		return nil
	}

	if payload.Success {
		return c.guard.MarkProcessed(ctx, event.ID)
	}

	alert, err := c.candidateConnectionFailureAlert(tenantID, payload, event.ID)
	if err != nil {
		_ = c.guard.Release(ctx, event.ID)
		return err
	}

	existing, err := c.alertService.FindRecentEventAlert(ctx, tenantID, dataConnectionFailureSource, "source_id", payload.ID, dataConnectionDedupWindow)
	switch {
	case err == nil:
		alert.ID = existing.ID
		alert.FirstEventAt = existing.FirstEventAt
		if _, updateErr := c.alertService.UpdateEventAlert(ctx, alert); updateErr != nil {
			_ = c.guard.Release(ctx, event.ID)
			return updateErr
		}
	case errors.Is(err, repository.ErrNotFound):
		if _, createErr := c.alertService.CreateFromEvent(ctx, alert); createErr != nil {
			_ = c.guard.Release(ctx, event.ID)
			return createErr
		}
	default:
		_ = c.guard.Release(ctx, event.ID)
		return err
	}

	if c.metrics != nil {
		c.metrics.AlertsCreatedTotal.WithLabelValues(dataConsumerName, string(model.SeverityLow)).Inc()
	}
	return c.guard.MarkProcessed(ctx, event.ID)
}

func (c *DataEventConsumer) candidateConnectionFailureAlert(tenantID uuid.UUID, payload dataSourceConnectionEvent, eventID string) (*model.Alert, error) {
	explanation := model.AlertExplanation{
		Summary: "A data source connectivity check failed, which can interrupt ingestion and downstream analytics.",
		ConfidenceFactors: []model.ConfidenceFactor{
			{Factor: "Direct connection test failure", Impact: 0.45},
			{Factor: "Potential ingestion disruption", Impact: 0.2},
		},
		RecommendedActions: []string{
			"Validate the source credentials and endpoint reachability",
			"Confirm the source system is available",
			"Review recent infrastructure or networking changes",
		},
		Details: map[string]any{
			"source_id":   payload.ID,
			"source_type": payload.Type,
			"latency_ms":  payload.LatencyMS,
			"error":       fallbackString(payload.Error, payload.Message),
			"event_id":    eventID,
		},
	}

	metadata, err := json.Marshal(map[string]any{
		"source_id":     payload.ID,
		"source_name":   payload.Name,
		"source_type":   payload.Type,
		"latency_ms":    payload.LatencyMS,
		"error":         fallbackString(payload.Error, payload.Message),
		"last_event_id": eventID,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal connection failure alert metadata: %w", err)
	}

	sourceLabel := fallbackString(payload.Name, payload.ID)
	now := c.now().UTC()
	return &model.Alert{
		TenantID:        tenantID,
		Title:           fmt.Sprintf("Data Source Connection Failure - %s", sourceLabel),
		Description:     fmt.Sprintf("Connectivity validation failed for data source %s.", sourceLabel),
		Severity:        model.SeverityLow,
		Status:          model.AlertStatusNew,
		Source:          dataConnectionFailureSource,
		Explanation:     explanation,
		ConfidenceScore: 0.7,
		EventCount:      1,
		FirstEventAt:    now,
		LastEventAt:     now,
		Metadata:        metadata,
	}, nil
}

type darkDataScanCompletedEvent struct {
	ScanID           string `json:"scan_id"`
	AssetsDiscovered int    `json:"assets_discovered"`
	PIIFound         int    `json:"pii_found"`
	HighRisk         int    `json:"high_risk"`
}

func (c *DataEventConsumer) handleDarkDataScanCompleted(ctx context.Context, event *events.Event) error {
	var payload darkDataScanCompletedEvent
	if err := event.Unmarshal(&payload); err != nil {
		c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("malformed event data")
		return nil
	}
	if strings.TrimSpace(payload.ScanID) == "" {
		c.logger.Warn().Str("event_id", event.ID).Msg("missing required field: scan_id")
		return nil
	}

	tenantID, err := uuid.Parse(strings.TrimSpace(event.TenantID))
	if err != nil {
		c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("invalid tenant id")
		return nil
	}

	processed, err := c.guard.IsProcessed(ctx, event.ID)
	if err != nil {
		return err
	}
	if processed {
		c.recordIdempotentSkip(event.Type)
		return nil
	}

	if c.dspmService == nil {
		return c.guard.MarkProcessed(ctx, event.ID)
	}

	debounceKey := fmt.Sprintf("cyber:dspm_sync:%s", event.TenantID)
	allowed, err := c.acquireDSPMSyncWindow(ctx, debounceKey)
	if err != nil {
		_ = c.guard.Release(ctx, event.ID)
		return err
	}
	if !allowed {
		return c.guard.MarkProcessed(ctx, event.ID)
	}

	systemUserID := uuid.MustParse(cyberDataSystemActorUserUUID)
	actor := &service.Actor{
		UserID:    systemUserID,
		UserName:  cyberDataSystemActorName,
		UserEmail: cyberDataSystemActorEmail,
	}
	if _, err := c.dspmService.TriggerScan(ctx, tenantID, systemUserID, actor); err != nil {
		_ = c.guard.Release(ctx, event.ID)
		_ = c.releaseDSPMSyncWindow(ctx, debounceKey)
		return err
	}

	return c.guard.MarkProcessed(ctx, event.ID)
}

func (c *DataEventConsumer) acquireDSPMSyncWindow(ctx context.Context, key string) (bool, error) {
	if c.redis == nil {
		return true, nil
	}
	created, err := c.redis.SetNX(ctx, key, "1", dspmSyncDebounceWindow).Result()
	if err != nil {
		return false, fmt.Errorf("acquire dspm sync debounce window: %w", err)
	}
	return created, nil
}

func (c *DataEventConsumer) releaseDSPMSyncWindow(ctx context.Context, key string) error {
	if c.redis == nil {
		return nil
	}
	return c.redis.Del(ctx, key).Err()
}

func (c *DataEventConsumer) recordIdempotentSkip(eventType string) {
	if c.metrics == nil {
		return
	}
	c.metrics.SkippedIdempotentTotal.WithLabelValues(dataConsumerName, eventType).Inc()
	c.metrics.ProcessedTotal.WithLabelValues(dataConsumerName, "data", eventType, "skipped").Inc()
}
