package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/aigovernance/benchmark"
	aigovdto "github.com/clario360/platform/internal/aigovernance/dto"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/clario360/platform/internal/aigovernance/repository"
)

type BenchmarkService struct {
	benchmarkRepo *repository.BenchmarkRepository
	serverRepo    *repository.InferenceServerRepository
	runner        *benchmark.Runner
	metrics       *Metrics
	logger        zerolog.Logger
}

func NewBenchmarkService(
	benchmarkRepo *repository.BenchmarkRepository,
	serverRepo *repository.InferenceServerRepository,
	runner *benchmark.Runner,
	metrics *Metrics,
	logger zerolog.Logger,
) *BenchmarkService {
	return &BenchmarkService{
		benchmarkRepo: benchmarkRepo,
		serverRepo:    serverRepo,
		runner:        runner,
		metrics:       metrics,
		logger:        logger.With().Str("service", "ai_benchmark").Logger(),
	}
}

// ── Inference Servers ───────────────────────────────────────────────────

func (s *BenchmarkService) CreateServer(ctx context.Context, tenantID uuid.UUID, req aigovdto.CreateInferenceServerRequest) (*aigovmodel.InferenceServer, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("inference server name is required")
	}
	if req.BaseURL == "" {
		return nil, fmt.Errorf("inference server base_url is required")
	}
	if req.BackendType == "" {
		return nil, fmt.Errorf("inference server backend_type is required")
	}
	now := time.Now().UTC()
	item := &aigovmodel.InferenceServer{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Name:           req.Name,
		BackendType:    req.BackendType,
		BaseURL:        req.BaseURL,
		HealthEndpoint: valueOr(req.HealthEndpoint, "/health"),
		ModelName:      req.ModelName,
		APIKey:         req.APIKey,
		Quantization:   req.Quantization,
		Status:         aigovmodel.ServerStatusProvisioning,
		CPUCores:       req.CPUCores,
		MemoryMB:       req.MemoryMB,
		GPUType:        req.GPUType,
		GPUCount:       valueOr(req.GPUCount, 0),
		MaxConcurrent:  valueOr(req.MaxConcurrent, 1),
		StreamCapable:  valueOr(req.StreamCapable, false),
		Metadata:       valueOrJSON(req.Metadata, "{}"),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.serverRepo.Create(ctx, item); err != nil {
		return nil, fmt.Errorf("create inference server: %w", err)
	}
	return item, nil
}

func (s *BenchmarkService) GetServer(ctx context.Context, tenantID, id uuid.UUID) (*aigovmodel.InferenceServer, error) {
	return s.serverRepo.GetByID(ctx, tenantID, id)
}

func (s *BenchmarkService) ListServers(ctx context.Context, tenantID uuid.UUID, params repository.ListServersParams) ([]aigovmodel.InferenceServer, int, error) {
	return s.serverRepo.List(ctx, tenantID, params)
}

func (s *BenchmarkService) UpdateServer(ctx context.Context, tenantID, id uuid.UUID, req aigovdto.UpdateInferenceServerRequest) (*aigovmodel.InferenceServer, error) {
	server, err := s.serverRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("load inference server: %w", err)
	}
	if server.Status == aigovmodel.ServerStatusDecommissioned {
		return nil, fmt.Errorf("cannot update decommissioned server %q", server.Name)
	}

	// Apply partial updates — only overwrite fields that are explicitly set.
	if req.Name != nil {
		server.Name = *req.Name
	}
	if req.BaseURL != nil {
		server.BaseURL = *req.BaseURL
	}
	if req.HealthEndpoint != nil {
		server.HealthEndpoint = *req.HealthEndpoint
	}
	if req.ModelName != nil {
		server.ModelName = req.ModelName
	}
	if req.APIKey != nil {
		server.APIKey = req.APIKey
	}
	if req.Quantization != nil {
		server.Quantization = req.Quantization
	}
	if req.CPUCores != nil {
		server.CPUCores = req.CPUCores
	}
	if req.MemoryMB != nil {
		server.MemoryMB = req.MemoryMB
	}
	if req.GPUType != nil {
		server.GPUType = req.GPUType
	}
	if req.GPUCount != nil {
		server.GPUCount = *req.GPUCount
	}
	if req.MaxConcurrent != nil {
		server.MaxConcurrent = *req.MaxConcurrent
	}
	if req.StreamCapable != nil {
		server.StreamCapable = *req.StreamCapable
	}
	if req.Metadata != nil {
		server.Metadata = *req.Metadata
	}
	server.UpdatedAt = time.Now().UTC()

	if err := s.serverRepo.Update(ctx, server); err != nil {
		return nil, fmt.Errorf("update inference server: %w", err)
	}
	return server, nil
}

