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
	"github.com/clario360/platform/internal/events"
)

const (
	iamConsumerName       = "cyber_iam_consumer"
	bruteForceSource      = "iam_brute_force"
	mfaDowngradeSource    = "iam_mfa_disabled"
	bruteForceWindow      = 5 * time.Minute
	bruteForceDedupWindow = 30 * time.Minute
)

type alertEventService interface {
	CreateFromEvent(ctx context.Context, alert *model.Alert) (*model.Alert, error)
	FindRecentEventAlert(ctx context.Context, tenantID uuid.UUID, source, metadataKey, metadataValue string, window time.Duration) (*model.Alert, error)
	UpdateEventAlert(ctx context.Context, alert *model.Alert) (*model.Alert, error)
}

type IAMEventConsumer struct {
	alertService alertEventService
	redis        *redis.Client
	guard        *events.IdempotencyGuard
	producer     *events.Producer
	logger       zerolog.Logger
	metrics      *events.CrossSuiteMetrics
	now          func() time.Time
}

func NewIAMEventConsumer(alertService alertEventService, redisClient *redis.Client, guard *events.IdempotencyGuard, producer *events.Producer, logger zerolog.Logger, metrics *events.CrossSuiteMetrics) *IAMEventConsumer {
	return &IAMEventConsumer{
		alertService: alertService,
		redis:        redisClient,
		guard:        guard,
		producer:     producer,
		logger:       logger.With().Str("component", iamConsumerName).Logger(),
		metrics:      metrics,
		now:          time.Now,
	}
}

func (c *IAMEventConsumer) EventTypes() []string {
	return []string{
		"com.clario360.iam.user.login.failed",
		"com.clario360.iam.user.mfa.disabled",
	}
}

func (c *IAMEventConsumer) Handle(ctx context.Context, event *events.Event) error {
	switch event.Type {
	case "com.clario360.iam.user.login.failed":
		return c.handleBruteForceDetection(ctx, event)
	case "com.clario360.iam.user.mfa.disabled":
		return c.handleMFADowngrade(ctx, event)
	default:
		return nil
	}
}

type loginFailedEvent struct {
	UserID       string    `json:"user_id"`
	Email        string    `json:"email"`
	IPAddress    string    `json:"ip_address"`
	AttemptCount int       `json:"attempt_count"`
	UserAgent    string    `json:"user_agent"`
	Timestamp    time.Time `json:"timestamp"`
	Reason       string    `json:"reason"`
}

func (c *IAMEventConsumer) handleBruteForceDetection(ctx context.Context, event *events.Event) error {
	var payload loginFailedEvent
	if err := event.Unmarshal(&payload); err != nil {
		c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("malformed event data")
		return nil
	}
	if strings.TrimSpace(payload.IPAddress) == "" {
		c.logger.Warn().Str("event_id", event.ID).Msg("missing required field: ip_address")
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

	now := c.now().UTC()
	if payload.Timestamp.IsZero() {
		payload.Timestamp = event.Time
	}
	if payload.Timestamp.IsZero() {
		payload.Timestamp = now
	}

	count, err := c.recordBruteForceAttempt(ctx, event.TenantID, payload.IPAddress, event.ID, payload.Timestamp)
	if err != nil {
		_ = c.guard.Release(ctx, event.ID)
		return err
	}

	if count >= 5 {
		if err := c.upsertBruteForceAlert(ctx, tenantID, payload, count, event.ID); err != nil {
			_ = c.guard.Release(ctx, event.ID)
			return err
		}
	}

	return c.guard.MarkProcessed(ctx, event.ID)
}

func (c *IAMEventConsumer) recordBruteForceAttempt(ctx context.Context, tenantID, ipAddress, eventID string, timestamp time.Time) (int64, error) {
	if c.redis == nil {
		return 0, fmt.Errorf("redis client is required")
	}

	key := fmt.Sprintf("brute_force:%s:%s", tenantID, ipAddress)
	now := c.now().UTC()
	windowStart := now.Add(-bruteForceWindow).UnixMilli()
	score := float64(timestamp.UTC().UnixMilli())

	pipe := c.redis.TxPipeline()
	pipe.ZAdd(ctx, key, redis.Z{Score: score, Member: eventID})
	pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", windowStart))
	countCmd := pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, 10*time.Minute)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("record brute force attempt: %w", err)
	}

	return countCmd.Val(), nil
}

