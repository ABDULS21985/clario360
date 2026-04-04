package repository

import (
	"context"
	"fmt"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type ShadowComparisonRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewShadowComparisonRepository(db *pgxpool.Pool, logger zerolog.Logger) *ShadowComparisonRepository {
	return &ShadowComparisonRepository{db: db, logger: loggerWithRepo(logger, "ai_shadow_comparison")}
}

func (r *ShadowComparisonRepository) Create(ctx context.Context, item *aigovmodel.ShadowComparison) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO ai_shadow_comparisons (
			id, tenant_id, model_id, production_version_id, shadow_version_id, period_start, period_end,
			total_predictions, agreement_count, disagreement_count, agreement_rate, production_metrics,
			shadow_metrics, metrics_delta, divergence_samples, divergence_by_use_case, recommendation,
			recommendation_reason, recommendation_factors, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		)`,
		item.ID, item.TenantID, item.ModelID, item.ProductionVersionID, item.ShadowVersionID, item.PeriodStart,
		item.PeriodEnd, item.TotalPredictions, item.AgreementCount, item.DisagreementCount, item.AgreementRate,
		item.ProductionMetrics, item.ShadowMetrics, item.MetricsDelta, item.DivergenceSamples,
		item.DivergenceByUseCase, item.Recommendation, item.RecommendationReason, item.RecommendationFactors,
		item.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert ai shadow comparison: %w", err)
	}
	return nil
}

func (r *ShadowComparisonRepository) LatestByModel(ctx context.Context, tenantID, modelID uuid.UUID) (*aigovmodel.ShadowComparison, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, model_id, production_version_id, shadow_version_id, period_start, period_end,
		       total_predictions, agreement_count, disagreement_count, agreement_rate, production_metrics,
		       shadow_metrics, metrics_delta, divergence_samples, divergence_by_use_case, recommendation,
		       recommendation_reason, recommendation_factors, created_at
		FROM ai_shadow_comparisons
		WHERE tenant_id = $1 AND model_id = $2
		ORDER BY created_at DESC
		LIMIT 1`,
		tenantID, modelID,
	)
	item, err := scanShadowComparison(row)
	if err != nil {
		return nil, rowNotFound(err)
	}
	return item, nil
}

func (r *ShadowComparisonRepository) LatestByShadowVersion(ctx context.Context, tenantID, shadowVersionID uuid.UUID) (*aigovmodel.ShadowComparison, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, model_id, production_version_id, shadow_version_id, period_start, period_end,
		       total_predictions, agreement_count, disagreement_count, agreement_rate, production_metrics,
		       shadow_metrics, metrics_delta, divergence_samples, divergence_by_use_case, recommendation,
		       recommendation_reason, recommendation_factors, created_at
		FROM ai_shadow_comparisons
		WHERE tenant_id = $1 AND shadow_version_id = $2
		ORDER BY created_at DESC
		LIMIT 1`,
		tenantID, shadowVersionID,
	)
	item, err := scanShadowComparison(row)
	if err != nil {
		return nil, rowNotFound(err)
	}
	return item, nil
}

func (r *ShadowComparisonRepository) History(ctx context.Context, tenantID, modelID uuid.UUID, limit int) ([]aigovmodel.ShadowComparison, error) {
	if limit <= 0 {
		limit = 24
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, model_id, production_version_id, shadow_version_id, period_start, period_end,
		       total_predictions, agreement_count, disagreement_count, agreement_rate, production_metrics,
		       shadow_metrics, metrics_delta, divergence_samples, divergence_by_use_case, recommendation,
		       recommendation_reason, recommendation_factors, created_at
		FROM ai_shadow_comparisons
		WHERE tenant_id = $1 AND model_id = $2
		ORDER BY created_at DESC
		LIMIT $3`,
		tenantID, modelID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list ai shadow comparison history: %w", err)
	}
	defer rows.Close()
	items := make([]aigovmodel.ShadowComparison, 0)
	for rows.Next() {
		item, err := scanShadowComparison(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

type shadowScannable interface {
	Scan(dest ...any) error
}

func scanShadowComparison(row shadowScannable) (*aigovmodel.ShadowComparison, error) {
	item := &aigovmodel.ShadowComparison{}
	var productionMetrics, shadowMetrics, metricsDelta, divergenceSamples, divergenceByUseCase, recommendationFactors []byte
	if err := row.Scan(
		&item.ID, &item.TenantID, &item.ModelID, &item.ProductionVersionID, &item.ShadowVersionID,
		&item.PeriodStart, &item.PeriodEnd, &item.TotalPredictions, &item.AgreementCount,
		&item.DisagreementCount, &item.AgreementRate, &productionMetrics, &shadowMetrics,
		&metricsDelta, &divergenceSamples, &divergenceByUseCase, &item.Recommendation,
		&item.RecommendationReason, &recommendationFactors, &item.CreatedAt,
	); err != nil {
		return nil, err
	}
	item.ProductionMetrics = nullJSON(productionMetrics, "{}")
	item.ShadowMetrics = nullJSON(shadowMetrics, "{}")
	item.MetricsDelta = nullJSON(metricsDelta, "{}")
	item.DivergenceSamples = nullJSON(divergenceSamples, "[]")
	item.DivergenceByUseCase = nullJSON(divergenceByUseCase, "{}")
	item.RecommendationFactors = nullJSON(recommendationFactors, "[]")
	return item, nil
}
