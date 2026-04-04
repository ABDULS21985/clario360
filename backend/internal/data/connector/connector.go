package connector

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/model"
)

type Connector interface {
	TestConnection(ctx context.Context) (*ConnectionTestResult, error)
	DiscoverSchema(ctx context.Context, opts DiscoveryOptions) (*model.DiscoveredSchema, error)
	FetchData(ctx context.Context, table string, params FetchParams) (*DataBatch, error)
	ReadQuery(ctx context.Context, query string, args []any) (*DataBatch, error)
	WriteData(ctx context.Context, table string, rows []map[string]any, params WriteParams) (*WriteResult, error)
	EstimateSize(ctx context.Context) (*SizeEstimate, error)
	Close() error
}

type ConnectorFactory func(config json.RawMessage) (Connector, error)

type DiscoveryOptions struct {
	MaxTables    int
	MaxColumns   int
	SampleValues bool
	MaxSamples   int
	IncludeViews bool
	SchemaFilter string
}

type FetchParams struct {
	Columns   []string
	Filters   map[string]any
	OrderBy   string
	BatchSize int
	Offset    int64
	Cursor    string
}

type DataBatch struct {
	Columns  []string           `json:"columns"`
	Rows     []map[string]any   `json:"rows"`
	RowCount int                `json:"row_count"`
	HasMore  bool               `json:"has_more"`
	Cursor   string             `json:"cursor,omitempty"`
}

type WriteParams struct {
	Strategy string
	MergeKeys []string
	Replace   bool
}

type WriteResult struct {
	RowsWritten  int64 `json:"rows_written"`
	RowsFailed   int64 `json:"rows_failed"`
	BytesWritten int64 `json:"bytes_written"`
}

type ConnectionTestResult struct {
	Success     bool          `json:"success"`
	LatencyMs   int64         `json:"latency_ms"`
	Version     string        `json:"version,omitempty"`
	Message     string        `json:"message"`
	Permissions []string      `json:"permissions,omitempty"`
	Warnings    []string      `json:"warnings,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
}

type SizeEstimate struct {
	TableCount int   `json:"table_count"`
	TotalRows  int64 `json:"total_rows"`
	TotalBytes int64 `json:"total_bytes"`
}

type ConnectorLimits struct {
	MaxPoolSize      int
	StatementTimeout time.Duration
	ConnectTimeout   time.Duration
	MaxSampleRows    int
	MaxTables        int
	APIRateLimit     int
}

type FactoryOptions struct {
	Limits ConnectorLimits
	Logger zerolog.Logger
}

var ErrCapabilityUnsupported = errors.New("connector capability unsupported")
