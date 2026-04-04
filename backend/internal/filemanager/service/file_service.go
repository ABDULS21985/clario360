package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/filemanager/dto"
	"github.com/clario360/platform/internal/filemanager/metrics"
	"github.com/clario360/platform/internal/filemanager/model"
	"github.com/clario360/platform/internal/filemanager/repository"
	"github.com/clario360/platform/pkg/storage"
)

const fileEventsTopic = "platform.file.events"

// FileService handles core file lifecycle operations.
type FileService struct {
	repo      *repository.FileRepository
	store     storage.Storage
	encryptor *storage.Encryptor
	producer  *events.Producer
	metrics   *metrics.FileMetrics
	logger    zerolog.Logger

	bucketPrefix     string
	quarantineBucket string
	presignedExpiry  time.Duration

	// In-memory bucket existence cache
	bucketCache   map[string]bool
	bucketCacheMu sync.RWMutex
}

// NewFileService creates a new file service.
func NewFileService(
	repo *repository.FileRepository,
	store storage.Storage,
	encryptor *storage.Encryptor,
	producer *events.Producer,
	fm *metrics.FileMetrics,
	logger zerolog.Logger,
	bucketPrefix, quarantineBucket string,
	presignedExpiry time.Duration,
) *FileService {
	return &FileService{
		repo:             repo,
		store:            store,
		encryptor:        encryptor,
		producer:         producer,
		metrics:          fm,
		logger:           logger,
		bucketPrefix:     bucketPrefix,
		quarantineBucket: quarantineBucket,
		presignedExpiry:  presignedExpiry,
		bucketCache:      make(map[string]bool),
	}
}

