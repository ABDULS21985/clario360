package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/connector"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/events"
)

const (
	accessEventsTopic             = "data.access_events"
	defaultAccessCollectionWindow = 24 * time.Hour
	defaultFailureWindow          = 24 * time.Hour
	defaultCollectionInterval     = 15 * time.Minute
)

type eventPublisher interface {
	Publish(ctx context.Context, topic string, event *events.Event) error
}

type configDecryptor interface {
	Decrypt(ciphertext []byte) ([]byte, error)
}

type sourceRepository interface {
	ListActive(ctx context.Context, tenantID uuid.UUID) ([]*repository.SourceRecord, error)
	ListActiveTenants(ctx context.Context) ([]uuid.UUID, error)
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status model.DataSourceStatus, lastError *string) error
}

type connectorRegistry interface {
	CreateWithSourceContext(sourceType model.DataSourceType, configJSON json.RawMessage, sourceID, tenantID uuid.UUID) (connector.Connector, error)
}

type CollectorMetrics struct {
	CyclesTotal         *prometheus.CounterVec
	SourcesTotal        *prometheus.CounterVec
	EventsCollected     *prometheus.CounterVec
	ErrorsTotal         *prometheus.CounterVec
	DegradedSources     prometheus.Counter
	LastCollectionGauge *prometheus.GaugeVec
}

func newCollectorMetrics(registerer prometheus.Registerer) *CollectorMetrics {
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}
	metrics := &CollectorMetrics{
		CyclesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "data_access_log_collection_cycles_total",
			Help: "Number of access-log collection cycles by outcome.",
		}, []string{"result"}),
		SourcesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "data_access_log_sources_total",
			Help: "Number of data sources processed by source type and outcome.",
		}, []string{"source_type", "result"}),
		EventsCollected: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "data_access_log_events_collected_total",
			Help: "Number of data access events collected per source type.",
		}, []string{"source_type"}),
		ErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "data_access_log_collection_errors_total",
			Help: "Number of access-log collection errors by source type and operation.",
		}, []string{"source_type", "operation"}),
		DegradedSources: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "data_access_log_sources_degraded_total",
			Help: "Number of sources marked degraded after repeated collection failures.",
		}),
		LastCollectionGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "data_access_log_last_collection_timestamp",
			Help: "Unix timestamp of the last successful access-log collection per source.",
		}, []string{"source_id"}),
	}
	registerer.MustRegister(
		metrics.CyclesTotal,
		metrics.SourcesTotal,
		metrics.EventsCollected,
		metrics.ErrorsTotal,
		metrics.DegradedSources,
		metrics.LastCollectionGauge,
	)
	return metrics
}

type AccessLogCollector struct {
	connectorRegistry connectorRegistry
	sourceRepo        sourceRepository
	decryptor         configDecryptor
	redis             *redis.Client
	eventProducer     eventPublisher
	logger            zerolog.Logger
	metrics           *CollectorMetrics
}

func NewAccessLogCollector(
	registry connectorRegistry,
	sourceRepo sourceRepository,
	decryptor configDecryptor,
	rdb *redis.Client,
	eventProducer eventPublisher,
	logger zerolog.Logger,
	registerer prometheus.Registerer,
) *AccessLogCollector {
	return &AccessLogCollector{
		connectorRegistry: registry,
		sourceRepo:        sourceRepo,
		decryptor:         decryptor,
		redis:             rdb,
		eventProducer:     eventProducer,
		logger:            logger.With().Str("component", "data-access-log-collector").Logger(),
		metrics:           newCollectorMetrics(registerer),
	}
}

func (c *AccessLogCollector) CollectAll(ctx context.Context, tenantID uuid.UUID) ([]connector.DataAccessEvent, error) {
	records, err := c.sourceRepo.ListActive(ctx, tenantID)
	if err != nil {
		c.metrics.CyclesTotal.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("list active sources for tenant %s: %w", tenantID, err)
	}

	collected := make([]connector.DataAccessEvent, 0)
	var errs []error
	for _, record := range records {
		if record == nil || record.Source == nil {
			continue
		}
		eventsForSource, sourceErr := c.collectForSource(ctx, record)
		if sourceErr != nil {
			errs = append(errs, sourceErr)
			continue
		}
		collected = append(collected, eventsForSource...)
	}

	if len(errs) > 0 {
		c.metrics.CyclesTotal.WithLabelValues("partial").Inc()
		return collected, errors.Join(errs...)
	}
	c.metrics.CyclesTotal.WithLabelValues("success").Inc()
	return collected, nil
}

