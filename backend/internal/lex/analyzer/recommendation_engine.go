package analyzer

import (
	"strings"

	"github.com/clario360/platform/internal/lex/model"
)

type RecommendationEngine struct {
	orgJurisdiction string
}

func NewRecommendationEngine(orgJurisdiction string) *RecommendationEngine {
	return &RecommendationEngine{orgJurisdiction: strings.TrimSpace(orgJurisdiction)}
}

func (e *RecommendationEngine) Recommend(clauseType model.ClauseType, riskLevel model.RiskLevel, riskKeywords []string, content string) []string {
	lowerKeywords := strings.ToLower(strings.Join(riskKeywords, " "))
	lowerContent := strings.ToLower(content)
	switch clauseType {
	case model.ClauseTypeIndemnification:
		if strings.Contains(lowerKeywords, "unlimited") || strings.Contains(lowerKeywords, "uncapped") || strings.Contains(lowerContent, "unlimited liability") {
			return []string{"Negotiate a cap on indemnification liability."}
		}
		return []string{"Limit indemnification to direct losses and carve out disproportionate obligations."}
	case model.ClauseTypeTermination:
		if strings.Contains(lowerKeywords, "no cure period") || strings.Contains(lowerKeywords, "immediate") {
			return []string{"Request a cure period before termination takes effect."}
		}
		return []string{"Ensure termination rights are mutual and tied to a reasonable notice period."}
	case model.ClauseTypeGoverningLaw:
		if strings.Contains(lowerKeywords, "foreign law") || strings.Contains(lowerKeywords, "vendor's jurisdiction") || strings.Contains(lowerContent, "laws of") {
			return []string{"Negotiate governing law to local jurisdiction."}
		}
		return []string{"Confirm the governing law aligns with approved legal jurisdiction policy."}
	case model.ClauseTypeAuditRights:
		if strings.Contains(lowerKeywords, "no audit right") {
			return []string{"Include audit rights clause per vendor management policy."}
		}
		return []string{"Clarify audit scope, timing, and evidence obligations."}
	case model.ClauseTypeDataProtection:
		return []string{"Add breach notification, deletion, and transfer controls to the data protection clause."}
	case model.ClauseTypeInsurance:
		return []string{"Specify minimum insurance coverage amounts and proof-of-coverage obligations."}
	case model.ClauseTypeAutoRenewal:
		return []string{"Require explicit renewal notice and cap renewal price increases."}
	}

	switch riskLevel {
	case model.RiskLevelCritical:
		return []string{"Escalate this clause for immediate legal renegotiation before execution."}
	case model.RiskLevelHigh:
		return []string{"Request revisions to narrow the clause and add reciprocal protections."}
	case model.RiskLevelMedium:
		return []string{"Document fallback wording and confirm commercial approval before acceptance."}
	case model.RiskLevelLow:
		return []string{"Monitor the clause during negotiation and confirm drafting consistency."}
	default:
		return []string{"Retain the clause wording and verify it remains consistent across versions."}
	}
}
