package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/vciso/llm/model"
)

type AuditResponse struct {
	MessageID       uuid.UUID               `json:"message_id"`
	Provider        string                  `json:"provider"`
	Model           string                  `json:"model"`
	PromptTokens    int                     `json:"prompt_tokens"`
	CompletionTokens int                    `json:"completion_tokens"`
	TotalTokens     int                     `json:"total_tokens"`
	ToolCalls       []model.ToolCallAudit   `json:"tool_calls"`
	ReasoningTrace  []model.ReasoningStep   `json:"reasoning_trace"`
	GroundingResult string                  `json:"grounding_result"`
	EngineUsed      string                  `json:"engine_used"`
	RoutingReason   string                  `json:"routing_reason,omitempty"`
	CreatedAt       time.Time               `json:"created_at"`
}

type UsageResponse struct {
	CallsToday      int     `json:"calls_today"`
	TokensToday     int     `json:"tokens_today"`
	CostToday       float64 `json:"cost_today"`
	CallsThisMonth  int     `json:"calls_this_month"`
	CostThisMonth   float64 `json:"cost_this_month"`
}

type HealthResponse struct {
	Provider            string `json:"provider"`
	Model               string `json:"model"`
	Status              string `json:"status"`
	LatencyMS           int    `json:"latency_ms"`
	RateLimitRemaining  int    `json:"rate_limit_remaining"`
}

type UpdateConfigRequest struct {
	Provider    string  `json:"provider"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
}

type PromptVersionRequest struct {
	Version     string `json:"version"`
	PromptText  string `json:"prompt_text"`
	Description string `json:"description"`
}

type PromptVersionResponse struct {
	ID          uuid.UUID `json:"id"`
	Version     string    `json:"version"`
	Description string    `json:"description,omitempty"`
	Active      bool      `json:"active"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}
