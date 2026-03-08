package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	aigovdto "github.com/clario360/platform/internal/aigovernance/dto"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type PredictionLogRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

type RecentModelStat struct {
	ModelID        uuid.UUID
	Predictions24h int64
	AvgConfidence  *float64
}

func NewPredictionLogRepository(db *pgxpool.Pool, logger zerolog.Logger) *PredictionLogRepository {
	return &PredictionLogRepository{db: db, logger: loggerWithRepo(logger, "ai_prediction_log")}
}

func (r *PredictionLogRepository) InsertBatch(ctx context.Context, items []*aigovmodel.PredictionLog) error {
	if len(items) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, item := range items {
		batch.Queue(`
			INSERT INTO ai_prediction_logs (
				id, tenant_id, model_id, model_version_id, input_hash, input_summary, prediction, confidence,
				explanation_structured, explanation_text, explanation_factors, suite, use_case, entity_type,
				entity_id, is_shadow, shadow_production_version_id, shadow_divergence, feedback_correct,
				feedback_by, feedback_at, feedback_notes, feedback_corrected_output, latency_ms, created_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18,
				$19, $20, $21, $22, $23, $24, $25
			)`,
			item.ID, item.TenantID, item.ModelID, item.ModelVersionID, item.InputHash, item.InputSummary,
			item.Prediction, item.Confidence, item.ExplanationStructured, item.ExplanationText,
			item.ExplanationFactors, item.Suite, item.UseCase, item.EntityType, item.EntityID, item.IsShadow,
			item.ShadowProductionVersionID, item.ShadowDivergence, item.FeedbackCorrect, item.FeedbackBy,
			item.FeedbackAt, item.FeedbackNotes, item.FeedbackCorrectedOutput, item.LatencyMS, item.CreatedAt,
		)
	}
	br := r.db.SendBatch(ctx, batch)
	defer br.Close()
	for range items {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("insert ai prediction log batch: %w", err)
		}
	}
	return nil
}

func (r *PredictionLogRepository) Get(ctx context.Context, tenantID, predictionID uuid.UUID) (*aigovmodel.PredictionLog, error) {
	row := r.db.QueryRow(ctx, `
		SELECT l.id, l.tenant_id, l.model_id, l.model_version_id, m.slug, v.version_number, l.input_hash,
		       l.input_summary, l.prediction, l.confidence, l.explanation_structured, l.explanation_text,
		       l.explanation_factors, l.suite, l.use_case, COALESCE(l.entity_type, ''), l.entity_id,
		       l.is_shadow, l.shadow_production_version_id, l.shadow_divergence, l.feedback_correct,
		       l.feedback_by, l.feedback_at, l.feedback_notes, l.feedback_corrected_output, l.latency_ms,
		       l.created_at
		FROM ai_prediction_logs l
		JOIN ai_models m ON m.id = l.model_id
		JOIN ai_model_versions v ON v.id = l.model_version_id
		WHERE l.tenant_id = $1 AND l.id = $2
		ORDER BY l.created_at DESC
		LIMIT 1`,
		tenantID, predictionID,
	)
	item, err := scanPrediction(row)
	if err != nil {
		return nil, rowNotFound(err)
	}
	return item, nil
}

