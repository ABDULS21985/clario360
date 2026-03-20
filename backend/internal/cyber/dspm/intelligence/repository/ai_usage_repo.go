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

	"github.com/clario360/platform/internal/cyber/dspm/intelligence/dto"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// AIUsageRepository handles persistence for AI data usage records.
type AIUsageRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewAIUsageRepository creates a new AIUsageRepository.
func NewAIUsageRepository(db *pgxpool.Pool, logger zerolog.Logger) *AIUsageRepository {
	return &AIUsageRepository{db: db, logger: logger}
}

// Upsert inserts or updates an AI data usage record.
func (r *AIUsageRepository) Upsert(ctx context.Context, usage *model.AIDataUsage) error {
	if usage.ID == uuid.Nil {
		usage.ID = uuid.New()
	}
	now := time.Now().UTC()
	usage.UpdatedAt = now
	if usage.CreatedAt.IsZero() {
		usage.CreatedAt = now
	}
	if usage.FirstDetectedAt.IsZero() {
		usage.FirstDetectedAt = now
	}
	if usage.LastDetectedAt.IsZero() {
		usage.LastDetectedAt = now
	}

	riskFactorsJSON, err := json.Marshal(usage.RiskFactors)
	if err != nil {
		return fmt.Errorf("marshal risk factors: %w", err)
	}

	if usage.PIITypes == nil {
		usage.PIITypes = []string{}
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO dspm_ai_data_usage (
			id, tenant_id, data_asset_id, data_asset_name, data_classification,
			contains_pii, pii_types, usage_type,
			model_id, model_name, model_slug,
			pipeline_id, pipeline_name,
			ai_risk_score, ai_risk_level, risk_factors,
			consent_verified, data_minimization, anonymization_level, retention_compliant,
			status, first_detected_at, last_detected_at,
			created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$24
		)
		ON CONFLICT (tenant_id, data_asset_id, usage_type, COALESCE(model_slug,''))
		DO UPDATE SET
			data_asset_name = EXCLUDED.data_asset_name,
			data_classification = EXCLUDED.data_classification,
			contains_pii = EXCLUDED.contains_pii,
			pii_types = EXCLUDED.pii_types,
			model_id = EXCLUDED.model_id,
			model_name = EXCLUDED.model_name,
			pipeline_id = EXCLUDED.pipeline_id,
			pipeline_name = EXCLUDED.pipeline_name,
			ai_risk_score = EXCLUDED.ai_risk_score,
			ai_risk_level = EXCLUDED.ai_risk_level,
			risk_factors = EXCLUDED.risk_factors,
			consent_verified = EXCLUDED.consent_verified,
			data_minimization = EXCLUDED.data_minimization,
			anonymization_level = EXCLUDED.anonymization_level,
			retention_compliant = EXCLUDED.retention_compliant,
			status = EXCLUDED.status,
			last_detected_at = EXCLUDED.last_detected_at,
			updated_at = EXCLUDED.updated_at`,
		usage.ID, usage.TenantID, usage.DataAssetID, usage.DataAssetName, usage.DataClassification,
		usage.ContainsPII, usage.PIITypes, usage.UsageType,
		usage.ModelID, usage.ModelName, usage.ModelSlug,
		usage.PipelineID, usage.PipelineName,
		usage.AIRiskScore, usage.AIRiskLevel, riskFactorsJSON,
		usage.ConsentVerified, usage.DataMinimization, usage.AnonymizationLevel, usage.RetentionCompliant,
		usage.Status, usage.FirstDetectedAt, usage.LastDetectedAt,
		now,
	)
	if err != nil {
		return fmt.Errorf("upsert ai data usage: %w", err)
	}
	return nil
}

// ListByTenant returns paginated AI usage records with filtering.
func (r *AIUsageRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, params *dto.AIUsageListParams) ([]model.AIDataUsage, int, error) {
	if params == nil {
		params = &dto.AIUsageListParams{}
	}
	params.SetDefaults()

	conds := []string{"tenant_id = $1"}
	args := []interface{}{tenantID}
	i := 2

	if params.UsageType != nil && *params.UsageType != "" {
		conds = append(conds, fmt.Sprintf("usage_type = $%d", i))
		args = append(args, *params.UsageType)
		i++
	}
	if params.RiskLevel != nil && *params.RiskLevel != "" {
		conds = append(conds, fmt.Sprintf("ai_risk_level = $%d", i))
		args = append(args, *params.RiskLevel)
		i++
	}
	if params.ModelSlug != nil && *params.ModelSlug != "" {
		conds = append(conds, fmt.Sprintf("model_slug = $%d", i))
		args = append(args, *params.ModelSlug)
		i++
	}
	if params.PIIOnly != nil && *params.PIIOnly {
		conds = append(conds, "contains_pii = true")
	}
	if params.Status != nil && *params.Status != "" {
		conds = append(conds, fmt.Sprintf("status = $%d", i))
		args = append(args, *params.Status)
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM dspm_ai_data_usage "+where, args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count ai usage: %w", err)
	}

	allowedSorts := map[string]string{
		"ai_risk_score":    "ai_risk_score",
		"data_asset_name":  "data_asset_name",
		"last_detected_at": "last_detected_at",
		"created_at":       "created_at",
		"usage_type":       "usage_type",
	}
	order := "ai_risk_score"
	if mapped, ok := allowedSorts[params.Sort]; ok {
		order = mapped
	}
	dir := "DESC"
	if strings.ToLower(params.Order) == "asc" {
		dir = "ASC"
	}

	offset := (params.Page - 1) * params.PerPage
	query := fmt.Sprintf(`
		SELECT id, tenant_id, data_asset_id, data_asset_name, data_classification,
		       contains_pii, pii_types, usage_type,
		       model_id, model_name, model_slug,
		       pipeline_id, pipeline_name,
		       ai_risk_score, ai_risk_level, risk_factors,
		       consent_verified, data_minimization, anonymization_level, retention_compliant,
		       status, first_detected_at, last_detected_at,
		       created_at, updated_at
		FROM dspm_ai_data_usage
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		where, order, dir, i, i+1)
	args = append(args, params.PerPage, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list ai usage: %w", err)
	}
	defer rows.Close()

	items := make([]model.AIDataUsage, 0)
	for rows.Next() {
		u, err := scanAIDataUsage(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *u)
	}
	return items, total, rows.Err()
}

