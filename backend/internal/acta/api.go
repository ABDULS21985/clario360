package acta

import (
	"fmt"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	aigovmiddleware "github.com/clario360/platform/internal/aigovernance/middleware"
	"github.com/clario360/platform/internal/acta/ai"
	"github.com/clario360/platform/internal/acta/handler"
	actametrics "github.com/clario360/platform/internal/acta/metrics"
	"github.com/clario360/platform/internal/acta/repository"
	"github.com/clario360/platform/internal/acta/service"
	"github.com/clario360/platform/internal/auth"
	workflowrepo "github.com/clario360/platform/internal/workflow/repository"
)

type Dependencies struct {
	DB                *pgxpool.Pool
	Redis             *redis.Client
	Publisher         service.Publisher
	Logger            zerolog.Logger
	Registerer        prometheus.Registerer
	DashboardCacheTTL time.Duration
	KafkaTopic        string
	WorkflowDefRepo   *workflowrepo.DefinitionRepository
	WorkflowInstRepo  *workflowrepo.InstanceRepository
	PredictionLogger  *aigovmiddleware.PredictionLogger
}

type Application struct {
	Store *repository.Store

	Metrics *actametrics.Metrics

	CommitteeService  *service.CommitteeService
	MeetingService    *service.MeetingService
	AgendaService     *service.AgendaService
	MinutesService    *service.MinutesService
	ActionItemService *service.ActionItemService
	ComplianceService *service.ComplianceService
	DashboardService  *service.DashboardService

	CommitteeHandler  *handler.CommitteeHandler
	MeetingHandler    *handler.MeetingHandler
	AgendaHandler     *handler.AgendaHandler
	MinutesHandler    *handler.MinutesHandler
	ActionItemHandler *handler.ActionItemHandler
	ComplianceHandler *handler.ComplianceHandler
	DashboardHandler  *handler.DashboardHandler
}

func NewApplication(deps Dependencies) (*Application, error) {
	store := repository.NewStore(deps.DB, deps.Logger)
	reg := deps.Registerer
	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	metrics := actametrics.New(reg)
	generator, err := ai.NewMinutesGenerator()
	if err != nil {
		return nil, fmt.Errorf("create minutes generator: %w", err)
	}

	app := &Application{
		Store: store,

		Metrics: metrics,
	}

	app.CommitteeService = service.NewCommitteeService(store, deps.Publisher, metrics, deps.Logger)
	app.MeetingService = service.NewMeetingService(store, deps.Publisher, metrics, deps.KafkaTopic, deps.WorkflowDefRepo, deps.WorkflowInstRepo, deps.Logger)
	app.AgendaService = service.NewAgendaService(store, deps.Publisher, metrics, deps.Logger)
	app.MinutesService = service.NewMinutesService(store, generator, deps.Publisher, metrics, deps.Logger, deps.PredictionLogger)
	app.ActionItemService = service.NewActionItemService(store, deps.Publisher, metrics, deps.Logger)
	app.ComplianceService = service.NewComplianceService(store, deps.Publisher, metrics, deps.Logger)
	app.DashboardService = service.NewDashboardService(store, deps.Redis, deps.DashboardCacheTTL, deps.Logger)

	app.CommitteeHandler = handler.NewCommitteeHandler(app.CommitteeService, deps.Logger)
	app.MeetingHandler = handler.NewMeetingHandler(app.MeetingService, deps.Logger)
	app.AgendaHandler = handler.NewAgendaHandler(app.AgendaService, deps.Logger)
	app.MinutesHandler = handler.NewMinutesHandler(app.MinutesService, deps.Logger)
	app.ActionItemHandler = handler.NewActionItemHandler(app.ActionItemService, deps.Logger)
	app.ComplianceHandler = handler.NewComplianceHandler(app.ComplianceService, deps.Logger)
	app.DashboardHandler = handler.NewDashboardHandler(app.DashboardService, deps.Logger)

	return app, nil
}

func (a *Application) RegisterRoutes(r chi.Router, jwtMgr *auth.JWTManager, rdb *redis.Client, rateLimitPerMinute int) {
	handler.RegisterRoutes(r, handler.RouteDependencies{
		Committee:       a.CommitteeHandler,
		Meeting:         a.MeetingHandler,
		Agenda:          a.AgendaHandler,
		Minutes:         a.MinutesHandler,
		ActionItem:      a.ActionItemHandler,
		Compliance:      a.ComplianceHandler,
		Dashboard:       a.DashboardHandler,
		JWTManager:      jwtMgr,
		Redis:           rdb,
		RateLimitPerMin: rateLimitPerMinute,
	})
}
