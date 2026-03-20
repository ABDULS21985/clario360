package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/lex/model"
)

type ClauseRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewClauseRepository(db *pgxpool.Pool, logger zerolog.Logger) *ClauseRepository {
	return &ClauseRepository{db: db, logger: logger}
}

func (r *ClauseRepository) DB() *pgxpool.Pool {
	return r.db
}

func (r *ClauseRepository) ReplaceForContract(ctx context.Context, q Queryer, tenantID, contractID uuid.UUID, clauses []model.ExtractedClause) error {
	if _, err := q.Exec(ctx, `DELETE FROM contract_clauses WHERE tenant_id = $1 AND contract_id = $2`, tenantID, contractID); err != nil {
		return err
	}
	for _, clause := range clauses {
		normalizeExtractedClause(&clause)
		sectionReference := clause.SectionReference
		pageNumber := clause.PageNumber
		analysisSummary := clause.AnalysisSummary
		if _, err := q.Exec(ctx, `
			INSERT INTO contract_clauses (
				id, tenant_id, contract_id, clause_type, title, content, section_reference, page_number,
				risk_level, risk_score, risk_keywords, analysis_summary, recommendations, compliance_flags,
				review_status, extraction_confidence
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,$8,
				$9,$10,$11,$12,$13,$14,
				$15,$16
			)`,
			uuid.New(), tenantID, contractID, clause.ClauseType, clause.Title, clause.Content, sectionReference, pageNumber,
			clause.RiskLevel, clause.RiskScore, clause.RiskKeywords, analysisSummary, clause.Recommendations, clause.ComplianceFlags,
			model.ClauseReviewPending, clause.ExtractionConfidence,
		); err != nil {
			return err
		}
	}
	return nil
}

func normalizeExtractedClause(clause *model.ExtractedClause) {
	if clause == nil {
		return
	}
	if clause.MatchedTypes == nil {
		clause.MatchedTypes = []model.ClauseType{}
	}
	if clause.RiskKeywords == nil {
		clause.RiskKeywords = []string{}
	}
	if clause.Recommendations == nil {
		clause.Recommendations = []string{}
	}
	if clause.ComplianceFlags == nil {
		clause.ComplianceFlags = []string{}
	}
}

func (r *ClauseRepository) ListByContract(ctx context.Context, tenantID, contractID uuid.UUID) ([]model.Clause, error) {
	query := `
		SELECT row_to_json(t)
		FROM (
			SELECT id, tenant_id, contract_id, clause_type, title, content, section_reference, page_number,
			       risk_level, risk_score::float8 AS risk_score, risk_keywords,
			       analysis_summary, recommendations, compliance_flags,
			       review_status, reviewed_by, reviewed_at, review_notes,
			       extraction_confidence::float8 AS extraction_confidence, created_at, updated_at
			FROM contract_clauses
			WHERE tenant_id = $1 AND contract_id = $2
			ORDER BY risk_score DESC, created_at ASC
		) t`
	return queryListJSON[model.Clause](ctx, r.db, query, tenantID, contractID)
}

func (r *ClauseRepository) Get(ctx context.Context, tenantID, contractID, clauseID uuid.UUID) (*model.Clause, error) {
	query := `
		SELECT row_to_json(t)
		FROM (
			SELECT id, tenant_id, contract_id, clause_type, title, content, section_reference, page_number,
			       risk_level, risk_score::float8 AS risk_score, risk_keywords,
			       analysis_summary, recommendations, compliance_flags,
			       review_status, reviewed_by, reviewed_at, review_notes,
			       extraction_confidence::float8 AS extraction_confidence, created_at, updated_at
			FROM contract_clauses
			WHERE tenant_id = $1 AND contract_id = $2 AND id = $3
		) t`
	return queryRowJSON[model.Clause](ctx, r.db, query, tenantID, contractID, clauseID)
}

func (r *ClauseRepository) UpdateReview(ctx context.Context, q Queryer, tenantID, contractID, clauseID uuid.UUID, status model.ClauseReviewStatus, reviewedBy *uuid.UUID, notes string, reviewedAt time.Time) error {
	ct, err := q.Exec(ctx, `
		UPDATE contract_clauses
		SET review_status = $4,
		    reviewed_by = $5,
		    reviewed_at = $6,
		    review_notes = $7,
		    updated_at = now()
		WHERE tenant_id = $1 AND contract_id = $2 AND id = $3`,
		tenantID, contractID, clauseID, status, reviewedBy, reviewedAt, notes,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ClauseRepository) RiskSummary(ctx context.Context, tenantID, contractID uuid.UUID) ([]model.Clause, error) {
	query := `
		SELECT row_to_json(t)
		FROM (
			SELECT id, tenant_id, contract_id, clause_type, title, content, section_reference, page_number,
			       risk_level, risk_score::float8 AS risk_score, risk_keywords,
			       analysis_summary, recommendations, compliance_flags,
			       review_status, reviewed_by, reviewed_at, review_notes,
			       extraction_confidence::float8 AS extraction_confidence, created_at, updated_at
			FROM contract_clauses
			WHERE tenant_id = $1
			  AND contract_id = $2
			  AND risk_level IN ('critical', 'high')
			ORDER BY risk_score DESC, created_at ASC
		) t`
	return queryListJSON[model.Clause](ctx, r.db, query, tenantID, contractID)
}