// ListByAsset returns all AI usage records for a specific asset.
func (r *AIUsageRepository) ListByAsset(ctx context.Context, tenantID, assetID uuid.UUID) ([]model.AIDataUsage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, data_asset_id, data_asset_name, data_classification,
		       contains_pii, pii_types, usage_type,
		       model_id, model_name, model_slug,
		       pipeline_id, pipeline_name,
		       ai_risk_score, ai_risk_level, risk_factors,
		       consent_verified, data_minimization, anonymization_level, retention_compliant,
		       status, first_detected_at, last_detected_at,
		       created_at, updated_at
		FROM dspm_ai_data_usage
		WHERE tenant_id = $1 AND data_asset_id = $2
		ORDER BY ai_risk_score DESC`,
		tenantID, assetID,
	)
	if err != nil {
		return nil, fmt.Errorf("list ai usage by asset: %w", err)
	}
	defer rows.Close()

	items := make([]model.AIDataUsage, 0)
	for rows.Next() {
		u, err := scanAIDataUsage(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *u)
	}
	return items, rows.Err()
}

// ListByModel returns all AI usage records for a specific model.
func (r *AIUsageRepository) ListByModel(ctx context.Context, tenantID uuid.UUID, modelSlug string) ([]model.AIDataUsage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, data_asset_id, data_asset_name, data_classification,
		       contains_pii, pii_types, usage_type,
		       model_id, model_name, model_slug,
		       pipeline_id, pipeline_name,
		       ai_risk_score, ai_risk_level, risk_factors,
		       consent_verified, data_minimization, anonymization_level, retention_compliant,
		       status, first_detected_at, last_detected_at,
		       created_at, updated_at
		FROM dspm_ai_data_usage
		WHERE tenant_id = $1 AND model_slug = $2
		ORDER BY ai_risk_score DESC`,
		tenantID, modelSlug,
	)
	if err != nil {
		return nil, fmt.Errorf("list ai usage by model: %w", err)
	}
	defer rows.Close()

	items := make([]model.AIDataUsage, 0)
	for rows.Next() {
		u, err := scanAIDataUsage(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *u)
	}
	return items, rows.Err()
}