// Upload handles the full upload flow.
func (s *FileService) Upload(ctx context.Context, req *dto.UploadRequest, file io.Reader, fileSize int64, filename, contentType, tenantID, userID, ipAddress, userAgent string) (*model.FileRecord, error) {
	start := time.Now()

	// 1. Validate content type via magic bytes
	validation, err := storage.ValidateContent(file, contentType, req.Suite)
	if err != nil {
		return nil, fmt.Errorf("content validation: %w", err)
	}
	if validation.Blocked {
		s.metrics.BlockedUploadTotal.WithLabelValues("blocked_content_type").Inc()
		return nil, &ServiceError{Code: http.StatusUnsupportedMediaType, ErrCode: "UNSUPPORTED_MEDIA_TYPE",
			Message: fmt.Sprintf("content type %s is blocked", validation.DetectedType)}
	}
	if !validation.Allowed {
		s.metrics.BlockedUploadTotal.WithLabelValues("suite_content_type").Inc()
		return nil, &ServiceError{Code: http.StatusUnsupportedMediaType, ErrCode: "UNSUPPORTED_MEDIA_TYPE",
			Message: fmt.Sprintf("content type %s not allowed for suite %s", validation.DetectedType, req.Suite)}
	}
	if validation.Mismatch {
		s.metrics.ContentTypeMismatchTotal.WithLabelValues(contentType, validation.DetectedType).Inc()
		s.logger.Warn().
			Str("declared", contentType).
			Str("detected", validation.DetectedType).
			Msg("content type mismatch")
	}

	// Use replayed reader (with magic bytes prepended)
	reader := validation.Reader

	// 2. Compute SHA-256 checksum while reading
	hasher := sha256.New()
	teeReader := io.TeeReader(reader, hasher)

	// Read all content for checksum (and possible encryption)
	content, err := io.ReadAll(teeReader)
	if err != nil {
		return nil, fmt.Errorf("reading file content: %w", err)
	}
	checksum := hex.EncodeToString(hasher.Sum(nil))
	actualSize := int64(len(content))

	// 3. Duplicate check
	if req.DedupCheck && req.EntityType != "" && req.EntityID != "" {
		existing, _ := s.repo.FindByChecksum(ctx, tenantID, checksum, req.EntityType, req.EntityID)
		if existing != nil {
			s.metrics.DuplicateDetectedTotal.WithLabelValues(req.Suite).Inc()
			return existing, nil
		}
	}

	// 4. Encrypt if requested
	var uploadReader io.Reader
	var uploadSize int64
	var encMeta *storage.EncryptionMetadata
	encrypted := req.Encrypt

	if encrypted && s.encryptor != nil {
		encStart := time.Now()
		ciphertext, cSize, meta, err := s.encryptor.Encrypt(bytes.NewReader(content))
		if err != nil {
			return nil, fmt.Errorf("encryption: %w", err)
		}
		s.metrics.EncryptionDuration.WithLabelValues("encrypt").Observe(time.Since(encStart).Seconds())
		uploadReader = bytes.NewReader(ciphertext)
		uploadSize = cSize
		encMeta = meta
	} else {
		encrypted = false
		uploadReader = bytes.NewReader(content)
		uploadSize = actualSize
	}

	// 5. Generate storage key
	sanitizedName := storage.SanitizeFilename(filename)
	storageKey := storage.GenerateStorageKey(tenantID, req.Suite, filename)
	bucket := s.bucketName(req.Suite)

	// 6. Ensure bucket exists (cached)
	if err := s.ensureBucket(ctx, bucket); err != nil {
		s.metrics.MinIOErrorsTotal.WithLabelValues("ensure_bucket").Inc()
		return nil, fmt.Errorf("ensuring bucket: %w", err)
	}

	// 7. Upload to storage
	result, err := s.store.Upload(ctx, storage.UploadParams{
		Bucket:      bucket,
		Key:         storageKey,
		Body:        uploadReader,
		Size:        uploadSize,
		ContentType: contentType,
		Metadata: map[string]string{
			"tenant-id":   tenantID,
			"uploaded-by": userID,
		},
	})
	if err != nil {
		s.metrics.MinIOErrorsTotal.WithLabelValues("upload").Inc()
		return nil, &ServiceError{Code: http.StatusBadGateway, ErrCode: "STORAGE_ERROR", Message: "failed to store file"}
	}

	// 8. Build and persist FileRecord
	fileID := uuid.New().String()
	now := time.Now().UTC()

	var encMetaJSON json.RawMessage
	if encMeta != nil {
		b, _ := json.Marshal(encMeta)
		encMetaJSON = b
	}

	// Version number
	versionNumber := 1
	if req.EntityType != "" && req.EntityID != "" {
		v, _ := s.repo.GetLatestVersionNumber(ctx, tenantID, req.EntityType, req.EntityID, sanitizedName)
		versionNumber = v + 1
	}

	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err == nil {
			expiresAt = &t
		}
	}

	var entityType, entityID *string
	if req.EntityType != "" {
		entityType = &req.EntityType
	}
	if req.EntityID != "" {
		entityID = &req.EntityID
	}

	var versionID *string
	if result.VersionID != "" {
		versionID = &result.VersionID
	}

	record := &model.FileRecord{
		ID:                  fileID,
		TenantID:            tenantID,
		Bucket:              bucket,
		StorageKey:          storageKey,
		OriginalName:        filename,
		SanitizedName:       sanitizedName,
		ContentType:         contentType,
		DetectedContentType: validation.DetectedType,
		SizeBytes:           actualSize,
		ChecksumSHA256:      checksum,
		Encrypted:           encrypted,
		EncryptionMetadata:  encMetaJSON,
		VirusScanStatus:     model.ScanStatusPending,
		UploadedBy:          userID,
		Suite:               req.Suite,
		EntityType:          entityType,
		EntityID:            entityID,
		Tags:                req.Tags,
		VersionID:           versionID,
		VersionNumber:       versionNumber,
		IsPublic:            false,
		LifecyclePolicy:     req.LifecyclePolicy,
		ExpiresAt:           expiresAt,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if record.Tags == nil {
		record.Tags = []string{}
	}

	if err := s.repo.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("persisting file record: %w", err)
	}

	// 9. Log access
	s.logAccess(ctx, fileID, tenantID, userID, "upload", ipAddress, userAgent)

	// 10. Queue virus scan event
	s.publishFileEvent(ctx, "com.clario360.file.uploaded", tenantID, userID, map[string]interface{}{
		"file_id":      fileID,
		"tenant_id":    tenantID,
		"suite":        req.Suite,
		"size_bytes":   actualSize,
		"content_type": contentType,
		"encrypted":    encrypted,
	})

	// 11. Record metrics
	encLabel := strconv.FormatBool(encrypted)
	s.metrics.UploadsTotal.WithLabelValues(req.Suite, encLabel, req.LifecyclePolicy).Inc()
	s.metrics.UploadSizeBytes.WithLabelValues(req.Suite).Observe(float64(actualSize))
	s.metrics.UploadDuration.WithLabelValues(req.Suite, encLabel).Observe(time.Since(start).Seconds())

	return record, nil
}