func (c *AccessLogCollector) CollectAllTenants(ctx context.Context) error {
	tenantIDs, err := c.sourceRepo.ListActiveTenants(ctx)
	if err != nil {
		c.metrics.CyclesTotal.WithLabelValues("error").Inc()
		return fmt.Errorf("list active tenants for access-log collection: %w", err)
	}
	var errs []error
	for _, tenantID := range tenantIDs {
		if _, collectErr := c.CollectAll(ctx, tenantID); collectErr != nil {
			errs = append(errs, collectErr)
		}
	}
	return errors.Join(errs...)
}

func (c *AccessLogCollector) Run(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = defaultCollectionInterval
	}
	if err := c.CollectAllTenants(ctx); err != nil && !errors.Is(err, context.Canceled) {
		c.logger.Warn().Err(err).Msg("initial access-log collection completed with errors")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := c.CollectAllTenants(ctx); err != nil && !errors.Is(err, context.Canceled) {
				c.logger.Warn().Err(err).Msg("scheduled access-log collection completed with errors")
			}
		}
	}
}

func (c *AccessLogCollector) collectForSource(ctx context.Context, record *repository.SourceRecord) ([]connector.DataAccessEvent, error) {
	source := record.Source
	if source == nil {
		return nil, nil
	}

	configJSON, err := c.decryptor.Decrypt(record.EncryptedConfig)
	if err != nil {
		c.metrics.ErrorsTotal.WithLabelValues(string(source.Type), "decrypt").Inc()
		return nil, c.handleSourceFailure(ctx, source, fmt.Errorf("decrypt source config: %w", err))
	}

	instance, err := c.connectorRegistry.CreateWithSourceContext(source.Type, configJSON, source.ID, source.TenantID)
	if err != nil {
		c.metrics.ErrorsTotal.WithLabelValues(string(source.Type), "create").Inc()
		return nil, c.handleSourceFailure(ctx, source, fmt.Errorf("instantiate connector: %w", err))
	}
	defer func() {
		if closeErr := instance.Close(); closeErr != nil {
			c.logger.Warn().Err(closeErr).Str("source_id", source.ID.String()).Msg("close access-log connector")
		}
	}()

	securityConnector, ok := instance.(connector.SecurityAwareConnector)
	if !ok {
		c.metrics.SourcesTotal.WithLabelValues(string(source.Type), "skipped").Inc()
		return nil, nil
	}

	since, err := c.lastCollectionTime(ctx, source.ID)
	if err != nil {
		c.metrics.ErrorsTotal.WithLabelValues(string(source.Type), "cursor").Inc()
		return nil, c.handleSourceFailure(ctx, source, fmt.Errorf("load last collection time: %w", err))
	}

	eventsForSource, err := securityConnector.QueryAccessLogs(ctx, since)
	if err != nil {
		c.metrics.ErrorsTotal.WithLabelValues(string(source.Type), "query").Inc()
		return nil, c.handleSourceFailure(ctx, source, fmt.Errorf("query access logs: %w", err))
	}
	c.metrics.SourcesTotal.WithLabelValues(string(source.Type), "success").Inc()
	c.metrics.EventsCollected.WithLabelValues(string(source.Type)).Add(float64(len(eventsForSource)))

	now := time.Now().UTC()
	if err := c.publishEvents(ctx, source, eventsForSource); err != nil {
		c.metrics.ErrorsTotal.WithLabelValues(string(source.Type), "publish").Inc()
		return eventsForSource, c.handleSourceFailure(ctx, source, fmt.Errorf("publish access-log events: %w", err))
	}
	if err := c.storeLastCollectionTime(ctx, source.ID, now); err != nil {
		c.metrics.ErrorsTotal.WithLabelValues(string(source.Type), "cursor").Inc()
		return eventsForSource, c.handleSourceFailure(ctx, source, fmt.Errorf("store last collection time: %w", err))
	}
	if err := c.clearFailureCount(ctx, source.ID); err != nil {
		c.logger.Warn().Err(err).Str("source_id", source.ID.String()).Msg("clear access-log failure count")
	}
	c.metrics.LastCollectionGauge.WithLabelValues(source.ID.String()).Set(float64(now.Unix()))
	return eventsForSource, nil
}

