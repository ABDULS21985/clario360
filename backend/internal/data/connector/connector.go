package connector

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/model"
)

type Connector interface {
	TestConnection(ctx context.Context) (*ConnectionTestResult, error)
	DiscoverSchema(ctx context.Context, opts DiscoveryOptions) (*model.DiscoveredSchema, error)
	FetchData(ctx context.Context, table string, params FetchParams) (*DataBatch, error)
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
