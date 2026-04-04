package analyzer

import (
	"strings"

	"github.com/clario360/platform/internal/lex/model"
)

type ComplianceChecker struct {
	orgJurisdiction string
}

func NewComplianceChecker(orgJurisdiction string) *ComplianceChecker {
	return &ComplianceChecker{orgJurisdiction: strings.ToLower(strings.TrimSpace(orgJurisdiction))}
}

func (c *ComplianceChecker) Check(contract *model.Contract, clauses []model.ExtractedClause, text string) []model.ComplianceFlag {
	found := make(map[model.ClauseType]model.ExtractedClause, len(clauses))
	for _, clause := range clauses {
		if _, exists := found[clause.ClauseType]; !exists {
			found[clause.ClauseType] = clause
		}
	}

	var flags []model.ComplianceFlag
	lowerText := strings.ToLower(text)
	if containsAny(lowerText, []string{"personal data", "personally identifiable", "pii", "data subject"}) {
		if _, ok := found[model.ClauseTypeDataProtection]; !ok {
			flags = append(flags, model.ComplianceFlag{
				Code:        "pii_without_data_protection",
				Title:       "PII handling without data protection clause",
				Description: "The contract appears to cover personal data but does not contain a data protection clause.",
				Severity:    model.RiskLevelHigh,
			})
		}
	}
	if contract.Type == model.ContractTypeVendor {
		if _, ok := found[model.ClauseTypeAuditRights]; !ok {
			flags = append(flags, model.ComplianceFlag{
				Code:        "vendor_without_audit_rights",
				Title:       "Vendor contract missing audit rights",
				Description: "Vendor contracts must include audit rights under vendor management policy.",
				Severity:    model.RiskLevelHigh,
			})
		}
	}
	if contract.TotalValue != nil && *contract.TotalValue > 1_000_000 {
		if _, ok := found[model.ClauseTypeInsurance]; !ok {
			flags = append(flags, model.ComplianceFlag{
				Code:        "high_value_without_insurance",
				Title:       "High-value contract missing insurance clause",
				Description: "Contracts above 1,000,000 require insurance requirements.",
				Severity:    model.RiskLevelHigh,
			})
		}
	}
	if clause, ok := found[model.ClauseTypeGoverningLaw]; ok && c.orgJurisdiction != "" {
		content := strings.ToLower(clause.Content)
		if containsAny(content, []string{"new york", "england", "delaware", "california", "vendor's jurisdiction", "foreign law"}) && !strings.Contains(content, c.orgJurisdiction) {
			ref := clause.SectionReference
			flags = append(flags, model.ComplianceFlag{
				Code:            "foreign_governing_law",
				Title:           "Governing law differs from approved jurisdiction",
				Description:     "The governing law clause appears to select a foreign jurisdiction.",
				Severity:        model.RiskLevelMedium,
				ClauseReference: &ref,
			})
		}
	}
	if contract.AutoRenew && contract.RenewalNoticeDays < 30 {
		flags = append(flags, model.ComplianceFlag{
			Code:        "renewal_notice_too_short",
			Title:       "Renewal notice window is below policy minimum",
			Description: "Auto-renew contracts must provide at least 30 days notice before renewal.",
			Severity:    model.RiskLevelMedium,
		})
	}
	return flags
}

func containsAny(value string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
