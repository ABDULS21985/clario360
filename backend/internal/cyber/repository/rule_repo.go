package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/database"
)

// RuleRepository handles detection rule persistence and historical security events.
type RuleRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewRuleRepository creates a new RuleRepository.
func NewRuleRepository(db *pgxpool.Pool, logger zerolog.Logger) *RuleRepository {
	return &RuleRepository{db: db, logger: logger}
}

// List returns a paginated list of tenant-scoped detection rules.
func (r *RuleRepository) List(ctx context.Context, tenantID uuid.UUID, params *dto.RuleListParams) ([]*model.DetectionRule, int, error) {
	baseSelect := `
		SELECT
			a.id, a.tenant_id, a.name, a.description, a.rule_type, a.severity,
			a.enabled, a.rule_content, a.mitre_tactic_ids, a.mitre_technique_ids,
			a.base_confidence, a.false_positive_count, a.true_positive_count,
			a.last_triggered_at, a.trigger_count, a.tags, a.is_template,
			a.template_id, a.created_by, a.created_at, a.updated_at, a.deleted_at
		FROM detection_rules a`

	qb := database.NewQueryBuilder(baseSelect)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.Where("a.deleted_at IS NULL")
	qb.Where("a.is_template = false")
	if params.Search != nil && strings.TrimSpace(*params.Search) != "" {
		search := "%" + strings.TrimSpace(*params.Search) + "%"
		qb.Where("(a.name ILIKE ? OR a.description ILIKE ?)", search, search)
	}
	if len(params.Types) > 0 {
		qb.WhereIn("a.rule_type", params.Types)
	}
	if len(params.Severities) > 0 {
		qb.WhereIn("a.severity", params.Severities)
	}
	if params.Enabled != nil {
		qb.Where("a.enabled = ?", *params.Enabled)
	}
	if params.Tag != nil && *params.Tag != "" {
		qb.WhereArrayContains("a.tags", *params.Tag)
	}
	if params.MITRETacticID != nil && strings.TrimSpace(*params.MITRETacticID) != "" {
		qb.Where("? = ANY(a.mitre_tactic_ids)", strings.TrimSpace(*params.MITRETacticID))
	}
	if params.MITRETechniqueID != nil && strings.TrimSpace(*params.MITRETechniqueID) != "" {
		qb.Where("? = ANY(a.mitre_technique_ids)", strings.TrimSpace(*params.MITRETechniqueID))
	}
	qb.OrderBy(params.Sort, params.Order, []string{"name", "severity", "enabled", "trigger_count", "last_triggered_at", "created_at"})
	qb.Paginate(params.Page, params.PerPage)

	var total int
	rules := make([]*model.DetectionRule, 0)
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		countSQL, countArgs := qb.BuildCount()
		if err := db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
			return fmt.Errorf("count rules: %w", err)
		}

		sql, args := qb.Build()
		rows, err := db.Query(ctx, sql, args...)
		if err != nil {
			return fmt.Errorf("list rules: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			rule, err := scanRule(rows)
			if err != nil {
				return err
			}
			rules = append(rules, rule)
		}
		return rows.Err()
	})
	return rules, total, err
}

// ListTemplates returns the system-wide template rules.
func (r *RuleRepository) ListTemplates(ctx context.Context) ([]*model.DetectionRule, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			id, tenant_id, name, description, rule_type, severity,
			enabled, rule_content, mitre_tactic_ids, mitre_technique_ids,
			base_confidence, false_positive_count, true_positive_count,
			last_triggered_at, trigger_count, tags, is_template,
			template_id, created_by, created_at, updated_at, deleted_at
		FROM detection_rules
		WHERE is_template = true AND deleted_at IS NULL
		ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	templates := make([]*model.DetectionRule, 0)
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, rule)
	}
	return templates, rows.Err()
}

