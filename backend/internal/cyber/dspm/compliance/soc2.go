package compliance

// SOC2Tagger maps PII types to SOC2 Trust Services Criteria.
type SOC2Tagger struct {
	mappings map[string][]ComplianceTag
}

// NewSOC2Tagger creates a SOC2 compliance tagger.
func NewSOC2Tagger() *SOC2Tagger {
	t := &SOC2Tagger{
		mappings: make(map[string][]ComplianceTag),
	}
	t.init()
	return t
}

func (t *SOC2Tagger) Framework() string { return "soc2" }

func (t *SOC2Tagger) Tag(piiType string) []ComplianceTag {
	tags, ok := t.mappings[piiType]
	if !ok {
		return nil
	}
	out := make([]ComplianceTag, len(tags))
	copy(out, tags)
	return out
}

func (t *SOC2Tagger) init() {
	accessControl := ComplianceTag{
		Framework:   "soc2",
		Article:     "CC6.1",
		Category:    "access_control",
		Requirement: "Logical access security measures to protect information assets.",
		Impact:      "Must implement role-based access controls and authentication mechanisms.",
		Severity:    "high",
	}

	encryption := ComplianceTag{
		Framework:   "soc2",
		Article:     "CC6.7",
		Category:    "encryption",
		Requirement: "Data must be protected during transmission and at rest using encryption.",
		Impact:      "Must encrypt sensitive data in transit and at rest. Key management required.",
		Severity:    "high",
	}

	authControl := ComplianceTag{
		Framework:   "soc2",
		Article:     "CC6.2",
		Category:    "authentication",
		Requirement: "Authentication mechanisms to protect credentials and access.",
		Impact:      "Must implement strong authentication and credential management.",
		Severity:    "high",
	}

	// Access control — CC6.1 (all PII types need access control)
	for _, pii := range []string{
		"email", "phone", "name", "address", "dob", "salary",
		"health", "medical", "gender", "ethnicity", "religion", "ip_address",
	} {
		t.mappings[pii] = []ComplianceTag{accessControl}
	}

	// Access control + Encryption — CC6.1 + CC6.7
	for _, pii := range []string{"ssn", "national_id", "biometric"} {
		t.mappings[pii] = []ComplianceTag{accessControl, encryption}
	}

	// Encryption — CC6.7
	for _, pii := range []string{"credit_card", "bank_account", "bvn"} {
		t.mappings[pii] = []ComplianceTag{encryption}
	}

	// Authentication — CC6.2
	t.mappings["credential"] = []ComplianceTag{authControl}
}
