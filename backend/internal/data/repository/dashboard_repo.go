package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/database"
)

type DashboardRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewDashboardRepository(db *pgxpool.Pool, logger zerolog.Logger) *DashboardRepository {
	return &DashboardRepository{db: db, logger: logger}
}

func (r *DashboardRepository) RecentRuns(ctx context.Context, tenantID uuid.UUID, limit int) ([]dto.PipelineRunSummary, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.Query(ctx, `
		SELECT pr.id::text, pr.pipeline_id::text, p.name, pr.status, pr.started_at, pr.completed_at, pr.duration_ms
		FROM pipeline_runs pr
		JOIN pipelines p ON p.id = pr.pipeline_id
		WHERE pr.tenant_id = $1
		ORDER BY pr.started_at DESC
		LIMIT $2`,
		tenantID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query recent pipeline runs: %w", err)
	}
	defer rows.Close()

	items := make([]dto.PipelineRunSummary, 0, limit)
	for rows.Next() {
		var item dto.PipelineRunSummary
		if err := rows.Scan(&item.ID, &item.PipelineID, &item.PipelineName, &item.Status, &item.StartedAt, &item.CompletedAt, &item.DurationMs); err != nil {
			return nil, fmt.Errorf("scan recent pipeline run: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *DashboardRepository) PipelineTrend(ctx context.Context, tenantID uuid.UUID, days int) ([]dto.DailyMetric, error) {
	if days <= 0 {
		days = 30
	}
	rows, err := r.db.Query(ctx, `
		SELECT DATE_TRUNC('day', started_at) AS day, COUNT(*)::float8 AS value
		FROM pipeline_runs
		WHERE tenant_id = $1
		  AND started_at >= NOW() - ($2::int * INTERVAL '1 day')
		GROUP BY DATE_TRUNC('day', started_at)
		ORDER BY day ASC`,
		tenantID, days,
	)
	if err != nil {
		return nil, fmt.Errorf("query pipeline trend: %w", err)
	}
	defer rows.Close()

	points := make([]dto.DailyMetric, 0)
	for rows.Next() {
		var point dto.DailyMetric
		if err := rows.Scan(&point.Day, &point.Value); err != nil {
			return nil, fmt.Errorf("scan pipeline trend: %w", err)
		}
		points = append(points, point)
	}
	return points, rows.Err()
}

func (r *DashboardRepository) PipelineSuccessRate(ctx context.Context, tenantID uuid.UUID, days int) (float64, error) {
	if days <= 0 {
		days = 30
	}
	row := r.db.QueryRow(ctx, `
		SELECT COALESCE(
			ROUND(
				(COUNT(*) FILTER (WHERE status = 'completed')::numeric / NULLIF(COUNT(*)::numeric, 0)) * 100,
				2
			),
			0
		)
		FROM pipeline_runs
		WHERE tenant_id = $1
		  AND started_at >= NOW() - ($2::int * INTERVAL '1 day')`,
		tenantID, days,
	)
	var rate float64
	if err := row.Scan(&rate); err != nil {
		return 0, fmt.Errorf("query pipeline success rate: %w", err)
	}
	return rate, nil
}

func (r *DashboardRepository) FailedPipelines24h(ctx context.Context, tenantID uuid.UUID) (int, error) {
	row := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pipeline_runs
		WHERE tenant_id = $1
		  AND status = 'failed'
		  AND started_at >= NOW() - INTERVAL '24 hours'`,
		tenantID,
	)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("query failed pipelines 24h: %w", err)
	}
	return count, nil
}

func (r *DashboardRepository) SourceCountDelta(ctx context.Context, tenantID uuid.UUID) (int, error) {
	row := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '1 day')
			-
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '2 days' AND created_at < NOW() - INTERVAL '1 day')
		FROM data_sources
		WHERE tenant_id = $1 AND deleted_at IS NULL`,
		tenantID,
	)
	var delta int
	if err := row.Scan(&delta); err != nil {
		return 0, fmt.Errorf("query source count delta: %w", err)
	}
	return delta, nil
}

func (r *DashboardRepository) ContradictionsDelta(ctx context.Context, tenantID uuid.UUID) (int, error) {
	row := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '1 day')
			-
			COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '2 days' AND created_at < NOW() - INTERVAL '1 day')
		FROM contradictions
		WHERE tenant_id = $1`,
		tenantID,
	)
	var delta int
	if err := row.Scan(&delta); err != nil {
		return 0, fmt.Errorf("query contradiction delta: %w", err)
	}
	return delta, nil
}

func (r *DashboardRepository) TotalModels(ctx context.Context, tenantID uuid.UUID) (int, error) {
	row := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM data_models WHERE tenant_id = $1 AND deleted_at IS NULL`,
		tenantID,
	)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("query total models: %w", err)
	}
	return count, nil
}