// Download handles the full download flow.
func (s *FileService) Download(ctx context.Context, tenantID, fileID, userID, ipAddress, userAgent string) (io.ReadCloser, *model.FileRecord, error) {
	start := time.Now()

	// 1. Load file record
	record, err := s.repo.GetByID(ctx, tenantID, fileID)
	if err != nil {
		return nil, nil, fmt.Errorf("loading file: %w", err)
	}
	if record == nil {
		return nil, nil, &ServiceError{Code: http.StatusNotFound, ErrCode: "NOT_FOUND", Message: "file not found"}
	}

	// 2. Check quarantine status
	if record.VirusScanStatus == model.ScanStatusInfected {
		return nil, nil, &ServiceError{Code: http.StatusForbidden, ErrCode: "FORBIDDEN", Message: "file is quarantined"}
	}

	// 3. Download from storage
	reader, _, err := s.store.Download(ctx, record.Bucket, record.StorageKey)
	if err != nil {
		s.metrics.MinIOErrorsTotal.WithLabelValues("download").Inc()
		return nil, nil, &ServiceError{Code: http.StatusBadGateway, ErrCode: "STORAGE_ERROR", Message: "failed to retrieve file"}
	}

	// 4. Decrypt if encrypted
	if record.Encrypted && s.encryptor != nil && len(record.EncryptionMetadata) > 0 {
		var encMeta storage.EncryptionMetadata
		if err := json.Unmarshal(record.EncryptionMetadata, &encMeta); err != nil {
			reader.Close()
			return nil, nil, fmt.Errorf("parsing encryption metadata: %w", err)
		}

		decStart := time.Now()
		decrypted, err := s.encryptor.Decrypt(reader, &encMeta)
		reader.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("decrypting file: %w", err)
		}
		s.metrics.EncryptionDuration.WithLabelValues("decrypt").Observe(time.Since(decStart).Seconds())

		// Wrap decrypted reader as ReadCloser
		reader = io.NopCloser(decrypted)
	}

	// 5. Log access
	s.logAccess(ctx, fileID, tenantID, userID, "download", ipAddress, userAgent)

	// 6. Publish download event
	s.publishFileEvent(ctx, "com.clario360.file.downloaded", tenantID, userID, map[string]interface{}{
		"file_id":   fileID,
		"tenant_id": tenantID,
		"user_id":   userID,
	})

	encLabel := strconv.FormatBool(record.Encrypted)
	s.metrics.DownloadsTotal.WithLabelValues(record.Suite).Inc()
	s.metrics.DownloadDuration.WithLabelValues(record.Suite, encLabel).Observe(time.Since(start).Seconds())

	return reader, record, nil
}

// GetFile retrieves file metadata.
func (s *FileService) GetFile(ctx context.Context, tenantID, fileID, userID, ipAddress, userAgent string) (*model.FileRecord, error) {
	record, err := s.repo.GetByID(ctx, tenantID, fileID)
	if err != nil {
		return nil, fmt.Errorf("loading file: %w", err)
	}
	if record == nil {
		return nil, &ServiceError{Code: http.StatusNotFound, ErrCode: "NOT_FOUND", Message: "file not found"}
	}

	s.logAccess(ctx, fileID, tenantID, userID, "view_metadata", ipAddress, userAgent)
	return record, nil
}

