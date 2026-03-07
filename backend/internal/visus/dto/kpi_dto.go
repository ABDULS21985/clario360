package dto

import "github.com/google/uuid"

type CreateKPIRequest struct {
	Name              string         `json:"name"`
	Description       string         `json:"description"`
	Category          string         `json:"category"`
	Suite             string         `json:"suite"`
	Icon              *string        `json:"icon"`
	QueryEndpoint     string         `json:"query_endpoint"`
	QueryParams       map[string]any `json:"query_params"`
	ValuePath         string         `json:"value_path"`
	Unit              string         `json:"unit"`
	FormatPattern     *string        `json:"format_pattern"`
	TargetValue       *float64       `json:"target_value"`
	WarningThreshold  *float64       `json:"warning_threshold"`
	CriticalThreshold *float64       `json:"critical_threshold"`
	Direction         string         `json:"direction"`
	CalculationType   string         `json:"calculation_type"`
	CalculationWindow *string        `json:"calculation_window"`
	SnapshotFrequency string         `json:"snapshot_frequency"`
	Enabled           *bool          `json:"enabled"`
	Tags              []string       `json:"tags"`
}

type UpdateKPIRequest = CreateKPIRequest

type KPISummaryResponse struct {
	Items []uuid.UUID `json:"items"`
}
