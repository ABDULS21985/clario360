package provider

import (
	"context"

	llmmodel "github.com/clario360/platform/internal/cyber/vciso/llm/model"
)

type LLMProvider interface {
	Complete(ctx context.Context, request *CompletionRequest) (*CompletionResponse, error)
	Name() string
	Model() string
	SupportsParallelToolCalls() bool
	MaxContextTokens() int
	EstimateCost(promptTokens, completionTokens int) float64
	HealthCheck(ctx context.Context) (*HealthStatus, error)
}

type CompletionRequest struct {
	SystemPrompt   string
	Messages       []llmmodel.LLMMessage
	Tools          []llmmodel.ToolSchema
	MaxTokens      int
	Temperature    float64
	TopP           float64
	ResponseFormat string
}

type CompletionResponse struct {
	Content      string                 `json:"content"`
	ToolCalls    []llmmodel.LLMToolCall `json:"tool_calls,omitempty"`
	FinishReason string                 `json:"finish_reason"`
	Usage        TokenUsage             `json:"usage"`
}

type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type HealthStatus struct {
	Provider           string `json:"provider"`
	Model              string `json:"model"`
	Status             string `json:"status"`
	LatencyMS          int    `json:"latency_ms"`
	RateLimitRemaining int    `json:"rate_limit_remaining"`
}
