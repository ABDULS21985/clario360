package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7/pkg/tags"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	iammodel "github.com/clario360/platform/internal/iam/model"
	onboardingdto "github.com/clario360/platform/internal/onboarding/dto"
	"github.com/clario360/platform/pkg/storage"
)

type TenantDeprovisioner struct {
	store          tenantLifecycleStore
	onboardingRepo tenantIdentityRepository
	storage        *storage.MinIOStorage
	redis          *redis.Client
	producer       *events.Producer
	logger         zerolog.Logger
	metrics        *Metrics
}

type tenantLifecycleStore interface {
	SuspendUsers(ctx context.Context, tenantID uuid.UUID) error
	DeleteSessions(ctx context.Context, tenantID uuid.UUID) error
	RevokeAPIKeys(ctx context.Context, tenantID uuid.UUID) error
	SoftDeleteTenantRows(ctx context.Context, tenantID uuid.UUID) error
	RestoreTenantRows(ctx context.Context, tenantID uuid.UUID) error
	MarkTenantDeprovisioned(ctx context.Context, tenantID, adminID uuid.UUID, retainUntil time.Time) error
	MarkTenantActive(ctx context.Context, tenantID uuid.UUID) error
	InsertAuditLog(ctx context.Context, tenantID, adminID uuid.UUID, action string, metadata map[string]any) error
}

type sqlTenantLifecycleStore struct {
	platformPool *pgxpool.Pool
	dbPools      map[string]*pgxpool.Pool
}

func NewTenantDeprovisioner(
	platformPool *pgxpool.Pool,
	dbPools map[string]*pgxpool.Pool,
	onboardingRepo tenantIdentityRepository,
	storageClient *storage.MinIOStorage,
	redisClient *redis.Client,
	producer *events.Producer,
	logger zerolog.Logger,
	metrics *Metrics,
) *TenantDeprovisioner {
	return &TenantDeprovisioner{
		store:          &sqlTenantLifecycleStore{platformPool: platformPool, dbPools: dbPools},
		onboardingRepo: onboardingRepo,
		storage:        storageClient,
		redis:          redisClient,
		producer:       producer,
		logger:         logger.With().Str("service", "tenant_deprovisioner").Logger(),
		metrics:        metrics,
	}
}

func (d *TenantDeprovisioner) Deprovision(ctx context.Context, tenantID, adminID uuid.UUID, req onboardingdto.DeprovisionRequest) error {
	if strings.TrimSpace(req.Reason) == "" {
		return fmt.Errorf("deprovision reason is required: %w", iammodel.ErrValidation)
	}
	name, slug, status, _, err := d.onboardingRepo.GetTenantIdentity(ctx, tenantID)
	if err != nil {
		return err
	}
	if status == iammodel.TenantStatusDeprovisioned {
		return fmt.Errorf("tenant is already deprovisioned: %w", iammodel.ErrConflict)
	}
	retainUntil := time.Now().AddDate(0, 0, req.RetainDays)

	if err := d.store.SuspendUsers(ctx, tenantID); err != nil {
		return err
	}
	if err := d.store.DeleteSessions(ctx, tenantID); err != nil {
		return err
	}
	if err := d.clearRedisSessions(ctx, tenantID); err != nil {
		d.logger.Warn().Err(err).Str("tenant_id", tenantID.String()).Msg("redis session invalidation failed")
	}
	if err := d.store.RevokeAPIKeys(ctx, tenantID); err != nil {
		return err
	}
	if err := d.store.SoftDeleteTenantRows(ctx, tenantID); err != nil {
		return err
	}
	if err := d.tagTenantBuckets(ctx, slug, retainUntil); err != nil {
		return err
	}
	if err := d.store.MarkTenantDeprovisioned(ctx, tenantID, adminID, retainUntil); err != nil {
		return err
	}
	if err := d.store.InsertAuditLog(ctx, tenantID, adminID, "tenant.deprovisioned", map[string]any{
		"reason":       req.Reason,
		"retain_days":  req.RetainDays,
		"retain_until": retainUntil,
		"tenant_name":  name,
	}); err != nil {
		return err
	}
	publishOnboardingEvent(ctx, d.producer,
		"com.clario360.platform.tenant.deprovisioned",
		tenantID,
		&adminID,
		map[string]any{"tenant_id": tenantID.String(), "reason": req.Reason},
		d.logger,
	)
	if d.metrics != nil && d.metrics.deprovisionsTotal != nil {
		d.metrics.deprovisionsTotal.WithLabelValues().Inc()
	}
	return nil
}

