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

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	actarepo "github.com/clario360/platform/internal/acta/repository"
	actaservice "github.com/clario360/platform/internal/acta/service"
	"github.com/clario360/platform/internal/aigovernance"
	aigovconsumer "github.com/clario360/platform/internal/aigovernance/consumer"
	aigovintegration "github.com/clario360/platform/internal/aigovernance/integration"
	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/cyber/classifier"
	cyberconfig "github.com/clario360/platform/internal/cyber/config"
	"github.com/clario360/platform/internal/cyber/consumer"
	cyberctem "github.com/clario360/platform/internal/cyber/ctem"
	cyberdashboard "github.com/clario360/platform/internal/cyber/dashboard"
	"github.com/clario360/platform/internal/cyber/detection"
	cyberdspm "github.com/clario360/platform/internal/cyber/dspm"
	cybercompliance "github.com/clario360/platform/internal/cyber/dspm/compliance"
	cybercontinuous "github.com/clario360/platform/internal/cyber/dspm/continuous"
	cybershadow "github.com/clario360/platform/internal/cyber/dspm/shadow"
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
	uebacollector "github.com/clario360/platform/internal/cyber/ueba/collector"
	uebaengine "github.com/clario360/platform/internal/cyber/ueba/engine"
	uebahandler "github.com/clario360/platform/internal/cyber/ueba/handler"
	uebarepository "github.com/clario360/platform/internal/cyber/ueba/repository"
	uebaservice "github.com/clario360/platform/internal/cyber/ueba/service"
	cybervciso "github.com/clario360/platform/internal/cyber/vciso"
	vcisochatengine "github.com/clario360/platform/internal/cyber/vciso/chat/engine"
	vcisochathandler "github.com/clario360/platform/internal/cyber/vciso/chat/handler"
	vcisorepository "github.com/clario360/platform/internal/cyber/vciso/chat/repository"
	vcisotools "github.com/clario360/platform/internal/cyber/vciso/chat/tools"
	datarepo "github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/events"
	lexrepo "github.com/clario360/platform/internal/lex/repository"
	lexservice "github.com/clario360/platform/internal/lex/service"
	platformmw "github.com/clario360/platform/internal/middleware"
	bootstrap "github.com/clario360/platform/internal/observability/bootstrap"
	"github.com/clario360/platform/internal/observability/tracing"
	"github.com/clario360/platform/internal/rca"
	"github.com/clario360/platform/internal/suiteapi"
	visusrepo "github.com/clario360/platform/internal/visus/repository"
	visusservice "github.com/clario360/platform/internal/visus/service"
	workflowrepo "github.com/clario360/platform/internal/workflow/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	uebaMetrics := uebaengine.NewMetrics(m.Registry)
	vcisoChatMetrics := vcisochatengine.NewMetrics(m.Registry)
	promGatherers := prometheus.Gatherers{svc.Metrics.Registry(), m.Registry}

	// ── 5. Kafka producer ──────────────────────────────────────────────────────
	var producer *events.Producer
	if len(cfg.Kafka.Brokers) > 0 && cfg.Kafka.Brokers[0] != "" {
		producer, err = events.NewProducer(cfg.Kafka, logger)
		if err != nil {
			logger.Warn().Err(err).Msg("Kafka producer unavailable — events will not be published")
		}
	}
	aiRuntime, aiErr := aigovintegration.NewCyberRuntime(ctx, cfg, svc.Metrics.Registry(), producer, logger)
	if aiErr != nil {
		logger.Warn().Err(aiErr).Msg("ai governance runtime unavailable for cyber-service")
	} else {
		defer aiRuntime.Close()
	}
	var platformPool *pgxpool.Pool
	platformPoolOwned := false
	if aiRuntime != nil && aiRuntime.Pool != nil {
		platformPool = aiRuntime.Pool
	} else {
		platformDSN := aigovernance.BuildPlatformCoreDSN(cfg.Database)
		poolCfg, parseErr := pgxpool.ParseConfig(platformDSN)
		if parseErr != nil {
			logger.Warn().Err(parseErr).Msg("platform core pool unavailable for ueba")
		} else {
			poolCfg.MinConns = 1
			poolCfg.MaxConns = 5
			poolCfg.MaxConnLifetime = cfg.Database.ConnMaxLifetime
			poolCfg.MaxConnIdleTime = 5 * time.Minute
			poolCfg.HealthCheckPeriod = time.Minute
			pool, openErr := pgxpool.NewWithConfig(ctx, poolCfg)
			if openErr != nil {
				logger.Warn().Err(openErr).Msg("platform core pool unavailable for ueba")
			} else if pingErr := pool.Ping(ctx); pingErr != nil {
				logger.Warn().Err(pingErr).Msg("platform core pool unavailable for ueba")
				pool.Close()
			} else {
				platformPool = pool
				platformPoolOwned = true
			}
		}
	}
	if platformPoolOwned {
		defer platformPool.Close()
	}
	dataPool, dataPoolOwned := openOptionalDBPool(ctx, envOr("DATA_DB_URL", buildPostgresURL(cfg.Database, "data_db")), cfg, logger, "data")
	if dataPoolOwned {
		defer dataPool.Close()
	}
	actaPool, actaPoolOwned := openOptionalDBPool(ctx, envOr("ACTA_DB_URL", buildPostgresURL(cfg.Database, "acta_db")), cfg, logger, "acta")
	if actaPoolOwned {
		defer actaPool.Close()
	}
	lexPool, lexPoolOwned := openOptionalDBPool(ctx, envOr("LEX_DB_URL", buildPostgresURL(cfg.Database, "lex_db")), cfg, logger, "lex")
	if lexPoolOwned {
		defer lexPool.Close()
	}
	visusPool, visusPoolOwned := openOptionalDBPool(ctx, envOr("VISUS_DB_URL", buildPostgresURL(cfg.Database, "visus_db")), cfg, logger, "visus")
	if visusPoolOwned {
		defer visusPool.Close()
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
	vcisoConversationRepo := vcisorepository.NewConversationRepository(db, logger)
	uebaProfileRepo := uebarepository.NewProfileRepository(db, logger)
	uebaEventRepo := uebarepository.NewEventRepository(db, logger)
	uebaAlertRepo := uebarepository.NewAlertRepository(db, logger)

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
	if err := uebaEventRepo.EnsurePartitions(ctx); err != nil {
		logger.Warn().Err(err).Msg("failed to ensure ueba event partitions")
	}
	uebaConfigStore := uebaengine.NewConfigStore(ctx, rdb, logger)
	var uebaCollector *uebacollector.AccessEventCollector
	if platformPool != nil {
		uebaCollector = uebacollector.New(platformPool, rdb, uebaEventRepo, logger)
	} else {
		logger.Warn().Msg("platform core pool unavailable - ueba collection disabled")
	}
	uebaEngine := uebaengine.NewEngine(
		uebaCollector,
		uebaProfileRepo,
		uebaEventRepo,
		uebaAlertRepo,
		alertRepo,
		nil,
		producer,
		uebaConfigStore,
		rdb,
		uebaMetrics,
		logger,
	)
	if aiRuntime != nil {
		uebaEngine = uebaengine.NewEngine(
			uebaCollector,
			uebaProfileRepo,
			uebaEventRepo,
			uebaAlertRepo,
			alertRepo,
			aiRuntime.PredictionLogger,
			producer,
			uebaConfigStore,
			rdb,
			uebaMetrics,
			logger,
		)
	}
	uebaSvc := uebaservice.NewUEBAService(db, uebaEngine, uebaMetrics, uebaProfileRepo, uebaEventRepo, uebaAlertRepo, producer, logger)
	var uebaScheduler *uebaengine.Scheduler
	if uebaCollector != nil {
		uebaScheduler = uebaengine.NewScheduler(uebaEngine, uebaProfileRepo, uebaCollector, logger)
	}
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
	if aiRuntime != nil {
		assetSvc.SetPredictionLogger(aiRuntime.PredictionLogger)
		detectionEngine.SetPredictionLogger(aiRuntime.PredictionLogger)
		ctemEngine.SetPredictionLogger(aiRuntime.PredictionLogger)
		riskScorer.SetPredictionLogger(aiRuntime.PredictionLogger)
	}
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
	dspmTagger := cybercompliance.NewComplianceTagger()
	dspmShadowDetector := cybershadow.NewDetector(db, dataPool, logger)
	continuousDSPM := cybercontinuous.NewEngine(
		db,
		dataPool,
		dspmRepo,
		alertRepo,
		dspmClassifier,
		dspmTagger,
		dspmShadowDetector,
		producer,
		cybercontinuous.DefaultConfig(),
		logger,
	)
	dspmSvc := service.NewDSPMService(dspmRepo, dspmScanner, dspmDependency, dspmTagger, dspmShadowDetector, producer, logger)
	rcaEngine := rca.NewEngine(db, dataPool, logger)
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
	var (
		dataPipelineRepo    *datarepo.PipelineRepository
		dataPipelineRunRepo *datarepo.PipelineRunRepository
		actaStore           *actarepo.Store
		actaComplianceSvc   *actaservice.ComplianceService
		lexContractRepo     *lexrepo.ContractRepository
		lexAlertRepo        *lexrepo.AlertRepository
		lexDocumentRepo     *lexrepo.DocumentRepository
		lexClauseRepo       *lexrepo.ClauseRepository
		lexComplianceRepo   *lexrepo.ComplianceRepository
		lexComplianceSvc    *lexservice.ComplianceService
		visusDashboardRepo  *visusrepo.DashboardRepository
		visusWidgetRepo     *visusrepo.WidgetRepository
		visusKPIRepo        *visusrepo.KPIRepository
		visusSnapshotRepo   *visusrepo.KPISnapshotRepository
		visusAlertRepo      *visusrepo.AlertRepository
		visusDashboardSvc   *visusservice.DashboardService
		visusWidgetSvc      *visusservice.WidgetService
	)
	if dataPool != nil {
		dataPipelineRepo = datarepo.NewPipelineRepository(dataPool, logger)
		dataPipelineRunRepo = datarepo.NewPipelineRunRepository(dataPool, logger)
	}
	if actaPool != nil {
		actaStore = actarepo.NewStore(actaPool, logger)
		actaComplianceSvc = actaservice.NewComplianceService(actaStore, producer, nil, logger)
	}
	if lexPool != nil {
		lexContractRepo = lexrepo.NewContractRepository(lexPool, logger)
		lexAlertRepo = lexrepo.NewAlertRepository(lexPool, logger)
		lexDocumentRepo = lexrepo.NewDocumentRepository(lexPool, logger)
		lexClauseRepo = lexrepo.NewClauseRepository(lexPool, logger)
		lexComplianceRepo = lexrepo.NewComplianceRepository(lexPool, logger)
		lexComplianceSvc = lexservice.NewComplianceService(
			lexPool,
			lexContractRepo,
			lexClauseRepo,
			lexDocumentRepo,
			lexComplianceRepo,
			lexAlertRepo,
			producer,
			nil,
			events.Topics.LexEvents,
			logger,
		)
	}
	if visusPool != nil {
		visusDashboardRepo = visusrepo.NewDashboardRepository(visusPool, logger)
		visusWidgetRepo = visusrepo.NewWidgetRepository(visusPool, logger)
		visusKPIRepo = visusrepo.NewKPIRepository(visusPool, logger)
		visusSnapshotRepo = visusrepo.NewKPISnapshotRepository(visusPool, logger)
		visusAlertRepo = visusrepo.NewAlertRepository(visusPool, logger)
		visusDashboardSvc = visusservice.NewDashboardService(visusDashboardRepo, visusWidgetRepo, producer, nil, logger)
		visusWidgetSvc = visusservice.NewWidgetService(
			visusDashboardRepo,
			visusWidgetRepo,
			visusKPIRepo,
			visusSnapshotRepo,
			visusAlertRepo,
			nil,
			nil,
			logger,
		)
	}
	vcisoToolDeps := &vcisotools.Dependencies{
		CyberDB:               db,
		AlertService:          alertSvc,
		AlertRepo:             alertRepo,
		AssetService:          assetSvc,
		AssetRepo:             assetRepo,
		VulnerabilityService:  vulnerabilitySvc,
		RiskService:           riskSvc,
		RuleService:           ruleSvc,
		UEBAService:           uebaSvc,
		VCISOService:          vcisoSvc,
		RemediationService:    remediationSvc,
		DataPool:              dataPool,
		DataPipelineRepo:      dataPipelineRepo,
		DataPipelineRunRepo:   dataPipelineRunRepo,
		ActaPool:              actaPool,
		ActaStore:             actaStore,
		ActaComplianceService: actaComplianceSvc,
		LexPool:               lexPool,
		LexContractRepo:       lexContractRepo,
		LexAlertRepo:          lexAlertRepo,
		LexDocumentRepo:       lexDocumentRepo,
		LexClauseRepo:         lexClauseRepo,
		LexComplianceRepo:     lexComplianceRepo,
		LexComplianceService:  lexComplianceSvc,
		VisusPool:             visusPool,
		VisusDashboardRepo:    visusDashboardRepo,
		VisusWidgetRepo:       visusWidgetRepo,
		VisusKPIRepo:          visusKPIRepo,
		VisusSnapshotRepo:     visusSnapshotRepo,
		VisusAlertRepo:        visusAlertRepo,
		VisusDashboardService: visusDashboardSvc,
		VisusWidgetService:    visusWidgetSvc,
		Producer:              producer,
		Logger:                logger,
	}
	vcisoClassifier := vcisochatengine.NewIntentClassifier()
	vcisoContextManager := vcisochatengine.NewContextManager(func() time.Time { return time.Now().UTC() }, 30*time.Minute)
	vcisoChatEngine := vcisochatengine.NewEngine(
		vcisoClassifier,
		vcisochatengine.NewEntityExtractor(func() time.Time { return time.Now().UTC() }),
		vcisoContextManager,
		vcisotools.NewRegistry(vcisoToolDeps),
		vcisochatengine.NewToolRouter(vcisotools.NewRegistry(vcisoToolDeps), vcisoChatMetrics, logger),
		vcisochatengine.NewResponseFormatter(),
		vcisochatengine.NewSuggestionEngine(vcisoToolDeps, vcisoChatMetrics, logger),
		vcisoConversationRepo,
		nil,
		producer,
		vcisoChatMetrics,
		logger,
	)
	if aiRuntime != nil && aiRuntime.PredictionLogger != nil {
		registry := vcisotools.NewRegistry(vcisoToolDeps)
		vcisoChatEngine = vcisochatengine.NewEngine(
			vcisoClassifier,
			vcisochatengine.NewEntityExtractor(func() time.Time { return time.Now().UTC() }),
			vcisoContextManager,
			registry,
			vcisochatengine.NewToolRouter(registry, vcisoChatMetrics, logger),
			vcisochatengine.NewResponseFormatter(),
			vcisochatengine.NewSuggestionEngine(vcisoToolDeps, vcisoChatMetrics, logger),
			vcisoConversationRepo,
			aiRuntime.PredictionLogger,
			producer,
			vcisoChatMetrics,
			logger,
		)
	}
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
	uebaHTTPHandler := uebahandler.NewUEBAHandler(uebaSvc)
	rcaHandler := rca.NewHandler(rcaEngine, logger)
	vcisoChatHandler := vcisochathandler.NewChatHandler(vcisoChatEngine, vcisoConversationRepo, logger)
	vcisoWSHandler := vcisochathandler.NewWebSocketHandler(vcisoChatEngine, jwtMgr, logger)
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
		uebaHTTPHandler,
		jwtMgr,
		rdb,
	)
	svc.Router.Group(func(r chi.Router) {
		r.Use(platformmw.Auth(jwtMgr))
		r.Use(platformmw.Tenant)
		rcaHandler.RegisterRoutes(r)
	})
	vcisochathandler.RegisterRoutes(svc.Router, vcisochathandler.RouteDeps{
		ChatHandler:          vcisoChatHandler,
		WSHandler:            vcisoWSHandler,
		JWTManager:           jwtMgr,
		Redis:                rdb,
		Logger:               logger,
		VCISOBriefing:        vcisoHandler.Briefing,
		VCISOBriefingHistory: vcisoHandler.BriefingHistory,
		VCISORecommendations: vcisoHandler.Recommendations,
		VCISOReport:          vcisoHandler.Report,
		VCISOPostureSummary:  vcisoHandler.PostureSummary,
	})
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
			kafkaConsumer.Subscribe(events.Topics.PipelineEvents, events.EventHandlerFunc(continuousDSPM.HandlePipelineEvent))
			kafkaConsumer.Subscribe(events.Topics.FileEvents, consumer.NewFileEventConsumer(alertSvc, guard, producer, logger, crossSuiteMetrics))
			if aiRuntime != nil && aiRuntime.PredictionLogger != nil {
				kafkaConsumer.Subscribe(events.Topics.AIEvents, aigovconsumer.NewCacheInvalidationConsumer(aiRuntime.PredictionLogger, logger))
			}
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
		err := continuousDSPM.Start(gCtx)
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
	if uebaScheduler != nil {
		g.Go(func() error {
			err := uebaScheduler.Start(gCtx)
			if err != nil && !errors.Is(err, context.Canceled) {
				return err
			}
			return nil
		})
	}

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

