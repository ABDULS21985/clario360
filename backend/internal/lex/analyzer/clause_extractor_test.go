package analyzer

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/clario360/platform/internal/lex/model"
)

func TestExtract_Indemnification(t *testing.T) {
	extractor := NewClauseExtractor(NewRecommendationEngine("Saudi Arabia"))
	clauses, err := extractor.ExtractClauses("Section 1 Indemnification\nThe supplier shall indemnify and hold harmless Clario from third-party claims.")
	if err != nil {
		t.Fatalf("ExtractClauses() error = %v", err)
	}
	if len(clauses) != 1 {
		t.Fatalf("len(clauses) = %d, want 1", len(clauses))
	}
	if clauses[0].ClauseType != model.ClauseTypeIndemnification {
		t.Fatalf("ClauseType = %s, want %s", clauses[0].ClauseType, model.ClauseTypeIndemnification)
	}
}

func TestExtract_Termination(t *testing.T) {
	extractor := NewClauseExtractor(NewRecommendationEngine("Saudi Arabia"))
	text := "Section 1 Termination\nEither party may terminate this agreement with 30 days notice for material breach."
	clauses, err := extractor.ExtractClauses(text)
	if err != nil {
		t.Fatalf("ExtractClauses() error = %v", err)
	}
	if len(clauses) != 1 || clauses[0].ClauseType != model.ClauseTypeTermination {
		t.Fatalf("unexpected clauses: %+v", clauses)
	}
}

