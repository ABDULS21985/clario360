package model

import "github.com/google/uuid"

type Factor struct {
	Name        string  `json:"name"`
	Value       string  `json:"value"`
	Impact      float64 `json:"impact"`
	Direction   string  `json:"direction"`
	Description string  `json:"description"`
}

type Explanation struct {
	Structured    map[string]any `json:"structured"`
	HumanReadable string         `json:"human_readable"`
	Factors       []Factor       `json:"factors"`
	Confidence    float64        `json:"confidence"`
	ExplainerType string         `json:"explainer_type"`
	ModelSlug     string         `json:"model_slug"`
	ModelVersion  int            `json:"model_version"`
}

type ExplanationSearchResult struct {
	PredictionID     uuid.UUID `json:"prediction_id"`
	ModelSlug        string    `json:"model_slug"`
	ExplanationText  string    `json:"explanation_text"`
	Confidence       *float64  `json:"confidence,omitempty"`
	UseCase          string    `json:"use_case"`
	EntityType       string    `json:"entity_type,omitempty"`
	MatchedHighlight string    `json:"matched_highlight,omitempty"`
}
