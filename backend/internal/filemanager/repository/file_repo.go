package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/filemanager/model"
)

// FileRepository handles file metadata persistence.
type FileRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewFileRepository creates a new file repository.
func NewFileRepository(db *pgxpool.Pool, logger zerolog.Logger) *FileRepository {
	return &FileRepository{db: db, logger: logger}
}

// parseTenant helper
func parseTenant(tenantID string) (uuid.UUID, error) {
	u, err := uuid.Parse(tenantID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid tenant UUID %q: %w", tenantID, err)
	}
	return u, nil
}

// Create inserts a new file record.
func (r *FileRepository) Create(ctx context.Context, f *model.FileRecord) error {
	tenant, err := parseTenant(f.TenantID)
	if err != nil {
		return err
	}

	return database.RunWithTenant(ctx, r.db, tenant, func(tx pgx.Tx) error {
		query := `INSERT INTO files (
			id, tenant_id, bucket, storage_key, original_name, sanitized_name,
			content_type, detected_content_type, size_bytes, checksum_sha256,
			encrypted, encryption_metadata, virus_scan_status,
			uploaded_by, suite, entity_type, entity_id, tags,
			version_id, version_number, is_public, lifecycle_policy, expires_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13,
			$14, $15, $16, $17, $18,
			$19, $20, $21, $22, $23,
			$24, $25
		)`

		_, txErr := tx.Exec(ctx, query,
			f.ID, f.TenantID, f.Bucket, f.StorageKey, f.OriginalName, f.SanitizedName,
			f.ContentType, f.DetectedContentType, f.SizeBytes, f.ChecksumSHA256,
			f.Encrypted, f.EncryptionMetadata, f.VirusScanStatus,
			f.UploadedBy, f.Suite, f.EntityType, f.EntityID, f.Tags,
			f.VersionID, f.VersionNumber, f.IsPublic, f.LifecyclePolicy, f.ExpiresAt,
			f.CreatedAt, f.UpdatedAt,
		)
		if txErr != nil {
			return fmt.Errorf("creating file record: %w", txErr)
		}
		return nil
	})
}

// GetByID retrieves a file by ID with tenant isolation.
func (r *FileRepository) GetByID(ctx context.Context, tenantID, fileID string) (*model.FileRecord, error) {
	tenant, err := parseTenant(tenantID)
	if err != nil {
		return nil, err
	}

	var record *model.FileRecord
	err = database.RunReadWithTenant(ctx, r.db, tenant, func(tx pgx.Tx) error {
		query := `SELECT id, tenant_id, bucket, storage_key, original_name, sanitized_name,
			content_type, detected_content_type, size_bytes, checksum_sha256,
			encrypted, encryption_metadata, virus_scan_status, virus_scan_result, virus_scanned_at,
			uploaded_by, suite, entity_type, entity_id, tags,
			version_id, version_number, is_public, lifecycle_policy, expires_at,
			created_at, updated_at, deleted_at
		FROM files WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`

		var txErr error
		record, txErr = r.scanRow(tx.QueryRow(ctx, query, fileID, tenantID))
		return txErr
	})
	return record, err
}

