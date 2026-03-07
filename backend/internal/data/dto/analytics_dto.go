package dto

import (
	"github.com/google/uuid"

	"github.com/clario360/platform/internal/data/model"
)

type ExecuteAnalyticsQueryRequest struct {
	ModelID uuid.UUID            `json:"model_id"`
	Query   model.AnalyticsQuery `json:"query"`
}

type ExplainAnalyticsQueryRequest struct {
	ModelID uuid.UUID            `json:"model_id"`
	Query   model.AnalyticsQuery `json:"query"`
}

type SaveQueryRequest struct {
	Name            string               `json:"name"`
	Description     string               `json:"description"`
	ModelID         uuid.UUID            `json:"model_id"`
	QueryDefinition model.AnalyticsQuery `json:"query_definition"`
	Visibility      string               `json:"visibility"`
	Tags            []string             `json:"tags,omitempty"`
}

type UpdateSavedQueryRequest struct {
	Description     *string               `json:"description,omitempty"`
	QueryDefinition *model.AnalyticsQuery `json:"query_definition,omitempty"`
	Visibility      *string               `json:"visibility,omitempty"`
	Tags            []string              `json:"tags,omitempty"`
}

type ListSavedQueriesParams struct {
	Page       int
	PerPage    int
	ModelID    string
	Visibility string
	Search     string
	Sort       string
	Order      string
}

type ListAnalyticsAuditParams struct {
	Page           int
	PerPage        int
	ModelID        string
	UserID         string
	Classification string
	PIIAccessed    *bool
	Sort           string
	Order          string
}
