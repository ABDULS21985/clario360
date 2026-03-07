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
	"github.com/clario360/platform/internal/database"
)

type CTEMFindingRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewCTEMFindingRepository(db *pgxpool.Pool, logger zerolog.Logger) *CTEMFindingRepository {
	return &CTEMFindingRepository{db: db, logger: logger}
}

func (r *CTEMFindingRepository) DeleteByAssessment(ctx context.Context, tenantID, assessmentID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM ctem_findings WHERE tenant_id = $1 AND assessment_id = $2`, tenantID, assessmentID)
	return err
}

func (r *CTEMFindingRepository) BulkInsert(ctx context.Context, findings []*model.CTEMFinding) error {
	if len(findings) == 0 {
		return nil
	}

	rows := make([][]any, 0, len(findings))
	for _, finding := range findings {
		rows = append(rows, []any{
			finding.ID, finding.TenantID, finding.AssessmentID, string(finding.Type), string(finding.Category),
			finding.Severity, finding.Title, finding.Description, jsonDefault(finding.Evidence, "{}"),
			finding.AffectedAssetIDs, finding.AffectedAssetCount, finding.PrimaryAssetID, finding.VulnerabilityIDs,
			finding.CVEIDs, finding.BusinessImpactScore, jsonDefault(finding.BusinessImpactFactors, "[]"),
			finding.ExploitabilityScore, jsonDefault(finding.ExploitabilityFactors, "[]"), finding.PriorityScore,
			finding.PriorityGroup, finding.PriorityRank, string(finding.ValidationStatus), finding.CompensatingControls,
			finding.ValidationNotes, finding.ValidatedAt, remediationTypeValue(finding.RemediationType),
			finding.RemediationDescription, remediationEffortValue(finding.RemediationEffort), finding.RemediationGroupID,
			finding.EstimatedDays, string(finding.Status), finding.StatusChangedBy, finding.StatusChangedAt,
			finding.StatusNotes, jsonNilOrValue(finding.AttackPath), finding.AttackPathLength, jsonDefault(finding.Metadata, "{}"),
			finding.CreatedAt, finding.UpdatedAt,
		})
	}

	_, err := r.db.CopyFrom(
		ctx,
		pgx.Identifier{"ctem_findings"},
		[]string{
			"id", "tenant_id", "assessment_id", "type", "category", "severity", "title", "description",
			"evidence", "affected_asset_ids", "affected_asset_count", "primary_asset_id", "vulnerability_ids",
			"cve_ids", "business_impact_score", "business_impact_factors", "exploitability_score",
			"exploitability_factors", "priority_score", "priority_group", "priority_rank", "validation_status",
			"compensating_controls", "validation_notes", "validated_at", "remediation_type",
			"remediation_description", "remediation_effort", "remediation_group_id", "estimated_days",
			"status", "status_changed_by", "status_changed_at", "status_notes", "attack_path",
			"attack_path_length", "metadata", "created_at", "updated_at",
		},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("bulk insert findings: %w", err)
	}
	return nil
}

func (r *CTEMFindingRepository) ListByAssessment(ctx context.Context, tenantID, assessmentID uuid.UUID, params *dto.CTEMFindingsListParams) ([]*model.CTEMFinding, int, error) {
	qb := database.NewQueryBuilder(`
		SELECT id, tenant_id, assessment_id, type, category, severity, title, description, evidence,
		       affected_asset_ids, affected_asset_count, primary_asset_id, vulnerability_ids, cve_ids,
		       business_impact_score, business_impact_factors, exploitability_score, exploitability_factors,
		       priority_score, priority_group, priority_rank, validation_status, compensating_controls,
		       validation_notes, validated_at, remediation_type, remediation_description, remediation_effort,
		       remediation_group_id, estimated_days, status, status_changed_by, status_changed_at, status_notes,
		       attack_path, attack_path_length, metadata, created_at, updated_at
		FROM ctem_findings a`)
	qb.Where("a.tenant_id = ?", tenantID)
	qb.Where("a.assessment_id = ?", assessmentID)
	if params.Severity != nil {
		qb.Where("a.severity = ?", *params.Severity)
	}
	if params.Type != nil {
		qb.Where("a.type = ?", *params.Type)
	}
	if params.Status != nil {
		qb.Where("a.status = ?", *params.Status)
	}
	if params.PriorityGroup != nil {
		qb.Where("a.priority_group = ?", *params.PriorityGroup)
	}
	if params.Search != nil && *params.Search != "" {
		qb.Where("(a.title ILIKE ? OR a.description ILIKE ?)", "%"+*params.Search+"%", "%"+*params.Search+"%")
	}
	qb.OrderBy(params.Sort, params.Order, []string{"priority_score", "priority_rank", "severity", "created_at", "updated_at"})
	qb.Paginate(params.Page, params.PerPage)

	countSQL, countArgs := qb.BuildCount()
	var total int
	if err := r.db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count findings: %w", err)
	}

	sql, args := qb.Build()
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list findings: %w", err)
	}
	defer rows.Close()
	items := make([]*model.CTEMFinding, 0)
	for rows.Next() {
		item, err := scanCTEMFinding(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan finding: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *CTEMFindingRepository) ListAllByAssessment(ctx context.Context, tenantID, assessmentID uuid.UUID) ([]*model.CTEMFinding, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, assessment_id, type, category, severity, title, description, evidence,
		       affected_asset_ids, affected_asset_count, primary_asset_id, vulnerability_ids, cve_ids,
		       business_impact_score, business_impact_factors, exploitability_score, exploitability_factors,
		       priority_score, priority_group, priority_rank, validation_status, compensating_controls,
		       validation_notes, validated_at, remediation_type, remediation_description, remediation_effort,
		       remediation_group_id, estimated_days, status, status_changed_by, status_changed_at, status_notes,
		       attack_path, attack_path_length, metadata, created_at, updated_at
		FROM ctem_findings
		WHERE tenant_id = $1 AND assessment_id = $2
		ORDER BY priority_score DESC, severity_order(severity) DESC, created_at ASC`,
		tenantID, assessmentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*model.CTEMFinding, 0)
	for rows.Next() {
		item, err := scanCTEMFinding(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *CTEMFindingRepository) GetByID(ctx context.Context, tenantID, findingID uuid.UUID) (*model.CTEMFinding, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, assessment_id, type, category, severity, title, description, evidence,
		       affected_asset_ids, affected_asset_count, primary_asset_id, vulnerability_ids, cve_ids,
		       business_impact_score, business_impact_factors, exploitability_score, exploitability_factors,
		       priority_score, priority_group, priority_rank, validation_status, compensating_controls,
		       validation_notes, validated_at, remediation_type, remediation_description, remediation_effort,
		       remediation_group_id, estimated_days, status, status_changed_by, status_changed_at, status_notes,
		       attack_path, attack_path_length, metadata, created_at, updated_at
		FROM ctem_findings
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, findingID,
	)
	item, err := scanCTEMFinding(row)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	return item, err
}

