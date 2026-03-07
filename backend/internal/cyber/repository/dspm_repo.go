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

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
)

// DSPMRepository handles dspm_data_assets and dspm_scans table operations.
type DSPMRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewDSPMRepository creates a new DSPMRepository.
func NewDSPMRepository(db *pgxpool.Pool, logger zerolog.Logger) *DSPMRepository {
	return &DSPMRepository{db: db, logger: logger}
}

// UpsertDataAsset inserts or updates a DSPM data asset record.
func (r *DSPMRepository) UpsertDataAsset(ctx context.Context, asset *model.DSPMDataAsset) error {
	if asset.ID == uuid.Nil {
		asset.ID = uuid.New()
	}
	now := time.Now().UTC()
	asset.UpdatedAt = now

	riskFactorsJSON, _ := json.Marshal(asset.RiskFactors)
	postureJSON, _ := json.Marshal(asset.PostureFindings)
	schemaJSON, _ := json.Marshal(asset.SchemaInfo)
	metaJSON, _ := json.Marshal(asset.Metadata)
	if asset.PIITypes == nil {
		asset.PIITypes = []string{}
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO dspm_data_assets (
			id, tenant_id, asset_id, scan_id, data_classification, sensitivity_score,
			contains_pii, pii_types, pii_column_count, estimated_record_count,
			encrypted_at_rest, encrypted_in_transit, access_control_type, network_exposure,
			backup_configured, audit_logging, last_access_review,
			risk_score, risk_factors, posture_score, posture_findings,
			consumer_count, producer_count, database_type, schema_info, metadata,
			last_scanned_at, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$28
		)
		ON CONFLICT (tenant_id, asset_id) DO UPDATE SET
			scan_id=$4, data_classification=$5, sensitivity_score=$6,
			contains_pii=$7, pii_types=$8, pii_column_count=$9, estimated_record_count=$10,
			encrypted_at_rest=$11, encrypted_in_transit=$12, access_control_type=$13, network_exposure=$14,
			backup_configured=$15, audit_logging=$16, last_access_review=$17,
			risk_score=$18, risk_factors=$19, posture_score=$20, posture_findings=$21,
			consumer_count=$22, producer_count=$23, database_type=$24, schema_info=$25, metadata=$26,
			last_scanned_at=$27, updated_at=$28`,
		asset.ID, asset.TenantID, asset.AssetID, asset.ScanID, asset.DataClassification, asset.SensitivityScore,
		asset.ContainsPII, asset.PIITypes, asset.PIIColumnCount, asset.EstimatedRecordCount,
		asset.EncryptedAtRest, asset.EncryptedInTransit, asset.AccessControlType, asset.NetworkExposure,
		asset.BackupConfigured, asset.AuditLogging, asset.LastAccessReview,
		asset.RiskScore, riskFactorsJSON, asset.PostureScore, postureJSON,
		asset.ConsumerCount, asset.ProducerCount, asset.DatabaseType, schemaJSON, metaJSON,
		now, now,
	)
	if err != nil {
		return fmt.Errorf("upsert dspm data asset: %w", err)
	}
	return nil
}

// GetDataAssetByID retrieves a DSPM data asset by ID.
func (r *DSPMRepository) GetDataAssetByID(ctx context.Context, tenantID, id uuid.UUID) (*model.DSPMDataAsset, error) {
	row := r.db.QueryRow(ctx, `
		SELECT da.id, da.tenant_id, da.asset_id, a.name, a.type, da.scan_id,
		       da.data_classification, da.sensitivity_score, da.contains_pii, da.pii_types,
		       da.pii_column_count, da.estimated_record_count, da.encrypted_at_rest, da.encrypted_in_transit,
		       da.access_control_type, da.network_exposure, da.backup_configured, da.audit_logging, da.last_access_review,
		       da.risk_score, da.risk_factors, da.posture_score, da.posture_findings,
		       da.consumer_count, da.producer_count, da.database_type, da.schema_info, da.metadata,
		       da.last_scanned_at, da.created_at, da.updated_at
		FROM dspm_data_assets da
		LEFT JOIN assets a ON a.id = da.asset_id
		WHERE da.id=$1 AND da.tenant_id=$2`,
		id, tenantID,
	)
	return scanDSPMAsset(row)
}

// GetDataAssetByAssetID retrieves a DSPM data asset by its linked cyber asset ID.
func (r *DSPMRepository) GetDataAssetByAssetID(ctx context.Context, tenantID, assetID uuid.UUID) (*model.DSPMDataAsset, error) {
	row := r.db.QueryRow(ctx, `
		SELECT da.id, da.tenant_id, da.asset_id, a.name, a.type, da.scan_id,
		       da.data_classification, da.sensitivity_score, da.contains_pii, da.pii_types,
		       da.pii_column_count, da.estimated_record_count, da.encrypted_at_rest, da.encrypted_in_transit,
		       da.access_control_type, da.network_exposure, da.backup_configured, da.audit_logging, da.last_access_review,
		       da.risk_score, da.risk_factors, da.posture_score, da.posture_findings,
		       da.consumer_count, da.producer_count, da.database_type, da.schema_info, da.metadata,
		       da.last_scanned_at, da.created_at, da.updated_at
		FROM dspm_data_assets da
		LEFT JOIN assets a ON a.id = da.asset_id
		WHERE da.asset_id=$1 AND da.tenant_id=$2`,
		assetID, tenantID,
	)
	return scanDSPMAsset(row)
}

// ListDataAssets retrieves DSPM data assets with filtering and pagination.
func (r *DSPMRepository) ListDataAssets(ctx context.Context, tenantID uuid.UUID, params *dto.DSPMAssetListParams) ([]*model.DSPMDataAsset, int, error) {
	conds := []string{"da.tenant_id=$1"}
	args := []interface{}{tenantID}
	i := 2

	if params.Classification != nil {
		conds = append(conds, fmt.Sprintf("da.data_classification=$%d", i))
		args = append(args, *params.Classification)
		i++
	}
	if params.ContainsPII != nil {
		conds = append(conds, fmt.Sprintf("da.contains_pii=$%d", i))
		args = append(args, *params.ContainsPII)
		i++
	}
	if params.MinRiskScore != nil {
		conds = append(conds, fmt.Sprintf("da.risk_score>=$%d", i))
		args = append(args, *params.MinRiskScore)
		i++
	}
	if params.NetworkExposure != nil {
		conds = append(conds, fmt.Sprintf("da.network_exposure=$%d", i))
		args = append(args, *params.NetworkExposure)
		i++
	}
	if params.AssetID != nil {
		conds = append(conds, fmt.Sprintf("da.asset_id=$%d", i))
		args = append(args, *params.AssetID)
		i++
	}
	if params.Search != nil && *params.Search != "" {
		conds = append(conds, fmt.Sprintf("a.name ILIKE $%d", i))
		args = append(args, "%"+*params.Search+"%")
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM dspm_data_assets da LEFT JOIN assets a ON a.id = da.asset_id "+where, args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count dspm assets: %w", err)
	}

	order := "da.risk_score"
	allowedSorts := map[string]string{
		"risk_score":          "da.risk_score",
		"posture_score":       "da.posture_score",
		"data_classification": "da.data_classification",
		"sensitivity_score":   "da.sensitivity_score",
		"created_at":          "da.created_at",
		"updated_at":          "da.updated_at",
	}
	if mapped, ok := allowedSorts[params.Sort]; ok {
		order = mapped
	}
	dir := "DESC"
	if strings.ToLower(params.Order) == "asc" {
		dir = "ASC"
	}

	offset := (params.Page - 1) * params.PerPage
	query := fmt.Sprintf(`
		SELECT da.id, da.tenant_id, da.asset_id, a.name, a.type, da.scan_id,
		       da.data_classification, da.sensitivity_score, da.contains_pii, da.pii_types,
		       da.pii_column_count, da.estimated_record_count, da.encrypted_at_rest, da.encrypted_in_transit,
		       da.access_control_type, da.network_exposure, da.backup_configured, da.audit_logging, da.last_access_review,
		       da.risk_score, da.risk_factors, da.posture_score, da.posture_findings,
		       da.consumer_count, da.producer_count, da.database_type, da.schema_info, da.metadata,
		       da.last_scanned_at, da.created_at, da.updated_at
		FROM dspm_data_assets da
		LEFT JOIN assets a ON a.id = da.asset_id
		%s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		where, order, dir, i, i+1,
	)
	args = append(args, params.PerPage, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list dspm assets: %w", err)
	}
	defer rows.Close()

	var assets []*model.DSPMDataAsset
	for rows.Next() {
		a, err := scanDSPMAsset(rows)
		if err != nil {
			return nil, 0, err
		}
		assets = append(assets, a)
	}
	return assets, total, rows.Err()
}

