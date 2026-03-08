package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type ListModelsParams struct {
	Suite  string
	Type   string
	Status string
	Page   int
	PerPage int
}

type ModelRegistryRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewModelRegistryRepository(db *pgxpool.Pool, logger zerolog.Logger) *ModelRegistryRepository {
	return &ModelRegistryRepository{db: db, logger: loggerWithRepo(logger, "ai_model_registry")}
}

func (r *ModelRegistryRepository) CreateModel(ctx context.Context, item *aigovmodel.RegisteredModel) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO ai_models (
			id, tenant_id, name, slug, description, model_type, suite, owner_user_id, owner_team,
			risk_tier, status, tags, metadata, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)`,
		item.ID, item.TenantID, item.Name, item.Slug, item.Description, item.ModelType, item.Suite,
		item.OwnerUserID, item.OwnerTeam, item.RiskTier, item.Status, item.Tags, item.Metadata,
		item.CreatedBy, item.CreatedAt, item.UpdatedAt,
	)
	if pgErr := new(pgconn.PgError); err != nil && strings.Contains(err.Error(), "idx_ai_models_tenant_slug_unique") {
		return fmt.Errorf("model slug already exists: %w", err)
	} else if err != nil && errorsAs(err, &pgErr) {
		return fmt.Errorf("insert ai model: %w", err)
	}
	return err
}

func (r *ModelRegistryRepository) ListModels(ctx context.Context, tenantID uuid.UUID, params ListModelsParams) ([]aigovmodel.RegisteredModel, int, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 {
		params.PerPage = 25
	}
	args := []any{tenantID}
	where := []string{"tenant_id = $1", "deleted_at IS NULL"}
	if params.Suite != "" {
		args = append(args, params.Suite)
		where = append(where, fmt.Sprintf("suite = $%d", len(args)))
	}
	if params.Type != "" {
		args = append(args, params.Type)
		where = append(where, fmt.Sprintf("model_type = $%d", len(args)))
	}
	if params.Status != "" {
		args = append(args, params.Status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")
	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM ai_models WHERE "+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count ai models: %w", err)
	}
	args = append(args, params.PerPage, (params.Page-1)*params.PerPage)
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, name, slug, description, model_type, suite, owner_user_id, owner_team,
		       risk_tier, status, tags, metadata, created_by, created_at, updated_at, deleted_at
		FROM ai_models
		WHERE `+whereSQL+`
		ORDER BY name ASC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list ai models: %w", err)
	}
	defer rows.Close()

	items := make([]aigovmodel.RegisteredModel, 0)
	for rows.Next() {
		item, err := scanModel(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (r *ModelRegistryRepository) GetModel(ctx context.Context, tenantID, modelID uuid.UUID) (*aigovmodel.RegisteredModel, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, slug, description, model_type, suite, owner_user_id, owner_team,
		       risk_tier, status, tags, metadata, created_by, created_at, updated_at, deleted_at
		FROM ai_models
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, modelID,
	)
	item, err := scanModel(row)
	if err != nil {
		return nil, rowNotFound(err)
	}
	return item, nil
}

func (r *ModelRegistryRepository) GetModelBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*aigovmodel.RegisteredModel, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, slug, description, model_type, suite, owner_user_id, owner_team,
		       risk_tier, status, tags, metadata, created_by, created_at, updated_at, deleted_at
		FROM ai_models
		WHERE tenant_id = $1 AND slug = $2 AND deleted_at IS NULL`,
		tenantID, slug,
	)
	item, err := scanModel(row)
	if err != nil {
		return nil, rowNotFound(err)
	}
	return item, nil
}

func (r *ModelRegistryRepository) UpdateModelMetadata(ctx context.Context, item *aigovmodel.RegisteredModel) error {
	cmd, err := r.db.Exec(ctx, `
		UPDATE ai_models
		SET name = $3,
		    description = $4,
		    owner_user_id = $5,
		    owner_team = $6,
		    risk_tier = $7,
		    status = $8,
		    tags = $9,
		    metadata = $10,
		    updated_at = $11
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		item.TenantID, item.ID, item.Name, item.Description, item.OwnerUserID, item.OwnerTeam,
		item.RiskTier, item.Status, item.Tags, item.Metadata, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update ai model: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ModelRegistryRepository) CreateVersion(ctx context.Context, item *aigovmodel.ModelVersion) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO ai_model_versions (
			id, tenant_id, model_id, version_number, status, description, artifact_type, artifact_config,
			artifact_hash, explainability_type, explanation_template, training_data_desc, training_data_hash,
			training_metrics, prediction_count, feedback_count, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
		)`,
		item.ID, item.TenantID, item.ModelID, item.VersionNumber, item.Status, item.Description,
		item.ArtifactType, item.ArtifactConfig, item.ArtifactHash, item.ExplainabilityType,
		item.ExplanationTemplate, item.TrainingDataDesc, item.TrainingDataHash, item.TrainingMetrics,
		item.PredictionCount, item.FeedbackCount, item.CreatedBy, item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert ai model version: %w", err)
	}
	return nil
}

