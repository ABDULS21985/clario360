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

	"github.com/clario360/platform/internal/lex/model"
)

type ContractRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewContractRepository(db *pgxpool.Pool, logger zerolog.Logger) *ContractRepository {
	return &ContractRepository{db: db, logger: logger}
}

func (r *ContractRepository) Create(ctx context.Context, q Queryer, contract *model.Contract) error {
	query := `
		INSERT INTO contracts (
			id, tenant_id, title, contract_number, type, description,
			party_a_name, party_a_entity, party_b_name, party_b_entity, party_b_contact,
			total_value, currency, payment_terms,
			effective_date, expiry_date, renewal_date, auto_renew, renewal_notice_days, signed_date,
			status, previous_status, status_changed_at, status_changed_by,
			owner_user_id, owner_name, legal_reviewer_id, legal_reviewer_name,
			risk_score, risk_level, analysis_status, last_analyzed_at,
			document_file_id, document_text, current_version,
			parent_contract_id, workflow_instance_id, department, tags, metadata,
			created_by
		) VALUES (
			$1,$2,$3,$4,$5,$6,
			$7,$8,$9,$10,$11,
			$12,$13,$14,
			$15,$16,$17,$18,$19,$20,
			$21,$22,$23,$24,
			$25,$26,$27,$28,
			$29,$30,$31,$32,
			$33,$34,$35,
			$36,$37,$38,$39,$40,
			$41
		)
		RETURNING created_at, updated_at`
	return q.QueryRow(ctx, query,
		contract.ID, contract.TenantID, contract.Title, contract.ContractNumber, contract.Type, contract.Description,
		contract.PartyAName, contract.PartyAEntity, contract.PartyBName, contract.PartyBEntity, contract.PartyBContact,
		contract.TotalValue, contract.Currency, contract.PaymentTerms,
		datePtr(contract.EffectiveDate), datePtr(contract.ExpiryDate), datePtr(contract.RenewalDate), contract.AutoRenew, contract.RenewalNoticeDays, datePtr(contract.SignedDate),
		contract.Status, contract.PreviousStatus, contract.StatusChangedAt, contract.StatusChangedBy,
		contract.OwnerUserID, contract.OwnerName, contract.LegalReviewerID, contract.LegalReviewerName,
		contract.RiskScore, contract.RiskLevel, contract.AnalysisStatus, contract.LastAnalyzedAt,
		contract.DocumentFileID, contract.DocumentText, contract.CurrentVersion,
		contract.ParentContractID, contract.WorkflowInstanceID, contract.Department, contract.Tags, contract.Metadata,
		contract.CreatedBy,
	).Scan(&contract.CreatedAt, &contract.UpdatedAt)
}

func (r *ContractRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.Contract, error) {
	query := contractJSONSelect(`
			c.tenant_id = $1 AND c.id = $2 AND c.deleted_at IS NULL`)
	return queryRowJSON[model.Contract](ctx, r.db, query, tenantID, id)
}