func (d *TenantDeprovisioner) Reactivate(ctx context.Context, tenantID, adminID uuid.UUID) error {
	_, slug, status, retainUntil, err := d.onboardingRepo.GetTenantIdentity(ctx, tenantID)
	if err != nil {
		return err
	}
	if status != iammodel.TenantStatusDeprovisioned {
		return fmt.Errorf("tenant is not deprovisioned: %w", iammodel.ErrConflict)
	}
	if retainUntil == nil || retainUntil.Before(time.Now()) {
		return fmt.Errorf("tenant retention window has expired: %w", iammodel.ErrForbidden)
	}
	if err := d.store.RestoreTenantRows(ctx, tenantID); err != nil {
		return err
	}
	if err := d.activateTenantBuckets(ctx, slug); err != nil {
		return err
	}
	if err := d.store.MarkTenantActive(ctx, tenantID); err != nil {
		return err
	}
	if err := d.store.InsertAuditLog(ctx, tenantID, adminID, "tenant.reactivated", map[string]any{}); err != nil {
		return err
	}
	publishOnboardingEvent(ctx, d.producer,
		"com.clario360.platform.tenant.reactivated",
		tenantID,
		&adminID,
		map[string]any{"tenant_id": tenantID.String()},
		d.logger,
	)
	return nil
}

func (d *TenantDeprovisioner) clearRedisSessions(ctx context.Context, tenantID uuid.UUID) error {
	if d.redis == nil {
		return nil
	}
	pattern := fmt.Sprintf("session:*:%s:*", tenantID.String())
	var cursor uint64
	for {
		keys, nextCursor, err := d.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := d.redis.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}
	return nil
}

func (s *sqlTenantLifecycleStore) SuspendUsers(ctx context.Context, tenantID uuid.UUID) error {
	if _, err := s.platformPool.Exec(ctx, `UPDATE users SET status = 'suspended', updated_at = now() WHERE tenant_id = $1`, tenantID); err != nil {
		return err
	}
	return nil
}

func (s *sqlTenantLifecycleStore) DeleteSessions(ctx context.Context, tenantID uuid.UUID) error {
	if _, err := s.platformPool.Exec(ctx, `DELETE FROM sessions WHERE tenant_id = $1`, tenantID); err != nil {
		return err
	}
	return nil
}

func (s *sqlTenantLifecycleStore) RevokeAPIKeys(ctx context.Context, tenantID uuid.UUID) error {
	if _, err := s.platformPool.Exec(ctx, `UPDATE api_keys SET revoked_at = now() WHERE tenant_id = $1 AND revoked_at IS NULL`, tenantID); err != nil {
		return err
	}
	return nil
}

func (s *sqlTenantLifecycleStore) SoftDeleteTenantRows(ctx context.Context, tenantID uuid.UUID) error {
	return s.updateDeletedAtAcrossPools(ctx, tenantID, "deleted_at = now()")
}

func (s *sqlTenantLifecycleStore) RestoreTenantRows(ctx context.Context, tenantID uuid.UUID) error {
	return s.updateDeletedAtAcrossPools(ctx, tenantID, "deleted_at = NULL")
}

func (s *sqlTenantLifecycleStore) MarkTenantDeprovisioned(ctx context.Context, tenantID, adminID uuid.UUID, retainUntil time.Time) error {
	if _, err := s.platformPool.Exec(ctx, `
		UPDATE tenants
		SET status = 'deprovisioned',
		    deprovisioned_at = now(),
		    deprovisioned_by = $2,
		    retain_until = $3,
		    updated_at = now()
		WHERE id = $1`,
		tenantID,
		adminID,
		retainUntil,
	); err != nil {
		return err
	}
	return nil
}

