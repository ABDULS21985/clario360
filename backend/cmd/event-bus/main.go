package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"

	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/observability"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("loading config: " + err.Error())
	}

	logger := observability.NewLogger(
		cfg.Observability.LogLevel,
		cfg.Observability.LogFormat,
		"event-bus",
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize tracing
	shutdownTracer, err := observability.InitTracer(ctx, "event-bus", cfg.Observability.OTLPEndpoint)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to initialize tracer")
	} else {
		defer shutdownTracer(ctx)
	}

	// Redis client for idempotency store
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	// Kafka producer (for DLQ and replay)
	producer, err := events.NewProducer(cfg.Kafka, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create kafka producer")
	}
	defer producer.Close()

	// Consumer group manager
	manager, err := events.NewConsumerGroupManager(cfg.Kafka, "event-bus", logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create consumer group manager")
	}

	// Middleware setup
	idempotencyStore := events.NewIdempotencyStore(rdb, 0)
	consumerMetrics := events.NewEventConsumerMetrics("clario360")

	// Dead letter consumer
	dlqStore := events.NewDeadLetterStore()
	dlqConsumer := events.NewDeadLetterConsumer(dlqStore, producer, logger)

	// Build middleware chain for event handlers
	middlewareChain := func(handler events.EventHandler) events.EventHandler {
		return events.ApplyMiddleware(handler,
			events.WithTracing(observability.Tracer("event-bus")),
			events.WithLogging(logger),
			events.WithMetrics(consumerMetrics),
			events.WithIdempotency(idempotencyStore, "event-bus"),
			events.WithDeadLetter(producer, logger),
			events.WithRetry(3, events.ExponentialBackoff(100_000_000, 2_000_000_000)), // 100ms base, 2s max
		)
	}

	// Event handler registry
	registry := events.NewHandlerRegistry()

	// Register handlers for domain events
	registry.RegisterFunc("com.clario360.user.registered", func(ctx context.Context, event *events.Event) error {
		logger.Info().
			Str("event_id", event.ID).
			Str("tenant_id", event.TenantID).
			Str("user_id", event.UserID).
			Msg("user registered event received")
		return nil
	})

	registry.RegisterFunc("com.clario360.user.login.success", func(ctx context.Context, event *events.Event) error {
		logger.Info().
			Str("event_id", event.ID).
			Str("tenant_id", event.TenantID).
			Str("user_id", event.UserID).
			Msg("user login success event received")
		return nil
	})

	registry.RegisterFunc("com.clario360.user.login.failed", func(ctx context.Context, event *events.Event) error {
		logger.Warn().
			Str("event_id", event.ID).
			Str("tenant_id", event.TenantID).
			Msg("user login failed event received")
		return nil
	})

	// Wrap registry with middleware chain
	wrappedHandler := middlewareChain(registry)

	// Subscribe to all platform topics
	allTopics := []string{
		events.Topics.IAMEvents,
		events.Topics.AuditEvents,
		events.Topics.NotificationEvents,
		events.Topics.WorkflowEvents,
		events.Topics.AssetEvents,
		events.Topics.ThreatEvents,
		events.Topics.AlertEvents,
		events.Topics.RemediationEvents,
		events.Topics.DataSourceEvents,
		events.Topics.PipelineEvents,
		events.Topics.QualityEvents,
		events.Topics.ContradictionEvents,
		events.Topics.LineageEvents,
		events.Topics.ActaEvents,
		events.Topics.LexEvents,
		events.Topics.VisusEvents,
	}

	for _, topic := range allTopics {
		manager.Subscribe(topic, wrappedHandler)
	}

	// Subscribe DLQ consumer to dead letter topic
	manager.Subscribe(events.Topics.DeadLetter, events.EventHandlerFunc(dlqConsumer.Handle))

	// Start consumer group manager
	if err := manager.Start(ctx); err != nil {
		logger.Fatal().Err(err).Msg("failed to start consumer group manager")
	}

	// Health checker
	healthChecker := events.NewHealthChecker(cfg.Kafka, logger)

	// HTTP server for health and DLQ API
	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		status := healthChecker.Check(r.Context(), allTopics)
		w.Header().Set("Content-Type", "application/json")
		if status.Status == "unhealthy" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(w).Encode(status)
	})

	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		health := manager.Health()
		w.Header().Set("Content-Type", "application/json")
		if health.Status != "healthy" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(w).Encode(health)
	})

	// Dead letter queue API
	r.Route("/api/v1/events/dead-letter", func(r chi.Router) {
		r.Get("/", dlqListHandler(dlqStore))
		r.Get("/{id}", dlqGetHandler(dlqStore))
		r.Post("/{id}/replay", dlqReplayHandler(dlqConsumer))
		r.Delete("/{id}", dlqDeleteHandler(dlqStore))
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		logger.Info().Str("addr", addr).Msg("event-bus HTTP server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	// Handle shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	logger.Info().Msg("shutting down event-bus")

	cancel()
	_ = manager.Stop()
	_ = srv.Shutdown(context.Background())

	logger.Info().Msg("event-bus stopped")
}

func dlqListHandler(store *events.DeadLetterStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		status := r.URL.Query().Get("status")
		entries := store.List(tenantID, status, 100, 0)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"entries": entries,
			"total":   store.Count(),
		})
	}
}

func dlqGetHandler(store *events.DeadLetterStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		entry, ok := store.Get(id)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "entry not found"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(entry)
	}
}

func dlqReplayHandler(consumer *events.DeadLetterConsumer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := consumer.Replay(r.Context(), id); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "replayed"})
	}
}

func dlqDeleteHandler(store *events.DeadLetterStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if !store.Delete(id) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "entry not found"})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
