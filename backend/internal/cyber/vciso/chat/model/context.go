package model

import (
	"time"

	"github.com/google/uuid"
)

type EntityReference struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Name  string `json:"name"`
	Index int    `json:"index"`
}

type SuggestedAction struct {
	Label  string            `json:"label"`
	Type   string            `json:"type"`
	Params map[string]string `json:"params"`
}

type Turn struct {
	Role     string            `json:"role"`
	Content  string            `json:"content"`
	Intent   string            `json:"intent"`
	ToolName string            `json:"tool_name"`
	Entities map[string]string `json:"entities"`
	At       time.Time         `json:"at"`
}

type ConversationContext struct {
	ConversationID uuid.UUID          `json:"conversation_id"`
	UserID         uuid.UUID          `json:"user_id"`
	TenantID       uuid.UUID          `json:"tenant_id"`
	Turns          []Turn             `json:"turns"`
	LastEntities   []EntityReference  `json:"last_entities"`
	ActiveFilters  map[string]string  `json:"active_filters"`
	StartedAt      time.Time          `json:"started_at"`
	LastActivityAt time.Time          `json:"last_activity_at"`
	IdleTimeoutMin int                `json:"idle_timeout_min"`
}

