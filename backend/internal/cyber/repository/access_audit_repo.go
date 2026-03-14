package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/access/dto"
	"github.com/clario360/platform/internal/cyber/dspm/access/model"
)

// AccessAuditRepository handles dspm_access_audit table operations.
type AccessAuditRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewAccessAuditRepository creates a new access audit repository.
func NewAccessAuditRepository(db *pgxpool.Pool, logger zerolog.Logger) *AccessAuditRepository {
	return &AccessAuditRepository{db: db, logger: logger}
}

// Insert writes a single audit entry.
func (r *AccessAuditRepository) Insert(ctx context.Context, entry *model.AccessAuditEntry) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO dspm_access_audit (
			id, tenant_id, identity_type, identity_id, data_asset_id,
			action, source_ip, query_hash, rows_affected, duration_ms, success,
			access_mapping_id, table_name, database_name,
			event_timestamp, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
	`, entry.ID, entry.TenantID, entry.IdentityType, entry.IdentityID, entry.DataAssetID,
		entry.Action, entry.SourceIP, entry.QueryHash, entry.RowsAffected, entry.DurationMs, entry.Success,
		entry.AccessMappingID, entry.TableName, entry.DatabaseName,
		entry.EventTimestamp, entry.CreatedAt,
	)
	return err
}

// ListByAsset returns paginated audit entries for a data asset.
func (r *AccessAuditRepository) ListByAsset(ctx context.Context, tenantID, assetID uuid.UUID, params *dto.AuditListParams) ([]model.AccessAuditEntry, int, error) {
	params.SetDefaults()

	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM dspm_access_audit
		WHERE tenant_id = $1 AND data_asset_id = $2
	`, tenantID, assetID).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (params.Page - 1) * params.PerPage
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT %s FROM dspm_access_audit
		WHERE tenant_id = $1 AND data_asset_id = $2
		ORDER BY created_at DESC
		LIMIT %d OFFSET %d
	`, auditColumns(), params.PerPage, offset), tenantID, assetID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	entries, err := scanAuditEntries(rows)
	return entries, total, err
}

// ListByIdentity returns paginated audit entries for an identity.
func (r *AccessAuditRepository) ListByIdentity(ctx context.Context, tenantID uuid.UUID, identityID string, params *dto.AuditListParams) ([]model.AccessAuditEntry, int, error) {
	params.SetDefaults()

	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM dspm_access_audit
		WHERE tenant_id = $1 AND identity_id = $2
	`, tenantID, identityID).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (params.Page - 1) * params.PerPage
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT %s FROM dspm_access_audit
		WHERE tenant_id = $1 AND identity_id = $2
		ORDER BY created_at DESC
		LIMIT %d OFFSET %d
	`, auditColumns(), params.PerPage, offset), tenantID, identityID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	entries, err := scanAuditEntries(rows)
	return entries, total, err
}

// CountAccessLast24h returns how many access events an identity had in the last 24 hours.
func (r *AccessAuditRepository) CountAccessLast24h(ctx context.Context, tenantID uuid.UUID, identityID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM dspm_access_audit
		WHERE tenant_id = $1 AND identity_id = $2
		  AND created_at >= now() - interval '24 hours'
	`, tenantID, identityID).Scan(&count)
	return count, err
}

