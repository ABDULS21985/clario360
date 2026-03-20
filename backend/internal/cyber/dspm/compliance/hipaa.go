package compliance

// HIPAATagger maps PII types to HIPAA sections.
type HIPAATagger struct {
	mappings map[string][]ComplianceTag
}

// NewHIPAATagger creates a HIPAA compliance tagger.
func NewHIPAATagger() *HIPAATagger {
	t := &HIPAATagger{
		mappings: make(map[string][]ComplianceTag),
	}
	t.init()
	return t
}

func (t *HIPAATagger) Framework() string { return "hipaa" }

func (t *HIPAATagger) Tag(piiType string) []ComplianceTag {
	tags, ok := t.mappings[piiType]
	if !ok {
		return nil
	}
	out := make([]ComplianceTag, len(tags))
	copy(out, tags)
	return out
}

func (t *HIPAATagger) init() {
	phiIdentifier := ComplianceTag{
		Framework:   "hipaa",
		Article:     "§164.514(b)",
		Category:    "phi_identifier",
		Requirement: "Protected Health Information identifier that must be de-identified per Safe Harbor method.",
		Impact:      "Must remove or encrypt these identifiers for de-identification. Breach notification required if exposed.",
		Severity:    "high",
	}

	clinicalData := ComplianceTag{
		Framework:   "hipaa",
		Article:     "§164.530(c)",
		Category:    "clinical_data",
		Requirement: "Clinical and treatment information subject to minimum necessary standard.",
		Impact:      "Must implement access controls and audit trails. Subject to accounting of disclosures.",
		Severity:    "high",
	}

	// PHI identifiers — §164.514(b) (the 18 HIPAA identifiers)
	for _, pii := range []string{"email", "phone", "name", "address", "dob", "ssn", "national_id", "biometric"} {
		t.mappings[pii] = []ComplianceTag{phiIdentifier}
	}

	// Clinical/health data — §164.530(c)
	t.mappings["health"] = []ComplianceTag{clinicalData}
	t.mappings["medical"] = []ComplianceTag{clinicalData}
}