// List retrieves files for a tenant with filters and pagination.
func (r *FileRepository) List(ctx context.Context, tenantID, suite, entityType, entityID, uploadedBy, tag string, page, perPage int) ([]*model.FileRecord, int, error) {
	tenant, err := parseTenant(tenantID)
	if err != nil {
		return nil, 0, err
	}

	var files []*model.FileRecord
	var total int

	err = database.RunReadWithTenant(ctx, r.db, tenant, func(tx pgx.Tx) error {
		var conditions []string
		var args []interface{}
		argIdx := 1

		conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
		args = append(args, tenantID)
		argIdx++

		conditions = append(conditions, "deleted_at IS NULL")

		if suite != "" {
			conditions = append(conditions, fmt.Sprintf("suite = $%d", argIdx))
			args = append(args, suite)
			argIdx++
		}
		if entityType != "" {
			conditions = append(conditions, fmt.Sprintf("entity_type = $%d", argIdx))
			args = append(args, entityType)
			argIdx++
		}
		if entityID != "" {
			conditions = append(conditions, fmt.Sprintf("entity_id = $%d", argIdx))
			args = append(args, entityID)
			argIdx++
		}
		if uploadedBy != "" {
			conditions = append(conditions, fmt.Sprintf("uploaded_by = $%d", argIdx))
			args = append(args, uploadedBy)
			argIdx++
		}
		if tag != "" {
			conditions = append(conditions, fmt.Sprintf("$%d = ANY(tags)", argIdx))
			args = append(args, tag)
			argIdx++
		}

		where := strings.Join(conditions, " AND ")

		countQuery := "SELECT COUNT(*) FROM files WHERE " + where
		if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
			return fmt.Errorf("counting files: %w", err)
		}

		offset := (page - 1) * perPage
		dataQuery := fmt.Sprintf(`SELECT id, tenant_id, bucket, storage_key, original_name, sanitized_name,
			content_type, detected_content_type, size_bytes, checksum_sha256,
			encrypted, encryption_metadata, virus_scan_status, virus_scan_result, virus_scanned_at,
			uploaded_by, suite, entity_type, entity_id, tags,
			version_id, version_number, is_public, lifecycle_policy, expires_at,
			created_at, updated_at, deleted_at
		FROM files WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
			where, argIdx, argIdx+1)
		args = append(args, perPage, offset)

		rows, err := tx.Query(ctx, dataQuery, args...)
		if err != nil {
			return fmt.Errorf("listing files: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			f, err := r.scanRows(rows)
			if err != nil {
				return err
			}
			files = append(files, f)
		}
		return nil
	})

	return files, total, err
}

// FindByChecksum finds files with the same checksum for dedup.
func (r *FileRepository) FindByChecksum(ctx context.Context, tenantID, checksum, entityType, entityID string) (*model.FileRecord, error) {
	tenant, err := parseTenant(tenantID)
	if err != nil {
		return nil, err
	}

	var file *model.FileRecord
	err = database.RunReadWithTenant(ctx, r.db, tenant, func(tx pgx.Tx) error {
		query := `SELECT id, tenant_id, bucket, storage_key, original_name, sanitized_name,
			content_type, detected_content_type, size_bytes, checksum_sha256,
			encrypted, encryption_metadata, virus_scan_status, virus_scan_result, virus_scanned_at,
			uploaded_by, suite, entity_type, entity_id, tags,
			version_id, version_number, is_public, lifecycle_policy, expires_at,
			created_at, updated_at, deleted_at
		FROM files WHERE tenant_id = $1 AND checksum_sha256 = $2 AND entity_type = $3 AND entity_id = $4 AND deleted_at IS NULL
		LIMIT 1`

		var txErr error
		file, txErr = r.scanRow(tx.QueryRow(ctx, query, tenantID, checksum, entityType, entityID))
		return txErr
	})
	if err != nil || file == nil {
		return nil, nil // not found is fine for dedup
	}
	return file, nil
}

// SoftDelete marks a file as deleted.
func (r *FileRepository) SoftDelete(ctx context.Context, tenantID, fileID string) error {
	tenant, err := parseTenant(tenantID)
	if err != nil {
		return err
	}

	return database.RunWithTenant(ctx, r.db, tenant, func(tx pgx.Tx) error {
		query := `UPDATE files SET deleted_at = now(), updated_at = now()
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
		tag, txErr := tx.Exec(ctx, query, fileID, tenantID)
		if txErr != nil {
			return fmt.Errorf("soft-deleting file: %w", txErr)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("file not found")
		}
		return nil
	})
}

