package repository

import (
	"context"
	"encoding/json"
	"fmt"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type ValidationResultRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewValidationResultRepository(db *pgxpool.Pool, logger zerolog.Logger) *ValidationResultRepository {
	return &ValidationResultRepository{db: db, logger: loggerWithRepo(logger, "ai_validation_result")}
}

func (r *ValidationResultRepository) Create(ctx context.Context, item *aigovmodel.ValidationResult) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO ai_validation_results (
			id, tenant_id, model_id, version_id, dataset_type, dataset_size, positive_count, negative_count,
			true_positives, false_positives, true_negatives, false_negatives, precision, recall, f1_score,
			false_positive_rate, accuracy, auc, roc_curve, production_metrics, deltas, by_severity,
			by_rule_type, false_positive_samples, false_negative_samples, recommendation,
			recommendation_reason, warnings, validated_at, duration_ms
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22,
			$23, $24, $25, $26,
			$27, $28, $29, $30
		)`,
		item.ID, item.TenantID, item.ModelID, item.VersionID, item.DatasetType, item.DatasetSize, item.PositiveCount, item.NegativeCount,
		item.TruePositives, item.FalsePositives, item.TrueNegatives, item.FalseNegatives, item.Precision, item.Recall, item.F1Score,
		item.FalsePositiveRate, item.Accuracy, item.AUC, marshalJSON(item.ROCCurve), marshalJSON(item.ProductionMetrics),
		marshalJSON(item.Deltas), marshalJSON(item.BySeverity), marshalJSON(item.ByRuleType), marshalJSON(item.FPSamples),
		marshalJSON(item.FNSamples), item.Recommendation, item.RecommendationReason, marshalJSON(item.Warnings),
		item.ValidatedAt, item.DurationMs,
	)
	if err != nil {
		return fmt.Errorf("insert ai validation result: %w", err)
	}
	return nil
}

func (r *ValidationResultRepository) LatestByVersion(ctx context.Context, tenantID, versionID uuid.UUID) (*aigovmodel.ValidationResult, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, model_id, version_id, dataset_type, dataset_size, positive_count, negative_count,
		       true_positives, false_positives, true_negatives, false_negatives, precision, recall, f1_score,
		       false_positive_rate, accuracy, auc, roc_curve, production_metrics, deltas, by_severity,
		       by_rule_type, false_positive_samples, false_negative_samples, recommendation,
		       recommendation_reason, warnings, validated_at, duration_ms
		FROM ai_validation_results
		WHERE tenant_id = $1 AND version_id = $2
		ORDER BY validated_at DESC
		LIMIT 1`,
		tenantID, versionID,
	)
	item, err := scanValidationResult(row)
	if err != nil {
		return nil, rowNotFound(err)
	}
	return item, nil
}

func (r *ValidationResultRepository) HistoryByVersion(ctx context.Context, tenantID, versionID uuid.UUID, limit int) ([]aigovmodel.ValidationResult, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, model_id, version_id, dataset_type, dataset_size, positive_count, negative_count,
		       true_positives, false_positives, true_negatives, false_negatives, precision, recall, f1_score,
		       false_positive_rate, accuracy, auc, roc_curve, production_metrics, deltas, by_severity,
		       by_rule_type, false_positive_samples, false_negative_samples, recommendation,
		       recommendation_reason, warnings, validated_at, duration_ms
		FROM ai_validation_results
		WHERE tenant_id = $1 AND version_id = $2
		ORDER BY validated_at DESC
		LIMIT $3`,
		tenantID, versionID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list ai validation history: %w", err)
	}
	defer rows.Close()
	items := make([]aigovmodel.ValidationResult, 0, limit)
	for rows.Next() {
		item, err := scanValidationResult(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

type validationScannable interface {
	Scan(dest ...any) error
}

func scanValidationResult(row validationScannable) (*aigovmodel.ValidationResult, error) {
	item := &aigovmodel.ValidationResult{}
	var (
		rocCurve             []byte
		productionMetrics    []byte
		deltas               []byte
		bySeverity           []byte
		byRuleType           []byte
		falsePositiveSamples []byte
		falseNegativeSamples []byte
		warnings             []byte
	)
	if err := row.Scan(
		&item.ID, &item.TenantID, &item.ModelID, &item.VersionID, &item.DatasetType, &item.DatasetSize, &item.PositiveCount, &item.NegativeCount,
		&item.TruePositives, &item.FalsePositives, &item.TrueNegatives, &item.FalseNegatives, &item.Precision, &item.Recall, &item.F1Score,
		&item.FalsePositiveRate, &item.Accuracy, &item.AUC, &rocCurve, &productionMetrics, &deltas, &bySeverity,
		&byRuleType, &falsePositiveSamples, &falseNegativeSamples, &item.Recommendation, &item.RecommendationReason,
		&warnings, &item.ValidatedAt, &item.DurationMs,
	); err != nil {
		return nil, err
	}
	if len(rocCurve) > 0 {
		if err := json.Unmarshal(rocCurve, &item.ROCCurve); err != nil {
			return nil, fmt.Errorf("decode ai validation roc_curve: %w", err)
		}
	}
	if len(productionMetrics) > 0 && string(productionMetrics) != "null" {
		var summary aigovmodel.MetricsSummary
		if err := json.Unmarshal(productionMetrics, &summary); err != nil {
			return nil, fmt.Errorf("decode ai validation production_metrics: %w", err)
		}
		item.ProductionMetrics = &summary
	}
	if len(deltas) > 0 && string(deltas) != "null" {
		if err := json.Unmarshal(deltas, &item.Deltas); err != nil {
			return nil, fmt.Errorf("decode ai validation deltas: %w", err)
		}
	}
	if len(bySeverity) > 0 && string(bySeverity) != "null" {
		if err := json.Unmarshal(bySeverity, &item.BySeverity); err != nil {
			return nil, fmt.Errorf("decode ai validation by_severity: %w", err)
		}
	}
	if len(byRuleType) > 0 && string(byRuleType) != "null" {
		if err := json.Unmarshal(byRuleType, &item.ByRuleType); err != nil {
			return nil, fmt.Errorf("decode ai validation by_rule_type: %w", err)
		}
	}
	if len(falsePositiveSamples) > 0 {
		if err := json.Unmarshal(falsePositiveSamples, &item.FPSamples); err != nil {
			return nil, fmt.Errorf("decode ai validation false_positive_samples: %w", err)
		}
	}
	if len(falseNegativeSamples) > 0 {
		if err := json.Unmarshal(falseNegativeSamples, &item.FNSamples); err != nil {
			return nil, fmt.Errorf("decode ai validation false_negative_samples: %w", err)
		}
	}
	if len(warnings) > 0 {
		if err := json.Unmarshal(warnings, &item.Warnings); err != nil {
			return nil, fmt.Errorf("decode ai validation warnings: %w", err)
		}
	}
	if item.ROCCurve == nil {
		item.ROCCurve = []aigovmodel.ROCPoint{}
	}
	if item.BySeverity == nil {
		item.BySeverity = map[string]aigovmodel.MetricsSummary{}
	}
	if item.FPSamples == nil {
		item.FPSamples = []aigovmodel.PredictionSample{}
	}
	if item.FNSamples == nil {
		item.FNSamples = []aigovmodel.PredictionSample{}
	}
	if item.Warnings == nil {
		item.Warnings = []string{}
	}
	return item, nil
}

func marshalJSON(value any) []byte {
	if value == nil {
		return []byte("null")
	}
	payload, _ := json.Marshal(value)
	return payload
}
