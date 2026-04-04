package repository

import (
	"context"
	"fmt"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type BenchmarkRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewBenchmarkRepository(db *pgxpool.Pool, logger zerolog.Logger) *BenchmarkRepository {
	return &BenchmarkRepository{db: db, logger: loggerWithRepo(logger, "ai_benchmark")}
}

// ── Suites ──────────────────────────────────────────────────────────────

func (r *BenchmarkRepository) CreateSuite(ctx context.Context, item *aigovmodel.BenchmarkSuite) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO ai_benchmark_suites (
			id, tenant_id, name, description, model_slug, prompt_dataset,
			dataset_size, warmup_count, iteration_count, concurrency,
			timeout_seconds, stream_enabled, max_retries,
			created_by, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)`,
		item.ID, item.TenantID, item.Name, item.Description, item.ModelSlug,
		item.PromptDataset, item.DatasetSize, item.WarmupCount, item.IterationCount,
		item.Concurrency, item.TimeoutSeconds, item.StreamEnabled, item.MaxRetries,
		item.CreatedBy, item.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert ai benchmark suite: %w", err)
	}
	return nil
}

func (r *BenchmarkRepository) GetSuite(ctx context.Context, tenantID, id uuid.UUID) (*aigovmodel.BenchmarkSuite, error) {
	row := r.db.QueryRow(ctx, benchmarkSuiteSelectSQL+`
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	item, err := scanBenchmarkSuite(row)
	if err != nil {
		return nil, rowNotFound(err)
	}
	return item, nil
}

func (r *BenchmarkRepository) ListSuites(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]aigovmodel.BenchmarkSuite, int, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 25
	}
	var total int
	err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM ai_benchmark_suites WHERE tenant_id = $1",
		tenantID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count ai benchmark suites: %w", err)
	}
	offset := (page - 1) * perPage
	rows, err := r.db.Query(ctx, benchmarkSuiteSelectSQL+`
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		tenantID, perPage, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list ai benchmark suites: %w", err)
	}
	defer rows.Close()
	items := make([]aigovmodel.BenchmarkSuite, 0)
	for rows.Next() {
		item, scanErr := scanBenchmarkSuite(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (r *BenchmarkRepository) UpdateSuite(ctx context.Context, item *aigovmodel.BenchmarkSuite) error {
	_, err := r.db.Exec(ctx, `
		UPDATE ai_benchmark_suites SET
			name = $3, description = $4, prompt_dataset = $5,
			dataset_size = $6, warmup_count = $7, iteration_count = $8,
			concurrency = $9, timeout_seconds = $10,
			stream_enabled = $11, max_retries = $12, updated_at = $13
		WHERE tenant_id = $1 AND id = $2`,
		item.TenantID, item.ID,
		item.Name, item.Description, item.PromptDataset,
		item.DatasetSize, item.WarmupCount, item.IterationCount,
		item.Concurrency, item.TimeoutSeconds,
		item.StreamEnabled, item.MaxRetries, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update ai benchmark suite: %w", err)
	}
	return nil
}

func (r *BenchmarkRepository) DeleteSuite(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM ai_benchmark_suites
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	if err != nil {
		return fmt.Errorf("delete ai benchmark suite: %w", err)
	}
	return nil
}

// ── Runs ────────────────────────────────────────────────────────────────

func (r *BenchmarkRepository) CreateRun(ctx context.Context, item *aigovmodel.BenchmarkRun) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO ai_benchmark_runs (
			id, tenant_id, suite_id, server_id, backend_type, model_name,
			quantization, status, stream_used, started_at,
			created_by, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)`,
		item.ID, item.TenantID, item.SuiteID, item.ServerID, item.BackendType,
		item.ModelName, item.Quantization, item.Status, item.StreamUsed,
		item.StartedAt, item.CreatedBy, item.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert ai benchmark run: %w", err)
	}
	return nil
}

