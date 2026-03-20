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

// FinancialRepository handles persistence for financial impact records.
type FinancialRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewFinancialRepository creates a new FinancialRepository.
func NewFinancialRepository(db *pgxpool.Pool, logger zerolog.Logger) *FinancialRepository {
	return &FinancialRepository{db: db, logger: logger}
}

// Upsert inserts or updates a financial impact record.
func (r *FinancialRepository) Upsert(ctx context.Context, impact *model.FinancialImpact) error {
	if impact.ID == uuid.Nil {
		impact.ID = uuid.New()
	}
	now := time.Now().UTC()
	impact.UpdatedAt = now
	if impact.CalculatedAt.IsZero() {
		impact.CalculatedAt = now
	}

	breakdownJSON, err := json.Marshal(impact.Breakdown)
	if err != nil {
		return fmt.Errorf("marshal cost breakdown: %w", err)
	}
	methodologyDetailsJSON, err := json.Marshal(impact.MethodologyDetails)
	if err != nil {
		return fmt.Errorf("marshal methodology details: %w", err)
	}
	if impact.ApplicableRegulations == nil {
		impact.ApplicableRegulations = []string{}
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO dspm_financial_impact (
			id, tenant_id, data_asset_id,
			estimated_breach_cost, cost_per_record, record_count,
			cost_breakdown, methodology, methodology_details,
			applicable_regulations, max_regulatory_fine,
			breach_probability_annual, annual_expected_loss,
			calculated_at, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$15
		)
		ON CONFLICT (tenant_id, data_asset_id)
		DO UPDATE SET
			estimated_breach_cost = EXCLUDED.estimated_breach_cost,
			cost_per_record = EXCLUDED.cost_per_record,
			record_count = EXCLUDED.record_count,
			cost_breakdown = EXCLUDED.cost_breakdown,
			methodology = EXCLUDED.methodology,
			methodology_details = EXCLUDED.methodology_details,
			applicable_regulations = EXCLUDED.applicable_regulations,
			max_regulatory_fine = EXCLUDED.max_regulatory_fine,
			breach_probability_annual = EXCLUDED.breach_probability_annual,
			annual_expected_loss = EXCLUDED.annual_expected_loss,
			calculated_at = EXCLUDED.calculated_at,
			updated_at = EXCLUDED.updated_at`,
		impact.ID, impact.TenantID, impact.DataAssetID,
		impact.EstimatedBreachCost, impact.CostPerRecord, impact.RecordCount,
		breakdownJSON, impact.Methodology, methodologyDetailsJSON,
		impact.ApplicableRegulations, impact.MaxRegulatoryFine,
		impact.BreachProbabilityAnnual, impact.AnnualExpectedLoss,
		impact.CalculatedAt, now,
	)
	if err != nil {
		return fmt.Errorf("upsert financial impact: %w", err)
	}
	return nil
}

// GetByAsset returns the financial impact for a specific asset.
func (r *FinancialRepository) GetByAsset(ctx context.Context, tenantID, assetID uuid.UUID) (*model.FinancialImpact, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, data_asset_id,
		       estimated_breach_cost, cost_per_record, record_count,
		       cost_breakdown, methodology, methodology_details,
		       applicable_regulations, max_regulatory_fine,
		       breach_probability_annual, annual_expected_loss,
		       calculated_at, created_at, updated_at
		FROM dspm_financial_impact
		WHERE tenant_id = $1 AND data_asset_id = $2`,
		tenantID, assetID,
	)
	fi, err := scanFinancialImpact(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("financial impact not found")
		}
		return nil, err
	}
	return fi, nil
}