// ListFiles lists files with filters.
func (s *FileService) ListFiles(ctx context.Context, tenantID string, params dto.ListFilesParams) ([]*model.FileRecord, int, error) {
	return s.repo.List(ctx, tenantID, params.Suite, params.EntityType, params.EntityID, params.UploadedBy, params.Tag, params.Page, params.PerPage)
}

// DeleteFile soft-deletes a file.
func (s *FileService) DeleteFile(ctx context.Context, tenantID, fileID, userID, ipAddress, userAgent string) error {
	record, err := s.repo.GetByID(ctx, tenantID, fileID)
	if err != nil {
		return fmt.Errorf("loading file: %w", err)
	}
	if record == nil {
		return &ServiceError{Code: http.StatusNotFound, ErrCode: "NOT_FOUND", Message: "file not found"}
	}

	if err := s.repo.SoftDelete(ctx, tenantID, fileID); err != nil {
		return fmt.Errorf("deleting file: %w", err)
	}

	s.logAccess(ctx, fileID, tenantID, userID, "delete", ipAddress, userAgent)
	s.metrics.DeletesTotal.WithLabelValues(record.Suite).Inc()

	s.publishFileEvent(ctx, "com.clario360.file.deleted", tenantID, userID, map[string]interface{}{
		"file_id":   fileID,
		"tenant_id": tenantID,
		"user_id":   userID,
	})

	return nil
}

// GetVersions returns version history for a file.
func (s *FileService) GetVersions(ctx context.Context, tenantID, fileID string) ([]*model.FileRecord, error) {
	return s.repo.GetVersions(ctx, tenantID, fileID)
}

// GetAccessLog returns file access history.
func (s *FileService) GetAccessLog(ctx context.Context, tenantID, fileID string, page, perPage int) ([]*model.FileAccessLog, int, error) {
	return s.repo.GetAccessLog(ctx, tenantID, fileID, page, perPage)
}

// GeneratePresignedUpload creates a presigned upload URL and pending record.
func (s *FileService) GeneratePresignedUpload(ctx context.Context, req *dto.PresignedUploadRequest, tenantID, userID string) (*dto.PresignedUploadResponse, error) {
	sanitizedName := storage.SanitizeFilename(req.Filename)
	storageKey := storage.GenerateStorageKey(tenantID, req.Suite, req.Filename)
	bucket := s.bucketName(req.Suite)

	if err := s.ensureBucket(ctx, bucket); err != nil {
		return nil, fmt.Errorf("ensuring bucket: %w", err)
	}

	presigned, err := s.store.GeneratePresignedUploadURL(ctx, storage.PresignedUploadParams{
		Bucket:       bucket,
		Key:          storageKey,
		ContentType:  req.ContentType,
		MaxSizeBytes: req.SizeBytes,
		Expiry:       s.presignedExpiry,
	})
	if err != nil {
		s.metrics.MinIOErrorsTotal.WithLabelValues("presigned_upload").Inc()
		return nil, &ServiceError{Code: http.StatusBadGateway, ErrCode: "STORAGE_ERROR", Message: "failed to generate presigned URL"}
	}

	// Create pending file record
	fileID := uuid.New().String()
	now := time.Now().UTC()

	var entityType, entityID *string
	if req.EntityType != "" {
		entityType = &req.EntityType
	}
	if req.EntityID != "" {
		entityID = &req.EntityID
	}

	policy := model.LifecycleStandard
	if req.LifecyclePolicy != "" {
		policy = req.LifecyclePolicy
	}

	record := &model.FileRecord{
		ID:              fileID,
		TenantID:        tenantID,
		Bucket:          bucket,
		StorageKey:      storageKey,
		OriginalName:    req.Filename,
		SanitizedName:   sanitizedName,
		ContentType:     req.ContentType,
		SizeBytes:       0, // updated on confirm
		ChecksumSHA256:  "", // computed on confirm
		Encrypted:       req.Encrypt,
		VirusScanStatus: model.ScanStatusPending,
		UploadedBy:      userID,
		Suite:           req.Suite,
		EntityType:      entityType,
		EntityID:        entityID,
		Tags:            []string{},
		VersionNumber:   1,
		LifecyclePolicy: policy,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("creating pending file: %w", err)
	}

	s.metrics.PresignedURLsGenerated.WithLabelValues("upload", req.Suite).Inc()
	s.logAccess(ctx, fileID, tenantID, userID, "presigned_upload", "", "")

	return &dto.PresignedUploadResponse{
		FileID:    fileID,
		URL:       presigned.URL,
		Method:    presigned.Method,
		Headers:   presigned.Headers,
		ExpiresAt: presigned.ExpiresAt.Format(time.RFC3339),
	}, nil
}

