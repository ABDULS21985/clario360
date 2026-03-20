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

	"github.com/clario360/platform/internal/cyber/dspm/access/dto"
	"github.com/clario360/platform/internal/cyber/dspm/access/model"
)

// AccessMappingRepository handles dspm_access_mappings and dspm_identity_profiles CRUD.
type AccessMappingRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewAccessMappingRepository creates a new access mapping repository.
func NewAccessMappingRepository(db *pgxpool.Pool, logger zerolog.Logger) *AccessMappingRepository {
	return &AccessMappingRepository{db: db, logger: logger}
}

// UpsertMapping inserts or updates an access mapping using the natural unique key.
func (r *AccessMappingRepository) UpsertMapping(ctx context.Context, mapping *model.AccessMapping) error {
	if mapping.ID == uuid.Nil {
		mapping.ID = uuid.New()
	}
	now := time.Now().UTC()
	mapping.UpdatedAt = now

	_, err := r.db.Exec(ctx, `
		INSERT INTO dspm_access_mappings (
			id, tenant_id, identity_type, identity_id, identity_name, identity_source,
			data_asset_id, data_asset_name, data_classification,
			permission_type, permission_source, permission_path, is_wildcard,
			last_used_at, usage_count_30d, usage_count_90d, is_stale,
			sensitivity_weight, access_risk_score,
			status, expires_at, discovered_at, last_verified_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9,
			$10, $11, $12, $13,
			$14, $15, $16, $17,
			$18, $19,
			$20, $21, $22, $23, $24, $25
		)
		ON CONFLICT (tenant_id, identity_type, identity_id, data_asset_id, permission_type) DO UPDATE SET
			identity_name      = EXCLUDED.identity_name,
			identity_source    = EXCLUDED.identity_source,
			data_asset_name    = EXCLUDED.data_asset_name,
			data_classification = EXCLUDED.data_classification,
			permission_source  = EXCLUDED.permission_source,
			permission_path    = EXCLUDED.permission_path,
			is_wildcard        = EXCLUDED.is_wildcard,
			sensitivity_weight = EXCLUDED.sensitivity_weight,
			access_risk_score  = EXCLUDED.access_risk_score,
			status             = 'active',
			expires_at         = COALESCE(EXCLUDED.expires_at, dspm_access_mappings.expires_at),
			last_verified_at   = EXCLUDED.last_verified_at,
			updated_at         = EXCLUDED.updated_at
	`, mapping.ID, mapping.TenantID, mapping.IdentityType, mapping.IdentityID, mapping.IdentityName, mapping.IdentitySource,
		mapping.DataAssetID, mapping.DataAssetName, mapping.DataClassification,
		mapping.PermissionType, mapping.PermissionSource, mapping.PermissionPath, mapping.IsWildcard,
		mapping.LastUsedAt, mapping.UsageCount30d, mapping.UsageCount90d, mapping.IsStale,
		mapping.SensitivityWeight, mapping.AccessRiskScore,
		mapping.Status, mapping.ExpiresAt, mapping.DiscoveredAt, mapping.LastVerifiedAt, mapping.CreatedAt, mapping.UpdatedAt,
	)
	return err
}

// MarkUnseen sets status='revoked' for active mappings not verified since the given time.
func (r *AccessMappingRepository) MarkUnseen(ctx context.Context, tenantID uuid.UUID, verifiedBefore time.Time) (int, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE dspm_access_mappings
		SET status = 'revoked', updated_at = now()
		WHERE tenant_id = $1
		  AND status = 'active'
		  AND last_verified_at < $2
	`, tenantID, verifiedBefore)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// ListActiveByTenant returns all active access mappings for a tenant.
func (r *AccessMappingRepository) ListActiveByTenant(ctx context.Context, tenantID uuid.UUID) ([]*model.AccessMapping, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+mappingColumns()+`
		FROM dspm_access_mappings
		WHERE tenant_id = $1 AND status = 'active'
		ORDER BY access_risk_score DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMappings(rows)
}