func (r *ModelRegistryRepository) NextVersionNumber(ctx context.Context, tenantID, modelID uuid.UUID) (int, error) {
	var current int
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(MAX(version_number), 0)
		FROM ai_model_versions
		WHERE tenant_id = $1 AND model_id = $2`,
		tenantID, modelID,
	).Scan(&current)
	if err != nil {
		return 0, fmt.Errorf("max ai model version: %w", err)
	}
	return current + 1, nil
}

func (r *ModelRegistryRepository) ListVersions(ctx context.Context, tenantID, modelID uuid.UUID) ([]aigovmodel.ModelVersion, error) {
	rows, err := r.db.Query(ctx, `
		SELECT v.id, v.tenant_id, v.model_id, m.slug, m.name, m.model_type, m.suite, m.risk_tier,
		       v.version_number, v.status, v.description, v.artifact_type, v.artifact_config,
		       v.artifact_hash, v.explainability_type, v.explanation_template, v.training_data_desc,
		       v.training_data_hash, v.training_metrics, v.prediction_count, v.avg_latency_ms,
		       v.avg_confidence, v.accuracy_metric, v.false_positive_rate, v.false_negative_rate,
		       v.feedback_count, v.promoted_to_staging_at, v.promoted_to_shadow_at,
		       v.promoted_to_production_at, v.promoted_by, v.retired_at, v.retired_by,
		       v.retirement_reason, v.rolled_back_at, v.rolled_back_by, v.rollback_reason,
		       v.replaced_version_id, v.created_by, v.created_at, v.updated_at
		FROM ai_model_versions v
		JOIN ai_models m ON m.id = v.model_id
		WHERE v.tenant_id = $1 AND v.model_id = $2
		ORDER BY v.version_number DESC`,
		tenantID, modelID,
	)
	if err != nil {
		return nil, fmt.Errorf("list ai model versions: %w", err)
	}
	defer rows.Close()

	items := make([]aigovmodel.ModelVersion, 0)
	for rows.Next() {
		item, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *ModelRegistryRepository) GetVersion(ctx context.Context, tenantID, modelID, versionID uuid.UUID) (*aigovmodel.ModelVersion, error) {
	row := r.db.QueryRow(ctx, `
		SELECT v.id, v.tenant_id, v.model_id, m.slug, m.name, m.model_type, m.suite, m.risk_tier,
		       v.version_number, v.status, v.description, v.artifact_type, v.artifact_config,
		       v.artifact_hash, v.explainability_type, v.explanation_template, v.training_data_desc,
		       v.training_data_hash, v.training_metrics, v.prediction_count, v.avg_latency_ms,
		       v.avg_confidence, v.accuracy_metric, v.false_positive_rate, v.false_negative_rate,
		       v.feedback_count, v.promoted_to_staging_at, v.promoted_to_shadow_at,
		       v.promoted_to_production_at, v.promoted_by, v.retired_at, v.retired_by,
		       v.retirement_reason, v.rolled_back_at, v.rolled_back_by, v.rollback_reason,
		       v.replaced_version_id, v.created_by, v.created_at, v.updated_at
		FROM ai_model_versions v
		JOIN ai_models m ON m.id = v.model_id
		WHERE v.tenant_id = $1 AND v.model_id = $2 AND v.id = $3`,
		tenantID, modelID, versionID,
	)
	item, err := scanVersion(row)
	if err != nil {
		return nil, rowNotFound(err)
	}
	return item, nil
}

func (r *ModelRegistryRepository) GetProductionVersionBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*aigovmodel.ModelVersion, error) {
	return r.getVersionBySlugAndStatus(ctx, tenantID, slug, aigovmodel.VersionStatusProduction)
}

func (r *ModelRegistryRepository) GetShadowVersionBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*aigovmodel.ModelVersion, error) {
	return r.getVersionBySlugAndStatus(ctx, tenantID, slug, aigovmodel.VersionStatusShadow)
}

func (r *ModelRegistryRepository) GetCurrentProductionVersion(ctx context.Context, tenantID, modelID uuid.UUID) (*aigovmodel.ModelVersion, error) {
	row := r.db.QueryRow(ctx, `
		SELECT v.id, v.tenant_id, v.model_id, m.slug, m.name, m.model_type, m.suite, m.risk_tier,
		       v.version_number, v.status, v.description, v.artifact_type, v.artifact_config,
		       v.artifact_hash, v.explainability_type, v.explanation_template, v.training_data_desc,
		       v.training_data_hash, v.training_metrics, v.prediction_count, v.avg_latency_ms,
		       v.avg_confidence, v.accuracy_metric, v.false_positive_rate, v.false_negative_rate,
		       v.feedback_count, v.promoted_to_staging_at, v.promoted_to_shadow_at,
		       v.promoted_to_production_at, v.promoted_by, v.retired_at, v.retired_by,
		       v.retirement_reason, v.rolled_back_at, v.rolled_back_by, v.rollback_reason,
		       v.replaced_version_id, v.created_by, v.created_at, v.updated_at
		FROM ai_model_versions v
		JOIN ai_models m ON m.id = v.model_id
		WHERE v.tenant_id = $1 AND v.model_id = $2 AND v.status = 'production'
		ORDER BY v.version_number DESC
		LIMIT 1`,
		tenantID, modelID,
	)
	item, err := scanVersion(row)
	if err != nil {
		return nil, rowNotFound(err)
	}
	return item, nil
}

func (r *ModelRegistryRepository) ListProductionVersions(ctx context.Context) ([]aigovmodel.ModelVersion, error) {
	rows, err := r.db.Query(ctx, `
		SELECT v.id, v.tenant_id, v.model_id, m.slug, m.name, m.model_type, m.suite, m.risk_tier,
		       v.version_number, v.status, v.description, v.artifact_type, v.artifact_config,
		       v.artifact_hash, v.explainability_type, v.explanation_template, v.training_data_desc,
		       v.training_data_hash, v.training_metrics, v.prediction_count, v.avg_latency_ms,
		       v.avg_confidence, v.accuracy_metric, v.false_positive_rate, v.false_negative_rate,
		       v.feedback_count, v.promoted_to_staging_at, v.promoted_to_shadow_at,
		       v.promoted_to_production_at, v.promoted_by, v.retired_at, v.retired_by,
		       v.retirement_reason, v.rolled_back_at, v.rolled_back_by, v.rollback_reason,
		       v.replaced_version_id, v.created_by, v.created_at, v.updated_at
		FROM ai_model_versions v
		JOIN ai_models m ON m.id = v.model_id
		WHERE v.status = 'production' AND m.deleted_at IS NULL`,
	)
	if err != nil {
		return nil, fmt.Errorf("list production ai model versions: %w", err)
	}
	defer rows.Close()
	items := make([]aigovmodel.ModelVersion, 0)
	for rows.Next() {
		item, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *ModelRegistryRepository) UpdateVersionStatus(ctx context.Context, version *aigovmodel.ModelVersion) error {
	cmd, err := r.db.Exec(ctx, `
		UPDATE ai_model_versions
		SET status = $4,
		    prediction_count = $5,
		    avg_latency_ms = $6,
		    avg_confidence = $7,
		    accuracy_metric = $8,
		    false_positive_rate = $9,
		    false_negative_rate = $10,
		    feedback_count = $11,
		    promoted_to_staging_at = $12,
		    promoted_to_shadow_at = $13,
		    promoted_to_production_at = $14,
		    promoted_by = $15,
		    retired_at = $16,
		    retired_by = $17,
		    retirement_reason = $18,
		    rolled_back_at = $19,
		    rolled_back_by = $20,
		    rollback_reason = $21,
		    replaced_version_id = $22,
		    updated_at = $23
		WHERE tenant_id = $1 AND model_id = $2 AND id = $3`,
		version.TenantID, version.ModelID, version.ID, version.Status, version.PredictionCount,
		version.AvgLatencyMS, version.AvgConfidence, version.AccuracyMetric, version.FalsePositiveRate,
		version.FalseNegativeRate, version.FeedbackCount, version.PromotedToStagingAt,
		version.PromotedToShadowAt, version.PromotedToProductionAt, version.PromotedBy, version.RetiredAt,
		version.RetiredBy, version.RetirementReason, version.RolledBackAt, version.RolledBackBy,
		version.RollbackReason, version.ReplacedVersionID, version.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update ai model version: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ModelRegistryRepository) UpdateVersionAggregates(ctx context.Context, tenantID, versionID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE ai_model_versions v
		SET prediction_count = stats.prediction_count,
		    avg_latency_ms = stats.avg_latency_ms,
		    avg_confidence = stats.avg_confidence,
		    feedback_count = stats.feedback_count,
		    accuracy_metric = stats.accuracy_metric,
		    false_positive_rate = stats.false_positive_rate,
		    false_negative_rate = stats.false_negative_rate,
		    updated_at = now()
		FROM (
			SELECT
				model_version_id,
				COUNT(*)::bigint AS prediction_count,
				AVG(latency_ms)::decimal(10,2) AS avg_latency_ms,
				AVG(confidence)::decimal(5,4) AS avg_confidence,
				COUNT(*) FILTER (WHERE feedback_correct IS NOT NULL)::int AS feedback_count,
				AVG(CASE WHEN feedback_correct IS NULL THEN NULL WHEN feedback_correct THEN 1 ELSE 0 END)::decimal(5,4) AS accuracy_metric,
				AVG(CASE WHEN feedback_correct IS NULL THEN NULL WHEN feedback_correct THEN 0 ELSE 1 END)::decimal(5,4) AS false_positive_rate,
				AVG(CASE WHEN feedback_correct IS NULL THEN NULL WHEN feedback_correct THEN 0 ELSE 1 END)::decimal(5,4) AS false_negative_rate
			FROM ai_prediction_logs
			WHERE tenant_id = $1 AND model_version_id = $2
			GROUP BY model_version_id
		) stats
		WHERE v.tenant_id = $1 AND v.id = $2 AND v.id = stats.model_version_id`,
		tenantID, versionID,
	)
	if err != nil {
		return fmt.Errorf("update ai model version aggregates: %w", err)
	}
	return nil
}

