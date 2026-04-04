package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/audit/dto"
	"github.com/clario360/platform/internal/audit/metrics"
	"github.com/clario360/platform/internal/audit/model"
	"github.com/clario360/platform/internal/audit/repository"
)

// ExportService handles streaming audit log exports.
type ExportService struct {
	repo               *repository.AuditRepository
	masking            *MaskingService
	logger             zerolog.Logger
	asyncThreshold     int
}

// NewExportService creates a new ExportService.
func NewExportService(
	repo *repository.AuditRepository,
	masking *MaskingService,
	logger zerolog.Logger,
	asyncThreshold int,
) *ExportService {
	return &ExportService{
		repo:           repo,
		masking:        masking,
		logger:         logger,
		asyncThreshold: asyncThreshold,
	}
}

// ExportCSV streams audit entries as CSV to the given writer.
func (s *ExportService) ExportCSV(ctx context.Context, w io.Writer, cfg *dto.ExportConfig, callerRoles []string) (int64, error) {
	start := time.Now()
	defer func() {
		metrics.ExportDuration.WithLabelValues("csv", "sync").Observe(time.Since(start).Seconds())
	}()

	csvWriter := csv.NewWriter(w)

	// Determine which columns to include.
	columns := dto.AllExportColumns
	if len(cfg.Columns) > 0 {
		columns = cfg.Columns
	}
	colSet := make(map[string]bool, len(columns))
	for _, c := range columns {
		colSet[c] = true
	}

	// Write header
	if err := csvWriter.Write(columns); err != nil {
		return 0, fmt.Errorf("writing CSV header: %w", err)
	}

	filter := s.buildFilter(cfg)
	var count int64
	var flushCounter int

	_, err := s.repo.StreamForExport(ctx, filter, func(entry *model.AuditEntry) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		masked := s.masking.MaskEntry(entry, callerRoles)
		allFields := entryToFieldMap(&masked)

		record := make([]string, 0, len(columns))
		for _, col := range columns {
			record = append(record, allFields[col])
		}

		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("writing CSV record: %w", err)
		}

		count++
		flushCounter++
		if flushCounter >= 1000 {
			csvWriter.Flush()
			if err := csvWriter.Error(); err != nil {
				return fmt.Errorf("flushing CSV: %w", err)
			}
			flushCounter = 0
		}
		return nil
	})

	csvWriter.Flush()
	if flushErr := csvWriter.Error(); flushErr != nil && err == nil {
		err = flushErr
	}

	return count, err
}

// ExportNDJSON streams audit entries as newline-delimited JSON.
func (s *ExportService) ExportNDJSON(ctx context.Context, w io.Writer, cfg *dto.ExportConfig, callerRoles []string) (int64, error) {
	start := time.Now()
	defer func() {
		metrics.ExportDuration.WithLabelValues("ndjson", "sync").Observe(time.Since(start).Seconds())
	}()

	encoder := json.NewEncoder(w)
	filter := s.buildFilter(cfg)
	var count int64

	// Build column set for filtering NDJSON output.
	var colSet map[string]bool
	if len(cfg.Columns) > 0 {
		colSet = make(map[string]bool, len(cfg.Columns))
		for _, c := range cfg.Columns {
			colSet[c] = true
		}
	}

	_, err := s.repo.StreamForExport(ctx, filter, func(entry *model.AuditEntry) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		masked := s.masking.MaskEntry(entry, callerRoles)

		if colSet != nil {
			// Emit only selected columns as a JSON object.
			fields := entryToFieldMap(&masked)
			filtered := make(map[string]string, len(colSet))
			for col := range colSet {
				filtered[col] = fields[col]
			}
			if err := encoder.Encode(filtered); err != nil {
				return fmt.Errorf("encoding NDJSON: %w", err)
			}
		} else {
			if err := encoder.Encode(masked); err != nil {
				return fmt.Errorf("encoding NDJSON: %w", err)
			}
		}
		count++
		return nil
	})

	return count, err
}

// entryToFieldMap converts an AuditEntry to a column-name→string-value map.
func entryToFieldMap(e *model.AuditEntry) map[string]string {
	userID := ""
	if e.UserID != nil {
		userID = *e.UserID
	}
	return map[string]string{
		"id":             e.ID,
		"tenant_id":      e.TenantID,
		"user_id":        userID,
		"user_email":     e.UserEmail,
		"service":        e.Service,
		"action":         e.Action,
		"severity":       e.Severity,
		"resource_type":  e.ResourceType,
		"resource_id":    e.ResourceID,
		"ip_address":     e.IPAddress,
		"user_agent":     e.UserAgent,
		"event_id":       e.EventID,
		"correlation_id": e.CorrelationID,
		"created_at":     e.CreatedAt.Format(time.RFC3339),
	}
}

// ShouldExportAsync determines if the export should be async based on record count.
func (s *ExportService) ShouldExportAsync(ctx context.Context, cfg *dto.ExportConfig) (bool, int64, error) {
	filter := s.buildFilter(cfg)
	count, err := s.repo.CountForExport(ctx, filter)
	if err != nil {
		return false, 0, err
	}
	return count > int64(s.asyncThreshold), count, nil
}

func (s *ExportService) buildFilter(cfg *dto.ExportConfig) repository.QueryFilter {
	return repository.QueryFilter{
		TenantID:     cfg.TenantID,
		UserID:       cfg.UserID,
		Service:      cfg.Service,
		Action:       cfg.Action,
		ResourceType: cfg.ResourceType,
		Severity:     cfg.Severity,
		DateFrom:     cfg.DateFrom,
		DateTo:       cfg.DateTo,
		Search:       cfg.Search,
		Sort:         "created_at",
		Order:        "asc",
	}
}