func (r *ContractRepository) List(ctx context.Context, tenantID uuid.UUID, filters model.ContractListFilters) ([]model.Contract, int, error) {
	args := []any{tenantID}
	arg := 2
	conditions := []string{"c.tenant_id = $1", "c.deleted_at IS NULL"}
	if filters.Search != "" {
		conditions = append(conditions, fmt.Sprintf(`(
			to_tsvector('english', coalesce(c.title,'') || ' ' || coalesce(c.party_b_name,'') || ' ' || coalesce(c.description,'')) @@ plainto_tsquery('english', $%d)
			OR c.title ILIKE '%%' || $%d || '%%'
			OR c.party_b_name ILIKE '%%' || $%d || '%%'
		)`, arg, arg, arg))
		args = append(args, strings.TrimSpace(filters.Search))
		arg++
	}
	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("c.status = $%d", arg))
		args = append(args, *filters.Status)
		arg++
	}
	if len(filters.Statuses) > 0 {
		statuses := make([]string, 0, len(filters.Statuses))
		for _, status := range filters.Statuses {
			statuses = append(statuses, string(status))
		}
		conditions = append(conditions, fmt.Sprintf("c.status = ANY($%d)", arg))
		args = append(args, statuses)
		arg++
	}
	if filters.Type != nil {
		conditions = append(conditions, fmt.Sprintf("c.type = $%d", arg))
		args = append(args, *filters.Type)
		arg++
	}
	if filters.OwnerUserID != nil {
		conditions = append(conditions, fmt.Sprintf("c.owner_user_id = $%d", arg))
		args = append(args, *filters.OwnerUserID)
		arg++
	}
	if filters.RiskLevel != nil {
		conditions = append(conditions, fmt.Sprintf("c.risk_level = $%d", arg))
		args = append(args, *filters.RiskLevel)
		arg++
	}
	if filters.Department != "" {
		conditions = append(conditions, fmt.Sprintf("c.department = $%d", arg))
		args = append(args, filters.Department)
		arg++
	}
	if filters.Tag != "" {
		conditions = append(conditions, fmt.Sprintf("$%d = ANY(c.tags)", arg))
		args = append(args, strings.ToLower(strings.TrimSpace(filters.Tag)))
		arg++
	}
	if filters.ExpiringInDays != nil {
		conditions = append(conditions, fmt.Sprintf("c.expiry_date IS NOT NULL AND c.expiry_date <= CURRENT_DATE + $%d::int", arg))
		args = append(args, *filters.ExpiringInDays)
		arg++
	}
	where := strings.Join(conditions, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM contracts c WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count contracts: %w", err)
	}
	if total == 0 {
		return []model.Contract{}, 0, nil
	}

	page := filters.Page
	if page < 1 {
		page = 1
	}
	perPage := filters.PerPage
	if perPage < 1 {
		perPage = 25
	}
	if perPage > 200 {
		perPage = 200
	}
	limitIdx := arg
	offsetIdx := arg + 1
	args = append(args, perPage, (page-1)*perPage)
	query := contractJSONSelect(where) + fmt.Sprintf(" ORDER BY c.updated_at DESC LIMIT $%d OFFSET $%d", limitIdx, offsetIdx)
	items, err := queryListJSON[model.Contract](ctx, r.db, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list contracts: %w", err)
	}
	return items, total, nil
}

func (r *ContractRepository) Update(ctx context.Context, q Queryer, contract *model.Contract) error {
	query := `
		UPDATE contracts
		SET title = $3,
		    contract_number = $4,
		    type = $5,
		    description = $6,
		    party_a_name = $7,
		    party_a_entity = $8,
		    party_b_name = $9,
		    party_b_entity = $10,
		    party_b_contact = $11,
		    total_value = $12,
		    currency = $13,
		    payment_terms = $14,
		    effective_date = $15,
		    expiry_date = $16,
		    renewal_date = $17,
		    auto_renew = $18,
		    renewal_notice_days = $19,
		    signed_date = $20,
		    owner_user_id = $21,
		    owner_name = $22,
		    legal_reviewer_id = $23,
		    legal_reviewer_name = $24,
		    department = $25,
		    tags = $26,
		    metadata = $27,
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL
		RETURNING updated_at`
	return q.QueryRow(ctx, query,
		contract.TenantID, contract.ID,
		contract.Title, contract.ContractNumber, contract.Type, contract.Description,
		contract.PartyAName, contract.PartyAEntity, contract.PartyBName, contract.PartyBEntity, contract.PartyBContact,
		contract.TotalValue, contract.Currency, contract.PaymentTerms,
		datePtr(contract.EffectiveDate), datePtr(contract.ExpiryDate), datePtr(contract.RenewalDate),
		contract.AutoRenew, contract.RenewalNoticeDays, datePtr(contract.SignedDate),
		contract.OwnerUserID, contract.OwnerName, contract.LegalReviewerID, contract.LegalReviewerName,
		contract.Department, contract.Tags, contract.Metadata,
	).Scan(&contract.UpdatedAt)
}

