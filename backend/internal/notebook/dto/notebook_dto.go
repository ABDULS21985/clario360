package dto

import "time"

type StartServerRequest struct {
	Profile string `json:"profile" validate:"required"`
}

type CopyTemplateRequest struct {
	TemplateID string `json:"template_id" validate:"required"`
}

type ActivityRequest struct {
	Kind        string                 `json:"kind" validate:"required,oneof=sdk_api data_query spark_job"`
	Endpoint    string                 `json:"endpoint,omitempty"`
	Status      string                 `json:"status,omitempty"`
	Source      string                 `json:"source,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	OccurredAt  *time.Time             `json:"occurred_at,omitempty"`
}

type MessageResponse struct {
	Message string `json:"message"`
}
