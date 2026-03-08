package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/config"
	dataanalytics "github.com/clario360/platform/internal/data/analytics"
	dataconfig "github.com/clario360/platform/internal/data/config"
	"github.com/clario360/platform/internal/data/connector"
	dataconsumer "github.com/clario360/platform/internal/data/consumer"
	datacontradiction "github.com/clario360/platform/internal/data/contradiction"
	datadarkdata "github.com/clario360/platform/internal/data/darkdata"
	datadarkdatastrategies "github.com/clario360/platform/internal/data/darkdata/strategies"
	datadashboard "github.com/clario360/platform/internal/data/dashboard"
	"github.com/clario360/platform/internal/data/handler"
	datahealth "github.com/clario360/platform/internal/data/health"
	datalineage "github.com/clario360/platform/internal/data/lineage"
	datametrics "github.com/clario360/platform/internal/data/metrics"
	datapipeline "github.com/clario360/platform/internal/data/pipeline"
	dataquality "github.com/clario360/platform/internal/data/quality"
	"github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/data/service"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/events"
	bootstrap "github.com/clario360/platform/internal/observability/bootstrap"
	"github.com/clario360/platform/internal/observability/tracing"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		os.Stderr.WriteString("loading platform config: " + err.Error() + "\n")
		os.Exit(1)
	}

	dataCfg, err := dataconfig.Load()
	if err != nil {
		os.Stderr.WriteString("loading data config: " + err.Error() + "\n")
		os.Exit(1)
	}

	if port, err := strconv.Atoi(dataCfg.HTTPPort); err == nil {
		cfg.Server.Port = port
	}
	cfg.Kafka.Brokers = dataCfg.KafkaBrokers
	cfg.Kafka.GroupID = dataCfg.KafkaGroupID
	publicKeyPEM, err := os.ReadFile(dataCfg.JWTPublicKeyPath)
	if err != nil {
		os.Stderr.WriteString("reading DATA_JWT_PUBLIC_KEY_PATH: " + err.Error() + "\n")
		os.Exit(1)
	}
	cfg.Auth.RSAPublicKeyPEM = string(publicKeyPEM)

	bootstrapCfg, err := buildBootstrapConfig(cfg, dataCfg)
	if err != nil {
		os.Stderr.WriteString("building bootstrap config: " + err.Error() + "\n")
		os.Exit(1)
	}
	svc, err := bootstrap.Bootstrap(ctx, bootstrapCfg)
	if err != nil {
		os.Stderr.WriteString("bootstrapping data-service: " + err.Error() + "\n")
		os.Exit(1)
	}
	logger := svc.Logger

	migrationsPath := envOr("DATA_MIGRATIONS_PATH", filepath.Join("migrations", "data_db"))
	if _, err := os.Stat(migrationsPath); err != nil {
		migrationsPath = filepath.Join("backend", "migrations", "data_db")
	}
	if err := database.RunMigrations(dataCfg.DBURL, migrationsPath); err != nil {
		logger.Fatal().Err(err).Str("path", migrationsPath).Msg("failed to run data migrations")
	}

	var producer *events.Producer
	if len(cfg.Kafka.Brokers) > 0 && cfg.Kafka.Brokers[0] != "" {
		producer, err = events.NewProducer(cfg.Kafka, logger)
		if err != nil {
			logger.Warn().Err(err).Msg("Kafka producer unavailable — events will not be published")
		}
	}

	sourceRepo := repository.NewSourceRepository(svc.DBPool, logger)
	modelRepo := repository.NewModelRepository(svc.DBPool, logger)
	syncRepo := repository.NewSyncRepository(svc.DBPool, logger)
	pipelineRepo := repository.NewPipelineRepository(svc.DBPool, logger)
	runRepo := repository.NewPipelineRunRepository(svc.DBPool, logger)
	logRepo := repository.NewPipelineRunLogRepository(svc.DBPool, logger)
	qualityRuleRepo := repository.NewQualityRuleRepository(svc.DBPool, logger)
	qualityResultRepo := repository.NewQualityResultRepository(svc.DBPool, logger)
	contradictionRepo := repository.NewContradictionRepository(svc.DBPool, logger)
	lineageRepo := repository.NewLineageRepository(svc.DBPool, logger)
	darkDataRepo := repository.NewDarkDataRepository(svc.DBPool, logger)
	analyticsRepo := repository.NewAnalyticsRepository(svc.DBPool, logger)
	dataDashboardRepo := repository.NewDashboardRepository(svc.DBPool, logger)
	dataMetrics := datametrics.New(svc.Metrics.Registry())

	registry := connector.NewConnectorRegistry(connector.ConnectorLimits{
		MaxPoolSize:      dataCfg.ConnectorMaxPoolSize,
		StatementTimeout: dataCfg.ConnectorStatementTimeout,
		ConnectTimeout:   dataCfg.ConnectorConnectTimeout,
		MaxSampleRows:    dataCfg.ConnectorMaxSampleRows,
		MaxTables:        dataCfg.ConnectorMaxTables,
		APIRateLimit:     dataCfg.ConnectorAPIRateLimit,
	}, logger)
	discoveryOpts := connector.DiscoveryOptions{
		MaxTables:    dataCfg.ConnectorMaxTables,
		MaxColumns:   dataCfg.DiscoveryMaxColumns,
		SampleValues: dataCfg.DiscoverySampleValues,
		MaxSamples:   dataCfg.ConnectorMaxSampleRows,
		IncludeViews: true,
	}
	encryptor, err := service.NewConfigEncryptorFromBytes(dataCfg.EncryptionKey)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize connection config encryptor")
	}
	decryptor := configDecryptorAdapter{encryptor: encryptor}

	tester := service.NewConnectionTester(registry, dataMetrics)
	discoverySvc := service.NewSchemaDiscoveryService(registry, discoveryOpts, dataMetrics)
	ingestionSvc := service.NewIngestionService(registry, sourceRepo, syncRepo, discoveryOpts, dataMetrics, logger)
	sourceSvc := service.NewSourceService(dataCfg, sourceRepo, syncRepo, tester, discoverySvc, ingestionSvc, encryptor, producer, dataMetrics, logger)
	modelSvc := service.NewModelService(modelRepo, sourceRepo, producer, dataMetrics, logger)
	extractor := datapipeline.NewExtractor(registry, decryptor)
	transformer := datapipeline.NewTransformer(logger)
	loader := datapipeline.NewLoader(registry, decryptor, modelRepo, sourceRepo)
	qualityGateEvaluator := datapipeline.NewQualityGateEvaluator()
	pipelineEngine := datapipeline.NewEngine(
		pipelineRepo,
		runRepo,
		logRepo,
		sourceRepo,
		modelRepo,
		extractor,
		transformer,
		loader,
		qualityGateEvaluator,
		producer,
		logger,
		envInt("DATA_PIPELINE_MAX_CONCURRENT", 10),
	)
	pipelineScheduler := datapipeline.NewScheduler(
		pipelineRepo,
		pipelineEngine,
		logger,
		envDuration("DATA_PIPELINE_SCHEDULER_INTERVAL", 15*time.Second),
	)
	qualityExecutor := dataquality.NewExecutor(
		registry,
		sourceRepo,
		modelRepo,
		qualityRuleRepo,
		qualityResultRepo,
		decryptor,
		producer,
		logger,
	)
	qualityScorer := dataquality.NewScorer(qualityRuleRepo, qualityResultRepo, modelRepo)
	qualityScheduler := dataquality.NewScheduler(
		qualityRuleRepo,
		qualityExecutor,
		logger,
		envDuration("DATA_QUALITY_SCHEDULER_INTERVAL", 30*time.Second),
	)
	contradictionDetector := datacontradiction.NewDetector(
		registry,
		sourceRepo,
		modelRepo,
		contradictionRepo,
		decryptor,
		producer,
		logger,
	)
	pipelineSvc := service.NewPipelineService(pipelineRepo, runRepo, logRepo, sourceRepo, modelRepo, pipelineEngine, producer, logger)
	qualitySvc := service.NewQualityService(qualityRuleRepo, qualityResultRepo, modelRepo, qualityExecutor, qualityScorer, producer)
	contradictionSvc := service.NewContradictionService(contradictionRepo, contradictionDetector, producer)
	lineageBuilder := datalineage.NewGraphBuilder(svc.DBPool, lineageRepo, logger)
	lineageAnalyzer := datalineage.NewImpactAnalyzer(lineageBuilder)
	lineageRecorder := datalineage.NewLineageRecorder(lineageRepo, sourceRepo, modelRepo, producer, logger)
	lineageSvc := service.NewLineageService(lineageRepo, lineageBuilder, lineageAnalyzer, lineageRecorder, producer, logger)
	darkDataClassifier := datadarkdata.NewClassifier()
	darkDataRiskScorer := datadarkdata.NewRiskScorer()
	darkDataScanner := datadarkdata.NewScanner([]datadarkdata.DarkDataStrategy{
		datadarkdatastrategies.NewUnmodeledTablesStrategy(sourceRepo, modelRepo),
		datadarkdatastrategies.NewOrphanedFilesStrategy(svc.DBPool, dataCfg.MinIOEndpoint, dataCfg.MinIOAccessKey, dataCfg.MinIOSecretKey, dataCfg.MinIOBucket),
		datadarkdatastrategies.NewStaleAssetsStrategy(svc.DBPool),
		datadarkdatastrategies.NewUngovernedDataStrategy(svc.DBPool),
	}, darkDataRepo, darkDataRiskScorer, darkDataClassifier, producer, logger)
	darkDataSvc := service.NewDarkDataService(darkDataRepo, darkDataScanner, modelSvc, sourceRepo, lineageSvc, producer, logger)
	auditRecorder := dataanalytics.NewAuditRecorder(analyticsRepo, logger)
	analyticsSvc := service.NewAnalyticsService(analyticsRepo, modelRepo, sourceRepo, registry, encryptor, auditRecorder, lineageSvc, producer, logger)
	dashboardCache := datadashboard.NewCache(svc.Redis, 60*time.Second)
	dashboardCalculator := datadashboard.NewCalculator(sourceRepo, pipelineRepo, dataDashboardRepo, contradictionRepo, darkDataRepo, lineageRepo, qualityScorer, dashboardCache, logger)
	dashboardSvc := service.NewDashboardService(dashboardCalculator)
	guard := events.NewIdempotencyGuard(svc.Redis, 24*time.Hour)
	crossSuiteMetrics := events.NewCrossSuiteMetrics(svc.Metrics.Registry())
	dlqTracker := events.NewDLQTracker(svc.Redis)

	jwtMgr, err := auth.NewJWTManager(cfg.Auth)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create JWT manager")
	}

	sourceHandler := handler.NewSourceHandler(sourceSvc, logger)
	modelHandler := handler.NewModelHandler(modelSvc, logger)
	pipelineHandler := handler.NewPipelineHandler(pipelineSvc, logger)
	qualityHandler := handler.NewQualityHandler(qualitySvc, logger)
	contradictionHandler := handler.NewContradictionHandler(contradictionSvc, logger)
	lineageHandler := handler.NewLineageHandler(lineageSvc, logger)
	darkDataHandler := handler.NewDarkDataHandler(darkDataSvc, logger)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsSvc, logger)
	dashboardHandler := handler.NewDashboardHandler(dashboardSvc, logger)

	datahealth.Register(svc.Router, svc.Health, "data-service", cfg.Observability.ServiceName)
	handler.RegisterRoutes(
		svc.Router,
		sourceHandler,
		modelHandler,
		pipelineHandler,
		qualityHandler,
		contradictionHandler,
		lineageHandler,
		darkDataHandler,
		analyticsHandler,
		dashboardHandler,
		jwtMgr,
		svc.Redis,
	)
	svc.Router.Get("/api/v1/admin/dlq/count", events.DLQCountHandler("data-service", dlqTracker, logger))
	pipelineScheduler.Start(ctx)
	qualityScheduler.Start(ctx)

	var dataConsumer *dataconsumer.DataConsumer
	var lineageConsumer *dataconsumer.LineageConsumer
	if len(cfg.Kafka.Brokers) > 0 && cfg.Kafka.Brokers[0] != "" {
		kafkaConsumer, err := events.NewConsumer(cfg.Kafka, logger)
		if err != nil {
			logger.Warn().Err(err).Msg("Kafka consumer unavailable — background discovery disabled")
		} else {
			kafkaConsumer.SetDeadLetterProducer(producer)
			kafkaConsumer.SetCrossSuiteMetrics(crossSuiteMetrics)
			kafkaConsumer.SetDLQTracker(dlqTracker, "data-service")
			dataConsumer = dataconsumer.NewDataConsumer(sourceSvc, kafkaConsumer, logger)
			lineageConsumer = dataconsumer.NewLineageConsumer(lineageSvc, pipelineRepo, runRepo, dashboardCache, kafkaConsumer, logger)
			kafkaConsumer.Subscribe(events.Topics.PipelineEvents, dataconsumer.NewFailureTracker(svc.Redis, guard, producer, logger, crossSuiteMetrics))
		}
	}

	g, gCtx := errgroup.WithContext(ctx)
	if dataConsumer != nil {
		g.Go(func() error {
			err := dataConsumer.Start(gCtx)
			if err != nil && !errors.Is(err, context.Canceled) {
				return err
			}
			return nil
		})
	}

	logger.Info().Int("port", bootstrapCfg.Port).Msg("data-service starting")
	runErr := svc.Run(ctx)
	cancel()
	if waitErr := g.Wait(); waitErr != nil {
		logger.Error().Err(waitErr).Msg("data-service background components stopped with error")
	}
	if runErr != nil && !errors.Is(runErr, context.Canceled) {
		logger.Error().Err(runErr).Msg("data-service stopped with error")
	}
	if dataConsumer != nil {
		_ = dataConsumer.Stop()
	}
	if lineageConsumer != nil {
		_ = lineageConsumer.Stop()
	}
	pipelineScheduler.Stop()
	qualityScheduler.Stop()
	if producer != nil {
		_ = producer.Close()
	}
}

