package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ConversationStatus string

const (
	ConversationStatusActive   ConversationStatus = "active"
	ConversationStatusArchived ConversationStatus = "archived"
	ConversationStatusDeleted  ConversationStatus = "deleted"
)

type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleSystem    MessageRole = "system"
)

type Conversation struct {
	ID            uuid.UUID           `json:"id"`
	TenantID      uuid.UUID           `json:"tenant_id"`
	UserID        uuid.UUID           `json:"user_id"`
	Title         string              `json:"title"`
	Status        ConversationStatus  `json:"status"`
	MessageCount  int                 `json:"message_count"`
	LastContext   ConversationContext `json:"last_context"`
	LastMessageAt *time.Time          `json:"last_message_at,omitempty"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
}

type Message struct {
	ID                uuid.UUID         `json:"id"`
	ConversationID    uuid.UUID         `json:"conversation_id"`
	TenantID          uuid.UUID         `json:"tenant_id"`
	Role              MessageRole       `json:"role"`
	Content           string            `json:"content"`
	Intent            *string           `json:"intent,omitempty"`
	IntentConfidence  *float64          `json:"intent_confidence,omitempty"`
	MatchMethod       *string           `json:"match_method,omitempty"`
	MatchedPattern    *string           `json:"matched_pattern,omitempty"`
	ExtractedEntities map[string]string `json:"extracted_entities,omitempty"`
	ToolName          *string           `json:"tool_name,omitempty"`
	ToolParams        map[string]string `json:"tool_params,omitempty"`
	ToolResult        json.RawMessage   `json:"tool_result,omitempty"`
	ToolLatencyMS     *int              `json:"tool_latency_ms,omitempty"`
	ToolError         *string           `json:"tool_error,omitempty"`
	ResponseType      *string           `json:"response_type,omitempty"`
	SuggestedActions  []SuggestedAction `json:"suggested_actions,omitempty"`
	EntityReferences  []EntityReference `json:"entity_references,omitempty"`
	PredictionLogID   *uuid.UUID        `json:"prediction_log_id,omitempty"`
	CreatedAt         time.Time         `json:"created_at"`
}

type ResponsePayload struct {
	Text     string            `json:"text"`
	Data     any               `json:"data,omitempty"`
	DataType string            `json:"data_type"`
	Actions  []SuggestedAction `json:"actions"`
	Entities []EntityReference `json:"entities,omitempty"`
}
