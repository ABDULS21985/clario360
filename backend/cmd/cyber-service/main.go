package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sync/errgroup"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/cyber/classifier"
	cyberconfig "github.com/clario360/platform/internal/cyber/config"
	"github.com/clario360/platform/internal/cyber/consumer"
	cyberctem "github.com/clario360/platform/internal/cyber/ctem"
	cyberdashboard "github.com/clario360/platform/internal/cyber/dashboard"
	"github.com/clario360/platform/internal/cyber/detection"
	cyberdspm "github.com/clario360/platform/internal/cyber/dspm"
	"github.com/clario360/platform/internal/cyber/enrichment"
	"github.com/clario360/platform/internal/cyber/handler"
	"github.com/clario360/platform/internal/cyber/indicator"
	cybermetrics "github.com/clario360/platform/internal/cyber/metrics"
	"github.com/clario360/platform/internal/cyber/model"
	cyberremediation "github.com/clario360/platform/internal/cyber/remediation"
	remediationstrategy "github.com/clario360/platform/internal/cyber/remediation/strategy"
	"github.com/clario360/platform/internal/cyber/repository"
	cyberrisk "github.com/clario360/platform/internal/cyber/risk"
	riskcomponents "github.com/clario360/platform/internal/cyber/risk/components"
	"github.com/clario360/platform/internal/cyber/scanner"
	"github.com/clario360/platform/internal/cyber/service"
	cybervciso "github.com/clario360/platform/internal/cyber/vciso"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/events"
	bootstrap "github.com/clario360/platform/internal/observability/bootstrap"
	"github.com/clario360/platform/internal/observability/tracing"
	"github.com/clario360/platform/internal/suiteapi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	workflowrepo "github.com/clario360/platform/internal/workflow/repository"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// ── 1. Load platform config (shared DB/Redis/Kafka/Auth settings) ─────────
	cfg, err := config.Load()
	if err != nil {
		os.Stderr.WriteString("loading platform config: " + err.Error() + "\n")
		os.Exit(1)
	}

	// ── 2. Load cyber-service specific config ─────────────────────────────────
	cyberCfg, err := cyberconfig.Load()
	if err != nil {
		// Cyber config is required; a missing env var is a fatal startup error.
		// Use fmt so we don't need a logger yet.
		os.Stderr.WriteString("loading cyber config: " + err.Error() + "\n")
		os.Exit(1)
	}
	if port, err := strconv.Atoi(cyberCfg.HTTPPort); err == nil {
		cfg.Server.Port = port
	}
	cfg.Kafka.Brokers = cyberCfg.KafkaBrokers
	cfg.Kafka.GroupID = cyberCfg.KafkaGroupID
	publicKeyPEM, err := os.ReadFile(cyberCfg.JWTPublicKeyPath)
	if err != nil {
		os.Stderr.WriteString("reading CYBER_JWT_PUBLIC_KEY_PATH: " + err.Error() + "\n")
		os.Exit(1)
	}
	cfg.Auth.RSAPublicKeyPEM = string(publicKeyPEM)

	// ── 3. Bootstrap shared infrastructure ─────────────────────────────────────
	bootstrapCfg, err := buildBootstrapConfig(cfg, cyberCfg)
	if err != nil {
		os.Stderr.WriteString("building bootstrap config: " + err.Error() + "\n")
		os.Exit(1)
	}
	svc, err := bootstrap.Bootstrap(ctx, bootstrapCfg)
	if err != nil {
		os.Stderr.WriteString("bootstrapping cyber-service: " + err.Error() + "\n")
		os.Exit(1)
	}
	logger := svc.Logger
	db := svc.DBPool
	rdb := svc.Redis

	migrationsPath := envOr("CYBER_MIGRATIONS_PATH", filepath.Join("migrations", "cyber_db"))
	if _, err := os.Stat(migrationsPath); err != nil {
		migrationsPath = filepath.Join("backend", "migrations", "cyber_db")
	}
	if err := database.RunMigrations(cyberCfg.DBURL, migrationsPath); err != nil {
		logger.Fatal().Err(err).Str("path", migrationsPath).Msg("failed to run cyber migrations")
	}
	if err := workflowrepo.RunMigration(ctx, db); err != nil {
		logger.Fatal().Err(err).Msg("failed to run workflow schema migration for cyber-service")
	}

	// ── 4. Prometheus registries ───────────────────────────────────────────────
	// Use a Gatherers to merge the standard Go/process metrics with the
	// shared bootstrap registry and cyber-service application metrics.
	m := cybermetrics.New()
	promGatherers := prometheus.Gatherers{svc.Metrics.Registry(), m.Registry}

	// ── 5. Kafka producer ──────────────────────────────────────────────────────
	var producer *events.Producer
	if len(cfg.Kafka.Brokers) > 0 && cfg.Kafka.Brokers[0] != "" {
		producer, err = events.NewProducer(cfg.Kafka, logger)
		if err != nil {
			logger.Warn().Err(err).Msg("Kafka producer unavailable — events will not be published")
		}
	}

	// ── 6. Repositories ────────────────────────────────────────────────────────
	assetRepo := repository.NewAssetRepository(db, logger)
	vulnRepo := repository.NewVulnerabilityRepository(db, logger)
	relRepo := repository.NewRelationshipRepository(db, logger)
	scanRepo := repository.NewScanRepository(db, logger)
	alertRepo := repository.NewAlertRepository(db, logger)
	commentRepo := repository.NewCommentRepository(db, logger)
	ruleRepo := repository.NewRuleRepository(db, logger)
	threatRepo := repository.NewThreatRepository(db, logger)
	indicatorRepo := repository.NewIndicatorRepository(db, logger)
	dashboardRepo := repository.NewDashboardRepository(db, logger)
	riskHistoryRepo := repository.NewRiskHistoryRepository(db, logger)
	ctemAssessmentRepo := repository.NewCTEMAssessmentRepository(db, logger)
	ctemFindingRepo := repository.NewCTEMFindingRepository(db, logger)
	ctemRemGroupRepo := repository.NewCTEMRemediationGroupRepository(db, logger)
	ctemSnapshotRepo := repository.NewCTEMSnapshotRepository(db, logger)
	remediationRepo := repository.NewRemediationRepository(db, logger)
	remediationAuditRepo := repository.NewRemediationAuditRepository(db, logger)
	dspmRepo := repository.NewDSPMRepository(db, logger)
	vcisoRepo := repository.NewVCISORepository(db, logger)

	workflowDefRepo := workflowrepo.NewDefinitionRepository(db)
	workflowInstRepo := workflowrepo.NewInstanceRepository(db)
	workflowTaskRepo := workflowrepo.NewTaskRepository(db)

	// ── 7. Classifier ──────────────────────────────────────────────────────────
	cls := classifier.NewAssetClassifier(logger)

	// ── 8. Enrichment pipeline ─────────────────────────────────────────────────
	dnsEnricher := enrichment.NewDNSEnricher(logger, time.Duration(cyberCfg.EnrichmentDNSTimeoutSec)*time.Second)
	cveEnricher := enrichment.NewCVEEnricher(logger, vulnRepo, cyberCfg.EnrichmentCVEEnabled)
	geoEnricher := enrichment.NewGeoEnricher(logger, cyberCfg.EnrichmentGeoDBPath, cyberCfg.EnrichmentGeoEnabled)
	pipeline := enrichment.NewPipeline(logger, dnsEnricher, cveEnricher, geoEnricher)

	// ── 9. Enrichment service ──────────────────────────────────────────────────
	enrichSvc := service.NewEnrichmentService(pipeline, assetRepo, m, logger)

	// ── 10. Scanner registry ───────────────────────────────────────────────────
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

	// ── 11. Asset service ──────────────────────────────────────────────────────
	assetSvc := service.NewAssetService(
		assetRepo, vulnRepo, relRepo, scanRepo,
		scanRegistry, cls, enrichSvc,
		producer, m, cyberCfg, db, logger,
	)
	alertSvc := service.NewAlertService(alertRepo, commentRepo, db, producer, logger)
	baselineStore := detection.NewBaselineStore(rdb, logger)
	ruleSvc := service.NewRuleService(ruleRepo, alertSvc, baselineStore, producer, logger)
	if err := ruleSvc.EnsureTemplates(ctx); err != nil {
		logger.Fatal().Err(err).Msg("failed to seed detection rule templates")
	}
	indicatorMatcher := indicator.NewMatcher(indicatorRepo, logger)
	detectionEngine := detection.NewDetectionEngine(
		ruleRepo,
		assetRepo,
		threatRepo,
		alertSvc,
		indicatorMatcher,
		rdb,
		producer,
		baselineStore,
		logger,
	)
	detectionSvc := service.NewDetectionService(detectionEngine, logger)
	detectionSvc.Start(ctx, time.Duration(cyberCfg.DetectionRuleRefreshSec)*time.Second)
	threatSvc := service.NewThreatService(threatRepo, indicatorRepo, producer, logger)
	workflowLauncher := service.NewWorkflowRemediationLauncher(workflowDefRepo, workflowInstRepo, workflowTaskRepo, logger)
	scoringEngine := cyberctem.NewScoringEngine(db, ctemSnapshotRepo, logger)
	ctemEngine := cyberctem.NewEngine(
		db,
		ctemAssessmentRepo,
		ctemFindingRepo,
		ctemSnapshotRepo,
		ctemRemGroupRepo,
		assetRepo,
		vulnRepo,
		relRepo,
		scoringEngine,
		producer,
		workflowLauncher,
		logger,
	)
	ctemSvc := service.NewCTEMService(
		db,
		ctemAssessmentRepo,
		ctemFindingRepo,
		ctemRemGroupRepo,
		ctemSnapshotRepo,
		assetRepo,
		ctemEngine,
		scoringEngine,
		producer,
		workflowLauncher,
		logger,
	)
	vulnerabilityRisk := riskcomponents.NewVulnerabilityRisk(db, logger)
	threatExposure := riskcomponents.NewThreatExposure(db, logger)
	configurationRisk := riskcomponents.NewConfigurationRisk(db, logger)
	attackSurfaceRisk := riskcomponents.NewAttackSurface(db, logger)
	complianceGapRisk := riskcomponents.NewComplianceGap(db, logger)
	contributorAnalyzer := cyberrisk.NewContributorAnalyzer(db, logger)
	recommendationEngine := cyberrisk.NewRecommendationEngine(db, logger)
	riskScorer := cyberrisk.NewRiskScorer(
		db,
		rdb,
		riskHistoryRepo,
		contributorAnalyzer,
		recommendationEngine,
		m,
		logger,
		vulnerabilityRisk,
		threatExposure,
		configurationRisk,
		attackSurfaceRisk,
		complianceGapRisk,
	)
	riskSnapshotSvc := cyberrisk.NewSnapshotService(db, riskScorer, riskHistoryRepo, producer, m, logger)
	riskSvc := service.NewRiskService(riskScorer, riskSnapshotSvc, riskHistoryRepo, vulnRepo, producer, logger)
	vulnerabilitySvc := service.NewVulnerabilityService(vulnRepo, producer, m, logger)
	mttrCalc := cyberdashboard.NewMTTRCalculator(db, m)
	dashboardSvc := service.NewDashboardService(
		cyberdashboard.NewCache(rdb),
		dashboardRepo,
		cyberdashboard.NewKPICalculator(db),
		cyberdashboard.NewTimelineCalculator(db),
		cyberdashboard.NewTrendCalculator(db),
		mttrCalc,
		cyberdashboard.NewWorkloadCalculator(db),
		cyberdashboard.NewMITREHeatmapCalculator(db),
		riskScorer,
		m,
		logger,
	)
	dspmClassifier := cyberdspm.NewDSPMClassifier()
	dspmPosture := cyberdspm.NewPostureAssessor()
	dspmDependency := cyberdspm.NewDependencyMapper(db)
	dspmScanner := cyberdspm.NewDSPMScanner(db, dspmRepo, dspmClassifier, dspmPosture, dspmDependency, logger)
	dspmSvc := service.NewDSPMService(dspmRepo, dspmScanner, dspmDependency, producer, logger)
	vcisoRecommender := cybervciso.NewRecommendationAggregator(db, recommendationEngine, logger)
	vcisoBriefing := cybervciso.NewBriefingGenerator(db, riskScorer, mttrCalc, vcisoRecommender, logger)
	vcisoReporter := cybervciso.NewReportGenerator(vcisoBriefing, logger)
	vcisoSvc := service.NewVCISOService(vcisoRepo, vcisoBriefing, vcisoRecommender, vcisoReporter, riskScorer, producer, logger)
	remediationAuditTrail := cyberremediation.NewAuditTrail(remediationAuditRepo, logger)
	remediationStrategies := map[model.RemediationType]remediationstrategy.RemediationStrategy{
		model.RemediationTypePatch:        remediationstrategy.NewPatchStrategy(db, logger),
		model.RemediationTypeConfigChange: remediationstrategy.NewConfigStrategy(db, logger),
		model.RemediationTypeBlockIP:      remediationstrategy.NewBlockStrategy(db, logger),
		model.RemediationTypeFirewallRule: remediationstrategy.NewBlockStrategy(db, logger),
		model.RemediationTypeIsolateAsset: remediationstrategy.NewIsolateStrategy(db, logger),
		model.RemediationTypeAccessRevoke: remediationstrategy.NewConfigStrategy(db, logger),
		model.RemediationTypeCertRenew:    remediationstrategy.NewConfigStrategy(db, logger),
		model.RemediationTypeCustom:       remediationstrategy.NewCustomStrategy(),
	}
	remediationExecutor := cyberremediation.NewRemediationExecutor(
		remediationStrategies,
		remediationAuditTrail,
		remediationRepo,
		alertRepo,
		vulnRepo,
		producer,
		logger,
	)
	remediationSvc := service.NewRemediationService(
		remediationRepo,
		remediationAuditRepo,
		assetRepo,
		remediationExecutor,
		remediationAuditTrail,
		producer,
		logger,
	)
	guard := events.NewIdempotencyGuard(rdb, 24*time.Hour)
	crossSuiteMetrics := events.NewCrossSuiteMetrics(svc.Metrics.Registry())
	dlqTracker := events.NewDLQTracker(rdb)

	// ── 12. Route registration ─────────────────────────────────────────────────
	svc.Router.Handle("/metrics", promhttp.HandlerFor(promGatherers, promhttp.HandlerOpts{}))
	svc.AdminRouter.Handle("/metrics", promhttp.HandlerFor(promGatherers, promhttp.HandlerOpts{}))

	jwtMgr, err := auth.NewJWTManager(cfg.Auth)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create JWT manager")
	}
	assetHandler := handler.NewAssetHandler(assetSvc, logger)
	ctemHandler := handler.NewCTEMHandler(ctemSvc, logger)
	ctemReportHandler := handler.NewCTEMReportHandler(ctemSvc, logger)
	alertHandler := handler.NewAlertHandler(alertSvc)
	ruleHandler := handler.NewRuleHandler(ruleSvc)
	threatHandler := handler.NewThreatHandler(threatSvc)
	mitreHandler := handler.NewMITREHandler(ruleSvc)
	riskHandler := handler.NewRiskHandler(riskSvc)
	dashboardHandler := handler.NewDashboardHandler(dashboardSvc)
	vulnerabilityHandler := handler.NewVulnerabilityHandler(vulnerabilitySvc)
	remediationHandler := handler.NewRemediationHandler(remediationSvc)
	dspmHandler := handler.NewDSPMHandler(dspmSvc)
	vcisoHandler := handler.NewVCISOHandler(vcisoSvc)
	handler.RegisterRoutes(
		svc.Router,
		assetHandler,
		alertHandler,
		ruleHandler,
		threatHandler,
		mitreHandler,
		ctemHandler,
		ctemReportHandler,
		riskHandler,
		dashboardHandler,
		vulnerabilityHandler,
		remediationHandler,
		dspmHandler,
		vcisoHandler,
		jwtMgr,
		rdb,
	)
	svc.Router.Get("/api/v1/internal/assets/owners", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Internal-Service") == "" {
			suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "internal access required", nil)
			return
		}

		tenantID, err := uuid.Parse(strings.TrimSpace(r.URL.Query().Get("tenant_id")))
		if err != nil {
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid tenant_id", nil)
			return
		}
		rawAssetIDs := r.URL.Query()["asset_id"]
		if len(rawAssetIDs) == 0 {
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "at least one asset_id is required", nil)
			return
		}

		userIDs := make([]string, 0, len(rawAssetIDs))
		seen := make(map[string]struct{}, len(rawAssetIDs))
		for _, rawAssetID := range rawAssetIDs {
			assetID, err := uuid.Parse(strings.TrimSpace(rawAssetID))
			if err != nil {
				suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid asset_id", nil)
				return
			}

			asset, err := assetRepo.GetByID(r.Context(), tenantID, assetID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					continue
				}
				logger.Error().Err(err).Str("tenant_id", tenantID.String()).Str("asset_id", assetID.String()).Msg("failed to resolve asset owner")
				suiteapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to resolve asset owners", nil)
				return
			}

			ownerUserID := ""
			if asset.Owner != nil {
				if parsed, parseErr := uuid.Parse(strings.TrimSpace(*asset.Owner)); parseErr == nil {
					ownerUserID = parsed.String()
				}
			}
			if ownerUserID == "" && asset.CreatedBy != nil {
				ownerUserID = asset.CreatedBy.String()
			}
			if ownerUserID == "" {
				continue
			}
			if _, ok := seen[ownerUserID]; ok {
				continue
			}
			seen[ownerUserID] = struct{}{}
			userIDs = append(userIDs, ownerUserID)
		}

		suiteapi.WriteJSON(w, http.StatusOK, map[string][]string{"user_ids": userIDs})
	})
	svc.Router.Get("/api/v1/admin/dlq/count", events.DLQCountHandler("cyber-service", dlqTracker, logger))

	// ── 13. Kafka consumer ─────────────────────────────────────────────────────
	var cyberConsumer *consumer.CyberConsumer
	if len(cfg.Kafka.Brokers) > 0 && cfg.Kafka.Brokers[0] != "" {
		kafkaConsumer, err := events.NewConsumer(cfg.Kafka, logger)
		if err != nil {
			logger.Warn().Err(err).Msg("Kafka consumer unavailable — event processing disabled")
		} else {
			kafkaConsumer.SetDeadLetterProducer(producer)
			kafkaConsumer.SetCrossSuiteMetrics(crossSuiteMetrics)
			kafkaConsumer.SetDLQTracker(dlqTracker, "cyber-service")
			cyberConsumer = consumer.NewCyberConsumer(assetSvc, detectionSvc, cyberCfg.SecurityEventTopic, kafkaConsumer, logger)
			_ = consumer.NewCTEMConsumer(ctemSvc, kafkaConsumer, logger)
			_ = consumer.NewRiskConsumer(riskSvc, dashboardSvc, rdb, kafkaConsumer, logger)
			_ = consumer.NewRemediationConsumer(remediationSvc, remediationRepo, ctemRemGroupRepo, ctemFindingRepo, kafkaConsumer, logger)
			dataEventConsumer := consumer.NewDataEventConsumer(alertSvc, dspmSvc, rdb, guard, producer, logger, crossSuiteMetrics)
			kafkaConsumer.Subscribe(events.Topics.IAMEvents, consumer.NewIAMEventConsumer(alertSvc, rdb, guard, producer, logger, crossSuiteMetrics))
			kafkaConsumer.Subscribe(events.Topics.DataSourceEvents, dataEventConsumer)
			kafkaConsumer.Subscribe(events.Topics.DarkDataEvents, dataEventConsumer)
			kafkaConsumer.Subscribe(events.Topics.FileEvents, consumer.NewFileEventConsumer(alertSvc, guard, producer, logger, crossSuiteMetrics))
		}
	}

	// ── 14. Scan scheduler ─────────────────────────────────────────────────────
	sched := scanner.NewScheduler(logger)
	// Add scheduled scans here via sched.Register(...)

	// ── 15. Start all components ───────────────────────────────────────────────
	g, gCtx := errgroup.WithContext(ctx)

	// Kafka consumer
	if cyberConsumer != nil {
		g.Go(func() error {
			err := cyberConsumer.Start(gCtx)
			if err != nil && !errors.Is(err, context.Canceled) {
				return err
			}
			return nil
		})
	}

	// Scheduler (no-op until scans are registered)
	g.Go(func() error {
		err := sched.Start(gCtx)
		if err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	})

	g.Go(func() error {
		err := riskSnapshotSvc.RunDailySnapshot(gCtx)
		if err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	})

	logger.Info().Int("port", bootstrapCfg.Port).Msg("cyber-service starting")
	runErr := svc.Run(ctx)
	cancel()
	if waitErr := g.Wait(); waitErr != nil {
		logger.Error().Err(waitErr).Msg("cyber background components stopped with error")
	}
	if runErr != nil && !errors.Is(runErr, context.Canceled) {
		logger.Error().Err(runErr).Msg("cyber-service stopped with error")
	}

	if cyberConsumer != nil {
		_ = cyberConsumer.Stop()
	}
	if producer != nil {
		_ = producer.Close()
	}

	logger.Info().Msg("cyber-service shutdown complete")
}

