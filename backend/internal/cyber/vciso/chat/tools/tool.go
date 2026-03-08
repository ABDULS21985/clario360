package tools

import (
	"context"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

type Tool interface {
	Name() string
	Description() string
	RequiredPermissions() []string
	Execute(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, params map[string]string) (*ToolResult, error)
}

type ToolResult struct {
	Text     string                      `json:"text"`
	Data     any                         `json:"data,omitempty"`
	DataType string                      `json:"data_type"`
	Actions  []chatmodel.SuggestedAction `json:"actions"`
	Entities []chatmodel.EntityReference `json:"entities"`
}

