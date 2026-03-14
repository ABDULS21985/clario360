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
)

// VCISOGovernanceRepository handles all vCISO governance table operations.
type VCISOGovernanceRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewVCISOGovernanceRepository creates a new VCISOGovernanceRepository.
func NewVCISOGovernanceRepository(db *pgxpool.Pool, logger zerolog.Logger) *VCISOGovernanceRepository {
	return &VCISOGovernanceRepository{db: db, logger: logger}
}

// ─── Risks ──────────────────────────────────────────────────────────────────

func (r *VCISOGovernanceRepository) ListRisks(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) ([]*model.VCISORiskEntry, int, error) {
	conds := []string{"tenant_id=$1"}
	args := []interface{}{tenantID}
	i := 2

	if params.Status != "" {
		conds = append(conds, fmt.Sprintf("status=$%d", i))
		args = append(args, params.Status)
		i++
	}
	if params.Category != "" {
		conds = append(conds, fmt.Sprintf("category=$%d", i))
		args = append(args, params.Category)
		i++
	}
	if params.Search != "" {
		conds = append(conds, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", i, i))
		args = append(args, "%"+params.Search+"%")
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM vciso_risks "+where, args...).Scan(&total)

	orderCol := "created_at"
	if params.Sort != "" {
		allowed := map[string]bool{"created_at": true, "updated_at": true, "inherent_score": true, "residual_score": true, "title": true, "status": true}
		if allowed[params.Sort] {
			orderCol = params.Sort
		}
	}
	dir := "DESC"
	if strings.EqualFold(params.Order, "asc") {
		dir = "ASC"
	}

	query := fmt.Sprintf(`SELECT id, tenant_id, title, description, category, department,
		inherent_score, residual_score, likelihood, impact, status, treatment,
		owner_id, owner_name, review_date, business_services, controls, tags,
		treatment_plan, acceptance_rationale, acceptance_approved_by, acceptance_approved_by_name,
		acceptance_expiry, created_at, updated_at
		FROM vciso_risks %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		where, orderCol, dir, i, i+1)
	args = append(args, params.PerPage, params.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list risks: %w", err)
	}
	defer rows.Close()

	var items []*model.VCISORiskEntry
	for rows.Next() {
		item := &model.VCISORiskEntry{}
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.Title, &item.Description, &item.Category, &item.Department,
			&item.InherentScore, &item.ResidualScore, &item.Likelihood, &item.Impact, &item.Status, &item.Treatment,
			&item.OwnerID, &item.OwnerName, &item.ReviewDate, &item.BusinessServices, &item.Controls, &item.Tags,
			&item.TreatmentPlan, &item.AcceptanceRationale, &item.AcceptanceApprovedBy, &item.AcceptanceApprovedByName,
			&item.AcceptanceExpiry, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan risk: %w", err)
		}
		if item.BusinessServices == nil {
			item.BusinessServices = []string{}
		}
		if item.Controls == nil {
			item.Controls = []string{}
		}
		if item.Tags == nil {
			item.Tags = []string{}
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *VCISOGovernanceRepository) CreateRisk(ctx context.Context, tenantID uuid.UUID, item *model.VCISORiskEntry) error {
	item.ID = uuid.New()
	now := time.Now().UTC()
	item.TenantID = tenantID
	item.CreatedAt = now
	item.UpdatedAt = now

	_, err := r.db.Exec(ctx, `INSERT INTO vciso_risks (
		id, tenant_id, title, description, category, department,
		inherent_score, residual_score, likelihood, impact, status, treatment,
		owner_id, owner_name, review_date, business_services, controls, tags,
		treatment_plan, acceptance_rationale, acceptance_expiry, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)`,
		item.ID, item.TenantID, item.Title, item.Description, item.Category, item.Department,
		item.InherentScore, item.ResidualScore, item.Likelihood, item.Impact, item.Status, item.Treatment,
		item.OwnerID, item.OwnerName, item.ReviewDate, item.BusinessServices, item.Controls, item.Tags,
		item.TreatmentPlan, item.AcceptanceRationale, item.AcceptanceExpiry, item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create risk: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) GetRisk(ctx context.Context, tenantID, id uuid.UUID) (*model.VCISORiskEntry, error) {
	item := &model.VCISORiskEntry{}
	err := r.db.QueryRow(ctx, `SELECT id, tenant_id, title, description, category, department,
		inherent_score, residual_score, likelihood, impact, status, treatment,
		owner_id, owner_name, review_date, business_services, controls, tags,
		treatment_plan, acceptance_rationale, acceptance_approved_by, acceptance_approved_by_name,
		acceptance_expiry, created_at, updated_at
		FROM vciso_risks WHERE id=$1 AND tenant_id=$2`, id, tenantID,
	).Scan(
		&item.ID, &item.TenantID, &item.Title, &item.Description, &item.Category, &item.Department,
		&item.InherentScore, &item.ResidualScore, &item.Likelihood, &item.Impact, &item.Status, &item.Treatment,
		&item.OwnerID, &item.OwnerName, &item.ReviewDate, &item.BusinessServices, &item.Controls, &item.Tags,
		&item.TreatmentPlan, &item.AcceptanceRationale, &item.AcceptanceApprovedBy, &item.AcceptanceApprovedByName,
		&item.AcceptanceExpiry, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get risk: %w", err)
	}
	if item.BusinessServices == nil {
		item.BusinessServices = []string{}
	}
	if item.Controls == nil {
		item.Controls = []string{}
	}
	if item.Tags == nil {
		item.Tags = []string{}
	}
	return item, nil
}

func (r *VCISOGovernanceRepository) UpdateRisk(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateRiskRequest) error {
	now := time.Now().UTC()
	ownerID := dto.ParseOptionalUUID(req.OwnerID)
	_, err := r.db.Exec(ctx, `UPDATE vciso_risks SET
		title=$3, description=$4, category=$5, department=$6,
		inherent_score=$7, residual_score=$8, likelihood=$9, impact=$10,
		status=$11, treatment=$12, owner_id=$13, owner_name=$14, review_date=$15,
		business_services=$16, controls=$17, tags=$18, treatment_plan=$19,
		acceptance_rationale=$20, acceptance_expiry=$21, updated_at=$22
		WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
		req.Title, req.Description, req.Category, req.Department,
		req.InherentScore, req.ResidualScore, req.Likelihood, req.Impact,
		req.Status, req.Treatment, ownerID, req.OwnerName, req.ReviewDate,
		req.BusinessServices, req.Controls, req.Tags, req.TreatmentPlan,
		req.AcceptanceRationale, req.AcceptanceExpiry, now,
	)
	if err != nil {
		return fmt.Errorf("update risk: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) DeleteRisk(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM vciso_risks WHERE id=$1 AND tenant_id=$2", id, tenantID)
	if err != nil {
		return fmt.Errorf("delete risk: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *VCISOGovernanceRepository) RiskStats(ctx context.Context, tenantID uuid.UUID) (*model.VCISORiskStats, error) {
	stats := &model.VCISORiskStats{
		ByStatus:    make(map[string]int),
		ByTreatment: make(map[string]int),
		ByLikelihood: make(map[string]int),
		ByImpact:    make(map[string]int),
	}
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*), COALESCE(AVG(inherent_score),0), COALESCE(AVG(residual_score),0),
		COUNT(*) FILTER (WHERE status='accepted'),
		COUNT(*) FILTER (WHERE review_date IS NOT NULL AND review_date < NOW()::text)
		FROM vciso_risks WHERE tenant_id=$1`, tenantID,
	).Scan(&stats.Total, &stats.AvgInherentScore, &stats.AvgResidualScore, &stats.AcceptedCount, &stats.OverdueReviews)

	rows, err := r.db.Query(ctx, "SELECT status, COUNT(*) FROM vciso_risks WHERE tenant_id=$1 GROUP BY status", tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var k string
			var v int
			_ = rows.Scan(&k, &v)
			stats.ByStatus[k] = v
		}
	}

	rows2, err := r.db.Query(ctx, "SELECT treatment, COUNT(*) FROM vciso_risks WHERE tenant_id=$1 GROUP BY treatment", tenantID)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var k string
			var v int
			_ = rows2.Scan(&k, &v)
			stats.ByTreatment[k] = v
		}
	}

	rows3, err := r.db.Query(ctx, "SELECT likelihood, COUNT(*) FROM vciso_risks WHERE tenant_id=$1 GROUP BY likelihood", tenantID)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var k string
			var v int
			_ = rows3.Scan(&k, &v)
			stats.ByLikelihood[k] = v
		}
	}

	rows4, err := r.db.Query(ctx, "SELECT impact, COUNT(*) FROM vciso_risks WHERE tenant_id=$1 GROUP BY impact", tenantID)
	if err == nil {
		defer rows4.Close()
		for rows4.Next() {
			var k string
			var v int
			_ = rows4.Scan(&k, &v)
			stats.ByImpact[k] = v
		}
	}

	return stats, nil
}

// ─── Policies ───────────────────────────────────────────────────────────────

func (r *VCISOGovernanceRepository) ListPolicies(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) ([]*model.VCISOPolicy, int, error) {
	conds := []string{"tenant_id=$1"}
	args := []interface{}{tenantID}
	i := 2

	if params.Status != "" {
		conds = append(conds, fmt.Sprintf("status=$%d", i))
		args = append(args, params.Status)
		i++
	}
	if params.Search != "" {
		conds = append(conds, fmt.Sprintf("(title ILIKE $%d OR domain ILIKE $%d)", i, i))
		args = append(args, "%"+params.Search+"%")
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM vciso_policies "+where, args...).Scan(&total)

	orderCol := "created_at"
	if params.Sort != "" {
		allowed := map[string]bool{"created_at": true, "updated_at": true, "title": true, "status": true}
		if allowed[params.Sort] {
			orderCol = params.Sort
		}
	}
	dir := "DESC"
	if strings.EqualFold(params.Order, "asc") {
		dir = "ASC"
	}

	query := fmt.Sprintf(`SELECT id, tenant_id, title, domain, version, status, content,
		owner_id, owner_name, reviewer_id, reviewer_name, approved_by, approved_by_name,
		approved_at, review_due, last_reviewed_at, tags, exceptions_count, created_at, updated_at
		FROM vciso_policies %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		where, orderCol, dir, i, i+1)
	args = append(args, params.PerPage, params.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list policies: %w", err)
	}
	defer rows.Close()

	var items []*model.VCISOPolicy
	for rows.Next() {
		item := &model.VCISOPolicy{}
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.Title, &item.Domain, &item.Version, &item.Status, &item.Content,
			&item.OwnerID, &item.OwnerName, &item.ReviewerID, &item.ReviewerName, &item.ApprovedBy, &item.ApprovedByName,
			&item.ApprovedAt, &item.ReviewDue, &item.LastReviewedAt, &item.Tags, &item.ExceptionsCount, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan policy: %w", err)
		}
		if item.Tags == nil {
			item.Tags = []string{}
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *VCISOGovernanceRepository) CreatePolicy(ctx context.Context, tenantID uuid.UUID, item *model.VCISOPolicy) error {
	item.ID = uuid.New()
	now := time.Now().UTC()
	item.TenantID = tenantID
	item.CreatedAt = now
	item.UpdatedAt = now

	_, err := r.db.Exec(ctx, `INSERT INTO vciso_policies (
		id, tenant_id, title, domain, version, status, content,
		owner_id, owner_name, review_due, tags, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		item.ID, item.TenantID, item.Title, item.Domain, item.Version, item.Status, item.Content,
		item.OwnerID, item.OwnerName, item.ReviewDue, item.Tags, item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create policy: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) GetPolicy(ctx context.Context, tenantID, id uuid.UUID) (*model.VCISOPolicy, error) {
	item := &model.VCISOPolicy{}
	err := r.db.QueryRow(ctx, `SELECT id, tenant_id, title, domain, version, status, content,
		owner_id, owner_name, reviewer_id, reviewer_name, approved_by, approved_by_name,
		approved_at, review_due, last_reviewed_at, tags, exceptions_count, created_at, updated_at
		FROM vciso_policies WHERE id=$1 AND tenant_id=$2`, id, tenantID,
	).Scan(
		&item.ID, &item.TenantID, &item.Title, &item.Domain, &item.Version, &item.Status, &item.Content,
		&item.OwnerID, &item.OwnerName, &item.ReviewerID, &item.ReviewerName, &item.ApprovedBy, &item.ApprovedByName,
		&item.ApprovedAt, &item.ReviewDue, &item.LastReviewedAt, &item.Tags, &item.ExceptionsCount, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get policy: %w", err)
	}
	if item.Tags == nil {
		item.Tags = []string{}
	}
	return item, nil
}

func (r *VCISOGovernanceRepository) UpdatePolicy(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreatePolicyRequest) error {
	now := time.Now().UTC()
	ownerID, _ := uuid.Parse(req.OwnerID)
	_, err := r.db.Exec(ctx, `UPDATE vciso_policies SET
		title=$3, domain=$4, version=$5, status=$6, content=$7,
		owner_id=$8, owner_name=$9, review_due=$10, tags=$11, updated_at=$12
		WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
		req.Title, req.Domain, req.Version, req.Status, req.Content,
		ownerID, req.OwnerName, req.ReviewDue, req.Tags, now,
	)
	if err != nil {
		return fmt.Errorf("update policy: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) DeletePolicy(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM vciso_policies WHERE id=$1 AND tenant_id=$2", id, tenantID)
	if err != nil {
		return fmt.Errorf("delete policy: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *VCISOGovernanceRepository) UpdatePolicyStatus(ctx context.Context, tenantID, id uuid.UUID, req *dto.UpdatePolicyStatusRequest) error {
	now := time.Now().UTC()
	reviewerID := dto.ParseOptionalUUID(req.ReviewerID)
	approvedByID := dto.ParseOptionalUUID(req.ApprovedBy)
	var approvedAt *time.Time
	if req.Status == "approved" || req.Status == "active" {
		approvedAt = &now
	}
	_, err := r.db.Exec(ctx, `UPDATE vciso_policies SET
		status=$3, reviewer_id=$4, reviewer_name=$5, approved_by=$6, approved_by_name=$7, approved_at=$8, updated_at=$9
		WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
		req.Status, reviewerID, req.ReviewerName, approvedByID, req.ApprovedByName, approvedAt, now,
	)
	if err != nil {
		return fmt.Errorf("update policy status: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) PolicyStats(ctx context.Context, tenantID uuid.UUID) (*dto.GovernanceListResponse, error) {
	stats := make(map[string]int)
	var total int
	rows, err := r.db.Query(ctx, "SELECT status, COUNT(*) FROM vciso_policies WHERE tenant_id=$1 GROUP BY status", tenantID)
	if err != nil {
		return nil, fmt.Errorf("policy stats: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var k string
		var v int
		_ = rows.Scan(&k, &v)
		stats[k] = v
		total += v
	}
	return &dto.GovernanceListResponse{
		Data: map[string]interface{}{"total": total, "by_status": stats},
		Meta: dto.NewPaginationMeta(1, total, total),
	}, nil
}

// ─── Policy Exceptions ──────────────────────────────────────────────────────

func (r *VCISOGovernanceRepository) ListPolicyExceptions(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) ([]*model.VCISOPolicyException, int, error) {
	conds := []string{"tenant_id=$1"}
	args := []interface{}{tenantID}
	i := 2

	if params.Status != "" {
		conds = append(conds, fmt.Sprintf("status=$%d", i))
		args = append(args, params.Status)
		i++
	}
	if params.Search != "" {
		conds = append(conds, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", i, i))
		args = append(args, "%"+params.Search+"%")
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM vciso_policy_exceptions "+where, args...).Scan(&total)

	query := fmt.Sprintf(`SELECT id, tenant_id, policy_id, policy_title, title, description,
		justification, compensating_controls, status, requested_by, requested_by_name,
		approved_by, approved_by_name, decision_notes, expires_at, created_at, updated_at
		FROM vciso_policy_exceptions %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, i, i+1)
	args = append(args, params.PerPage, params.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list policy exceptions: %w", err)
	}
	defer rows.Close()

	var items []*model.VCISOPolicyException
	for rows.Next() {
		item := &model.VCISOPolicyException{}
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.PolicyID, &item.PolicyTitle, &item.Title, &item.Description,
			&item.Justification, &item.CompensatingControls, &item.Status, &item.RequestedBy, &item.RequestedByName,
			&item.ApprovedBy, &item.ApprovedByName, &item.DecisionNotes, &item.ExpiresAt, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan policy exception: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *VCISOGovernanceRepository) CreatePolicyException(ctx context.Context, tenantID, userID uuid.UUID, item *model.VCISOPolicyException) error {
	item.ID = uuid.New()
	now := time.Now().UTC()
	item.TenantID = tenantID
	item.RequestedBy = userID
	item.Status = "pending"
	item.CreatedAt = now
	item.UpdatedAt = now

	_, err := r.db.Exec(ctx, `INSERT INTO vciso_policy_exceptions (
		id, tenant_id, policy_id, policy_title, title, description,
		justification, compensating_controls, status, requested_by, requested_by_name,
		expires_at, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		item.ID, item.TenantID, item.PolicyID, item.PolicyTitle, item.Title, item.Description,
		item.Justification, item.CompensatingControls, item.Status, item.RequestedBy, item.RequestedByName,
		item.ExpiresAt, item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create policy exception: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) DecidePolicyException(ctx context.Context, tenantID, id, userID uuid.UUID, req *dto.DecidePolicyExceptionRequest, userName string) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `UPDATE vciso_policy_exceptions SET
		status=$3, approved_by=$4, approved_by_name=$5, decision_notes=$6, updated_at=$7
		WHERE id=$1 AND tenant_id=$2`,
		id, tenantID, req.Status, userID, userName, req.DecisionNotes, now,
	)
	if err != nil {
		return fmt.Errorf("decide policy exception: %w", err)
	}
	return nil
}

// ─── Vendors ────────────────────────────────────────────────────────────────

func (r *VCISOGovernanceRepository) ListVendors(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) ([]*model.VCISOVendor, int, error) {
	conds := []string{"tenant_id=$1"}
	args := []interface{}{tenantID}
	i := 2

	if params.Status != "" {
		conds = append(conds, fmt.Sprintf("status=$%d", i))
		args = append(args, params.Status)
		i++
	}
	if params.Category != "" {
		conds = append(conds, fmt.Sprintf("category=$%d", i))
		args = append(args, params.Category)
		i++
	}
	if params.Search != "" {
		conds = append(conds, fmt.Sprintf("name ILIKE $%d", i))
		args = append(args, "%"+params.Search+"%")
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM vciso_vendors "+where, args...).Scan(&total)

	orderCol := "created_at"
	if params.Sort != "" {
		allowed := map[string]bool{"created_at": true, "updated_at": true, "risk_score": true, "name": true, "status": true}
		if allowed[params.Sort] {
			orderCol = params.Sort
		}
	}
	dir := "DESC"
	if strings.EqualFold(params.Order, "asc") {
		dir = "ASC"
	}

	query := fmt.Sprintf(`SELECT id, tenant_id, name, category, risk_tier, status, risk_score,
		last_assessment_date, next_review_date, contact_name, contact_email,
		services_provided, data_shared, compliance_frameworks,
		controls_met, controls_total, open_findings, created_at, updated_at
		FROM vciso_vendors %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		where, orderCol, dir, i, i+1)
	args = append(args, params.PerPage, params.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list vendors: %w", err)
	}
	defer rows.Close()

	var items []*model.VCISOVendor
	for rows.Next() {
		item := &model.VCISOVendor{}
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.Name, &item.Category, &item.RiskTier, &item.Status, &item.RiskScore,
			&item.LastAssessmentDate, &item.NextReviewDate, &item.ContactName, &item.ContactEmail,
			&item.ServicesProvided, &item.DataShared, &item.ComplianceFrameworks,
			&item.ControlsMet, &item.ControlsTotal, &item.OpenFindings, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan vendor: %w", err)
		}
		if item.ServicesProvided == nil {
			item.ServicesProvided = []string{}
		}
		if item.DataShared == nil {
			item.DataShared = []string{}
		}
		if item.ComplianceFrameworks == nil {
			item.ComplianceFrameworks = []string{}
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *VCISOGovernanceRepository) CreateVendor(ctx context.Context, tenantID uuid.UUID, item *model.VCISOVendor) error {
	item.ID = uuid.New()
	now := time.Now().UTC()
	item.TenantID = tenantID
	item.CreatedAt = now
	item.UpdatedAt = now

	_, err := r.db.Exec(ctx, `INSERT INTO vciso_vendors (
		id, tenant_id, name, category, risk_tier, status, risk_score,
		next_review_date, contact_name, contact_email,
		services_provided, data_shared, compliance_frameworks,
		controls_met, controls_total, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		item.ID, item.TenantID, item.Name, item.Category, item.RiskTier, item.Status, item.RiskScore,
		item.NextReviewDate, item.ContactName, item.ContactEmail,
		item.ServicesProvided, item.DataShared, item.ComplianceFrameworks,
		item.ControlsMet, item.ControlsTotal, item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create vendor: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) GetVendor(ctx context.Context, tenantID, id uuid.UUID) (*model.VCISOVendor, error) {
	item := &model.VCISOVendor{}
	err := r.db.QueryRow(ctx, `SELECT id, tenant_id, name, category, risk_tier, status, risk_score,
		last_assessment_date, next_review_date, contact_name, contact_email,
		services_provided, data_shared, compliance_frameworks,
		controls_met, controls_total, open_findings, created_at, updated_at
		FROM vciso_vendors WHERE id=$1 AND tenant_id=$2`, id, tenantID,
	).Scan(
		&item.ID, &item.TenantID, &item.Name, &item.Category, &item.RiskTier, &item.Status, &item.RiskScore,
		&item.LastAssessmentDate, &item.NextReviewDate, &item.ContactName, &item.ContactEmail,
		&item.ServicesProvided, &item.DataShared, &item.ComplianceFrameworks,
		&item.ControlsMet, &item.ControlsTotal, &item.OpenFindings, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get vendor: %w", err)
	}
	if item.ServicesProvided == nil {
		item.ServicesProvided = []string{}
	}
	if item.DataShared == nil {
		item.DataShared = []string{}
	}
	if item.ComplianceFrameworks == nil {
		item.ComplianceFrameworks = []string{}
	}
	return item, nil
}

func (r *VCISOGovernanceRepository) UpdateVendor(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateVendorRequest) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `UPDATE vciso_vendors SET
		name=$3, category=$4, risk_tier=$5, status=$6, risk_score=$7,
		next_review_date=$8, contact_name=$9, contact_email=$10,
		services_provided=$11, data_shared=$12, compliance_frameworks=$13,
		controls_met=$14, controls_total=$15, updated_at=$16
		WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
		req.Name, req.Category, req.RiskTier, req.Status, req.RiskScore,
		req.NextReviewDate, req.ContactName, req.ContactEmail,
		req.ServicesProvided, req.DataShared, req.ComplianceFrameworks,
		req.ControlsMet, req.ControlsTotal, now,
	)
	if err != nil {
		return fmt.Errorf("update vendor: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) DeleteVendor(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM vciso_vendors WHERE id=$1 AND tenant_id=$2", id, tenantID)
	if err != nil {
		return fmt.Errorf("delete vendor: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *VCISOGovernanceRepository) UpdateVendorStatus(ctx context.Context, tenantID, id uuid.UUID, req *dto.UpdateVendorStatusRequest) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `UPDATE vciso_vendors SET status=$3, updated_at=$4 WHERE id=$1 AND tenant_id=$2`,
		id, tenantID, req.Status, now,
	)
	if err != nil {
		return fmt.Errorf("update vendor status: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) VendorStats(ctx context.Context, tenantID uuid.UUID) (*dto.VendorStatsResponse, error) {
	stats := &dto.VendorStatsResponse{
		ByRiskTier: make(map[string]int),
		ByStatus:   make(map[string]int),
	}
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*), COALESCE(AVG(risk_score),0),
		COUNT(*) FILTER (WHERE next_review_date < NOW()::text)
		FROM vciso_vendors WHERE tenant_id=$1`, tenantID,
	).Scan(&stats.Total, &stats.AvgRiskScore, &stats.OverdueReviews)

	rows, err := r.db.Query(ctx, "SELECT risk_tier, COUNT(*) FROM vciso_vendors WHERE tenant_id=$1 GROUP BY risk_tier", tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var k string
			var v int
			_ = rows.Scan(&k, &v)
			stats.ByRiskTier[k] = v
		}
	}

	rows2, err := r.db.Query(ctx, "SELECT status, COUNT(*) FROM vciso_vendors WHERE tenant_id=$1 GROUP BY status", tenantID)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var k string
			var v int
			_ = rows2.Scan(&k, &v)
			stats.ByStatus[k] = v
		}
	}

	return stats, nil
}

// ─── Questionnaires ─────────────────────────────────────────────────────────

func (r *VCISOGovernanceRepository) ListQuestionnaires(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) ([]*model.VCISOQuestionnaire, int, error) {
	conds := []string{"tenant_id=$1"}
	args := []interface{}{tenantID}
	i := 2

	if params.Status != "" {
		conds = append(conds, fmt.Sprintf("status=$%d", i))
		args = append(args, params.Status)
		i++
	}
	if params.Search != "" {
		conds = append(conds, fmt.Sprintf("title ILIKE $%d", i))
		args = append(args, "%"+params.Search+"%")
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM vciso_questionnaires "+where, args...).Scan(&total)

	query := fmt.Sprintf(`SELECT id, tenant_id, title, type, status, vendor_id, vendor_name,
		total_questions, answered_questions, due_date, completed_at, score,
		assigned_to, assigned_to_name, created_at, updated_at
		FROM vciso_questionnaires %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, i, i+1)
	args = append(args, params.PerPage, params.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list questionnaires: %w", err)
	}
	defer rows.Close()

	var items []*model.VCISOQuestionnaire
	for rows.Next() {
		item := &model.VCISOQuestionnaire{}
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.Title, &item.Type, &item.Status, &item.VendorID, &item.VendorName,
			&item.TotalQuestions, &item.AnsweredQuestions, &item.DueDate, &item.CompletedAt, &item.Score,
			&item.AssignedTo, &item.AssignedToName, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan questionnaire: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *VCISOGovernanceRepository) CreateQuestionnaire(ctx context.Context, tenantID uuid.UUID, item *model.VCISOQuestionnaire) error {
	item.ID = uuid.New()
	now := time.Now().UTC()
	item.TenantID = tenantID
	item.CreatedAt = now
	item.UpdatedAt = now

	vendorID := dto.ParseOptionalUUID(func() *string {
		if item.VendorID != nil {
			s := item.VendorID.String()
			return &s
		}
		return nil
	}())
	assignedTo := dto.ParseOptionalUUID(func() *string {
		if item.AssignedTo != nil {
			s := item.AssignedTo.String()
			return &s
		}
		return nil
	}())

	_, err := r.db.Exec(ctx, `INSERT INTO vciso_questionnaires (
		id, tenant_id, title, type, status, vendor_id, vendor_name,
		total_questions, due_date, assigned_to, assigned_to_name, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		item.ID, item.TenantID, item.Title, item.Type, item.Status, vendorID, item.VendorName,
		item.TotalQuestions, item.DueDate, assignedTo, item.AssignedToName, item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create questionnaire: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) UpdateQuestionnaire(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateQuestionnaireRequest) error {
	now := time.Now().UTC()
	vendorID := dto.ParseOptionalUUID(req.VendorID)
	assignedTo := dto.ParseOptionalUUID(req.AssignedTo)

	_, err := r.db.Exec(ctx, `UPDATE vciso_questionnaires SET
		title=$3, type=$4, status=$5, vendor_id=$6, vendor_name=$7,
		total_questions=$8, due_date=$9, assigned_to=$10, assigned_to_name=$11, updated_at=$12
		WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
		req.Title, req.Type, req.Status, vendorID, req.VendorName,
		req.TotalQuestions, req.DueDate, assignedTo, req.AssignedToName, now,
	)
	if err != nil {
		return fmt.Errorf("update questionnaire: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) UpdateQuestionnaireStatus(ctx context.Context, tenantID, id uuid.UUID, req *dto.UpdateQuestionnaireStatusRequest) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `UPDATE vciso_questionnaires SET
		status=$3, answered_questions=COALESCE($4, answered_questions), score=COALESCE($5, score), updated_at=$6
		WHERE id=$1 AND tenant_id=$2`,
		id, tenantID, req.Status, req.AnsweredQuestions, req.Score, now,
	)
	if err != nil {
		return fmt.Errorf("update questionnaire status: %w", err)
	}
	return nil
}

// ─── Evidence ───────────────────────────────────────────────────────────────

func (r *VCISOGovernanceRepository) ListEvidence(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) ([]*model.VCISOEvidence, int, error) {
	conds := []string{"tenant_id=$1"}
	args := []interface{}{tenantID}
	i := 2

	if params.Status != "" {
		conds = append(conds, fmt.Sprintf("status=$%d", i))
		args = append(args, params.Status)
		i++
	}
	if params.Type != "" {
		conds = append(conds, fmt.Sprintf("type=$%d", i))
		args = append(args, params.Type)
		i++
	}
	if params.Search != "" {
		conds = append(conds, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", i, i))
		args = append(args, "%"+params.Search+"%")
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM vciso_evidence "+where, args...).Scan(&total)

	orderCol := "created_at"
	if params.Sort != "" {
		allowed := map[string]bool{"created_at": true, "updated_at": true, "title": true, "status": true}
		if allowed[params.Sort] {
			orderCol = params.Sort
		}
	}
	dir := "DESC"
	if strings.EqualFold(params.Order, "asc") {
		dir = "ASC"
	}

	query := fmt.Sprintf(`SELECT id, tenant_id, title, description, type, source, status,
		frameworks, control_ids, file_name, file_size, file_url,
		collected_at, expires_at, collector_name, last_verified_at, created_at, updated_at
		FROM vciso_evidence %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		where, orderCol, dir, i, i+1)
	args = append(args, params.PerPage, params.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list evidence: %w", err)
	}
	defer rows.Close()

	var items []*model.VCISOEvidence
	for rows.Next() {
		item := &model.VCISOEvidence{}
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.Title, &item.Description, &item.Type, &item.Source, &item.Status,
			&item.Frameworks, &item.ControlIDs, &item.FileName, &item.FileSize, &item.FileURL,
			&item.CollectedAt, &item.ExpiresAt, &item.CollectorName, &item.LastVerifiedAt, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan evidence: %w", err)
		}
		if item.Frameworks == nil {
			item.Frameworks = []string{}
		}
		if item.ControlIDs == nil {
			item.ControlIDs = []string{}
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *VCISOGovernanceRepository) CreateEvidence(ctx context.Context, tenantID uuid.UUID, item *model.VCISOEvidence) error {
	item.ID = uuid.New()
	now := time.Now().UTC()
	item.TenantID = tenantID
	item.CreatedAt = now
	item.UpdatedAt = now

	_, err := r.db.Exec(ctx, `INSERT INTO vciso_evidence (
		id, tenant_id, title, description, type, source, status,
		frameworks, control_ids, file_name, file_size, file_url,
		collected_at, expires_at, collector_name, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		item.ID, item.TenantID, item.Title, item.Description, item.Type, item.Source, item.Status,
		item.Frameworks, item.ControlIDs, item.FileName, item.FileSize, item.FileURL,
		item.CollectedAt, item.ExpiresAt, item.CollectorName, item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create evidence: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) GetEvidence(ctx context.Context, tenantID, id uuid.UUID) (*model.VCISOEvidence, error) {
	item := &model.VCISOEvidence{}
	err := r.db.QueryRow(ctx, `SELECT id, tenant_id, title, description, type, source, status,
		frameworks, control_ids, file_name, file_size, file_url,
		collected_at, expires_at, collector_name, last_verified_at, created_at, updated_at
		FROM vciso_evidence WHERE id=$1 AND tenant_id=$2`, id, tenantID,
	).Scan(
		&item.ID, &item.TenantID, &item.Title, &item.Description, &item.Type, &item.Source, &item.Status,
		&item.Frameworks, &item.ControlIDs, &item.FileName, &item.FileSize, &item.FileURL,
		&item.CollectedAt, &item.ExpiresAt, &item.CollectorName, &item.LastVerifiedAt, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get evidence: %w", err)
	}
	if item.Frameworks == nil {
		item.Frameworks = []string{}
	}
	if item.ControlIDs == nil {
		item.ControlIDs = []string{}
	}
	return item, nil
}

func (r *VCISOGovernanceRepository) UpdateEvidence(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateEvidenceRequest) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `UPDATE vciso_evidence SET
		title=$3, description=$4, type=$5, source=$6,
		frameworks=$7, control_ids=$8, file_name=$9, file_size=$10, file_url=$11,
		collected_at=$12, expires_at=$13, collector_name=$14, updated_at=$15
		WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
		req.Title, req.Description, req.Type, req.Source,
		req.Frameworks, req.ControlIDs, req.FileName, req.FileSize, req.FileURL,
		req.CollectedAt, req.ExpiresAt, req.CollectorName, now,
	)
	if err != nil {
		return fmt.Errorf("update evidence: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) DeleteEvidence(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM vciso_evidence WHERE id=$1 AND tenant_id=$2", id, tenantID)
	if err != nil {
		return fmt.Errorf("delete evidence: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *VCISOGovernanceRepository) VerifyEvidence(ctx context.Context, tenantID, id, userID uuid.UUID, status string) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `UPDATE vciso_evidence SET
		status=$3, last_verified_at=$4, updated_at=$4
		WHERE id=$1 AND tenant_id=$2`,
		id, tenantID, status, now,
	)
	if err != nil {
		return fmt.Errorf("verify evidence: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) EvidenceStats(ctx context.Context, tenantID uuid.UUID) (*model.VCISOEvidenceStats, error) {
	stats := &model.VCISOEvidenceStats{
		ByStatus: make(map[string]int),
		ByType:   make(map[string]int),
		BySource: make(map[string]int),
	}
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*),
		COUNT(*) FILTER (WHERE expires_at IS NOT NULL AND expires_at < NOW()),
		COUNT(*) FILTER (WHERE last_verified_at IS NULL OR last_verified_at < NOW() - INTERVAL '90 days')
		FROM vciso_evidence WHERE tenant_id=$1`, tenantID,
	).Scan(&stats.Total, &stats.ExpiredCount, &stats.StaleCount)

	rows, err := r.db.Query(ctx, "SELECT status, COUNT(*) FROM vciso_evidence WHERE tenant_id=$1 GROUP BY status", tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var k string
			var v int
			_ = rows.Scan(&k, &v)
			stats.ByStatus[k] = v
		}
	}

	rows2, err := r.db.Query(ctx, "SELECT type, COUNT(*) FROM vciso_evidence WHERE tenant_id=$1 GROUP BY type", tenantID)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var k string
			var v int
			_ = rows2.Scan(&k, &v)
			stats.ByType[k] = v
		}
	}

	rows3, err := r.db.Query(ctx, "SELECT source, COUNT(*) FROM vciso_evidence WHERE tenant_id=$1 GROUP BY source", tenantID)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var k string
			var v int
			_ = rows3.Scan(&k, &v)
			stats.BySource[k] = v
		}
	}

	return stats, nil
}

// ─── Maturity ───────────────────────────────────────────────────────────────

func (r *VCISOGovernanceRepository) ListMaturityAssessments(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) ([]*model.VCISOMaturityAssessment, int, error) {
	conds := []string{"tenant_id=$1"}
	args := []interface{}{tenantID}
	i := 2

	if params.Framework != "" {
		conds = append(conds, fmt.Sprintf("framework=$%d", i))
		args = append(args, params.Framework)
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM vciso_maturity_assessments "+where, args...).Scan(&total)

	query := fmt.Sprintf(`SELECT id, tenant_id, framework, status, overall_score, overall_level,
		dimensions, assessor_name, assessed_at, created_at, updated_at
		FROM vciso_maturity_assessments %s ORDER BY assessed_at DESC LIMIT $%d OFFSET $%d`,
		where, i, i+1)
	args = append(args, params.PerPage, params.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list maturity: %w", err)
	}
	defer rows.Close()

	var items []*model.VCISOMaturityAssessment
	for rows.Next() {
		item := &model.VCISOMaturityAssessment{}
		var dimensionsJSON []byte
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.Framework, &item.Status, &item.OverallScore, &item.OverallLevel,
			&dimensionsJSON, &item.AssessorName, &item.AssessedAt, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan maturity: %w", err)
		}
		if dimensionsJSON != nil {
			_ = json.Unmarshal(dimensionsJSON, &item.Dimensions)
		}
		if item.Dimensions == nil {
			item.Dimensions = []model.VCISOMaturityDimension{}
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *VCISOGovernanceRepository) CreateMaturityAssessment(ctx context.Context, tenantID uuid.UUID, item *model.VCISOMaturityAssessment) error {
	item.ID = uuid.New()
	now := time.Now().UTC()
	item.TenantID = tenantID
	item.CreatedAt = now
	item.UpdatedAt = now

	dimensionsJSON, _ := json.Marshal(item.Dimensions)

	_, err := r.db.Exec(ctx, `INSERT INTO vciso_maturity_assessments (
		id, tenant_id, framework, status, overall_score, overall_level,
		dimensions, assessor_name, assessed_at, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		item.ID, item.TenantID, item.Framework, item.Status, item.OverallScore, item.OverallLevel,
		dimensionsJSON, item.AssessorName, item.AssessedAt, item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create maturity: %w", err)
	}
	return nil
}

// ─── Benchmarks ─────────────────────────────────────────────────────────────

func (r *VCISOGovernanceRepository) ListBenchmarks(ctx context.Context, tenantID uuid.UUID, params *dto.BenchmarkListParams) ([]model.VCISOBenchmark, error) {
	// Return mock/seeded benchmark data comparing org to industry
	items := []model.VCISOBenchmark{
		{Dimension: "Identity & Access Management", Category: "security", OrganizationScore: 72, IndustryAverage: 65, IndustryTopQuartile: 82, PeerAverage: 68, Gap: 7},
		{Dimension: "Data Protection", Category: "security", OrganizationScore: 68, IndustryAverage: 70, IndustryTopQuartile: 85, PeerAverage: 71, Gap: -2},
		{Dimension: "Incident Response", Category: "operations", OrganizationScore: 75, IndustryAverage: 60, IndustryTopQuartile: 80, PeerAverage: 63, Gap: 15},
		{Dimension: "Vulnerability Management", Category: "security", OrganizationScore: 60, IndustryAverage: 62, IndustryTopQuartile: 78, PeerAverage: 64, Gap: -2},
		{Dimension: "Security Awareness", Category: "people", OrganizationScore: 55, IndustryAverage: 58, IndustryTopQuartile: 75, PeerAverage: 60, Gap: -3},
		{Dimension: "Cloud Security", Category: "security", OrganizationScore: 65, IndustryAverage: 63, IndustryTopQuartile: 80, PeerAverage: 66, Gap: 2},
		{Dimension: "Compliance", Category: "governance", OrganizationScore: 78, IndustryAverage: 72, IndustryTopQuartile: 88, PeerAverage: 74, Gap: 6},
		{Dimension: "Risk Management", Category: "governance", OrganizationScore: 70, IndustryAverage: 66, IndustryTopQuartile: 82, PeerAverage: 69, Gap: 4},
	}

	if params.Framework != "" || params.Category != "" {
		var filtered []model.VCISOBenchmark
		for _, b := range items {
			if params.Category != "" && b.Category != params.Category {
				continue
			}
			filtered = append(filtered, b)
		}
		items = filtered
	}

	return items, nil
}

// ─── Budget ─────────────────────────────────────────────────────────────────

func (r *VCISOGovernanceRepository) ListBudgetItems(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) ([]*model.VCISOBudgetItem, int, error) {
	conds := []string{"tenant_id=$1"}
	args := []interface{}{tenantID}
	i := 2

	if params.Status != "" {
		conds = append(conds, fmt.Sprintf("status=$%d", i))
		args = append(args, params.Status)
		i++
	}
	if params.Category != "" {
		conds = append(conds, fmt.Sprintf("category=$%d", i))
		args = append(args, params.Category)
		i++
	}
	if params.Search != "" {
		conds = append(conds, fmt.Sprintf("title ILIKE $%d", i))
		args = append(args, "%"+params.Search+"%")
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM vciso_budget_items "+where, args...).Scan(&total)

	query := fmt.Sprintf(`SELECT id, tenant_id, title, category, type, amount, currency, status,
		risk_reduction_estimate, priority, justification, linked_risk_ids, linked_recommendation_ids,
		fiscal_year, quarter, owner_name, created_at, updated_at
		FROM vciso_budget_items %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, i, i+1)
	args = append(args, params.PerPage, params.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list budget: %w", err)
	}
	defer rows.Close()

	var items []*model.VCISOBudgetItem
	for rows.Next() {
		item := &model.VCISOBudgetItem{}
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.Title, &item.Category, &item.Type, &item.Amount, &item.Currency, &item.Status,
			&item.RiskReductionEstimate, &item.Priority, &item.Justification, &item.LinkedRiskIDs, &item.LinkedRecommendationIDs,
			&item.FiscalYear, &item.Quarter, &item.OwnerName, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan budget: %w", err)
		}
		if item.LinkedRiskIDs == nil {
			item.LinkedRiskIDs = []string{}
		}
		if item.LinkedRecommendationIDs == nil {
			item.LinkedRecommendationIDs = []string{}
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *VCISOGovernanceRepository) CreateBudgetItem(ctx context.Context, tenantID uuid.UUID, item *model.VCISOBudgetItem) error {
	item.ID = uuid.New()
	now := time.Now().UTC()
	item.TenantID = tenantID
	item.CreatedAt = now
	item.UpdatedAt = now

	_, err := r.db.Exec(ctx, `INSERT INTO vciso_budget_items (
		id, tenant_id, title, category, type, amount, currency, status,
		risk_reduction_estimate, priority, justification, linked_risk_ids, linked_recommendation_ids,
		fiscal_year, quarter, owner_name, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		item.ID, item.TenantID, item.Title, item.Category, item.Type, item.Amount, item.Currency, item.Status,
		item.RiskReductionEstimate, item.Priority, item.Justification, item.LinkedRiskIDs, item.LinkedRecommendationIDs,
		item.FiscalYear, item.Quarter, item.OwnerName, item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create budget: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) UpdateBudgetItem(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateBudgetItemRequest) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `UPDATE vciso_budget_items SET
		title=$3, category=$4, type=$5, amount=$6, currency=$7, status=$8,
		risk_reduction_estimate=$9, priority=$10, justification=$11,
		linked_risk_ids=$12, linked_recommendation_ids=$13,
		fiscal_year=$14, quarter=$15, owner_name=$16, updated_at=$17
		WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
		req.Title, req.Category, req.Type, req.Amount, req.Currency, req.Status,
		req.RiskReductionEstimate, req.Priority, req.Justification,
		req.LinkedRiskIDs, req.LinkedRecommendationIDs,
		req.FiscalYear, req.Quarter, req.OwnerName, now,
	)
	if err != nil {
		return fmt.Errorf("update budget: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) DeleteBudgetItem(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM vciso_budget_items WHERE id=$1 AND tenant_id=$2", id, tenantID)
	if err != nil {
		return fmt.Errorf("delete budget: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *VCISOGovernanceRepository) BudgetSummary(ctx context.Context, tenantID uuid.UUID) (*dto.BudgetSummaryResponse, error) {
	summary := &dto.BudgetSummaryResponse{
		ByCategory: make(map[string]float64),
		ByStatus:   make(map[string]float64),
		Currency:   "USD",
	}
	_ = r.db.QueryRow(ctx, `SELECT
		COALESCE(SUM(amount),0),
		COALESCE(SUM(amount) FILTER (WHERE status='approved'),0),
		COALESCE(SUM(amount) FILTER (WHERE status='spent'),0),
		COALESCE(SUM(risk_reduction_estimate),0)
		FROM vciso_budget_items WHERE tenant_id=$1`, tenantID,
	).Scan(&summary.TotalProposed, &summary.TotalApproved, &summary.TotalSpent, &summary.TotalRiskReduction)

	rows, err := r.db.Query(ctx, "SELECT category, SUM(amount) FROM vciso_budget_items WHERE tenant_id=$1 GROUP BY category", tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var k string
			var v float64
			_ = rows.Scan(&k, &v)
			summary.ByCategory[k] = v
		}
	}

	rows2, err := r.db.Query(ctx, "SELECT status, SUM(amount) FROM vciso_budget_items WHERE tenant_id=$1 GROUP BY status", tenantID)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var k string
			var v float64
			_ = rows2.Scan(&k, &v)
			summary.ByStatus[k] = v
		}
	}

	return summary, nil
}

// ─── Awareness ──────────────────────────────────────────────────────────────

func (r *VCISOGovernanceRepository) ListAwarenessPrograms(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) ([]*model.VCISOAwarenessProgram, int, error) {
	conds := []string{"tenant_id=$1"}
	args := []interface{}{tenantID}
	i := 2

	if params.Status != "" {
		conds = append(conds, fmt.Sprintf("status=$%d", i))
		args = append(args, params.Status)
		i++
	}
	if params.Search != "" {
		conds = append(conds, fmt.Sprintf("name ILIKE $%d", i))
		args = append(args, "%"+params.Search+"%")
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM vciso_awareness_programs "+where, args...).Scan(&total)

	query := fmt.Sprintf(`SELECT id, tenant_id, name, type, status, total_users, completed_users,
		passed_users, failed_users, completion_rate, pass_rate, start_date, end_date, created_at, updated_at
		FROM vciso_awareness_programs %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, i, i+1)
	args = append(args, params.PerPage, params.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list awareness: %w", err)
	}
	defer rows.Close()

	var items []*model.VCISOAwarenessProgram
	for rows.Next() {
		item := &model.VCISOAwarenessProgram{}
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.Name, &item.Type, &item.Status, &item.TotalUsers, &item.CompletedUsers,
			&item.PassedUsers, &item.FailedUsers, &item.CompletionRate, &item.PassRate, &item.StartDate, &item.EndDate,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan awareness: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *VCISOGovernanceRepository) CreateAwarenessProgram(ctx context.Context, tenantID uuid.UUID, item *model.VCISOAwarenessProgram) error {
	item.ID = uuid.New()
	now := time.Now().UTC()
	item.TenantID = tenantID
	item.CreatedAt = now
	item.UpdatedAt = now
	if item.TotalUsers > 0 {
		item.CompletionRate = float64(item.CompletedUsers) / float64(item.TotalUsers) * 100
		item.PassRate = float64(item.PassedUsers) / float64(item.TotalUsers) * 100
	}

	_, err := r.db.Exec(ctx, `INSERT INTO vciso_awareness_programs (
		id, tenant_id, name, type, status, total_users, completed_users,
		passed_users, failed_users, completion_rate, pass_rate, start_date, end_date, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		item.ID, item.TenantID, item.Name, item.Type, item.Status, item.TotalUsers, item.CompletedUsers,
		item.PassedUsers, item.FailedUsers, item.CompletionRate, item.PassRate, item.StartDate, item.EndDate,
		item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create awareness: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) UpdateAwarenessProgram(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateAwarenessProgramRequest) error {
	now := time.Now().UTC()
	completionRate := 0.0
	passRate := 0.0
	if req.TotalUsers > 0 {
		completionRate = float64(req.CompletedUsers) / float64(req.TotalUsers) * 100
		passRate = float64(req.PassedUsers) / float64(req.TotalUsers) * 100
	}
	_, err := r.db.Exec(ctx, `UPDATE vciso_awareness_programs SET
		name=$3, type=$4, status=$5, total_users=$6, completed_users=$7,
		passed_users=$8, failed_users=$9, completion_rate=$10, pass_rate=$11,
		start_date=$12, end_date=$13, updated_at=$14
		WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
		req.Name, req.Type, req.Status, req.TotalUsers, req.CompletedUsers,
		req.PassedUsers, req.FailedUsers, completionRate, passRate,
		req.StartDate, req.EndDate, now,
	)
	if err != nil {
		return fmt.Errorf("update awareness: %w", err)
	}
	return nil
}

// ─── IAM Findings ───────────────────────────────────────────────────────────

func (r *VCISOGovernanceRepository) ListIAMFindings(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) ([]*model.VCISOIAMFinding, int, error) {
	conds := []string{"tenant_id=$1"}
	args := []interface{}{tenantID}
	i := 2

	if params.Status != "" {
		conds = append(conds, fmt.Sprintf("status=$%d", i))
		args = append(args, params.Status)
		i++
	}
	if params.Type != "" {
		conds = append(conds, fmt.Sprintf("type=$%d", i))
		args = append(args, params.Type)
		i++
	}
	if params.Search != "" {
		conds = append(conds, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", i, i))
		args = append(args, "%"+params.Search+"%")
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM vciso_iam_findings "+where, args...).Scan(&total)

	query := fmt.Sprintf(`SELECT id, tenant_id, type, severity, title, description,
		affected_users, status, remediation, discovered_at, resolved_at, created_at, updated_at
		FROM vciso_iam_findings %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, i, i+1)
	args = append(args, params.PerPage, params.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list iam findings: %w", err)
	}
	defer rows.Close()

	var items []*model.VCISOIAMFinding
	for rows.Next() {
		item := &model.VCISOIAMFinding{}
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.Type, &item.Severity, &item.Title, &item.Description,
			&item.AffectedUsers, &item.Status, &item.Remediation, &item.DiscoveredAt, &item.ResolvedAt,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan iam finding: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *VCISOGovernanceRepository) UpdateIAMFinding(ctx context.Context, tenantID, id uuid.UUID, req *dto.UpdateIAMFindingRequest) error {
	now := time.Now().UTC()
	var resolvedAt *time.Time
	if req.Status == "remediated" || req.Status == "accepted" {
		resolvedAt = &now
	}
	_, err := r.db.Exec(ctx, `UPDATE vciso_iam_findings SET
		status=$3, remediation=$4, resolved_at=$5, updated_at=$6
		WHERE id=$1 AND tenant_id=$2`,
		id, tenantID, req.Status, req.Remediation, resolvedAt, now,
	)
	if err != nil {
		return fmt.Errorf("update iam finding: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) IAMFindingSummary(ctx context.Context, tenantID uuid.UUID) (*model.VCISOIAMSummary, error) {
	summary := &model.VCISOIAMSummary{
		ByType:     make(map[string]int),
		BySeverity: make(map[string]int),
	}
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*),
		COUNT(*) FILTER (WHERE type='over_privileged'),
		COUNT(*) FILTER (WHERE type='orphaned_account'),
		COUNT(*) FILTER (WHERE type='stale_access')
		FROM vciso_iam_findings WHERE tenant_id=$1`, tenantID,
	).Scan(&summary.TotalFindings, &summary.PrivilegedAccounts, &summary.OrphanedAccounts, &summary.StaleAccessCount)

	rows, err := r.db.Query(ctx, "SELECT type, COUNT(*) FROM vciso_iam_findings WHERE tenant_id=$1 GROUP BY type", tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var k string
			var v int
			_ = rows.Scan(&k, &v)
			summary.ByType[k] = v
		}
	}

	rows2, err := r.db.Query(ctx, "SELECT severity, COUNT(*) FROM vciso_iam_findings WHERE tenant_id=$1 GROUP BY severity", tenantID)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var k string
			var v int
			_ = rows2.Scan(&k, &v)
			summary.BySeverity[k] = v
		}
	}

	return summary, nil
}

// ─── Escalation Rules ───────────────────────────────────────────────────────

func (r *VCISOGovernanceRepository) ListEscalationRules(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) ([]*model.VCISOEscalationRule, int, error) {
	conds := []string{"tenant_id=$1"}
	args := []interface{}{tenantID}
	i := 2

	if params.Search != "" {
		conds = append(conds, fmt.Sprintf("name ILIKE $%d", i))
		args = append(args, "%"+params.Search+"%")
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM vciso_escalation_rules "+where, args...).Scan(&total)

	query := fmt.Sprintf(`SELECT id, tenant_id, name, description, trigger_type, trigger_condition,
		escalation_target, target_contacts, notification_channels, enabled,
		last_triggered_at, trigger_count, created_at, updated_at
		FROM vciso_escalation_rules %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, i, i+1)
	args = append(args, params.PerPage, params.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list escalation rules: %w", err)
	}
	defer rows.Close()

	var items []*model.VCISOEscalationRule
	for rows.Next() {
		item := &model.VCISOEscalationRule{}
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.Name, &item.Description, &item.TriggerType, &item.TriggerCondition,
			&item.EscalationTarget, &item.TargetContacts, &item.NotificationChannels, &item.Enabled,
			&item.LastTriggeredAt, &item.TriggerCount, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan escalation rule: %w", err)
		}
		if item.TargetContacts == nil {
			item.TargetContacts = []string{}
		}
		if item.NotificationChannels == nil {
			item.NotificationChannels = []string{}
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *VCISOGovernanceRepository) CreateEscalationRule(ctx context.Context, tenantID uuid.UUID, item *model.VCISOEscalationRule) error {
	item.ID = uuid.New()
	now := time.Now().UTC()
	item.TenantID = tenantID
	item.CreatedAt = now
	item.UpdatedAt = now

	_, err := r.db.Exec(ctx, `INSERT INTO vciso_escalation_rules (
		id, tenant_id, name, description, trigger_type, trigger_condition,
		escalation_target, target_contacts, notification_channels, enabled, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		item.ID, item.TenantID, item.Name, item.Description, item.TriggerType, item.TriggerCondition,
		item.EscalationTarget, item.TargetContacts, item.NotificationChannels, item.Enabled,
		item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create escalation rule: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) UpdateEscalationRule(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateEscalationRuleRequest) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `UPDATE vciso_escalation_rules SET
		name=$3, description=$4, trigger_type=$5, trigger_condition=$6,
		escalation_target=$7, target_contacts=$8, notification_channels=$9, enabled=$10, updated_at=$11
		WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
		req.Name, req.Description, req.TriggerType, req.TriggerCondition,
		req.EscalationTarget, req.TargetContacts, req.NotificationChannels, req.Enabled, now,
	)
	if err != nil {
		return fmt.Errorf("update escalation rule: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) DeleteEscalationRule(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM vciso_escalation_rules WHERE id=$1 AND tenant_id=$2", id, tenantID)
	if err != nil {
		return fmt.Errorf("delete escalation rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Playbooks ──────────────────────────────────────────────────────────────

func (r *VCISOGovernanceRepository) ListPlaybooks(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) ([]*model.VCISOPlaybook, int, error) {
	conds := []string{"tenant_id=$1"}
	args := []interface{}{tenantID}
	i := 2

	if params.Status != "" {
		conds = append(conds, fmt.Sprintf("status=$%d", i))
		args = append(args, params.Status)
		i++
	}
	if params.Search != "" {
		conds = append(conds, fmt.Sprintf("name ILIKE $%d", i))
		args = append(args, "%"+params.Search+"%")
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM vciso_playbooks "+where, args...).Scan(&total)

	query := fmt.Sprintf(`SELECT id, tenant_id, name, scenario, status, last_tested_at, next_test_date,
		owner_id, owner_name, steps_count, dependencies, rto_hours, rpo_hours,
		last_simulation_result, created_at, updated_at
		FROM vciso_playbooks %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, i, i+1)
	args = append(args, params.PerPage, params.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list playbooks: %w", err)
	}
	defer rows.Close()

	var items []*model.VCISOPlaybook
	for rows.Next() {
		item := &model.VCISOPlaybook{}
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.Name, &item.Scenario, &item.Status, &item.LastTestedAt, &item.NextTestDate,
			&item.OwnerID, &item.OwnerName, &item.StepsCount, &item.Dependencies, &item.RTOHours, &item.RPOHours,
			&item.LastSimulationResult, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan playbook: %w", err)
		}
		if item.Dependencies == nil {
			item.Dependencies = []string{}
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *VCISOGovernanceRepository) CreatePlaybook(ctx context.Context, tenantID uuid.UUID, item *model.VCISOPlaybook) error {
	item.ID = uuid.New()
	now := time.Now().UTC()
	item.TenantID = tenantID
	item.CreatedAt = now
	item.UpdatedAt = now

	_, err := r.db.Exec(ctx, `INSERT INTO vciso_playbooks (
		id, tenant_id, name, scenario, status, next_test_date,
		owner_id, owner_name, steps_count, dependencies, rto_hours, rpo_hours, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		item.ID, item.TenantID, item.Name, item.Scenario, item.Status, item.NextTestDate,
		item.OwnerID, item.OwnerName, item.StepsCount, item.Dependencies, item.RTOHours, item.RPOHours,
		item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create playbook: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) UpdatePlaybook(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreatePlaybookRequest) error {
	now := time.Now().UTC()
	ownerID, _ := uuid.Parse(req.OwnerID)
	_, err := r.db.Exec(ctx, `UPDATE vciso_playbooks SET
		name=$3, scenario=$4, status=$5, next_test_date=$6,
		owner_id=$7, owner_name=$8, steps_count=$9, dependencies=$10, rto_hours=$11, rpo_hours=$12, updated_at=$13
		WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
		req.Name, req.Scenario, req.Status, req.NextTestDate,
		ownerID, req.OwnerName, req.StepsCount, req.Dependencies, req.RTOHours, req.RPOHours, now,
	)
	if err != nil {
		return fmt.Errorf("update playbook: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) DeletePlaybook(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM vciso_playbooks WHERE id=$1 AND tenant_id=$2", id, tenantID)
	if err != nil {
		return fmt.Errorf("delete playbook: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *VCISOGovernanceRepository) SimulatePlaybook(ctx context.Context, tenantID, id uuid.UUID, result string) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `UPDATE vciso_playbooks SET
		last_tested_at=$3, last_simulation_result=$4, updated_at=$3
		WHERE id=$1 AND tenant_id=$2`,
		id, tenantID, now, result,
	)
	if err != nil {
		return fmt.Errorf("simulate playbook: %w", err)
	}
	return nil
}

// ─── Obligations ────────────────────────────────────────────────────────────

func (r *VCISOGovernanceRepository) ListObligations(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) ([]*model.VCISORegulatoryObligation, int, error) {
	conds := []string{"tenant_id=$1"}
	args := []interface{}{tenantID}
	i := 2

	if params.Status != "" {
		conds = append(conds, fmt.Sprintf("status=$%d", i))
		args = append(args, params.Status)
		i++
	}
	if params.Type != "" {
		conds = append(conds, fmt.Sprintf("type=$%d", i))
		args = append(args, params.Type)
		i++
	}
	if params.Search != "" {
		conds = append(conds, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", i, i))
		args = append(args, "%"+params.Search+"%")
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM vciso_obligations "+where, args...).Scan(&total)

	query := fmt.Sprintf(`SELECT id, tenant_id, name, type, jurisdiction, description,
		requirements, status, mapped_controls, total_requirements, met_requirements,
		owner_id, owner_name, effective_date, review_date, created_at, updated_at
		FROM vciso_obligations %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, i, i+1)
	args = append(args, params.PerPage, params.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list obligations: %w", err)
	}
	defer rows.Close()

	var items []*model.VCISORegulatoryObligation
	for rows.Next() {
		item := &model.VCISORegulatoryObligation{}
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.Name, &item.Type, &item.Jurisdiction, &item.Description,
			&item.Requirements, &item.Status, &item.MappedControls, &item.TotalRequirements, &item.MetRequirements,
			&item.OwnerID, &item.OwnerName, &item.EffectiveDate, &item.ReviewDate, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan obligation: %w", err)
		}
		if item.Requirements == nil {
			item.Requirements = []string{}
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *VCISOGovernanceRepository) CreateObligation(ctx context.Context, tenantID uuid.UUID, item *model.VCISORegulatoryObligation) error {
	item.ID = uuid.New()
	now := time.Now().UTC()
	item.TenantID = tenantID
	item.CreatedAt = now
	item.UpdatedAt = now

	_, err := r.db.Exec(ctx, `INSERT INTO vciso_obligations (
		id, tenant_id, name, type, jurisdiction, description,
		requirements, status, mapped_controls, total_requirements, met_requirements,
		owner_id, owner_name, effective_date, review_date, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		item.ID, item.TenantID, item.Name, item.Type, item.Jurisdiction, item.Description,
		item.Requirements, item.Status, item.MappedControls, item.TotalRequirements, item.MetRequirements,
		item.OwnerID, item.OwnerName, item.EffectiveDate, item.ReviewDate, item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create obligation: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) UpdateObligation(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateObligationRequest) error {
	now := time.Now().UTC()
	ownerID := dto.ParseOptionalUUID(req.OwnerID)
	_, err := r.db.Exec(ctx, `UPDATE vciso_obligations SET
		name=$3, type=$4, jurisdiction=$5, description=$6,
		requirements=$7, status=$8, mapped_controls=$9, total_requirements=$10, met_requirements=$11,
		owner_id=$12, owner_name=$13, effective_date=$14, review_date=$15, updated_at=$16
		WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
		req.Name, req.Type, req.Jurisdiction, req.Description,
		req.Requirements, req.Status, req.MappedControls, req.TotalRequirements, req.MetRequirements,
		ownerID, req.OwnerName, req.EffectiveDate, req.ReviewDate, now,
	)
	if err != nil {
		return fmt.Errorf("update obligation: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) DeleteObligation(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM vciso_obligations WHERE id=$1 AND tenant_id=$2", id, tenantID)
	if err != nil {
		return fmt.Errorf("delete obligation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Control Tests ──────────────────────────────────────────────────────────

func (r *VCISOGovernanceRepository) ListControlTests(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) ([]*model.VCISOControlTest, int, error) {
	conds := []string{"tenant_id=$1"}
	args := []interface{}{tenantID}
	i := 2

	if params.Framework != "" {
		conds = append(conds, fmt.Sprintf("framework=$%d", i))
		args = append(args, params.Framework)
		i++
	}
	if params.Search != "" {
		conds = append(conds, fmt.Sprintf("(control_name ILIKE $%d OR test_name ILIKE $%d)", i, i))
		args = append(args, "%"+params.Search+"%")
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM vciso_control_tests "+where, args...).Scan(&total)

	query := fmt.Sprintf(`SELECT id, tenant_id, control_id, control_name, framework, test_type,
		result, tester_name, test_date, next_test_date, findings, evidence_ids, created_at, updated_at
		FROM vciso_control_tests %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, i, i+1)
	args = append(args, params.PerPage, params.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list control tests: %w", err)
	}
	defer rows.Close()

	var items []*model.VCISOControlTest
	for rows.Next() {
		item := &model.VCISOControlTest{}
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.ControlID, &item.ControlName, &item.Framework, &item.TestType,
			&item.Result, &item.TesterName, &item.TestDate, &item.NextTestDate, &item.Findings, &item.EvidenceIDs,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan control test: %w", err)
		}
		if item.EvidenceIDs == nil {
			item.EvidenceIDs = []string{}
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *VCISOGovernanceRepository) CreateControlTest(ctx context.Context, tenantID uuid.UUID, item *model.VCISOControlTest) error {
	item.ID = uuid.New()
	now := time.Now().UTC()
	item.TenantID = tenantID
	item.CreatedAt = now
	item.UpdatedAt = now

	_, err := r.db.Exec(ctx, `INSERT INTO vciso_control_tests (
		id, tenant_id, control_id, control_name, framework, test_type,
		result, tester_name, test_date, next_test_date, findings, evidence_ids, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		item.ID, item.TenantID, item.ControlID, item.ControlName, item.Framework, item.TestType,
		item.Result, item.TesterName, item.TestDate, item.NextTestDate, item.Findings, item.EvidenceIDs,
		item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create control test: %w", err)
	}
	return nil
}

// ─── Control Dependencies ───────────────────────────────────────────────────

func (r *VCISOGovernanceRepository) ListControlDependencies(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) ([]model.VCISOControlDependency, int, error) {
	// Return mock control dependency graph data
	items := []model.VCISOControlDependency{
		{ControlID: "AC-1", ControlName: "Access Control Policy", Framework: "NIST", DependsOn: []string{}, DependedBy: []string{"AC-2", "AC-3"}, RiskDomains: []string{"identity"}, ComplianceDomains: []string{"access_control"}, FailureImpact: "high"},
		{ControlID: "AC-2", ControlName: "Account Management", Framework: "NIST", DependsOn: []string{"AC-1"}, DependedBy: []string{"AC-6"}, RiskDomains: []string{"identity"}, ComplianceDomains: []string{"access_control"}, FailureImpact: "high"},
		{ControlID: "AC-3", ControlName: "Access Enforcement", Framework: "NIST", DependsOn: []string{"AC-1"}, DependedBy: []string{"AC-6"}, RiskDomains: []string{"identity"}, ComplianceDomains: []string{"access_control"}, FailureImpact: "critical"},
		{ControlID: "IR-1", ControlName: "Incident Response Policy", Framework: "NIST", DependsOn: []string{}, DependedBy: []string{"IR-4", "IR-5"}, RiskDomains: []string{"operations"}, ComplianceDomains: []string{"incident_response"}, FailureImpact: "high"},
		{ControlID: "RA-1", ControlName: "Risk Assessment Policy", Framework: "NIST", DependsOn: []string{}, DependedBy: []string{"RA-3", "RA-5"}, RiskDomains: []string{"governance"}, ComplianceDomains: []string{"risk_assessment"}, FailureImpact: "medium"},
	}

	if params.Framework != "" {
		var filtered []model.VCISOControlDependency
		for _, d := range items {
			if d.Framework == params.Framework {
				filtered = append(filtered, d)
			}
		}
		items = filtered
	}

	total := len(items)
	return items, total, nil
}

// ─── Integrations ───────────────────────────────────────────────────────────

func (r *VCISOGovernanceRepository) ListIntegrations(ctx context.Context, tenantID uuid.UUID) ([]*model.VCISOIntegration, error) {
	rows, err := r.db.Query(ctx, `SELECT id, tenant_id, name, type, provider, status,
		last_sync_at, sync_frequency, items_synced, config, health_status, error_message,
		created_at, updated_at
		FROM vciso_integrations WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list integrations: %w", err)
	}
	defer rows.Close()

	var items []*model.VCISOIntegration
	for rows.Next() {
		item := &model.VCISOIntegration{}
		var configJSON []byte
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.Name, &item.Type, &item.Provider, &item.Status,
			&item.LastSyncAt, &item.SyncFrequency, &item.ItemsSynced, &configJSON, &item.HealthStatus, &item.ErrorMessage,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan integration: %w", err)
		}
		if configJSON != nil {
			_ = json.Unmarshal(configJSON, &item.Config)
		}
		if item.Config == nil {
			item.Config = make(map[string]interface{})
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *VCISOGovernanceRepository) CreateIntegration(ctx context.Context, tenantID uuid.UUID, item *model.VCISOIntegration) error {
	item.ID = uuid.New()
	now := time.Now().UTC()
	item.TenantID = tenantID
	item.HealthStatus = "healthy"
	item.CreatedAt = now
	item.UpdatedAt = now

	configJSON, _ := json.Marshal(item.Config)

	_, err := r.db.Exec(ctx, `INSERT INTO vciso_integrations (
		id, tenant_id, name, type, provider, status,
		sync_frequency, config, health_status, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		item.ID, item.TenantID, item.Name, item.Type, item.Provider, item.Status,
		item.SyncFrequency, configJSON, item.HealthStatus, item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create integration: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) UpdateIntegration(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateIntegrationRequest) error {
	now := time.Now().UTC()
	configJSON, _ := json.Marshal(req.Config)
	_, err := r.db.Exec(ctx, `UPDATE vciso_integrations SET
		name=$3, type=$4, provider=$5, status=$6, sync_frequency=$7, config=$8, updated_at=$9
		WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
		req.Name, req.Type, req.Provider, req.Status, req.SyncFrequency, configJSON, now,
	)
	if err != nil {
		return fmt.Errorf("update integration: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) DeleteIntegration(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM vciso_integrations WHERE id=$1 AND tenant_id=$2", id, tenantID)
	if err != nil {
		return fmt.Errorf("delete integration: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *VCISOGovernanceRepository) SyncIntegration(ctx context.Context, tenantID, id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `UPDATE vciso_integrations SET last_sync_at=$3, updated_at=$3 WHERE id=$1 AND tenant_id=$2`,
		id, tenantID, now,
	)
	if err != nil {
		return fmt.Errorf("sync integration: %w", err)
	}
	return nil
}

// ─── Control Ownership ──────────────────────────────────────────────────────

func (r *VCISOGovernanceRepository) ListControlOwnerships(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) ([]*model.VCISOControlOwnership, int, error) {
	conds := []string{"tenant_id=$1"}
	args := []interface{}{tenantID}
	i := 2

	if params.Framework != "" {
		conds = append(conds, fmt.Sprintf("framework=$%d", i))
		args = append(args, params.Framework)
		i++
	}
	if params.Search != "" {
		conds = append(conds, fmt.Sprintf("(control_name ILIKE $%d OR owner_name ILIKE $%d)", i, i))
		args = append(args, "%"+params.Search+"%")
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM vciso_control_ownership "+where, args...).Scan(&total)

	query := fmt.Sprintf(`SELECT id, tenant_id, control_id, control_name, framework,
		owner_id, owner_name, delegate_id, delegate_name, status,
		last_reviewed_at, next_review_date, created_at, updated_at
		FROM vciso_control_ownership %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, i, i+1)
	args = append(args, params.PerPage, params.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list control ownership: %w", err)
	}
	defer rows.Close()

	var items []*model.VCISOControlOwnership
	for rows.Next() {
		item := &model.VCISOControlOwnership{}
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.ControlID, &item.ControlName, &item.Framework,
			&item.OwnerID, &item.OwnerName, &item.DelegateID, &item.DelegateName, &item.Status,
			&item.LastReviewedAt, &item.NextReviewDate, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan control ownership: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *VCISOGovernanceRepository) CreateControlOwnership(ctx context.Context, tenantID uuid.UUID, item *model.VCISOControlOwnership) error {
	item.ID = uuid.New()
	now := time.Now().UTC()
	item.TenantID = tenantID
	item.CreatedAt = now
	item.UpdatedAt = now

	_, err := r.db.Exec(ctx, `INSERT INTO vciso_control_ownership (
		id, tenant_id, control_id, control_name, framework,
		owner_id, owner_name, delegate_id, delegate_name, status,
		next_review_date, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		item.ID, item.TenantID, item.ControlID, item.ControlName, item.Framework,
		item.OwnerID, item.OwnerName, item.DelegateID, item.DelegateName, item.Status,
		item.NextReviewDate, item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create control ownership: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) UpdateControlOwnership(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateControlOwnershipRequest) error {
	now := time.Now().UTC()
	ownerID, _ := uuid.Parse(req.OwnerID)
	delegateID := dto.ParseOptionalUUID(req.DelegateID)
	_, err := r.db.Exec(ctx, `UPDATE vciso_control_ownership SET
		control_id=$3, control_name=$4, framework=$5,
		owner_id=$6, owner_name=$7, delegate_id=$8, delegate_name=$9, status=$10,
		next_review_date=$11, updated_at=$12
		WHERE id=$1 AND tenant_id=$2`,
		id, tenantID,
		req.ControlID, req.ControlName, req.Framework,
		ownerID, req.OwnerName, delegateID, req.DelegateName, req.Status,
		req.NextReviewDate, now,
	)
	if err != nil {
		return fmt.Errorf("update control ownership: %w", err)
	}
	return nil
}

func (r *VCISOGovernanceRepository) MarkControlOwnershipReviewed(ctx context.Context, tenantID, id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `UPDATE vciso_control_ownership SET last_reviewed_at=$3, updated_at=$3 WHERE id=$1 AND tenant_id=$2`,
		id, tenantID, now,
	)
	if err != nil {
		return fmt.Errorf("mark control ownership reviewed: %w", err)
	}
	return nil
}

// ─── Approvals ──────────────────────────────────────────────────────────────

func (r *VCISOGovernanceRepository) ListApprovals(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) ([]*model.VCISOApprovalRequest, int, error) {
	conds := []string{"tenant_id=$1"}
	args := []interface{}{tenantID}
	i := 2

	if params.Status != "" {
		conds = append(conds, fmt.Sprintf("status=$%d", i))
		args = append(args, params.Status)
		i++
	}
	if params.Type != "" {
		conds = append(conds, fmt.Sprintf("type=$%d", i))
		args = append(args, params.Type)
		i++
	}
	if params.Search != "" {
		conds = append(conds, fmt.Sprintf("title ILIKE $%d", i))
		args = append(args, "%"+params.Search+"%")
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM vciso_approvals "+where, args...).Scan(&total)

	query := fmt.Sprintf(`SELECT id, tenant_id, type, title, description, status,
		requested_by, requested_by_name, approver_id, approver_name, priority,
		decision_notes, decided_at, deadline, linked_entity_type, linked_entity_id,
		created_at, updated_at
		FROM vciso_approvals %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, i, i+1)
	args = append(args, params.PerPage, params.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list approvals: %w", err)
	}
	defer rows.Close()

	var items []*model.VCISOApprovalRequest
	for rows.Next() {
		item := &model.VCISOApprovalRequest{}
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.Type, &item.Title, &item.Description, &item.Status,
			&item.RequestedBy, &item.RequestedByName, &item.ApproverID, &item.ApproverName, &item.Priority,
			&item.DecisionNotes, &item.DecidedAt, &item.Deadline, &item.LinkedEntityType, &item.LinkedEntityID,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan approval: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *VCISOGovernanceRepository) DecideApproval(ctx context.Context, tenantID, id, userID uuid.UUID, req *dto.UpdateApprovalRequest) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(ctx, `UPDATE vciso_approvals SET
		status=$3, decision_notes=$4, decided_at=$5, updated_at=$5
		WHERE id=$1 AND tenant_id=$2`,
		id, tenantID, req.Status, req.DecisionNotes, now,
	)
	if err != nil {
		return fmt.Errorf("decide approval: %w", err)
	}
	return nil
}

// ─── Generate Policy (AI mock) ──────────────────────────────────────────────

func (r *VCISOGovernanceRepository) GeneratePolicyContent(ctx context.Context, tenantID uuid.UUID, domain string) (string, error) {
	content := fmt.Sprintf(`# %s Policy

## Purpose
This policy establishes the requirements for %s within the organization.

## Scope
This policy applies to all employees, contractors, and third-party personnel.

## Policy Statements
1. All personnel must comply with the organization's %s requirements.
2. Regular assessments and reviews shall be conducted.
3. Non-compliance must be reported and remediated promptly.

## Responsibilities
- CISO: Overall policy oversight
- Department Heads: Implementation within their teams
- All Staff: Compliance with policy requirements

## Review
This policy shall be reviewed annually or upon significant changes.

_Generated by Clario360 AI Policy Assistant_`, domain, domain, domain)
	return content, nil
}
