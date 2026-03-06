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

	// Write header
	header := []string{
		"id", "tenant_id", "user_id", "user_email", "service", "action",
		"severity", "resource_type", "resource_id", "ip_address",
		"user_agent", "event_id", "correlation_id", "created_at",
	}
	if err := csvWriter.Write(header); err != nil {
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

		userID := ""
		if masked.UserID != nil {
			userID = *masked.UserID
		}

		record := []string{
			masked.ID, masked.TenantID, userID, masked.UserEmail,
			masked.Service, masked.Action, masked.Severity,
			masked.ResourceType, masked.ResourceID, masked.IPAddress,
			masked.UserAgent, masked.EventID, masked.CorrelationID,
			masked.CreatedAt.Format(time.RFC3339),
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

	_, err := s.repo.StreamForExport(ctx, filter, func(entry *model.AuditEntry) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		masked := s.masking.MaskEntry(entry, callerRoles)
		if err := encoder.Encode(masked); err != nil {
			return fmt.Errorf("encoding NDJSON: %w", err)
		}
		count++
		return nil
	})

	return count, err
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
