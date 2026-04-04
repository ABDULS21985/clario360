package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ComputeBackendType identifies the inference runtime for a model.
type ComputeBackendType string

const (
	ComputeBackendInlineGo    ComputeBackendType = "inline_go"
	ComputeBackendVLLMGPU     ComputeBackendType = "vllm_gpu"
	ComputeBackendVLLMCPU     ComputeBackendType = "vllm_cpu"
	ComputeBackendLlamaCppCPU ComputeBackendType = "llamacpp_cpu"
	ComputeBackendLlamaCppGPU ComputeBackendType = "llamacpp_gpu"
	ComputeBackendBitNetCPU   ComputeBackendType = "bitnet_cpu"
	ComputeBackendONNXCPU     ComputeBackendType = "onnx_cpu"
	ComputeBackendONNXGPU     ComputeBackendType = "onnx_gpu"
)

// Additional artifact types for quantised / 1-bit models.
const (
	ArtifactTypeGGUFModel   ArtifactType = "gguf_model"
	ArtifactTypeBitNetModel ArtifactType = "bitnet_model"
	ArtifactTypeONNXModel   ArtifactType = "onnx_model"
)

// InferenceServerStatus represents the health of an inference endpoint.
type InferenceServerStatus string

const (
	ServerStatusProvisioning   InferenceServerStatus = "provisioning"
	ServerStatusHealthy        InferenceServerStatus = "healthy"
	ServerStatusDegraded       InferenceServerStatus = "degraded"
	ServerStatusOffline        InferenceServerStatus = "offline"
	ServerStatusDecommissioned InferenceServerStatus = "decommissioned"
)

// InferenceServer is a registered inference endpoint.
type InferenceServer struct {
	ID             uuid.UUID             `json:"id" db:"id"`
	TenantID       uuid.UUID             `json:"tenant_id" db:"tenant_id"`
	Name           string                `json:"name" db:"name"`
	BackendType    ComputeBackendType    `json:"backend_type" db:"backend_type"`
	BaseURL        string                `json:"base_url" db:"base_url"`
	HealthEndpoint string                `json:"health_endpoint" db:"health_endpoint"`
	ModelName      *string               `json:"model_name,omitempty" db:"model_name"`
	APIKey         *string               `json:"api_key,omitempty" db:"api_key"`
	Quantization   *string               `json:"quantization,omitempty" db:"quantization"`
	Status         InferenceServerStatus `json:"status" db:"status"`
	CPUCores       *int                  `json:"cpu_cores,omitempty" db:"cpu_cores"`
	MemoryMB       *int                  `json:"memory_mb,omitempty" db:"memory_mb"`
	GPUType        *string               `json:"gpu_type,omitempty" db:"gpu_type"`
	GPUCount       int                   `json:"gpu_count" db:"gpu_count"`
	MaxConcurrent  int                   `json:"max_concurrent" db:"max_concurrent"`
	StreamCapable  bool                  `json:"stream_capable" db:"stream_capable"`
	Metadata       json.RawMessage       `json:"metadata" db:"metadata"`
	CreatedAt      time.Time             `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at" db:"updated_at"`
}

