package dto

import (
	"encoding/json"

	"github.com/google/uuid"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
)

// ── Inference Servers ───────────────────────────────────────────────────

type CreateInferenceServerRequest struct {
	Name           string                        `json:"name"`
	BackendType    aigovmodel.ComputeBackendType `json:"backend_type"`
	BaseURL        string                        `json:"base_url"`
	HealthEndpoint string                        `json:"health_endpoint"`
	ModelName      *string                       `json:"model_name"`
	APIKey         *string                       `json:"api_key,omitempty"`
	Quantization   *string                       `json:"quantization"`
	CPUCores       *int                          `json:"cpu_cores"`
	MemoryMB       *int                          `json:"memory_mb"`
	GPUType        *string                       `json:"gpu_type"`
	GPUCount       int                           `json:"gpu_count"`
	MaxConcurrent  int                           `json:"max_concurrent"`
	StreamCapable  bool                          `json:"stream_capable"`
	Metadata       json.RawMessage               `json:"metadata"`
}

// UpdateInferenceServerRequest uses pointer fields so callers can perform
// partial (PATCH-style) updates — only non-nil fields are applied.
type UpdateInferenceServerRequest struct {
	Name           *string          `json:"name,omitempty"`
	BaseURL        *string          `json:"base_url,omitempty"`
	HealthEndpoint *string          `json:"health_endpoint,omitempty"`
	ModelName      *string          `json:"model_name,omitempty"`
	APIKey         *string          `json:"api_key,omitempty"`
	Quantization   *string          `json:"quantization,omitempty"`
	CPUCores       *int             `json:"cpu_cores,omitempty"`
	MemoryMB       *int             `json:"memory_mb,omitempty"`
	GPUType        *string          `json:"gpu_type,omitempty"`
	GPUCount       *int             `json:"gpu_count,omitempty"`
	MaxConcurrent  *int             `json:"max_concurrent,omitempty"`
	StreamCapable  *bool            `json:"stream_capable,omitempty"`
	Metadata       *json.RawMessage `json:"metadata,omitempty"`
}

type UpdateInferenceServerStatusRequest struct {
	Status aigovmodel.InferenceServerStatus `json:"status"`
}

// ── Benchmark Suites ────────────────────────────────────────────────────

type CreateBenchmarkSuiteRequest struct {
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	ModelSlug      string          `json:"model_slug"`
	PromptDataset  json.RawMessage `json:"prompt_dataset"`
	WarmupCount    int             `json:"warmup_count"`
	IterationCount int             `json:"iteration_count"`
	Concurrency    int             `json:"concurrency"`
	TimeoutSeconds int             `json:"timeout_seconds"`
	StreamEnabled  bool            `json:"stream_enabled"`
	MaxRetries     int             `json:"max_retries"`
}

// UpdateBenchmarkSuiteRequest uses pointer fields for partial updates.
type UpdateBenchmarkSuiteRequest struct {
	Name           *string          `json:"name,omitempty"`
	Description    *string          `json:"description,omitempty"`
	PromptDataset  *json.RawMessage `json:"prompt_dataset,omitempty"`
	WarmupCount    *int             `json:"warmup_count,omitempty"`
	IterationCount *int             `json:"iteration_count,omitempty"`
	Concurrency    *int             `json:"concurrency,omitempty"`
	TimeoutSeconds *int             `json:"timeout_seconds,omitempty"`
	StreamEnabled  *bool            `json:"stream_enabled,omitempty"`
	MaxRetries     *int             `json:"max_retries,omitempty"`
}

// ── Benchmark Runs ──────────────────────────────────────────────────────

type RunBenchmarkRequest struct {
	ServerID uuid.UUID `json:"server_id"`
}

type CompareRunsRequest struct {
	RunIDs []uuid.UUID `json:"run_ids"`
}

// ── Cost Models ─────────────────────────────────────────────────────────

type CreateComputeCostModelRequest struct {
	Name               string                        `json:"name"`
	BackendType        aigovmodel.ComputeBackendType `json:"backend_type"`
	InstanceType       string                        `json:"instance_type"`
	HourlyCostUSD      float64                       `json:"hourly_cost_usd"`
	CPUCores           *int                          `json:"cpu_cores"`
	MemoryGB           *int                          `json:"memory_gb"`
	GPUType            *string                       `json:"gpu_type"`
	GPUCount           int                           `json:"gpu_count"`
	MaxTokensPerSecond *float64                      `json:"max_tokens_per_second"`
	Notes              *string                       `json:"notes"`
}

type EstimateCostSavingsRequest struct {
	CPURunID uuid.UUID `json:"cpu_run_id"`
	GPURunID uuid.UUID `json:"gpu_run_id"`
}