// EnsureTemplate inserts or updates a system template.
func (r *RuleRepository) EnsureTemplate(ctx context.Context, template *model.DetectionRule) error {
	var existingID uuid.UUID
	err := r.db.QueryRow(ctx, `
		SELECT id
		FROM detection_rules
		WHERE is_template = true AND template_id = $1 AND deleted_at IS NULL`,
		template.TemplateID,
	).Scan(&existingID)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("check template existence: %w", err)
	}

	if err == pgx.ErrNoRows {
		_, err = r.db.Exec(ctx, `
			INSERT INTO detection_rules (
				id, tenant_id, name, description, rule_type, severity, enabled, rule_content,
				mitre_tactic_ids, mitre_technique_ids, base_confidence, tags,
				is_template, template_id, created_at, updated_at
			) VALUES (
				$1, NULL, $2, $3, $4, $5, true, $6, $7, $8, $9, $10, true, $11, now(), now()
			)`,
			template.ID, template.Name, template.Description, template.RuleType, template.Severity,
			template.RuleContent, template.MITRETacticIDs, template.MITRETechniqueIDs,
			template.BaseConfidence, template.Tags, template.TemplateID,
		)
		if err != nil {
			return fmt.Errorf("insert template: %w", err)
		}
		return nil
	}

	_, err = r.db.Exec(ctx, `
		UPDATE detection_rules
		SET
			name = $2,
			description = $3,
			rule_type = $4,
			severity = $5,
			rule_content = $6,
			mitre_tactic_ids = $7,
			mitre_technique_ids = $8,
			base_confidence = $9,
			tags = $10,
			updated_at = now(),
			deleted_at = NULL
		WHERE id = $1`,
		existingID, template.Name, template.Description, template.RuleType, template.Severity,
		template.RuleContent, template.MITRETacticIDs, template.MITRETechniqueIDs,
		template.BaseConfidence, template.Tags,
	)
	if err != nil {
		return fmt.Errorf("update template: %w", err)
	}
	return nil
}

