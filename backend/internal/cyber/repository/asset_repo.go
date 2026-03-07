package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/database"
)

// AssetRepository handles all asset table operations.
type AssetRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewAssetRepository creates a new AssetRepository.
func NewAssetRepository(db *pgxpool.Pool, logger zerolog.Logger) *AssetRepository {
	return &AssetRepository{db: db, logger: logger}
}

// Create inserts a new asset and returns it with its generated ID.
func (r *AssetRepository) Create(ctx context.Context, tenantID, userID uuid.UUID, req *dto.CreateAssetRequest) (*model.Asset, error) {
	now := time.Now().UTC()
	id := uuid.New()

	metadata := req.Metadata
	if metadata == nil {
		metadata = json.RawMessage("{}")
	}
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}
	discoverySource := "manual"

	_, err := r.db.Exec(ctx, `
		INSERT INTO assets (
			id, tenant_id, name, type, ip_address, hostname, mac_address,
			os, os_version, owner, department, location, criticality, status,
			discovered_at, last_seen_at, discovery_source,
			metadata, tags, created_by, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22
		)`,
		id, tenantID, req.Name, string(req.Type),
		req.IPAddress, req.Hostname, req.MACAddress,
		req.OS, req.OSVersion,
		req.Owner,
		req.Department, req.Location,
		string(req.Criticality), string(model.AssetStatusActive),
		now, now, discoverySource,
		metadata, tags,
		userID, now, now,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("insert asset: %w", err)
	}

	return r.GetByID(ctx, tenantID, id)
}

// GetByID fetches a single asset by ID with vulnerability and relationship counts.
func (r *AssetRepository) GetByID(ctx context.Context, tenantID, assetID uuid.UUID) (*model.Asset, error) {
	const q = `
		SELECT
			a.id, a.tenant_id, a.name, a.type, a.ip_address::text, a.hostname,
			a.mac_address, a.os, a.os_version, a.owner, a.department, a.location,
			a.criticality, a.status, a.discovered_at, a.last_seen_at, a.discovery_source,
			a.metadata, a.tags, a.created_by, a.created_at, a.updated_at, a.deleted_at,
			COALESCE((
				SELECT COUNT(*) FROM vulnerabilities v
				WHERE v.asset_id = a.id AND v.status NOT IN ('resolved','accepted','false_positive')
				  AND v.deleted_at IS NULL
			), 0) AS open_vulnerability_count,
			(
				SELECT v.severity FROM vulnerabilities v
				WHERE v.asset_id = a.id AND v.status NOT IN ('resolved','accepted','false_positive')
				  AND v.deleted_at IS NULL
				ORDER BY severity_order(v.severity) DESC LIMIT 1
			) AS highest_vulnerability_severity,
			COALESCE((
				SELECT COUNT(*) FROM asset_relationships r
				WHERE r.tenant_id = a.tenant_id
				  AND (r.source_asset_id = a.id OR r.target_asset_id = a.id)
			), 0) AS relationship_count
		FROM assets a
		WHERE a.tenant_id = $1 AND a.id = $2 AND a.deleted_at IS NULL`

	row := r.db.QueryRow(ctx, q, tenantID, assetID)
	return scanAsset(row)
}

