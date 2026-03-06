package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/filemanager/metrics"
	"github.com/clario360/platform/internal/filemanager/model"
	"github.com/clario360/platform/internal/filemanager/repository"
	"github.com/clario360/platform/pkg/storage"
)

const maxScanRetries = 3

// ScanService handles virus scan orchestration.
type ScanService struct {
	repo      *repository.FileRepository
	store     storage.Storage
	scanner   *storage.VirusScanner
	encryptor *storage.Encryptor
	producer  *events.Producer
	metrics   *metrics.FileMetrics
	logger    zerolog.Logger

	quarantineBucket string
}

// NewScanService creates a new scan service.
func NewScanService(
	repo *repository.FileRepository,
	store storage.Storage,
	scanner *storage.VirusScanner,
	encryptor *storage.Encryptor,
	producer *events.Producer,
	fm *metrics.FileMetrics,
	logger zerolog.Logger,
	quarantineBucket string,
) *ScanService {
	return &ScanService{
		repo:             repo,
		store:            store,
		scanner:          scanner,
		encryptor:        encryptor,
		producer:         producer,
		metrics:          fm,
		logger:           logger,
		quarantineBucket: quarantineBucket,
	}
}

// ScanFile performs virus scanning on a file.
func (s *ScanService) ScanFile(ctx context.Context, fileID string) error {
	// 1. Load file record
	record, err := s.repo.GetByIDNoTenant(ctx, fileID)
	if err != nil {
		return fmt.Errorf("loading file for scan: %w", err)
	}
	if record == nil {
		s.logger.Warn().Str("file_id", fileID).Msg("file not found for scan, skipping")
		return nil
	}

	// Skip if already scanned
	if record.VirusScanStatus != model.ScanStatusPending {
		return nil
	}

	// 2. Atomic CAS: pending → scanning
	updated, err := s.repo.UpdateScanStatus(ctx, fileID, model.ScanStatusPending, model.ScanStatusScanning, nil, nil)
	if err != nil {
		return fmt.Errorf("updating scan status: %w", err)
	}
	if !updated {
		// Another consumer got it first
		return nil
	}

	// 3. Download from storage
	reader, _, err := s.store.Download(ctx, record.Bucket, record.StorageKey)
	if err != nil {
		s.handleScanError(ctx, record, fmt.Errorf("downloading for scan: %w", err))
		return nil
	}
	defer reader.Close()

	// If encrypted, decrypt before scanning
	var scanReader io.Reader = reader
	if record.Encrypted && s.encryptor != nil && len(record.EncryptionMetadata) > 0 {
		var encMeta storage.EncryptionMetadata
		if err := json.Unmarshal(record.EncryptionMetadata, &encMeta); err != nil {
			s.handleScanError(ctx, record, fmt.Errorf("parsing encryption metadata: %w", err))
			return nil
		}

		// Read all ciphertext for decryption
		ciphertext, err := io.ReadAll(reader)
		if err != nil {
			s.handleScanError(ctx, record, fmt.Errorf("reading ciphertext: %w", err))
			return nil
		}

		decrypted, err := s.encryptor.Decrypt(bytes.NewReader(ciphertext), &encMeta)
		if err != nil {
			s.handleScanError(ctx, record, fmt.Errorf("decrypting for scan: %w", err))
			return nil
		}
		scanReader = decrypted
	}

	// 4. Scan via ClamAV
	if s.scanner == nil {
		s.logger.Warn().Str("file_id", fileID).Msg("virus scanner not available, keeping status as pending")
		// Reset status back to pending for retry
		now := time.Now()
		s.repo.UpdateScanStatus(ctx, fileID, model.ScanStatusScanning, model.ScanStatusPending, nil, &now)
		return nil
	}

	result, err := s.scanner.Scan(scanReader, record.SizeBytes)
	if err != nil {
		s.metrics.ClamAVErrorsTotal.Inc()
		s.handleScanError(ctx, record, fmt.Errorf("clamd scan: %w", err))
		return nil
	}

	s.metrics.VirusScanDuration.WithLabelValues().Observe(result.Duration.Seconds())

	now := time.Now()

	switch result.Status {
	case storage.ScanClean:
		s.handleClean(ctx, record, &now)

	case storage.ScanInfected:
		s.handleInfected(ctx, record, result, &now)

	case storage.ScanSkipped:
		statusStr := string(storage.ScanSkipped)
		s.repo.UpdateScanStatus(ctx, fileID, model.ScanStatusScanning, model.ScanStatusSkipped, &statusStr, &now)
		s.logger.Warn().Str("file_id", fileID).Str("reason", result.Reason).Msg("virus scan skipped")
		s.metrics.VirusScansTotal.WithLabelValues("skipped").Inc()

	case storage.ScanError:
		s.handleScanError(ctx, record, fmt.Errorf("scan error: %s", result.Reason))
	}

	return nil
}