// Dashboard returns aggregated AI security metrics.
func (r *AIUsageRepository) Dashboard(ctx context.Context, tenantID uuid.UUID) (*model.AISecurityDashboard, error) {
	dash := &model.AISecurityDashboard{
		RiskByLevel: make(map[string]int),
		UsageByType: make(map[string]int),
	}

	err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE contains_pii AND usage_type = 'training_data'),
			COUNT(*) FILTER (WHERE consent_verified = false),
			COUNT(*) FILTER (WHERE ai_risk_level IN ('high', 'critical')),
			COUNT(DISTINCT model_slug) FILTER (WHERE model_slug IS NOT NULL AND model_slug != ''),
			COALESCE(AVG(ai_risk_score), 0)
		FROM dspm_ai_data_usage
		WHERE tenant_id = $1`,
		tenantID,
	).Scan(
		&dash.TotalAILinkedAssets,
		&dash.PIIInTrainingData,
		&dash.UnverifiedConsent,
		&dash.HighRiskUsageCount,
		&dash.ModelsGoverned,
		&dash.AvgAIRiskScore,
	)
	if err != nil {
		return nil, fmt.Errorf("ai dashboard aggregates: %w", err)
	}

	// Risk breakdown by level
	riskRows, err := r.db.Query(ctx, `
		SELECT ai_risk_level::text, COUNT(*)
		FROM dspm_ai_data_usage
		WHERE tenant_id = $1
		GROUP BY ai_risk_level`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("ai dashboard risk breakdown: %w", err)
	}
	defer riskRows.Close()
	for riskRows.Next() {
		var level string
		var cnt int
		if err := riskRows.Scan(&level, &cnt); err != nil {
			return nil, fmt.Errorf("scan risk level: %w", err)
		}
		dash.RiskByLevel[level] = cnt
	}
	if err := riskRows.Err(); err != nil {
		return nil, err
	}

	// Usage breakdown by type
	typeRows, err := r.db.Query(ctx, `
		SELECT usage_type::text, COUNT(*)
		FROM dspm_ai_data_usage
		WHERE tenant_id = $1
		GROUP BY usage_type`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("ai dashboard usage breakdown: %w", err)
	}
	defer typeRows.Close()
	for typeRows.Next() {
		var usageType string
		var cnt int
		if err := typeRows.Scan(&usageType, &cnt); err != nil {
			return nil, fmt.Errorf("scan usage type: %w", err)
		}
		dash.UsageByType[usageType] = cnt
	}
	if err := typeRows.Err(); err != nil {
		return nil, err
	}

	return dash, nil
}

// scanAIDataUsage scans a single AI data usage row.
func scanAIDataUsage(row interface{ Scan(...interface{}) error }) (*model.AIDataUsage, error) {
	var u model.AIDataUsage
	var riskFactorsJSON []byte
	var piiTypes []string

	err := row.Scan(
		&u.ID, &u.TenantID, &u.DataAssetID, &u.DataAssetName, &u.DataClassification,
		&u.ContainsPII, &piiTypes, &u.UsageType,
		&u.ModelID, &u.ModelName, &u.ModelSlug,
		&u.PipelineID, &u.PipelineName,
		&u.AIRiskScore, &u.AIRiskLevel, &riskFactorsJSON,
		&u.ConsentVerified, &u.DataMinimization, &u.AnonymizationLevel, &u.RetentionCompliant,
		&u.Status, &u.FirstDetectedAt, &u.LastDetectedAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("ai data usage not found")
		}
		return nil, fmt.Errorf("scan ai data usage: %w", err)
	}

	u.PIITypes = piiTypes
	if u.PIITypes == nil {
		u.PIITypes = []string{}
	}

	if len(riskFactorsJSON) > 0 {
		_ = json.Unmarshal(riskFactorsJSON, &u.RiskFactors)
	}
	if u.RiskFactors == nil {
		u.RiskFactors = []model.AIRiskFactor{}
	}

	return &u, nil
}
