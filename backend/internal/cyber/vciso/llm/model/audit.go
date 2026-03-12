package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID                 uuid.UUID        `json:"id"`
	MessageID          uuid.UUID        `json:"message_id"`
	ConversationID     uuid.UUID        `json:"conversation_id"`
	TenantID           uuid.UUID        `json:"tenant_id"`
	UserID             uuid.UUID        `json:"user_id"`
	Provider           string           `json:"provider"`
	Model              string           `json:"model"`
	PromptTokens       int              `json:"prompt_tokens"`
	CompletionTokens   int              `json:"completion_tokens"`
	TotalTokens        int              `json:"total_tokens"`
	EstimatedCostUSD   float64          `json:"estimated_cost_usd"`
	LLMLatencyMS       int              `json:"llm_latency_ms"`
	TotalLatencyMS     int              `json:"total_latency_ms"`
	SystemPromptHash   string           `json:"system_prompt_hash"`
	SystemPromptVersion string          `json:"system_prompt_version"`
	UserMessage        string           `json:"user_message"`
	ContextTurns       int              `json:"context_turns"`
	RawCompletion      string           `json:"raw_completion"`
	ToolCallsJSON      json.RawMessage  `json:"tool_calls_json"`
	ToolCallCount      int              `json:"tool_call_count"`
	ReasoningTrace     json.RawMessage  `json:"reasoning_trace"`
	GroundingResult    string           `json:"grounding_result"`
	PIIDetections      int              `json:"pii_detections"`
	InjectionFlags     int              `json:"injection_flags"`
	FinalResponse      string           `json:"final_response"`
	PredictionLogID    *uuid.UUID       `json:"prediction_log_id,omitempty"`
	EngineUsed         string           `json:"engine_used"`
	RoutingReason      string           `json:"routing_reason,omitempty"`
	CreatedAt          time.Time        `json:"created_at"`
}

type SystemPrompt struct {
	ID          uuid.UUID       `json:"id"`
	Version     string          `json:"version"`
	PromptText  string          `json:"prompt_text"`
	PromptHash  string          `json:"prompt_hash"`
	ToolSchemas json.RawMessage `json:"tool_schemas"`
	Description *string         `json:"description,omitempty"`
	CreatedBy   string          `json:"created_by"`
	Active      bool            `json:"active"`
	CreatedAt   time.Time       `json:"created_at"`
}

type RateLimitRecord struct {
	ID                 uuid.UUID `json:"id"`
	TenantID           uuid.UUID `json:"tenant_id"`
	MaxCallsPerMinute  int       `json:"max_calls_per_minute"`
	MaxCallsPerHour    int       `json:"max_calls_per_hour"`
	MaxCallsPerDay     int       `json:"max_calls_per_day"`
	MaxTokensPerDay    int       `json:"max_tokens_per_day"`
	MaxCostPerDayUSD   float64   `json:"max_cost_per_day_usd"`
	CurrentCallsMinute int       `json:"current_calls_minute"`
	CurrentCallsHour   int       `json:"current_calls_hour"`
	CurrentCallsDay    int       `json:"current_calls_day"`
	CurrentTokensDay   int       `json:"current_tokens_day"`
	CurrentCostDayUSD  float64   `json:"current_cost_day_usd"`
	MinuteResetAt      time.Time `json:"minute_reset_at"`
	HourResetAt        time.Time `json:"hour_reset_at"`
	DayResetAt         time.Time `json:"day_reset_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type UsageStats struct {
	CallsToday     int
	TokensToday    int
	CostToday      float64
	CallsThisMonth int
	CostThisMonth  float64
}