func (r *PredictionLogRepository) List(ctx context.Context, tenantID uuid.UUID, params aigovdto.PredictionQuery) ([]aigovmodel.PredictionLog, int, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 {
		params.PerPage = 25
	}
	args := []any{tenantID}
	where := []string{"l.tenant_id = $1"}
	if params.ModelID != nil {
		args = append(args, *params.ModelID)
		where = append(where, fmt.Sprintf("l.model_id = $%d", len(args)))
	}
	if params.Suite != "" {
		args = append(args, params.Suite)
		where = append(where, fmt.Sprintf("l.suite = $%d", len(args)))
	}
	if params.UseCase != "" {
		args = append(args, params.UseCase)
		where = append(where, fmt.Sprintf("l.use_case = $%d", len(args)))
	}
	if params.EntityType != "" {
		args = append(args, params.EntityType)
		where = append(where, fmt.Sprintf("l.entity_type = $%d", len(args)))
	}
	if params.IsShadow != nil {
		args = append(args, *params.IsShadow)
		where = append(where, fmt.Sprintf("l.is_shadow = $%d", len(args)))
	}
	if params.Search != "" {
		args = append(args, "%"+strings.TrimSpace(params.Search)+"%")
		where = append(where, fmt.Sprintf("(l.explanation_text ILIKE $%d OR l.explanation_structured::text ILIKE $%d)", len(args), len(args)))
	}
	whereSQL := strings.Join(where, " AND ")
	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM ai_prediction_logs l WHERE "+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count ai prediction logs: %w", err)
	}
	args = append(args, params.PerPage, (params.Page-1)*params.PerPage)
	rows, err := r.db.Query(ctx, `
		SELECT l.id, l.tenant_id, l.model_id, l.model_version_id, m.slug, v.version_number, l.input_hash,
		       l.input_summary, l.prediction, l.confidence, l.explanation_structured, l.explanation_text,
		       l.explanation_factors, l.suite, l.use_case, COALESCE(l.entity_type, ''), l.entity_id,
		       l.is_shadow, l.shadow_production_version_id, l.shadow_divergence, l.feedback_correct,
		       l.feedback_by, l.feedback_at, l.feedback_notes, l.feedback_corrected_output, l.latency_ms,
		       l.created_at
		FROM ai_prediction_logs l
		JOIN ai_models m ON m.id = l.model_id
		JOIN ai_model_versions v ON v.id = l.model_version_id
		WHERE `+whereSQL+`
		ORDER BY l.created_at DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list ai prediction logs: %w", err)
	}
	defer rows.Close()
	items := make([]aigovmodel.PredictionLog, 0)
	for rows.Next() {
		item, err := scanPrediction(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (r *PredictionLogRepository) SubmitFeedback(ctx context.Context, tenantID, predictionID, userID uuid.UUID, req aigovdto.PredictionFeedbackRequest) error {
	now := time.Now().UTC()
	cmd, err := r.db.Exec(ctx, `
		UPDATE ai_prediction_logs
		SET feedback_correct = $3,
		    feedback_by = $4,
		    feedback_at = $5,
		    feedback_notes = $6,
		    feedback_corrected_output = $7
		WHERE tenant_id = $1 AND id = $2 AND created_at = (
			SELECT created_at FROM ai_prediction_logs WHERE tenant_id = $1 AND id = $2 ORDER BY created_at DESC LIMIT 1
		)`,
		tenantID, predictionID, req.Correct, userID, now, nullString(req.Notes), req.CorrectedOutput,
	)
	if err != nil {
		return fmt.Errorf("update ai prediction feedback: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PredictionLogRepository) Stats(ctx context.Context, tenantID uuid.UUID) ([]aigovmodel.PredictionStats, error) {
	rows, err := r.db.Query(ctx, `
		SELECT l.model_id, m.slug, l.suite, l.use_case,
		       COUNT(*)::bigint AS total,
		       COUNT(*) FILTER (WHERE l.is_shadow)::bigint AS shadow_total,
		       AVG(l.confidence)::float8 AS avg_confidence,
		       AVG(l.latency_ms)::float8 AS avg_latency_ms,
		       COUNT(*) FILTER (WHERE l.feedback_correct = true)::bigint AS correct_feedback,
		       COUNT(*) FILTER (WHERE l.feedback_correct = false)::bigint AS wrong_feedback
		FROM ai_prediction_logs l
		JOIN ai_models m ON m.id = l.model_id
		WHERE l.tenant_id = $1
		GROUP BY l.model_id, m.slug, l.suite, l.use_case
		ORDER BY total DESC, m.slug ASC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list ai prediction stats: %w", err)
	}
	defer rows.Close()
	stats := make([]aigovmodel.PredictionStats, 0)
	for rows.Next() {
		var item aigovmodel.PredictionStats
		if err := rows.Scan(&item.ModelID, &item.ModelSlug, &item.Suite, &item.UseCase, &item.Total, &item.ShadowTotal, &item.AvgConfidence, &item.AvgLatencyMS, &item.CorrectFeedback, &item.WrongFeedback); err != nil {
			return nil, fmt.Errorf("scan ai prediction stats: %w", err)
		}
		stats = append(stats, item)
	}
	return stats, rows.Err()
}