// UpdateScanStatus atomically updates virus scan status.
func (r *FileRepository) UpdateScanStatus(ctx context.Context, tenantID, fileID, expectedStatus, newStatus string, result *string, scannedAt *time.Time) (bool, error) {
	tenant, err := parseTenant(tenantID)
	if err != nil {
		return false, err
	}

	var updated bool
	err = database.RunWithTenant(ctx, r.db, tenant, func(tx pgx.Tx) error {
		query := `UPDATE files SET virus_scan_status = $1, virus_scan_result = $2, virus_scanned_at = $3, updated_at = now()
		WHERE id = $4 AND tenant_id = $5 AND virus_scan_status = $6`
		tag, txErr := tx.Exec(ctx, query, newStatus, result, scannedAt, fileID, tenantID, expectedStatus)
		if txErr != nil {
			return fmt.Errorf("updating scan status: %w", txErr)
		}
		updated = tag.RowsAffected() > 0
		return nil
	})
	return updated, err
}

// UpdateAfterPresignedUpload updates a file record after presigned upload confirmation.
func (r *FileRepository) UpdateAfterPresignedUpload(ctx context.Context, tenantID, fileID string, sizeBytes int64, versionID string) error {
	tenant, err := parseTenant(tenantID)
	if err != nil {
		return err
	}
	return database.RunWithTenant(ctx, r.db, tenant, func(tx pgx.Tx) error {
		query := `UPDATE files SET size_bytes = $1, version_id = $2, updated_at = now()
		WHERE id = $3 AND tenant_id = $4`
		_, txErr := tx.Exec(ctx, query, sizeBytes, versionID, fileID, tenantID)
		if txErr != nil {
			return fmt.Errorf("updating presigned upload: %w", txErr)
		}
		return nil
	})
}

// LogAccess records a file access entry.
func (r *FileRepository) LogAccess(ctx context.Context, log *model.FileAccessLog) error {
	tenant, err := parseTenant(log.TenantID)
	if err != nil {
		return err
	}
	return database.RunWithTenant(ctx, r.db, tenant, func(tx pgx.Tx) error {
		query := `INSERT INTO file_access_log (file_id, tenant_id, user_id, action, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6)`
		_, txErr := tx.Exec(ctx, query, log.FileID, log.TenantID, log.UserID, log.Action, log.IPAddress, log.UserAgent)
		if txErr != nil {
			return fmt.Errorf("logging file access: %w", txErr)
		}
		return nil
	})
}

