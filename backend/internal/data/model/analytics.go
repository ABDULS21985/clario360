package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AnalyticsFilter struct {
	Column   string `json:"column"`
	Operator string `json:"operator"`
	Value    any    `json:"value,omitempty"`
}

type AnalyticsOrder struct {
	Column    string `json:"column"`
	Direction string `json:"direction"`
}

type AnalyticsAggregation struct {
	Function string  `json:"function"`
	Column   string  `json:"column"`
	Alias    string  `json:"alias"`
	Distinct bool    `json:"distinct,omitempty"`
}

type AnalyticsQuery struct {
	Columns      []string               `json:"columns,omitempty"`
	Filters      []AnalyticsFilter      `json:"filters,omitempty"`
	GroupBy      []string               `json:"group_by,omitempty"`
	Aggregations []AnalyticsAggregation `json:"aggregations,omitempty"`
	OrderBy      []AnalyticsOrder       `json:"order_by,omitempty"`
	Limit        int                    `json:"limit,omitempty"`
	Offset       int                    `json:"offset,omitempty"`
}

type QueryExplain struct {
	SQL        string `json:"sql"`
	CountSQL   string `json:"count_sql"`
	Parameters []any  `json:"parameters"`
}

type ColumnMeta struct {
	Name           string `json:"name"`
	DataType       string `json:"data_type"`
	Classification string `json:"classification"`
	IsPII          bool   `json:"is_pii"`
	Masked         bool   `json:"masked"`
}

type QueryMetadata struct {
	ModelName          string   `json:"model_name"`
	DataClassification string   `json:"data_classification"`
	PIIMaskingApplied  bool     `json:"pii_masking_applied"`
	ColumnsMasked      []string `json:"columns_masked,omitempty"`
	ExecutionTimeMs    int64    `json:"execution_time_ms"`
	CachedResult       bool     `json:"cached_result"`
}

type QueryResult struct {
	Columns    []ColumnMeta      `json:"columns"`
	Rows       []map[string]any  `json:"rows"`
	RowCount   int               `json:"row_count"`
	TotalCount int64             `json:"total_count"`
	Truncated  bool              `json:"truncated"`
	Metadata   QueryMetadata     `json:"metadata"`
}

type SavedQueryVisibility string

const (
	SavedQueryVisibilityPrivate      SavedQueryVisibility = "private"
	SavedQueryVisibilityTeam         SavedQueryVisibility = "team"
	SavedQueryVisibilityOrganization SavedQueryVisibility = "organization"
)

type SavedQuery struct {
	ID              uuid.UUID            `json:"id"`
	TenantID        uuid.UUID            `json:"tenant_id"`
	Name            string               `json:"name"`
	Description     string               `json:"description"`
	ModelID         uuid.UUID            `json:"model_id"`
	QueryDefinition AnalyticsQuery       `json:"query_definition"`
	LastRunAt       *time.Time           `json:"last_run_at,omitempty"`
	RunCount        int                  `json:"run_count"`
	Visibility      SavedQueryVisibility `json:"visibility"`
	Tags            []string             `json:"tags"`
	CreatedBy       uuid.UUID            `json:"created_by"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
	DeletedAt       *time.Time           `json:"deleted_at,omitempty"`
}

type AnalyticsAuditLog struct {
	ID                 uuid.UUID        `json:"id"`
	TenantID           uuid.UUID        `json:"tenant_id"`
	UserID             uuid.UUID        `json:"user_id"`
	ModelID            uuid.UUID        `json:"model_id"`
	SourceID           uuid.UUID        `json:"source_id"`
	QueryDefinition    AnalyticsQuery   `json:"query_definition"`
	ColumnsAccessed    []string         `json:"columns_accessed"`
	FiltersApplied     json.RawMessage  `json:"filters_applied"`
	DataClassification string           `json:"data_classification"`
	PIIColumnsAccessed []string         `json:"pii_columns_accessed"`
	PIIMaskingApplied  bool             `json:"pii_masking_applied"`
	RowsReturned       int              `json:"rows_returned"`
	Truncated          bool             `json:"truncated"`
	ExecutionTimeMs    *int64           `json:"execution_time_ms,omitempty"`
	ErrorOccurred      bool             `json:"error_occurred"`
	ErrorMessage       *string          `json:"error_message,omitempty"`
	SavedQueryID       *uuid.UUID       `json:"saved_query_id,omitempty"`
	IPAddress          *string          `json:"ip_address,omitempty"`
	UserAgent          *string          `json:"user_agent,omitempty"`
	ExecutedAt         time.Time        `json:"executed_at"`
}