func (r *ContractRepository) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	ct, err := r.db.Exec(ctx, `UPDATE contracts SET deleted_at = now(), updated_at = now() WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ContractRepository) InsertVersion(ctx context.Context, q Queryer, version *model.ContractVersion) error {
	query := `
		INSERT INTO contract_versions (
			id, tenant_id, contract_id, version, file_id, file_name, file_size_bytes,
			content_hash, extracted_text, change_summary, uploaded_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING uploaded_at`
	return q.QueryRow(ctx, query,
		version.ID, version.TenantID, version.ContractID, version.Version, version.FileID, version.FileName, version.FileSizeBytes,
		version.ContentHash, version.ExtractedText, version.ChangeSummary, version.UploadedBy,
	).Scan(&version.UploadedAt)
}

func (r *ContractRepository) UpdateDocument(ctx context.Context, q Queryer, tenantID, contractID uuid.UUID, fileID uuid.UUID, extractedText string, currentVersion int) error {
	ct, err := q.Exec(ctx, `
		UPDATE contracts
		SET document_file_id = $3,
		    document_text = $4,
		    current_version = $5,
		    analysis_status = 'pending',
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, contractID, fileID, extractedText, currentVersion,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ContractRepository) ListVersions(ctx context.Context, tenantID, contractID uuid.UUID) ([]model.ContractVersion, error) {
	query := `
		SELECT row_to_json(t)
		FROM (
			SELECT id, tenant_id, contract_id, version, file_id, file_name, file_size_bytes,
			       content_hash, extracted_text, change_summary, uploaded_by, uploaded_at
			FROM contract_versions
			WHERE tenant_id = $1 AND contract_id = $2
			ORDER BY version DESC
		) t`
	return queryListJSON[model.ContractVersion](ctx, r.db, query, tenantID, contractID)
}

func (r *ContractRepository) GetLatestVersion(ctx context.Context, tenantID, contractID uuid.UUID) (*model.ContractVersion, error) {
	query := `
		SELECT row_to_json(t)
		FROM (
			SELECT id, tenant_id, contract_id, version, file_id, file_name, file_size_bytes,
			       content_hash, extracted_text, change_summary, uploaded_by, uploaded_at
			FROM contract_versions
			WHERE tenant_id = $1 AND contract_id = $2
			ORDER BY version DESC
			LIMIT 1
		) t`
	return queryRowJSON[model.ContractVersion](ctx, r.db, query, tenantID, contractID)
}

func (r *ContractRepository) InsertAnalysis(ctx context.Context, q Queryer, analysis *model.ContractRiskAnalysis) error {
	normalizeContractAnalysis(analysis)
	query := `
		INSERT INTO contract_analyses (
			id, tenant_id, contract_id, contract_version, overall_risk, risk_score,
			clause_count, high_risk_clause_count, missing_clauses, key_findings, recommendations,
			compliance_flags, extracted_parties, extracted_dates, extracted_amounts,
			analysis_duration_ms, analyzed_by, analyzed_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,
			$7,$8,$9,$10,$11,
			$12,$13,$14,$15,
			$16,$17,$18
		)
		RETURNING created_at`
	return q.QueryRow(ctx, query,
		analysis.ID, analysis.TenantID, analysis.ContractID, analysis.ContractVersion, analysis.OverallRisk, analysis.RiskScore,
		analysis.ClauseCount, analysis.HighRiskClauseCount, analysis.MissingClauses, analysis.KeyFindings, analysis.Recommendations,
		analysis.ComplianceFlags, analysis.ExtractedParties, analysis.ExtractedDates, analysis.ExtractedAmounts,
		analysis.AnalysisDurationMS, analysis.AnalyzedBy, analysis.AnalyzedAt,
	).Scan(&analysis.CreatedAt)
}

func normalizeContractAnalysis(analysis *model.ContractRiskAnalysis) {
	if analysis == nil {
		return
	}
	if analysis.MissingClauses == nil {
		analysis.MissingClauses = []model.ClauseType{}
	}
	if analysis.KeyFindings == nil {
		analysis.KeyFindings = []model.RiskFinding{}
	}
	if analysis.Recommendations == nil {
		analysis.Recommendations = []string{}
	}
	if analysis.ComplianceFlags == nil {
		analysis.ComplianceFlags = []model.ComplianceFlag{}
	}
	if analysis.ExtractedParties == nil {
		analysis.ExtractedParties = []model.PartyExtraction{}
	}
	if analysis.ExtractedDates == nil {
		analysis.ExtractedDates = []model.ExtractedDate{}
	}
	if analysis.ExtractedAmounts == nil {
		analysis.ExtractedAmounts = []model.ExtractedAmount{}
	}
}

func (r *ContractRepository) GetLatestAnalysis(ctx context.Context, tenantID, contractID uuid.UUID) (*model.ContractRiskAnalysis, error) {
	query := `
		SELECT row_to_json(t)
		FROM (
			SELECT id, tenant_id, contract_id, contract_version, overall_risk, risk_score,
			       clause_count, high_risk_clause_count, missing_clauses, key_findings, recommendations,
			       compliance_flags, extracted_parties, extracted_dates, extracted_amounts,
			       analysis_duration_ms, analyzed_by, analyzed_at, created_at
			FROM contract_analyses
			WHERE tenant_id = $1 AND contract_id = $2
			ORDER BY analyzed_at DESC
			LIMIT 1
		) t`
	return queryRowJSON[model.ContractRiskAnalysis](ctx, r.db, query, tenantID, contractID)
}

func (r *ContractRepository) UpdateAnalysisFields(ctx context.Context, q Queryer, tenantID, contractID uuid.UUID, riskScore float64, riskLevel model.RiskLevel, status model.AnalysisStatus, analyzedAt time.Time) error {
	ct, err := q.Exec(ctx, `
		UPDATE contracts
		SET risk_score = $3,
		    risk_level = $4,
		    analysis_status = $5,
		    last_analyzed_at = $6,
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, contractID, riskScore, riskLevel, status, analyzedAt,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ContractRepository) SetAnalysisStatus(ctx context.Context, tenantID, contractID uuid.UUID, status model.AnalysisStatus) error {
	ct, err := r.db.Exec(ctx, `UPDATE contracts SET analysis_status = $3, updated_at = now() WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, contractID, status)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ContractRepository) UpdateStatus(ctx context.Context, q Queryer, tenantID, contractID uuid.UUID, previousStatus *model.ContractStatus, status model.ContractStatus, changedBy *uuid.UUID, changedAt time.Time, signedDate *time.Time) error {
	ct, err := q.Exec(ctx, `
		UPDATE contracts
		SET previous_status = $3,
		    status = $4,
		    status_changed_by = $5,
		    status_changed_at = $6,
		    signed_date = COALESCE($7, signed_date),
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, contractID, previousStatus, status, changedBy, changedAt, datePtr(signedDate),
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ContractRepository) SetWorkflowInstance(ctx context.Context, q Queryer, tenantID, contractID uuid.UUID, workflowInstanceID *uuid.UUID) error {
	ct, err := q.Exec(ctx, `
		UPDATE contracts
		SET workflow_instance_id = $3,
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, contractID, workflowInstanceID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ContractRepository) Search(ctx context.Context, tenantID uuid.UUID, search string, page, perPage int) ([]model.ContractSummary, int, error) {
	filters := model.ContractListFilters{Page: page, PerPage: perPage, Search: search}
	items, total, err := r.List(ctx, tenantID, filters)
	if err != nil {
		return nil, 0, err
	}
	summaries := make([]model.ContractSummary, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, summarizeContract(item))
	}
	return summaries, total, nil
}

func (r *ContractRepository) Stats(ctx context.Context, tenantID uuid.UUID) (*model.ContractStats, error) {
	stats := &model.ContractStats{
		ByStatus:    map[string]int{},
		ByType:      map[string]int{},
		ByRiskLevel: map[string]int{},
	}
	rows, err := r.db.Query(ctx, `SELECT status, COUNT(*) FROM contracts WHERE tenant_id = $1 AND deleted_at IS NULL GROUP BY status`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			return nil, err
		}
		stats.ByStatus[key] = count
	}
	for _, query := range []struct {
		sql string
		dst map[string]int
	}{
		{`SELECT type, COUNT(*) FROM contracts WHERE tenant_id = $1 AND deleted_at IS NULL GROUP BY type`, stats.ByType},
		{`SELECT risk_level, COUNT(*) FROM contracts WHERE tenant_id = $1 AND deleted_at IS NULL GROUP BY risk_level`, stats.ByRiskLevel},
	} {
		rows, err := r.db.Query(ctx, query.sql, tenantID)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var key string
			var count int
			if err := rows.Scan(&key, &count); err != nil {
				rows.Close()
				return nil, err
			}
			query.dst[key] = count
		}
		rows.Close()
	}
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM contracts WHERE tenant_id = $1 AND status = 'active' AND expiry_date IS NOT NULL AND expiry_date <= CURRENT_DATE + 30 AND deleted_at IS NULL`, tenantID).Scan(&stats.Expiring30Days); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM contracts WHERE tenant_id = $1 AND status = 'active' AND expiry_date IS NOT NULL AND expiry_date <= CURRENT_DATE + 7 AND deleted_at IS NULL`, tenantID).Scan(&stats.Expiring7Days); err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *ContractRepository) ListExpiring(ctx context.Context, tenantID uuid.UUID, horizonDays int) ([]model.ExpiringContractSummary, error) {
	query := `
		SELECT row_to_json(t)
		FROM (
			SELECT c.id, c.title, c.type, c.status, c.party_b_name,
			       c.expiry_date::timestamptz AS expiry_date,
			       GREATEST((c.expiry_date - CURRENT_DATE), 0) AS days_until_expiry,
			       c.owner_name, c.legal_reviewer_name
			FROM contracts c
			WHERE c.tenant_id = $1
			  AND c.status = 'active'
			  AND c.expiry_date IS NOT NULL
			  AND c.deleted_at IS NULL
			  AND c.expiry_date <= CURRENT_DATE + $2::int
			ORDER BY c.expiry_date ASC
		) t`
	return queryListJSON[model.ExpiringContractSummary](ctx, r.db, query, tenantID, horizonDays)
}