// ConfirmPresignedUpload verifies and completes a presigned upload.
func (s *FileService) ConfirmPresignedUpload(ctx context.Context, fileID, tenantID, userID string) (*model.FileRecord, error) {
	record, err := s.repo.GetByID(ctx, tenantID, fileID)
	if err != nil {
		return nil, fmt.Errorf("loading file: %w", err)
	}
	if record == nil {
		return nil, &ServiceError{Code: http.StatusNotFound, ErrCode: "NOT_FOUND", Message: "file not found"}
	}

	// Verify file exists in storage
	info, err := s.store.GetObjectInfo(ctx, record.Bucket, record.StorageKey)
	if err != nil {
		return nil, &ServiceError{Code: http.StatusNotFound, ErrCode: "NOT_FOUND", Message: "file not yet uploaded to storage"}
	}

	// Update record with actual details
	versionID := ""
	if info.VersionID != "" {
		versionID = info.VersionID
	}

	if err := s.repo.UpdateAfterPresignedUpload(ctx, tenantID, fileID, info.Size, versionID); err != nil {
		return nil, fmt.Errorf("updating file record: %w", err)
	}

	// Queue virus scan
	s.publishFileEvent(ctx, "com.clario360.file.uploaded", tenantID, userID, map[string]interface{}{
		"file_id":      fileID,
		"tenant_id":    tenantID,
		"suite":        record.Suite,
		"size_bytes":   info.Size,
		"content_type": record.ContentType,
		"encrypted":    record.Encrypted,
	})

	record.SizeBytes = info.Size
	return record, nil
}

// GeneratePresignedDownload creates a presigned download URL.
func (s *FileService) GeneratePresignedDownload(ctx context.Context, tenantID, fileID, userID, ipAddress, userAgent string) (*dto.PresignedDownloadResponse, error) {
	record, err := s.repo.GetByID(ctx, tenantID, fileID)
	if err != nil {
		return nil, fmt.Errorf("loading file: %w", err)
	}
	if record == nil {
		return nil, &ServiceError{Code: http.StatusNotFound, ErrCode: "NOT_FOUND", Message: "file not found"}
	}
	if record.VirusScanStatus == model.ScanStatusInfected {
		return nil, &ServiceError{Code: http.StatusForbidden, ErrCode: "FORBIDDEN", Message: "file is quarantined"}
	}
	if record.Encrypted {
		return nil, &ServiceError{Code: http.StatusBadRequest, ErrCode: "VALIDATION_ERROR", Message: "presigned download not available for encrypted files; use direct download"}
	}

	presigned, err := s.store.GeneratePresignedDownloadURL(ctx, record.Bucket, record.StorageKey, s.presignedExpiry)
	if err != nil {
		s.metrics.MinIOErrorsTotal.WithLabelValues("presigned_download").Inc()
		return nil, &ServiceError{Code: http.StatusBadGateway, ErrCode: "STORAGE_ERROR", Message: "failed to generate presigned URL"}
	}

	s.logAccess(ctx, fileID, tenantID, userID, "presigned_download", ipAddress, userAgent)
	s.metrics.PresignedURLsGenerated.WithLabelValues("download", record.Suite).Inc()

	return &dto.PresignedDownloadResponse{
		URL:       presigned.URL,
		Method:    presigned.Method,
		ExpiresAt: presigned.ExpiresAt.Format(time.RFC3339),
	}, nil
}