func (s *BenchmarkService) UpdateServerStatus(ctx context.Context, tenantID, id uuid.UUID, status aigovmodel.InferenceServerStatus) error {
	return s.serverRepo.UpdateStatus(ctx, tenantID, id, status)
}

func (s *BenchmarkService) DeleteServer(ctx context.Context, tenantID, id uuid.UUID) (*aigovmodel.InferenceServer, error) {
	server, err := s.serverRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("load inference server: %w", err)
	}
	if server.Status == aigovmodel.ServerStatusDecommissioned {
		return nil, fmt.Errorf("server %q is already decommissioned", server.Name)
	}

	if err := s.serverRepo.Delete(ctx, tenantID, id); err != nil {
		return nil, fmt.Errorf("decommission inference server: %w", err)
	}

	// Refresh to return the final state (with decommissioned status / timestamps).
	server, err = s.serverRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("reload decommissioned server: %w", err)
	}
	return server, nil
}

// ── Benchmark Suites ────────────────────────────────────────────────────

func (s *BenchmarkService) CreateSuite(ctx context.Context, tenantID, userID uuid.UUID, req aigovdto.CreateBenchmarkSuiteRequest) (*aigovmodel.BenchmarkSuite, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("benchmark suite name is required")
	}
	if req.ModelSlug == "" {
		return nil, fmt.Errorf("benchmark suite model_slug is required")
	}

	var prompts []benchmark.BenchmarkPrompt
	if len(req.PromptDataset) > 0 {
		if err := json.Unmarshal(req.PromptDataset, &prompts); err != nil {
			return nil, fmt.Errorf("invalid prompt_dataset: must be array of {system_prompt, user_message}")
		}
	}

	item := &aigovmodel.BenchmarkSuite{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Name:           req.Name,
		Description:    req.Description,
		ModelSlug:      req.ModelSlug,
		PromptDataset:  defaultJSON(req.PromptDataset, "[]"),
		DatasetSize:    len(prompts),
		WarmupCount:    valueOr(req.WarmupCount, 5),
		IterationCount: valueOr(req.IterationCount, 100),
		Concurrency:    valueOr(req.Concurrency, 1),
		TimeoutSeconds: valueOr(req.TimeoutSeconds, 60),
		StreamEnabled:  valueOr(req.StreamEnabled, false),
		MaxRetries:     valueOr(req.MaxRetries, 3),
		CreatedBy:      userID,
		CreatedAt:      time.Now().UTC(),
	}
	if err := s.benchmarkRepo.CreateSuite(ctx, item); err != nil {
		return nil, fmt.Errorf("create benchmark suite: %w", err)
	}
	return item, nil
}

func (s *BenchmarkService) GetSuite(ctx context.Context, tenantID, id uuid.UUID) (*aigovmodel.BenchmarkSuite, error) {
	return s.benchmarkRepo.GetSuite(ctx, tenantID, id)
}

func (s *BenchmarkService) ListSuites(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]aigovmodel.BenchmarkSuite, int, error) {
	return s.benchmarkRepo.ListSuites(ctx, tenantID, page, perPage)
}

func (s *BenchmarkService) UpdateSuite(ctx context.Context, tenantID, userID, suiteID uuid.UUID, req aigovdto.UpdateBenchmarkSuiteRequest) (*aigovmodel.BenchmarkSuite, error) {
	suite, err := s.benchmarkRepo.GetSuite(ctx, tenantID, suiteID)
	if err != nil {
		return nil, fmt.Errorf("load benchmark suite: %w", err)
	}

	if req.Name != nil {
		suite.Name = *req.Name
	}
	if req.Description != nil {
		suite.Description = *req.Description
	}
	if req.PromptDataset != nil {
		var prompts []benchmark.BenchmarkPrompt
		if err := json.Unmarshal(*req.PromptDataset, &prompts); err != nil {
			return nil, fmt.Errorf("invalid prompt_dataset: must be array of {system_prompt, user_message}")
		}
		if len(prompts) == 0 {
			return nil, fmt.Errorf("prompt_dataset must contain at least one prompt")
		}
		suite.PromptDataset = *req.PromptDataset
		suite.DatasetSize = len(prompts)
	}
	if req.WarmupCount != nil {
		suite.WarmupCount = *req.WarmupCount
	}
	if req.IterationCount != nil {
		suite.IterationCount = *req.IterationCount
	}
	if req.Concurrency != nil {
		suite.Concurrency = *req.Concurrency
	}
	if req.TimeoutSeconds != nil {
		suite.TimeoutSeconds = *req.TimeoutSeconds
	}
	if req.StreamEnabled != nil {
		suite.StreamEnabled = *req.StreamEnabled
	}
	if req.MaxRetries != nil {
		suite.MaxRetries = *req.MaxRetries
	}
	suite.UpdatedAt = timePtr(time.Now().UTC())

	if err := s.benchmarkRepo.UpdateSuite(ctx, suite); err != nil {
		return nil, fmt.Errorf("update benchmark suite: %w", err)
	}
	return suite, nil
}

