package model

import (
	"encoding/json"
	"time"
)

// WorkflowTemplate is a reusable, pre-built workflow definition that tenants
// can instantiate to create their own WorkflowDefinition. Templates are
// system-level (not tenant-scoped) and are typically seeded at deployment time.
type WorkflowTemplate struct {
	ID              string          `json:"id" db:"id"`
	Name            string          `json:"name" db:"name"`
	Description     string          `json:"description" db:"description"`
	Category        string          `json:"category" db:"category"`
	DefinitionJSON  json.RawMessage `json:"definition_json" db:"definition_json"`
	Icon            string          `json:"icon" db:"icon"`
	PreviewImageURL *string         `json:"preview_image_url,omitempty" db:"preview_image_url"`
	Tags            []string        `json:"tags,omitempty" db:"tags"`
	UsageCount      int             `json:"usage_count" db:"usage_count"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
}

// TemplateDefinitionContent represents the parsed content of DefinitionJSON
// used for enriching the template response with steps and variables.
type TemplateDefinitionContent struct {
	Steps     []StepDefinition       `json:"steps"`
	Variables map[string]VariableDef `json:"variables"`
}
