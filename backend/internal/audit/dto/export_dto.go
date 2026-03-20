package dto

import (
	"fmt"
	"time"
)

// ExportFormat defines supported export formats.
type ExportFormat string

const (
	ExportFormatCSV    ExportFormat = "csv"
	ExportFormatNDJSON ExportFormat = "ndjson"
)

// ExportConfig holds parameters for an audit log export.
type ExportConfig struct {
	Format       ExportFormat `json:"format"`
	TenantID     string       `json:"tenant_id"`
	DateFrom     time.Time    `json:"date_from"`
	DateTo       time.Time    `json:"date_to"`
	UserID       string       `json:"user_id,omitempty"`
	Service      string       `json:"service,omitempty"`
	Action       string       `json:"action,omitempty"`
	ResourceType string       `json:"resource_type,omitempty"`
	Severity     string       `json:"severity,omitempty"`
	Search       string       `json:"search,omitempty"`
	Columns      []string     `json:"columns,omitempty"` // optional: filter output to these columns only
	CallerRole   string       `json:"-"`
}

// AllExportColumns is the full ordered set of export columns.
var AllExportColumns = []string{
	"id", "tenant_id", "user_id", "user_email", "service", "action",
	"severity", "resource_type", "resource_id", "ip_address",
	"user_agent", "event_id", "correlation_id", "created_at",
}

// ValidExportColumns is a set for O(1) membership checks.
var ValidExportColumns = func() map[string]bool {
	m := make(map[string]bool, len(AllExportColumns))
	for _, c := range AllExportColumns {
		m[c] = true
	}
	return m
}()

// Validate checks that the export configuration is valid.
func (ec *ExportConfig) Validate() error {
	if ec.Format == "" {
		ec.Format = ExportFormatCSV
	}
	if ec.Format != ExportFormatCSV && ec.Format != ExportFormatNDJSON {
		return fmt.Errorf("invalid export format %q; allowed: csv, ndjson", ec.Format)
	}
	if ec.TenantID == "" {
		return fmt.Errorf("tenant_id is required for export")
	}
	if ec.DateFrom.IsZero() {
		return fmt.Errorf("date_from is required for export")
	}
	if ec.DateTo.IsZero() {
		ec.DateTo = time.Now().UTC()
	}
	if ec.DateTo.Before(ec.DateFrom) {
		return fmt.Errorf("date_to must be after date_from")
	}
	return nil
}

// ExportJobStatus represents the response for an async export job.
type ExportJobStatus struct {
	JobID       string `json:"job_id"`
	Status      string `json:"status"`
	PollURL     string `json:"poll_url,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
	RecordCount int64  `json:"record_count,omitempty"`
	Error       string `json:"error,omitempty"`
}
