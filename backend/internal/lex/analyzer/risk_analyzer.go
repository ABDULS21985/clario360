package analyzer

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/lex/metrics"
	"github.com/clario360/platform/internal/lex/model"
)

type EntityExtractor interface {
	Extract(text string) ([]model.PartyExtraction, []model.ExtractedDate, []model.ExtractedAmount)
	WarnOnMetadataMismatch(contractPartyA, contractPartyB string, parties []model.PartyExtraction) []string
}

type RiskAnalyzer struct {
	extractor       *ClauseExtractor
	missingDetector *MissingClauseDetector
	entityExtractor EntityExtractor
	compliance      *ComplianceChecker
	recommendations *RecommendationEngine
	metrics         *metrics.Metrics
	now             func() time.Time
}

func NewRiskAnalyzer(
	extractor *ClauseExtractor,
	missingDetector *MissingClauseDetector,
	entityExtractor EntityExtractor,
	compliance *ComplianceChecker,
	recommendations *RecommendationEngine,
	m *metrics.Metrics,
) *RiskAnalyzer {
	return &RiskAnalyzer{
		extractor:       extractor,
		missingDetector: missingDetector,
		entityExtractor: entityExtractor,
		compliance:      compliance,
		recommendations: recommendations,
		metrics:         m,
		now:             time.Now,
	}
}

func (a *RiskAnalyzer) SetNow(now func() time.Time) {
	if now != nil {
		a.now = now
	}
}

func (a *RiskAnalyzer) Analyze(contract *model.Contract, text string) (*model.ContractRiskAnalysis, error) {
	result, err := a.AnalyzeDetailed(contract, text)
	if err != nil {
		return nil, err
	}
	return result.Analysis, nil
}

func (a *RiskAnalyzer) AnalyzeDetailed(contract *model.Contract, text string) (*model.AnalysisResult, error) {
	if contract == nil {
		return nil, fmt.Errorf("contract is required")
	}
	start := a.now()

	clauses, err := a.extractor.ExtractClauses(text)
	if err != nil {
		return nil, fmt.Errorf("extract clauses: %w", err)
	}
	found := make(map[model.ClauseType]bool, len(clauses))
	clauseRiskSum := 0.0
	highRiskCount := 0
	for _, clause := range clauses {
		found[clause.ClauseType] = true
		clauseRiskSum += clause.RiskScore
		if clause.RiskLevel == model.RiskLevelCritical || clause.RiskLevel == model.RiskLevelHigh {
			highRiskCount++
		}
		if a.metrics != nil {
			a.metrics.ClauseExtractionTotal.WithLabelValues(string(clause.ClauseType)).Inc()
			a.metrics.ClauseRiskTotal.WithLabelValues(string(clause.RiskLevel)).Inc()
		}
	}

	missing := a.missingDetector.Detect(contract.Type, found)
	for _, clauseType := range missing {
		if a.metrics != nil {
			a.metrics.MissingClausesTotal.WithLabelValues(string(clauseType)).Inc()
		}
	}

	parties, dates, amounts := a.entityExtractor.Extract(text)
	complianceFlags := a.compliance.Check(contract, clauses, text)
	metadataWarnings := a.entityExtractor.WarnOnMetadataMismatch(contract.PartyAName, contract.PartyBName, parties)

	clauseRiskAvg := 0.0
	if len(clauses) > 0 {
		clauseRiskAvg = clauseRiskSum / float64(len(clauses))
	}
	missingPenalty := float64(len(missing) * 8)
	valueFactor := 0.0
	if contract.TotalValue != nil {
		switch {
		case *contract.TotalValue > 10_000_000:
			valueFactor = 15
		case *contract.TotalValue > 1_000_000:
			valueFactor = 10
		}
	}
	expiryFactor := expiryFactor(contract.ExpiryDate, a.now())
	compliancePenalty := float64(len(complianceFlags) * 5)

	rawScore := clauseRiskAvg + missingPenalty + valueFactor + expiryFactor + compliancePenalty
	if rawScore > 100 {
		rawScore = 100
	}
	riskLevel := model.RiskLevelFromScore(rawScore)

	recommendations := uniqueRecommendations(clauses, missing, complianceFlags, a.recommendations)
	findings := buildFindings(clauses, missing, complianceFlags, metadataWarnings)
	if len(findings) > 5 {
		findings = findings[:5]
	}

	duration := a.now().Sub(start)
	analysis := &model.ContractRiskAnalysis{
		ID:                  uuid.New(),
		TenantID:            contract.TenantID,
		ContractID:          contract.ID,
		ContractVersion:     contract.CurrentVersion,
		OverallRisk:         riskLevel,
		RiskScore:           rawScore,
		ClauseCount:         len(clauses),
		HighRiskClauseCount: highRiskCount,
		MissingClauses:      missing,
		KeyFindings:         findings,
		Recommendations:     recommendations,
		ComplianceFlags:     complianceFlags,
		ExtractedParties:    parties,
		ExtractedDates:      dates,
		ExtractedAmounts:    amounts,
		AnalysisDurationMS:  duration.Milliseconds(),
		AnalyzedBy:          "system",
		AnalyzedAt:          a.now().UTC(),
		CreatedAt:           a.now().UTC(),
	}
	if a.metrics != nil {
		a.metrics.ContractAnalysisDuration.Observe(duration.Seconds())
	}
	return &model.AnalysisResult{Analysis: analysis, Clauses: clauses}, nil
}

