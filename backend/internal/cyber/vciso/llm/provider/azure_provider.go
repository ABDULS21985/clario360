package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	llmcfg "github.com/clario360/platform/internal/cyber/vciso/llm"
	llmmodel "github.com/clario360/platform/internal/cyber/vciso/llm/model"
)

type AzureProvider struct {
	model       string
	baseURL     string
	apiKey      string
	apiVersion  string
	deployment  string
	timeout     time.Duration
	temperature float64
	maxTokens   int
	client      *http.Client
}

func NewAzureProvider(cfg llmcfg.ProviderConfig) *AzureProvider {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv(cfg.BaseURLEnv)), "/")
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &AzureProvider{
		model:       cfg.DeploymentName,
		baseURL:     baseURL,
		apiKey:      strings.TrimSpace(os.Getenv(cfg.APIKeyEnv)),
		apiVersion:  cfg.APIVersion,
		deployment:  cfg.DeploymentName,
		timeout:     timeout,
		temperature: cfg.Temperature,
		maxTokens:   cfg.MaxTokens,
		client:      &http.Client{Timeout: timeout},
	}
}

func (p *AzureProvider) Name() string { return "azure" }
func (p *AzureProvider) Model() string { return p.model }
func (p *AzureProvider) SupportsParallelToolCalls() bool { return true }
func (p *AzureProvider) MaxContextTokens() int { return 128000 }
func (p *AzureProvider) EstimateCost(promptTokens, completionTokens int) float64 {
	return float64(promptTokens)*0.0000025 + float64(completionTokens)*0.00001
}

func (p *AzureProvider) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	if p.apiKey == "" || p.baseURL == "" {
		return &HealthStatus{Provider: p.Name(), Model: p.Model(), Status: "unconfigured"}, nil
	}
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/openai/models?api-version="+p.apiVersion, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("api-key", p.apiKey)
	resp, err := p.client.Do(req)
	if err != nil {
		return &HealthStatus{Provider: p.Name(), Model: p.Model(), Status: "unavailable"}, err
	}
	defer resp.Body.Close()
	status := "healthy"
	if resp.StatusCode >= 400 {
		status = "degraded"
	}
	return &HealthStatus{Provider: p.Name(), Model: p.Model(), Status: status, LatencyMS: int(time.Since(start).Milliseconds())}, nil
}

func (p *AzureProvider) Complete(ctx context.Context, request *CompletionRequest) (*CompletionResponse, error) {
	if p.apiKey == "" || p.baseURL == "" {
		return nil, fmt.Errorf("azure openai is not configured")
	}
	body, err := json.Marshal(map[string]any{
		"messages":    p.buildMessages(request.SystemPrompt, request.Messages),
		"tools":       p.buildTools(request.Tools),
		"tool_choice": "auto",
		"max_tokens":  fallbackInt(request.MaxTokens, p.maxTokens, 4096),
		"temperature": fallbackFloat(request.Temperature, p.temperature, 0.1),
	})
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s", p.baseURL, p.deployment, p.apiVersion)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("api-key", p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("azure openai completion failed: %s", strings.TrimSpace(string(respBody)))
	}
	return parseOpenAIResponse(respBody)
}

func (p *AzureProvider) buildMessages(systemPrompt string, items []llmmodel.LLMMessage) []map[string]any {
	return (&OpenAIProvider{}).buildMessages(systemPrompt, items)
}

func (p *AzureProvider) buildTools(items []llmmodel.ToolSchema) []map[string]any {
	return (&OpenAIProvider{}).buildTools(items)
}

func parseOpenAIResponse(payload []byte) (*CompletionResponse, error) {
	var decoded struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	if len(decoded.Choices) == 0 {
		return nil, fmt.Errorf("completion returned no choices")
	}
	out := &CompletionResponse{
		Content:      strings.TrimSpace(decoded.Choices[0].Message.Content),
		FinishReason: decoded.Choices[0].FinishReason,
		Usage: TokenUsage{
			PromptTokens:     decoded.Usage.PromptTokens,
			CompletionTokens: decoded.Usage.CompletionTokens,
			TotalTokens:      decoded.Usage.TotalTokens,
		},
	}
	for _, call := range decoded.Choices[0].Message.ToolCalls {
		args := map[string]any{}
		if strings.TrimSpace(call.Function.Arguments) != "" {
			_ = json.Unmarshal([]byte(call.Function.Arguments), &args)
		}
		out.ToolCalls = append(out.ToolCalls, llmmodel.LLMToolCall{ID: call.ID, FunctionName: call.Function.Name, Arguments: args})
	}
	return out, nil
}
