package dto

import aigovmodel "github.com/clario360/platform/internal/aigovernance/model"

type CustomValidationSample struct {
	InputHash     string                     `json:"input_hash"`
	ExpectedLabel aigovmodel.ValidationLabel `json:"expected_label"`
}

type ValidateRequest struct {
	DatasetType          aigovmodel.ValidationDatasetType `json:"dataset_type"`
	TimeRange            string                           `json:"time_range,omitempty"`
	CustomData           []CustomValidationSample         `json:"custom_data,omitempty"`
	ConfidenceThresholds []float64                        `json:"confidence_thresholds,omitempty"`
}

type ValidationPreviewResponse struct {
	DatasetType   aigovmodel.ValidationDatasetType `json:"dataset_type"`
	DatasetSize   int                              `json:"dataset_size"`
	PositiveCount int                              `json:"positive_count"`
	NegativeCount int                              `json:"negative_count"`
	Warnings      []string                         `json:"warnings"`
}