func expiryFactor(expiryDate *time.Time, now time.Time) float64 {
	if expiryDate == nil {
		return 0
	}
	days := int(expiryDate.UTC().Sub(time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)).Hours() / 24)
	switch {
	case days <= 7:
		return 20
	case days <= 30:
		return 10
	default:
		return 0
	}
}

func buildFindings(clauses []model.ExtractedClause, missing []model.ClauseType, complianceFlags []model.ComplianceFlag, metadataWarnings []string) []model.RiskFinding {
	findings := make([]model.RiskFinding, 0, len(clauses)+len(missing)+len(complianceFlags)+len(metadataWarnings))
	for _, clause := range clauses {
		if clause.RiskLevel == model.RiskLevelNone {
			continue
		}
		ref := clause.SectionReference
		clauseType := clause.ClauseType
		findings = append(findings, model.RiskFinding{
			Title:           fmt.Sprintf("%s clause flagged", strings.Title(strings.ReplaceAll(string(clause.ClauseType), "_", " "))),
			Description:     clause.AnalysisSummary,
			Severity:        clause.RiskLevel,
			ClauseReference: &ref,
			Recommendation:  strings.Join(clause.Recommendations, " "),
			ClauseType:      &clauseType,
		})
	}
	for _, missingClause := range missing {
		title := fmt.Sprintf("Missing %s clause", strings.ReplaceAll(string(missingClause), "_", " "))
		findings = append(findings, model.RiskFinding{
			Title:          title,
			Description:    "Required standard clause is missing from the contract.",
			Severity:       model.RiskLevelHigh,
			Recommendation: "Add the missing clause before contract approval.",
			ClauseType:     &missingClause,
		})
	}
	for _, flag := range complianceFlags {
		findings = append(findings, model.RiskFinding{
			Title:           flag.Title,
			Description:     flag.Description,
			Severity:        flag.Severity,
			ClauseReference: flag.ClauseReference,
			Recommendation:  flag.Description,
		})
	}
	for _, warning := range metadataWarnings {
		findings = append(findings, model.RiskFinding{
			Title:          "Metadata mismatch",
			Description:    warning,
			Severity:       model.RiskLevelMedium,
			Recommendation: "Confirm contract metadata matches the executed document text.",
		})
	}

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Severity.Weight() != findings[j].Severity.Weight() {
			return findings[i].Severity.Weight() > findings[j].Severity.Weight()
		}
		left := ""
		if findings[i].ClauseType != nil {
			left = string(*findings[i].ClauseType)
		}
		right := ""
		if findings[j].ClauseType != nil {
			right = string(*findings[j].ClauseType)
		}
		return left < right
	})
	return findings
}

func uniqueRecommendations(clauses []model.ExtractedClause, missing []model.ClauseType, complianceFlags []model.ComplianceFlag, engine *RecommendationEngine) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(clauses)+len(missing)+len(complianceFlags))
	appendUnique := func(values ...string) {
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if _, exists := seen[value]; exists {
				continue
			}
			seen[value] = struct{}{}
			out = append(out, value)
		}
	}
	for _, clause := range clauses {
		appendUnique(clause.Recommendations...)
	}
	for _, missingClause := range missing {
		appendUnique(fmt.Sprintf("Insert a standard %s clause before approval.", strings.ReplaceAll(string(missingClause), "_", " ")))
	}
	for _, flag := range complianceFlags {
		switch flag.Code {
		case "pii_without_data_protection":
			appendUnique("Add a data protection clause covering breach notice, deletion, and transfer controls.")
		case "vendor_without_audit_rights":
			appendUnique("Include audit rights clause per vendor management policy.")
		case "foreign_governing_law":
			appendUnique("Negotiate governing law to local jurisdiction.")
		case "high_value_without_insurance":
			appendUnique("Require evidence of insurance coverage for high-value commitments.")
		default:
			appendUnique(flag.Description)
		}
	}
	sort.Strings(out)
	return out
}
