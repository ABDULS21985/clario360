package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"

	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/middleware"
	"github.com/clario360/platform/internal/observability"
	"github.com/clario360/platform/internal/server"

	wfcfg "github.com/clario360/platform/internal/workflow/config"
	"github.com/clario360/platform/internal/workflow/consumer"
	"github.com/clario360/platform/internal/workflow/executor"
	"github.com/clario360/platform/internal/workflow/handler"
	"github.com/clario360/platform/internal/workflow/health"
	_ "github.com/clario360/platform/internal/workflow/metrics" // registers Prometheus metrics on import
	"github.com/clario360/platform/internal/workflow/repository"
	"github.com/clario360/platform/internal/workflow/service"
)

func main() {
	// 1. Load platform config
	cfg, err := config.Load()
	if err != nil {
		panic("loading config: " + err.Error())
	}

	// Load workflow-specific config from env
	wfCfg := wfcfg.LoadFromEnv()
	if err := wfCfg.Validate(); err != nil {
		panic("invalid workflow config: " + err.Error())
	}
	cfg.Server.Port = wfCfg.HTTPPort

	// 2. Initialize structured logger
	logger := observability.NewLogger(
		cfg.Observability.LogLevel,
		cfg.Observability.LogFormat,
		"workflow-engine",
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 3. Initialize tracer
	shutdownTracer, err := observability.InitTracer(ctx, "workflow-engine", cfg.Observability.OTLPEndpoint)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to initialize tracer")
	} else {
		defer shutdownTracer(ctx)
	}

	// 4. Connect PostgreSQL
	db, err := database.NewPostgresPool(ctx, cfg.Database, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	// 5. Run schema migration
	if err := repository.RunMigration(ctx, db); err != nil {
		logger.Fatal().Err(err).Msg("failed to run workflow schema migration")
	}
	logger.Info().Msg("workflow schema migration completed")

	// 6. Connect Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Warn().Err(err).Msg("redis connection failed — continuing with degraded functionality")
	}

	// 7. Create HTTP server with middleware stack
	srv, err := server.New(cfg, db, rdb, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create server")
	}

	// 8. Initialize Kafka producer (optional)
	var producer *events.Producer
	kafkaProducer, err := events.NewProducer(cfg.Kafka, logger)
	if err != nil {
		logger.Warn().Err(err).Msg("kafka producer unavailable — workflow events will not be published")
	} else {
		producer = kafkaProducer
		defer producer.Close()
	}

	// 9. Initialize repositories
	defRepo := repository.NewDefinitionRepository(db)
	instRepo := repository.NewInstanceRepository(db)
	taskRepo := repository.NewTaskRepository(db)

	// 11. Initialize executor registry
	execRegistry := executor.NewExecutorRegistry()
	execRegistry.Register("service_task", executor.NewServiceTaskExecutor(wfCfg.ServiceURLs, logger))
	execRegistry.Register("human_task", executor.NewHumanTaskExecutor(taskRepo, logger))
	execRegistry.Register("event_task", executor.NewEventTaskExecutor(producer, rdb, logger))
	execRegistry.Register("condition", executor.NewConditionExecutor())
	execRegistry.Register("timer", executor.NewTimerTaskExecutor(rdb, taskRepo, logger))

	// 12. Initialize services
	defSvc := service.NewDefinitionService(defRepo, logger)
	engineSvc := service.NewEngineService(instRepo, defRepo, taskRepo, execRegistry, producer, logger)
	taskSvc := service.NewTaskService(taskRepo, engineSvc, logger)
	templateSvc := service.NewTemplateService(defRepo, logger)
	schedulerSvc := service.NewSchedulerService(rdb, taskRepo, engineSvc, producer, logger,
		wfCfg.TimerPollIntervalSec, wfCfg.SLACheckIntervalSec)
	recoverySvc := service.NewRecoveryService(instRepo, defRepo, taskRepo, rdb, engineSvc, logger,
		wfCfg.InstanceRecoveryBatch)

	// 13. Initialize handlers
	defHandler := handler.NewDefinitionHandler(defSvc, logger)
	instHandler := handler.NewInstanceHandler(engineSvc, instRepo, logger)
	taskHandler := handler.NewTaskHandler(taskSvc, logger)
	templateHandler := handler.NewTemplateHandler(templateSvc, logger)

	// 14. Initialize health checker
	healthChecker := health.NewChecker(db, rdb, logger)

	// Override health endpoints
	srv.Router.Get("/healthz", healthChecker.LivenessHandler())
	srv.Router.Get("/readyz", healthChecker.ReadinessHandler())
	srv.Router.Handle("/metrics", promhttp.Handler())

	// 15. Register API routes
	srv.Router.Route("/api/v1/workflows", func(r chi.Router) {
		r.Use(middleware.Auth(srv.JWTManager))
		r.Use(middleware.Tenant)

		r.Route("/definitions", func(r chi.Router) {
			r.Mount("/", defHandler.Routes())
		})

		r.Route("/instances", func(r chi.Router) {
			r.Mount("/", instHandler.Routes())
		})

		r.Route("/tasks", func(r chi.Router) {
			r.Mount("/", taskHandler.Routes())
		})

		r.Route("/templates", func(r chi.Router) {
			r.Mount("/", templateHandler.Routes())
		})
	})

	// 16. Initialize Kafka consumers
	triggerConsumer := consumer.NewTriggerConsumer(defRepo, engineSvc, rdb, logger)
	eventWaitConsumer := consumer.NewEventWaitConsumer(rdb, engineSvc, logger)

	// 17. Start all components via errgroup
	g, gCtx := errgroup.WithContext(ctx)

	// HTTP server
	g.Go(func() error {
		logger.Info().Int("port", cfg.Server.Port).Msg("workflow-engine HTTP server starting")
		if err := srv.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})

	// Scheduler (timer poller + SLA monitor)
	g.Go(func() error {
		schedulerSvc.Start(gCtx)
		return nil
	})

	// Recovery service on startup
	g.Go(func() error {
		if err := recoverySvc.Recover(gCtx); err != nil {
			logger.Error().Err(err).Msg("instance recovery encountered errors")
		}
		return nil
	})

	// Kafka consumers
	var kafkaConsumerClient *events.Consumer
	if len(cfg.Kafka.Brokers) > 0 {
		kc, err := events.NewConsumer(cfg.Kafka, logger)
		if err != nil {
			logger.Warn().Err(err).Msg("kafka consumer unavailable — event consumers disabled")
		} else {
			kafkaConsumerClient = kc

			// Subscribe trigger consumer to domain event topics
			kc.Subscribe("platform.workflow.events", triggerConsumer)
			kc.Subscribe("platform.cyber.events", triggerConsumer)
			kc.Subscribe("platform.iam.events", triggerConsumer)
			kc.Subscribe("platform.data.events", triggerConsumer)
			kc.Subscribe("platform.enterprise.events", triggerConsumer)

			// Event wait consumer for correlation matching
			kc.Subscribe("platform.workflow.events", eventWaitConsumer)

			g.Go(func() error {
				if err := kc.Start(gCtx); err != nil {
					logger.Error().Err(err).Msg("kafka consumer error")
				}
				return nil
			})
		}
	}

	// 18. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		logger.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	case <-gCtx.Done():
		logger.Info().Msg("context cancelled")
	}

	cancel()

	schedulerSvc.Stop()

	if kafkaConsumerClient != nil {
		kafkaConsumerClient.Stop()
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.HTTPServer.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("HTTP server shutdown error")
	}

	if err := g.Wait(); err != nil {
		logger.Error().Err(err).Msg("errgroup error during shutdown")
	}

	logger.Info().Msg("workflow-engine stopped gracefully")
}
