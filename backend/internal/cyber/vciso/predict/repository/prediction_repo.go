package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	predictmodel "github.com/clario360/platform/internal/cyber/vciso/predict/model"
)

var ErrNotFound = errors.New("not found")

type PredictionRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewPredictionRepository(db *pgxpool.Pool, logger zerolog.Logger) *PredictionRepository {
	return &PredictionRepository{
		db:     db,
		logger: logger.With().Str("component", "vciso_predict_repo").Logger(),
	}
}

func (r *PredictionRepository) CreatePrediction(ctx context.Context, item *predictmodel.StoredPrediction) error {
	if item == nil {
		return fmt.Errorf("prediction is required")
	}
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	}
	confidenceInterval, err := json.Marshal(item.ConfidenceInterval)
	if err != nil {
		return fmt.Errorf("marshal confidence interval: %w", err)
	}
	topFeatures, err := json.Marshal(item.TopFeatures)
	if err != nil {
		return fmt.Errorf("marshal top features: %w", err)
	}
	if len(item.PredictionJSON) == 0 {
		item.PredictionJSON = json.RawMessage(`{}`)
	}
	if len(item.OutcomeValue) == 0 {
		item.OutcomeValue = nil
	}
	return r.db.QueryRow(ctx, `
		INSERT INTO vciso_predictions (
			id, tenant_id, prediction_type, model_version, prediction_json, confidence_score,
			confidence_interval, top_features, explanation_text, target_entity_type, target_entity_id,
			forecast_start, forecast_end, outcome_observed, outcome_value, accuracy_score,
			prediction_log_id, created_at, evaluated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,
			$7,$8,$9,$10,$11,
			$12,$13,$14,$15,$16,
			$17,$18,$19
		)
		RETURNING created_at`,
		item.ID,
		item.TenantID,
		item.PredictionType,
		item.ModelVersion,
		item.PredictionJSON,
		item.ConfidenceScore,
		confidenceInterval,
		topFeatures,
		item.ExplanationText,
		item.TargetEntityType,
		item.TargetEntityID,
		item.ForecastStart,
		item.ForecastEnd,
		item.OutcomeObserved,
		item.OutcomeValue,
		item.AccuracyScore,
		item.PredictionLogID,
		item.CreatedAt,
		item.EvaluatedAt,
	).Scan(&item.CreatedAt)
}

