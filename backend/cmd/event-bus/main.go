package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

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

	shutdownTracer, err := observability.InitTracer(ctx, "event-bus", cfg.Observability.OTLPEndpoint)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to initialize tracer")
	} else {
		defer shutdownTracer(ctx)
	}

	consumer, err := events.NewConsumer(cfg.Kafka, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create kafka consumer")
	}
	defer consumer.Close()

	// Register event handlers
	registry := events.NewHandlerRegistry()

	registry.RegisterFunc(events.TopicAuditLog, func(ctx context.Context, event *events.Event) error {
		logger.Info().
			Str("event_id", event.ID).
			Str("tenant_id", event.TenantID).
			Msg("audit event received")
		return nil
	})

	registry.RegisterFunc(events.TopicUserCreated, func(ctx context.Context, event *events.Event) error {
		logger.Info().
			Str("event_id", event.ID).
			Str("type", event.Type).
			Msg("user created event received")
		return nil
	})

	// Subscribe to topics
	consumer.Subscribe(events.TopicAuditLog, registry)
	consumer.Subscribe(events.TopicUserCreated, registry)

	// Handle shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		logger.Info().Msg("shutting down event-bus")
		cancel()
	}()

	logger.Info().Msg("event-bus starting")
	if err := consumer.Start(ctx); err != nil && ctx.Err() == nil {
		logger.Fatal().Err(err).Msg("consumer failed")
		os.Exit(1)
	}

	logger.Info().Msg("event-bus stopped")
}
