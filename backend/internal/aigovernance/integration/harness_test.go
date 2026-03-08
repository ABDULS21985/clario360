//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	tc "github.com/testcontainers/testcontainers-go"
	postgresmod "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/clario360/platform/internal/aigovernance"
	aigovdto "github.com/clario360/platform/internal/aigovernance/dto"
	aigovmiddleware "github.com/clario360/platform/internal/aigovernance/middleware"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/clario360/platform/internal/aigovernance/repository"
	aigovservice "github.com/clario360/platform/internal/aigovernance/service"
	"github.com/clario360/platform/internal/database"
)

var (
	sharedEnvOnce sync.Once
	sharedEnv     *aigovIntegrationEnv
	sharedEnvErr  error
)

type aigovIntegrationEnv struct {
	ctx    context.Context
	cancel context.CancelFunc

	postgres *postgresmod.PostgresContainer
	db       *pgxpool.Pool

	registryRepo   *repository.ModelRegistryRepository
	predictionRepo *repository.PredictionLogRepository
	shadowRepo     *repository.ShadowComparisonRepository
	driftRepo      *repository.DriftReportRepository

	registrySvc      *aigovservice.RegistryService
	predictionSvc    *aigovservice.PredictionService
	comparisonSvc    *aigovservice.ComparisonService
	shadowSvc        *aigovservice.ShadowService
	lifecycleSvc     *aigovservice.LifecycleService
	driftSvc         *aigovservice.DriftService
	predictionLogger *aigovmiddleware.PredictionLogger
}

type aigovHarness struct {
	env      *aigovIntegrationEnv
	tenantID uuid.UUID
	userID   uuid.UUID
}

func TestMain(m *testing.M) {
	code := m.Run()
	if sharedEnv != nil {
		sharedEnv.Close()
	}
	os.Exit(code)
}

func newHarness(t *testing.T) *aigovHarness {
	t.Helper()
	testcontainersSkipIfUnavailable(t)
	env := mustSharedEnv(t)
	return &aigovHarness{
		env:      env,
		tenantID: uuid.New(),
		userID:   uuid.New(),
	}
}

func mustSharedEnv(t *testing.T) *aigovIntegrationEnv {
	t.Helper()
	sharedEnvOnce.Do(func() {
		sharedEnv, sharedEnvErr = startSharedEnv()
	})
	if sharedEnvErr != nil {
		t.Fatalf("start ai governance integration environment: %v", sharedEnvErr)
	}
	return sharedEnv
}

func startSharedEnv() (*aigovIntegrationEnv, error) {
	ctx, cancel := context.WithCancel(context.Background())

	container, err := postgresmod.Run(ctx, "postgres:16-alpine",
		postgresmod.WithDatabase("platform_core"),
		postgresmod.WithUsername("postgres"),
		postgresmod.WithPassword("postgres"),
		postgresmod.BasicWaitStrategies(),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("start postgres container: %w", err)
	}

	dbURL := container.MustConnectionString(ctx, "sslmode=disable")
	if err := database.RunMigrations(dbURL, platformCoreMigrationsPath()); err != nil {
		_ = container.Terminate(context.Background())
		cancel()
		return nil, fmt.Errorf("run platform_core migrations: %w", err)
	}

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		_ = container.Terminate(context.Background())
		cancel()
		return nil, fmt.Errorf("open platform_core pool: %w", err)
	}
	if err := db.Ping(ctx); err != nil {
		db.Close()
		_ = container.Terminate(context.Background())
		cancel()
		return nil, fmt.Errorf("ping platform_core: %w", err)
	}

	reg := prometheus.NewRegistry()
	logger := zerolog.Nop()
	metrics := aigovservice.NewMetrics(reg)
	registryRepo := repository.NewModelRegistryRepository(db, logger)
	predictionRepo := repository.NewPredictionLogRepository(db, logger)
	shadowRepo := repository.NewShadowComparisonRepository(db, logger)
	driftRepo := repository.NewDriftReportRepository(db, logger)
	explanationSvc := aigovservice.NewExplanationService(logger)

	return &aigovIntegrationEnv{
		ctx:              ctx,
		cancel:           cancel,
		postgres:         container,
		db:               db,
		registryRepo:     registryRepo,
		predictionRepo:   predictionRepo,
		shadowRepo:       shadowRepo,
		driftRepo:        driftRepo,
		registrySvc:      aigovservice.NewRegistryService(registryRepo, nil, metrics, logger),
		predictionSvc:    aigovservice.NewPredictionService(predictionRepo, registryRepo, nil, metrics, logger),
		comparisonSvc:    aigovservice.NewComparisonService(registryRepo, predictionRepo, shadowRepo, nil, metrics, logger),
		shadowSvc:        aigovservice.NewShadowService(registryRepo, shadowRepo, predictionRepo, nil, metrics, logger),
		lifecycleSvc:     aigovservice.NewLifecycleService(registryRepo, shadowRepo, nil, metrics, logger),
		driftSvc:         aigovservice.NewDriftService(registryRepo, predictionRepo, driftRepo, nil, metrics, logger),
		predictionLogger: aigovmiddleware.NewPredictionLogger(ctx, explanationSvc, predictionRepo, registryRepo, nil, metrics, logger),
	}, nil
}