// BenchmarkSuite defines a reusable set of prompts and configuration for
// benchmarking inference backends.
type BenchmarkSuite struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	TenantID       uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	Name           string          `json:"name" db:"name"`
	Description    string          `json:"description" db:"description"`
	ModelSlug      string          `json:"model_slug" db:"model_slug"`
	PromptDataset  json.RawMessage `json:"prompt_dataset" db:"prompt_dataset"`
	DatasetSize    int             `json:"dataset_size" db:"dataset_size"`
	WarmupCount    int             `json:"warmup_count" db:"warmup_count"`
	IterationCount int             `json:"iteration_count" db:"iteration_count"`
	Concurrency    int             `json:"concurrency" db:"concurrency"`
	TimeoutSeconds int             `json:"timeout_seconds" db:"timeout_seconds"`
	StreamEnabled  bool            `json:"stream_enabled" db:"stream_enabled"`
	MaxRetries     int             `json:"max_retries" db:"max_retries"`
	CreatedBy      uuid.UUID       `json:"created_by" db:"created_by"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt      *time.Time      `json:"updated_at,omitempty" db:"updated_at"`
}

// BenchmarkRunStatus tracks the lifecycle of a single benchmark execution.
type BenchmarkRunStatus string

const (
	BenchmarkStatusPending   BenchmarkRunStatus = "pending"
	BenchmarkStatusRunning   BenchmarkRunStatus = "running"
	BenchmarkStatusCompleted BenchmarkRunStatus = "completed"
	BenchmarkStatusFailed    BenchmarkRunStatus = "failed"
	BenchmarkStatusCancelled BenchmarkRunStatus = "cancelled"
)

// BenchmarkRun stores results from one benchmark execution against a server.
type BenchmarkRun struct {
	ID           uuid.UUID          `json:"id" db:"id"`
	TenantID     uuid.UUID          `json:"tenant_id" db:"tenant_id"`
	SuiteID      uuid.UUID          `json:"suite_id" db:"suite_id"`
	ServerID     uuid.UUID          `json:"server_id" db:"server_id"`
	BackendType  ComputeBackendType `json:"backend_type" db:"backend_type"`
	ModelName    string             `json:"model_name" db:"model_name"`
	Quantization *string            `json:"quantization,omitempty" db:"quantization"`
	Status       BenchmarkRunStatus `json:"status" db:"status"`
	StreamUsed   bool               `json:"stream_used" db:"stream_used"`

	// Latency metrics.
	P50LatencyMS *float64 `json:"p50_latency_ms,omitempty" db:"p50_latency_ms"`
	P95LatencyMS *float64 `json:"p95_latency_ms,omitempty" db:"p95_latency_ms"`
	P99LatencyMS *float64 `json:"p99_latency_ms,omitempty" db:"p99_latency_ms"`
	AvgLatencyMS *float64 `json:"avg_latency_ms,omitempty" db:"avg_latency_ms"`
	MinLatencyMS *float64 `json:"min_latency_ms,omitempty" db:"min_latency_ms"`
	MaxLatencyMS *float64 `json:"max_latency_ms,omitempty" db:"max_latency_ms"`

	// Throughput metrics.
	TokensPerSecond   *float64 `json:"tokens_per_second,omitempty" db:"tokens_per_second"`
	RequestsPerSecond *float64 `json:"requests_per_second,omitempty" db:"requests_per_second"`
	TotalTokens       *int64   `json:"total_tokens,omitempty" db:"total_tokens"`
	TotalRequests     *int     `json:"total_requests,omitempty" db:"total_requests"`
	FailedRequests    int      `json:"failed_requests" db:"failed_requests"`
	RetriedRequests   int      `json:"retried_requests" db:"retried_requests"`

	// Time-to-first-token metrics (populated only when streaming is used).
	P50TTFT_MS *float64 `json:"p50_ttft_ms,omitempty" db:"p50_ttft_ms"`
	P95TTFT_MS *float64 `json:"p95_ttft_ms,omitempty" db:"p95_ttft_ms"`
	AvgTTFT_MS *float64 `json:"avg_ttft_ms,omitempty" db:"avg_ttft_ms"`

	// Quality metrics.
	AvgPerplexity      *float64 `json:"avg_perplexity,omitempty" db:"avg_perplexity"`
	BLEUScore          *float64 `json:"bleu_score,omitempty" db:"bleu_score"`
	ROUGELScore        *float64 `json:"rouge_l_score,omitempty" db:"rouge_l_score"`
	SemanticSimilarity *float64 `json:"semantic_similarity,omitempty" db:"semantic_similarity"`
	FactualAccuracy    *float64 `json:"factual_accuracy,omitempty" db:"factual_accuracy"`

	// Resource utilisation.
	PeakCPUPercent *float64 `json:"peak_cpu_percent,omitempty" db:"peak_cpu_percent"`
	PeakMemoryMB   *int     `json:"peak_memory_mb,omitempty" db:"peak_memory_mb"`
	AvgCPUPercent  *float64 `json:"avg_cpu_percent,omitempty" db:"avg_cpu_percent"`
	AvgMemoryMB    *int     `json:"avg_memory_mb,omitempty" db:"avg_memory_mb"`

	// Cost estimation.
	EstimatedHourlyCostUSD *float64 `json:"estimated_hourly_cost_usd,omitempty" db:"estimated_hourly_cost_usd"`
	CostPer1kTokensUSD     *float64 `json:"cost_per_1k_tokens_usd,omitempty" db:"cost_per_1k_tokens_usd"`

	// Timing & lifecycle.
	StartedAt       *time.Time      `json:"started_at,omitempty" db:"started_at"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
	DurationSeconds *int            `json:"duration_seconds,omitempty" db:"duration_seconds"`
	ErrorMessage    *string         `json:"error_message,omitempty" db:"error_message"`
	RawResults      json.RawMessage `json:"raw_results" db:"raw_results"`
	CreatedBy       uuid.UUID       `json:"created_by" db:"created_by"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
}

// ComputeCostModel holds pricing data for a specific infrastructure configuration.
type ComputeCostModel struct {
	ID                 uuid.UUID          `json:"id" db:"id"`
	TenantID           uuid.UUID          `json:"tenant_id" db:"tenant_id"`
	Name               string             `json:"name" db:"name"`
	BackendType        ComputeBackendType `json:"backend_type" db:"backend_type"`
	InstanceType       string             `json:"instance_type" db:"instance_type"`
	HourlyCostUSD      float64            `json:"hourly_cost_usd" db:"hourly_cost_usd"`
	CPUCores           *int               `json:"cpu_cores,omitempty" db:"cpu_cores"`
	MemoryGB           *int               `json:"memory_gb,omitempty" db:"memory_gb"`
	GPUType            *string            `json:"gpu_type,omitempty" db:"gpu_type"`
	GPUCount           int                `json:"gpu_count" db:"gpu_count"`
	MaxTokensPerSecond *float64           `json:"max_tokens_per_second,omitempty" db:"max_tokens_per_second"`
	Notes              *string            `json:"notes,omitempty" db:"notes"`
	CreatedAt          time.Time          `json:"created_at" db:"created_at"`
}

// BenchmarkComparison aggregates multiple runs for side-by-side analysis.
type BenchmarkComparison struct {
	Runs                 []BenchmarkRun `json:"runs"`
	CostDeltaMonthlyUSD  float64        `json:"cost_delta_monthly_usd"`
	LatencyDeltaPct      float64        `json:"latency_delta_percent"`
	QualityDeltaPct      float64        `json:"quality_delta_percent"`
	Recommendation       string         `json:"recommendation"`
	RecommendationReason string         `json:"recommendation_reason"`
}
