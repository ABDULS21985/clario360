package dto

import (
	"time"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

type ChatRequest struct {
	ConversationID *uuid.UUID `json:"conversation_id,omitempty"`
	Message        string     `json:"message"`
	PreferEngine   string     `json:"prefer_engine,omitempty"`
}

type ChatResponse struct {
	ConversationID uuid.UUID                 `json:"conversation_id"`
	MessageID      uuid.UUID                 `json:"message_id"`
	Response       chatmodel.ResponsePayload `json:"response"`
	Intent         string                    `json:"intent"`
	Confidence     float64                   `json:"confidence"`
	Engine         string                    `json:"engine,omitempty"`
	Meta           *ResponseMeta             `json:"meta,omitempty"`
}

type ResponseMeta struct {
	Intent             string  `json:"intent"`
	Confidence         float64 `json:"confidence"`
	ToolCallsCount     int     `json:"tool_calls_count,omitempty"`
	ReasoningSteps     int     `json:"reasoning_steps,omitempty"`
	LatencyMS          int     `json:"latency_ms,omitempty"`
	SynthesisLatencyMs int64   `json:"synthesis_latency_ms,omitempty"`
	TokensUsed         int     `json:"tokens_used,omitempty"`
	Grounding          string  `json:"grounding,omitempty"`
	Engine             string  `json:"engine,omitempty"`
	RoutingReason      string  `json:"routing_reason,omitempty"`
}

type Suggestion struct {
	Text     string `json:"text"`
	Category string `json:"category"`
	Priority int    `json:"priority"`
	Reason   string `json:"reason"`
}

type SuggestionResponse struct {
	Suggestions []Suggestion `json:"suggestions"`
}

type ConversationListItem struct {
	ID            uuid.UUID  `json:"id"`
	Title         string     `json:"title"`
	Status        string     `json:"status"`
	MessageCount  int        `json:"message_count"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type ConversationDetail struct {
	ID            uuid.UUID             `json:"id"`
	Title         string                `json:"title"`
	Status        string                `json:"status"`
	MessageCount  int                   `json:"message_count"`
	LastMessageAt *time.Time            `json:"last_message_at,omitempty"`
	CreatedAt     time.Time             `json:"created_at"`
	Messages      []ConversationMessage `json:"messages"`
}

type ConversationMessage struct {
	ID           uuid.UUID                   `json:"id"`
	Role         string                      `json:"role"`
	Content      string                      `json:"content"`
	Intent       *string                     `json:"intent,omitempty"`
	ResponseType *string                     `json:"response_type,omitempty"`
	Actions      []chatmodel.SuggestedAction `json:"actions"`
	ToolResult   any                         `json:"tool_result,omitempty"`
	Engine       string                      `json:"engine,omitempty"`
	CreatedAt    time.Time                   `json:"created_at"`
}