// ListMappings returns paginated, filtered access mappings.
func (r *AccessMappingRepository) ListMappings(ctx context.Context, tenantID uuid.UUID, params *dto.AccessMappingListParams) ([]*model.AccessMapping, int, error) {
	params.SetDefaults()
	if err := params.Validate(); err != nil {
		return nil, 0, err
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if len(params.IdentityType) > 0 {
		placeholders := make([]string, len(params.IdentityType))
		for i, v := range params.IdentityType {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, v)
			argIdx++
		}
		conditions = append(conditions, "identity_type IN ("+strings.Join(placeholders, ", ")+")")
	}
	if params.IdentityID != nil {
		conditions = append(conditions, fmt.Sprintf("identity_id = $%d", argIdx))
		args = append(args, *params.IdentityID)
		argIdx++
	}
	if params.DataAssetID != nil {
		conditions = append(conditions, fmt.Sprintf("data_asset_id = $%d", argIdx))
		args = append(args, *params.DataAssetID)
		argIdx++
	}
	if len(params.PermissionType) > 0 {
		placeholders := make([]string, len(params.PermissionType))
		for i, v := range params.PermissionType {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, v)
			argIdx++
		}
		conditions = append(conditions, "permission_type IN ("+strings.Join(placeholders, ", ")+")")
	}
	if len(params.DataClassification) > 0 {
		placeholders := make([]string, len(params.DataClassification))
		for i, v := range params.DataClassification {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, v)
			argIdx++
		}
		conditions = append(conditions, "data_classification IN ("+strings.Join(placeholders, ", ")+")")
	}
	if len(params.Status) > 0 {
		placeholders := make([]string, len(params.Status))
		for i, v := range params.Status {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, v)
			argIdx++
		}
		conditions = append(conditions, "status IN ("+strings.Join(placeholders, ", ")+")")
	}
	if params.IsStale != nil {
		conditions = append(conditions, fmt.Sprintf("is_stale = $%d", argIdx))
		args = append(args, *params.IsStale)
		argIdx++
	}
	if params.Search != nil {
		conditions = append(conditions, fmt.Sprintf("(identity_name ILIKE $%d OR data_asset_name ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+*params.Search+"%")
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	// Count.
	var total int
	countQuery := "SELECT COUNT(*) FROM dspm_access_mappings " + where
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Sort.
	sortCol := "access_risk_score"
	validSorts := map[string]string{
		"access_risk_score":  "access_risk_score",
		"last_used_at":       "last_used_at",
		"sensitivity_weight": "sensitivity_weight",
		"created_at":         "created_at",
		"updated_at":         "updated_at",
	}
	if s, ok := validSorts[params.Sort]; ok {
		sortCol = s
	}
	order := "DESC"
	if strings.ToLower(params.Order) == "asc" {
		order = "ASC"
	}

	offset := (params.Page - 1) * params.PerPage
	query := fmt.Sprintf(`SELECT %s FROM dspm_access_mappings %s ORDER BY %s %s NULLS LAST LIMIT %d OFFSET %d`,
		mappingColumns(), where, sortCol, order, params.PerPage, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items, err := scanMappings(rows)
	return items, total, err
}

// ListByIdentity returns all active mappings for a specific identity.
func (r *AccessMappingRepository) ListByIdentity(ctx context.Context, tenantID uuid.UUID, identityID string) ([]*model.AccessMapping, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+mappingColumns()+`
		FROM dspm_access_mappings
		WHERE tenant_id = $1 AND identity_id = $2 AND status = 'active'
		ORDER BY access_risk_score DESC
	`, tenantID, identityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMappings(rows)
}

// ListByAsset returns all active mappings for a specific data asset.
func (r *AccessMappingRepository) ListByAsset(ctx context.Context, tenantID uuid.UUID, assetID uuid.UUID) ([]*model.AccessMapping, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+mappingColumns()+`
		FROM dspm_access_mappings
		WHERE tenant_id = $1 AND data_asset_id = $2 AND status = 'active'
		ORDER BY access_risk_score DESC
	`, tenantID, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMappings(rows)
}

// UpdateStatus sets the status on a specific mapping.
func (r *AccessMappingRepository) UpdateStatus(ctx context.Context, mappingID uuid.UUID, status string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE dspm_access_mappings SET status = $1, updated_at = now() WHERE id = $2
	`, status, mappingID)
	return err
}

// MarkStale sets is_stale=true for mappings unused beyond thresholdDays.
func (r *AccessMappingRepository) MarkStale(ctx context.Context, tenantID uuid.UUID, thresholdDays int) (int, error) {
	cutoff := time.Now().UTC().Add(-time.Duration(thresholdDays) * 24 * time.Hour)
	tag, err := r.db.Exec(ctx, `
		UPDATE dspm_access_mappings
		SET is_stale = true, updated_at = now()
		WHERE tenant_id = $1
		  AND status = 'active'
		  AND is_stale = false
		  AND (last_used_at IS NULL OR last_used_at < $2)
	`, tenantID, cutoff)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// ExpireTimeBoundGrants marks active mappings with expires_at <= now as expired.
func (r *AccessMappingRepository) ExpireTimeBoundGrants(ctx context.Context, tenantID uuid.UUID, now time.Time) (int, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE dspm_access_mappings
		SET status = 'expired', updated_at = now()
		WHERE tenant_id = $1
		  AND status = 'active'
		  AND expires_at IS NOT NULL
		  AND expires_at <= $2
	`, tenantID, now)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// AllAssetWeights returns sensitivity weights for all active data assets in a tenant.
func (r *AccessMappingRepository) AllAssetWeights(ctx context.Context, tenantID uuid.UUID) ([]float64, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT data_classification
		FROM dspm_access_mappings
		WHERE tenant_id = $1 AND status = 'active'
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var weights []float64
	for rows.Next() {
		var classification string
		if err := rows.Scan(&classification); err != nil {
			continue
		}
		weights = append(weights, model.SensitivityWeight(classification))
	}
	return weights, rows.Err()
}

// CountOverprivileged returns the count of overprivileged mappings (usage_count_90d=0).
func (r *AccessMappingRepository) CountOverprivileged(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM dspm_access_mappings
		WHERE tenant_id = $1 AND status = 'active' AND usage_count_90d = 0
	`, tenantID).Scan(&count)
	return count, err
}

// CountStale returns the count of stale mappings.
func (r *AccessMappingRepository) CountStale(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM dspm_access_mappings
		WHERE tenant_id = $1 AND status = 'active' AND is_stale = true
	`, tenantID).Scan(&count)
	return count, err
}

// CountActive returns the count of active mappings.
func (r *AccessMappingRepository) CountActive(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM dspm_access_mappings
		WHERE tenant_id = $1 AND status = 'active'
	`, tenantID).Scan(&count)
	return count, err
}

// CountTotal returns the total count of all mappings.
func (r *AccessMappingRepository) CountTotal(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM dspm_access_mappings WHERE tenant_id = $1
	`, tenantID).Scan(&count)
	return count, err
}

// ClassificationAccessBreakdown returns counts of active mappings per classification.
func (r *AccessMappingRepository) ClassificationAccessBreakdown(ctx context.Context, tenantID uuid.UUID) (map[string]int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT COALESCE(data_classification, 'unknown'), COUNT(*)
		FROM dspm_access_mappings
		WHERE tenant_id = $1 AND status = 'active'
		GROUP BY data_classification
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var class string
		var count int
		if err := rows.Scan(&class, &count); err != nil {
			continue
		}
		result[class] = count
	}
	return result, rows.Err()
}