func (r *CTEMFindingRepository) UpdateStatus(ctx context.Context, tenantID, findingID, changedBy uuid.UUID, req *dto.UpdateCTEMFindingStatusRequest) (*model.CTEMFinding, error) {
	now := time.Now().UTC()
	tag, err := r.db.Exec(ctx, `
		UPDATE ctem_findings
		SET status = $3,
		    status_changed_by = $4,
		    status_changed_at = $5,
		    status_notes = $6,
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, findingID, string(req.Status), changedBy, now, req.Notes,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, tenantID, findingID)
}

func (r *CTEMFindingRepository) SaveAnalysis(ctx context.Context, tenantID, assessmentID uuid.UUID, findings []*model.CTEMFinding) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for _, finding := range findings {
		_, err := tx.Exec(ctx, `
			UPDATE ctem_findings
			SET business_impact_score = $4,
			    business_impact_factors = $5,
			    exploitability_score = $6,
			    exploitability_factors = $7,
			    priority_score = $8,
			    priority_group = $9,
			    priority_rank = $10,
			    validation_status = $11,
			    compensating_controls = $12,
			    validation_notes = $13,
			    validated_at = $14,
			    remediation_type = $15,
			    remediation_description = $16,
			    remediation_effort = $17,
			    remediation_group_id = $18,
			    estimated_days = $19,
			    status = $20,
			    status_changed_by = $21,
			    status_changed_at = $22,
			    status_notes = $23,
			    attack_path = $24,
			    attack_path_length = $25,
			    metadata = $26,
			    updated_at = now()
			WHERE tenant_id = $1 AND assessment_id = $2 AND id = $3`,
			tenantID, assessmentID, finding.ID,
			finding.BusinessImpactScore, jsonDefault(finding.BusinessImpactFactors, "[]"),
			finding.ExploitabilityScore, jsonDefault(finding.ExploitabilityFactors, "[]"),
			finding.PriorityScore, finding.PriorityGroup, finding.PriorityRank, string(finding.ValidationStatus),
			finding.CompensatingControls, finding.ValidationNotes, finding.ValidatedAt, remediationTypeValue(finding.RemediationType),
			finding.RemediationDescription, remediationEffortValue(finding.RemediationEffort), finding.RemediationGroupID,
			finding.EstimatedDays, string(finding.Status), finding.StatusChangedBy, finding.StatusChangedAt,
			finding.StatusNotes, jsonNilOrValue(finding.AttackPath), finding.AttackPathLength, jsonDefault(finding.Metadata, "{}"),
		)
		if err != nil {
			return fmt.Errorf("save finding analysis: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *CTEMFindingRepository) ClearRemediationAssignments(ctx context.Context, tenantID, assessmentID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE ctem_findings
		SET remediation_type = NULL,
		    remediation_description = NULL,
		    remediation_effort = NULL,
		    remediation_group_id = NULL,
		    estimated_days = NULL,
		    updated_at = now()
		WHERE tenant_id = $1 AND assessment_id = $2`,
		tenantID, assessmentID,
	)
	return err
}

