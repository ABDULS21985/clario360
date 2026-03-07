package dto

import (
	"encoding/json"

	"github.com/google/uuid"
)

type CreateQualityRuleRequest struct {
	ModelID     uuid.UUID       `json:"model_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	RuleType    string          `json:"rule_type"`
	Severity    string          `json:"severity"`
	ColumnName  *string         `json:"column_name,omitempty"`
	Config      json.RawMessage `json:"config"`
	Schedule    *string         `json:"schedule,omitempty"`
	Enabled     *bool           `json:"enabled,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
}

type UpdateQualityRuleRequest struct {
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Severity    *string         `json:"severity,omitempty"`
	ColumnName  *string         `json:"column_name,omitempty"`
	Config      json.RawMessage `json:"config,omitempty"`
	Schedule    *string         `json:"schedule,omitempty"`
	Enabled     *bool           `json:"enabled,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
}

type ListQualityRulesParams struct {
	Page      int
	PerPage   int
	ModelID   string
	Severity  string
	Status    string
	Enabled   *bool
	Search    string
	Sort      string
	Order     string
}

type ListQualityResultsParams struct {
	Page     int
	PerPage  int
	RuleID   string
	ModelID  string
	Status   string
	Sort     string
	Order    string
}