// GetAccessLog retrieves file access history.
func (r *FileRepository) GetAccessLog(ctx context.Context, tenantID, fileID string, page, perPage int) ([]*model.FileAccessLog, int, error) {
	tenant, err := parseTenant(tenantID)
	if err != nil {
		return nil, 0, err
	}
	logs := make([]*model.FileAccessLog, 0, perPage)
	var total int

	err = database.RunReadWithTenant(ctx, r.db, tenant, func(tx pgx.Tx) error {
		countQuery := `SELECT COUNT(*) FROM file_access_log WHERE file_id = $1 AND tenant_id = $2`
		if err := tx.QueryRow(ctx, countQuery, fileID, tenantID).Scan(&total); err != nil {
			return fmt.Errorf("counting access log: %w", err)
		}

		offset := (page - 1) * perPage
		query := `SELECT id, file_id, tenant_id, user_id, action, ip_address, user_agent, created_at
		FROM file_access_log WHERE file_id = $1 AND tenant_id = $2
		ORDER BY created_at DESC LIMIT $3 OFFSET $4`

		rows, err := tx.Query(ctx, query, fileID, tenantID, perPage, offset)
		if err != nil {
			return fmt.Errorf("querying access log: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var l model.FileAccessLog
			if err := rows.Scan(&l.ID, &l.FileID, &l.TenantID, &l.UserID, &l.Action, &l.IPAddress, &l.UserAgent, &l.CreatedAt); err != nil {
				return fmt.Errorf("scanning access log: %w", err)
			}
			logs = append(logs, &l)
		}
		return nil
	})
	return logs, total, err
}

// CreateQuarantineLog records a quarantined file.
func (r *FileRepository) CreateQuarantineLog(ctx context.Context, q *model.QuarantineLog) error {
	tenant, err := parseTenant(q.TenantID)
	if err != nil {
		return err
	}
	return database.RunWithTenant(ctx, r.db, tenant, func(tx pgx.Tx) error {
		query := `INSERT INTO file_quarantine_log (
			file_id, tenant_id, original_bucket, original_key, quarantine_bucket, quarantine_key,
			virus_name, scanned_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
		_, txErr := tx.Exec(ctx, query,
			q.FileID, q.TenantID, q.OriginalBucket, q.OriginalKey, q.QuarantineBucket, q.QuarantineKey,
			q.VirusName, q.ScannedAt,
		)
		if txErr != nil {
			return fmt.Errorf("creating quarantine log: %w", txErr)
		}
		return nil
	})
}

// ListQuarantined lists unresolved quarantine entries.
func (r *FileRepository) ListQuarantined(ctx context.Context, tenantID string, page, perPage int) ([]*model.QuarantineLog, int, error) {
	tenant, err := parseTenant(tenantID)
	if err != nil {
		return nil, 0, err
	}
	logs := make([]*model.QuarantineLog, 0, perPage)
	var total int

	err = database.RunReadWithTenant(ctx, r.db, tenant, func(tx pgx.Tx) error {
		countQuery := `SELECT COUNT(*) FROM file_quarantine_log WHERE resolved = false AND tenant_id = $1`
		if err := tx.QueryRow(ctx, countQuery, tenantID).Scan(&total); err != nil {
			return fmt.Errorf("counting quarantine: %w", err)
		}

		offset := (page - 1) * perPage
		query := `SELECT id, file_id, tenant_id, original_bucket, original_key, quarantine_bucket, quarantine_key,
			virus_name, scanned_at, quarantined_at, resolved, resolved_by, resolved_at, resolution_action
		FROM file_quarantine_log WHERE resolved = false AND tenant_id = $1
		ORDER BY quarantined_at DESC LIMIT $2 OFFSET $3`

		rows, err := tx.Query(ctx, query, tenantID, perPage, offset)
		if err != nil {
			return fmt.Errorf("listing quarantine: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var q model.QuarantineLog
			if err := rows.Scan(&q.ID, &q.FileID, &q.TenantID, &q.OriginalBucket, &q.OriginalKey,
				&q.QuarantineBucket, &q.QuarantineKey, &q.VirusName, &q.ScannedAt,
				&q.QuarantinedAt, &q.Resolved, &q.ResolvedBy, &q.ResolvedAt, &q.ResolutionAction); err != nil {
				return fmt.Errorf("scanning quarantine: %w", err)
			}
			logs = append(logs, &q)
		}
		return nil
	})
	return logs, total, err
}

// ResolveQuarantine marks a quarantine entry as resolved.
func (r *FileRepository) ResolveQuarantine(ctx context.Context, tenantID, quarantineID, resolvedBy, action string) error {
	tenant, err := parseTenant(tenantID)
	if err != nil {
		return err
	}
	return database.RunWithTenant(ctx, r.db, tenant, func(tx pgx.Tx) error {
		query := `UPDATE file_quarantine_log SET resolved = true, resolved_by = $1, resolved_at = now(), resolution_action = $2
		WHERE id = $3 AND tenant_id = $4 AND resolved = false`
		tag, txErr := tx.Exec(ctx, query, resolvedBy, action, quarantineID, tenantID)
		if txErr != nil {
			return fmt.Errorf("resolving quarantine: %w", txErr)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("quarantine entry not found or already resolved")
		}
		return nil
	})
}

// GetExpiredTemporary finds temporary files past their expiry.
func (r *FileRepository) GetExpiredTemporary(ctx context.Context, limit int) ([]*model.FileRecord, error) {
	var files []*model.FileRecord
	err := database.RunSystemRead(ctx, r.db, func(tx pgx.Tx) error {
		query := `SELECT id, tenant_id, bucket, storage_key, original_name, sanitized_name,
			content_type, detected_content_type, size_bytes, checksum_sha256,
			encrypted, encryption_metadata, virus_scan_status, virus_scan_result, virus_scanned_at,
			uploaded_by, suite, entity_type, entity_id, tags,
			version_id, version_number, is_public, lifecycle_policy, expires_at,
			created_at, updated_at, deleted_at
		FROM files WHERE lifecycle_policy = 'temporary' AND expires_at < now() AND deleted_at IS NULL
		LIMIT $1`

		rows, err := tx.Query(ctx, query, limit)
		if err != nil {
			return fmt.Errorf("getting expired temporary: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			f, err := r.scanRows(rows)
			if err != nil {
				return err
			}
			files = append(files, f)
		}
		return nil
	})
	return files, err
}

// GetSoftDeletedForPurge finds files soft-deleted more than retentionDays ago.
func (r *FileRepository) GetSoftDeletedForPurge(ctx context.Context, retentionDays int, limit int) ([]*model.FileRecord, error) {
	var files []*model.FileRecord
	err := database.RunSystemRead(ctx, r.db, func(tx pgx.Tx) error {
		query := `SELECT id, tenant_id, bucket, storage_key, original_name, sanitized_name,
			content_type, detected_content_type, size_bytes, checksum_sha256,
			encrypted, encryption_metadata, virus_scan_status, virus_scan_result, virus_scanned_at,
			uploaded_by, suite, entity_type, entity_id, tags,
			version_id, version_number, is_public, lifecycle_policy, expires_at,
			created_at, updated_at, deleted_at
		FROM files WHERE deleted_at IS NOT NULL AND deleted_at < now() - ($1::int * INTERVAL '1 day')
		LIMIT $2`

		rows, err := tx.Query(ctx, query, retentionDays, limit)
		if err != nil {
			return fmt.Errorf("getting purge candidates: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			f, err := r.scanRows(rows)
			if err != nil {
				return err
			}
			files = append(files, f)
		}
		return nil
	})
	return files, err
}

// HardDelete permanently removes a file record. (using RunSystemTx due to global reach if tenant unknown/not needed)
func (r *FileRepository) HardDelete(ctx context.Context, tenantID, fileID string) error {
	tenant, err := parseTenant(tenantID)
	if err != nil {
		return err
	}
	return database.RunWithTenant(ctx, r.db, tenant, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM file_access_log WHERE file_id = $1 AND tenant_id = $2`, fileID, tenantID); err != nil {
			return fmt.Errorf("deleting access logs: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM file_quarantine_log WHERE file_id = $1 AND tenant_id = $2`, fileID, tenantID); err != nil {
			return fmt.Errorf("deleting quarantine logs: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM files WHERE id = $1 AND tenant_id = $2`, fileID, tenantID); err != nil {
			return fmt.Errorf("hard-deleting file: %w", err)
		}
		return nil
	})
}

// GetPendingScans finds files with pending scan status older than the given age.
func (r *FileRepository) GetPendingScans(ctx context.Context, olderThan time.Duration, limit int) ([]*model.FileRecord, error) {
	var files []*model.FileRecord
	err := database.RunSystemRead(ctx, r.db, func(tx pgx.Tx) error {
		query := `SELECT id, tenant_id, bucket, storage_key, original_name, sanitized_name,
			content_type, detected_content_type, size_bytes, checksum_sha256,
			encrypted, encryption_metadata, virus_scan_status, virus_scan_result, virus_scanned_at,
			uploaded_by, suite, entity_type, entity_id, tags,
			version_id, version_number, is_public, lifecycle_policy, expires_at,
			created_at, updated_at, deleted_at
		FROM files WHERE virus_scan_status = 'pending' AND created_at < $1 AND deleted_at IS NULL
		LIMIT $2`

		cutoff := time.Now().Add(-olderThan)
		rows, err := tx.Query(ctx, query, cutoff, limit)
		if err != nil {
			return fmt.Errorf("getting pending scans: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			f, err := r.scanRows(rows)
			if err != nil {
				return err
			}
			files = append(files, f)
		}
		return nil
	})
	return files, err
}

// GetOldQuarantine finds quarantine entries older than the given days and unresolved.
func (r *FileRepository) GetOldQuarantine(ctx context.Context, olderThanDays int, limit int) ([]*model.QuarantineLog, error) {
	var logs []*model.QuarantineLog
	err := database.RunSystemRead(ctx, r.db, func(tx pgx.Tx) error {
		query := `SELECT id, file_id, tenant_id, original_bucket, original_key, quarantine_bucket, quarantine_key,
			virus_name, scanned_at, quarantined_at, resolved, resolved_by, resolved_at, resolution_action
		FROM file_quarantine_log WHERE resolved = false AND quarantined_at < now() - ($1::int * INTERVAL '1 day')
		LIMIT $2`

		rows, err := tx.Query(ctx, query, olderThanDays, limit)
		if err != nil {
			return fmt.Errorf("getting old quarantine: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var q model.QuarantineLog
			if err := rows.Scan(&q.ID, &q.FileID, &q.TenantID, &q.OriginalBucket, &q.OriginalKey,
				&q.QuarantineBucket, &q.QuarantineKey, &q.VirusName, &q.ScannedAt,
				&q.QuarantinedAt, &q.Resolved, &q.ResolvedBy, &q.ResolvedAt, &q.ResolutionAction); err != nil {
				return fmt.Errorf("scanning old quarantine: %w", err)
			}
			logs = append(logs, &q)
		}
		return nil
	})
	return logs, err
}

// GetStorageStats returns storage usage grouped by tenant and suite.
func (r *FileRepository) GetStorageStats(ctx context.Context, tenantID string) ([]StorageStat, error) {
	tenant, err := parseTenant(tenantID)
	if err != nil {
		return nil, err
	}
	var stats []StorageStat
	err = database.RunReadWithTenant(ctx, r.db, tenant, func(tx pgx.Tx) error {
		query := `SELECT tenant_id, suite, COUNT(*) as file_count, COALESCE(SUM(size_bytes), 0) as total_bytes
		FROM files WHERE deleted_at IS NULL AND tenant_id = $1
		GROUP BY tenant_id, suite
		ORDER BY total_bytes DESC`

		rows, err := tx.Query(ctx, query, tenantID)
		if err != nil {
			return fmt.Errorf("getting storage stats: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var s StorageStat
			if err := rows.Scan(&s.TenantID, &s.Suite, &s.FileCount, &s.TotalBytes); err != nil {
				return fmt.Errorf("scanning storage stats: %w", err)
			}
			stats = append(stats, s)
		}
		return nil
	})
	return stats, err
}

// StorageStat holds storage usage info.
type StorageStat struct {
	TenantID   string `json:"tenant_id"`
	Suite      string `json:"suite"`
	FileCount  int64  `json:"file_count"`
	TotalBytes int64  `json:"total_bytes"`
}

// GetVersions retrieves all versions of a file by entity association.
// Both the file lookup and the version scan run inside a single read transaction to eliminate
// the race between fetching the anchor file and querying its sibling versions.
func (r *FileRepository) GetVersions(ctx context.Context, tenantID, fileID string) ([]*model.FileRecord, error) {
	tenant, err := parseTenant(tenantID)
	if err != nil {
		return nil, err
	}

	const fileQuery = `SELECT id, tenant_id, bucket, storage_key, original_name, sanitized_name,
		content_type, detected_content_type, size_bytes, checksum_sha256,
		encrypted, encryption_metadata, virus_scan_status, virus_scan_result, virus_scanned_at,
		uploaded_by, suite, entity_type, entity_id, tags,
		version_id, version_number, is_public, lifecycle_policy, expires_at,
		created_at, updated_at, deleted_at
	FROM files WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`

	const versionsQuery = `SELECT id, tenant_id, bucket, storage_key, original_name, sanitized_name,
		content_type, detected_content_type, size_bytes, checksum_sha256,
		encrypted, encryption_metadata, virus_scan_status, virus_scan_result, virus_scanned_at,
		uploaded_by, suite, entity_type, entity_id, tags,
		version_id, version_number, is_public, lifecycle_policy, expires_at,
		created_at, updated_at, deleted_at
	FROM files WHERE tenant_id = $1 AND entity_type = $2 AND entity_id = $3 AND sanitized_name = $4 AND deleted_at IS NULL
	ORDER BY version_number DESC`

	var results []*model.FileRecord
	err = database.RunReadWithTenant(ctx, r.db, tenant, func(tx pgx.Tx) error {
		// 1. Load the anchor file within the same transaction snapshot.
		file, err := r.scanRow(tx.QueryRow(ctx, fileQuery, fileID, tenantID))
		if err != nil {
			return err
		}
		if file == nil {
			return fmt.Errorf("file not found")
		}

		// 2. Files without an entity association have no versioning lineage; return just the file.
		if file.EntityType == nil || file.EntityID == nil {
			results = []*model.FileRecord{file}
			return nil
		}

		// 3. Fetch sibling versions from the same transaction snapshot — consistent with step 1.
		rows, err := tx.Query(ctx, versionsQuery, tenantID, file.EntityType, file.EntityID, file.SanitizedName)
		if err != nil {
			return fmt.Errorf("getting versions: %w", err)
		}
		defer rows.Close()

		versions := make([]*model.FileRecord, 0, 8)
		for rows.Next() {
			f, err := r.scanRows(rows)
			if err != nil {
				return err
			}
			versions = append(versions, f)
		}

		// 4. Guard: version scan should always include the anchor file itself; fall back if not.
		if len(versions) == 0 {
			versions = []*model.FileRecord{file}
		}
		results = versions
		return nil
	})
	return results, err
}

// GetLatestVersionNumber returns the highest version number for a given entity+name combination.
func (r *FileRepository) GetLatestVersionNumber(ctx context.Context, tenantID, entityType, entityID, sanitizedName string) (int, error) {
	tenant, err := parseTenant(tenantID)
	if err != nil {
		return 0, err
	}
	var version int
	err = database.RunReadWithTenant(ctx, r.db, tenant, func(tx pgx.Tx) error {
		query := `SELECT COALESCE(MAX(version_number), 0) FROM files
		WHERE tenant_id = $1 AND entity_type = $2 AND entity_id = $3 AND sanitized_name = $4 AND deleted_at IS NULL`

		err := tx.QueryRow(ctx, query, tenantID, entityType, entityID, sanitizedName).Scan(&version)
		if err != nil {
			return fmt.Errorf("getting latest version: %w", err)
		}
		return nil
	})
	return version, err
}

func (r *FileRepository) scanRow(row pgx.Row) (*model.FileRecord, error) {
	var f model.FileRecord
	var encMeta []byte
	err := row.Scan(
		&f.ID, &f.TenantID, &f.Bucket, &f.StorageKey, &f.OriginalName, &f.SanitizedName,
		&f.ContentType, &f.DetectedContentType, &f.SizeBytes, &f.ChecksumSHA256,
		&f.Encrypted, &encMeta, &f.VirusScanStatus, &f.VirusScanResult, &f.VirusScannedAt,
		&f.UploadedBy, &f.Suite, &f.EntityType, &f.EntityID, &f.Tags,
		&f.VersionID, &f.VersionNumber, &f.IsPublic, &f.LifecyclePolicy, &f.ExpiresAt,
		&f.CreatedAt, &f.UpdatedAt, &f.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scanning file: %w", err)
	}
	if encMeta != nil {
		f.EncryptionMetadata = json.RawMessage(encMeta)
	}
	return &f, nil
}

func (r *FileRepository) scanRows(rows pgx.Rows) (*model.FileRecord, error) {
	var f model.FileRecord
	var encMeta []byte
	err := rows.Scan(
		&f.ID, &f.TenantID, &f.Bucket, &f.StorageKey, &f.OriginalName, &f.SanitizedName,
		&f.ContentType, &f.DetectedContentType, &f.SizeBytes, &f.ChecksumSHA256,
		&f.Encrypted, &encMeta, &f.VirusScanStatus, &f.VirusScanResult, &f.VirusScannedAt,
		&f.UploadedBy, &f.Suite, &f.EntityType, &f.EntityID, &f.Tags,
		&f.VersionID, &f.VersionNumber, &f.IsPublic, &f.LifecyclePolicy, &f.ExpiresAt,
		&f.CreatedAt, &f.UpdatedAt, &f.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scanning file row: %w", err)
	}
	if encMeta != nil {
		f.EncryptionMetadata = json.RawMessage(encMeta)
	}
	return &f, nil
}
