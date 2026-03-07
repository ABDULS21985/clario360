package visus

import (
	"fmt"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/visus/aggregator"
	visusalert "github.com/clario360/platform/internal/visus/alert"
	visusconfig "github.com/clario360/platform/internal/visus/config"
	"github.com/clario360/platform/internal/visus/consumer"
	"github.com/clario360/platform/internal/visus/handler"
	"github.com/clario360/platform/internal/visus/kpi"
	"github.com/clario360/platform/internal/visus/metrics"
	"github.com/clario360/platform/internal/visus/report"
	"github.com/clario360/platform/internal/visus/repository"
	"github.com/clario360/platform/internal/visus/service"
)

type Dependencies struct {
	DB         *pgxpool.Pool
	Redis      *redis.Client
	Publisher  *events.Producer
	Logger     zerolog.Logger
	Registerer prometheus.Registerer
	Config     *visusconfig.Config
	JWTManager *auth.JWTManager
}

type Application struct {
	Store            *repository.Store
	Metrics          *metrics.Metrics
	DashboardService *service.DashboardService
	WidgetService    *service.WidgetService
	KPIService       *service.KPIService
	AlertService     *service.AlertService
	ReportService    *service.ReportService
	ExecutiveService *service.ExecutiveService
	KPIScheduler     *kpi.Scheduler
	ReportScheduler  *report.Scheduler
	Consumer         *consumer.VisusConsumer

	dashboardHandler *handler.DashboardHandler
	widgetHandler    *handler.WidgetHandler
	kpiHandler       *handler.KPIHandler
	alertHandler     *handler.AlertHandler
	reportHandler    *handler.ReportHandler
	executiveHandler *handler.ExecutiveHandler
	cfg              *visusconfig.Config
	logger           zerolog.Logger
}

func NewApplication(deps Dependencies) (*Application, error) {
	if deps.DB == nil {
		return nil, fmt.Errorf("database pool is required")
	}
	cfg := deps.Config
	if cfg == nil {
		cfg = visusconfig.Default()
	}
	reg := deps.Registerer
	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	store := repository.NewStore(deps.DB, deps.Logger)
	appMetrics := metrics.New(reg)

	tokenProvider := aggregator.NewServiceTokenProvider(cfg.ServiceAccountToken, deps.JWTManager, defaultServiceUserID(cfg.ServiceAccountUserID), cfg.ServiceAccountEmail, cfg.ServiceTokenTTL)
	suiteCache := aggregator.NewSuiteCache(deps.Redis, store.SuiteCache, cfg.SuiteCacheTTL, deps.Logger)
	suiteClient := aggregator.NewSuiteClient(map[string]string{
		"cyber": cfg.SuiteCyberURL,
		"data":  cfg.SuiteDataURL,
		"acta":  cfg.SuiteActaURL,
		"lex":   cfg.SuiteLexURL,
	}, tokenProvider, suiteCache, cfg.SuiteTimeout, cfg.SuiteMaxRetries, cfg.CircuitThreshold, cfg.CircuitReset, appMetrics, deps.Logger)
	crossAggregator := aggregator.NewCrossSuiteAggregator(suiteClient, store.KPISnapshots, store.Alerts, appMetrics)
	deduplicator := visusalert.NewDeduplicator(store.Alerts)
	alertGenerator := visusalert.NewGenerator(store.Alerts, deduplicator, deps.Publisher, deps.Logger)
	correlator := visusalert.NewCorrelator(store.KPIs, store.KPISnapshots, store.Alerts, suiteClient)
	escalator := visusalert.NewEscalator(store.Alerts)
	kpiEngine := kpi.NewEngine(kpi.NewFetcher(suiteClient), kpi.NewCalculator(), kpi.NewThresholdEvaluator(), store.KPISnapshots, store.KPIs, alertGenerator, appMetrics, deps.Logger)
	reportGenerator := report.NewGenerator(store.Reports, store.ReportSnapshots, store.KPIs, store.KPISnapshots, suiteClient, deps.Publisher, deps.Logger)

	app := &Application{
		Store:            store,
		Metrics:          appMetrics,
		DashboardService: service.NewDashboardService(store.Dashboards, store.Widgets, deps.Publisher, appMetrics, deps.Logger),
		WidgetService:    service.NewWidgetService(store.Dashboards, store.Widgets, store.KPIs, store.KPISnapshots, store.Alerts, suiteClient, appMetrics, deps.Logger),
		KPIService:       service.NewKPIService(store.KPIs, store.KPISnapshots, kpiEngine, deps.Publisher, appMetrics, deps.Logger),
		AlertService:     service.NewAlertService(store.Alerts, alertGenerator, correlator, escalator, deps.Publisher, appMetrics, deps.Logger),
		ReportService:    service.NewReportService(store.Reports, store.ReportSnapshots, reportGenerator, appMetrics, deps.Logger),
		ExecutiveService: service.NewExecutiveService(crossAggregator, deps.Publisher, appMetrics, deps.Logger),
		KPIScheduler:     kpi.NewScheduler(kpiEngine, store.KPIs, cfg.SchedulerInterval, deps.Logger),
		ReportScheduler:  report.NewScheduler(store.Reports, reportGenerator, cfg.ReportSchedulerInterval, deps.Logger),
		Consumer:         consumer.NewVisusConsumer(deps.Logger),
		cfg:              cfg,
		logger:           deps.Logger,
	}

	app.dashboardHandler = handler.NewDashboardHandler(app.DashboardService, deps.Logger)
	app.widgetHandler = handler.NewWidgetHandler(app.WidgetService, deps.Logger)
	app.kpiHandler = handler.NewKPIHandler(app.KPIService, deps.Logger)
	app.alertHandler = handler.NewAlertHandler(app.AlertService, deps.Logger)
	app.reportHandler = handler.NewReportHandler(app.ReportService, deps.Logger)
	app.executiveHandler = handler.NewExecutiveHandler(app.ExecutiveService, deps.Logger)
	app.Consumer = app.Consumer.WithAlertService(app.AlertService)

	return app, nil
}

func (a *Application) RegisterRoutes(r chi.Router, jwtMgr *auth.JWTManager, rdb *redis.Client, rateLimitPerMinute int) {
	handler.RegisterRoutes(r, handler.RouteDependencies{
		Dashboard:       a.dashboardHandler,
		Widget:          a.widgetHandler,
		KPI:             a.kpiHandler,
		Alert:           a.alertHandler,
		Report:          a.reportHandler,
		Executive:       a.executiveHandler,
		JWTManager:      jwtMgr,
		Redis:           rdb,
		RateLimitPerMin: rateLimitPerMinute,
	})
}

func defaultServiceUserID(raw string) string {
	if _, err := uuid.Parse(raw); err == nil {
		return raw
	}
	return "00000000-0000-0000-0000-000000000360"
}
