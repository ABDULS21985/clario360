package connector

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/data/model"
)

type SecurityAwareConnector interface {
	Connector
	QueryAccessLogs(ctx context.Context, since time.Time) ([]DataAccessEvent, error)
	ListDataLocations(ctx context.Context) ([]DataLocation, error)
}

type DataAccessEvent struct {
	Timestamp    time.Time `json:"timestamp"`
	User         string    `json:"user"`
	SourceIP     string    `json:"source_ip,omitempty"`
	Action       string    `json:"action"`
	Database     string    `json:"database"`
	Table        string    `json:"table,omitempty"`
	QueryHash    string    `json:"query_hash"`
	QueryPreview string    `json:"query_preview"`
	RowsRead     int64     `json:"rows_read"`
	RowsWritten  int64     `json:"rows_written"`
	BytesRead    int64     `json:"bytes_read"`
	BytesWritten int64     `json:"bytes_written"`
	DurationMs   int64     `json:"duration_ms"`
	Success      bool      `json:"success"`
	ErrorMsg     string    `json:"error_message,omitempty"`
	SourceType   string    `json:"source_type"`
	SourceID     uuid.UUID `json:"source_id"`
	TenantID     uuid.UUID `json:"tenant_id"`
}

type DataLocation struct {
	SourceID     uuid.UUID `json:"source_id"`
	SourceType   string    `json:"source_type"`
	Table        string    `json:"table"`
	Database     string    `json:"database"`
	Location     string    `json:"location"`
	Format       string    `json:"format"`
	SizeBytes    int64     `json:"size_bytes"`
	LastModified time.Time `json:"last_modified"`
	Partitioned  bool      `json:"partitioned"`
	Partitions   int       `json:"partitions,omitempty"`
}

type FileScanResult struct {
	Path           string               `json:"path"`
	Format         string               `json:"format"`
	SizeBytes      int64                `json:"size_bytes"`
	Columns        []model.DiscoveredColumn `json:"columns"`
	PIIFindings    []PIIFinding         `json:"pii_findings"`
	Classification string               `json:"classification"`
	SampledBytes   int64                `json:"sampled_bytes"`
}

type PIIFinding struct {
	Column      string  `json:"column"`
	PIIType     string  `json:"pii_type"`
	Confidence  float64 `json:"confidence"`
	SampleCount int     `json:"sample_count"`
}

type FileInfo struct {
	Path        string    `json:"path"`
	SizeBytes   int64     `json:"size_bytes"`
	ModTime     time.Time `json:"mod_time"`
	Replication int       `json:"replication,omitempty"`
	BlockSize   int64     `json:"block_size,omitempty"`
}