func (r *ContractRepository) CountByType(ctx context.Context, tenantID uuid.UUID) (map[string]int, error) {
	return r.aggregateCounts(ctx, tenantID, "type")
}

func (r *ContractRepository) CountByStatus(ctx context.Context, tenantID uuid.UUID) (map[string]int, error) {
	return r.aggregateCounts(ctx, tenantID, "status")
}

func (r *ContractRepository) RecentContracts(ctx context.Context, tenantID uuid.UUID, limit int) ([]model.ContractSummary, error) {
	query := contractJSONSelect(`c.tenant_id = $1 AND c.deleted_at IS NULL`) + ` ORDER BY c.created_at DESC LIMIT $2`
	contracts, err := queryListJSON[model.Contract](ctx, r.db, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]model.ContractSummary, 0, len(contracts))
	for _, contract := range contracts {
		out = append(out, summarizeContract(contract))
	}
	return out, nil
}

func (r *ContractRepository) HighRiskContracts(ctx context.Context, tenantID uuid.UUID, limit int) ([]model.ContractRiskSummary, error) {
	query := `
		SELECT row_to_json(t)
		FROM (
			SELECT id, title, type, status, risk_level, COALESCE(risk_score, 0) AS risk_score, party_b_name, expiry_date::timestamptz AS expiry_date
			FROM contracts
			WHERE tenant_id = $1 AND risk_level IN ('critical','high') AND deleted_at IS NULL
			ORDER BY COALESCE(risk_score, 0) DESC, updated_at DESC
			LIMIT $2
		) t`
	return queryListJSON[model.ContractRiskSummary](ctx, r.db, query, tenantID, limit)
}

