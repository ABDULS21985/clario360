package cti

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

	var data json.RawMessage
	if event.Data != nil {
		data = event.Data
	} else {
		data = json.RawMessage("{}")
	}

	c.hub.Broadcast(tenantID, event.Type, data)
	return nil
}

// ---------------------------------------------------------------------------
// AlertNotificationConsumer bridges CTI alerts to the notification service.
// Subscribes to: cyber.cti.alerts
// ---------------------------------------------------------------------------

// NotificationSender abstracts the notification-service HTTP API.
type NotificationSender interface {
	SendNotification(ctx context.Context, req NotificationRequest) error
}

// NotificationRequest maps to the notification-service's create-notification endpoint.
type NotificationRequest struct {
	TenantID string            `json:"tenant_id"`
	Type     string            `json:"type"`
	Title    string            `json:"title"`
	Body     string            `json:"body"`
	Priority string            `json:"priority"`
	Channel  string            `json:"channel"`
	Category string            `json:"category"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type AlertNotificationConsumer struct {
	sender NotificationSender
	logger zerolog.Logger
}

func NewAlertNotificationConsumer(sender NotificationSender, logger zerolog.Logger) *AlertNotificationConsumer {
	return &AlertNotificationConsumer{
		sender: sender,
		logger: logger.With().Str("component", "cti-alert-notification").Logger(),
	}
}

func (c *AlertNotificationConsumer) Handle(ctx context.Context, event *events.Event) error {
	var alert AlertPayload
	if err := json.Unmarshal(event.Data, &alert); err != nil {
		c.logger.Warn().Err(err).Msg("unmarshal alert payload")
		return nil // don't retry malformed payloads
	}

	channel := "websocket"
	priority := "high"
	if alert.SeverityCode == "critical" {
		channel = "email,websocket"
		priority = "urgent"
	}

	req := NotificationRequest{
		TenantID: alert.TenantID,
		Type:     "cyber_threat_intelligence",
		Title:    alert.Title,
		Body:     alert.Description,
		Priority: priority,
		Channel:  channel,
		Category: "cyber_threat_intelligence",
		Metadata: map[string]string{
			"alert_type":    alert.AlertType,
			"source_entity": alert.SourceEntity,
			"source_id":     alert.SourceID,
			"severity":      alert.SeverityCode,
			"action_url":    alert.ActionURL,
		},
	}

	if err := c.sender.SendNotification(ctx, req); err != nil {
		c.logger.Error().Err(err).
			Str("tenant_id", alert.TenantID).
			Str("alert_type", alert.AlertType).
			Msg("failed to send CTI alert notification")
		return err // retry via DLQ
	}

	c.logger.Info().
		Str("tenant_id", alert.TenantID).
		Str("alert_type", alert.AlertType).
		Str("severity", alert.SeverityCode).
		Msg("CTI alert notification sent")
	return nil
}

// ---------------------------------------------------------------------------
// HTTPNotificationSender sends notifications via HTTP to the notification-service.
// ---------------------------------------------------------------------------

type HTTPNotificationSender struct {
	baseURL string
	client  *http.Client
	logger  zerolog.Logger
}

func NewHTTPNotificationSender(notificationServiceURL string, logger zerolog.Logger) *HTTPNotificationSender {
	return &HTTPNotificationSender{
		baseURL: strings.TrimRight(notificationServiceURL, "/"),
		client:  &http.Client{Timeout: 10 * time.Second},
		logger:  logger,
	}
}

func (s *HTTPNotificationSender) SendNotification(ctx context.Context, req NotificationRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}

	url := s.baseURL + "/api/v1/notifications/internal"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Internal-Service", "cyber-service")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("notification-service returned %d", resp.StatusCode)
	}
	return nil
}