// ListByTenant returns paginated financial impact records.
func (r *FinancialRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, params *dto.FinancialImpactListParams) ([]model.FinancialImpact, int, error) {
	if params == nil {
		params = &dto.FinancialImpactListParams{}
	}
	params.SetDefaults()

	conds := []string{"tenant_id = $1"}
	args := []interface{}{tenantID}
	i := 2

	if params.MinBreachCost != nil {
		conds = append(conds, fmt.Sprintf("estimated_breach_cost >= $%d", i))
		args = append(args, *params.MinBreachCost)
		i++
	}
	if params.Regulation != nil && *params.Regulation != "" {
		conds = append(conds, fmt.Sprintf("$%d = ANY(applicable_regulations)", i))
		args = append(args, *params.Regulation)
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM dspm_financial_impact "+where, args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count financial impact: %w", err)
	}

	allowedSorts := map[string]string{
		"annual_expected_loss":  "annual_expected_loss",
		"estimated_breach_cost": "estimated_breach_cost",
		"max_regulatory_fine":   "max_regulatory_fine",
		"record_count":          "record_count",
		"calculated_at":         "calculated_at",
	}
	order := "annual_expected_loss"
	if mapped, ok := allowedSorts[params.Sort]; ok {
		order = mapped
	}
	dir := "DESC"
	if strings.ToLower(params.Order) == "asc" {
		dir = "ASC"
	}

	offset := (params.Page - 1) * params.PerPage
	query := fmt.Sprintf(`
		SELECT id, tenant_id, data_asset_id,
		       estimated_breach_cost, cost_per_record, record_count,
		       cost_breakdown, methodology, methodology_details,
		       applicable_regulations, max_regulatory_fine,
		       breach_probability_annual, annual_expected_loss,
		       calculated_at, created_at, updated_at
		FROM dspm_financial_impact
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		where, order, dir, i, i+1)
	args = append(args, params.PerPage, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list financial impact: %w", err)
	}
	defer rows.Close()

	items := make([]model.FinancialImpact, 0)
	for rows.Next() {
		fi, err := scanFinancialImpact(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *fi)
	}
	return items, total, rows.Err()
}

// PortfolioRisk returns aggregate financial risk metrics.
func (r *FinancialRepository) PortfolioRisk(ctx context.Context, tenantID uuid.UUID) (*model.PortfolioRisk, error) {
	pr := &model.PortfolioRisk{
		CostByClassification: make(map[string]float64),
		CostByRegulation:     make(map[string]float64),
	}

	err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(estimated_breach_cost), 0),
			COALESCE(SUM(annual_expected_loss), 0),
			COALESCE(MAX(estimated_breach_cost), 0),
			COALESCE(SUM(max_regulatory_fine), 0),
			COUNT(*),
			COUNT(*) FILTER (WHERE annual_expected_loss > 100000),
			COALESCE(AVG(breach_probability_annual), 0)
		FROM dspm_financial_impact
		WHERE tenant_id = $1`,
		tenantID,
	).Scan(
		&pr.TotalBreachCost,
		&pr.TotalAnnualExpectedLoss,
		&pr.MaxSingleAssetExposure,
		&pr.TotalRegulatoryFines,
		&pr.AssetCount,
		&pr.HighRiskAssetCount,
		&pr.AvgBreachProbability,
	)
	if err != nil {
		return nil, fmt.Errorf("portfolio risk aggregates: %w", err)
	}

	// Cost by classification: join with dspm_data_assets to get classification
	classRows, err := r.db.Query(ctx, `
		SELECT COALESCE(da.data_classification, 'unknown'), SUM(fi.estimated_breach_cost)
		FROM dspm_financial_impact fi
		LEFT JOIN dspm_data_assets da ON da.asset_id = fi.data_asset_id AND da.tenant_id = fi.tenant_id
		WHERE fi.tenant_id = $1
		GROUP BY da.data_classification`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("portfolio risk by classification: %w", err)
	}
	defer classRows.Close()
	for classRows.Next() {
		var cls string
		var cost float64
		if err := classRows.Scan(&cls, &cost); err != nil {
			return nil, fmt.Errorf("scan classification cost: %w", err)
		}
		pr.CostByClassification[cls] = cost
	}
	if err := classRows.Err(); err != nil {
		return nil, err
	}

	// Cost by regulation: unnest applicable_regulations
	regRows, err := r.db.Query(ctx, `
		SELECT reg, SUM(estimated_breach_cost)
		FROM dspm_financial_impact, unnest(applicable_regulations) AS reg
		WHERE tenant_id = $1
		GROUP BY reg`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("portfolio risk by regulation: %w", err)
	}
	defer regRows.Close()
	for regRows.Next() {
		var reg string
		var cost float64
		if err := regRows.Scan(&reg, &cost); err != nil {
			return nil, fmt.Errorf("scan regulation cost: %w", err)
		}
		pr.CostByRegulation[reg] = cost
	}
	if err := regRows.Err(); err != nil {
		return nil, err
	}

	return pr, nil
}

// TopRisks returns the top N assets by annual expected loss.
func (r *FinancialRepository) TopRisks(ctx context.Context, tenantID uuid.UUID, limit int) ([]model.FinancialImpact, error) {
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, data_asset_id,
		       estimated_breach_cost, cost_per_record, record_count,
		       cost_breakdown, methodology, methodology_details,
		       applicable_regulations, max_regulatory_fine,
		       breach_probability_annual, annual_expected_loss,
		       calculated_at, created_at, updated_at
		FROM dspm_financial_impact
		WHERE tenant_id = $1
		ORDER BY annual_expected_loss DESC
		LIMIT $2`,
		tenantID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("top financial risks: %w", err)
	}
	defer rows.Close()

	items := make([]model.FinancialImpact, 0, limit)
	for rows.Next() {
		fi, err := scanFinancialImpact(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *fi)
	}
	return items, rows.Err()
}

// scanFinancialImpact scans a single financial impact row.
func scanFinancialImpact(row interface{ Scan(...interface{}) error }) (*model.FinancialImpact, error) {
	var fi model.FinancialImpact
	var breakdownJSON, methodologyDetailsJSON []byte
	var regulations []string

	err := row.Scan(
		&fi.ID, &fi.TenantID, &fi.DataAssetID,
		&fi.EstimatedBreachCost, &fi.CostPerRecord, &fi.RecordCount,
		&breakdownJSON, &fi.Methodology, &methodologyDetailsJSON,
		&regulations, &fi.MaxRegulatoryFine,
		&fi.BreachProbabilityAnnual, &fi.AnnualExpectedLoss,
		&fi.CalculatedAt, &fi.CreatedAt, &fi.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("scan financial impact: %w", err)
	}

	fi.ApplicableRegulations = regulations
	if fi.ApplicableRegulations == nil {
		fi.ApplicableRegulations = []string{}
	}

	if len(breakdownJSON) > 0 {
		_ = json.Unmarshal(breakdownJSON, &fi.Breakdown)
	}
	if len(methodologyDetailsJSON) > 0 {
		_ = json.Unmarshal(methodologyDetailsJSON, &fi.MethodologyDetails)
	}

	return &fi, nil
}