// RescanFile re-triggers a virus scan for a file.
func (s *FileService) RescanFile(ctx context.Context, tenantID, fileID string) error {
	record, err := s.repo.GetByID(ctx, tenantID, fileID)
	if err != nil {
		return fmt.Errorf("loading file: %w", err)
	}
	if record == nil {
		return &ServiceError{Code: http.StatusNotFound, ErrCode: "NOT_FOUND", Message: "file not found"}
	}

	// Reset scan status to pending; bail out if the optimistic update fails (concurrent state change).
	now := time.Now()
	updated, err := s.repo.UpdateScanStatus(ctx, tenantID, fileID, record.VirusScanStatus, model.ScanStatusPending, nil, &now)
	if err != nil {
		return fmt.Errorf("resetting scan status: %w", err)
	}
	if !updated {
		// Another process changed the scan status between our read and this update; treat as a no-op.
		s.logger.Warn().Str("file_id", fileID).Str("expected_status", record.VirusScanStatus).Msg("rescan: scan status changed concurrently, skipping re-queue")
		return nil
	}

	s.publishFileEvent(ctx, "com.clario360.file.uploaded", record.TenantID, "", map[string]interface{}{
		"file_id":      fileID,
		"tenant_id":    record.TenantID,
		"suite":        record.Suite,
		"size_bytes":   record.SizeBytes,
		"content_type": record.ContentType,
		"encrypted":    record.Encrypted,
	})

	return nil
}

// GetStorageStats returns storage usage statistics.
func (s *FileService) GetStorageStats(ctx context.Context, tenantID string) ([]repository.StorageStat, error) {
	return s.repo.GetStorageStats(ctx, tenantID)
}

// ListQuarantined lists unresolved quarantine entries.
func (s *FileService) ListQuarantined(ctx context.Context, tenantID string, page, perPage int) ([]*model.QuarantineLog, int, error) {
	return s.repo.ListQuarantined(ctx, tenantID, page, perPage)
}

// ResolveQuarantine marks a quarantine entry as resolved.
func (s *FileService) ResolveQuarantine(ctx context.Context, tenantID, quarantineID, resolvedBy, action string) error {
	return s.repo.ResolveQuarantine(ctx, tenantID, quarantineID, resolvedBy, action)
}

func (s *FileService) bucketName(suite string) string {
	return s.bucketPrefix + "-" + suite
}

func (s *FileService) ensureBucket(ctx context.Context, bucket string) error {
	s.bucketCacheMu.RLock()
	if s.bucketCache[bucket] {
		s.bucketCacheMu.RUnlock()
		return nil
	}
	s.bucketCacheMu.RUnlock()

	if err := s.store.EnsureBucket(ctx, bucket); err != nil {
		return err
	}

	s.bucketCacheMu.Lock()
	s.bucketCache[bucket] = true
	s.bucketCacheMu.Unlock()
	return nil
}

func (s *FileService) logAccess(ctx context.Context, fileID, tenantID, userID, action, ipAddress, userAgent string) {
	log := &model.FileAccessLog{
		FileID:    fileID,
		TenantID:  tenantID,
		UserID:    userID,
		Action:    action,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}
	if err := s.repo.LogAccess(ctx, log); err != nil {
		s.logger.Error().Err(err).Str("file_id", fileID).Str("action", action).Msg("failed to log file access")
	}
	s.metrics.AccessLogEntriesTotal.WithLabelValues(action).Inc()
}

func (s *FileService) publishFileEvent(ctx context.Context, eventType, tenantID, userID string, data map[string]interface{}) {
	if s.producer == nil {
		return
	}
	event, err := events.NewEvent(eventType, "file-service", tenantID, data)
	if err != nil {
		s.logger.Error().Err(err).Str("type", eventType).Msg("failed to create file event")
		return
	}
	event.UserID = userID
	if err := s.producer.Publish(ctx, fileEventsTopic, event); err != nil {
		s.logger.Error().Err(err).Str("type", eventType).Msg("failed to publish file event")
	}
}

// ServiceError is a structured error with HTTP status and error code.
type ServiceError struct {
	Code    int    `json:"-"`
	ErrCode string `json:"code"`
	Message string `json:"message"`
}

func (e *ServiceError) Error() string {
	return e.Message
}
