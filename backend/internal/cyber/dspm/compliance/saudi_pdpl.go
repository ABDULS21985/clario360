package compliance

// SaudiPDPLTagger maps PII types to Saudi Personal Data Protection Law articles.
type SaudiPDPLTagger struct {
	mappings map[string][]ComplianceTag
}

// NewSaudiPDPLTagger creates a Saudi PDPL compliance tagger.
func NewSaudiPDPLTagger() *SaudiPDPLTagger {
	t := &SaudiPDPLTagger{
		mappings: make(map[string][]ComplianceTag),
	}
	t.init()
	return t
}

func (t *SaudiPDPLTagger) Framework() string { return "saudi_pdpl" }

func (t *SaudiPDPLTagger) Tag(piiType string) []ComplianceTag {
	tags, ok := t.mappings[piiType]
	if !ok {
		return nil
	}
	out := make([]ComplianceTag, len(tags))
	copy(out, tags)
	return out
}

func (t *SaudiPDPLTagger) init() {
	personalData := ComplianceTag{
		Framework:   "saudi_pdpl",
		Article:     "Art. 5",
		Category:    "personal_data",
		Requirement: "Personal data may only be collected for a specific, clear, and legitimate purpose.",
		Impact:      "Must have consent or lawful basis. Data subject has right to access and correction.",
		Severity:    "high",
	}

	sensitiveData := ComplianceTag{
		Framework:   "saudi_pdpl",
		Article:     "Art. 11",
		Category:    "sensitive_data",
		Requirement: "Sensitive personal data requires explicit consent and enhanced protection measures.",
		Impact:      "Processing prohibited without explicit consent. Must implement additional safeguards.",
		Severity:    "high",
	}

	// Personal data — Art. 5
	for _, pii := range []string{
		"email", "phone", "name", "address", "credit_card",
		"bank_account", "salary", "ip_address", "bvn",
	} {
		t.mappings[pii] = []ComplianceTag{personalData}
	}

	// Sensitive data — Art. 11
	for _, pii := range []string{
		"dob", "ssn", "national_id", "health", "medical",
		"gender", "ethnicity", "religion", "biometric",
	} {
		t.mappings[pii] = []ComplianceTag{sensitiveData}
	}

	// Credential gets security article
	t.mappings["credential"] = []ComplianceTag{{
		Framework:   "saudi_pdpl",
		Article:     "Art. 18",
		Category:    "security_measures",
		Requirement: "Implement appropriate technical and organizational measures to protect personal data.",
		Impact:      "Credentials must be protected with strong security measures.",
		Severity:    "high",
	}}
}
