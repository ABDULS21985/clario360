package watchers

import (
	"context"

	"github.com/google/uuid"
)

// AlertCreator is the interface for creating alerts from watcher detections.
type AlertCreator interface {
	CreateDSPMAlert(ctx context.Context, tenantID uuid.UUID, title, description, severity, source string, assetID *uuid.UUID, metadata map[string]interface{}) error
}

// Watcher is the interface for continuous DSPM scan watchers.
type Watcher interface {
	// Name returns the watcher identifier.
	Name() string

	// Start begins the watcher. It should block until ctx is cancelled.
	Start(ctx context.Context) error

	// Stop gracefully stops the watcher.
	Stop() error
}

// EventData is the common structure for pipeline events consumed by watchers.
type EventData struct {
	PipelineID   uuid.UUID `json:"pipeline_id"`
	PipelineName string    `json:"pipeline_name"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Status       string    `json:"status"` // "running", "completed", "failed"
	SourceID     *uuid.UUID `json:"source_id,omitempty"`
	TargetID     *uuid.UUID `json:"target_id,omitempty"`
	SourceTable  string    `json:"source_table,omitempty"`
	TargetTable  string    `json:"target_table,omitempty"`
	RecordsCount int64     `json:"records_count,omitempty"`
}