func TestExtract_LimitationOfLiability(t *testing.T) {
	extractor := NewClauseExtractor(NewRecommendationEngine("Saudi Arabia"))
	text := "Section 1 Liability\nThe aggregate liability of either party shall not exceed the fees paid under this agreement."
	clauses, err := extractor.ExtractClauses(text)
	if err != nil {
		t.Fatalf("ExtractClauses() error = %v", err)
	}
	found := false
	for _, clause := range clauses {
		if clause.ClauseType == model.ClauseTypeLimitationOfLiability {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected limitation_of_liability in extracted clauses: %+v", clauses)
	}
}

func TestExtract_MultipleTypes(t *testing.T) {
	extractor := NewClauseExtractor(NewRecommendationEngine("Saudi Arabia"))
	text := "Section 2 Remedies\nEither party may terminate this agreement for breach and may exercise a right to terminate after notice. The supplier shall indemnify and hold harmless Clario from losses."
	clauses, err := extractor.ExtractClauses(text)
	if err != nil {
		t.Fatalf("ExtractClauses() error = %v", err)
	}
	if len(clauses) != 2 {
		t.Fatalf("len(clauses) = %d, want 2", len(clauses))
	}
	for _, clause := range clauses {
		if clause.PrimaryType != model.ClauseTypeTermination {
			t.Fatalf("PrimaryType = %s, want %s", clause.PrimaryType, model.ClauseTypeTermination)
		}
	}
}

func TestExtract_RiskKeywords_Unlimited(t *testing.T) {
	extractor := NewClauseExtractor(NewRecommendationEngine("Saudi Arabia"))
	text := "Section 3 Liability\nThe limitation of liability clause states there is unlimited liability and no cap for any claim."
	clauses, err := extractor.ExtractClauses(text)
	if err != nil {
		t.Fatalf("ExtractClauses() error = %v", err)
	}
	if len(clauses) != 1 {
		t.Fatalf("len(clauses) = %d, want 1", len(clauses))
	}
	if clauses[0].RiskLevel != model.RiskLevelCritical {
		t.Fatalf("RiskLevel = %s, want %s", clauses[0].RiskLevel, model.RiskLevelCritical)
	}
}

func TestExtract_RiskKeywords_None(t *testing.T) {
	extractor := NewClauseExtractor(NewRecommendationEngine("Saudi Arabia"))
	text := "Section 4 Confidentiality\nConfidential information may be used only to perform the agreement and must be returned on request."
	clauses, err := extractor.ExtractClauses(text)
	if err != nil {
		t.Fatalf("ExtractClauses() error = %v", err)
	}
	if len(clauses) != 1 {
		t.Fatalf("len(clauses) = %d, want 1", len(clauses))
	}
	if clauses[0].RiskLevel != model.RiskLevelNone {
		t.Fatalf("RiskLevel = %s, want %s", clauses[0].RiskLevel, model.RiskLevelNone)
	}
}

func TestExtract_SectionSplitting(t *testing.T) {
	extractor := NewClauseExtractor(NewRecommendationEngine("Saudi Arabia"))
	text := strings.Join([]string{
		"Section 1 Termination",
		"Either party may terminate for material breach after notice.",
		"",
		"Section 2 Confidentiality",
		"Confidential information must be protected and returned on request.",
	}, "\n")
	clauses, err := extractor.ExtractClauses(text)
	if err != nil {
		t.Fatalf("ExtractClauses() error = %v", err)
	}
	if len(clauses) != 2 {
		t.Fatalf("len(clauses) = %d, want 2", len(clauses))
	}
	if clauses[0].SectionReference != "Section 1" || clauses[1].SectionReference != "Section 2" {
		t.Fatalf("unexpected section references: %+v", clauses)
	}
}

func TestExtract_ArticleSplitting(t *testing.T) {
	extractor := NewClauseExtractor(NewRecommendationEngine("Saudi Arabia"))
	text := strings.Join([]string{
		"Article I",
		"The governing law clause states that the agreement is governed by the laws of Saudi Arabia.",
		"",
		"Article II",
		"Any dispute shall be resolved by arbitration in Riyadh.",
	}, "\n")
	clauses, err := extractor.ExtractClauses(text)
	if err != nil {
		t.Fatalf("ExtractClauses() error = %v", err)
	}
	if len(clauses) != 2 {
		t.Fatalf("len(clauses) = %d, want 2", len(clauses))
	}
	if clauses[0].SectionReference != "Article I" || clauses[1].SectionReference != "Article II" {
		t.Fatalf("unexpected section references: %+v", clauses)
	}
}

func TestExtract_Confidence_HighMultipleMatches(t *testing.T) {
	extractor := NewClauseExtractor(NewRecommendationEngine("Saudi Arabia"))
	text := "Section 5 Termination\nThe right to terminate applies if either party terminates or exercises cancellation rights."
	clauses, err := extractor.ExtractClauses(text)
	if err != nil {
		t.Fatalf("ExtractClauses() error = %v", err)
	}
	if len(clauses) != 1 {
		t.Fatalf("len(clauses) = %d, want 1", len(clauses))
	}
	if clauses[0].ExtractionConfidence != 0.95 {
		t.Fatalf("ExtractionConfidence = %.2f, want 0.95", clauses[0].ExtractionConfidence)
	}
}

func TestExtract_Confidence_LowWeakMatch(t *testing.T) {
	extractor := NewClauseExtractor(NewRecommendationEngine("Saudi Arabia"))
	clauses, err := extractor.ExtractClauses("Payment")
	if err != nil {
		t.Fatalf("ExtractClauses() error = %v", err)
	}
	if len(clauses) != 1 {
		t.Fatalf("len(clauses) = %d, want 1", len(clauses))
	}
	if clauses[0].ExtractionConfidence != 0.50 {
		t.Fatalf("ExtractionConfidence = %.2f, want 0.50", clauses[0].ExtractionConfidence)
	}
}

func TestExtract_AllClauseTypes(t *testing.T) {
	extractor := NewClauseExtractor(NewRecommendationEngine("Saudi Arabia"))
	var sections []string
	for idx, clauseType := range model.AllClauseTypes() {
		sections = append(sections, allTypeSection(idx+1, clauseType))
	}
	clauses, err := extractor.ExtractClauses(strings.Join(sections, "\n\n"))
	if err != nil {
		t.Fatalf("ExtractClauses() error = %v", err)
	}
	found := map[model.ClauseType]bool{}
	for _, clause := range clauses {
		found[clause.ClauseType] = true
	}
	for _, clauseType := range model.AllClauseTypes() {
		if !found[clauseType] {
			t.Fatalf("clause type %s was not extracted", clauseType)
		}
	}
}

func TestExtract_EmptyText(t *testing.T) {
	extractor := NewClauseExtractor(NewRecommendationEngine("Saudi Arabia"))
	clauses, err := extractor.ExtractClauses("")
	if err != nil {
		t.Fatalf("ExtractClauses() error = %v", err)
	}
	if len(clauses) != 0 {
		t.Fatalf("len(clauses) = %d, want 0", len(clauses))
	}
}

func TestExtract_Deterministic(t *testing.T) {
	extractor := NewClauseExtractor(NewRecommendationEngine("Saudi Arabia"))
	text := "Section 1 Liability\nThe limitation of liability clause states there is no cap and no limitation for any claim."
	first, err := extractor.ExtractClauses(text)
	if err != nil {
		t.Fatalf("first ExtractClauses() error = %v", err)
	}
	second, err := extractor.ExtractClauses(text)
	if err != nil {
		t.Fatalf("second ExtractClauses() error = %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("extraction was not deterministic:\nfirst=%+v\nsecond=%+v", first, second)
	}
}

func allTypeSection(index int, clauseType model.ClauseType) string {
	switch clauseType {
	case model.ClauseTypeIndemnification:
		return section(index, "Indemnification", "The supplier shall indemnify and hold harmless Clario for direct claims.")
	case model.ClauseTypeTermination:
		return section(index, "Termination", "Either party may terminate for material breach after 30 days notice.")
	case model.ClauseTypeLimitationOfLiability:
		return section(index, "Limitation of Liability", "The aggregate liability shall not exceed fees paid.")
	case model.ClauseTypeConfidentiality:
		return section(index, "Confidentiality", "Confidential information must remain confidential.")
	case model.ClauseTypeIPOwnership:
		return section(index, "IP Ownership", "Intellectual property and work product will belong to Clario.")
	case model.ClauseTypeNonCompete:
		return section(index, "Non-Compete", "The non-compete restriction applies only to competing services.")
	case model.ClauseTypePaymentTerms:
		return section(index, "Payment", "Payment terms require invoicing and compensation within 30 days.")
	case model.ClauseTypeWarranty:
		return section(index, "Warranty", "Each party warrants professional services and no as-is delivery.")
	case model.ClauseTypeForceMajeure:
		return section(index, "Force Majeure", "Force majeure events beyond control suspend obligations.")
	case model.ClauseTypeDisputeResolution:
		return section(index, "Dispute Resolution", "Any dispute shall be resolved by arbitration.")
	case model.ClauseTypeDataProtection:
		return section(index, "Data Protection", "Personal data and privacy protections apply to the services.")
	case model.ClauseTypeGoverningLaw:
		return section(index, "Governing Law", "The agreement is governed by the laws of Saudi Arabia.")
	case model.ClauseTypeAssignment:
		return section(index, "Assignment", "Neither party may assign or transfer the agreement without consent.")
	case model.ClauseTypeInsurance:
		return section(index, "Insurance", "The supplier must maintain cyber insurance.")
	case model.ClauseTypeAuditRights:
		return section(index, "Audit Rights", "Clario may audit and inspect records annually.")
	case model.ClauseTypeSLA:
		return section(index, "Service Levels", "The SLA includes uptime, availability, and response time commitments.")
	case model.ClauseTypeAutoRenewal:
		return section(index, "Auto Renewal", "The agreement includes automatic renewal unless notice is given.")
	case model.ClauseTypeRepresentations:
		return section(index, "Representations", "Each party makes representations and undertakings regarding authority.")
	case model.ClauseTypeNonSolicitation:
		return section(index, "Non-Solicitation", "The parties will not solicit the other party's employees.")
	default:
		return section(index, "Other", "General clause.")
	}
}

func section(index int, title string, body string) string {
	return "Section " + strconv.Itoa(index) + " " + title + "\n" + body
}