func (c *AccessLogCollector) publishEvents(ctx context.Context, source *model.DataSource, collected []connector.DataAccessEvent) error {
	if c.eventProducer == nil || len(collected) == 0 {
		return nil
	}
	for _, accessEvent := range collected {
		payload := map[string]any{
			"timestamp":     accessEvent.Timestamp,
			"user":          accessEvent.User,
			"source_ip":     accessEvent.SourceIP,
			"action":        accessEvent.Action,
			"database":      accessEvent.Database,
			"table":         accessEvent.Table,
			"query_hash":    accessEvent.QueryHash,
			"query_preview": accessEvent.QueryPreview,
			"rows_read":     accessEvent.RowsRead,
			"rows_written":  accessEvent.RowsWritten,
			"bytes_read":    accessEvent.BytesRead,
			"bytes_written": accessEvent.BytesWritten,
			"duration_ms":   accessEvent.DurationMs,
			"success":       accessEvent.Success,
			"error_message": accessEvent.ErrorMsg,
			"source_type":   accessEvent.SourceType,
			"source_id":     accessEvent.SourceID,
			"tenant_id":     accessEvent.TenantID,
			"source_name":   source.Name,
		}
		event, err := events.NewEvent("data.access.event.collected", "data-service", source.TenantID.String(), payload)
		if err != nil {
			return fmt.Errorf("create access event envelope: %w", err)
		}
		if err := c.eventProducer.Publish(ctx, accessEventsTopic, event); err != nil {
			return fmt.Errorf("publish %s for source %s: %w", accessEventsTopic, source.ID, err)
		}
	}
	return nil
}

func (c *AccessLogCollector) handleSourceFailure(ctx context.Context, source *model.DataSource, err error) error {
	if source == nil {
		return err
	}
	c.metrics.SourcesTotal.WithLabelValues(string(source.Type), "error").Inc()
	c.logger.Warn().
		Err(err).
		Str("source_id", source.ID.String()).
		Str("source_type", string(source.Type)).
		Str("tenant_id", source.TenantID.String()).
		Msg("access-log collection failed for source")

	if c.redis == nil {
		return err
	}
	key := failureCountKey(source.ID)
	failures, incrErr := c.redis.Incr(ctx, key).Result()
	if incrErr != nil {
		return errors.Join(err, fmt.Errorf("increment access-log failure count: %w", incrErr))
	}
	_ = c.redis.Expire(ctx, key, defaultFailureWindow).Err()

	if failures < 5 {
		return err
	}

	message := truncateError(err.Error(), 512)
	if updateErr := c.sourceRepo.UpdateStatus(ctx, source.TenantID, source.ID, model.DataSourceStatusError, &message); updateErr != nil {
		err = errors.Join(err, fmt.Errorf("mark source degraded: %w", updateErr))
	}
	c.metrics.DegradedSources.Inc()

	if c.eventProducer != nil {
		degradedEvent, eventErr := events.NewEvent("data.source.degraded", "data-service", source.TenantID.String(), map[string]any{
			"id":            source.ID,
			"name":          source.Name,
			"type":          source.Type,
			"tenant_id":     source.TenantID,
			"failure_count": failures,
			"error":         message,
		})
		if eventErr == nil {
			if publishErr := c.eventProducer.Publish(ctx, events.Topics.DataSourceEvents, degradedEvent); publishErr != nil {
				err = errors.Join(err, fmt.Errorf("publish degraded source event: %w", publishErr))
			}
		}
	}
	return err
}

func (c *AccessLogCollector) lastCollectionTime(ctx context.Context, sourceID uuid.UUID) (time.Time, error) {
	if c.redis == nil {
		return time.Now().UTC().Add(-defaultAccessCollectionWindow), nil
	}
	value, err := c.redis.Get(ctx, lastCollectionKey(sourceID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return time.Now().UTC().Add(-defaultAccessCollectionWindow), nil
		}
		return time.Time{}, err
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse last collection time %q: %w", value, err)
	}
	return parsed.UTC(), nil
}

func (c *AccessLogCollector) storeLastCollectionTime(ctx context.Context, sourceID uuid.UUID, timestamp time.Time) error {
	if c.redis == nil {
		return nil
	}
	return c.redis.Set(ctx, lastCollectionKey(sourceID), timestamp.UTC().Format(time.RFC3339Nano), 30*24*time.Hour).Err()
}

func (c *AccessLogCollector) clearFailureCount(ctx context.Context, sourceID uuid.UUID) error {
	if c.redis == nil {
		return nil
	}
	return c.redis.Del(ctx, failureCountKey(sourceID)).Err()
}

func lastCollectionKey(sourceID uuid.UUID) string {
	return "access_log:last:" + sourceID.String()
}

func failureCountKey(sourceID uuid.UUID) string {
	return "access_log:failures:" + sourceID.String()
}

func truncateError(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit]
}
