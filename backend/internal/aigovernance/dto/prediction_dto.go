package dto

import (
	"encoding/json"

	"github.com/google/uuid"
)

type PredictionFeedbackRequest struct {
	Correct         bool            `json:"correct"`
	CorrectedOutput json.RawMessage `json:"corrected_output"`
	Notes           string          `json:"notes"`
}

type PredictionQuery struct {
	ModelID   *uuid.UUID
	Suite     string
	UseCase   string
	EntityType string
	IsShadow  *bool
	Search    string
	Page      int
	PerPage   int
}
