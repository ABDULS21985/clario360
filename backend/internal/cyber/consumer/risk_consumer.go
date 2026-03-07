package consumer

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/service"
	"github.com/clario360/platform/internal/events"
)

type RiskConsumer struct {
	riskSvc      *service.RiskService
	dashboardSvc *service.DashboardService
	rdb          *redis.Client
	consumer     *events.Consumer
	logger       zerolog.Logger
}

func NewRiskConsumer(
	riskSvc *service.RiskService,
	dashboardSvc *service.DashboardService,
	rdb *redis.Client,
	consumer *events.Consumer,
	logger zerolog.Logger,
) *RiskConsumer {
	c := &RiskConsumer{
		riskSvc:      riskSvc,
		dashboardSvc: dashboardSvc,
		rdb:          rdb,
		consumer:     consumer,
		logger:       logger.With().Str("component", "risk-consumer").Logger(),
	}
	consumer.Subscribe(events.Topics.AlertEvents, events.EventHandlerFunc(c.handle))
	consumer.Subscribe(events.Topics.VulnerabilityEvents, events.EventHandlerFunc(c.handle))
	consumer.Subscribe(events.Topics.ThreatEvents, events.EventHandlerFunc(c.handle))
	consumer.Subscribe(events.Topics.CtemEvents, events.EventHandlerFunc(c.handle))
	return c
}

func (c *RiskConsumer) handle(ctx context.Context, event *events.Event) error {
	tenantID, err := uuid.Parse(event.TenantID)
	if err != nil {
		return err
	}

	if err := c.dashboardSvc.InvalidateCache(ctx, tenantID); err != nil {
		c.logger.Error().Err(err).Str("tenant_id", tenantID.String()).Msg("invalidate dashboard cache")
	}
	if err := c.riskSvc.InvalidateCache(ctx, tenantID); err != nil {
		c.logger.Error().Err(err).Str("tenant_id", tenantID.String()).Msg("invalidate risk cache")
	}

	eventType := strings.TrimPrefix(event.Type, "com.clario360.")
	if !c.isSignificant(eventType, event.Data) {
		return nil
	}
	if !c.acquireDebounce(ctx, tenantID) {
		return nil
	}

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if _, err := c.riskSvc.SaveEventTriggeredSnapshot(bgCtx, tenantID, eventType); err != nil {
			c.logger.Error().Err(err).Str("tenant_id", tenantID.String()).Str("event_type", eventType).Msg("event-triggered risk snapshot failed")
		}
	}()
	return nil
}

func (c *RiskConsumer) isSignificant(eventType string, payload []byte) bool {
	switch eventType {
	case "cyber.alert.resolved", "cyber.ctem.assessment.completed":
		return true
	case "cyber.alert.created", "cyber.vulnerability.created":
		var data struct {
			Severity string `json:"severity"`
		}
		if len(payload) > 0 && jsonUnmarshal(payload, &data) == nil {
			return data.Severity == "critical" || data.Severity == "high"
		}
		return false
	case "cyber.vulnerability.updated":
		var data struct {
			NewStatus string `json:"new_status"`
		}
		if len(payload) > 0 && jsonUnmarshal(payload, &data) == nil {
			switch data.NewStatus {
			case "resolved", "mitigated", "accepted", "false_positive":
				return true
			}
		}
		return false
	default:
		return false
	}
}

func (c *RiskConsumer) acquireDebounce(ctx context.Context, tenantID uuid.UUID) bool {
	if c.rdb == nil {
		return true
	}
	ok, err := c.rdb.SetNX(ctx, "cyber:risk_recalc:"+tenantID.String(), "1", 5*time.Minute).Result()
	if err != nil {
		c.logger.Error().Err(err).Str("tenant_id", tenantID.String()).Msg("risk debounce setnx failed")
		return true
	}
	return ok
}

func jsonUnmarshal(payload []byte, target interface{}) error {
	return json.Unmarshal(payload, target)
}
