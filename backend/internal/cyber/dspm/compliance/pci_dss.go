package compliance

// PCIDSSTagger maps PII types to PCI DSS requirements.
type PCIDSSTagger struct {
	mappings map[string][]ComplianceTag
}

// NewPCIDSSTagger creates a PCI DSS compliance tagger.
func NewPCIDSSTagger() *PCIDSSTagger {
	t := &PCIDSSTagger{
		mappings: make(map[string][]ComplianceTag),
	}
	t.init()
	return t
}

func (t *PCIDSSTagger) Framework() string { return "pci_dss" }

func (t *PCIDSSTagger) Tag(piiType string) []ComplianceTag {
	tags, ok := t.mappings[piiType]
	if !ok {
		return nil
	}
	out := make([]ComplianceTag, len(tags))
	copy(out, tags)
	return out
}

func (t *PCIDSSTagger) init() {
	maskRequirement := ComplianceTag{
		Framework:   "pci_dss",
		Article:     "Req 3.4",
		Category:    "cardholder_data",
		Requirement: "Render PAN unreadable anywhere it is stored using encryption, hashing, truncation, or tokenization.",
		Impact:      "Card numbers must be masked or encrypted at rest. Full PAN must never be displayed.",
		Severity:    "high",
	}

	authRequirement := ComplianceTag{
		Framework:   "pci_dss",
		Article:     "Req 8.2",
		Category:    "authentication",
		Requirement: "Authenticate users with unique IDs and strong authentication mechanisms.",
		Impact:      "Must implement MFA, strong passwords, and credential rotation.",
		Severity:    "high",
	}

	// Cardholder data — Req 3.4
	t.mappings["credit_card"] = []ComplianceTag{maskRequirement}

	// Authentication — Req 8.2
	t.mappings["credential"] = []ComplianceTag{authRequirement}
}
