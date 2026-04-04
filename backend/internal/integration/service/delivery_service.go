package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	intmetrics "github.com/clario360/platform/internal/integration/metrics"
	"github.com/clario360/platform/internal/integration/encryption"
	intmodel "github.com/clario360/platform/internal/integration/model"
	intrepo "github.com/clario360/platform/internal/integration/repository"
	"github.com/clario360/platform/internal/integration/service/jira"
	"github.com/clario360/platform/internal/integration/service/servicenow"
	"github.com/clario360/platform/internal/integration/service/slack"
	"github.com/clario360/platform/internal/integration/service/teams"
	"github.com/clario360/platform/internal/integration/service/webhook"
)

type DeliveryService struct {
	deliveryRepo    *intrepo.DeliveryRepository
	integrationRepo *intrepo.IntegrationRepository
	encryptor       *encryption.ConfigEncryptor
	redis           *redis.Client
	logger          zerolog.Logger
	slackClient     *slack.Client
	teamsClient     *teams.Client
	jiraClient      *jira.Service
	snClient        *servicenow.Service
	webhookClient   *webhook.Client
}

func NewDeliveryService(
	deliveryRepo *intrepo.DeliveryRepository,
	integrationRepo *intrepo.IntegrationRepository,
	encryptor *encryption.ConfigEncryptor,
	rdb *redis.Client,
	slackClient *slack.Client,
	teamsClient *teams.Client,
	jiraClient *jira.Service,
	snClient *servicenow.Service,
	webhookClient *webhook.Client,
	logger zerolog.Logger,
) *DeliveryService {
	return &DeliveryService{
		deliveryRepo:    deliveryRepo,
		integrationRepo: integrationRepo,
		encryptor:       encryptor,
		redis:           rdb,
		slackClient:     slackClient,
		teamsClient:     teamsClient,
		jiraClient:      jiraClient,
		snClient:        snClient,
		webhookClient:   webhookClient,
		logger:          logger.With().Str("component", "integration_delivery_service").Logger(),
	}
}

func (s *DeliveryService) QueueEvent(ctx context.Context, integration *intmodel.Integration, event *events.Event) (string, error) {
	record := &intmodel.DeliveryRecord{
		TenantID:      integration.TenantID,
		IntegrationID: integration.ID,
		EventType:     event.Type,
		EventID:       event.ID,
		EventData:     truncateJSON(event.Data, 5*1024),
		Status:        intmodel.DeliveryStatusPending,
		MaxAttempts:   4,
	}

	if s.redis != nil {
		key := "integration_rate:" + integration.ID
		count, err := s.redis.Incr(ctx, key).Result()
		if err == nil {
			_ = s.redis.Expire(ctx, key, time.Minute).Err()
			intmetrics.IntegrationDeliveryRate.WithLabelValues(integration.ID).Set(float64(count))
			if count > 100 {
				nextRetryAt := time.Now().UTC().Add(time.Minute)
				record.Status = intmodel.DeliveryStatusRetrying
				record.NextRetryAt = &nextRetryAt
			}
		}
	}

	id, err := s.deliveryRepo.Create(ctx, record)
	if err != nil {
		return "", err
	}
	intmetrics.IntegrationDeliveryQueueSize.WithLabelValues(string(integration.Type)).Inc()
	return id, nil
}

