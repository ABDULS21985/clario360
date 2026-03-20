package lex

import (
	"fmt"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	aigovmiddleware "github.com/clario360/platform/internal/aigovernance/middleware"
	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/lex/analyzer"
	lexconfig "github.com/clario360/platform/internal/lex/config"
	"github.com/clario360/platform/internal/lex/handler"
	"github.com/clario360/platform/internal/lex/metrics"
	"github.com/clario360/platform/internal/lex/repository"
	"github.com/clario360/platform/internal/lex/service"
	workflowrepo "github.com/clario360/platform/internal/workflow/repository"
)

type Dependencies struct {
	DB                 *pgxpool.Pool
	Redis              *redis.Client
	Publisher          *events.Producer
	Logger             zerolog.Logger
	Registerer         prometheus.Registerer
	WorkflowDefRepo    *workflowrepo.DefinitionRepository
	WorkflowInstRepo   *workflowrepo.InstanceRepository
	WorkflowTaskRepo   *workflowrepo.TaskRepository
	Config             *lexconfig.Config
	DashboardCacheTTL  time.Duration
	OrgJurisdiction    string
	KafkaTopic         string
	PredictionLogger   *aigovmiddleware.PredictionLogger
}

type Application struct {
	Store              *repository.Store
	Metrics            *metrics.Metrics
	ClauseExtractor    *analyzer.ClauseExtractor
	RiskAnalyzer       *analyzer.RiskAnalyzer
	ContractService    *service.ContractService
	ClauseService      *service.ClauseService
	DocumentService    *service.DocumentService
	ComplianceService  *service.ComplianceService
	WorkflowService    *service.WorkflowService
	DashboardService   *service.DashboardService
	ContractHandler    *handler.ContractHandler
	ClauseHandler      *handler.ClauseHandler
	DocumentHandler    *handler.DocumentHandler
	ComplianceHandler  *handler.ComplianceHandler
	DashboardHandler   *handler.DashboardHandler
}

func NewApplication(deps Dependencies) (*Application, error) {
	if deps.DB == nil {
		return nil, fmt.Errorf("database pool is required")
	}
	reg := deps.Registerer
	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	cfg := deps.Config
	if cfg == nil {
		cfg = lexconfig.Default()
	}
	store := repository.NewStore(deps.DB, deps.Logger)
	appMetrics := metrics.New(reg)

	recommendationEngine := analyzer.NewRecommendationEngine(deps.OrgJurisdiction)
	clauseExtractor := analyzer.NewClauseExtractor(recommendationEngine)
	riskAnalyzer := analyzer.NewRiskAnalyzer(
		clauseExtractor,
		analyzer.NewMissingClauseDetector(),
		analyzer.NewEntityExtractor(),
		analyzer.NewComplianceChecker(deps.OrgJurisdiction),
		recommendationEngine,
		appMetrics,
	)

	publisher := eventsPublisher(deps.Publisher)
	workflowService := service.NewWorkflowService(
		deps.DB,
		deps.WorkflowDefRepo,
		deps.WorkflowInstRepo,
		deps.WorkflowTaskRepo,
		store.Contracts,
		publisher,
		appMetrics,
		deps.KafkaTopic,
		deps.Logger,
	)
	contractService := service.NewContractService(
		deps.DB,
		store.Contracts,
		store.Clauses,
		store.Documents,
		store.Compliance,
		store.Alerts,
		workflowService,
		riskAnalyzer,
		publisher,
		appMetrics,
		deps.KafkaTopic,
		deps.Logger,
		deps.PredictionLogger,
	)
	clauseService := service.NewClauseService(store.Contracts, store.Clauses, publisher, appMetrics, deps.KafkaTopic, deps.Logger)
	documentService := service.NewDocumentService(deps.DB, store.Contracts, store.Documents, publisher, appMetrics, deps.KafkaTopic, deps.Logger)
	complianceService := service.NewComplianceService(
		deps.DB,
		store.Contracts,
		store.Clauses,
		store.Documents,
		store.Compliance,
		store.Alerts,
		publisher,
		appMetrics,
		deps.KafkaTopic,
		deps.Logger,
	)
	dashboardService := service.NewDashboardService(
		deps.DB,
		deps.Redis,
		store.Contracts,
		store.Documents,
		store.Alerts,
		complianceService,
		appMetrics,
		deps.Logger,
		deps.DashboardCacheTTL,
	)

	app := &Application{
		Store:             store,
		Metrics:           appMetrics,
		ClauseExtractor:   clauseExtractor,
		RiskAnalyzer:      riskAnalyzer,
		ContractService:   contractService,
		ClauseService:     clauseService,
		DocumentService:   documentService,
		ComplianceService: complianceService,
		WorkflowService:   workflowService,
		DashboardService:  dashboardService,
	}

	app.ContractHandler = handler.NewContractHandler(contractService, workflowService, deps.Logger)
	app.ClauseHandler = handler.NewClauseHandler(clauseService, deps.Logger)
	app.DocumentHandler = handler.NewDocumentHandler(documentService, deps.Logger)
	app.ComplianceHandler = handler.NewComplianceHandler(complianceService, deps.Logger)
	app.DashboardHandler = handler.NewDashboardHandler(dashboardService, deps.Logger)

	return app, nil
}

func (a *Application) RegisterRoutes(r chi.Router, jwtMgr *auth.JWTManager, rdb *redis.Client, rateLimitPerMinute int) {
	handler.RegisterRoutes(r, handler.RouteDependencies{
		Contract:        a.ContractHandler,
		Clause:          a.ClauseHandler,
		Document:        a.DocumentHandler,
		Compliance:      a.ComplianceHandler,
		Dashboard:       a.DashboardHandler,
		JWTManager:      jwtMgr,
		Redis:           rdb,
		RateLimitPerMin: rateLimitPerMinute,
	})
}

func eventsPublisher(producer *events.Producer) service.Publisher {
	if producer == nil {
		return nil
	}
	return producer
}

func mustUUID(raw string) uuid.UUID {
	return uuid.MustParse(raw)
}