func buildPostgresURL(cfg config.DatabaseConfig, dbName string) string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Path:   dbName,
	}
	q := u.Query()
	q.Set("sslmode", cfg.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}

func openOptionalDBPool(ctx context.Context, dsn string, cfg *config.Config, logger zerolog.Logger, name string) (*pgxpool.Pool, bool) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, false
	}
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		logger.Warn().Err(err).Str("db", name).Msg("failed to parse optional suite database dsn")
		return nil, false
	}
	minConns := cfg.Database.MaxIdleConns
	if minConns < 1 {
		minConns = 1
	}
	maxConns := cfg.Database.MaxOpenConns
	if maxConns < minConns {
		maxConns = minConns
	}
	poolCfg.MinConns = int32(minConns)
	poolCfg.MaxConns = int32(maxConns)
	poolCfg.MaxConnLifetime = cfg.Database.ConnMaxLifetime
	poolCfg.MaxConnIdleTime = 5 * time.Minute
	poolCfg.HealthCheckPeriod = time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		logger.Warn().Err(err).Str("db", name).Msg("failed to open optional suite database pool")
		return nil, false
	}
	if err := pool.Ping(ctx); err != nil {
		logger.Warn().Err(err).Str("db", name).Msg("failed to ping optional suite database pool")
		pool.Close()
		return nil, false
	}
	return pool, true
}
