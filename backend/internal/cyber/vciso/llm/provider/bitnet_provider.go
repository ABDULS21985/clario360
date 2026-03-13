package provider

import (
	"net/http"
	"strings"
	"time"

	llmcfg "github.com/clario360/platform/internal/cyber/vciso/llm"
)

// BitNetProvider wraps a BitNet 1-bit model served via llama.cpp's
// OpenAI-compatible server. BitNet models use 1.58-bit quantisation
// ({-1, 0, 1} ternary weights) enabling pure-CPU inference via
// integer addition — no matrix multiplication required.
type BitNetProvider struct {
	*OpenAIProvider
}

func NewBitNetProvider(cfg llmcfg.ProviderConfig) *BitNetProvider {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "http://localhost:8082/v1"
	}
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second // BitNet inference is slower; allow longer timeout.
	}
	return &BitNetProvider{
		OpenAIProvider: &OpenAIProvider{
			name:        "bitnet",
			model:       cfg.Model,
			baseURL:     baseURL,
			timeout:     timeout,
			temperature: cfg.Temperature,
			maxTokens:   cfg.MaxTokens,
			client:      &http.Client{Timeout: timeout},
		},
	}
}

func (p *BitNetProvider) Name() string { return "bitnet" }

// MaxContextTokens for current BitNet models is typically 4096.
func (p *BitNetProvider) MaxContextTokens() int { return 4096 }

func (p *BitNetProvider) SupportsParallelToolCalls() bool { return false }

// EstimateCost is dramatically lower — CPU-only, no GPU rental.
// Approximate: $0.10/hr for 8-core CPU → ~72K tokens/hr → ~$0.0000014/token.
func (p *BitNetProvider) EstimateCost(promptTokens, completionTokens int) float64 {
	return float64(promptTokens+completionTokens) * 0.0000014
}