func (s *ScanService) handleClean(ctx context.Context, record *model.FileRecord, scannedAt *time.Time) {
	cleanStr := string(storage.ScanClean)
	s.repo.UpdateScanStatus(ctx, record.ID, model.ScanStatusScanning, model.ScanStatusClean, &cleanStr, scannedAt)
	s.metrics.VirusScansTotal.WithLabelValues("clean").Inc()

	s.publishScanEvent(ctx, "com.clario360.file.scan.completed", record.TenantID, map[string]interface{}{
		"file_id": record.ID,
		"status":  "clean",
	})
}

func (s *ScanService) handleInfected(ctx context.Context, record *model.FileRecord, result *storage.ScanResult, scannedAt *time.Time) {
	infectedStr := result.VirusName
	s.repo.UpdateScanStatus(ctx, record.ID, model.ScanStatusScanning, model.ScanStatusInfected, &infectedStr, scannedAt)
	s.metrics.VirusScansTotal.WithLabelValues("infected").Inc()
	s.metrics.QuarantinedTotal.Inc()

	// Copy to quarantine bucket
	quarantineKey := fmt.Sprintf("quarantine/%s/%s", record.TenantID, record.ID)
	if err := s.store.EnsureBucket(ctx, s.quarantineBucket); err != nil {
		s.logger.Error().Err(err).Msg("failed to ensure quarantine bucket")
	}

	if err := s.store.CopyObject(ctx, record.Bucket, record.StorageKey, s.quarantineBucket, quarantineKey); err != nil {
		s.logger.Error().Err(err).Str("file_id", record.ID).Msg("failed to copy to quarantine")
	} else {
		// Delete from source bucket
		if err := s.store.Delete(ctx, record.Bucket, record.StorageKey); err != nil {
			s.logger.Error().Err(err).Str("file_id", record.ID).Msg("failed to delete infected file from source")
		}
	}

	// Log quarantine
	qLog := &model.QuarantineLog{
		FileID:           record.ID,
		OriginalBucket:   record.Bucket,
		OriginalKey:      record.StorageKey,
		QuarantineBucket: s.quarantineBucket,
		QuarantineKey:    quarantineKey,
		VirusName:        result.VirusName,
		ScannedAt:        *scannedAt,
	}
	if err := s.repo.CreateQuarantineLog(ctx, qLog); err != nil {
		s.logger.Error().Err(err).Str("file_id", record.ID).Msg("failed to create quarantine log")
	}

	// Publish events
	s.publishScanEvent(ctx, "com.clario360.file.scan.infected", record.TenantID, map[string]interface{}{
		"file_id":    record.ID,
		"virus_name": result.VirusName,
	})
	s.publishScanEvent(ctx, "com.clario360.file.quarantined", record.TenantID, map[string]interface{}{
		"file_id":           record.ID,
		"virus_name":        result.VirusName,
		"original_bucket":   record.Bucket,
		"quarantine_bucket": s.quarantineBucket,
	})

	s.logger.Error().
		Str("file_id", record.ID).
		Str("virus", result.VirusName).
		Str("tenant_id", record.TenantID).
		Msg("INFECTED FILE QUARANTINED")
}

func (s *ScanService) handleScanError(ctx context.Context, record *model.FileRecord, scanErr error) {
	s.logger.Error().Err(scanErr).Str("file_id", record.ID).Msg("virus scan error")
	s.metrics.VirusScansTotal.WithLabelValues("error").Inc()

	errStr := scanErr.Error()
	now := time.Now()
	s.repo.UpdateScanStatus(ctx, record.ID, model.ScanStatusScanning, model.ScanStatusError, &errStr, &now)

	s.publishScanEvent(ctx, "com.clario360.file.scan.error", record.TenantID, map[string]interface{}{
		"file_id": record.ID,
		"error":   scanErr.Error(),
	})
}

func (s *ScanService) publishScanEvent(ctx context.Context, eventType, tenantID string, data map[string]interface{}) {
	if s.producer == nil {
		return
	}
	event, err := events.NewEvent(eventType, "file-service", tenantID, data)
	if err != nil {
		s.logger.Error().Err(err).Str("type", eventType).Msg("failed to create scan event")
		return
	}
	if err := s.producer.Publish(ctx, fileEventsTopic, event); err != nil {
		s.logger.Error().Err(err).Str("type", eventType).Msg("failed to publish scan event")
	}
}