func (r *PredictionRepository) ListPredictions(ctx context.Context, tenantID uuid.UUID, predictionType *predictmodel.PredictionType, limit int) ([]predictmodel.StoredPrediction, error) {
	if limit <= 0 {
		limit = 50
	}
	args := []any{tenantID}
	query := `
		SELECT id, tenant_id, prediction_type, model_version, prediction_json, confidence_score,
		       confidence_interval, top_features, explanation_text, target_entity_type, target_entity_id,
		       forecast_start, forecast_end, outcome_observed, outcome_value, accuracy_score,
		       prediction_log_id, created_at, evaluated_at
		FROM vciso_predictions
		WHERE tenant_id = $1`
	if predictionType != nil {
		args = append(args, *predictionType)
		query += fmt.Sprintf(" AND prediction_type = $%d", len(args))
	}
	args = append(args, limit)
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", len(args))
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list predictions: %w", err)
	}
	defer rows.Close()
	items := make([]predictmodel.StoredPrediction, 0, limit)
	for rows.Next() {
		item, err := scanPrediction(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *PredictionRepository) ListPendingEvaluation(ctx context.Context, tenantID uuid.UUID, now time.Time, limit int) ([]predictmodel.StoredPrediction, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, prediction_type, model_version, prediction_json, confidence_score,
		       confidence_interval, top_features, explanation_text, target_entity_type, target_entity_id,
		       forecast_start, forecast_end, outcome_observed, outcome_value, accuracy_score,
		       prediction_log_id, created_at, evaluated_at
		FROM vciso_predictions
		WHERE tenant_id = $1
		  AND outcome_observed IS NULL
		  AND forecast_end <= $2
		ORDER BY forecast_end ASC
		LIMIT $3`,
		tenantID, now, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list pending evaluations: %w", err)
	}
	defer rows.Close()
	items := make([]predictmodel.StoredPrediction, 0, limit)
	for rows.Next() {
		item, err := scanPrediction(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *PredictionRepository) UpdateOutcome(ctx context.Context, predictionID uuid.UUID, observed bool, outcome any, accuracy float64, evaluatedAt time.Time) error {
	outcomeJSON, err := json.Marshal(outcome)
	if err != nil {
		return fmt.Errorf("marshal outcome: %w", err)
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE vciso_predictions
		SET outcome_observed = $2,
		    outcome_value = $3,
		    accuracy_score = $4,
		    evaluated_at = $5
		WHERE id = $1`,
		predictionID, observed, outcomeJSON, accuracy, evaluatedAt,
	)
	if err != nil {
		return fmt.Errorf("update prediction outcome: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PredictionRepository) AccuracyByType(ctx context.Context, tenantID uuid.UUID) (map[predictmodel.PredictionType]float64, error) {
	rows, err := r.db.Query(ctx, `
		SELECT prediction_type, COALESCE(AVG(accuracy_score), 0)::float8
		FROM vciso_predictions
		WHERE tenant_id = $1
		  AND outcome_observed IS NOT NULL
		GROUP BY prediction_type`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("prediction accuracy by type: %w", err)
	}
	defer rows.Close()
	out := map[predictmodel.PredictionType]float64{}
	for rows.Next() {
		var predictionType predictmodel.PredictionType
		var value float64
		if err := rows.Scan(&predictionType, &value); err != nil {
			return nil, err
		}
		out[predictionType] = value
	}
	return out, rows.Err()
}

func (r *PredictionRepository) UpsertModel(ctx context.Context, item *predictmodel.PredictionModel) error {
	if item == nil {
		return fmt.Errorf("prediction model is required")
	}
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO vciso_prediction_models (
			id, model_type, version, model_artifact_path, model_framework,
			backtest_accuracy, backtest_precision, backtest_recall, backtest_f1, backtest_mape,
			feature_count, training_samples, training_duration_seconds, status, active,
			last_drift_check, drift_score, created_at, activated_at, deprecated_at
		) VALUES (
			$1,$2,$3,$4,$5,
			$6,$7,$8,$9,$10,
			$11,$12,$13,$14,$15,
			$16,$17,$18,$19,$20
		)
		ON CONFLICT (model_type, version) DO UPDATE SET
			model_artifact_path = EXCLUDED.model_artifact_path,
			model_framework = EXCLUDED.model_framework,
			backtest_accuracy = EXCLUDED.backtest_accuracy,
			backtest_precision = EXCLUDED.backtest_precision,
			backtest_recall = EXCLUDED.backtest_recall,
			backtest_f1 = EXCLUDED.backtest_f1,
			backtest_mape = EXCLUDED.backtest_mape,
			feature_count = EXCLUDED.feature_count,
			training_samples = EXCLUDED.training_samples,
			training_duration_seconds = EXCLUDED.training_duration_seconds,
			status = EXCLUDED.status,
			active = EXCLUDED.active,
			last_drift_check = EXCLUDED.last_drift_check,
			drift_score = EXCLUDED.drift_score,
			activated_at = EXCLUDED.activated_at,
			deprecated_at = EXCLUDED.deprecated_at`,
		item.ID,
		item.ModelType,
		item.Version,
		item.ModelArtifactPath,
		item.ModelFramework,
		item.BacktestAccuracy,
		item.BacktestPrecision,
		item.BacktestRecall,
		item.BacktestF1,
		item.BacktestMAPE,
		item.FeatureCount,
		item.TrainingSamples,
		item.TrainingDurationSeconds,
		item.Status,
		item.Active,
		item.LastDriftCheck,
		item.DriftScore,
		item.CreatedAt,
		item.ActivatedAt,
		item.DeprecatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert prediction model: %w", err)
	}
	if item.Active {
		_, err = r.db.Exec(ctx, `
			UPDATE vciso_prediction_models
			SET active = false
			WHERE model_type = $1 AND version != $2`,
			item.ModelType, item.Version,
		)
		if err != nil {
			return fmt.Errorf("deactivate previous prediction models: %w", err)
		}
	}
	return nil
}

func (r *PredictionRepository) GetActiveModel(ctx context.Context, modelType predictmodel.PredictionType) (*predictmodel.PredictionModel, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, model_type, version, model_artifact_path, model_framework,
		       backtest_accuracy, backtest_precision, backtest_recall, backtest_f1, backtest_mape,
		       feature_count, training_samples, training_duration_seconds, status, active,
		       last_drift_check, drift_score, created_at, activated_at, deprecated_at
		FROM vciso_prediction_models
		WHERE model_type = $1 AND active = true
		ORDER BY created_at DESC
		LIMIT 1`,
		modelType,
	)
	return scanPredictionModel(row)
}

func (r *PredictionRepository) ListModels(ctx context.Context) ([]predictmodel.PredictionModel, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, model_type, version, model_artifact_path, model_framework,
		       backtest_accuracy, backtest_precision, backtest_recall, backtest_f1, backtest_mape,
		       feature_count, training_samples, training_duration_seconds, status, active,
		       last_drift_check, drift_score, created_at, activated_at, deprecated_at
		FROM vciso_prediction_models
		ORDER BY model_type, created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list prediction models: %w", err)
	}
	defer rows.Close()
	items := make([]predictmodel.PredictionModel, 0, 16)
	for rows.Next() {
		item, err := scanPredictionModel(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func scanPrediction(row interface{ Scan(...any) error }) (*predictmodel.StoredPrediction, error) {
	var (
		item           predictmodel.StoredPrediction
		confidenceRaw  []byte
		topFeaturesRaw []byte
	)
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.PredictionType,
		&item.ModelVersion,
		&item.PredictionJSON,
		&item.ConfidenceScore,
		&confidenceRaw,
		&topFeaturesRaw,
		&item.ExplanationText,
		&item.TargetEntityType,
		&item.TargetEntityID,
		&item.ForecastStart,
		&item.ForecastEnd,
		&item.OutcomeObserved,
		&item.OutcomeValue,
		&item.AccuracyScore,
		&item.PredictionLogID,
		&item.CreatedAt,
		&item.EvaluatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if len(confidenceRaw) > 0 {
		if err := json.Unmarshal(confidenceRaw, &item.ConfidenceInterval); err != nil {
			return nil, fmt.Errorf("decode prediction confidence interval: %w", err)
		}
	}
	if len(topFeaturesRaw) > 0 {
		if err := json.Unmarshal(topFeaturesRaw, &item.TopFeatures); err != nil {
			return nil, fmt.Errorf("decode prediction top features: %w", err)
		}
	}
	return &item, nil
}

func scanPredictionModel(row interface{ Scan(...any) error }) (*predictmodel.PredictionModel, error) {
	var item predictmodel.PredictionModel
	if err := row.Scan(
		&item.ID,
		&item.ModelType,
		&item.Version,
		&item.ModelArtifactPath,
		&item.ModelFramework,
		&item.BacktestAccuracy,
		&item.BacktestPrecision,
		&item.BacktestRecall,
		&item.BacktestF1,
		&item.BacktestMAPE,
		&item.FeatureCount,
		&item.TrainingSamples,
		&item.TrainingDurationSeconds,
		&item.Status,
		&item.Active,
		&item.LastDriftCheck,
		&item.DriftScore,
		&item.CreatedAt,
		&item.ActivatedAt,
		&item.DeprecatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}