func (s *BenchmarkService) DeleteSuite(ctx context.Context, tenantID, userID, suiteID uuid.UUID) error {
	suite, err := s.benchmarkRepo.GetSuite(ctx, tenantID, suiteID)
	if err != nil {
		return fmt.Errorf("load benchmark suite: %w", err)
	}

	// Prevent deletion if active runs exist.
	activeRuns, _, err := s.benchmarkRepo.ListRuns(ctx, tenantID, &suiteID, 1, 1)
	if err != nil {
		return fmt.Errorf("check active runs: %w", err)
	}
	for _, run := range activeRuns {
		if run.Status == aigovmodel.BenchmarkStatusRunning {
			return fmt.Errorf("cannot delete suite %q: active run %s is in progress", suite.Name, run.ID)
		}
	}

	if err := s.benchmarkRepo.DeleteSuite(ctx, tenantID, suiteID); err != nil {
		return fmt.Errorf("delete benchmark suite: %w", err)
	}
	return nil
}

// ── Benchmark Runs ──────────────────────────────────────────────────────

func (s *BenchmarkService) RunBenchmark(ctx context.Context, tenantID, userID, suiteID, serverID uuid.UUID) (*aigovmodel.BenchmarkRun, error) {
	suite, err := s.benchmarkRepo.GetSuite(ctx, tenantID, suiteID)
	if err != nil {
		return nil, fmt.Errorf("load benchmark suite: %w", err)
	}
	server, err := s.serverRepo.GetByID(ctx, tenantID, serverID)
	if err != nil {
		return nil, fmt.Errorf("load inference server: %w", err)
	}
	if server.Status == aigovmodel.ServerStatusDecommissioned {
		return nil, fmt.Errorf("inference server %q is decommissioned", server.Name)
	}

	// Determine whether streaming should be used: suite must opt-in AND the
	// server must declare itself capable.
	useStream := suite.StreamEnabled && server.StreamCapable

	now := time.Now().UTC()
	run := &aigovmodel.BenchmarkRun{
		ID:           uuid.New(),
		TenantID:     tenantID,
		SuiteID:      suiteID,
		ServerID:     serverID,
		BackendType:  server.BackendType,
		ModelName:    valueOrPtr(server.ModelName, "unknown"),
		Quantization: server.Quantization,
		Status:       aigovmodel.BenchmarkStatusRunning,
		StreamUsed:   useStream,
		StartedAt:    &now,
		CreatedBy:    userID,
		CreatedAt:    now,
		RawResults:   json.RawMessage("[]"),
	}
	if err := s.benchmarkRepo.CreateRun(ctx, run); err != nil {
		return nil, fmt.Errorf("create benchmark run: %w", err)
	}

	// Parse prompts.
	var prompts []benchmark.BenchmarkPrompt
	if err := json.Unmarshal(suite.PromptDataset, &prompts); err != nil {
		s.failRun(ctx, run, fmt.Sprintf("parse prompts: %v", err))
		return run, nil
	}

	// Build retry config from suite settings.
	retryCfg := benchmark.DefaultRetryConfig()
	retryCfg.MaxRetries = suite.MaxRetries

	// Execute the benchmark with streaming, auth, and retry support.
	cfg := benchmark.RunConfig{
		BaseURL:        server.BaseURL,
		ModelName:      valueOrPtr(server.ModelName, ""),
		APIKey:         valueOrPtr(server.APIKey, ""),
		Prompts:        prompts,
		WarmupCount:    suite.WarmupCount,
		IterationCount: suite.IterationCount,
		Concurrency:    suite.Concurrency,
		Timeout:        time.Duration(suite.TimeoutSeconds) * time.Second,
		Stream:         useStream,
		Retry:          retryCfg,
	}
	results, err := s.runner.Execute(ctx, cfg)
	if err != nil {
		s.failRun(ctx, run, err.Error())
		return run, nil
	}

	// Populate the run with aggregated results.
	completed := time.Now().UTC()
	rawJSON, _ := json.Marshal(results.Raw)
	run.Status = aigovmodel.BenchmarkStatusCompleted
	run.P50LatencyMS = &results.P50LatencyMS
	run.P95LatencyMS = &results.P95LatencyMS
	run.P99LatencyMS = &results.P99LatencyMS
	run.AvgLatencyMS = &results.AvgLatencyMS
	run.MinLatencyMS = &results.MinLatencyMS
	run.MaxLatencyMS = &results.MaxLatencyMS
	run.TokensPerSecond = &results.TokensPerSecond
	run.RequestsPerSecond = &results.RequestsPerSecond
	run.TotalTokens = &results.TotalTokens
	run.TotalRequests = &results.TotalRequests
	run.FailedRequests = results.FailedRequests
	run.RetriedRequests = results.RetriedRequests
	run.CompletedAt = &completed
	run.DurationSeconds = &results.DurationSeconds
	run.RawResults = json.RawMessage(rawJSON)

	// TTFT metrics (only populated when streaming was used).
	if results.P50TTFT_MS > 0 {
		run.P50TTFT_MS = &results.P50TTFT_MS
		run.P95TTFT_MS = &results.P95TTFT_MS
		run.AvgTTFT_MS = &results.AvgTTFT_MS
	}

	if err := s.benchmarkRepo.UpdateRunResults(ctx, run); err != nil {
		s.logger.Error().Err(err).Str("run_id", run.ID.String()).Msg("failed to store benchmark results")
	}

	if s.metrics != nil {
		s.metrics.BenchmarkRunsTotal.WithLabelValues(string(run.BackendType), string(run.Status)).Inc()
	}

	return run, nil
}

