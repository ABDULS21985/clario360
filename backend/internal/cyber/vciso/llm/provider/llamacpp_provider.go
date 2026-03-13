package provider

import (
	"net/http"
	"strings"
	"time"

	llmcfg "github.com/clario360/platform/internal/cyber/vciso/llm"
)

// LlamaCppProvider wraps an OpenAI-compatible llama.cpp server.
// llama.cpp's built-in server exposes /v1/chat/completions with the
// same schema as the OpenAI API, so the only differences from
// LocalProvider are the identity fields and cost model.
type LlamaCppProvider struct {
	*OpenAIProvider
}

func NewLlamaCppProvider(cfg llmcfg.ProviderConfig) *LlamaCppProvider {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "http://localhost:8081/v1"
	}
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &LlamaCppProvider{
		OpenAIProvider: &OpenAIProvider{
			name:        "llamacpp",
			model:       cfg.Model,
			baseURL:     baseURL,
			timeout:     timeout,
			temperature: cfg.Temperature,
			maxTokens:   cfg.MaxTokens,
			client:      &http.Client{Timeout: timeout},
		},
	}
}

func (p *LlamaCppProvider) Name() string { return "llamacpp" }

func (p *LlamaCppProvider) MaxContextTokens() int { return 32768 }

func (p *LlamaCppProvider) SupportsParallelToolCalls() bool { return false }

// EstimateCost returns a CPU-based cost estimate which is roughly 10x
// cheaper than GPU inference.
func (p *LlamaCppProvider) EstimateCost(promptTokens, completionTokens int) float64 {
	return float64(promptTokens)*0.00000025 + float64(completionTokens)*0.000001
}
