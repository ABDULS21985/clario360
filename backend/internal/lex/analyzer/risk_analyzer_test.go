package analyzer

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/lex/model"
)

func TestRisk_HighRiskClauses(t *testing.T) {
	analyzer := newTestRiskAnalyzer()
	contract := testContract(model.ContractTypeOther, 500000, 60)
	text := `
Section 1 Liability
The limitation of liability clause states there is unlimited liability, no cap, no limitation, excluding consequential losses, and excluding indirect losses.

Section 2 Termination
Either party may terminate without cause, with immediate effect, with no notice, on a unilateral basis, and with no cure period.

Section 3 Indemnification
The supplier shall indemnify Clario on an uncapped, first dollar, sole expense basis for all claims regardless of fault.`
	analysis, err := analyzer.Analyze(contract, text)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if analysis.OverallRisk != model.RiskLevelHigh {
		t.Fatalf("OverallRisk = %s, want %s", analysis.OverallRisk, model.RiskLevelHigh)
	}
}

func TestRisk_MissingClauses(t *testing.T) {
	analyzer := newTestRiskAnalyzer()
	contract := testContract(model.ContractTypeServiceAgreement, 250000, 90)
	text := `Section 1 Termination
Either party may terminate for material breach after notice.

Section 2 Confidentiality
Confidential information must remain confidential.`
	analysis, err := analyzer.Analyze(contract, text)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if len(analysis.MissingClauses) < 3 {
		t.Fatalf("len(MissingClauses) = %d, want at least 3", len(analysis.MissingClauses))
	}
	if analysis.RiskScore <= 24 {
		t.Fatalf("RiskScore = %.2f, want missing clause penalty applied", analysis.RiskScore)
	}
}

func TestRisk_HighValue(t *testing.T) {
	analyzer := newTestRiskAnalyzer()
	contract := testContract(model.ContractTypeOther, 1500000, 120)
	analysis, err := analyzer.Analyze(contract, "")
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if analysis.RiskScore < 10 {
		t.Fatalf("RiskScore = %.2f, want high-value factor to add at least 10", analysis.RiskScore)
	}
}

func TestRisk_ExpiringContract(t *testing.T) {
	analyzer := newTestRiskAnalyzer()
	contract := testContract(model.ContractTypeOther, 100000, 7)
	analysis, err := analyzer.Analyze(contract, "")
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if analysis.RiskScore < 20 {
		t.Fatalf("RiskScore = %.2f, want expiry factor to add 20", analysis.RiskScore)
	}
}

func TestRisk_ComplianceFlags(t *testing.T) {
	analyzer := newTestRiskAnalyzer()
	contract := testContract(model.ContractTypeServiceAgreement, 500000, 30)
	text := `
This agreement covers personally identifiable information and customer account identifiers.

Section 1 Termination
Either party may terminate for material breach after notice.`
	analysis, err := analyzer.Analyze(contract, text)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	found := false
	for _, flag := range analysis.ComplianceFlags {
		if flag.Code == "pii_without_data_protection" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected pii_without_data_protection compliance flag")
	}
}

func TestRisk_ScoreClamped(t *testing.T) {
	analyzer := newTestRiskAnalyzer()
	contract := testContract(model.ContractTypeServiceAgreement, 20000000, 1)
	text := `
This agreement covers personal data.

Section 1 Liability
The limitation of liability clause states there is unlimited liability, no cap, no limitation, excluding consequential losses, and excluding indirect losses.

Section 2 Termination
Either party may terminate without cause, with immediate effect, with no notice, on a unilateral basis, and with no cure period.

Section 3 Indemnification
The supplier shall indemnify Clario on an uncapped, first dollar, sole expense basis for all claims regardless of fault.`
	analysis, err := analyzer.Analyze(contract, text)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if analysis.RiskScore != 100 {
		t.Fatalf("RiskScore = %.2f, want 100", analysis.RiskScore)
	}
}

func TestRisk_LevelMapping(t *testing.T) {
	tests := []struct {
		score float64
		want  model.RiskLevel
	}{
		{80, model.RiskLevelCritical},
		{60, model.RiskLevelHigh},
		{50, model.RiskLevelMedium},
		{30, model.RiskLevelLow},
	}
	for _, tc := range tests {
		if got := model.RiskLevelFromScore(tc.score); got != tc.want {
			t.Fatalf("RiskLevelFromScore(%.2f) = %s, want %s", tc.score, got, tc.want)
		}
	}
}

func TestRisk_Deterministic(t *testing.T) {
	analyzer := newTestRiskAnalyzer()
	contract := testContract(model.ContractTypeVendor, 1200000, 25)
	text := `
This agreement covers personal data.

Section 1 Governing Law
The governing law clause states the agreement is governed by foreign law and the vendor's jurisdiction.

Section 2 Payment
Payment terms require invoicing under net 90 with no penalty for late payment.`
	first, err := analyzer.Analyze(contract, text)
	if err != nil {
		t.Fatalf("first Analyze() error = %v", err)
	}
	second, err := analyzer.Analyze(contract, text)
	if err != nil {
		t.Fatalf("second Analyze() error = %v", err)
	}
	if first.RiskScore != second.RiskScore ||
		first.OverallRisk != second.OverallRisk ||
		!reflect.DeepEqual(first.MissingClauses, second.MissingClauses) ||
		!reflect.DeepEqual(first.Recommendations, second.Recommendations) ||
		!reflect.DeepEqual(first.ComplianceFlags, second.ComplianceFlags) {
		t.Fatalf("analysis was not deterministic:\nfirst=%+v\nsecond=%+v", first, second)
	}
}

func newTestRiskAnalyzer() *RiskAnalyzer {
	recommendations := NewRecommendationEngine("Saudi Arabia")
	riskAnalyzer := NewRiskAnalyzer(
		NewClauseExtractor(recommendations),
		NewMissingClauseDetector(),
		NewEntityExtractor(),
		NewComplianceChecker("Saudi Arabia"),
		recommendations,
		nil,
	)
	riskAnalyzer.SetNow(func() time.Time {
		return time.Date(2026, time.March, 7, 9, 0, 0, 0, time.UTC)
	})
	return riskAnalyzer
}

func testContract(contractType model.ContractType, value float64, expiryInDays int) *model.Contract {
	effectiveDate := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	expiryDate := time.Date(2026, time.March, 7, 0, 0, 0, 0, time.UTC).AddDate(0, 0, expiryInDays)
	return &model.Contract{
		ID:                uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		TenantID:          uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		Title:             "Test Contract",
		Type:              contractType,
		PartyAName:        "Clario Holdings Limited",
		PartyBName:        "Counterparty Ltd.",
		TotalValue:        &value,
		Currency:          "SAR",
		EffectiveDate:     &effectiveDate,
		ExpiryDate:        &expiryDate,
		RenewalNoticeDays: 30,
		Status:            model.ContractStatusActive,
		OwnerUserID:       uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"),
		OwnerName:         "Owner User",
		RiskLevel:         model.RiskLevelNone,
		CurrentVersion:    1,
	}
}