func (r *ContractRepository) TotalValueBreakdown(ctx context.Context, tenantID uuid.UUID) (model.TotalValueBreakdown, error) {
	breakdown := model.TotalValueBreakdown{
		ByType:     map[string]float64{},
		ByCurrency: map[string]float64{},
	}
	rows, err := r.db.Query(ctx, `SELECT type, COALESCE(SUM(total_value),0)::float8 FROM contracts WHERE tenant_id = $1 AND status = 'active' AND deleted_at IS NULL GROUP BY type`, tenantID)
	if err != nil {
		return breakdown, err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var value float64
		if err := rows.Scan(&key, &value); err != nil {
			return breakdown, err
		}
		breakdown.ByType[key] = value
	}
	rows, err = r.db.Query(ctx, `SELECT currency, COALESCE(SUM(total_value),0)::float8 FROM contracts WHERE tenant_id = $1 AND status = 'active' AND deleted_at IS NULL GROUP BY currency`, tenantID)
	if err != nil {
		return breakdown, err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var value float64
		if err := rows.Scan(&key, &value); err != nil {
			return breakdown, err
		}
		breakdown.ByCurrency[key] = value
	}
	return breakdown, nil
}

func (r *ContractRepository) MonthlyActivity(ctx context.Context, tenantID uuid.UUID) ([]model.MonthlyContractActivity, error) {
	query := `
		SELECT row_to_json(t)
		FROM (
			WITH months AS (
				SELECT generate_series(date_trunc('month', CURRENT_DATE) - interval '11 months', date_trunc('month', CURRENT_DATE), interval '1 month') AS month_start
			)
			SELECT to_char(month_start, 'YYYY-MM') AS month,
			       COALESCE((SELECT COUNT(*) FROM contracts c WHERE c.tenant_id = $1 AND date_trunc('month', c.created_at) = month_start AND c.deleted_at IS NULL), 0) AS created,
			       COALESCE((SELECT COUNT(*) FROM contracts c WHERE c.tenant_id = $1 AND c.status = 'active' AND date_trunc('month', c.status_changed_at) = month_start AND c.deleted_at IS NULL), 0) AS activated,
			       COALESCE((SELECT COUNT(*) FROM contracts c WHERE c.tenant_id = $1 AND c.status = 'expired' AND date_trunc('month', c.status_changed_at) = month_start AND c.deleted_at IS NULL), 0) AS expired,
			       COALESCE((SELECT COUNT(*) FROM contracts c WHERE c.tenant_id = $1 AND c.status = 'renewed' AND date_trunc('month', c.status_changed_at) = month_start AND c.deleted_at IS NULL), 0) AS renewed
			FROM months
			ORDER BY month_start
		) t`
	return queryListJSON[model.MonthlyContractActivity](ctx, r.db, query, tenantID)
}