func (s *sqlTenantLifecycleStore) MarkTenantActive(ctx context.Context, tenantID uuid.UUID) error {
	if _, err := s.platformPool.Exec(ctx, `
		UPDATE users SET status = 'active', updated_at = now() WHERE tenant_id = $1 AND status = 'suspended'`,
		tenantID,
	); err != nil {
		return err
	}
	if _, err := s.platformPool.Exec(ctx, `
		UPDATE tenants
		SET status = 'active',
		    deprovisioned_at = NULL,
		    deprovisioned_by = NULL,
		    retain_until = NULL,
		    updated_at = now()
		WHERE id = $1`,
		tenantID,
	); err != nil {
		return err
	}
	return nil
}

func (s *sqlTenantLifecycleStore) InsertAuditLog(ctx context.Context, tenantID, adminID uuid.UUID, action string, metadata map[string]any) error {
	if _, err := s.platformPool.Exec(ctx, `
		INSERT INTO audit_logs (tenant_id, user_id, service, action, resource_type, resource_id, metadata)
		VALUES ($1, $2, 'iam-service', $3, 'tenant', $1, $4::jsonb)`,
		tenantID,
		adminID,
		action,
		marshalJSON(metadata),
	); err != nil {
		return err
	}
	return nil
}

func (s *sqlTenantLifecycleStore) updateDeletedAtAcrossPools(ctx context.Context, tenantID uuid.UUID, assignment string) error {
	allPools := map[string]*pgxpool.Pool{"platform_core": s.platformPool}
	for name, pool := range s.dbPools {
		allPools[name] = pool
	}
	for _, pool := range allPools {
		if pool == nil {
			continue
		}
		rows, err := pool.Query(ctx, `
			SELECT table_name
			FROM information_schema.columns
			WHERE table_schema = 'public'
			GROUP BY table_name
			HAVING bool_or(column_name = 'tenant_id') AND bool_or(column_name = 'deleted_at')`)
		if err != nil {
			return err
		}
		tables := make([]string, 0)
		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				rows.Close()
				return err
			}
			tables = append(tables, tableName)
		}
		rows.Close()
		for _, tableName := range tables {
			query := fmt.Sprintf(`UPDATE "%s" SET %s WHERE tenant_id = $1`, tableName, assignment)
			if _, err := pool.Exec(ctx, query, tenantID); err != nil {
				return fmt.Errorf("update soft delete state for %s: %w", tableName, err)
			}
		}
	}
	return nil
}

func (d *TenantDeprovisioner) tagTenantBuckets(ctx context.Context, slug string, retainUntil time.Time) error {
	if d.storage == nil {
		return nil
	}
	for _, bucket := range []string{
		"clario360-" + slug + "-cyber",
		"clario360-" + slug + "-data",
		"clario360-" + slug + "-acta",
		"clario360-" + slug + "-lex",
		"clario360-" + slug + "-visus",
		"clario360-" + slug + "-platform",
	} {
		bucketTags, err := tags.MapToBucketTags(map[string]string{
			"lifecycle":        "deprovisioned",
			"deprovision_date": time.Now().UTC().Format(time.RFC3339),
			"retain_until":     retainUntil.UTC().Format(time.RFC3339),
		})
		if err != nil {
			return err
		}
		if err := d.storage.Client().SetBucketTagging(ctx, bucket, bucketTags); err != nil {
			return fmt.Errorf("set bucket tags for %s: %w", bucket, err)
		}
	}
	return nil
}

func (d *TenantDeprovisioner) activateTenantBuckets(ctx context.Context, slug string) error {
	if d.storage == nil {
		return nil
	}
	for _, bucket := range []string{
		"clario360-" + slug + "-cyber",
		"clario360-" + slug + "-data",
		"clario360-" + slug + "-acta",
		"clario360-" + slug + "-lex",
		"clario360-" + slug + "-visus",
		"clario360-" + slug + "-platform",
	} {
		bucketTags, err := tags.MapToBucketTags(map[string]string{
			"lifecycle": "active",
		})
		if err != nil {
			return err
		}
		if err := d.storage.Client().SetBucketTagging(ctx, bucket, bucketTags); err != nil {
			return fmt.Errorf("set active bucket tags for %s: %w", bucket, err)
		}
	}
	return nil
}