func (s *BenchmarkService) GetRun(ctx context.Context, tenantID, id uuid.UUID) (*aigovmodel.BenchmarkRun, error) {
	return s.benchmarkRepo.GetRun(ctx, tenantID, id)
}

func (s *BenchmarkService) ListRuns(ctx context.Context, tenantID uuid.UUID, suiteID *uuid.UUID, page, perPage int) ([]aigovmodel.BenchmarkRun, int, error) {
	return s.benchmarkRepo.ListRuns(ctx, tenantID, suiteID, page, perPage)
}

// CompareRuns produces a side-by-side analysis of multiple benchmark runs.
func (s *BenchmarkService) CompareRuns(ctx context.Context, tenantID uuid.UUID, runIDs []uuid.UUID) (*aigovmodel.BenchmarkComparison, error) {
	if len(runIDs) < 2 {
		return nil, fmt.Errorf("comparison requires at least 2 runs")
	}
	runs, err := s.benchmarkRepo.GetRunsByIDs(ctx, tenantID, runIDs)
	if err != nil {
		return nil, fmt.Errorf("load benchmark runs: %w", err)
	}
	if len(runs) < 2 {
		return nil, fmt.Errorf("insufficient completed runs for comparison")
	}

	comparison := &aigovmodel.BenchmarkComparison{Runs: runs}
	s.computeComparison(comparison)
	return comparison, nil
}

// ── Cost Models ─────────────────────────────────────────────────────────

func (s *BenchmarkService) CreateCostModel(ctx context.Context, tenantID uuid.UUID, req aigovdto.CreateComputeCostModelRequest) (*aigovmodel.ComputeCostModel, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("cost model name is required")
	}
	item := &aigovmodel.ComputeCostModel{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		Name:               req.Name,
		BackendType:        req.BackendType,
		InstanceType:       req.InstanceType,
		HourlyCostUSD:      req.HourlyCostUSD,
		CPUCores:           req.CPUCores,
		MemoryGB:           req.MemoryGB,
		GPUType:            req.GPUType,
		GPUCount:           valueOr(req.GPUCount, 0),
		MaxTokensPerSecond: req.MaxTokensPerSecond,
		Notes:              req.Notes,
		CreatedAt:          time.Now().UTC(),
	}
	if err := s.benchmarkRepo.CreateCostModel(ctx, item); err != nil {
		return nil, fmt.Errorf("create compute cost model: %w", err)
	}
	return item, nil
}