// ── Identity Profile Methods ─────────────────────────────────────────────────

// UpsertIdentityProfile inserts or updates an identity profile.
func (r *AccessMappingRepository) UpsertIdentityProfile(ctx context.Context, profile *model.IdentityProfile) error {
	if profile.ID == uuid.Nil {
		profile.ID = uuid.New()
	}
	now := time.Now().UTC()
	profile.UpdatedAt = now

	riskFactorsJSON, _ := json.Marshal(profile.RiskFactors)
	patternJSON, _ := json.Marshal(profile.AccessPatternSummary)
	recsJSON, _ := json.Marshal(profile.Recommendations)

	_, err := r.db.Exec(ctx, `
		INSERT INTO dspm_identity_profiles (
			id, tenant_id, identity_type, identity_id, identity_name, identity_email, identity_source,
			total_assets_accessible, sensitive_assets_count, permission_count,
			overprivileged_count, stale_permission_count,
			blast_radius_score, blast_radius_level,
			access_risk_score, access_risk_level, risk_factors,
			last_activity_at, avg_daily_access_count, access_pattern_summary,
			recommendations, status, last_review_at, next_review_due,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10,
			$11, $12,
			$13, $14,
			$15, $16, $17,
			$18, $19, $20,
			$21, $22, $23, $24,
			$25, $26
		)
		ON CONFLICT (tenant_id, identity_type, identity_id) DO UPDATE SET
			identity_name          = EXCLUDED.identity_name,
			identity_email         = EXCLUDED.identity_email,
			total_assets_accessible = EXCLUDED.total_assets_accessible,
			sensitive_assets_count = EXCLUDED.sensitive_assets_count,
			permission_count       = EXCLUDED.permission_count,
			overprivileged_count   = EXCLUDED.overprivileged_count,
			stale_permission_count = EXCLUDED.stale_permission_count,
			blast_radius_score     = EXCLUDED.blast_radius_score,
			blast_radius_level     = EXCLUDED.blast_radius_level,
			access_risk_score      = EXCLUDED.access_risk_score,
			access_risk_level      = EXCLUDED.access_risk_level,
			risk_factors           = EXCLUDED.risk_factors,
			last_activity_at       = EXCLUDED.last_activity_at,
			avg_daily_access_count = EXCLUDED.avg_daily_access_count,
			access_pattern_summary = EXCLUDED.access_pattern_summary,
			recommendations        = EXCLUDED.recommendations,
			status                 = EXCLUDED.status,
			last_review_at         = EXCLUDED.last_review_at,
			next_review_due        = EXCLUDED.next_review_due,
			updated_at             = EXCLUDED.updated_at
	`, profile.ID, profile.TenantID, profile.IdentityType, profile.IdentityID, profile.IdentityName, profile.IdentityEmail, profile.IdentitySource,
		profile.TotalAssetsAccessible, profile.SensitiveAssetsCount, profile.PermissionCount,
		profile.OverprivilegedCount, profile.StalePermissionCount,
		profile.BlastRadiusScore, profile.BlastRadiusLevel,
		profile.AccessRiskScore, profile.AccessRiskLevel, riskFactorsJSON,
		profile.LastActivityAt, profile.AvgDailyAccessCount, patternJSON,
		recsJSON, profile.Status, profile.LastReviewAt, profile.NextReviewDue,
		profile.CreatedAt, profile.UpdatedAt,
	)
	return err
}