// List returns a paginated, filtered list of assets.
func (r *AssetRepository) List(ctx context.Context, tenantID uuid.UUID, params *dto.AssetListParams) ([]*model.Asset, int, error) {
	const baseSelect = `
		SELECT
			a.id, a.tenant_id, a.name, a.type, a.ip_address::text, a.hostname,
			a.mac_address, a.os, a.os_version, a.owner, a.department, a.location,
			a.criticality, a.status, a.discovered_at, a.last_seen_at, a.discovery_source,
			a.metadata, a.tags, a.created_by, a.created_at, a.updated_at, a.deleted_at,
			COALESCE(vc.open_count, 0) AS open_vulnerability_count,
			vc.highest_severity AS highest_vulnerability_severity,
			COALESCE(rc.cnt, 0) AS relationship_count
		FROM assets a
		LEFT JOIN LATERAL (
			SELECT COUNT(*) AS open_count,
			       (SELECT v2.severity FROM vulnerabilities v2 WHERE v2.asset_id = a.id
			          AND v2.status NOT IN ('resolved','accepted','false_positive') AND v2.deleted_at IS NULL
			          ORDER BY severity_order(v2.severity) DESC LIMIT 1) AS highest_severity
			FROM vulnerabilities v
			WHERE v.asset_id = a.id AND v.status NOT IN ('resolved','accepted','false_positive') AND v.deleted_at IS NULL
		) vc ON true
		LEFT JOIN LATERAL (
			SELECT COUNT(*) AS cnt FROM asset_relationships r
			WHERE r.tenant_id = a.tenant_id AND (r.source_asset_id = a.id OR r.target_asset_id = a.id)
		) rc ON true`

	allowedSorts := []string{
		"name", "type", "criticality", "status",
		"discovered_at", "last_seen_at", "vulnerability_count", "created_at",
	}

	qb := database.NewQueryBuilder(baseSelect)
	r.applyAssetFilters(qb, tenantID, params)

	qb.OrderBy(params.Sort, params.Order, allowedSorts)
	qb.Paginate(params.Page, params.PerPage)

	// Count query
	countSQL, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count assets: %w", err)
	}

	sql, args := qb.Build()
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list assets: %w", err)
	}
	defer rows.Close()

	var assets []*model.Asset
	for rows.Next() {
		a, err := scanAsset(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan asset: %w", err)
		}
		assets = append(assets, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return assets, total, nil
}

func (r *AssetRepository) applyAssetFilters(qb *database.QueryBuilder, tenantID uuid.UUID, params *dto.AssetListParams) {
	qb.Where("a.tenant_id = ?", tenantID)
	qb.Where("a.deleted_at IS NULL")

	if params.Search != nil && *params.Search != "" {
		qb.WhereFTS(
			[]string{"a.name", "a.hostname", "host(a.ip_address)", "a.os", "a.department", "a.owner", "a.location"},
			*params.Search,
		)
	}
	if len(params.Types) > 0 {
		qb.WhereIn("a.type", params.Types)
	}
	if len(params.Criticalities) > 0 {
		qb.WhereIn("a.criticality", params.Criticalities)
	}
	if len(params.Statuses) > 0 {
		qb.WhereIn("a.status", params.Statuses)
	}
	if params.OS != nil {
		qb.Where("a.os ILIKE ?", "%"+*params.OS+"%")
	}
	if params.Department != nil {
		qb.Where("a.department = ?", *params.Department)
	}
	if params.Owner != nil {
		qb.Where("a.owner ILIKE ?", "%"+*params.Owner+"%")
	}
	if params.Location != nil {
		qb.Where("a.location = ?", *params.Location)
	}
	if params.DiscoverySource != nil {
		qb.Where("a.discovery_source = ?", *params.DiscoverySource)
	}
	if params.DiscoveredAfter != nil {
		qb.Where("a.discovered_at >= ?", *params.DiscoveredAfter)
	}
	if params.DiscoveredBefore != nil {
		qb.Where("a.discovered_at < ?", *params.DiscoveredBefore)
	}
	if params.LastSeenAfter != nil {
		qb.Where("a.last_seen_at >= ?", *params.LastSeenAfter)
	}
	if len(params.Tags) > 0 {
		qb.WhereArrayContainsAll("a.tags", params.Tags)
	}
	if params.HasVulnerabilities != nil {
		if *params.HasVulnerabilities {
			qb.WhereExists(`SELECT 1 FROM vulnerabilities v WHERE v.asset_id = a.id AND v.status NOT IN ('resolved','accepted','false_positive') AND v.deleted_at IS NULL`)
		} else {
			qb.Where(`NOT EXISTS (SELECT 1 FROM vulnerabilities v WHERE v.asset_id = a.id AND v.status NOT IN ('resolved','accepted','false_positive') AND v.deleted_at IS NULL)`)
		}
	}
	if params.VulnerabilitySeverity != nil {
		qb.WhereExists(
			`SELECT 1 FROM vulnerabilities v WHERE v.asset_id = a.id AND v.severity = ? AND v.status NOT IN ('resolved','accepted','false_positive') AND v.deleted_at IS NULL`,
			*params.VulnerabilitySeverity,
		)
	}
	if params.MinVulnCount != nil {
		qb.Where("COALESCE(vc.open_count, 0) >= ?", *params.MinVulnCount)
	}
}

// Update applies a partial update to an asset (only non-nil fields).
func (r *AssetRepository) Update(ctx context.Context, tenantID, assetID uuid.UUID, req *dto.UpdateAssetRequest) (*model.Asset, error) {
	setClauses := []string{"updated_at = now()"}
	args := []any{}
	argIdx := 1

	addField := func(col string, val any) {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, argIdx))
		args = append(args, val)
		argIdx++
	}

	if req.Name != nil {
		addField("name", *req.Name)
	}
	if req.Type != nil {
		addField("type", string(*req.Type))
	}
	if req.IPAddress != nil {
		addField("ip_address", *req.IPAddress)
	}
	if req.Hostname != nil {
		addField("hostname", *req.Hostname)
	}
	if req.MACAddress != nil {
		addField("mac_address", *req.MACAddress)
	}
	if req.OS != nil {
		addField("os", *req.OS)
	}
	if req.OSVersion != nil {
		addField("os_version", *req.OSVersion)
	}
	if req.Owner != nil {
		addField("owner", *req.Owner)
	}
	if req.Department != nil {
		addField("department", *req.Department)
	}
	if req.Location != nil {
		addField("location", *req.Location)
	}
	if req.Criticality != nil {
		addField("criticality", string(*req.Criticality))
	}
	if req.Status != nil {
		addField("status", string(*req.Status))
	}
	if req.Metadata != nil {
		addField("metadata", req.Metadata)
	}
	if req.Tags != nil {
		addField("tags", *req.Tags)
	}

	args = append(args, tenantID, assetID)
	whereClause := fmt.Sprintf("WHERE tenant_id = $%d AND id = $%d AND deleted_at IS NULL", argIdx, argIdx+1)

	q := fmt.Sprintf("UPDATE assets SET %s %s", strings.Join(setClauses, ", "), whereClause)
	tag, err := r.db.Exec(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("update asset: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}

	return r.GetByID(ctx, tenantID, assetID)
}

// SoftDelete marks an asset as deleted.
func (r *AssetRepository) SoftDelete(ctx context.Context, tenantID, assetID uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE assets SET deleted_at = now(), updated_at = now()
		 WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, assetID,
	)
	if err != nil {
		return fmt.Errorf("soft delete asset: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// PatchTags adds and removes tags atomically.
func (r *AssetRepository) PatchTags(ctx context.Context, tenantID, assetID uuid.UUID, req *dto.TagPatchRequest) (*model.Asset, error) {
	// Remove first, then add, then deduplicate
	tag, err := r.db.Exec(ctx, `
		UPDATE assets
		SET tags = ARRAY(
			SELECT DISTINCT unnest(
				array_cat(
					ARRAY(SELECT unnest(tags) EXCEPT SELECT unnest($3::text[])),
					$4::text[]
				)
			)
		), updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, assetID,
		req.Remove, req.Add,
	)
	if err != nil {
		return nil, fmt.Errorf("patch tags: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, tenantID, assetID)
}

// BulkInsert inserts multiple assets using pgx.CopyFrom for performance.
// Returns the list of generated UUIDs.
func (r *AssetRepository) BulkInsert(ctx context.Context, tx pgx.Tx, tenantID, userID uuid.UUID, assets []model.Asset) ([]uuid.UUID, error) {
	now := time.Now().UTC()
	ids := make([]uuid.UUID, len(assets))
	rows := make([][]any, len(assets))

	for i, a := range assets {
		id := uuid.New()
		ids[i] = id
		metadata := a.Metadata
		if metadata == nil {
			metadata = json.RawMessage("{}")
		}
		tags := a.Tags
		if tags == nil {
			tags = []string{}
		}
		rows[i] = []any{
			id, tenantID, a.Name, string(a.Type),
			a.IPAddress, a.Hostname, a.MACAddress,
			a.OS, a.OSVersion, a.Owner,
			a.Department, a.Location,
			string(a.Criticality), string(model.AssetStatusActive),
			now, now, "import",
			metadata, tags,
			userID, now, now,
		}
	}

	cols := []string{
		"id", "tenant_id", "name", "type",
		"ip_address", "hostname", "mac_address",
		"os", "os_version", "owner",
		"department", "location",
		"criticality", "status",
		"discovered_at", "last_seen_at", "discovery_source",
		"metadata", "tags",
		"created_by", "created_at", "updated_at",
	}

	_, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"assets"},
		cols,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return nil, fmt.Errorf("copy assets: %w", err)
	}
	return ids, nil
}

// UpsertFromScan inserts or updates an asset discovered during scanning (network, cloud, agent).
// Returns the asset ID and whether it was a new insert.
// The DiscoveredAsset.ExtraMetadata and MACAddress fields are merged into the stored record.
func (r *AssetRepository) UpsertFromScan(ctx context.Context, tenantID uuid.UUID, d *model.DiscoveredAsset) (uuid.UUID, bool, error) {
	var assetID uuid.UUID
	var isNew bool

	metaMap := map[string]any{
		"open_ports": d.OpenPorts,
		"banners":    d.Banners,
	}
	for k, v := range d.ExtraMetadata {
		metaMap[k] = v
	}
	meta, _ := json.Marshal(metaMap)

	discoverySource := d.DiscoverySource
	if discoverySource == "" {
		discoverySource = "network_scan"
	}

	err := r.db.QueryRow(ctx, `
		INSERT INTO assets (
			id, tenant_id, name, type,
			ip_address, hostname, mac_address, os, os_version,
			criticality, status, discovered_at, last_seen_at,
			discovery_source, metadata, tags
		) VALUES (
			gen_random_uuid(), $1, $2, $3,
			$4::inet, $5, $6::macaddr, $7, $8,
			'low', 'active', now(), now(),
			$9, $10, '{}'
		)
		ON CONFLICT (tenant_id, ip_address) WHERE ip_address IS NOT NULL AND deleted_at IS NULL
		DO UPDATE SET
			last_seen_at    = now(),
			hostname        = COALESCE(EXCLUDED.hostname, assets.hostname),
			mac_address     = COALESCE(EXCLUDED.mac_address, assets.mac_address),
			os              = COALESCE(EXCLUDED.os, assets.os),
			os_version      = COALESCE(EXCLUDED.os_version, assets.os_version),
			metadata        = assets.metadata || EXCLUDED.metadata,
			updated_at      = now()
		RETURNING id, (xmax = 0) AS is_new`,
		tenantID, d.IPAddress, string(d.AssetType),
		d.IPAddress, d.Hostname, d.MACAddress, d.OS, d.OSVersion,
		discoverySource, json.RawMessage(meta),
	).Scan(&assetID, &isNew)

	if err != nil {
		return uuid.Nil, false, fmt.Errorf("upsert from scan: %w", err)
	}
	return assetID, isNew, nil
}

// BulkUpdateCriticality batch-updates criticality for a set of asset IDs.
func (r *AssetRepository) BulkUpdateCriticality(ctx context.Context, tenantID uuid.UUID, updates map[uuid.UUID]model.Criticality) error {
	if len(updates) == 0 {
		return nil
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for id, crit := range updates {
		_, err := tx.Exec(ctx,
			`UPDATE assets SET criticality = $1, updated_at = now() WHERE tenant_id = $2 AND id = $3 AND deleted_at IS NULL`,
			string(crit), tenantID, id,
		)
		if err != nil {
			return fmt.Errorf("update criticality for %s: %w", id, err)
		}
	}
	return tx.Commit(ctx)
}

// BulkSoftDelete soft-deletes multiple assets.
func (r *AssetRepository) BulkSoftDelete(ctx context.Context, tenantID uuid.UUID, assetIDs []uuid.UUID) error {
	if len(assetIDs) == 0 {
		return nil
	}
	_, err := r.db.Exec(ctx,
		`UPDATE assets SET deleted_at = now(), updated_at = now()
		 WHERE tenant_id = $1 AND id = ANY($2) AND deleted_at IS NULL`,
		tenantID, assetIDs,
	)
	return err
}

// BulkUpdateTags adds/removes tags on a set of assets.
func (r *AssetRepository) BulkUpdateTags(ctx context.Context, tenantID uuid.UUID, assetIDs []uuid.UUID, add, remove []string) error {
	if len(assetIDs) == 0 {
		return nil
	}
	_, err := r.db.Exec(ctx, `
		UPDATE assets
		SET tags = ARRAY(
			SELECT DISTINCT unnest(
				array_cat(
					ARRAY(SELECT unnest(tags) EXCEPT SELECT unnest($3::text[])),
					$4::text[]
				)
			)
		), updated_at = now()
		WHERE tenant_id = $1 AND id = ANY($2) AND deleted_at IS NULL`,
		tenantID, assetIDs, remove, add,
	)
	return err
}

// Stats returns aggregated asset statistics for a tenant.
func (r *AssetRepository) Stats(ctx context.Context, tenantID uuid.UUID) (map[string]any, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'active') AS active,
			COUNT(*) FILTER (WHERE type = 'server') AS type_server,
			COUNT(*) FILTER (WHERE type = 'endpoint') AS type_endpoint,
			COUNT(*) FILTER (WHERE type = 'network_device') AS type_network_device,
			COUNT(*) FILTER (WHERE type = 'cloud_resource') AS type_cloud_resource,
			COUNT(*) FILTER (WHERE type = 'iot_device') AS type_iot_device,
			COUNT(*) FILTER (WHERE type = 'application') AS type_application,
			COUNT(*) FILTER (WHERE type = 'database') AS type_database,
			COUNT(*) FILTER (WHERE type = 'container') AS type_container,
			COUNT(*) FILTER (WHERE criticality = 'critical') AS crit_critical,
			COUNT(*) FILTER (WHERE criticality = 'high') AS crit_high,
			COUNT(*) FILTER (WHERE criticality = 'medium') AS crit_medium,
			COUNT(*) FILTER (WHERE criticality = 'low') AS crit_low,
			COUNT(*) FILTER (WHERE status = 'active') AS status_active,
			COUNT(*) FILTER (WHERE status = 'inactive') AS status_inactive,
			COUNT(*) FILTER (WHERE status = 'decommissioned') AS status_decommissioned,
			COUNT(*) FILTER (WHERE status = 'unknown') AS status_unknown,
			COUNT(*) FILTER (WHERE discovery_source = 'manual') AS src_manual,
			COUNT(*) FILTER (WHERE discovery_source = 'network_scan') AS src_network_scan,
			COUNT(*) FILTER (WHERE discovery_source = 'cloud_scan') AS src_cloud_scan,
			COUNT(*) FILTER (WHERE discovery_source = 'agent') AS src_agent,
			COUNT(*) FILTER (WHERE discovery_source = 'import') AS src_import
		FROM assets
		WHERE tenant_id = $1 AND deleted_at IS NULL`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return map[string]any{}, nil
	}

	var (
		total, active                                             int
		typeServer, typeEndpoint, typeNetwork, typeCloud, typeIoT int
		typeApp, typeDB, typeContainer                            int
		critCritical, critHigh, critMedium, critLow               int
		statusActive, statusInactive, statusDecom, statusUnknown  int
		srcManual, srcNetScan, srcCloudScan, srcAgent, srcImport  int
	)

	if err := rows.Scan(
		&total, &active,
		&typeServer, &typeEndpoint, &typeNetwork, &typeCloud, &typeIoT,
		&typeApp, &typeDB, &typeContainer,
		&critCritical, &critHigh, &critMedium, &critLow,
		&statusActive, &statusInactive, &statusDecom, &statusUnknown,
		&srcManual, &srcNetScan, &srcCloudScan, &srcAgent, &srcImport,
	); err != nil {
		return nil, fmt.Errorf("scan stats: %w", err)
	}

	result := map[string]any{
		"total_assets":  total,
		"active_assets": active,
		"by_type": map[string]int{
			"server": typeServer, "endpoint": typeEndpoint,
			"network_device": typeNetwork, "cloud_resource": typeCloud,
			"iot_device": typeIoT, "application": typeApp,
			"database": typeDB, "container": typeContainer,
		},
		"by_criticality": map[string]int{
			"critical": critCritical, "high": critHigh,
			"medium": critMedium, "low": critLow,
		},
		"by_status": map[string]int{
			"active": statusActive, "inactive": statusInactive,
			"decommissioned": statusDecom, "unknown": statusUnknown,
		},
		"by_discovery_source": map[string]int{
			"manual": srcManual, "network_scan": srcNetScan,
			"cloud_scan": srcCloudScan, "agent": srcAgent, "import": srcImport,
		},
	}

	byDepartment, topDepartments, err := r.topCounts(ctx, tenantID, "department")
	if err != nil {
		return nil, err
	}
	byOS, topOS, err := r.topCounts(ctx, tenantID, "os")
	if err != nil {
		return nil, err
	}
	result["by_department"] = byDepartment
	result["top_departments"] = topDepartments
	result["by_os"] = byOS
	result["top_os"] = topOS

	return result, nil
}

func (r *AssetRepository) topCounts(ctx context.Context, tenantID uuid.UUID, column string) (map[string]int, []model.AssetCountByName, error) {
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT %s AS name, COUNT(*) AS count
		FROM assets
		WHERE tenant_id = $1 AND deleted_at IS NULL AND COALESCE(%s, '') <> ''
		GROUP BY %s
		ORDER BY COUNT(*) DESC, %s ASC
		LIMIT 10`, column, column, column, column), tenantID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	byName := make(map[string]int)
	top := make([]model.AssetCountByName, 0, 10)
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			return nil, nil, err
		}
		byName[name] = count
		top = append(top, model.AssetCountByName{Name: name, Count: count})
	}
	return byName, top, rows.Err()
}

// CountByParams returns a simple asset count using the same filters as List.
func (r *AssetRepository) CountByParams(ctx context.Context, tenantID uuid.UUID, params *dto.AssetListParams) (int, error) {
	qb := database.NewQueryBuilder(`
		SELECT a.id
		FROM assets a
		LEFT JOIN LATERAL (
			SELECT COUNT(*) AS open_count
			FROM vulnerabilities v
			WHERE v.asset_id = a.id AND v.status NOT IN ('resolved','accepted','false_positive') AND v.deleted_at IS NULL
		) vc ON true`)
	r.applyAssetFilters(qb, tenantID, params)
	sql, args := qb.BuildCount()
	var count int
	if err := r.db.QueryRow(ctx, sql, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// GetMany returns multiple assets by IDs (same tenant).
func (r *AssetRepository) GetMany(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) ([]*model.Asset, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT
			a.id, a.tenant_id, a.name, a.type, a.ip_address::text, a.hostname,
			a.mac_address, a.os, a.os_version, a.owner, a.department, a.location,
			a.criticality, a.status, a.discovered_at, a.last_seen_at, a.discovery_source,
			a.metadata, a.tags, a.created_by, a.created_at, a.updated_at, a.deleted_at,
			0::bigint AS open_vulnerability_count,
			NULL::text AS highest_vulnerability_severity,
			0::bigint AS relationship_count
		FROM assets a
		WHERE a.tenant_id = $1 AND a.id = ANY($2) AND a.deleted_at IS NULL`,
		tenantID, ids,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var assets []*model.Asset
	for rows.Next() {
		a, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		assets = append(assets, a)
	}
	return assets, rows.Err()
}

// scanAsset reads one asset row (including computed columns) from any pgx row source.
func scanAsset(row interface {
	Scan(dest ...any) error
}) (*model.Asset, error) {
	var a model.Asset
	var ipStr *string

	err := row.Scan(
		&a.ID, &a.TenantID, &a.Name, &a.Type,
		&ipStr, &a.Hostname, &a.MACAddress,
		&a.OS, &a.OSVersion, &a.Owner,
		&a.Department, &a.Location,
		&a.Criticality, &a.Status,
		&a.DiscoveredAt, &a.LastSeenAt, &a.DiscoverySource,
		&a.Metadata, &a.Tags,
		&a.CreatedBy, &a.CreatedAt, &a.UpdatedAt, &a.DeletedAt,
		&a.OpenVulnerabilityCount, &a.HighestVulnSeverity, &a.RelationshipCount,
	)
	if err != nil {
		return nil, err
	}
	a.IPAddress = ipStr
	if a.Metadata == nil {
		a.Metadata = json.RawMessage("{}")
	}
	if a.Tags == nil {
		a.Tags = []string{}
	}
	return &a, nil
}