func (r *PredictionLogRepository) SearchExplanations(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]aigovmodel.ExplanationSearchResult, error) {
	if limit <= 0 {
		limit = 25
	}
	search := "%" + strings.TrimSpace(query) + "%"
	rows, err := r.db.Query(ctx, `
		SELECT l.id, m.slug, l.explanation_text, l.confidence, l.use_case, COALESCE(l.entity_type, '')
		FROM ai_prediction_logs l
		JOIN ai_models m ON m.id = l.model_id
		WHERE l.tenant_id = $1
		  AND ($2 = '%%' OR l.explanation_text ILIKE $2 OR l.explanation_factors::text ILIKE $2)
		ORDER BY l.created_at DESC
		LIMIT $3`,
		tenantID, search, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search ai explanations: %w", err)
	}
	defer rows.Close()
	results := make([]aigovmodel.ExplanationSearchResult, 0)
	for rows.Next() {
		var item aigovmodel.ExplanationSearchResult
		if err := rows.Scan(&item.PredictionID, &item.ModelSlug, &item.ExplanationText, &item.Confidence, &item.UseCase, &item.EntityType); err != nil {
			return nil, fmt.Errorf("scan ai explanation search: %w", err)
		}
		item.MatchedHighlight = item.ExplanationText
		results = append(results, item)
	}
	return results, rows.Err()
}

func (r *PredictionLogRepository) ListByVersionAndWindow(ctx context.Context, tenantID, versionID uuid.UUID, start, end time.Time, isShadow *bool) ([]aigovmodel.PredictionLog, error) {
	args := []any{tenantID, versionID, start, end}
	where := []string{"tenant_id = $1", "model_version_id = $2", "created_at >= $3", "created_at <= $4"}
	if isShadow != nil {
		args = append(args, *isShadow)
		where = append(where, fmt.Sprintf("is_shadow = $%d", len(args)))
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, model_id, model_version_id, '' AS model_slug, 0 AS version_number, input_hash,
		       input_summary, prediction, confidence, explanation_structured, explanation_text,
		       explanation_factors, suite, use_case, COALESCE(entity_type, ''), entity_id,
		       is_shadow, shadow_production_version_id, shadow_divergence, feedback_correct,
		       feedback_by, feedback_at, feedback_notes, feedback_corrected_output, latency_ms,
		       created_at
		FROM ai_prediction_logs
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY created_at ASC`, args...)
	if err != nil {
		return nil, fmt.Errorf("list ai logs by version and window: %w", err)
	}
	defer rows.Close()
	items := make([]aigovmodel.PredictionLog, 0)
	for rows.Next() {
		item, err := scanPrediction(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *PredictionLogRepository) PerformanceSeries(ctx context.Context, tenantID, modelID uuid.UUID, since time.Time) ([]aigovmodel.PerformancePoint, error) {
	rows, err := r.db.Query(ctx, `
		SELECT date_trunc('day', created_at) AS period_start,
		       COUNT(*)::bigint AS volume,
		       AVG(latency_ms)::float8 AS avg_latency_ms,
		       AVG(CASE WHEN feedback_correct IS NULL THEN NULL WHEN feedback_correct THEN 1 ELSE 0 END)::float8 AS accuracy
		FROM ai_prediction_logs
		WHERE tenant_id = $1 AND model_id = $2 AND created_at >= $3
		GROUP BY 1
		ORDER BY 1 ASC`,
		tenantID, modelID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("load ai performance series: %w", err)
	}
	defer rows.Close()
	points := make([]aigovmodel.PerformancePoint, 0)
	for rows.Next() {
		var item aigovmodel.PerformancePoint
		if err := rows.Scan(&item.PeriodStart, &item.Volume, &item.AvgLatency, &item.Accuracy); err != nil {
			return nil, fmt.Errorf("scan ai performance point: %w", err)
		}
		points = append(points, item)
	}
	return points, rows.Err()
}

func (r *PredictionLogRepository) CountSince(ctx context.Context, tenantID uuid.UUID, since time.Time) (int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)::bigint
		FROM ai_prediction_logs
		WHERE tenant_id = $1 AND created_at >= $2`,
		tenantID, since,
	).Scan(&total); err != nil {
		return 0, fmt.Errorf("count ai predictions since window: %w", err)
	}
	return total, nil
}

