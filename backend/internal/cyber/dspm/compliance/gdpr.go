package compliance

// GDPRTagger maps PII types to GDPR articles.
type GDPRTagger struct {
	mappings map[string][]ComplianceTag
}

// NewGDPRTagger creates a GDPR compliance tagger.
func NewGDPRTagger() *GDPRTagger {
	t := &GDPRTagger{
		mappings: make(map[string][]ComplianceTag),
	}
	t.init()
	return t
}

func (t *GDPRTagger) Framework() string { return "gdpr" }

func (t *GDPRTagger) Tag(piiType string) []ComplianceTag {
	tags, ok := t.mappings[piiType]
	if !ok {
		return nil
	}
	out := make([]ComplianceTag, len(tags))
	copy(out, tags)
	return out
}

func (t *GDPRTagger) init() {
	personalData := ComplianceTag{
		Framework:   "gdpr",
		Article:     "Art. 4(1)",
		Category:    "personal_data",
		Requirement: "Any information relating to an identified or identifiable natural person.",
		Impact:      "Must have lawful basis for processing. Subject to data subject rights.",
		Severity:    "high",
	}

	specialCategory := ComplianceTag{
		Framework:   "gdpr",
		Article:     "Art. 9",
		Category:    "special_category",
		Requirement: "Processing of special categories of personal data is prohibited unless an exception applies.",
		Impact:      "Requires explicit consent or specific legal basis. Enhanced safeguards required.",
		Severity:    "high",
	}

	nationalID := ComplianceTag{
		Framework:   "gdpr",
		Article:     "Art. 87",
		Category:    "national_id",
		Requirement: "Processing of national identification numbers subject to specific conditions.",
		Impact:      "Member states may determine specific conditions. Enhanced protection required.",
		Severity:    "high",
	}

	securityTag := ComplianceTag{
		Framework:   "gdpr",
		Article:     "Art. 32",
		Category:    "security_measures",
		Requirement: "Appropriate technical and organizational measures to ensure security of processing.",
		Impact:      "Credentials must be protected with state-of-the-art security measures.",
		Severity:    "high",
	}

	// Personal data — Art. 4(1)
	for _, pii := range []string{"email", "phone", "name", "address", "credit_card", "bank_account", "salary", "ip_address", "bvn"} {
		t.mappings[pii] = []ComplianceTag{personalData}
	}

	// Special category — Art. 9
	for _, pii := range []string{"dob", "health", "medical", "gender", "ethnicity", "religion", "biometric"} {
		t.mappings[pii] = []ComplianceTag{specialCategory}
	}

	// National ID — Art. 87
	t.mappings["ssn"] = []ComplianceTag{nationalID}
	t.mappings["national_id"] = []ComplianceTag{nationalID}

	// Security — Art. 32
	t.mappings["credential"] = []ComplianceTag{securityTag}
}
