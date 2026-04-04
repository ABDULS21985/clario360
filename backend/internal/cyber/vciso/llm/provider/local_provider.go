package provider

import (
	"net/http"
	"os"
	"strings"
	"time"

	llmcfg "github.com/clario360/platform/internal/cyber/vciso/llm"
)

type LocalProvider struct {
	*OpenAIProvider
}

func NewLocalProvider(cfg llmcfg.ProviderConfig) *LocalProvider {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &LocalProvider{
		OpenAIProvider: &OpenAIProvider{
			name:        "local",
			model:       cfg.Model,
			baseURL:     strings.TrimRight(cfg.BaseURL, "/"),
			apiKey:      strings.TrimSpace(os.Getenv(cfg.APIKeyEnv)),
			timeout:     timeout,
			temperature: cfg.Temperature,
			maxTokens:   cfg.MaxTokens,
			client:      &http.Client{Timeout: timeout},
		},
	}
}