func (r *PredictionLogRepository) ListDivergences(ctx context.Context, tenantID, modelID uuid.UUID, page, perPage int) ([]aigovmodel.PredictionLog, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 25
	}
	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM ai_prediction_logs
		WHERE tenant_id = $1
		  AND model_id = $2
		  AND is_shadow = true
		  AND shadow_divergence IS NOT NULL
		  AND shadow_divergence::text NOT IN ('{}', 'null')`,
		tenantID, modelID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count ai divergences: %w", err)
	}
	rows, err := r.db.Query(ctx, `
		SELECT l.id, l.tenant_id, l.model_id, l.model_version_id, m.slug, v.version_number, l.input_hash,
		       l.input_summary, l.prediction, l.confidence, l.explanation_structured, l.explanation_text,
		       l.explanation_factors, l.suite, l.use_case, COALESCE(l.entity_type, ''), l.entity_id,
		       l.is_shadow, l.shadow_production_version_id, l.shadow_divergence, l.feedback_correct,
		       l.feedback_by, l.feedback_at, l.feedback_notes, l.feedback_corrected_output, l.latency_ms,
		       l.created_at
		FROM ai_prediction_logs l
		JOIN ai_models m ON m.id = l.model_id
		JOIN ai_model_versions v ON v.id = l.model_version_id
		WHERE l.tenant_id = $1
		  AND l.model_id = $2
		  AND l.is_shadow = true
		  AND l.shadow_divergence IS NOT NULL
		  AND l.shadow_divergence::text NOT IN ('{}', 'null')
		ORDER BY l.created_at DESC
		LIMIT $3 OFFSET $4`,
		tenantID, modelID, perPage, (page-1)*perPage,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list ai divergences: %w", err)
	}
	defer rows.Close()
	items := make([]aigovmodel.PredictionLog, 0, perPage)
	for rows.Next() {
		item, err := scanPrediction(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (r *PredictionLogRepository) RecentModelStats(ctx context.Context, tenantID uuid.UUID, since time.Time) (map[uuid.UUID]RecentModelStat, error) {
	rows, err := r.db.Query(ctx, `
		SELECT model_id, COUNT(*)::bigint AS total, AVG(confidence)::float8 AS avg_confidence
		FROM ai_prediction_logs
		WHERE tenant_id = $1 AND created_at >= $2
		GROUP BY model_id`,
		tenantID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("load ai recent model stats: %w", err)
	}
	defer rows.Close()
	out := make(map[uuid.UUID]RecentModelStat)
	for rows.Next() {
		var item RecentModelStat
		if err := rows.Scan(&item.ModelID, &item.Predictions24h, &item.AvgConfidence); err != nil {
			return nil, fmt.Errorf("scan ai recent model stats: %w", err)
		}
		out[item.ModelID] = item
	}
	return out, rows.Err()
}

type predictionScannable interface {
	Scan(dest ...any) error
}

func scanPrediction(row predictionScannable) (*aigovmodel.PredictionLog, error) {
	item := &aigovmodel.PredictionLog{}
	var (
		inputSummary          []byte
		prediction            []byte
		explanationStructured []byte
		explanationFactors    []byte
		shadowDivergence      []byte
		correctedOutput       []byte
	)
	if err := row.Scan(
		&item.ID, &item.TenantID, &item.ModelID, &item.ModelVersionID, &item.ModelSlug, &item.ModelVersionNumber, &item.InputHash,
		&inputSummary, &prediction, &item.Confidence, &explanationStructured, &item.ExplanationText,
		&explanationFactors, &item.Suite, &item.UseCase, &item.EntityType, &item.EntityID, &item.IsShadow,
		&item.ShadowProductionVersionID, &shadowDivergence, &item.FeedbackCorrect, &item.FeedbackBy,
		&item.FeedbackAt, &item.FeedbackNotes, &correctedOutput, &item.LatencyMS, &item.CreatedAt,
	); err != nil {
		return nil, err
	}
	item.InputSummary = nullJSON(inputSummary, "{}")
	item.Prediction = nullJSON(prediction, "{}")
	item.ExplanationStructured = nullJSON(explanationStructured, "{}")
	item.ExplanationFactors = nullJSON(explanationFactors, "[]")
	item.ShadowDivergence = nullJSON(shadowDivergence, "{}")
	item.FeedbackCorrectedOutput = nullJSON(correctedOutput, "{}")
	return item, nil
}