func (s *BenchmarkService) ListCostModels(ctx context.Context, tenantID uuid.UUID) ([]aigovmodel.ComputeCostModel, error) {
	return s.benchmarkRepo.ListCostModels(ctx, tenantID)
}

// ── Cost Estimation ─────────────────────────────────────────────────────

func (s *BenchmarkService) EstimateCostSavings(ctx context.Context, tenantID uuid.UUID, cpuRunID, gpuRunID uuid.UUID) (map[string]any, error) {
	cpuRun, err := s.benchmarkRepo.GetRun(ctx, tenantID, cpuRunID)
	if err != nil {
		return nil, fmt.Errorf("load CPU run: %w", err)
	}
	gpuRun, err := s.benchmarkRepo.GetRun(ctx, tenantID, gpuRunID)
	if err != nil {
		return nil, fmt.Errorf("load GPU run: %w", err)
	}

	hoursPerMonth := 730.0
	cpuHourly := ptrFloat(cpuRun.EstimatedHourlyCostUSD)
	gpuHourly := ptrFloat(gpuRun.EstimatedHourlyCostUSD)

	return map[string]any{
		"cpu_monthly_cost":     cpuHourly * hoursPerMonth,
		"gpu_monthly_cost":     gpuHourly * hoursPerMonth,
		"monthly_savings":      (gpuHourly - cpuHourly) * hoursPerMonth,
		"savings_percent":      safePercent(gpuHourly-cpuHourly, gpuHourly),
		"cpu_tokens_per_sec":   ptrFloat(cpuRun.TokensPerSecond),
		"gpu_tokens_per_sec":   ptrFloat(gpuRun.TokensPerSecond),
		"latency_increase_pct": safePercent(ptrFloat(cpuRun.P95LatencyMS)-ptrFloat(gpuRun.P95LatencyMS), ptrFloat(gpuRun.P95LatencyMS)),
	}, nil
}

// ── Helpers ─────────────────────────────────────────────────────────────

func (s *BenchmarkService) failRun(ctx context.Context, run *aigovmodel.BenchmarkRun, msg string) {
	run.Status = aigovmodel.BenchmarkStatusFailed
	run.ErrorMessage = &msg
	_ = s.benchmarkRepo.UpdateRunStatus(ctx, run.TenantID, run.ID, aigovmodel.BenchmarkStatusFailed, &msg)
}

func (s *BenchmarkService) computeComparison(comp *aigovmodel.BenchmarkComparison) {
	if len(comp.Runs) < 2 {
		return
	}
	r0 := comp.Runs[0]
	r1 := comp.Runs[1]

	costDiff := ptrFloat(r0.EstimatedHourlyCostUSD) - ptrFloat(r1.EstimatedHourlyCostUSD)
	comp.CostDeltaMonthlyUSD = costDiff * 730

	p95_0 := ptrFloat(r0.P95LatencyMS)
	p95_1 := ptrFloat(r1.P95LatencyMS)
	comp.LatencyDeltaPct = safePercent(p95_0-p95_1, math.Max(p95_0, p95_1))

	sim0 := ptrFloat(r0.SemanticSimilarity)
	sim1 := ptrFloat(r1.SemanticSimilarity)
	comp.QualityDeltaPct = safePercent(sim0-sim1, math.Max(sim0, sim1))

	// Simple heuristic for recommendation.
	latencyRatio := p95_0 / math.Max(p95_1, 1)
	switch {
	case latencyRatio < 3 && comp.QualityDeltaPct > -10:
		comp.Recommendation = "cpu_viable"
		comp.RecommendationReason = "CPU latency is within 3x of GPU with acceptable quality."
	case latencyRatio >= 3:
		comp.Recommendation = "gpu_required"
		comp.RecommendationReason = fmt.Sprintf("CPU latency is %.1fx GPU — not suitable for production.", latencyRatio)
	default:
		comp.Recommendation = "needs_more_data"
		comp.RecommendationReason = "Results are inconclusive; run more iterations or a larger dataset."
	}
}

func valueOr[T comparable](v T, def T) T {
	var zero T
	if v == zero {
		return def
	}
	return v
}

func valueOrJSON(v json.RawMessage, def string) json.RawMessage {
	if len(v) == 0 {
		return json.RawMessage(def)
	}
	return v
}

func valueOrPtr(v *string, def string) string {
	if v == nil {
		return def
	}
	return *v
}

func ptrFloat(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func safePercent(diff, base float64) float64 {
	if base == 0 {
		return 0
	}
	return (diff / base) * 100
}
