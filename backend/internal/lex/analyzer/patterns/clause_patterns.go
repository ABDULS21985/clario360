package patterns

import (
	"regexp"

	"github.com/clario360/platform/internal/lex/model"
)

type ClausePattern struct {
	Type         model.ClauseType
	Title        string
	Regexps      []*regexp.Regexp
	RiskKeywords []string
}

func DefaultClausePatterns() []ClausePattern {
	return []ClausePattern{
		mustClausePattern(model.ClauseTypeIndemnification, "Indemnification", []string{
			`(?i)(indemnif|hold\s+harmless|defend\s+and\s+indemnify|save\s+harmless)`,
		}, []string{"unlimited", "uncapped", "sole expense", "first dollar", "all claims", "regardless of fault", "broadly defined losses"}),
		mustClausePattern(model.ClauseTypeTermination, "Termination", []string{
			`(?i)(terminat|right\s+to\s+cancel|cancellation\s+rights|right\s+to\s+terminate)`,
		}, []string{"without cause", "immediate", "no notice", "at will", "unilateral", "no cure period", "automatic termination"}),
		mustClausePattern(model.ClauseTypeLimitationOfLiability, "Limitation of Liability", []string{
			`(?i)(limitation\s+of\s+liability|liability\s+cap|maximum\s+liability|aggregate\s+liability|limit.*liab)`,
		}, []string{"unlimited", "no cap", "no limitation", "excluding consequential", "excluding indirect", "waiver of liability"}),
		mustClausePattern(model.ClauseTypeConfidentiality, "Confidentiality", []string{
			`(?i)(confidential|non-disclosure|nda|proprietary\s+information|trade\s+secret)`,
		}, []string{"perpetual", "no exceptions", "residual knowledge", "unrestricted use"}),
		mustClausePattern(model.ClauseTypeIPOwnership, "IP Ownership", []string{
			`(?i)(intellectual\s+property|ip\s+ownership|work\s+product|proprietary\s+rights|copyright\s+assign|work\s+for\s+hire)`,
		}, []string{"vendor retains", "shared ownership", "license back", "non-exclusive", "pre-existing IP", "joint ownership"}),
		mustClausePattern(model.ClauseTypeNonCompete, "Non-Compete", []string{
			`(?i)(non-compet|restrictive\s+covenant|exclusivity|non-solicitation)`,
		}, []string{"worldwide", "perpetual", "all industries", "no geographic limit"}),
		mustClausePattern(model.ClauseTypePaymentTerms, "Payment Terms", []string{
			`(?i)(payment|compensation|fee|invoice|billing|remuneration)`,
		}, []string{"net 90", "net 120", "upon completion only", "milestone-based only", "no penalty for late payment"}),
		mustClausePattern(model.ClauseTypeWarranty, "Warranty", []string{
			`(?i)(warrant|guarantee|representation\s+and\s+warrant|as-is)`,
		}, []string{"as-is", "no warranty", "disclaims all", "implied warranties excluded"}),
		mustClausePattern(model.ClauseTypeForceMajeure, "Force Majeure", []string{
			`(?i)(force\s+majeure|act\s+of\s+god|extraordinary\s+event|beyond.*control)`,
		}, []string{"pandemic excluded", "economic downturn excluded", "no termination right"}),
		mustClausePattern(model.ClauseTypeDisputeResolution, "Dispute Resolution", []string{
			`(?i)(dispute|arbitrat|mediat|jurisdiction|forum\s+selection|choice\s+of\s+court)`,
		}, []string{"foreign jurisdiction", "binding arbitration only", "waive right to trial", "vendor's jurisdiction"}),
		mustClausePattern(model.ClauseTypeDataProtection, "Data Protection", []string{
			`(?i)(data\s+protect|privacy|gdpr|personal\s+data|data\s+process|data\s+breach)`,
		}, []string{"no breach notification", "unlimited processing", "no data deletion", "cross-border transfer unrestricted"}),
		mustClausePattern(model.ClauseTypeGoverningLaw, "Governing Law", []string{
			`(?i)(governing\s+law|applicable\s+law|governed\s+by|laws\s+of)`,
		}, []string{"foreign law", "vendor's jurisdiction"}),
		mustClausePattern(model.ClauseTypeAssignment, "Assignment", []string{
			`(?i)(assign|transfer|novation|delegation|subcontract)`,
		}, []string{"freely assignable", "no consent required", "to any affiliate"}),
		mustClausePattern(model.ClauseTypeInsurance, "Insurance", []string{
			`(?i)(insurance|indemnity\s+insurance|professional\s+liability|cyber\s+insurance)`,
		}, []string{"no insurance requirement", "minimum not specified"}),
		mustClausePattern(model.ClauseTypeAuditRights, "Audit Rights", []string{
			`(?i)(audit|inspect|access\s+to\s+records|right\s+to\s+examine)`,
		}, []string{"no audit right", "only with consent", "limited frequency"}),
		mustClausePattern(model.ClauseTypeSLA, "Service Levels", []string{
			`(?i)(service\s+level|sla|uptime|availability|response\s+time|support\s+level)`,
		}, []string{"best effort", "no penalty", "no credit", "commercially reasonable"}),
		mustClausePattern(model.ClauseTypeAutoRenewal, "Auto Renewal", []string{
			`(?i)(auto.?renew|automatic\s+renewal|evergreen|rollover)`,
		}, []string{"without notice", "annual renewal", "opt-out only", "price increase on renewal"}),
		mustClausePattern(model.ClauseTypeRepresentations, "Representations", []string{
			`(?i)(represent|warrant|covenant|undertaking|declaration)`,
		}, []string{"unilateral representations", "perpetual", "survive termination indefinitely"}),
		mustClausePattern(model.ClauseTypeNonSolicitation, "Non-Solicitation", []string{
			`(?i)(non.?solicit|no.?poach|hiring\s+restriction|employee\s+solicitation)`,
		}, []string{"perpetual", "worldwide", "all employees", "includes independent contractors"}),
	}
}

func mustClausePattern(clauseType model.ClauseType, title string, regexps, riskKeywords []string) ClausePattern {
	out := ClausePattern{
		Type:         clauseType,
		Title:        title,
		Regexps:      make([]*regexp.Regexp, 0, len(regexps)),
		RiskKeywords: riskKeywords,
	}
	for _, raw := range regexps {
		out.Regexps = append(out.Regexps, regexp.MustCompile(raw))
	}
	return out
}
