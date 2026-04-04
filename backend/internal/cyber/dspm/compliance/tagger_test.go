package compliance

import (
	"testing"
)

func TestNewComplianceTagger_HasAllFrameworks(t *testing.T) {
	ct := NewComplianceTagger()
	frameworks := ct.Frameworks()

	expected := []string{"gdpr", "hipaa", "soc2", "pci_dss", "saudi_pdpl"}
	if len(frameworks) != len(expected) {
		t.Fatalf("expected %d frameworks, got %d: %v", len(expected), len(frameworks), frameworks)
	}
	for i, name := range expected {
		if frameworks[i] != name {
			t.Errorf("framework[%d] = %q, want %q", i, frameworks[i], name)
		}
	}
}

// --- GDPR ---

func TestGDPR_Email_PersonalData(t *testing.T) {
	tagger := NewGDPRTagger()
	tags := tagger.Tag("email")
	assertHasArticle(t, tags, "gdpr", "Art. 4(1)")
}

func TestGDPR_DOB_SpecialCategory(t *testing.T) {
	tagger := NewGDPRTagger()
	tags := tagger.Tag("dob")
	assertHasArticle(t, tags, "gdpr", "Art. 9")
}

func TestGDPR_SSN_NationalID(t *testing.T) {
	tagger := NewGDPRTagger()
	tags := tagger.Tag("ssn")
	assertHasArticle(t, tags, "gdpr", "Art. 87")
}

func TestGDPR_Credential_Security(t *testing.T) {
	tagger := NewGDPRTagger()
	tags := tagger.Tag("credential")
	assertHasArticle(t, tags, "gdpr", "Art. 32")
}

func TestGDPR_Biometric_SpecialCategory(t *testing.T) {
	tagger := NewGDPRTagger()
	tags := tagger.Tag("biometric")
	assertHasArticle(t, tags, "gdpr", "Art. 9")
}

func TestGDPR_UnknownPII_ReturnsNil(t *testing.T) {
	tagger := NewGDPRTagger()
	tags := tagger.Tag("unknown_pii_type")
	if len(tags) != 0 {
		t.Errorf("expected no tags for unknown PII type, got %d", len(tags))
	}
}

// --- HIPAA ---

func TestHIPAA_Email_PHIIdentifier(t *testing.T) {
	tagger := NewHIPAATagger()
	tags := tagger.Tag("email")
	assertHasArticle(t, tags, "hipaa", "§164.514(b)")
}

func TestHIPAA_Health_ClinicalData(t *testing.T) {
	tagger := NewHIPAATagger()
	tags := tagger.Tag("health")
	assertHasArticle(t, tags, "hipaa", "§164.530(c)")
}

func TestHIPAA_Medical_ClinicalData(t *testing.T) {
	tagger := NewHIPAATagger()
	tags := tagger.Tag("medical")
	assertHasArticle(t, tags, "hipaa", "§164.530(c)")
}

func TestHIPAA_SSN_PHIIdentifier(t *testing.T) {
	tagger := NewHIPAATagger()
	tags := tagger.Tag("ssn")
	assertHasArticle(t, tags, "hipaa", "§164.514(b)")
}

func TestHIPAA_Biometric_PHIIdentifier(t *testing.T) {
	tagger := NewHIPAATagger()
	tags := tagger.Tag("biometric")
	assertHasArticle(t, tags, "hipaa", "§164.514(b)")
}

// --- SOC2 ---

func TestSOC2_CreditCard_Encryption(t *testing.T) {
	tagger := NewSOC2Tagger()
	tags := tagger.Tag("credit_card")
	if len(tags) == 0 {
		t.Fatal("expected SOC2 tags for credit_card")
	}
	assertHasArticle(t, tags, "soc2", "CC6.7")
}

func TestSOC2_Credential_Authentication(t *testing.T) {
	tagger := NewSOC2Tagger()
	tags := tagger.Tag("credential")
	if len(tags) == 0 {
		t.Fatal("expected SOC2 tags for credential")
	}
	assertHasArticle(t, tags, "soc2", "CC6.2")
}

// --- PCI DSS ---

func TestPCIDSS_CreditCard_Masking(t *testing.T) {
	tagger := NewPCIDSSTagger()
	tags := tagger.Tag("credit_card")
	assertHasArticle(t, tags, "pci_dss", "Req 3.4")
}

func TestPCIDSS_Credential_Auth(t *testing.T) {
	tagger := NewPCIDSSTagger()
	tags := tagger.Tag("credential")
	assertHasArticle(t, tags, "pci_dss", "Req 8.2")
}