func (r *ModelRegistryRepository) LifecycleHistory(ctx context.Context, tenantID, modelID uuid.UUID) ([]aigovmodel.LifecycleHistoryEntry, error) {
	versions, err := r.ListVersions(ctx, tenantID, modelID)
	if err != nil {
		return nil, err
	}
	history := make([]aigovmodel.LifecycleHistoryEntry, 0, len(versions)*3)
	for _, version := range versions {
		history = appendHistory(history, version, version.PromotedToStagingAt, nil, aigovmodel.VersionStatusStaging, version.PromotedBy, "")
		history = appendHistory(history, version, version.PromotedToShadowAt, ptrStatus(aigovmodel.VersionStatusStaging), aigovmodel.VersionStatusShadow, version.PromotedBy, "")
		history = appendHistory(history, version, version.PromotedToProductionAt, ptrStatus(aigovmodel.VersionStatusShadow), aigovmodel.VersionStatusProduction, version.PromotedBy, "")
		history = appendHistory(history, version, version.RetiredAt, ptrStatus(aigovmodel.VersionStatusProduction), aigovmodel.VersionStatusRetired, version.RetiredBy, derefString(version.RetirementReason))
		history = appendHistory(history, version, version.RolledBackAt, ptrStatus(aigovmodel.VersionStatusProduction), aigovmodel.VersionStatusRolledBack, version.RolledBackBy, derefString(version.RollbackReason))
	}
	sort.Slice(history, func(i, j int) bool {
		return history[i].ChangedAt.After(history[j].ChangedAt)
	})
	return history, nil
}