func (r *ContractRepository) ListDueForExpiryBucket(ctx context.Context, lowerExclusive, upperInclusive int) ([]model.Contract, error) {
	query := contractJSONSelect(`
			c.status = 'active'
			AND c.expiry_date IS NOT NULL
			AND c.deleted_at IS NULL
			AND (c.expiry_date - CURRENT_DATE) <= $1
			AND (c.expiry_date - CURRENT_DATE) > $2`)
	return queryListJSON[model.Contract](ctx, r.db, query, upperInclusive, lowerExclusive)
}

func (r *ContractRepository) ListExpiredActive(ctx context.Context) ([]model.Contract, error) {
	query := contractJSONSelect(`c.status = 'active' AND c.expiry_date IS NOT NULL AND c.expiry_date < CURRENT_DATE AND c.deleted_at IS NULL`)
	return queryListJSON[model.Contract](ctx, r.db, query)
}

func (r *ContractRepository) RecordExpiryNotification(ctx context.Context, q Queryer, tenantID, contractID uuid.UUID, horizon int) (bool, error) {
	var id uuid.UUID
	err := q.QueryRow(ctx, `
		INSERT INTO expiry_notifications (tenant_id, contract_id, horizon_days)
		VALUES ($1, $2, $3)
		ON CONFLICT (contract_id, horizon_days) DO NOTHING
		RETURNING id`,
		tenantID, contractID, horizon,
	).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *ContractRepository) GetByWorkflowInstance(ctx context.Context, workflowInstanceID uuid.UUID) (*model.Contract, error) {
	query := contractJSONSelect(`c.workflow_instance_id = $1 AND c.deleted_at IS NULL`)
	return queryRowJSON[model.Contract](ctx, r.db, query, workflowInstanceID)
}

