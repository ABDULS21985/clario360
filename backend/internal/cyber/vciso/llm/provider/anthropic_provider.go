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

type AnthropicProvider struct {
	model       string
	baseURL     string
	apiKey      string
	timeout     time.Duration
	temperature float64
	maxTokens   int
	client      *http.Client
}

func NewAnthropicProvider(cfg llmcfg.ProviderConfig) *AnthropicProvider {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	return &AnthropicProvider{
		model:       cfg.Model,
		baseURL:     baseURL,
		apiKey:      strings.TrimSpace(os.Getenv(cfg.APIKeyEnv)),
		timeout:     timeout,
		temperature: cfg.Temperature,
		maxTokens:   cfg.MaxTokens,
		client:      &http.Client{Timeout: timeout},
	}
}

func (p *AnthropicProvider) Name() string { return "anthropic" }
func (p *AnthropicProvider) Model() string { return p.model }
func (p *AnthropicProvider) SupportsParallelToolCalls() bool { return true }
func (p *AnthropicProvider) MaxContextTokens() int { return 200000 }
func (p *AnthropicProvider) EstimateCost(promptTokens, completionTokens int) float64 {
	return float64(promptTokens)*0.000003 + float64(completionTokens)*0.000015
}

func (p *AnthropicProvider) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	if p.apiKey == "" {
		return &HealthStatus{Provider: p.Name(), Model: p.Model(), Status: "unconfigured"}, nil
	}
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
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

func (p *AnthropicProvider) Complete(ctx context.Context, request *CompletionRequest) (*CompletionResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("anthropic api key is not configured")
	}
	payload := map[string]any{
		"model":       p.model,
		"system":      request.SystemPrompt,
		"messages":    p.buildMessages(request.Messages),
		"tools":       p.buildTools(request.Tools),
		"max_tokens":  fallbackInt(request.MaxTokens, p.maxTokens, 4096),
		"temperature": fallbackFloat(request.Temperature, p.temperature, 0.1),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")
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
		return nil, fmt.Errorf("anthropic completion failed: %s", strings.TrimSpace(string(respBody)))
	}
	var decoded struct {
		StopReason string `json:"stop_reason"`
		Content []struct {
			Type  string `json:"type"`
			Text  string `json:"text,omitempty"`
			ID    string `json:"id,omitempty"`
			Name  string `json:"name,omitempty"`
			Input map[string]any `json:"input,omitempty"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return nil, err
	}
	out := &CompletionResponse{
		FinishReason: decoded.StopReason,
		Usage: TokenUsage{
			PromptTokens:     decoded.Usage.InputTokens,
			CompletionTokens: decoded.Usage.OutputTokens,
			TotalTokens:      decoded.Usage.InputTokens + decoded.Usage.OutputTokens,
		},
	}
	textParts := make([]string, 0, len(decoded.Content))
	for _, item := range decoded.Content {
		switch item.Type {
		case "text":
			if strings.TrimSpace(item.Text) != "" {
				textParts = append(textParts, item.Text)
			}
		case "tool_use":
			out.ToolCalls = append(out.ToolCalls, llmmodel.LLMToolCall{
				ID:           item.ID,
				FunctionName: item.Name,
				Arguments:    item.Input,
			})
		}
	}
	out.Content = strings.TrimSpace(strings.Join(textParts, "\n"))
	return out, nil
}

func (p *AnthropicProvider) buildMessages(items []llmmodel.LLMMessage) []map[string]any {
	messages := make([]map[string]any, 0, len(items))
	for _, item := range items {
		content := []map[string]any{{"type": "text", "text": item.Content}}
		if item.Role == "tool" {
			content = []map[string]any{{
				"type":       "tool_result",
				"tool_use_id": item.ToolCallID,
				"content":    item.Content,
			}}
		}
		messages = append(messages, map[string]any{
			"role":    normalizeAnthropicRole(item.Role),
			"content": content,
		})
	}
	return messages
}

func (p *AnthropicProvider) buildTools(items []llmmodel.ToolSchema) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"name":         item.Name,
			"description":  item.Description,
			"input_schema": item.Parameters,
		})
	}
	return out
}

func normalizeAnthropicRole(role string) string {
	switch role {
	case "assistant", "user":
		return role
	default:
		return "user"
	}
}