func (e *aigovIntegrationEnv) Close() {
	if e == nil {
		return
	}
	if e.cancel != nil {
		e.cancel()
	}
	if e.db != nil {
		e.db.Close()
	}
	if e.postgres != nil {
		_ = e.postgres.Terminate(context.Background())
	}
}

func (h *aigovHarness) registerRuleModel(t *testing.T, slug string, riskTier aigovmodel.RiskTier) *aigovmodel.RegisteredModel {
	t.Helper()
	model, err := h.env.registrySvc.RegisterModel(h.env.ctx, h.tenantID, h.userID, aigovdto.RegisterModelRequest{
		Name:        "Integration Rule Model " + slug,
		Slug:        slug,
		Description: "Integration test model",
		ModelType:   aigovmodel.ModelTypeRuleBased,
		Suite:       aigovmodel.SuiteCyber,
		OwnerTeam:   "integration",
		RiskTier:    riskTier,
		Tags:        []string{"integration"},
	})
	if err != nil {
		t.Fatalf("RegisterModel() error = %v", err)
	}
	return model
}

func (h *aigovHarness) createRuleVersion(t *testing.T, modelID uuid.UUID, description string) *aigovmodel.ModelVersion {
	t.Helper()
	template := "Matched rules: {{join .matched_rules \", \"}}."
	version, err := h.env.registrySvc.CreateVersion(h.env.ctx, h.tenantID, h.userID, modelID, aigovdto.CreateVersionRequest{
		Description:         description,
		ArtifactType:        aigovmodel.ArtifactTypeRuleSet,
		ArtifactConfig:      []byte(`{"rules":["integration"],"strategy":"first_match"}`),
		ExplainabilityType:  aigovmodel.ExplainabilityRuleTrace,
		ExplanationTemplate: &template,
	})
	if err != nil {
		t.Fatalf("CreateVersion() error = %v", err)
	}
	return version
}

func (h *aigovHarness) promote(t *testing.T, modelID, versionID uuid.UUID, approvedBy *uuid.UUID, override bool) *aigovmodel.ModelVersion {
	t.Helper()
	version, err := h.env.lifecycleSvc.Promote(h.env.ctx, h.tenantID, modelID, versionID, approvedBy, override)
	if err != nil {
		t.Fatalf("Promote() error = %v", err)
	}
	h.env.predictionLogger.InvalidateModel(version.ModelSlug)
	return version
}

func (h *aigovHarness) getVersion(t *testing.T, modelID, versionID uuid.UUID) *aigovmodel.ModelVersion {
	t.Helper()
	version, err := h.env.registryRepo.GetVersion(h.env.ctx, h.tenantID, modelID, versionID)
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}
	return version
}

func (h *aigovHarness) currentProduction(t *testing.T, modelID uuid.UUID) *aigovmodel.ModelVersion {
	t.Helper()
	version, err := h.env.registryRepo.GetCurrentProductionVersion(h.env.ctx, h.tenantID, modelID)
	if err != nil {
		t.Fatalf("GetCurrentProductionVersion() error = %v", err)
	}
	return version
}

func (h *aigovHarness) currentShadow(t *testing.T, modelID uuid.UUID) *aigovmodel.ModelVersion {
	t.Helper()
	version, err := h.env.registryRepo.GetCurrentShadowVersion(h.env.ctx, h.tenantID, modelID)
	if err != nil {
		t.Fatalf("GetCurrentShadowVersion() error = %v", err)
	}
	return version
}

func (h *aigovHarness) invokeRulePrediction(t *testing.T, slug string, index int, shadow bool) {
	t.Helper()
	params := h.rulePredictionParams(slug, index)
	if shadow {
		params.ShadowModelFunc = ruleModelOutput("Integration Rule", 0.99)
	}
	if _, err := h.env.predictionLogger.Predict(h.env.ctx, params); err != nil {
		t.Fatalf("Predict() error = %v", err)
	}
}