// Create inserts a tenant-scoped rule.
func (r *RuleRepository) Create(ctx context.Context, tenantID, userID uuid.UUID, rule *model.DetectionRule) (*model.DetectionRule, error) {
	id := uuid.New()
	now := time.Now().UTC()
	err := runWithTenantWrite(ctx, r.db, tenantID, func(db dbtx) error {
		_, err := db.Exec(ctx, `
			INSERT INTO detection_rules (
				id, tenant_id, name, description, rule_type, severity, enabled, rule_content,
				mitre_tactic_ids, mitre_technique_ids, base_confidence, false_positive_count,
				true_positive_count, trigger_count, tags, is_template, template_id,
				created_by, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8,
				$9, $10, $11, 0, 0, 0, $12, false, $13,
				$14, $15, $15
			)`,
			id, tenantID, rule.Name, rule.Description, rule.RuleType, rule.Severity, rule.Enabled, rule.RuleContent,
			rule.MITRETacticIDs, rule.MITRETechniqueIDs, rule.BaseConfidence, rule.Tags, rule.TemplateID,
			userID, now,
		)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				return ErrConflict
			}
			return fmt.Errorf("insert rule: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, tenantID, id)
}

// GetByID fetches a single tenant-scoped rule.
func (r *RuleRepository) GetByID(ctx context.Context, tenantID, ruleID uuid.UUID) (*model.DetectionRule, error) {
	var item *model.DetectionRule
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		row := db.QueryRow(ctx, `
			SELECT
				id, tenant_id, name, description, rule_type, severity,
				enabled, rule_content, mitre_tactic_ids, mitre_technique_ids,
				base_confidence, false_positive_count, true_positive_count,
				last_triggered_at, trigger_count, tags, is_template,
				template_id, created_by, created_at, updated_at, deleted_at
			FROM detection_rules
			WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
			tenantID, ruleID,
		)
		rule, err := scanRule(row)
		if err != nil {
			if err == pgx.ErrNoRows {
				return ErrNotFound
			}
			return fmt.Errorf("get rule: %w", err)
		}
		item = rule
		return nil
	})
	return item, err
}

// Update replaces the mutable fields of a tenant-scoped rule.
func (r *RuleRepository) Update(ctx context.Context, tenantID, ruleID uuid.UUID, rule *model.DetectionRule) (*model.DetectionRule, error) {
	err := runWithTenantWrite(ctx, r.db, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `
			UPDATE detection_rules
			SET
				name = $3,
				description = $4,
				severity = $5,
				enabled = $6,
				rule_content = $7,
				mitre_tactic_ids = $8,
				mitre_technique_ids = $9,
				base_confidence = $10,
				tags = $11,
				updated_at = now()
			WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
			tenantID, ruleID, rule.Name, rule.Description, rule.Severity, rule.Enabled, rule.RuleContent,
			rule.MITRETacticIDs, rule.MITRETechniqueIDs, rule.BaseConfidence, rule.Tags,
		)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				return ErrConflict
			}
			return fmt.Errorf("update rule: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, tenantID, ruleID)
}

// Stats returns aggregate rule metrics for a tenant.
func (r *RuleRepository) Stats(ctx context.Context, tenantID uuid.UUID) (*dto.RuleStatsResponse, error) {
	stats := &dto.RuleStatsResponse{}
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		if err := db.QueryRow(ctx, `
			SELECT
				COUNT(*) AS total,
				COUNT(*) FILTER (WHERE enabled = true) AS active
			FROM detection_rules
			WHERE tenant_id = $1 AND is_template = false AND deleted_at IS NULL`,
			tenantID,
		).Scan(&stats.Total, &stats.Active); err != nil {
			return fmt.Errorf("rule totals: %w", err)
		}

		buildCounts := func(query string) ([]model.NamedCount, error) {
			rows, err := db.Query(ctx, query, tenantID)
			if err != nil {
				return nil, err
			}
			defer rows.Close()

			items := make([]model.NamedCount, 0)
			for rows.Next() {
				var item model.NamedCount
				if err := rows.Scan(&item.Name, &item.Count); err != nil {
					return nil, err
				}
				items = append(items, item)
			}
			return items, rows.Err()
		}

		var err error
		if stats.ByType, err = buildCounts(`
			SELECT rule_type::text, COUNT(*)
			FROM detection_rules
			WHERE tenant_id = $1 AND is_template = false AND deleted_at IS NULL
			GROUP BY rule_type
			ORDER BY COUNT(*) DESC, rule_type ASC`); err != nil {
			return fmt.Errorf("rule stats by type: %w", err)
		}
		if stats.BySeverity, err = buildCounts(`
			SELECT severity::text, COUNT(*)
			FROM detection_rules
			WHERE tenant_id = $1 AND is_template = false AND deleted_at IS NULL
			GROUP BY severity
			ORDER BY COUNT(*) DESC, severity ASC`); err != nil {
			return fmt.Errorf("rule stats by severity: %w", err)
		}

		var tpTotal int
		var fpTotal int
		if err := db.QueryRow(ctx, `
			SELECT
				COALESCE(SUM(true_positive_count), 0),
				COALESCE(SUM(false_positive_count), 0)
			FROM detection_rules
			WHERE tenant_id = $1 AND is_template = false AND deleted_at IS NULL`,
			tenantID,
		).Scan(&tpTotal, &fpTotal); err != nil {
			return fmt.Errorf("rule feedback stats: %w", err)
		}
		if totalFeedback := tpTotal + fpTotal; totalFeedback > 0 {
			stats.TruePositiveRate = float64(tpTotal) / float64(totalFeedback)
		}
		if err := db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM alerts
			WHERE tenant_id = $1
			  AND rule_id IS NOT NULL
			  AND deleted_at IS NULL
			  AND created_at >= now() - interval '30 days'`,
			tenantID,
		).Scan(&stats.AlertsLast30Days); err != nil {
			return fmt.Errorf("rule alerts last 30 days: %w", err)
		}
		return nil
	})
	return stats, err
}

// RulePerformance returns operational metrics for a single rule.
func (r *RuleRepository) RulePerformance(ctx context.Context, tenantID, ruleID uuid.UUID) (*dto.RulePerformanceResponse, error) {
	perf := &dto.RulePerformanceResponse{
		SeverityDistribution: []model.NamedCount{},
		AlertTrend:           []dto.RuleAlertTrendPoint{},
		TopAssets:            []dto.RuleTopAsset{},
	}
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		var tpTotal int
		var fpTotal int
		if err := db.QueryRow(ctx, `
			SELECT
				true_positive_count,
				false_positive_count
			FROM detection_rules
			WHERE tenant_id = $1
			  AND id = $2
			  AND is_template = false
			  AND deleted_at IS NULL`,
			tenantID, ruleID,
		).Scan(&tpTotal, &fpTotal); err != nil {
			if err == pgx.ErrNoRows {
				return ErrNotFound
			}
			return fmt.Errorf("rule performance feedback counters: %w", err)
		}
		if totalFeedback := tpTotal + fpTotal; totalFeedback > 0 {
			perf.TruePositiveRate = float64(tpTotal) / float64(totalFeedback)
			perf.FalsePositiveRate = float64(fpTotal) / float64(totalFeedback)
		}

		if err := db.QueryRow(ctx, `
			SELECT
				COUNT(*) FILTER (WHERE created_at >= now() - interval '30 days') AS alerts_last_30_days,
				COUNT(*) FILTER (WHERE created_at >= now() - interval '90 days') AS alerts_last_90_days
			FROM alerts
			WHERE tenant_id = $1
			  AND rule_id = $2
			  AND deleted_at IS NULL`,
			tenantID, ruleID,
		).Scan(&perf.AlertsLast30Days, &perf.AlertsLast90Days); err != nil {
			return fmt.Errorf("rule performance totals: %w", err)
		}

		rows, err := db.Query(ctx, `
			SELECT severity::text, COUNT(*)
			FROM alerts
			WHERE tenant_id = $1
			  AND rule_id = $2
			  AND deleted_at IS NULL
			  AND created_at >= now() - interval '90 days'
			GROUP BY severity
			ORDER BY COUNT(*) DESC, severity ASC`,
			tenantID, ruleID,
		)
		if err != nil {
			return fmt.Errorf("rule severity distribution: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var item model.NamedCount
			if err := rows.Scan(&item.Name, &item.Count); err != nil {
				return err
			}
			perf.SeverityDistribution = append(perf.SeverityDistribution, item)
		}
		if err := rows.Err(); err != nil {
			return err
		}

		trendRows, err := db.Query(ctx, `
			WITH buckets AS (
				SELECT generate_series(
					date_trunc('day', now() - interval '89 days'),
					date_trunc('day', now()),
					interval '1 day'
				) AS bucket
			)
			SELECT
				b.bucket,
				COALESCE(alerts.count, 0) AS count
			FROM buckets b
			LEFT JOIN (
				SELECT date_trunc('day', created_at) AS bucket, COUNT(*)::int AS count
				FROM alerts
				WHERE tenant_id = $1
				  AND rule_id = $2
				  AND deleted_at IS NULL
				  AND created_at >= now() - interval '90 days'
				GROUP BY 1
			) alerts ON alerts.bucket = b.bucket
			ORDER BY b.bucket ASC`,
			tenantID, ruleID,
		)
		if err != nil {
			return fmt.Errorf("rule alert trend: %w", err)
		}
		defer trendRows.Close()
		for trendRows.Next() {
			var point dto.RuleAlertTrendPoint
			if err := trendRows.Scan(&point.Date, &point.Count); err != nil {
				return err
			}
			perf.AlertTrend = append(perf.AlertTrend, point)
		}
		if err := trendRows.Err(); err != nil {
			return err
		}

		assetRows, err := db.Query(ctx, `
			SELECT
				a.asset_id,
				COALESCE(assets.name, CASE WHEN a.asset_id IS NOT NULL THEN a.asset_id::text ELSE 'Unknown asset' END) AS asset_name,
				COUNT(*)::int AS alert_count
			FROM alerts a
			LEFT JOIN assets ON assets.tenant_id = a.tenant_id AND assets.id = a.asset_id AND assets.deleted_at IS NULL
			WHERE a.tenant_id = $1
			  AND a.rule_id = $2
			  AND a.deleted_at IS NULL
			  AND a.created_at >= now() - interval '90 days'
			GROUP BY a.asset_id, asset_name
			ORDER BY alert_count DESC, asset_name ASC
			LIMIT 5`,
			tenantID, ruleID,
		)
		if err != nil {
			return fmt.Errorf("rule top assets: %w", err)
		}
		defer assetRows.Close()
		for assetRows.Next() {
			var item dto.RuleTopAsset
			if err := assetRows.Scan(&item.AssetID, &item.AssetName, &item.AlertCount); err != nil {
				return err
			}
			perf.TopAssets = append(perf.TopAssets, item)
		}
		return assetRows.Err()
	})
	return perf, err
}

// ListByTechnique returns tenant rules mapped to a MITRE technique.
func (r *RuleRepository) ListByTechnique(ctx context.Context, tenantID uuid.UUID, techniqueID string) ([]*model.DetectionRule, error) {
	items := make([]*model.DetectionRule, 0)
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, `
			SELECT
				id, tenant_id, name, description, rule_type, severity,
				enabled, rule_content, mitre_tactic_ids, mitre_technique_ids,
				base_confidence, false_positive_count, true_positive_count,
				last_triggered_at, trigger_count, tags, is_template,
				template_id, created_by, created_at, updated_at, deleted_at
			FROM detection_rules
			WHERE tenant_id = $1
			  AND is_template = false
			  AND deleted_at IS NULL
			  AND $2 = ANY(mitre_technique_ids)
			ORDER BY enabled DESC, trigger_count DESC, name ASC`,
			tenantID, techniqueID,
		)
		if err != nil {
			return fmt.Errorf("list rules by technique: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			item, err := scanRule(rows)
			if err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

// TechniqueCoverageContext captures tenant-specific threat and alert activity for a technique.
type TechniqueCoverageContext struct {
	AlertCount        int
	ThreatCount       int
	ActiveThreatCount int
	LastAlertAt       *time.Time
	Threats           []dto.MITREThreatReferenceDTO
}

// TechniqueCoverageContextMap returns alert and threat context for every MITRE technique seen by the tenant.
func (r *RuleRepository) TechniqueCoverageContextMap(ctx context.Context, tenantID uuid.UUID) (map[string]*TechniqueCoverageContext, error) {
	contextMap := make(map[string]*TechniqueCoverageContext)
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		alertRows, err := db.Query(ctx, `
			SELECT
				mitre_technique_id,
				COUNT(*)::int AS alert_count,
				MAX(created_at) AS last_alert_at
			FROM alerts
			WHERE tenant_id = $1
			  AND deleted_at IS NULL
			  AND mitre_technique_id IS NOT NULL
			  AND created_at >= now() - interval '90 days'
			GROUP BY mitre_technique_id`,
			tenantID,
		)
		if err != nil {
			return fmt.Errorf("mitre alert context: %w", err)
		}
		defer alertRows.Close()
		for alertRows.Next() {
			var (
				techniqueID string
				alertCount  int
				lastAlertAt *time.Time
			)
			if err := alertRows.Scan(&techniqueID, &alertCount, &lastAlertAt); err != nil {
				return err
			}
			contextMap[techniqueID] = &TechniqueCoverageContext{
				AlertCount:  alertCount,
				LastAlertAt: lastAlertAt,
				Threats:     []dto.MITREThreatReferenceDTO{},
			}
		}
		if err := alertRows.Err(); err != nil {
			return err
		}

		threatRows, err := db.Query(ctx, `
			SELECT
				t.id,
				t.name,
				t.type,
				t.severity,
				t.status,
				t.last_seen_at,
				technique_id
			FROM threats t
			CROSS JOIN LATERAL unnest(COALESCE(t.mitre_technique_ids, ARRAY[]::text[])) AS technique_id
			WHERE t.tenant_id = $1
			  AND t.deleted_at IS NULL`,
			tenantID,
		)
		if err != nil {
			return fmt.Errorf("mitre threat context: %w", err)
		}
		defer threatRows.Close()
		for threatRows.Next() {
			var (
				item        dto.MITREThreatReferenceDTO
				techniqueID string
			)
			if err := threatRows.Scan(
				&item.ID,
				&item.Name,
				&item.Type,
				&item.Severity,
				&item.Status,
				&item.LastSeenAt,
				&techniqueID,
			); err != nil {
				return err
			}
			entry, ok := contextMap[techniqueID]
			if !ok {
				entry = &TechniqueCoverageContext{Threats: []dto.MITREThreatReferenceDTO{}}
				contextMap[techniqueID] = entry
			}
			entry.ThreatCount++
			if item.Status != model.ThreatStatusClosed && item.Status != model.ThreatStatusEradicated {
				entry.ActiveThreatCount++
			}
			entry.Threats = append(entry.Threats, item)
		}
		return threatRows.Err()
	})
	return contextMap, err
}

// TechniqueRecentAlerts returns a compact alert list for a MITRE technique.
func (r *RuleRepository) TechniqueRecentAlerts(ctx context.Context, tenantID uuid.UUID, techniqueID string, limit int) ([]dto.MITREAlertReferenceDTO, error) {
	if limit <= 0 {
		limit = 10
	}
	items := make([]dto.MITREAlertReferenceDTO, 0, limit)
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, `
			SELECT
				a.id,
				a.title,
				a.severity,
				a.status,
				a.confidence_score,
				COALESCE(assets.name, NULL) AS asset_name,
				a.created_at
			FROM alerts a
			LEFT JOIN assets ON assets.tenant_id = a.tenant_id AND assets.id = a.asset_id AND assets.deleted_at IS NULL
			WHERE a.tenant_id = $1
			  AND a.deleted_at IS NULL
			  AND a.mitre_technique_id = $2
			ORDER BY a.created_at DESC
			LIMIT $3`,
			tenantID, techniqueID, limit,
		)
		if err != nil {
			return fmt.Errorf("technique recent alerts: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var item dto.MITREAlertReferenceDTO
			if err := rows.Scan(&item.ID, &item.Title, &item.Severity, &item.Status, &item.ConfidenceScore, &item.AssetName, &item.CreatedAt); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

// SoftDelete marks a rule as deleted.
func (r *RuleRepository) SoftDelete(ctx context.Context, tenantID, ruleID uuid.UUID) error {
	return runWithTenantWrite(ctx, r.db, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `
			UPDATE detection_rules
			SET deleted_at = now(), updated_at = now()
			WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
			tenantID, ruleID,
		)
		if err != nil {
			return fmt.Errorf("delete rule: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// Toggle enables or disables a rule.
func (r *RuleRepository) Toggle(ctx context.Context, tenantID, ruleID uuid.UUID, enabled bool) (*model.DetectionRule, error) {
	if err := runWithTenantWrite(ctx, r.db, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `
			UPDATE detection_rules
			SET enabled = $3, updated_at = now()
			WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
			tenantID, ruleID, enabled,
		)
		if err != nil {
			return fmt.Errorf("toggle rule: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, tenantID, ruleID)
}

// ListEnabledByTenant returns all enabled tenant-scoped rules.
func (r *RuleRepository) ListEnabledByTenant(ctx context.Context, tenantID uuid.UUID) ([]*model.DetectionRule, error) {
	rules := make([]*model.DetectionRule, 0)
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, `
			SELECT
				id, tenant_id, name, description, rule_type, severity,
				enabled, rule_content, mitre_tactic_ids, mitre_technique_ids,
				base_confidence, false_positive_count, true_positive_count,
				last_triggered_at, trigger_count, tags, is_template,
				template_id, created_by, created_at, updated_at, deleted_at
			FROM detection_rules
			WHERE tenant_id = $1 AND enabled = true AND deleted_at IS NULL
			ORDER BY created_at ASC`,
			tenantID,
		)
		if err != nil {
			return fmt.Errorf("list enabled rules: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			rule, err := scanRule(rows)
			if err != nil {
				return err
			}
			rules = append(rules, rule)
		}
		return rows.Err()
	})
	return rules, err
}

// UpdateTriggered increments trigger metrics for a rule.
func (r *RuleRepository) UpdateTriggered(ctx context.Context, tenantID, ruleID uuid.UUID, matchedAt time.Time) error {
	return runWithTenantWrite(ctx, r.db, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `
			UPDATE detection_rules
			SET
				trigger_count = trigger_count + 1,
				last_triggered_at = $3,
				updated_at = now()
			WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
			tenantID, ruleID, matchedAt,
		)
		if err != nil {
			return fmt.Errorf("update trigger counters: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// UpdateFeedbackCounters increments TP/FP counters and returns the updated rule.
func (r *RuleRepository) UpdateFeedbackCounters(ctx context.Context, tenantID, ruleID uuid.UUID, feedback string) (*model.DetectionRule, error) {
	var column string
	switch feedback {
	case "true_positive":
		column = "true_positive_count"
	case "false_positive":
		column = "false_positive_count"
	default:
		return nil, ErrInvalidInput
	}
	sql := fmt.Sprintf(`
		UPDATE detection_rules
		SET %s = %s + 1, updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		column, column,
	)
	if err := runWithTenantWrite(ctx, r.db, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, sql, tenantID, ruleID)
		if err != nil {
			return fmt.Errorf("update feedback counters: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, tenantID, ruleID)
}

// EnsureSecurityEventPartitions ensures partitions exist for the supplied timestamps.
func (r *RuleRepository) EnsureSecurityEventPartitions(ctx context.Context, timestamps []time.Time) error {
	seen := make(map[string]struct{})
	for _, ts := range timestamps {
		monthStart := time.Date(ts.UTC().Year(), ts.UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
		key := monthStart.Format("2006-01-02")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if _, err := r.db.Exec(ctx, `SELECT create_security_events_partition($1::date)`, monthStart.Format("2006-01-02")); err != nil {
			return fmt.Errorf("ensure security event partition %s: %w", key, err)
		}
	}
	return nil
}

// InsertSecurityEvents persists a batch of normalized events.
func (r *RuleRepository) InsertSecurityEvents(ctx context.Context, events []model.SecurityEvent) error {
	if len(events) == 0 {
		return nil
	}
	timestamps := make([]time.Time, 0, len(events))
	rowsByTenant := make(map[uuid.UUID][][]interface{}, len(events))
	for _, event := range events {
		if event.MatchedRules == nil {
			event.MatchedRules = []uuid.UUID{}
		}
		timestamps = append(timestamps, event.Timestamp)
		rowsByTenant[event.TenantID] = append(rowsByTenant[event.TenantID], []interface{}{
			event.ID,
			event.TenantID,
			event.Timestamp,
			event.Source,
			event.Type,
			event.Severity,
			event.SourceIP,
			event.DestIP,
			event.DestPort,
			event.Protocol,
			event.Username,
			event.Process,
			event.ParentProcess,
			event.CommandLine,
			event.FilePath,
			event.FileHash,
			event.AssetID,
			ensureRawMessage(event.RawEvent, "{}"),
			event.MatchedRules,
			event.ProcessedAt,
		})
	}
	if err := r.EnsureSecurityEventPartitions(ctx, timestamps); err != nil {
		return err
	}
	for tenantID, rows := range rowsByTenant {
		if err := runWithTenantWrite(ctx, r.db, tenantID, func(db dbtx) error {
			_, err := db.CopyFrom(ctx,
				pgx.Identifier{"security_events"},
				[]string{
					"id", "tenant_id", "timestamp", "source", "type", "severity", "source_ip", "dest_ip",
					"dest_port", "protocol", "username", "process", "parent_process", "command_line",
					"file_path", "file_hash", "asset_id", "raw_event", "matched_rules", "processed_at",
				},
				pgx.CopyFromRows(rows),
			)
			if err != nil {
				return fmt.Errorf("copy security events: %w", err)
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

// ListSecurityEvents returns historical events for dry-run rule testing.
func (r *RuleRepository) ListSecurityEvents(ctx context.Context, tenantID uuid.UUID, from, to *time.Time, limit int) ([]model.SecurityEvent, error) {
	if limit <= 0 || limit > 5000 {
		limit = 5000
	}
	args := []interface{}{tenantID}
	sql := `
		SELECT
			id, tenant_id, timestamp, source, type, severity,
			source_ip::text, dest_ip::text, dest_port, protocol, username,
			process, parent_process, command_line, file_path, file_hash,
			asset_id, raw_event, matched_rules, processed_at
		FROM security_events
		WHERE tenant_id = $1`
	if from != nil {
		args = append(args, *from)
		sql += fmt.Sprintf(" AND timestamp >= $%d", len(args))
	}
	if to != nil {
		args = append(args, *to)
		sql += fmt.Sprintf(" AND timestamp <= $%d", len(args))
	}
	args = append(args, limit)
	sql += fmt.Sprintf(" ORDER BY timestamp DESC LIMIT $%d", len(args))

	events := make([]model.SecurityEvent, 0)
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, sql, args...)
		if err != nil {
			return fmt.Errorf("list security events: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			event, err := scanSecurityEvent(rows)
			if err != nil {
				return err
			}
			events = append(events, *event)
		}
		return rows.Err()
	})
	return events, err
}

// eventSelectColumns is the standard column list for security_events queries.
const eventSelectColumns = `
	id, tenant_id, timestamp, source, type, severity,
	source_ip::text, dest_ip::text, dest_port, protocol, username,
	process, parent_process, command_line, file_path, file_hash,
	asset_id, raw_event, matched_rules, processed_at`

// eventSortAllowlist is the set of columns that may appear in ORDER BY.
var eventSortAllowlist = []string{
	"timestamp", "source", "type", "severity", "source_ip",
	"dest_ip", "username", "process", "processed_at",
}

// QuerySecurityEvents returns a paginated, filtered list of security events.
func (r *RuleRepository) QuerySecurityEvents(ctx context.Context, tenantID uuid.UUID, params *model.EventQueryParams) ([]model.SecurityEvent, int, error) {
	params.SetDefaults()

	qb := database.NewQueryBuilder("SELECT " + eventSelectColumns + " FROM security_events e")
	qb.Where("e.tenant_id = ?", tenantID)

	// Time range filters
	if params.From != nil {
		qb.Where("e.timestamp >= ?", *params.From)
	}
	if params.To != nil {
		qb.Where("e.timestamp <= ?", *params.To)
	}

	// Field filters
	qb.WhereIf(params.Source != "", "e.source = ?", params.Source)
	qb.WhereIf(params.Type != "", "e.type = ?", params.Type)
	if len(params.Severities) > 0 {
		qb.WhereIn("e.severity", params.Severities)
	}
	qb.WhereIf(params.SourceIP != "", "host(e.source_ip) = ?", params.SourceIP)
	qb.WhereIf(params.DestIP != "", "host(e.dest_ip) = ?", params.DestIP)
	qb.WhereIf(params.Protocol != "", "e.protocol = ?", params.Protocol)
	qb.WhereIf(params.Username != "", "e.username ILIKE ?", "%"+params.Username+"%")
	qb.WhereIf(params.Process != "", "e.process ILIKE ?", "%"+params.Process+"%")
	qb.WhereIf(params.CmdContains != "", "e.command_line ILIKE ?", "%"+params.CmdContains+"%")
	qb.WhereIf(params.FileHash != "", "e.file_hash = ?", params.FileHash)
	qb.WhereIf(params.Search != "", "e.raw_event::text ILIKE ?", "%"+params.Search+"%")
	if params.MatchedRule != "" {
		ruleID, err := uuid.Parse(params.MatchedRule)
		if err == nil {
			qb.Where("? = ANY(e.matched_rules)", ruleID)
		}
	}

	qb.OrderBy(params.Sort, params.Order, eventSortAllowlist)
	qb.Paginate(params.Page, params.PerPage)

	var total int
	events := make([]model.SecurityEvent, 0)
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		countSQL, countArgs := qb.BuildCount()
		if err := db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
			return fmt.Errorf("count security events: %w", err)
		}

		sql, args := qb.Build()
		rows, err := db.Query(ctx, sql, args...)
		if err != nil {
			return fmt.Errorf("query security events: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			event, err := scanSecurityEvent(rows)
			if err != nil {
				return err
			}
			events = append(events, *event)
		}
		return rows.Err()
	})
	return events, total, err
}

// GetSecurityEvent returns a single security event by ID.
func (r *RuleRepository) GetSecurityEvent(ctx context.Context, tenantID, eventID uuid.UUID) (*model.SecurityEvent, error) {
	sql := `SELECT ` + eventSelectColumns + `
		FROM security_events
		WHERE tenant_id = $1 AND id = $2
		LIMIT 1`
	var event *model.SecurityEvent
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		row := db.QueryRow(ctx, sql, tenantID, eventID)
		item, err := scanSecurityEvent(row)
		if err != nil {
			if err == pgx.ErrNoRows {
				return ErrNotFound
			}
			return fmt.Errorf("get security event: %w", err)
		}
		event = item
		return nil
	})
	return event, err
}

// GetSecurityEventStats returns aggregate counts grouped by source, type, and severity.
func (r *RuleRepository) GetSecurityEventStats(ctx context.Context, tenantID uuid.UUID, from, to *time.Time) (*model.EventStats, error) {
	args := []interface{}{tenantID}
	timeFilter := ""
	if from != nil {
		args = append(args, *from)
		timeFilter += fmt.Sprintf(" AND timestamp >= $%d", len(args))
	}
	if to != nil {
		args = append(args, *to)
		timeFilter += fmt.Sprintf(" AND timestamp <= $%d", len(args))
	}

	var stats model.EventStats
	err := runWithTenantRead(ctx, r.db, tenantID, func(db dbtx) error {
		countSQL := "SELECT COUNT(*) FROM security_events WHERE tenant_id = $1" + timeFilter
		if err := db.QueryRow(ctx, countSQL, args...).Scan(&stats.Total); err != nil {
			return fmt.Errorf("event stats total: %w", err)
		}

		bySourceSQL := `SELECT source AS name, COUNT(*)::int AS count
			FROM security_events WHERE tenant_id = $1` + timeFilter + `
			GROUP BY source ORDER BY count DESC LIMIT 20`
		stats.BySource = r.queryNamedCounts(ctx, db, bySourceSQL, args)

		byTypeSQL := `SELECT type AS name, COUNT(*)::int AS count
			FROM security_events WHERE tenant_id = $1` + timeFilter + `
			GROUP BY type ORDER BY count DESC LIMIT 20`
		stats.ByType = r.queryNamedCounts(ctx, db, byTypeSQL, args)

		bySevSQL := `SELECT severity AS name, COUNT(*)::int AS count
			FROM security_events WHERE tenant_id = $1` + timeFilter + `
			GROUP BY severity ORDER BY count DESC`
		stats.BySeverity = r.queryNamedCounts(ctx, db, bySevSQL, args)
		return nil
	})
	return &stats, err
}

// queryNamedCounts runs a query returning (name, count) rows and collects them.
func (r *RuleRepository) queryNamedCounts(ctx context.Context, db dbtx, sql string, args []interface{}) []model.NamedCount {
	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		r.logger.Warn().Err(err).Msg("queryNamedCounts failed")
		return []model.NamedCount{}
	}
	defer rows.Close()
	var result []model.NamedCount
	for rows.Next() {
		var nc model.NamedCount
		if err := rows.Scan(&nc.Name, &nc.Count); err != nil {
			continue
		}
		result = append(result, nc)
	}
	if result == nil {
		return []model.NamedCount{}
	}
	return result
}
