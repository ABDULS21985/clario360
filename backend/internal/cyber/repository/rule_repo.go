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
	qb.OrderBy("created_at", "desc", []string{"created_at"})
	qb.Paginate(params.Page, params.PerPage)

	countSQL, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count rules: %w", err)
	}

	sql, args := qb.Build()
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list rules: %w", err)
	}
	defer rows.Close()

	rules := make([]*model.DetectionRule, 0)
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, 0, err
		}
		rules = append(rules, rule)
	}
	return rules, total, rows.Err()
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
	_, err := r.db.Exec(ctx, `
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
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("insert rule: %w", err)
	}
	return r.GetByID(ctx, tenantID, id)
}

// GetByID fetches a single tenant-scoped rule.
func (r *RuleRepository) GetByID(ctx context.Context, tenantID, ruleID uuid.UUID) (*model.DetectionRule, error) {
	row := r.db.QueryRow(ctx, `
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
	item, err := scanRule(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get rule: %w", err)
	}
	return item, nil
}

// Update replaces the mutable fields of a tenant-scoped rule.
func (r *RuleRepository) Update(ctx context.Context, tenantID, ruleID uuid.UUID, rule *model.DetectionRule) (*model.DetectionRule, error) {
	tag, err := r.db.Exec(ctx, `
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
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("update rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, tenantID, ruleID)
}

// SoftDelete marks a rule as deleted.
func (r *RuleRepository) SoftDelete(ctx context.Context, tenantID, ruleID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
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
}

// Toggle enables or disables a rule.
func (r *RuleRepository) Toggle(ctx context.Context, tenantID, ruleID uuid.UUID, enabled bool) (*model.DetectionRule, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE detection_rules
		SET enabled = $3, updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, ruleID, enabled,
	)
	if err != nil {
		return nil, fmt.Errorf("toggle rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, tenantID, ruleID)
}

// ListEnabledByTenant returns all enabled tenant-scoped rules.
func (r *RuleRepository) ListEnabledByTenant(ctx context.Context, tenantID uuid.UUID) ([]*model.DetectionRule, error) {
	rows, err := r.db.Query(ctx, `
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
		return nil, fmt.Errorf("list enabled rules: %w", err)
	}
	defer rows.Close()

	rules := make([]*model.DetectionRule, 0)
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

// UpdateTriggered increments trigger metrics for a rule.
func (r *RuleRepository) UpdateTriggered(ctx context.Context, tenantID, ruleID uuid.UUID, matchedAt time.Time) error {
	tag, err := r.db.Exec(ctx, `
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
	tag, err := r.db.Exec(ctx, sql, tenantID, ruleID)
	if err != nil {
		return nil, fmt.Errorf("update feedback counters: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
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
	rows := make([][]interface{}, 0, len(events))
	for _, event := range events {
		if event.MatchedRules == nil {
			event.MatchedRules = []uuid.UUID{}
		}
		timestamps = append(timestamps, event.Timestamp)
		rows = append(rows, []interface{}{
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
	_, err := r.db.CopyFrom(ctx,
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

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list security events: %w", err)
	}
	defer rows.Close()

	events := make([]model.SecurityEvent, 0)
	for rows.Next() {
		event, err := scanSecurityEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, *event)
	}
	return events, rows.Err()
}