func (h *aigovHarness) rulePredictionParams(slug string, index int) aigovernance.PredictParams {
	return aigovernance.PredictParams{
		TenantID:     h.tenantID,
		ModelSlug:    slug,
		UseCase:      "integration_rule_evaluation",
		EntityType:   "integration_record",
		Input:        map[string]any{"index": index, "slug": slug},
		InputSummary: map[string]any{"index": index, "slug": slug},
		ModelFunc:    ruleModelOutput("Integration Rule", 0.95),
	}
}

func (h *aigovHarness) waitForPredictionCount(t *testing.T, modelID uuid.UUID, expected int) []aigovmodel.PredictionLog {
	t.Helper()
	var logs []aigovmodel.PredictionLog
	waitFor(t, 5*time.Second, func() (bool, error) {
		items, _, err := h.env.predictionSvc.List(h.env.ctx, h.tenantID, aigovdto.PredictionQuery{
			ModelID: &modelID,
			Page:    1,
			PerPage: expected + 20,
		})
		if err != nil {
			return false, err
		}
		logs = items
		return len(items) >= expected, nil
	})
	return logs
}

func (h *aigovHarness) backdateProductionVersion(t *testing.T, version *aigovmodel.ModelVersion, promotedAt time.Time) *aigovmodel.ModelVersion {
	t.Helper()
	version.PromotedToProductionAt = &promotedAt
	version.UpdatedAt = time.Now().UTC()
	if err := h.env.registryRepo.UpdateVersionStatus(h.env.ctx, version); err != nil {
		t.Fatalf("UpdateVersionStatus() error = %v", err)
	}
	h.env.predictionLogger.InvalidateModel(version.ModelSlug)
	return version
}

func (h *aigovHarness) insertPredictionLogs(t *testing.T, version *aigovmodel.ModelVersion, start time.Time, values []float64) {
	t.Helper()
	items := make([]*aigovmodel.PredictionLog, 0, len(values))
	for idx, value := range values {
		confidence := value
		inputHash := fmt.Sprintf("%s-%d", version.ID.String(), idx)
		items = append(items, &aigovmodel.PredictionLog{
			ID:                    uuid.New(),
			TenantID:              h.tenantID,
			ModelID:               version.ModelID,
			ModelVersionID:        version.ID,
			ModelSlug:             version.ModelSlug,
			ModelVersionNumber:    version.VersionNumber,
			InputHash:             inputHash,
			InputSummary:          []byte(fmt.Sprintf(`{"index":%d}`, idx)),
			Prediction:            []byte(fmt.Sprintf(`{"score":%.4f}`, value)),
			Confidence:            &confidence,
			ExplanationStructured: []byte(`{"type":"integration"}`),
			ExplanationText:       "integration prediction",
			ExplanationFactors:    []byte(`[]`),
			Suite:                 string(version.ModelSuite),
			UseCase:               "integration_drift",
			EntityType:            "integration_record",
			IsShadow:              false,
			LatencyMS:             4,
			CreatedAt:             start.Add(time.Duration(idx) * time.Hour),
		})
	}
	if err := h.env.predictionRepo.InsertBatch(h.env.ctx, items); err != nil {
		t.Fatalf("InsertBatch() error = %v", err)
	}
	if err := h.env.registryRepo.UpdateVersionAggregates(h.env.ctx, h.tenantID, version.ID); err != nil {
		t.Fatalf("UpdateVersionAggregates() error = %v", err)
	}
}

func ruleModelOutput(ruleName string, confidence float64) func(context.Context, any) (*aigovernance.ModelOutput, error) {
	return func(context.Context, any) (*aigovernance.ModelOutput, error) {
		return &aigovernance.ModelOutput{
			Output: map[string]any{
				"matched": true,
				"rule":    ruleName,
			},
			Confidence: confidence,
			Metadata: map[string]any{
				"matched":            true,
				"rule_name":          ruleName,
				"matched_rules":      []string{ruleName},
				"matched_conditions": []string{"field_match"},
				"rule_weights":       map[string]any{ruleName: confidence},
			},
		}, nil
	}
}

func waitFor(t *testing.T, timeout time.Duration, fn func() (bool, error)) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ok, err := fn()
		if err != nil {
			t.Fatalf("wait condition error: %v", err)
		}
		if ok {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("timed out waiting for condition")
}

func platformCoreMigrationsPath() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller(0) failed")
	}
	return filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "migrations", "platform_core")
}

func testcontainersSkipIfUnavailable(t *testing.T) {
	t.Helper()
	tc.SkipIfProviderIsNotHealthy(t)
}
