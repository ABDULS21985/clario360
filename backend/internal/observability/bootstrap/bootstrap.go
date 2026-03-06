package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"

	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/middleware"
	"github.com/clario360/platform/internal/observability/health"
	"github.com/clario360/platform/internal/observability/logger"
	"github.com/clario360/platform/internal/observability/metrics"
	"github.com/clario360/platform/internal/observability/profiling"
	"github.com/clario360/platform/internal/observability/tracing"
)

// Service is the fully initialized infrastructure bundle returned by Bootstrap.
type Service struct {
	Logger      zerolog.Logger
	Metrics     *metrics.Metrics
	Tracer      trace.Tracer
	DB          *database.InstrumentedDB // nil if no DB config
	DBPool      *pgxpool.Pool            // nil if no DB config (raw pool access)
	Redis       *redis.Client            // nil if no Redis config
	Router      *chi.Mux                 // main API router (Port)
	AdminRouter *chi.Mux                 // admin router: /metrics, /healthz, /readyz, /health, pprof (AdminPort)
	Health      *health.CompositeHealthChecker
	Config      *ServiceConfig

	// Internal shutdown functions called by Run().
	tracerShutdown func(context.Context) error
}

// Bootstrap initializes all observability infrastructure in the correct order.
// It fails fast on any error.
//
// Initialization order:
//  1. Logger
//  2. Metrics registry
//  3. Tracer provider
//  4. Database (if configured)
//  5. Redis (if configured)
//  6. Health checker
//  7. Main router with middleware stack
//  8. Admin router with /metrics, health endpoints, pprof
func Bootstrap(ctx context.Context, cfg *ServiceConfig) (*Service, error) {
	svc := &Service{Config: cfg}

	// 1. Logger.
	svc.Logger = logger.NewLogger(logger.LogConfig{
		Environment:     cfg.Environment,
		Level:           cfg.LogLevel,
		ServiceName:     cfg.Name,
		Version:         cfg.Version,
		DebugSampleRate: cfg.DebugSampleRate,
	})
	zerolog.DefaultContextLogger = &svc.Logger

	// 2. Metrics.
	svc.Metrics = metrics.NewMetrics(cfg.Name)

	// 3. Tracing.
	_, shutdown, err := tracing.InitTracer(ctx, cfg.Tracing)
	if err != nil {
		return nil, fmt.Errorf("initializing tracer: %w", err)
	}
	svc.tracerShutdown = shutdown
	svc.Tracer = tracing.Tracer(cfg.Name)

	// 4. Database (optional).
	if cfg.DB != nil {
		pool, dbErr := connectDB(ctx, cfg.DB, svc.Logger)
		if dbErr != nil {
			return nil, fmt.Errorf("connecting to database: %w", dbErr)
		}
		svc.DBPool = pool
		svc.DB = database.NewInstrumentedDB(pool, svc.Metrics.DB, svc.Tracer, svc.Logger)

		// Register pool collector.
		collector := metrics.NewPgxPoolCollector(pool, cfg.Name)
		svc.Metrics.Registry().MustRegister(collector)
	}

	// 5. Redis (optional).
	if cfg.Redis != nil {
		rdb := redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		if pingErr := rdb.Ping(ctx).Err(); pingErr != nil {
			return nil, fmt.Errorf("connecting to redis: %w", pingErr)
		}
		svc.Redis = rdb

		// Register Redis pool collector.
		redisCollector := metrics.NewRedisPoolCollector(rdb, cfg.Name)
		svc.Metrics.Registry().MustRegister(redisCollector)
	}

	// 6. Health checker — compose all initialized checkers.
	var checkers []health.HealthChecker
	if svc.DBPool != nil {
		checkers = append(checkers, health.NewPostgresHealthChecker(svc.DBPool))
	}
	if svc.Redis != nil {
		checkers = append(checkers, health.NewRedisHealthChecker(svc.Redis))
	}
	if cfg.Kafka != nil {
		checkers = append(checkers, health.NewKafkaHealthChecker(cfg.Kafka.Brokers))
	}
	svc.Health = health.NewCompositeHealthChecker(2*time.Second, checkers...)

	// 7. Main router.
	svc.Router = chi.NewRouter()
	svc.Router.Use(middleware.RequestID)
	svc.Router.Use(middleware.RecoveryWithLogger(svc.Logger))
	svc.Router.Use(tracing.ChiTracingMiddleware(cfg.Name))
	svc.Router.Use(metrics.ChiMetricsMiddleware(svc.Metrics.HTTP, cfg.Name))
	svc.Router.Use(middleware.Logging(svc.Logger))
	svc.Router.Use(middleware.CORS(middleware.DefaultCORSConfig()))

	// 8. Admin router.
	svc.AdminRouter = chi.NewRouter()
	svc.AdminRouter.Use(middleware.RecoveryWithLogger(svc.Logger))

	healthHandler := health.NewHandler(svc.Health, cfg.Name, cfg.Version)
	svc.AdminRouter.Get("/healthz", healthHandler.Healthz())
	svc.AdminRouter.Get("/readyz", healthHandler.Readyz())
	svc.AdminRouter.Get("/health", healthHandler.Health())
	svc.AdminRouter.Handle("/metrics", svc.Metrics.Handler())

	profiling.RegisterPprof(svc.AdminRouter, cfg.EnablePprof)

	svc.Logger.Info().
		Str("service", cfg.Name).
		Str("version", cfg.Version).
		Str("environment", cfg.Environment).
		Int("port", cfg.Port).
		Int("admin_port", cfg.AdminPort).
		Msg("service bootstrapped")

	return svc, nil
}

func connectDB(ctx context.Context, cfg *DBConfig, log zerolog.Logger) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parsing database URL: %w", err)
	}

	poolCfg.MinConns = int32(cfg.MinConns)
	poolCfg.MaxConns = int32(cfg.MaxConns)
	poolCfg.MaxConnLifetime = cfg.MaxConnLife
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdle
	poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	log.Info().
		Int32("max_conns", int32(cfg.MaxConns)).
		Msg("database connection pool established")

	return pool, nil
}

// AuthenticatedGroup returns a chi.Router with tracing span enrichment.
// Services should apply their own auth middleware on this sub-router.
func (s *Service) AuthenticatedGroup() chi.Router {
	r := chi.NewRouter()
	r.Use(tracing.SpanEnricher())
	return r
}

// AdminServer returns an *http.Server configured for the admin port.
func (s *Service) AdminServer() *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf(":%d", s.Config.AdminPort),
		Handler:      s.AdminRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
}

// MainServer returns an *http.Server configured for the main API port.
func (s *Service) MainServer() *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf(":%d", s.Config.Port),
		Handler:      s.Router,
		ReadTimeout:  s.Config.ReadTimeout,
		WriteTimeout: s.Config.WriteTimeout,
		IdleTimeout:  2 * time.Minute,
	}
}
