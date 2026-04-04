package compliance

// ComplianceTag maps a PII type to a specific regulation article.
type ComplianceTag struct {
	Framework   string `json:"framework"`
	Article     string `json:"article"`
	Category    string `json:"category"`
	Requirement string `json:"requirement"`
	Impact      string `json:"impact"`
	Severity    string `json:"severity"`
}

// FrameworkTagger tags PII types for a specific regulatory framework.
type FrameworkTagger interface {
	Framework() string
	Tag(piiType string) []ComplianceTag
}

// ComplianceTagger orchestrates compliance tagging across all frameworks.
type ComplianceTagger struct {
	taggers []FrameworkTagger
}

// NewComplianceTagger creates a tagger with all registered frameworks.
func NewComplianceTagger() *ComplianceTagger {
	return &ComplianceTagger{
		taggers: []FrameworkTagger{
			NewGDPRTagger(),
			NewHIPAATagger(),
			NewSOC2Tagger(),
			NewPCIDSSTagger(),
			NewSaudiPDPLTagger(),
		},
	}
}

// TagPIIType returns all compliance tags for a given PII type across all frameworks.
func (t *ComplianceTagger) TagPIIType(piiType string) []ComplianceTag {
	var tags []ComplianceTag
	for _, tagger := range t.taggers {
		tags = append(tags, tagger.Tag(piiType)...)
	}
	// Add cross-cutting tags that apply to all personal data
	tags = append(tags, crossCuttingTags(piiType)...)
	return tags
}

// TagPIITypes returns compliance tags for multiple PII types, deduplicated.
func (t *ComplianceTagger) TagPIITypes(piiTypes []string) []ComplianceTag {
	seen := make(map[string]bool)
	var tags []ComplianceTag
	for _, piiType := range piiTypes {
		for _, tag := range t.TagPIIType(piiType) {
			key := tag.Framework + "|" + tag.Article
			if !seen[key] {
				seen[key] = true
				tags = append(tags, tag)
			}
		}
	}
	return tags
}

// Frameworks returns the names of all registered frameworks.
func (t *ComplianceTagger) Frameworks() []string {
	names := make([]string, len(t.taggers))
	for i, tagger := range t.taggers {
		names[i] = tagger.Framework()
	}
	return names
}

// crossCuttingTags returns tags that apply to ALL personal data across frameworks.
func crossCuttingTags(piiType string) []ComplianceTag {
	if piiType == "" {
		return nil
	}
	return []ComplianceTag{
		{
			Framework:   "gdpr",
			Article:     "Art. 5(1)(c)",
			Category:    "data_minimization",
			Requirement: "Personal data must be adequate, relevant, and limited to what is necessary.",
			Impact:      "Excess data collection may violate minimization principle.",
			Severity:    "medium",
		},
		{
			Framework:   "gdpr",
			Article:     "Art. 5(1)(e)",
			Category:    "storage_limitation",
			Requirement: "Personal data must not be kept longer than necessary for the purpose.",
			Impact:      "Must implement retention policies and data deletion procedures.",
			Severity:    "medium",
		},
		{
			Framework:   "soc2",
			Article:     "CC6.7",
			Category:    "encryption",
			Requirement: "Data must be encrypted in transit and at rest.",
			Impact:      "Unencrypted personal data may result in SOC2 non-conformity.",
			Severity:    "high",
		},
		{
			Framework:   "saudi_pdpl",
			Article:     "Art. 18",
			Category:    "security_safeguards",
			Requirement: "Appropriate technical and organizational measures must protect personal data.",
			Impact:      "Inadequate security measures may violate PDPL requirements.",
			Severity:    "high",
		},
	}
}