func (r *ContractRepository) GetByFileID(ctx context.Context, fileID uuid.UUID) ([]model.Contract, error) {
	query := contractJSONSelect(`(c.document_file_id = $1 OR EXISTS (SELECT 1 FROM contract_versions v WHERE v.contract_id = c.id AND v.file_id = $1)) AND c.deleted_at IS NULL`)
	return queryListJSON[model.Contract](ctx, r.db, query, fileID)
}

func (r *ContractRepository) ListTenantIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `SELECT DISTINCT tenant_id FROM contracts WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var tenantID uuid.UUID
		if err := rows.Scan(&tenantID); err != nil {
			return nil, err
		}
		out = append(out, tenantID)
	}
	return out, rows.Err()
}

func (r *ContractRepository) aggregateCounts(ctx context.Context, tenantID uuid.UUID, column string) (map[string]int, error) {
	switch column {
	case "type", "status":
	default:
		return nil, fmt.Errorf("unsupported aggregate column %q", column)
	}
	rows, err := r.db.Query(ctx, fmt.Sprintf(`SELECT %s, COUNT(*) FROM contracts WHERE tenant_id = $1 AND deleted_at IS NULL GROUP BY %s`, column, column), tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			return nil, err
		}
		out[key] = count
	}
	return out, rows.Err()
}

func contractJSONSelect(where string) string {
	return `
		SELECT row_to_json(t)
		FROM (
			SELECT c.id, c.tenant_id, c.title, c.contract_number, c.type, c.description,
			       c.party_a_name, c.party_a_entity, c.party_b_name, c.party_b_entity, c.party_b_contact,
			       c.total_value::float8 AS total_value, c.currency, c.payment_terms,
			       c.effective_date::timestamptz AS effective_date,
			       c.expiry_date::timestamptz AS expiry_date,
			       c.renewal_date::timestamptz AS renewal_date,
			       c.auto_renew, c.renewal_notice_days,
			       c.signed_date::timestamptz AS signed_date,
			       c.status, c.previous_status,
			       c.status_changed_at, c.status_changed_by,
			       c.owner_user_id, c.owner_name, c.legal_reviewer_id, c.legal_reviewer_name,
			       c.risk_score::float8 AS risk_score, COALESCE(c.risk_level, 'none') AS risk_level,
			       COALESCE(c.analysis_status, 'pending') AS analysis_status,
			       c.last_analyzed_at, c.document_file_id, COALESCE(c.document_text, '') AS document_text,
			       c.current_version, c.parent_contract_id, c.workflow_instance_id,
			       c.department, COALESCE(c.tags, '{}') AS tags, COALESCE(c.metadata, '{}'::jsonb) AS metadata,
			       c.created_by, c.created_at, c.updated_at, c.deleted_at
			FROM contracts c
			WHERE ` + where + `
		) t`
}

func summarizeContract(contract model.Contract) model.ContractSummary {
	return model.ContractSummary{
		ID:             contract.ID,
		Title:          contract.Title,
		Type:           contract.Type,
		Status:         contract.Status,
		PartyBName:     contract.PartyBName,
		RiskLevel:      contract.RiskLevel,
		RiskScore:      contract.RiskScore,
		ExpiryDate:     contract.ExpiryDate,
		CurrentVersion: contract.CurrentVersion,
		CreatedAt:      contract.CreatedAt,
	}
}

func datePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	normalized := time.Date(value.UTC().Year(), value.UTC().Month(), value.UTC().Day(), 0, 0, 0, 0, time.UTC)
	return &normalized
}