func (r *CTEMFindingRepository) Summary(ctx context.Context, tenantID, assessmentID uuid.UUID) (map[string]any, error) {
	rows, err := r.db.Query(ctx, `
		SELECT 'severity' AS group_name, severity AS key, COUNT(*)::int AS count
		FROM ctem_findings
		WHERE tenant_id = $1 AND assessment_id = $2
		GROUP BY severity
		UNION ALL
		SELECT 'type' AS group_name, type AS key, COUNT(*)::int AS count
		FROM ctem_findings
		WHERE tenant_id = $1 AND assessment_id = $2
		GROUP BY type
		UNION ALL
		SELECT 'status' AS group_name, status AS key, COUNT(*)::int AS count
		FROM ctem_findings
		WHERE tenant_id = $1 AND assessment_id = $2
		GROUP BY status
		UNION ALL
		SELECT 'priority_group' AS group_name, priority_group::text AS key, COUNT(*)::int AS count
		FROM ctem_findings
		WHERE tenant_id = $1 AND assessment_id = $2
		GROUP BY priority_group`,
		tenantID, assessmentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summary := map[string]map[string]int{
		"severity":       {},
		"type":           {},
		"status":         {},
		"priority_group": {},
	}
	total := 0
	for rows.Next() {
		var group, key string
		var count int
		if err := rows.Scan(&group, &key, &count); err != nil {
			return nil, err
		}
		summary[group][key] = count
		total += count
	}
	return map[string]any{
		"total":          total,
		"severity":       summary["severity"],
		"type":           summary["type"],
		"status":         summary["status"],
		"priority_group": summary["priority_group"],
	}, rows.Err()
}

func scanCTEMFinding(row interface{ Scan(dest ...any) error }) (*model.CTEMFinding, error) {
	var (
		item                  model.CTEMFinding
		businessImpactFactors []byte
		exploitabilityFactors []byte
		remediationType       *string
		remediationEffort     *string
		attackPath            []byte
	)
	err := row.Scan(
		&item.ID, &item.TenantID, &item.AssessmentID, &item.Type, &item.Category, &item.Severity, &item.Title, &item.Description, &item.Evidence,
		&item.AffectedAssetIDs, &item.AffectedAssetCount, &item.PrimaryAssetID, &item.VulnerabilityIDs, &item.CVEIDs,
		&item.BusinessImpactScore, &businessImpactFactors, &item.ExploitabilityScore, &exploitabilityFactors,
		&item.PriorityScore, &item.PriorityGroup, &item.PriorityRank, &item.ValidationStatus, &item.CompensatingControls,
		&item.ValidationNotes, &item.ValidatedAt, &remediationType, &item.RemediationDescription, &remediationEffort,
		&item.RemediationGroupID, &item.EstimatedDays, &item.Status, &item.StatusChangedBy, &item.StatusChangedAt, &item.StatusNotes,
		&attackPath, &item.AttackPathLength, &item.Metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	item.BusinessImpactFactors = businessImpactFactors
	item.ExploitabilityFactors = exploitabilityFactors
	item.AttackPath = attackPath
	if remediationType != nil {
		value := model.CTEMRemediationType(*remediationType)
		item.RemediationType = &value
	}
	if remediationEffort != nil {
		value := model.CTEMRemediationEffort(*remediationEffort)
		item.RemediationEffort = &value
	}
	if item.Evidence == nil {
		item.Evidence = json.RawMessage("{}")
	}
	if item.BusinessImpactFactors == nil {
		item.BusinessImpactFactors = json.RawMessage("[]")
	}
	if item.ExploitabilityFactors == nil {
		item.ExploitabilityFactors = json.RawMessage("[]")
	}
	if item.Metadata == nil {
		item.Metadata = json.RawMessage("{}")
	}
	if item.AffectedAssetIDs == nil {
		item.AffectedAssetIDs = []uuid.UUID{}
	}
	if item.VulnerabilityIDs == nil {
		item.VulnerabilityIDs = []uuid.UUID{}
	}
	if item.CVEIDs == nil {
		item.CVEIDs = []string{}
	}
	if item.CompensatingControls == nil {
		item.CompensatingControls = []string{}
	}
	return &item, nil
}

func jsonDefault(value json.RawMessage, fallback string) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(fallback)
	}
	return value
}

func jsonNilOrValue(value json.RawMessage) any {
	if len(value) == 0 {
		return nil
	}
	return value
}

func remediationTypeValue(value *model.CTEMRemediationType) any {
	if value == nil {
		return nil
	}
	return string(*value)
}

func remediationEffortValue(value *model.CTEMRemediationEffort) any {
	if value == nil {
		return nil
	}
	return string(*value)
}

func findingMatchKey(f *model.CTEMFinding) string {
	asset := ""
	if f.PrimaryAssetID != nil {
		asset = f.PrimaryAssetID.String()
	}
	cve := ""
	if len(f.CVEIDs) > 0 {
		cve = f.CVEIDs[0]
	}
	return strings.Join([]string{string(f.Type), asset, cve, f.Title}, "|")
}
