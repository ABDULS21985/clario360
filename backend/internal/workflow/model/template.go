package model

import (
	"encoding/json"
	"time"
)

// WorkflowTemplate is a reusable, pre-built workflow definition that tenants
// can instantiate to create their own WorkflowDefinition. Templates are
// system-level (not tenant-scoped) and are typically seeded at deployment time.
type WorkflowTemplate struct {
	ID             string          `json:"id" db:"id"`
	Name           string          `json:"name" db:"name"`
	Description    string          `json:"description" db:"description"`
	Category       string          `json:"category" db:"category"`
	DefinitionJSON json.RawMessage `json:"definition_json" db:"definition_json"`
	Icon           string          `json:"icon" db:"icon"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
}