func (r *ModelRegistryRepository) getVersionBySlugAndStatus(ctx context.Context, tenantID uuid.UUID, slug string, status aigovmodel.VersionStatus) (*aigovmodel.ModelVersion, error) {
	row := r.db.QueryRow(ctx, `
		SELECT v.id, v.tenant_id, v.model_id, m.slug, m.name, m.model_type, m.suite, m.risk_tier,
		       v.version_number, v.status, v.description, v.artifact_type, v.artifact_config,
		       v.artifact_hash, v.explainability_type, v.explanation_template, v.training_data_desc,
		       v.training_data_hash, v.training_metrics, v.prediction_count, v.avg_latency_ms,
		       v.avg_confidence, v.accuracy_metric, v.false_positive_rate, v.false_negative_rate,
		       v.feedback_count, v.promoted_to_staging_at, v.promoted_to_shadow_at,
		       v.promoted_to_production_at, v.promoted_by, v.retired_at, v.retired_by,
		       v.retirement_reason, v.rolled_back_at, v.rolled_back_by, v.rollback_reason,
		       v.replaced_version_id, v.created_by, v.created_at, v.updated_at
		FROM ai_model_versions v
		JOIN ai_models m ON m.id = v.model_id
		WHERE v.tenant_id = $1 AND m.slug = $2 AND v.status = $3
		ORDER BY v.version_number DESC
		LIMIT 1`,
		tenantID, slug, status,
	)
	item, err := scanVersion(row)
	if err != nil {
		return nil, rowNotFound(err)
	}
	return item, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanModel(row scannable) (*aigovmodel.RegisteredModel, error) {
	item := &aigovmodel.RegisteredModel{}
	var metadata []byte
	if err := row.Scan(
		&item.ID, &item.TenantID, &item.Name, &item.Slug, &item.Description, &item.ModelType,
		&item.Suite, &item.OwnerUserID, &item.OwnerTeam, &item.RiskTier, &item.Status,
		&item.Tags, &metadata, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt,
	); err != nil {
		return nil, err
	}
	item.Metadata = nullJSON(metadata, "{}")
	if item.Tags == nil {
		item.Tags = []string{}
	}
	return item, nil
}

func scanVersion(row scannable) (*aigovmodel.ModelVersion, error) {
	item := &aigovmodel.ModelVersion{}
	var (
		artifactConfig []byte
		trainingMetrics []byte
		explanationTemplate *string
		trainingDataDesc *string
		trainingDataHash *string
		promotedToStaging *time.Time
		promotedToShadow *time.Time
		promotedToProduction *time.Time
		promotedBy *uuid.UUID
		retiredAt *time.Time
		retiredBy *uuid.UUID
		retirementReason *string
		rolledBackAt *time.Time
		rolledBackBy *uuid.UUID
		rollbackReason *string
		replacedVersionID *uuid.UUID
	)
	if err := row.Scan(
		&item.ID, &item.TenantID, &item.ModelID, &item.ModelSlug, &item.ModelName, &item.ModelType,
		&item.ModelSuite, &item.ModelRiskTier, &item.VersionNumber, &item.Status, &item.Description,
		&item.ArtifactType, &artifactConfig, &item.ArtifactHash, &item.ExplainabilityType,
		&explanationTemplate, &trainingDataDesc, &trainingDataHash, &trainingMetrics,
		&item.PredictionCount, &item.AvgLatencyMS, &item.AvgConfidence, &item.AccuracyMetric,
		&item.FalsePositiveRate, &item.FalseNegativeRate, &item.FeedbackCount, &promotedToStaging,
		&promotedToShadow, &promotedToProduction, &promotedBy, &retiredAt, &retiredBy,
		&retirementReason, &rolledBackAt, &rolledBackBy, &rollbackReason, &replacedVersionID,
		&item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.ArtifactConfig = nullJSON(artifactConfig, "{}")
	item.TrainingMetrics = nullJSON(trainingMetrics, "{}")
	item.ExplanationTemplate = explanationTemplate
	item.TrainingDataDesc = trainingDataDesc
	item.TrainingDataHash = trainingDataHash
	item.PromotedToStagingAt = ptrTime(promotedToStaging)
	item.PromotedToShadowAt = ptrTime(promotedToShadow)
	item.PromotedToProductionAt = ptrTime(promotedToProduction)
	item.PromotedBy = ptrUUID(promotedBy)
	item.RetiredAt = ptrTime(retiredAt)
	item.RetiredBy = ptrUUID(retiredBy)
	item.RetirementReason = retirementReason
	item.RolledBackAt = ptrTime(rolledBackAt)
	item.RolledBackBy = ptrUUID(rolledBackBy)
	item.RollbackReason = rollbackReason
	item.ReplacedVersionID = ptrUUID(replacedVersionID)
	return item, nil
}

func appendHistory(history []aigovmodel.LifecycleHistoryEntry, version aigovmodel.ModelVersion, changedAt *time.Time, from *aigovmodel.VersionStatus, to aigovmodel.VersionStatus, changedBy *uuid.UUID, reason string) []aigovmodel.LifecycleHistoryEntry {
	if changedAt == nil {
		return history
	}
	return append(history, aigovmodel.LifecycleHistoryEntry{
		VersionID:     version.ID,
		VersionNumber: version.VersionNumber,
		FromStatus:    from,
		ToStatus:      to,
		ChangedBy:     changedBy,
		Reason:        reason,
		ChangedAt:     changedAt.UTC(),
	})
}

func ptrStatus(value aigovmodel.VersionStatus) *aigovmodel.VersionStatus {
	v := value
	return &v
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func errorsAs(err error, target any) bool {
	return errors.As(err, target)
}
