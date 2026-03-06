package service

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/filemanager/metrics"
	"github.com/clario360/platform/internal/filemanager/repository"
	"github.com/clario360/platform/pkg/storage"
)

const lifecycleBatchSize = 500

// LifecycleService manages file expiry, purging, and cleanup.
type LifecycleService struct {
	repo     *repository.FileRepository
	store    storage.Storage
	producer *events.Producer
	metrics  *metrics.FileMetrics
	logger   zerolog.Logger

	quarantineBucket string
}

// NewLifecycleService creates a new lifecycle service.
func NewLifecycleService(
	repo *repository.FileRepository,
	store storage.Storage,
	producer *events.Producer,
	fm *metrics.FileMetrics,
	logger zerolog.Logger,
	quarantineBucket string,
) *LifecycleService {
	return &LifecycleService{
		repo:             repo,
		store:            store,
		producer:         producer,
		metrics:          fm,
		logger:           logger,
		quarantineBucket: quarantineBucket,
	}
}

// RunCleanup executes all lifecycle cleanup tasks.
func (s *LifecycleService) RunCleanup(ctx context.Context) {
	s.logger.Info().Msg("starting lifecycle cleanup")
	start := time.Now()

	s.expireTemporary(ctx)
	s.purgeSoftDeleted(ctx)
	s.cleanQuarantine(ctx)
	s.retryPendingScans(ctx)

	s.logger.Info().Dur("duration", time.Since(start)).Msg("lifecycle cleanup completed")
}

// expireTemporary soft-deletes temporary files past their expiry.
func (s *LifecycleService) expireTemporary(ctx context.Context) {
	for {
		files, err := s.repo.GetExpiredTemporary(ctx, lifecycleBatchSize)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to get expired temporary files")
			return
		}
		if len(files) == 0 {
			return
		}

		for _, f := range files {
			if ctx.Err() != nil {
				return
			}

			// Delete from storage
			if err := s.store.Delete(ctx, f.Bucket, f.StorageKey); err != nil {
				s.logger.Error().Err(err).Str("file_id", f.ID).Msg("failed to delete expired file from storage")
				continue
			}

			// Soft-delete in DB
			if err := s.repo.SoftDelete(ctx, f.TenantID, f.ID); err != nil {
				s.logger.Error().Err(err).Str("file_id", f.ID).Msg("failed to soft-delete expired file")
				continue
			}

			s.metrics.LifecycleExpiredTotal.Inc()

			s.publishLifecycleEvent(ctx, "com.clario360.file.expired", f.TenantID, map[string]interface{}{
				"file_id":          f.ID,
				"lifecycle_policy": f.LifecyclePolicy,
			})

			s.logger.Info().Str("file_id", f.ID).Str("tenant_id", f.TenantID).Msg("expired temporary file")
		}

		if len(files) < lifecycleBatchSize {
			return
		}
	}
}

// purgeSoftDeleted hard-deletes files that were soft-deleted more than 30 days ago.
func (s *LifecycleService) purgeSoftDeleted(ctx context.Context) {
	for {
		files, err := s.repo.GetSoftDeletedForPurge(ctx, 30, lifecycleBatchSize)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to get purge candidates")
			return
		}
		if len(files) == 0 {
			return
		}

		for _, f := range files {
			if ctx.Err() != nil {
				return
			}

			// Delete from storage (may already be gone)
			_ = s.store.Delete(ctx, f.Bucket, f.StorageKey)

			// Hard-delete from DB
			if err := s.repo.HardDelete(ctx, f.ID); err != nil {
				s.logger.Error().Err(err).Str("file_id", f.ID).Msg("failed to hard-delete file")
				continue
			}

			s.metrics.LifecyclePurgedTotal.Inc()
			s.logger.Info().Str("file_id", f.ID).Str("tenant_id", f.TenantID).Msg("purged soft-deleted file")
		}

		if len(files) < lifecycleBatchSize {
			return
		}
	}
}

// cleanQuarantine deletes quarantined files older than 90 days that are unresolved.
func (s *LifecycleService) cleanQuarantine(ctx context.Context) {
	entries, err := s.repo.GetOldQuarantine(ctx, 90, lifecycleBatchSize)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get old quarantine entries")
		return
	}

	for _, q := range entries {
		if ctx.Err() != nil {
			return
		}

		// Delete from quarantine bucket
		_ = s.store.Delete(ctx, q.QuarantineBucket, q.QuarantineKey)

		// Mark as resolved
		if err := s.repo.ResolveQuarantine(ctx, q.ID, "system", "deleted"); err != nil {
			s.logger.Error().Err(err).Str("quarantine_id", q.ID).Msg("failed to resolve old quarantine")
		}

		s.logger.Info().Str("quarantine_id", q.ID).Str("file_id", q.FileID).Msg("cleaned old quarantine entry")
	}
}

// retryPendingScans re-publishes scan events for files stuck in pending.
func (s *LifecycleService) retryPendingScans(ctx context.Context) {
	files, err := s.repo.GetPendingScans(ctx, 10*time.Minute, lifecycleBatchSize)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get pending scans for retry")
		return
	}

	for _, f := range files {
		if ctx.Err() != nil {
			return
		}

		s.publishLifecycleEvent(ctx, "com.clario360.file.uploaded", f.TenantID, map[string]interface{}{
			"file_id":      f.ID,
			"tenant_id":    f.TenantID,
			"suite":        f.Suite,
			"size_bytes":   f.SizeBytes,
			"content_type": f.ContentType,
			"encrypted":    f.Encrypted,
		})

		s.logger.Info().Str("file_id", f.ID).Msg("retried pending scan")
	}
}

func (s *LifecycleService) publishLifecycleEvent(ctx context.Context, eventType, tenantID string, data map[string]interface{}) {
	if s.producer == nil {
		return
	}
	event, err := events.NewEvent(eventType, "file-service", tenantID, data)
	if err != nil {
		s.logger.Error().Err(err).Str("type", eventType).Msg("failed to create lifecycle event")
		return
	}
	if err := s.producer.Publish(ctx, fileEventsTopic, event); err != nil {
		s.logger.Error().Err(err).Str("type", eventType).Msg("failed to publish lifecycle event")
	}
}