// CreateScan inserts a new DSPM scan record.
func (r *DSPMRepository) CreateScan(ctx context.Context, tenantID, createdBy uuid.UUID) (*model.DSPMScan, error) {
	id := uuid.New()
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `
		INSERT INTO dspm_scans (id, tenant_id, status, started_at, created_by, created_at)
		VALUES ($1,$2,'running',$3,$4,$3)`,
		id, tenantID, now, createdBy,
	)
	if err != nil {
		return nil, fmt.Errorf("create dspm scan: %w", err)
	}
	return r.GetScanByID(ctx, tenantID, id)
}

// UpdateScanCompleted marks a scan as completed with stats.
func (r *DSPMRepository) UpdateScanCompleted(ctx context.Context, tenantID, scanID uuid.UUID, assetsScanned, piiFound, highRisk, findings int, durationMs int64) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `
		UPDATE dspm_scans SET status='completed', assets_scanned=$1, pii_assets_found=$2,
		high_risk_found=$3, findings_count=$4, completed_at=$5, duration_ms=$6
		WHERE id=$7 AND tenant_id=$8`,
		assetsScanned, piiFound, highRisk, findings, now, durationMs, scanID, tenantID,
	)
	return err
}

// UpdateScanFailed marks a scan as failed.
func (r *DSPMRepository) UpdateScanFailed(ctx context.Context, tenantID, scanID uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `
		UPDATE dspm_scans SET status='failed', completed_at=$1 WHERE id=$2 AND tenant_id=$3`,
		now, scanID, tenantID,
	)
	return err
}

