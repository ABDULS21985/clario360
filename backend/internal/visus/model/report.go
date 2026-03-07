package model

import (
	"time"

	"github.com/google/uuid"
)

type ReportType string
type ReportFileFormat string

const (
	ReportTypeExecutiveSummary ReportType = "executive_summary"
	ReportTypeSecurityPosture  ReportType = "security_posture"
	ReportTypeDataIntelligence ReportType = "data_intelligence"
	ReportTypeGovernance       ReportType = "governance"
	ReportTypeLegal            ReportType = "legal"
	ReportTypeCustom           ReportType = "custom"
)

const (
	ReportFileJSON ReportFileFormat = "json"
	ReportFilePDF  ReportFileFormat = "pdf"
	ReportFileHTML ReportFileFormat = "html"
)

type ReportDefinition struct {
	ID                uuid.UUID      `json:"id"`
	TenantID          uuid.UUID      `json:"tenant_id"`
	Name              string         `json:"name"`
	Description       string         `json:"description"`
	ReportType        ReportType     `json:"report_type"`
	Sections          []string       `json:"sections"`
	Period            string         `json:"period"`
	CustomPeriodStart *time.Time     `json:"custom_period_start,omitempty"`
	CustomPeriodEnd   *time.Time     `json:"custom_period_end,omitempty"`
	Schedule          *string        `json:"schedule,omitempty"`
	NextRunAt         *time.Time     `json:"next_run_at,omitempty"`
	Recipients        []uuid.UUID    `json:"recipients"`
	AutoSend          bool           `json:"auto_send"`
	LastGeneratedAt   *time.Time     `json:"last_generated_at,omitempty"`
	TotalGenerated    int            `json:"total_generated"`
	CreatedBy         uuid.UUID      `json:"created_by"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         *time.Time     `json:"deleted_at,omitempty"`
}

type ReportSnapshot struct {
	ID               uuid.UUID         `json:"id"`
	TenantID         uuid.UUID         `json:"tenant_id"`
	ReportID         uuid.UUID         `json:"report_id"`
	ReportData       map[string]any    `json:"report_data"`
	Narrative        *string           `json:"narrative,omitempty"`
	FileID           *uuid.UUID        `json:"file_id,omitempty"`
	FileFormat       ReportFileFormat  `json:"file_format"`
	PeriodStart      time.Time         `json:"period_start"`
	PeriodEnd        time.Time         `json:"period_end"`
	SectionsIncluded []string          `json:"sections_included"`
	GenerationTimeMS *int64            `json:"generation_time_ms,omitempty"`
	SuiteFetchErrors map[string]string `json:"suite_fetch_errors"`
	GeneratedBy      *uuid.UUID        `json:"generated_by,omitempty"`
	GeneratedAt      time.Time         `json:"generated_at"`
}