func (s *DeliveryService) Process(ctx context.Context, record *intmodel.DeliveryRecord) error {
	integration, err := s.integrationRepo.GetByID(ctx, record.TenantID, record.IntegrationID)
	if err != nil {
		return err
	}
	if integration.Status != intmodel.IntegrationStatusActive {
		errText := "integration disabled"
		return s.deliveryRepo.MarkFailed(ctx, record.ID, errText, "integration", nil, nil)
	}

	config, err := s.encryptor.Decrypt(integration.ConfigEncrypted, integration.ConfigNonce)
	if err != nil {
		return s.handleFailure(ctx, integration, record, err, nil, nil)
	}

	event := &events.Event{
		ID:       record.EventID,
		Type:     record.EventType,
		TenantID: record.TenantID,
		Data:     record.EventData,
		Time:     record.CreatedAt,
	}

	start := time.Now()
	var (
		responseCode int
		responseBody string
	)

	switch integration.Type {
	case intmodel.IntegrationTypeSlack:
		var cfg intmodel.SlackConfig
		if err := DecodeInto(config, &cfg); err != nil {
			return s.handleFailure(ctx, integration, record, err, nil, nil)
		}
		responseCode, responseBody, err = s.slackClient.Send(ctx, cfg, event)
	case intmodel.IntegrationTypeTeams:
		var cfg intmodel.TeamsConfig
		if err := DecodeInto(config, &cfg); err != nil {
			return s.handleFailure(ctx, integration, record, err, nil, nil)
		}
		responseCode, responseBody, err = s.teamsClient.Send(ctx, cfg, event)
	case intmodel.IntegrationTypeWebhook:
		var cfg intmodel.WebhookConfig
		if err := DecodeInto(config, &cfg); err != nil {
			return s.handleFailure(ctx, integration, record, err, nil, nil)
		}
		responseCode, responseBody, err = s.webhookClient.Post(ctx, cfg, event)
	case intmodel.IntegrationTypeJira:
		var cfg intmodel.JiraConfig
		if err := DecodeInto(config, &cfg); err != nil {
			return s.handleFailure(ctx, integration, record, err, nil, nil)
		}
		entityID := extractEventEntityID(record.EventData)
		if entityID == "" {
			err = fmt.Errorf("jira delivery requires event payload id")
			break
		}
		systemToken, tokenErr := s.jiraClient.ClarioToken(ctx, integration.TenantID)
		if tokenErr != nil {
			err = tokenErr
			break
		}
		_, responseCode, responseBody, err = s.jiraClient.CreateFromEntity(ctx, integration, cfg, systemToken, "alert", entityID)
	case intmodel.IntegrationTypeServiceNow:
		var cfg intmodel.ServiceNowConfig
		if err := DecodeInto(config, &cfg); err != nil {
			return s.handleFailure(ctx, integration, record, err, nil, nil)
		}
		entityID := extractEventEntityID(record.EventData)
		if entityID == "" {
			err = fmt.Errorf("servicenow delivery requires event payload id")
			break
		}
		systemToken, tokenErr := s.snClient.ClarioToken(ctx, integration.TenantID)
		if tokenErr != nil {
			err = tokenErr
			break
		}
		_, responseCode, responseBody, err = s.snClient.CreateFromEntity(ctx, integration, cfg, systemToken, "alert", entityID)
	default:
		err = fmt.Errorf("unsupported integration type %q", integration.Type)
	}

	latencyMS := int(time.Since(start).Milliseconds())
	if err == nil {
		if responseCode == 0 {
			responseCode = 200
		}
		if err := s.deliveryRepo.MarkDelivered(ctx, record.ID, responseCode, responseBody, latencyMS); err != nil {
			return err
		}
		_ = s.integrationRepo.RecordSuccess(ctx, integration.ID)
		intmetrics.IntegrationDeliveriesTotal.WithLabelValues(string(integration.Type), "delivered").Inc()
		intmetrics.IntegrationDeliveryLatencySeconds.WithLabelValues(string(integration.Type)).Observe(float64(latencyMS) / 1000)
		intmetrics.IntegrationDeliveryQueueSize.WithLabelValues(string(integration.Type)).Dec()
		return nil
	}

	codePtr := optionalInt(responseCode)
	bodyPtr := optionalString(responseBody)
	return s.handleFailure(ctx, integration, record, err, codePtr, bodyPtr)
}

func (s *DeliveryService) handleFailure(ctx context.Context, integration *intmodel.Integration, record *intmodel.DeliveryRecord, deliveryErr error, responseCode *int, responseBody *string) error {
	category := categorizeError(deliveryErr, responseCode)
	if record.Attempts+1 < record.MaxAttempts {
		nextRetryAt := time.Now().UTC().Add(webhook.BackoffForAttempt(record.Attempts + 1))
		if err := s.deliveryRepo.ScheduleRetry(ctx, record.ID, nextRetryAt, deliveryErr.Error(), category, responseCode, responseBody); err != nil {
			return err
		}
		intmetrics.IntegrationDeliveryRetriesTotal.WithLabelValues(string(integration.Type)).Inc()
		intmetrics.IntegrationErrorsTotal.WithLabelValues(string(integration.Type), category).Inc()
		return nil
	}

	if err := s.deliveryRepo.MarkFailed(ctx, record.ID, deliveryErr.Error(), category, responseCode, responseBody); err != nil {
		return err
	}
	count, err := s.integrationRepo.RecordFailure(ctx, integration.ID, deliveryErr.Error())
	if err != nil {
		return err
	}
	intmetrics.IntegrationDeliveriesTotal.WithLabelValues(string(integration.Type), "failed").Inc()
	intmetrics.IntegrationErrorsTotal.WithLabelValues(string(integration.Type), category).Inc()
	if count >= 10 {
		_ = s.integrationRepo.SetAutoDisabled(ctx, integration.ID)
		intmetrics.IntegrationErrorThresholdTriggersTotal.WithLabelValues(string(integration.Type)).Inc()
	}
	return nil
}

func categorizeError(err error, responseCode *int) string {
	if responseCode != nil {
		switch {
		case *responseCode == 401 || *responseCode == 403:
			return "auth"
		case *responseCode == 429:
			return "rate_limit"
		case *responseCode >= 500:
			return "server"
		case *responseCode >= 400:
			return "payload"
		}
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return "network"
	}
	if strings.Contains(strings.ToLower(err.Error()), "timeout") {
		return "network"
	}
	return "payload"
}

func truncateJSON(data []byte, limit int) []byte {
	if len(data) <= limit {
		return data
	}
	return data[:limit]
}

func optionalInt(value int) *int {
	if value == 0 {
		return nil
	}
	return &value
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func extractEventEntityID(raw []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	if value, ok := payload["id"].(string); ok {
		return value
	}
	return ""
}
