package dto

import "github.com/google/uuid"

type CreateDashboardRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	GridColumns int            `json:"grid_columns"`
	Visibility  string         `json:"visibility"`
	SharedWith  []uuid.UUID    `json:"shared_with"`
	IsDefault   bool           `json:"is_default"`
	Tags        []string       `json:"tags"`
	Metadata    map[string]any `json:"metadata"`
}

type UpdateDashboardRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	GridColumns int            `json:"grid_columns"`
	Visibility  string         `json:"visibility"`
	SharedWith  []uuid.UUID    `json:"shared_with"`
	IsDefault   bool           `json:"is_default"`
	Tags        []string       `json:"tags"`
	Metadata    map[string]any `json:"metadata"`
}

type ShareDashboardRequest struct {
	Visibility string      `json:"visibility"`
	SharedWith []uuid.UUID `json:"shared_with"`
}