func (c *IAMEventConsumer) upsertBruteForceAlert(ctx context.Context, tenantID uuid.UUID, payload loginFailedEvent, count int64, eventID string) error {
	alert, err := c.candidateBruteForceAlert(tenantID, payload, count, eventID)
	if err != nil {
		return err
	}

	existing, err := c.alertService.FindRecentEventAlert(ctx, tenantID, bruteForceSource, "ip_address", payload.IPAddress, bruteForceDedupWindow)
	switch {
	case err == nil:
		alert.ID = existing.ID
		alert.FirstEventAt = existing.FirstEventAt
		if _, updateErr := c.alertService.UpdateEventAlert(ctx, alert); updateErr != nil {
			return updateErr
		}
		return nil
	case errors.Is(err, repository.ErrNotFound):
		_, createErr := c.alertService.CreateFromEvent(ctx, alert)
		return createErr
	default:
		return err
	}
}

func (c *IAMEventConsumer) candidateBruteForceAlert(tenantID uuid.UUID, payload loginFailedEvent, count int64, eventID string) (*model.Alert, error) {
	severity := model.SeverityHigh
	recommendedActions := []string{
		"Investigate the source IP address",
		"Consider blocking the IP if not recognized",
		"Verify the targeted user account has strong credentials",
		"Check if the IP appears in threat intelligence databases",
	}
	if count >= 20 {
		severity = model.SeverityCritical
		recommendedActions = append(recommendedActions, fmt.Sprintf("Immediately block %s at the firewall level.", payload.IPAddress))
	}

	explanation := model.AlertExplanation{
		Summary: "Multiple failed login attempts detected from a single IP address.",
		ConfidenceFactors: []model.ConfidenceFactor{
			{Factor: "5+ attempts in 5 minutes", Impact: 0.3},
			{Factor: "Single IP source", Impact: 0.2},
			{Factor: "Known attack pattern", Impact: 0.2},
			{Factor: "Automated behavior pattern", Impact: 0.15},
		},
		RecommendedActions: recommendedActions,
		Details: map[string]interface{}{
			"matched_conditions": []map[string]string{
				{"field": "ip_address", "operator": "eq", "value": payload.IPAddress},
				{"field": "attempt_count", "operator": "gte", "value": "5"},
				{"field": "window", "operator": "within", "value": "5 minutes"},
			},
			"event_id":      eventID,
			"window_count":  count,
			"email":         payload.Email,
			"reason":        payload.Reason,
			"attempt_count": payload.AttemptCount,
		},
	}

	metadata, err := json.Marshal(map[string]interface{}{
		"ip_address":    payload.IPAddress,
		"email":         payload.Email,
		"user_id":       payload.UserID,
		"user_agent":    payload.UserAgent,
		"reason":        payload.Reason,
		"last_event_id": eventID,
		"window_count":  count,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal brute force alert metadata: %w", err)
	}

	title := fmt.Sprintf("Brute Force Attack Detected - %d failed login attempts from %s", count, payload.IPAddress)
	description := fmt.Sprintf("User %s experienced %d failed login attempts from IP %s within 5 minutes.", fallbackString(payload.Email, payload.UserID), count, payload.IPAddress)
	techniqueID := "T1110"
	techniqueName := "Brute Force"
	tacticID := "TA0006"
	tacticName := "Credential Access"

	return &model.Alert{
		TenantID:           tenantID,
		Title:              title,
		Description:        description,
		Severity:           severity,
		Status:             model.AlertStatusNew,
		Source:             bruteForceSource,
		Explanation:        explanation,
		ConfidenceScore:    0.85,
		MITRETechniqueID:   &techniqueID,
		MITRETechniqueName: &techniqueName,
		MITRETacticID:      &tacticID,
		MITRETacticName:    &tacticName,
		EventCount:         int(count),
		FirstEventAt:       payload.Timestamp.UTC(),
		LastEventAt:        c.now().UTC(),
		Metadata:           metadata,
	}, nil
}

type mfaDisabledEvent struct {
	UserID     string    `json:"user_id"`
	Email      string    `json:"email"`
	DisabledBy string    `json:"disabled_by"`
	Reason     string    `json:"reason"`
	Timestamp  time.Time `json:"timestamp"`
}

func (c *IAMEventConsumer) handleMFADowngrade(ctx context.Context, event *events.Event) error {
	var payload mfaDisabledEvent
	if err := event.Unmarshal(&payload); err != nil {
		c.logger.Warn().Err(err).Str("event_id", event.ID).Msg("malformed event data")
		return nil
	}
	if strings.TrimSpace(payload.Email) == "" && strings.TrimSpace(payload.UserID) == "" {
		c.logger.Warn().Str("event_id", event.ID).Msg("missing required field: email")
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

	now := c.now().UTC()
	if payload.Timestamp.IsZero() {
		payload.Timestamp = event.Time
	}
	if payload.Timestamp.IsZero() {
		payload.Timestamp = now
	}

	explanation := model.AlertExplanation{
		Summary: "Two-factor authentication was disabled for a user account, reducing security posture.",
		ConfidenceFactors: []model.ConfidenceFactor{
			{Factor: "Direct system observation", Impact: 0.5},
		},
		RecommendedActions: []string{
			"Verify the MFA disable request was legitimate",
			"Contact the user to confirm they intended to disable MFA",
			"If unauthorized: re-enable MFA and investigate",
		},
		Details: map[string]interface{}{
			"disabled_by": payload.DisabledBy,
			"reason":      payload.Reason,
		},
	}

	metadata, err := json.Marshal(map[string]interface{}{
		"user_id":     payload.UserID,
		"email":       payload.Email,
		"disabled_by": payload.DisabledBy,
		"reason":      payload.Reason,
		"event_id":    event.ID,
	})
	if err != nil {
		_ = c.guard.Release(ctx, event.ID)
		return fmt.Errorf("marshal mfa alert metadata: %w", err)
	}

	techniqueID := "T1556.006"
	techniqueName := "Multi-Factor Authentication"
	alert := &model.Alert{
		TenantID:           tenantID,
		Title:              fmt.Sprintf("MFA Disabled for %s", fallbackString(payload.Email, payload.UserID)),
		Description:        "Multi-factor authentication was disabled for a user account.",
		Severity:           model.SeverityMedium,
		Status:             model.AlertStatusNew,
		Source:             mfaDowngradeSource,
		Explanation:        explanation,
		ConfidenceScore:    1.0,
		MITRETechniqueID:   &techniqueID,
		MITRETechniqueName: &techniqueName,
		EventCount:         1,
		FirstEventAt:       payload.Timestamp.UTC(),
		LastEventAt:        payload.Timestamp.UTC(),
		Metadata:           metadata,
	}

	if _, err := c.alertService.CreateFromEvent(ctx, alert); err != nil {
		_ = c.guard.Release(ctx, event.ID)
		return err
	}
	return c.guard.MarkProcessed(ctx, event.ID)
}

func (c *IAMEventConsumer) recordIdempotentSkip(eventType string) {
	if c.metrics == nil {
		return
	}
	c.metrics.SkippedIdempotentTotal.WithLabelValues(iamConsumerName, eventType).Inc()
	c.metrics.ProcessedTotal.WithLabelValues(iamConsumerName, "iam", eventType, "skipped").Inc()
}

func fallbackString(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
}