func buildBootstrapConfig(cfg *config.Config, dataCfg *dataconfig.Config) (*bootstrap.ServiceConfig, error) {
	redisURL, err := url.Parse(dataCfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	redisPassword, _ := redisURL.User.Password()
	redisDB := 0
	if dbSegment := strings.TrimPrefix(redisURL.Path, "/"); dbSegment != "" {
		if parsed, parseErr := strconv.Atoi(dbSegment); parseErr == nil {
			redisDB = parsed
		}
	}

	return &bootstrap.ServiceConfig{
		Name:            "data-service",
		Version:         cfg.Observability.ServiceName,
		Environment:     envOr("ENVIRONMENT", "development"),
		Port:            mustParsePort(dataCfg.HTTPPort, 8091),
		AdminPort:       cfg.Observability.MetricsPort,
		LogLevel:        cfg.Observability.LogLevel,
		DebugSampleRate: 100,
		ShutdownTimeout: cfg.Server.ShutdownTimeout,
		ReadTimeout:     cfg.Server.ReadTimeout,
		WriteTimeout:    cfg.Server.WriteTimeout,
		Tracing: tracing.TracerConfig{
			Enabled:     cfg.Observability.OTLPEndpoint != "",
			Endpoint:    cfg.Observability.OTLPEndpoint,
			ServiceName: "data-service",
			Version:     cfg.Observability.ServiceName,
			Environment: envOr("ENVIRONMENT", "development"),
			SampleRate:  1,
			Insecure:    true,
		},
		DB: &bootstrap.DBConfig{
			URL:               dataCfg.DBURL,
			MinConns:          dataCfg.DBMinConns,
			MaxConns:          dataCfg.DBMaxConns,
			MaxConnLife:       time.Hour,
			MaxConnIdle:       30 * time.Minute,
			HealthCheckPeriod: time.Minute,
		},
		Redis: &bootstrap.RedisConfig{
			Addr:     redisURL.Host,
			Password: redisPassword,
			DB:       redisDB,
		},
		Kafka: &bootstrap.KafkaConfig{
			Brokers: dataCfg.KafkaBrokers,
			GroupID: dataCfg.KafkaGroupID,
		},
	}, nil
}

func mustParsePort(raw string, fallback int) int {
	if port, err := strconv.Atoi(raw); err == nil {
		return port
	}
	return fallback
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return fallback
}

type configDecryptorAdapter struct {
	encryptor *service.ConfigEncryptor
}

func (a configDecryptorAdapter) Decrypt(ciphertext []byte, _ string) ([]byte, error) {
	return a.encryptor.Decrypt(ciphertext)
}