// NewRestrictedAssetsLast24h returns names of restricted assets accessed in the last 24h
// that weren't accessed in the previous 30 days by this identity.
func (r *AccessAuditRepository) NewRestrictedAssetsLast24h(ctx context.Context, tenantID uuid.UUID, identityID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		WITH recent AS (
			SELECT DISTINCT a.data_asset_id
			FROM dspm_access_audit a
			JOIN dspm_access_mappings m ON m.data_asset_id = a.data_asset_id AND m.tenant_id = a.tenant_id
			WHERE a.tenant_id = $1 AND a.identity_id = $2
			  AND a.created_at >= now() - interval '24 hours'
			  AND m.data_classification = 'restricted'
		),
		historical AS (
			SELECT DISTINCT data_asset_id
			FROM dspm_access_audit
			WHERE tenant_id = $1 AND identity_id = $2
			  AND created_at < now() - interval '24 hours'
			  AND created_at >= now() - interval '30 days'
		)
		SELECT COALESCE(m.data_asset_name, r.data_asset_id::text)
		FROM recent r
		LEFT JOIN dspm_access_mappings m ON m.data_asset_id = r.data_asset_id AND m.tenant_id = $1
		WHERE r.data_asset_id NOT IN (SELECT data_asset_id FROM historical)
		LIMIT 20
	`, tenantID, identityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// NewSourceIPsLast24h returns source IPs used in the last 24h that weren't used in the previous 30 days.
func (r *AccessAuditRepository) NewSourceIPsLast24h(ctx context.Context, tenantID uuid.UUID, identityID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		WITH recent_ips AS (
			SELECT DISTINCT source_ip FROM dspm_access_audit
			WHERE tenant_id = $1 AND identity_id = $2
			  AND created_at >= now() - interval '24 hours'
			  AND source_ip IS NOT NULL AND source_ip != ''
		),
		historical_ips AS (
			SELECT DISTINCT source_ip FROM dspm_access_audit
			WHERE tenant_id = $1 AND identity_id = $2
			  AND created_at < now() - interval '24 hours'
			  AND created_at >= now() - interval '30 days'
			  AND source_ip IS NOT NULL AND source_ip != ''
		)
		SELECT source_ip FROM recent_ips
		WHERE source_ip NOT IN (SELECT source_ip FROM historical_ips)
		LIMIT 10
	`, tenantID, identityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ips []string
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			continue
		}
		ips = append(ips, ip)
	}
	return ips, rows.Err()
}

// UsageCounts returns 30d and 90d usage counts and last_used_at for all mappings.
func (r *AccessAuditRepository) UsageCounts(ctx context.Context, tenantID uuid.UUID) (map[string]UsageInfo, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			identity_type, identity_id, data_asset_id,
			COUNT(*) FILTER (WHERE created_at >= now() - interval '30 days') AS count_30d,
			COUNT(*) FILTER (WHERE created_at >= now() - interval '90 days') AS count_90d,
			MAX(created_at) AS last_used
		FROM dspm_access_audit
		WHERE tenant_id = $1
		GROUP BY identity_type, identity_id, data_asset_id
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]UsageInfo)
	for rows.Next() {
		var identityType, identityID string
		var assetID uuid.UUID
		var info UsageInfo
		if err := rows.Scan(&identityType, &identityID, &assetID, &info.Count30d, &info.Count90d, &info.LastUsed); err != nil {
			continue
		}
		key := identityType + "|" + identityID + "|" + assetID.String()
		result[key] = info
	}
	return result, rows.Err()
}

// UsageInfo holds usage counts for a mapping.
type UsageInfo struct {
	Count30d int
	Count90d int
	LastUsed *time.Time
}

// ── Scan helpers ────────────────────────────────────────────────────────────

func auditColumns() string {
	return `id, tenant_id, identity_type, identity_id, data_asset_id,
		action, source_ip, query_hash, rows_affected, duration_ms, success,
		access_mapping_id, table_name, database_name,
		event_timestamp, created_at`
}

func scanAuditEntries(rows pgx.Rows) ([]model.AccessAuditEntry, error) {
	var results []model.AccessAuditEntry
	for rows.Next() {
		var e model.AccessAuditEntry
		err := rows.Scan(
			&e.ID, &e.TenantID, &e.IdentityType, &e.IdentityID, &e.DataAssetID,
			&e.Action, &e.SourceIP, &e.QueryHash, &e.RowsAffected, &e.DurationMs, &e.Success,
			&e.AccessMappingID, &e.TableName, &e.DatabaseName,
			&e.EventTimestamp, &e.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, e)
	}
	return results, rows.Err()
}