// ListIdentityProfiles returns paginated, filtered identity profiles.
func (r *AccessMappingRepository) ListIdentityProfiles(ctx context.Context, tenantID uuid.UUID, params *dto.IdentityListParams) ([]*model.IdentityProfile, int, error) {
	params.SetDefaults()
	if err := params.Validate(); err != nil {
		return nil, 0, err
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if len(params.IdentityType) > 0 {
		placeholders := make([]string, len(params.IdentityType))
		for i, v := range params.IdentityType {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, v)
			argIdx++
		}
		conditions = append(conditions, "identity_type IN ("+strings.Join(placeholders, ", ")+")")
	}
	if len(params.Status) > 0 {
		placeholders := make([]string, len(params.Status))
		for i, v := range params.Status {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, v)
			argIdx++
		}
		conditions = append(conditions, "status IN ("+strings.Join(placeholders, ", ")+")")
	}
	if params.MinRiskScore != nil {
		conditions = append(conditions, fmt.Sprintf("access_risk_score >= $%d", argIdx))
		args = append(args, *params.MinRiskScore)
		argIdx++
	}
	if params.Search != nil {
		conditions = append(conditions, fmt.Sprintf("(identity_name ILIKE $%d OR identity_email ILIKE $%d OR identity_id ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+*params.Search+"%")
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM dspm_identity_profiles "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	sortCol := "access_risk_score"
	validSorts := map[string]string{
		"access_risk_score":      "access_risk_score",
		"blast_radius_score":     "blast_radius_score",
		"overprivileged_count":   "overprivileged_count",
		"stale_permission_count": "stale_permission_count",
		"last_activity_at":       "last_activity_at",
		"created_at":             "created_at",
	}
	if s, ok := validSorts[params.Sort]; ok {
		sortCol = s
	}
	order := "DESC"
	if strings.ToLower(params.Order) == "asc" {
		order = "ASC"
	}

	offset := (params.Page - 1) * params.PerPage
	query := fmt.Sprintf(`SELECT %s FROM dspm_identity_profiles %s ORDER BY %s %s NULLS LAST LIMIT %d OFFSET %d`,
		profileColumns(), where, sortCol, order, params.PerPage, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items, err := scanProfiles(rows)
	return items, total, err
}

// GetIdentityProfile returns a single identity profile.
func (r *AccessMappingRepository) GetIdentityProfile(ctx context.Context, tenantID uuid.UUID, identityID string) (*model.IdentityProfile, error) {
	row := r.db.QueryRow(ctx, `
		SELECT `+profileColumns()+`
		FROM dspm_identity_profiles
		WHERE tenant_id = $1 AND identity_id = $2
	`, tenantID, identityID)

	profile, err := scanProfile(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return profile, nil
}

// GetByIdentityID returns a single identity profile by identity_id (alias for GetIdentityProfile).
func (r *AccessMappingRepository) GetByIdentityID(ctx context.Context, tenantID uuid.UUID, identityID string) (*model.IdentityProfile, error) {
	return r.GetIdentityProfile(ctx, tenantID, identityID)
}

// ListActive returns all active identity profiles for a tenant.
func (r *AccessMappingRepository) ListActive(ctx context.Context, tenantID uuid.UUID) ([]*model.IdentityProfile, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+profileColumns()+`
		FROM dspm_identity_profiles
		WHERE tenant_id = $1 AND status = 'active'
		ORDER BY access_risk_score DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProfiles(rows)
}

// CountIdentities returns the total active identity profile count.
func (r *AccessMappingRepository) CountIdentities(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM dspm_identity_profiles WHERE tenant_id = $1 AND status = 'active'
	`, tenantID).Scan(&count)
	return count, err
}

// CountHighRiskIdentities returns count of identities with access_risk_score >= 50.
func (r *AccessMappingRepository) CountHighRiskIdentities(ctx context.Context, tenantID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM dspm_identity_profiles
		WHERE tenant_id = $1 AND status = 'active' AND access_risk_score >= 50
	`, tenantID).Scan(&count)
	return count, err
}

// AvgBlastRadius returns the average blast radius score across all active profiles.
func (r *AccessMappingRepository) AvgBlastRadius(ctx context.Context, tenantID uuid.UUID) (float64, error) {
	var avg float64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(blast_radius_score), 0) FROM dspm_identity_profiles
		WHERE tenant_id = $1 AND status = 'active'
	`, tenantID).Scan(&avg)
	return avg, err
}

// TopRiskyIdentities returns top N identity profiles by risk score.
func (r *AccessMappingRepository) TopRiskyIdentities(ctx context.Context, tenantID uuid.UUID, limit int) ([]*model.IdentityProfile, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT %s FROM dspm_identity_profiles
		WHERE tenant_id = $1 AND status = 'active'
		ORDER BY access_risk_score DESC
		LIMIT %d
	`, profileColumns(), limit), tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanProfiles(rows)
}

// RiskDistribution returns count of identity profiles by risk level.
func (r *AccessMappingRepository) RiskDistribution(ctx context.Context, tenantID uuid.UUID) (map[string]int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT access_risk_level, COUNT(*)
		FROM dspm_identity_profiles
		WHERE tenant_id = $1 AND status = 'active'
		GROUP BY access_risk_level
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var level string
		var count int
		if err := rows.Scan(&level, &count); err != nil {
			continue
		}
		result[level] = count
	}
	return result, rows.Err()
}

// ── Scan helpers ─────────────────────────────────────────────────────────────

func mappingColumns() string {
	return `id, tenant_id, identity_type, identity_id, identity_name, identity_source,
		data_asset_id, data_asset_name, data_classification,
		permission_type, permission_source, permission_path, is_wildcard,
		last_used_at, usage_count_30d, usage_count_90d, is_stale,
		sensitivity_weight, access_risk_score,
		status, expires_at, discovered_at, last_verified_at, created_at, updated_at`
}

func scanMappings(rows pgx.Rows) ([]*model.AccessMapping, error) {
	var results []*model.AccessMapping
	for rows.Next() {
		m, err := scanMappingRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

func scanMappingRow(row pgx.Row) (*model.AccessMapping, error) {
	m := &model.AccessMapping{}
	err := row.Scan(
		&m.ID, &m.TenantID, &m.IdentityType, &m.IdentityID, &m.IdentityName, &m.IdentitySource,
		&m.DataAssetID, &m.DataAssetName, &m.DataClassification,
		&m.PermissionType, &m.PermissionSource, &m.PermissionPath, &m.IsWildcard,
		&m.LastUsedAt, &m.UsageCount30d, &m.UsageCount90d, &m.IsStale,
		&m.SensitivityWeight, &m.AccessRiskScore,
		&m.Status, &m.ExpiresAt, &m.DiscoveredAt, &m.LastVerifiedAt, &m.CreatedAt, &m.UpdatedAt,
	)
	return m, err
}

func profileColumns() string {
	return `id, tenant_id, identity_type, identity_id, identity_name, identity_email, identity_source,
		total_assets_accessible, sensitive_assets_count, permission_count,
		overprivileged_count, stale_permission_count,
		blast_radius_score, blast_radius_level,
		access_risk_score, access_risk_level, risk_factors,
		last_activity_at, avg_daily_access_count, access_pattern_summary,
		recommendations, status, last_review_at, next_review_due,
		created_at, updated_at`
}

func scanProfiles(rows pgx.Rows) ([]*model.IdentityProfile, error) {
	var results []*model.IdentityProfile
	for rows.Next() {
		p, err := scanProfileRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, rows.Err()
}

func scanProfile(row pgx.Row) (*model.IdentityProfile, error) {
	return scanProfileRow(row)
}

func scanProfileRow(row pgx.Row) (*model.IdentityProfile, error) {
	p := &model.IdentityProfile{}
	var riskFactorsJSON, patternJSON, recsJSON []byte

	err := row.Scan(
		&p.ID, &p.TenantID, &p.IdentityType, &p.IdentityID, &p.IdentityName, &p.IdentityEmail, &p.IdentitySource,
		&p.TotalAssetsAccessible, &p.SensitiveAssetsCount, &p.PermissionCount,
		&p.OverprivilegedCount, &p.StalePermissionCount,
		&p.BlastRadiusScore, &p.BlastRadiusLevel,
		&p.AccessRiskScore, &p.AccessRiskLevel, &riskFactorsJSON,
		&p.LastActivityAt, &p.AvgDailyAccessCount, &patternJSON,
		&recsJSON, &p.Status, &p.LastReviewAt, &p.NextReviewDue,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(riskFactorsJSON) > 0 {
		_ = json.Unmarshal(riskFactorsJSON, &p.RiskFactors)
	}
	if len(patternJSON) > 0 {
		_ = json.Unmarshal(patternJSON, &p.AccessPatternSummary)
	}
	if len(recsJSON) > 0 {
		_ = json.Unmarshal(recsJSON, &p.Recommendations)
	}

	if p.RiskFactors == nil {
		p.RiskFactors = []model.IdentityRiskFactor{}
	}
	if p.AccessPatternSummary == nil {
		p.AccessPatternSummary = make(map[string]interface{})
	}
	if p.Recommendations == nil {
		p.Recommendations = []model.Recommendation{}
	}

	return p, nil
}
