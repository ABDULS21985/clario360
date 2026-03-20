package model

import (
	"time"

	"github.com/google/uuid"
)

type ClauseType string

const (
	ClauseTypeIndemnification       ClauseType = "indemnification"
	ClauseTypeTermination           ClauseType = "termination"
	ClauseTypeLimitationOfLiability ClauseType = "limitation_of_liability"
	ClauseTypeConfidentiality       ClauseType = "confidentiality"
	ClauseTypeIPOwnership           ClauseType = "ip_ownership"
	ClauseTypeNonCompete            ClauseType = "non_compete"
	ClauseTypePaymentTerms          ClauseType = "payment_terms"
	ClauseTypeWarranty              ClauseType = "warranty"
	ClauseTypeForceMajeure          ClauseType = "force_majeure"
	ClauseTypeDisputeResolution     ClauseType = "dispute_resolution"
	ClauseTypeDataProtection        ClauseType = "data_protection"
	ClauseTypeGoverningLaw          ClauseType = "governing_law"
	ClauseTypeAssignment            ClauseType = "assignment"
	ClauseTypeInsurance             ClauseType = "insurance"
	ClauseTypeAuditRights           ClauseType = "audit_rights"
	ClauseTypeSLA                   ClauseType = "sla"
	ClauseTypeAutoRenewal           ClauseType = "auto_renewal"
	ClauseTypeRepresentations       ClauseType = "representations"
	ClauseTypeNonSolicitation       ClauseType = "non_solicitation"
	ClauseTypeOther                 ClauseType = "other"
)

type ClauseReviewStatus string

const (
	ClauseReviewPending  ClauseReviewStatus = "pending"
	ClauseReviewReviewed ClauseReviewStatus = "reviewed"
	ClauseReviewFlagged  ClauseReviewStatus = "flagged"
	ClauseReviewAccepted ClauseReviewStatus = "accepted"
	ClauseReviewRejected ClauseReviewStatus = "rejected"
)

type Clause struct {
	ID                   uuid.UUID          `json:"id"`
	TenantID             uuid.UUID          `json:"tenant_id"`
	ContractID           uuid.UUID          `json:"contract_id"`
	ClauseType           ClauseType         `json:"clause_type"`
	Title                string             `json:"title"`
	Content              string             `json:"content"`
	SectionReference     *string            `json:"section_reference,omitempty"`
	PageNumber           *int               `json:"page_number,omitempty"`
	RiskLevel            RiskLevel          `json:"risk_level"`
	RiskScore            float64            `json:"risk_score"`
	RiskKeywords         []string           `json:"risk_keywords"`
	AnalysisSummary      *string            `json:"analysis_summary,omitempty"`
	Recommendations      []string           `json:"recommendations"`
	ComplianceFlags      []string           `json:"compliance_flags"`
	ReviewStatus         ClauseReviewStatus `json:"review_status"`
	ReviewedBy           *uuid.UUID         `json:"reviewed_by,omitempty"`
	ReviewedAt           *time.Time         `json:"reviewed_at,omitempty"`
	ReviewNotes          *string            `json:"review_notes,omitempty"`
	ExtractionConfidence float64            `json:"extraction_confidence"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
}

type ExtractedClause struct {
	ClauseType           ClauseType    `json:"clause_type"`
	PrimaryType          ClauseType    `json:"primary_type"`
	MatchedTypes         []ClauseType  `json:"matched_types"`
	Title                string        `json:"title"`
	Content              string        `json:"content"`
	SectionReference     string        `json:"section_reference"`
	PageNumber           int           `json:"page_number"`
	RiskLevel            RiskLevel     `json:"risk_level"`
	RiskScore            float64       `json:"risk_score"`
	RiskKeywords         []string      `json:"risk_keywords"`
	AnalysisSummary      string        `json:"analysis_summary"`
	Recommendations      []string      `json:"recommendations"`
	ComplianceFlags      []string      `json:"compliance_flags"`
	ExtractionConfidence float64       `json:"extraction_confidence"`
	PatternHits          int           `json:"pattern_hits"`
	FirstMatchOffset     int           `json:"first_match_offset"`
}

func AllClauseTypes() []ClauseType {
	return []ClauseType{
		ClauseTypeIndemnification,
		ClauseTypeTermination,
		ClauseTypeLimitationOfLiability,
		ClauseTypeConfidentiality,
		ClauseTypeIPOwnership,
		ClauseTypeNonCompete,
		ClauseTypePaymentTerms,
		ClauseTypeWarranty,
		ClauseTypeForceMajeure,
		ClauseTypeDisputeResolution,
		ClauseTypeDataProtection,
		ClauseTypeGoverningLaw,
		ClauseTypeAssignment,
		ClauseTypeInsurance,
		ClauseTypeAuditRights,
		ClauseTypeSLA,
		ClauseTypeAutoRenewal,
		ClauseTypeRepresentations,
		ClauseTypeNonSolicitation,
	}
}