// --- Saudi PDPL ---

func TestSaudiPDPL_Email_PersonalData(t *testing.T) {
	tagger := NewSaudiPDPLTagger()
	tags := tagger.Tag("email")
	assertHasArticle(t, tags, "saudi_pdpl", "Art. 5")
}

func TestSaudiPDPL_Health_SensitiveData(t *testing.T) {
	tagger := NewSaudiPDPLTagger()
	tags := tagger.Tag("health")
	if len(tags) == 0 {
		t.Fatal("expected Saudi PDPL tags for health")
	}
	assertHasArticle(t, tags, "saudi_pdpl", "Art. 11")
}

// --- Cross-Cutting ---

func TestCrossCuttingTags_AllPersonalData(t *testing.T) {
	tags := crossCuttingTags("email")
	if len(tags) != 4 {
		t.Fatalf("expected 4 cross-cutting tags, got %d", len(tags))
	}

	// GDPR Art. 5(1)(c) — data minimization
	assertHasArticle(t, tags, "gdpr", "Art. 5(1)(c)")
	// GDPR Art. 5(1)(e) — storage limitation
	assertHasArticle(t, tags, "gdpr", "Art. 5(1)(e)")
	// SOC2 CC6.7 — encryption
	assertHasArticle(t, tags, "soc2", "CC6.7")
	// Saudi PDPL Art. 18 — security safeguards
	assertHasArticle(t, tags, "saudi_pdpl", "Art. 18")
}

func TestCrossCuttingTags_EmptyPII_ReturnsNil(t *testing.T) {
	tags := crossCuttingTags("")
	if tags != nil {
		t.Errorf("expected nil for empty PII type, got %d tags", len(tags))
	}
}

// --- Integration: All PII types have GDPR + SOC2 tags ---

func TestAllPIITypes_HaveGDPRTag(t *testing.T) {
	ct := NewComplianceTagger()
	piiTypes := []string{
		"email", "phone", "name", "address", "credit_card", "bank_account",
		"salary", "ip_address", "bvn", "dob", "health", "medical",
		"gender", "ethnicity", "religion", "biometric", "ssn", "national_id",
		"credential",
	}

	for _, pii := range piiTypes {
		tags := ct.TagPIIType(pii)
		foundGDPR := false
		for _, tag := range tags {
			if tag.Framework == "gdpr" {
				foundGDPR = true
				break
			}
		}
		if !foundGDPR {
			t.Errorf("PII type %q has no GDPR tag", pii)
		}
	}
}

func TestAllPIITypes_HaveSOC2Tag(t *testing.T) {
	ct := NewComplianceTagger()
	piiTypes := []string{
		"email", "phone", "name", "address", "credit_card", "bank_account",
		"salary", "ip_address", "bvn", "dob", "health", "medical",
		"gender", "ethnicity", "religion", "biometric", "ssn", "national_id",
		"credential",
	}

	for _, pii := range piiTypes {
		tags := ct.TagPIIType(pii)
		foundSOC2 := false
		for _, tag := range tags {
			if tag.Framework == "soc2" {
				foundSOC2 = true
				break
			}
		}
		if !foundSOC2 {
			t.Errorf("PII type %q has no SOC2 tag (at minimum CC6.7 from cross-cutting)", pii)
		}
	}
}

// --- TagPIITypes deduplication ---

func TestTagPIITypes_Deduplicates(t *testing.T) {
	ct := NewComplianceTagger()
	tags := ct.TagPIITypes([]string{"email", "phone"})

	seen := make(map[string]int)
	for _, tag := range tags {
		key := tag.Framework + "|" + tag.Article
		seen[key]++
		if seen[key] > 1 {
			t.Errorf("duplicate tag: %s %s", tag.Framework, tag.Article)
		}
	}
}

func TestTagPIITypes_Empty_ReturnsNil(t *testing.T) {
	ct := NewComplianceTagger()
	tags := ct.TagPIITypes(nil)
	if len(tags) != 0 {
		t.Errorf("expected no tags for nil input, got %d", len(tags))
	}
}

// --- Helpers ---

func assertHasArticle(t *testing.T, tags []ComplianceTag, framework, article string) {
	t.Helper()
	for _, tag := range tags {
		if tag.Framework == framework && tag.Article == article {
			return
		}
	}
	t.Errorf("expected tag %s %s not found in %d tags", framework, article, len(tags))
}
