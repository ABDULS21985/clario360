package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/cyber/classifier"
	cyberconfig "github.com/clario360/platform/internal/cyber/config"
	"github.com/clario360/platform/internal/cyber/consumer"
	"github.com/clario360/platform/internal/cyber/enrichment"
	"github.com/clario360/platform/internal/cyber/handler"
	cybermetrics "github.com/clario360/platform/internal/cyber/metrics"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/cyber/scanner"
	"github.com/clario360/platform/internal/cyber/service"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/observability"
	"github.com/clario360/platform/internal/server"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// ── 1. Load platform config (shared DB/Redis/Kafka/Auth settings) ─────────
	cfg, err := config.Load()
	if err != nil {
		panic("loading platform config: " + err.Error())
	}
	cfg.Server.Port = 8090

	// ── 2. Load cyber-service specific config ─────────────────────────────────
	cyberCfg, err := cyberconfig.Load()
	if err != nil {
		// Cyber config is required; a missing env var is a fatal startup error.
		// Use fmt so we don't need a logger yet.
		os.Stderr.WriteString("loading cyber config: " + err.Error() + "\n")
		os.Exit(1)
	}

	// ── 3. Logger ──────────────────────────────────────────────────────────────
	logger := observability.NewLogger(
		cfg.Observability.LogLevel,
		cfg.Observability.LogFormat,
		"cyber-service",
	)

	// ── 4. Tracer ──────────────────────────────────────────────────────────────
	shutdownTracer, err := observability.InitTracer(ctx, "cyber-service", cfg.Observability.OTLPEndpoint)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to initialize tracer — continuing without tracing")
	} else {
		defer shutdownTracer(ctx)
	}

	// ── 5. Database ────────────────────────────────────────────────────────────
	db, err := database.NewPostgresPool(ctx, cfg.Database, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	// ── 6. Redis ───────────────────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	// ── 7. Prometheus registries ───────────────────────────────────────────────
	// Use a Gatherers to merge the standard Go/process metrics with the
	// cyber-service application metrics so both are exposed at /metrics.
	m := cybermetrics.New()
	runtimeReg := prometheus.NewRegistry()
	runtimeReg.MustRegister(prometheus.NewGoCollector())
	runtimeReg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	promGatherers := prometheus.Gatherers{runtimeReg, m.Registry}

	// ── 8. Kafka producer ──────────────────────────────────────────────────────
	var producer *events.Producer
	if len(cfg.Kafka.Brokers) > 0 && cfg.Kafka.Brokers[0] != "" {
		producer, err = events.NewProducer(cfg.Kafka, logger)
		if err != nil {
			logger.Warn().Err(err).Msg("Kafka producer unavailable — events will not be published")
		} else {
			defer producer.Close()
		}
	}

	// ── 9. Repositories ────────────────────────────────────────────────────────
	assetRepo := repository.NewAssetRepository(db, logger)
	vulnRepo := repository.NewVulnerabilityRepository(db, logger)
	relRepo := repository.NewRelationshipRepository(db, logger)
	scanRepo := repository.NewScanRepository(db, logger)

	// ── 10. Classifier ─────────────────────────────────────────────────────────
	cls := classifier.NewAssetClassifier(logger)

	// ── 11. Enrichment pipeline ────────────────────────────────────────────────
	dnsEnricher := enrichment.NewDNSEnricher(logger, time.Duration(cyberCfg.EnrichmentDNSTimeoutSec)*time.Second)
	cveEnricher := enrichment.NewCVEEnricher(logger, vulnRepo, cyberCfg.EnrichmentCVEEnabled)
	geoEnricher := enrichment.NewGeoEnricher(logger, cyberCfg.EnrichmentGeoDBPath, cyberCfg.EnrichmentGeoEnabled)
	pipeline := enrichment.NewPipeline(logger, dnsEnricher, cveEnricher, geoEnricher)

	// ── 12. Enrichment service ─────────────────────────────────────────────────
	enrichSvc := service.NewEnrichmentService(pipeline, assetRepo, m, logger)

	// ── 13. Scanner registry ───────────────────────────────────────────────────
	scanRegistry := scanner.NewRegistry()

	networkScanner := scanner.NewNetworkScanner(
		assetRepo, pipeline, cls, logger,
		cyberCfg.ScanNetworkWorkers,
		cyberCfg.ScanNetworkTimeoutSec,
		cyberCfg.ScanNetworkMaxIPs,
		cyberCfg.ScanDefaultPorts,
	)
	cloudScanner := scanner.NewCloudScanner(assetRepo, logger)
	agentCollector := scanner.NewAgentCollector(assetRepo, logger)

	scanRegistry.Register(networkScanner)
	scanRegistry.Register(cloudScanner)
	scanRegistry.Register(agentCollector)

	// ── 14. Asset service ──────────────────────────────────────────────────────
	assetSvc := service.NewAssetService(
		assetRepo, vulnRepo, relRepo, scanRepo,
		scanRegistry, cls, enrichSvc,
		producer, m, cyberCfg, db, logger,
	)

	// ── 15. HTTP server ────────────────────────────────────────────────────────
	srv, err := server.New(cfg, db, rdb, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create HTTP server")
	}

	// Expose both runtime and cyber-service application metrics at /metrics.
	srv.Router.Handle("/metrics", promhttp.HandlerFor(promGatherers, promhttp.HandlerOpts{}))

	// Register JWT manager for route middleware
	jwtMgr, err := auth.NewJWTManager(cfg.Auth)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create JWT manager")
	}

	// ── 16. Routes ─────────────────────────────────────────────────────────────
	assetHandler := handler.NewAssetHandler(assetSvc, logger)
	handler.RegisterRoutes(srv.Router, assetHandler, jwtMgr)

	// ── 17. Kafka consumer ─────────────────────────────────────────────────────
	var cyberConsumer *consumer.CyberConsumer
	if len(cfg.Kafka.Brokers) > 0 && cfg.Kafka.Brokers[0] != "" {
		kafkaConsumer, err := events.NewConsumer(cfg.Kafka, logger)
		if err != nil {
			logger.Warn().Err(err).Msg("Kafka consumer unavailable — event processing disabled")
		} else {
			cyberConsumer = consumer.NewCyberConsumer(assetSvc, kafkaConsumer, logger)
		}
	}

	// ── 18. Scan scheduler ─────────────────────────────────────────────────────
	sched := scanner.NewScheduler(logger)
	// Add scheduled scans here via sched.Register(...)

	// ── 19. Start all components ───────────────────────────────────────────────
	g, gCtx := errgroup.WithContext(ctx)

	// HTTP server
	g.Go(func() error {
		logger.Info().Int("port", cfg.Server.Port).Msg("cyber-service starting")
		return srv.Start()
	})

	// Kafka consumer
	if cyberConsumer != nil {
		g.Go(func() error {
			return cyberConsumer.Start(gCtx)
		})
	}

	// Scheduler (no-op until scans are registered)
	g.Go(func() error {
		return sched.Start(gCtx)
	})

	// Wait for shutdown
	if err := g.Wait(); err != nil {
		logger.Error().Err(err).Msg("cyber-service stopped with error")
	}

	// Graceful shutdown
	if cyberConsumer != nil {
		_ = cyberConsumer.Stop()
	}

	logger.Info().Msg("cyber-service shutdown complete")
}