// GetScanByID retrieves a scan by ID.
func (r *DSPMRepository) GetScanByID(ctx context.Context, tenantID, id uuid.UUID) (*model.DSPMScan, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, status, assets_scanned, pii_assets_found, high_risk_found,
		       findings_count, started_at, completed_at, duration_ms, created_by, created_at
		FROM dspm_scans WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
	)
	return scanDSPMScan(row)
}

// ListScans retrieves scan history.
func (r *DSPMRepository) ListScans(ctx context.Context, tenantID uuid.UUID, params *dto.DSPMScanListParams) ([]*model.DSPMScan, int, error) {
	conds := []string{"tenant_id=$1"}
	args := []interface{}{tenantID}
	i := 2

	if params.Status != nil {
		conds = append(conds, fmt.Sprintf("status=$%d", i))
		args = append(args, *params.Status)
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM dspm_scans "+where, args...).Scan(&total)

	offset := (params.Page - 1) * params.PerPage
	query := fmt.Sprintf(
		`SELECT id, tenant_id, status, assets_scanned, pii_assets_found, high_risk_found,
		        findings_count, started_at, completed_at, duration_ms, created_by, created_at
		 FROM dspm_scans %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, i, i+1,
	)
	args = append(args, params.PerPage, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list scans: %w", err)
	}
	defer rows.Close()

	var scans []*model.DSPMScan
	for rows.Next() {
		s, err := scanDSPMScan(rows)
		if err != nil {
			return nil, 0, err
		}
		scans = append(scans, s)
	}
	return scans, total, rows.Err()
}

// Dashboard returns aggregate DSPM metrics.
func (r *DSPMRepository) Dashboard(ctx context.Context, tenantID uuid.UUID) (*model.DSPMDashboard, error) {
	dash := &model.DSPMDashboard{
		ClassificationBreakdown: make(map[string]int),
		ExposureBreakdown:       make(map[string]int),
		PIITypeFrequency:        make(map[string]int),
	}

	_ = r.db.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE contains_pii),
		       COUNT(*) FILTER (WHERE risk_score >= 70),
		       COALESCE(AVG(posture_score), 0), COALESCE(AVG(risk_score), 0)
		FROM dspm_data_assets WHERE tenant_id=$1`, tenantID,
	).Scan(&dash.TotalDataAssets, &dash.PIIAssetsCount, &dash.HighRiskAssetsCount, &dash.AvgPostureScore, &dash.AvgRiskScore)

	classRows, err := r.db.Query(ctx, `
		SELECT data_classification, COUNT(*) FROM dspm_data_assets
		WHERE tenant_id=$1 GROUP BY data_classification`, tenantID,
	)
	if err == nil {
		defer classRows.Close()
		for classRows.Next() {
			var cls string
			var cnt int
			_ = classRows.Scan(&cls, &cnt)
			dash.ClassificationBreakdown[cls] = cnt
		}
	}

	expRows, err := r.db.Query(ctx, `
		SELECT COALESCE(network_exposure, 'unknown'), COUNT(*) FROM dspm_data_assets
		WHERE tenant_id=$1 GROUP BY network_exposure`, tenantID,
	)
	if err == nil {
		defer expRows.Close()
		for expRows.Next() {
			var exp string
			var cnt int
			_ = expRows.Scan(&exp, &cnt)
			dash.ExposureBreakdown[exp] = cnt
		}
	}

	// Top 5 riskiest assets
	topParams := &dto.DSPMAssetListParams{Page: 1, PerPage: 5, Sort: "risk_score", Order: "desc"}
	topParams.SetDefaults()
	topAssets, _, _ := r.ListDataAssets(ctx, tenantID, topParams)
	for _, a := range topAssets {
		dash.TopRiskyAssets = append(dash.TopRiskyAssets, *a)
	}

	// Recent scans
	scanParams := &dto.DSPMScanListParams{Page: 1, PerPage: 5}
	recentScans, _, _ := r.ListScans(ctx, tenantID, scanParams)
	for _, s := range recentScans {
		dash.RecentScans = append(dash.RecentScans, *s)
	}

	return dash, nil
}

// ClassificationSummary returns per-classification counts.
func (r *DSPMRepository) ClassificationSummary(ctx context.Context, tenantID uuid.UUID) (*model.DSPMClassificationSummary, error) {
	summary := &model.DSPMClassificationSummary{}
	rows, err := r.db.Query(ctx, `
		SELECT data_classification, COUNT(*) FROM dspm_data_assets
		WHERE tenant_id=$1 GROUP BY data_classification`, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var cls string
		var cnt int
		_ = rows.Scan(&cls, &cnt)
		summary.Total += cnt
		switch cls {
		case "public":
			summary.Public = cnt
		case "internal":
			summary.Internal = cnt
		case "confidential":
			summary.Confidential = cnt
		case "restricted":
			summary.Restricted = cnt
		}
	}
	return summary, rows.Err()
}

// ExposureAnalysis returns data exposure statistics.
func (r *DSPMRepository) ExposureAnalysis(ctx context.Context, tenantID uuid.UUID) (*model.DSPMExposureAnalysis, error) {
	analysis := &model.DSPMExposureAnalysis{}
	rows, err := r.db.Query(ctx, `
		SELECT COALESCE(network_exposure, 'unknown'), COUNT(*) FROM dspm_data_assets
		WHERE tenant_id=$1 GROUP BY network_exposure`, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var exp string
		var cnt int
		_ = rows.Scan(&exp, &cnt)
		switch exp {
		case "internal_only":
			analysis.InternalOnly = cnt
		case "vpn_accessible":
			analysis.VPNAccessible = cnt
		case "internet_facing":
			analysis.InternetFacing = cnt
		default:
			analysis.Unknown += cnt
		}
	}

	// Internet-facing with restricted/confidential data
	critParams := &dto.DSPMAssetListParams{Page: 1, PerPage: 10}
	netExp := "internet_facing"
	critParams.NetworkExposure = &netExp
	critParams.SetDefaults()
	critAssets, _, _ := r.ListDataAssets(ctx, tenantID, critParams)
	for _, a := range critAssets {
		if a.DataClassification == "restricted" || a.DataClassification == "confidential" {
			analysis.CriticalExposures = append(analysis.CriticalExposures, *a)
		}
	}

	return analysis, nil
}

// AvgPostureScore returns the average posture score for a tenant.
func (r *DSPMRepository) AvgPostureScore(ctx context.Context, tenantID uuid.UUID) (float64, error) {
	var score float64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(posture_score), 0) FROM dspm_data_assets WHERE tenant_id=$1`, tenantID,
	).Scan(&score)
	return score, err
}

func scanDSPMAsset(row interface{ Scan(...interface{}) error }) (*model.DSPMDataAsset, error) {
	var a model.DSPMDataAsset
	var riskJSON, postureJSON, schemaJSON, metaJSON []byte
	var piiTypes []string

	err := row.Scan(
		&a.ID, &a.TenantID, &a.AssetID, &a.AssetName, &a.AssetType, &a.ScanID,
		&a.DataClassification, &a.SensitivityScore, &a.ContainsPII, &piiTypes,
		&a.PIIColumnCount, &a.EstimatedRecordCount, &a.EncryptedAtRest, &a.EncryptedInTransit,
		&a.AccessControlType, &a.NetworkExposure, &a.BackupConfigured, &a.AuditLogging, &a.LastAccessReview,
		&a.RiskScore, &riskJSON, &a.PostureScore, &postureJSON,
		&a.ConsumerCount, &a.ProducerCount, &a.DatabaseType, &schemaJSON, &metaJSON,
		&a.LastScannedAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan dspm asset: %w", err)
	}

	a.PIITypes = piiTypes
	if a.PIITypes == nil {
		a.PIITypes = []string{}
	}
	_ = json.Unmarshal(riskJSON, &a.RiskFactors)
	_ = json.Unmarshal(postureJSON, &a.PostureFindings)
	_ = json.Unmarshal(schemaJSON, &a.SchemaInfo)
	_ = json.Unmarshal(metaJSON, &a.Metadata)
	return &a, nil
}

func scanDSPMScan(row interface{ Scan(...interface{}) error }) (*model.DSPMScan, error) {
	var s model.DSPMScan
	err := row.Scan(
		&s.ID, &s.TenantID, &s.Status, &s.AssetsScanned, &s.PIIAssetsFound, &s.HighRiskFound,
		&s.FindingsCount, &s.StartedAt, &s.CompletedAt, &s.DurationMs, &s.CreatedBy, &s.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan dspm scan: %w", err)
	}
	return &s, nil
}
