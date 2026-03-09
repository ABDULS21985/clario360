package integration

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmiddleware "github.com/clario360/platform/internal/aigovernance/middleware"
	"github.com/clario360/platform/internal/aigovernance/repository"
	aigovservice "github.com/clario360/platform/internal/aigovernance/service"
	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/events"
)

type Runtime struct {
	Pool             *pgxpool.Pool
	RegistryRepo     *repository.ModelRegistryRepository
	PredictionRepo   *repository.PredictionLogRepository
	ShadowRepo       *repository.ShadowComparisonRepository
	DriftRepo        *repository.DriftReportRepository
	ValidationRepo   *repository.ValidationResultRepository
	Metrics          *aigovservice.Metrics
	ExplanationSvc   *aigovservice.ExplanationService
	PredictionLogger *aigovmiddleware.PredictionLogger
	ComparisonSvc    *aigovservice.ComparisonService
	DriftSvc         *aigovservice.DriftService
	ValidationSvc    *aigovservice.ValidationService
}

func NewRuntime(ctx context.Context, cfg *config.Config, reg prometheus.Registerer, producer *events.Producer, logger zerolog.Logger) (*Runtime, error) {
	dsn := aigovernance.BuildPlatformCoreDSN(cfg.Database)
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse platform core dsn: %w", err)
	}
	poolCfg.MinConns = 1
	poolCfg.MaxConns = 5
	poolCfg.MaxConnLifetime = cfg.Database.ConnMaxLifetime
	poolCfg.MaxConnIdleTime = 5 * time.Minute
	poolCfg.HealthCheckPeriod = time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("connect platform core ai governance pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping platform core ai governance pool: %w", err)
	}
	metrics := aigovservice.NewMetrics(reg)
	registryRepo := repository.NewModelRegistryRepository(pool, logger)
	predictionRepo := repository.NewPredictionLogRepository(pool, logger)
	shadowRepo := repository.NewShadowComparisonRepository(pool, logger)
	driftRepo := repository.NewDriftReportRepository(pool, logger)
	validationRepo := repository.NewValidationResultRepository(pool, logger)
	explanationSvc := aigovservice.NewExplanationService(logger)
	predictionLogger := aigovmiddleware.NewPredictionLogger(ctx, explanationSvc, predictionRepo, registryRepo, producer, metrics, logger)
	comparisonSvc := aigovservice.NewComparisonService(registryRepo, predictionRepo, shadowRepo, producer, metrics, logger)
	driftSvc := aigovservice.NewDriftService(registryRepo, predictionRepo, driftRepo, producer, metrics, logger)
	validationSvc := aigovservice.NewValidationService(registryRepo, predictionRepo, validationRepo, producer, metrics, nil, logger)
	return &Runtime{
		Pool:             pool,
		RegistryRepo:     registryRepo,
		PredictionRepo:   predictionRepo,
		ShadowRepo:       shadowRepo,
		DriftRepo:        driftRepo,
		ValidationRepo:   validationRepo,
		Metrics:          metrics,
		ExplanationSvc:   explanationSvc,
		PredictionLogger: predictionLogger,
		ComparisonSvc:    comparisonSvc,
		DriftSvc:         driftSvc,
		ValidationSvc:    validationSvc,
	}, nil
}

func (r *Runtime) Close() {
	if r == nil || r.Pool == nil {
		return
	}
	r.Pool.Close()
}
