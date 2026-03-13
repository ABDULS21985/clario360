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

type OpenAIProvider struct {
	name        string
	model       string
	baseURL     string
	apiKey      string
	timeout     time.Duration
	temperature float64
	maxTokens   int
	client      *http.Client
}

func NewOpenAIProvider(cfg llmcfg.ProviderConfig) *OpenAIProvider {
	apiKey := strings.TrimSpace(os.Getenv(cfg.APIKeyEnv))
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &OpenAIProvider{
		name:        "openai",
		model:       cfg.Model,
		baseURL:     baseURL,
		apiKey:      apiKey,
		timeout:     timeout,
		temperature: cfg.Temperature,
		maxTokens:   cfg.MaxTokens,
		client:      &http.Client{Timeout: timeout},
	}
}

func (p *OpenAIProvider) Name() string { return p.name }
func (p *OpenAIProvider) Model() string { return p.model }
func (p *OpenAIProvider) SupportsParallelToolCalls() bool { return true }
func (p *OpenAIProvider) MaxContextTokens() int { return 128000 }

func (p *OpenAIProvider) EstimateCost(promptTokens, completionTokens int) float64 {
	return float64(promptTokens)*0.0000025 + float64(completionTokens)*0.00001
}

func (p *OpenAIProvider) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	if strings.TrimSpace(p.apiKey) == "" && p.name != "local" {
		return &HealthStatus{Provider: p.Name(), Model: p.Model(), Status: "unconfigured"}, nil
	}
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return &HealthStatus{Provider: p.Name(), Model: p.Model(), Status: "unavailable"}, err
	}
	defer resp.Body.Close()
	status := "healthy"
	if resp.StatusCode >= 400 {
		status = "degraded"
	}
	return &HealthStatus{
		Provider:           p.Name(),
		Model:              p.Model(),
		Status:             status,
		LatencyMS:          int(time.Since(start).Milliseconds()),
		RateLimitRemaining: parseRemaining(resp.Header.Get("x-ratelimit-remaining-requests")),
	}, nil
}

func (p *OpenAIProvider) Complete(ctx context.Context, request *CompletionRequest) (*CompletionResponse, error) {
	if strings.TrimSpace(p.apiKey) == "" && p.name != "local" {
		return nil, fmt.Errorf("openai api key is not configured")
	}
	if request == nil {
		return nil, fmt.Errorf("completion request is required")
	}
	payload := map[string]any{
		"model":       p.model,
		"messages":    p.buildMessages(request.SystemPrompt, request.Messages),
		"tools":       p.buildTools(request.Tools),
		"tool_choice": "auto",
		"max_tokens":  fallbackInt(request.MaxTokens, p.maxTokens, 4096),
		"temperature": fallbackFloat(request.Temperature, p.temperature, 0.1),
	}
	if request.TopP > 0 {
		payload["top_p"] = request.TopP
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("openai completion failed: %s", strings.TrimSpace(string(respBody)))
	}
	var decoded struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
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
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return nil, err
	}
	if len(decoded.Choices) == 0 {
		return nil, fmt.Errorf("openai completion returned no choices")
	}
	choice := decoded.Choices[0]
	out := &CompletionResponse{
		Content:      strings.TrimSpace(choice.Message.Content),
		FinishReason: choice.FinishReason,
		Usage: TokenUsage{
			PromptTokens:     decoded.Usage.PromptTokens,
			CompletionTokens: decoded.Usage.CompletionTokens,
			TotalTokens:      decoded.Usage.TotalTokens,
		},
	}
	for _, call := range choice.Message.ToolCalls {
		arguments := map[string]any{}
		if strings.TrimSpace(call.Function.Arguments) != "" {
			_ = json.Unmarshal([]byte(call.Function.Arguments), &arguments)
		}
		out.ToolCalls = append(out.ToolCalls, llmmodel.LLMToolCall{
			ID:           call.ID,
			FunctionName: call.Function.Name,
			Arguments:    arguments,
		})
	}
	return out, nil
}

func (p *OpenAIProvider) buildMessages(systemPrompt string, items []llmmodel.LLMMessage) []map[string]any {
	messages := make([]map[string]any, 0, len(items)+1)
	if strings.TrimSpace(systemPrompt) != "" {
		messages = append(messages, map[string]any{"role": "system", "content": systemPrompt})
	}
	for _, item := range items {
		msg := map[string]any{
			"role":    item.Role,
			"content": item.Content,
		}
		if item.Name != "" {
			msg["name"] = item.Name
		}
		if item.ToolCallID != "" {
			msg["tool_call_id"] = item.ToolCallID
		}
		messages = append(messages, msg)
	}
	return messages
}

func (p *OpenAIProvider) buildTools(items []llmmodel.ToolSchema) []map[string]any {
	tools := make([]map[string]any, 0, len(items))
	for _, item := range items {
		tools = append(tools, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        item.Name,
				"description": item.Description,
				"parameters":  item.Parameters,
			},
		})
	}
	return tools
}

func fallbackInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func fallbackFloat(values ...float64) float64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func parseRemaining(value string) int {
	var remaining int
	_, _ = fmt.Sscanf(strings.TrimSpace(value), "%d", &remaining)
	return remaining
}
