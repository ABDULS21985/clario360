package model

import (
	"time"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

type LLMToolCall struct {
	ID           string         `json:"id"`
	FunctionName string         `json:"function_name"`
	Arguments    map[string]any `json:"arguments"`
}

type ToolCallResult struct {
	ToolName      string                      `json:"tool_name"`
	Success       bool                        `json:"success"`
	Data          any                         `json:"data"`
	Summary       string                      `json:"summary"`
	Error         string                      `json:"error,omitempty"`
	LatencyMs     int                         `json:"latency_ms"`
	Truncated     bool                        `json:"truncated"`
	TotalItems    int                         `json:"total_items"`
	ReturnedItems int                         `json:"returned_items"`
	DataType      string                      `json:"data_type,omitempty"`
	Actions       []chatmodel.SuggestedAction `json:"actions,omitempty"`
	Entities      []chatmodel.EntityReference `json:"entities,omitempty"`
	Text          string                      `json:"text,omitempty"`
}

type ToolCallAudit struct {
	Name          string         `json:"name"`
	Arguments     map[string]any `json:"arguments"`
	ResultSummary string         `json:"result_summary"`
	Success       bool           `json:"success"`
	LatencyMs     int            `json:"latency_ms"`
	CalledAt      time.Time      `json:"called_at"`
}

type ToolSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}