func (r *BenchmarkRepository) GetRun(ctx context.Context, tenantID, id uuid.UUID) (*aigovmodel.BenchmarkRun, error) {
	row := r.db.QueryRow(ctx, benchmarkRunSelectSQL+`
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	item, err := scanBenchmarkRun(row)
	if err != nil {
		return nil, rowNotFound(err)
	}
	return item, nil
}

func (r *BenchmarkRepository) ListRuns(ctx context.Context, tenantID uuid.UUID, suiteID *uuid.UUID, page, perPage int) ([]aigovmodel.BenchmarkRun, int, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 25
	}
	where := "WHERE tenant_id = $1"
	args := []any{tenantID}
	idx := 2
	if suiteID != nil {
		where += fmt.Sprintf(" AND suite_id = $%d", idx)
		args = append(args, *suiteID)
		idx++
	}

	var total int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM ai_benchmark_runs "+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count ai benchmark runs: %w", err)
	}

	offset := (page - 1) * perPage
	args = append(args, perPage, offset)
	query := fmt.Sprintf(benchmarkRunSelectSQL+` %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, where, idx, idx+1)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list ai benchmark runs: %w", err)
	}
	defer rows.Close()
	items := make([]aigovmodel.BenchmarkRun, 0)
	for rows.Next() {
		item, scanErr := scanBenchmarkRun(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (r *BenchmarkRepository) UpdateRunStatus(ctx context.Context, tenantID, id uuid.UUID, status aigovmodel.BenchmarkRunStatus, errorMsg *string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE ai_benchmark_runs SET status = $3, error_message = $4
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id, status, errorMsg,
	)
	if err != nil {
		return fmt.Errorf("update ai benchmark run status: %w", err)
	}
	return nil
}

func (r *BenchmarkRepository) UpdateRunResults(ctx context.Context, item *aigovmodel.BenchmarkRun) error {
	_, err := r.db.Exec(ctx, `
		UPDATE ai_benchmark_runs SET
			status = $3,
			p50_latency_ms = $4, p95_latency_ms = $5, p99_latency_ms = $6,
			avg_latency_ms = $7, min_latency_ms = $8, max_latency_ms = $9,
			tokens_per_second = $10, requests_per_second = $11,
			total_tokens = $12, total_requests = $13, failed_requests = $14,
			retried_requests = $15,
			p50_ttft_ms = $16, p95_ttft_ms = $17, avg_ttft_ms = $18,
			avg_perplexity = $19, bleu_score = $20, rouge_l_score = $21,
			semantic_similarity = $22, factual_accuracy = $23,
			peak_cpu_percent = $24, peak_memory_mb = $25,
			avg_cpu_percent = $26, avg_memory_mb = $27,
			estimated_hourly_cost_usd = $28, cost_per_1k_tokens_usd = $29,
			started_at = $30, completed_at = $31, duration_seconds = $32,
			error_message = $33, raw_results = $34
		WHERE tenant_id = $1 AND id = $2`,
		item.TenantID, item.ID, item.Status,
		item.P50LatencyMS, item.P95LatencyMS, item.P99LatencyMS,
		item.AvgLatencyMS, item.MinLatencyMS, item.MaxLatencyMS,
		item.TokensPerSecond, item.RequestsPerSecond,
		item.TotalTokens, item.TotalRequests, item.FailedRequests,
		item.RetriedRequests,
		item.P50TTFT_MS, item.P95TTFT_MS, item.AvgTTFT_MS,
		item.AvgPerplexity, item.BLEUScore, item.ROUGELScore,
		item.SemanticSimilarity, item.FactualAccuracy,
		item.PeakCPUPercent, item.PeakMemoryMB,
		item.AvgCPUPercent, item.AvgMemoryMB,
		item.EstimatedHourlyCostUSD, item.CostPer1kTokensUSD,
		item.StartedAt, item.CompletedAt, item.DurationSeconds,
		item.ErrorMessage, item.RawResults,
	)
	if err != nil {
		return fmt.Errorf("update ai benchmark run results: %w", err)
	}
	return nil
}

func (r *BenchmarkRepository) GetRunsByIDs(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]aigovmodel.BenchmarkRun, error) {
	rows, err := r.db.Query(ctx, benchmarkRunSelectSQL+`
		WHERE tenant_id = $1 AND id = ANY($2)
		ORDER BY created_at DESC`,
		tenantID, ids,
	)
	if err != nil {
		return nil, fmt.Errorf("get benchmark runs by ids: %w", err)
	}
	defer rows.Close()
	items := make([]aigovmodel.BenchmarkRun, 0, len(ids))
	for rows.Next() {
		item, scanErr := scanBenchmarkRun(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

// ── Cost Models ─────────────────────────────────────────────────────────

func (r *BenchmarkRepository) CreateCostModel(ctx context.Context, item *aigovmodel.ComputeCostModel) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO ai_compute_cost_models (
			id, tenant_id, name, backend_type, instance_type, hourly_cost_usd,
			cpu_cores, memory_gb, gpu_type, gpu_count, max_tokens_per_second,
			notes, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)`,
		item.ID, item.TenantID, item.Name, item.BackendType, item.InstanceType,
		item.HourlyCostUSD, item.CPUCores, item.MemoryGB, item.GPUType,
		item.GPUCount, item.MaxTokensPerSecond, item.Notes, item.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert ai compute cost model: %w", err)
	}
	return nil
}

func (r *BenchmarkRepository) ListCostModels(ctx context.Context, tenantID uuid.UUID) ([]aigovmodel.ComputeCostModel, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, name, backend_type, instance_type, hourly_cost_usd,
		       cpu_cores, memory_gb, gpu_type, gpu_count, max_tokens_per_second,
		       notes, created_at
		FROM ai_compute_cost_models
		WHERE tenant_id = $1
		ORDER BY backend_type, hourly_cost_usd`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list ai compute cost models: %w", err)
	}
	defer rows.Close()
	items := make([]aigovmodel.ComputeCostModel, 0)
	for rows.Next() {
		item, scanErr := scanCostModel(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

// ── Scan helpers ────────────────────────────────────────────────────────

type benchmarkScannable interface {
	Scan(dest ...any) error
}

const benchmarkSuiteSelectSQL = `
	SELECT id, tenant_id, name, description, model_slug, prompt_dataset,
	       dataset_size, warmup_count, iteration_count, concurrency,
	       timeout_seconds, stream_enabled, max_retries,
	       created_by, created_at, updated_at
	FROM ai_benchmark_suites`

const benchmarkRunSelectSQL = `
	SELECT id, tenant_id, suite_id, server_id, backend_type, model_name,
	       quantization, status, stream_used,
	       p50_latency_ms, p95_latency_ms, p99_latency_ms,
	       avg_latency_ms, min_latency_ms, max_latency_ms,
	       tokens_per_second, requests_per_second,
	       total_tokens, total_requests, failed_requests,
	       retried_requests,
	       p50_ttft_ms, p95_ttft_ms, avg_ttft_ms,
	       avg_perplexity, bleu_score, rouge_l_score,
	       semantic_similarity, factual_accuracy,
	       peak_cpu_percent, peak_memory_mb, avg_cpu_percent, avg_memory_mb,
	       estimated_hourly_cost_usd, cost_per_1k_tokens_usd,
	       started_at, completed_at, duration_seconds,
	       error_message, raw_results, created_by, created_at
	FROM ai_benchmark_runs`

func scanBenchmarkSuite(row benchmarkScannable) (*aigovmodel.BenchmarkSuite, error) {
	item := &aigovmodel.BenchmarkSuite{}
	var promptDataset []byte
	if err := row.Scan(
		&item.ID, &item.TenantID, &item.Name, &item.Description, &item.ModelSlug,
		&promptDataset, &item.DatasetSize, &item.WarmupCount, &item.IterationCount,
		&item.Concurrency, &item.TimeoutSeconds, &item.StreamEnabled, &item.MaxRetries,
		&item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.PromptDataset = nullJSON(promptDataset, "[]")
	return item, nil
}

func scanBenchmarkRun(row benchmarkScannable) (*aigovmodel.BenchmarkRun, error) {
	item := &aigovmodel.BenchmarkRun{}
	var rawResults []byte
	if err := row.Scan(
		&item.ID, &item.TenantID, &item.SuiteID, &item.ServerID,
		&item.BackendType, &item.ModelName, &item.Quantization,
		&item.Status, &item.StreamUsed,
		&item.P50LatencyMS, &item.P95LatencyMS, &item.P99LatencyMS,
		&item.AvgLatencyMS, &item.MinLatencyMS, &item.MaxLatencyMS,
		&item.TokensPerSecond, &item.RequestsPerSecond,
		&item.TotalTokens, &item.TotalRequests, &item.FailedRequests,
		&item.RetriedRequests,
		&item.P50TTFT_MS, &item.P95TTFT_MS, &item.AvgTTFT_MS,
		&item.AvgPerplexity, &item.BLEUScore, &item.ROUGELScore,
		&item.SemanticSimilarity, &item.FactualAccuracy,
		&item.PeakCPUPercent, &item.PeakMemoryMB, &item.AvgCPUPercent, &item.AvgMemoryMB,
		&item.EstimatedHourlyCostUSD, &item.CostPer1kTokensUSD,
		&item.StartedAt, &item.CompletedAt, &item.DurationSeconds,
		&item.ErrorMessage, &rawResults, &item.CreatedBy, &item.CreatedAt,
	); err != nil {
		return nil, err
	}
	item.RawResults = nullJSON(rawResults, "[]")
	return item, nil
}

func scanCostModel(row benchmarkScannable) (*aigovmodel.ComputeCostModel, error) {
	item := &aigovmodel.ComputeCostModel{}
	if err := row.Scan(
		&item.ID, &item.TenantID, &item.Name, &item.BackendType, &item.InstanceType,
		&item.HourlyCostUSD, &item.CPUCores, &item.MemoryGB, &item.GPUType,
		&item.GPUCount, &item.MaxTokensPerSecond, &item.Notes, &item.CreatedAt,
	); err != nil {
		return nil, err
	}
	return item, nil
}
