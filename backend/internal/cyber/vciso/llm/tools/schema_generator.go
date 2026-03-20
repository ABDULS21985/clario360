package tools

import (
	llmmodel "github.com/clario360/platform/internal/cyber/vciso/llm/model"
)

func GenerateToolSchemas(items []Tool, _ string) []llmmodel.ToolSchema {
	schemas := make([]llmmodel.ToolSchema, 0, len(items))
	for _, item := range items {
		schemas = append(schemas, llmmodel.ToolSchema{
			Name:        item.Name(),
			Description: item.Description(),
			Parameters:  item.Schema(),
		})
	}
	return schemas
}
