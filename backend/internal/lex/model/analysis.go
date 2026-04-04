package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type RiskLevel string

const (
	RiskLevelCritical RiskLevel = "critical"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelLow      RiskLevel = "low"
	RiskLevelNone     RiskLevel = "none"
)

type RiskFinding struct {
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Severity        RiskLevel  `json:"severity"`
	ClauseReference *string    `json:"clause_reference,omitempty"`
	Recommendation  string     `json:"recommendation"`
	ClauseType      *ClauseType `json:"clause_type,omitempty"`
}

type ComplianceFlag struct {
	Code            string     `json:"code"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Severity        RiskLevel  `json:"severity"`
	ClauseReference *string    `json:"clause_reference,omitempty"`
}

type PartyExtraction struct {
	Name   string `json:"name"`
	Role   string `json:"role"`
	Source string `json:"source"`
}

type ExtractedDate struct {
	Label  string     `json:"label"`
	Value  *time.Time `json:"value,omitempty"`
	Source string     `json:"source"`
}

type ExtractedAmount struct {
	Label    string  `json:"label"`
	Currency string  `json:"currency"`
	Value    float64 `json:"value"`
	Source   string  `json:"source"`
}

type ContractRiskAnalysis struct {
	ID                  uuid.UUID         `json:"id"`
	TenantID            uuid.UUID         `json:"tenant_id"`
	ContractID          uuid.UUID         `json:"contract_id"`
	ContractVersion     int               `json:"contract_version"`
	OverallRisk         RiskLevel         `json:"overall_risk"`
	RiskScore           float64           `json:"risk_score"`
	ClauseCount         int               `json:"clause_count"`
	HighRiskClauseCount int               `json:"high_risk_clause_count"`
	MissingClauses      []ClauseType      `json:"missing_clauses"`
	KeyFindings         []RiskFinding     `json:"key_findings"`
	Recommendations     []string          `json:"recommendations"`
	ComplianceFlags     []ComplianceFlag  `json:"compliance_flags"`
	ExtractedParties    []PartyExtraction `json:"extracted_parties"`
	ExtractedDates      []ExtractedDate   `json:"extracted_dates"`
	ExtractedAmounts    []ExtractedAmount `json:"extracted_amounts"`
	AnalysisDurationMS  int64             `json:"analysis_duration_ms"`
	AnalyzedBy          string            `json:"analyzed_by"`
	AnalyzedAt          time.Time         `json:"analyzed_at"`
	CreatedAt           time.Time         `json:"created_at"`
}

type AnalysisResult struct {
	Analysis *ContractRiskAnalysis `json:"analysis"`
	Clauses  []ExtractedClause     `json:"clauses"`
}

func (r RiskLevel) Score() float64 {
	switch r {
	case RiskLevelCritical:
		return 90
	case RiskLevelHigh:
		return 70
	case RiskLevelMedium:
		return 45
	case RiskLevelLow:
		return 20
	default:
		return 0
	}
}

func (r RiskLevel) Weight() int {
	switch r {
	case RiskLevelCritical:
		return 4
	case RiskLevelHigh:
		return 3
	case RiskLevelMedium:
		return 2
	case RiskLevelLow:
		return 1
	default:
		return 0
	}
}

func ParseRiskLevel(raw string) RiskLevel {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(RiskLevelCritical):
		return RiskLevelCritical
	case string(RiskLevelHigh):
		return RiskLevelHigh
	case string(RiskLevelMedium):
		return RiskLevelMedium
	case string(RiskLevelLow):
		return RiskLevelLow
	default:
		return RiskLevelNone
	}
}

func RiskLevelFromScore(score float64) RiskLevel {
	switch {
	case score > 75:
		return RiskLevelCritical
	case score > 55:
		return RiskLevelHigh
	case score > 35:
		return RiskLevelMedium
	case score > 15:
		return RiskLevelLow
	default:
		return RiskLevelNone
	}
}
