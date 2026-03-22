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

// HubStatus reports the reachability of the backing JupyterHub instance.
// "available" means the hub responded with a non-5xx status.
// "unavailable" means the hub could not be reached or returned 5xx.
type HubStatus struct {
	Status string `json:"status"`          // "available" | "unavailable"
	Error  string `json:"error,omitempty"` // human-readable reason when unavailable
}

// HubHealthResponse is the response body for GET /notebooks/health.
// The endpoint always returns HTTP 200 so clients can distinguish "service up
// but hub degraded" from a total outage.
type HubHealthResponse struct {
	Status     string    `json:"status"` // "ok" | "degraded"
	JupyterHub HubStatus `json:"jupyterhub"`
}