func buildBootstrapConfig(cfg *config.Config, cyberCfg *cyberconfig.Config) (*bootstrap.ServiceConfig, error) {
	redisURL, err := url.Parse(cyberCfg.RedisURL)
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
		Name:            "cyber-service",
		Version:         cfg.Observability.ServiceName,
		Environment:     envOr("ENVIRONMENT", "development"),
		Port:            mustParsePort(cyberCfg.HTTPPort, 8090),
		AdminPort:       cfg.Observability.MetricsPort,
		LogLevel:        cfg.Observability.LogLevel,
		DebugSampleRate: 100,
		ShutdownTimeout: cfg.Server.ShutdownTimeout,
		ReadTimeout:     cfg.Server.ReadTimeout,
		WriteTimeout:    cfg.Server.WriteTimeout,
		Tracing:         bootstrapTracingConfig(cfg),
		EnablePprof:     false,
		DB: &bootstrap.DBConfig{
			URL:               cyberCfg.DBURL,
			MinConns:          cyberCfg.DBMinConn,
			MaxConns:          cyberCfg.DBMaxConn,
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
			Brokers: cyberCfg.KafkaBrokers,
			GroupID: cyberCfg.KafkaGroupID,
		},
	}, nil
}

func bootstrapTracingConfig(cfg *config.Config) tracing.TracerConfig {
	return tracing.TracerConfig{
		Enabled:     cfg.Observability.OTLPEndpoint != "",
		Endpoint:    cfg.Observability.OTLPEndpoint,
		ServiceName: "cyber-service",
		Version:     cfg.Observability.ServiceName,
		Environment: envOr("ENVIRONMENT", "development"),
		SampleRate:  1,
		Insecure:    true,
	}
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