func (r *DashboardRepository) DataDashboardQualityByModel(ctx context.Context, tenantID uuid.UUID, limit int) ([]dto.ModelQualitySummary, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.Query(ctx, `
		SELECT m.id::text, m.display_name, m.data_classification,
		       COALESCE(ROUND(AVG(COALESCE(qr.pass_rate, 0)), 2), 0) AS score
		FROM data_models m
		LEFT JOIN quality_results qr ON qr.model_id = m.id
		WHERE m.tenant_id = $1 AND m.deleted_at IS NULL
		GROUP BY m.id, m.display_name, m.data_classification
		ORDER BY score DESC, m.display_name ASC
		LIMIT $2`,
		tenantID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query quality by model: %w", err)
	}
	defer rows.Close()

	items := make([]dto.ModelQualitySummary, 0, limit)
	for rows.Next() {
		var item dto.ModelQualitySummary
		if err := rows.Scan(&item.ModelID, &item.ModelName, &item.Classification, &item.Score); err != nil {
			return nil, fmt.Errorf("scan quality by model: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *DashboardRepository) DataDashboardTopFailures(ctx context.Context, tenantID uuid.UUID, limit int) ([]dto.QualityFailureSummary, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.Query(ctx, `
		SELECT qr.id::text, qr.name, dm.id::text, dm.display_name, qr.severity, COALESCE(MAX(qres.records_failed), 0)
		FROM quality_rules qr
		JOIN data_models dm ON dm.id = qr.model_id
		LEFT JOIN quality_results qres ON qres.rule_id = qr.id
		WHERE qr.tenant_id = $1 AND qr.deleted_at IS NULL
		GROUP BY qr.id, qr.name, dm.id, dm.display_name, qr.severity
		ORDER BY COALESCE(MAX(qres.records_failed), 0) DESC, qr.name ASC
		LIMIT $2`,
		tenantID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query top quality failures: %w", err)
	}
	defer rows.Close()

	items := make([]dto.QualityFailureSummary, 0, limit)
	for rows.Next() {
		var item dto.QualityFailureSummary
		if err := rows.Scan(&item.RuleID, &item.RuleName, &item.ModelID, &item.ModelName, &item.Severity, &item.RecordsFailed); err != nil {
			return nil, fmt.Errorf("scan top quality failures: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *DashboardRepository) QualityTrend(ctx context.Context, tenantID uuid.UUID, days int) ([]dto.DailyMetric, error) {
	if days <= 0 {
		days = 30
	}
	rows, err := r.db.Query(ctx, `
		SELECT DATE_TRUNC('day', checked_at) AS day, COALESCE(ROUND(AVG(COALESCE(pass_rate, 0)), 2), 0)::float8
		FROM quality_results
		WHERE tenant_id = $1
		  AND checked_at >= NOW() - ($2::int * INTERVAL '1 day')
		GROUP BY DATE_TRUNC('day', checked_at)
		ORDER BY day ASC`,
		tenantID, days,
	)
	if err != nil {
		return nil, fmt.Errorf("query quality trend: %w", err)
	}
	defer rows.Close()

	items := make([]dto.DailyMetric, 0)
	for rows.Next() {
		var item dto.DailyMetric
		if err := rows.Scan(&item.Day, &item.Value); err != nil {
			return nil, fmt.Errorf("scan quality trend: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *DashboardRepository) ContradictionBreakdown(ctx context.Context, tenantID uuid.UUID) (map[string]int, map[string]int, int, error) {
	byType := map[string]int{}
	bySeverity := map[string]int{}
	rows, err := r.db.Query(ctx, `SELECT type, COUNT(*) FROM contradictions WHERE tenant_id = $1 GROUP BY type`, tenantID)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("query contradiction by type: %w", err)
	}
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			rows.Close()
			return nil, nil, 0, fmt.Errorf("scan contradiction by type: %w", err)
		}
		byType[key] = count
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, 0, fmt.Errorf("iterate contradiction by type: %w", err)
	}
	rows.Close()

	rows, err = r.db.Query(ctx, `SELECT severity, COUNT(*) FROM contradictions WHERE tenant_id = $1 GROUP BY severity`, tenantID)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("query contradiction by severity: %w", err)
	}
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			rows.Close()
			return nil, nil, 0, fmt.Errorf("scan contradiction by severity: %w", err)
		}
		bySeverity[key] = count
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, 0, fmt.Errorf("iterate contradiction by severity: %w", err)
	}
	rows.Close()

	var openCount int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM contradictions
		WHERE tenant_id = $1 AND status IN ('detected', 'investigating')`,
		tenantID,
	).Scan(&openCount); err != nil {
		return nil, nil, 0, fmt.Errorf("query open contradictions: %w", err)
	}
	return byType, bySeverity, openCount, nil
}

func mustBuild(qb *database.QueryBuilder) (string, []any) {
	return qb.Build()
}
