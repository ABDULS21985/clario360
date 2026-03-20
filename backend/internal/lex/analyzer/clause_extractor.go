package analyzer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/clario360/platform/internal/lex/analyzer/patterns"
	"github.com/clario360/platform/internal/lex/model"
)

type ClauseExtractor struct {
	patterns []patterns.ClausePattern
	splitter *patterns.SectionSplitter
	recs     *RecommendationEngine
}

func NewClauseExtractor(recommendations *RecommendationEngine) *ClauseExtractor {
	return &ClauseExtractor{
		patterns: patterns.DefaultClausePatterns(),
		splitter: patterns.NewSectionSplitter(),
		recs:     recommendations,
	}
}

func (e *ClauseExtractor) ExtractClauses(text string) ([]model.ExtractedClause, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	sections := e.splitter.Split(text)
	if len(sections) == 0 {
		return nil, nil
	}

	var clauses []model.ExtractedClause
	for _, section := range sections {
		sectionMatches := e.matchSection(section.Text)
		if len(sectionMatches) == 0 {
			continue
		}
		sort.SliceStable(sectionMatches, func(i, j int) bool {
			if sectionMatches[i].PatternHits != sectionMatches[j].PatternHits {
				return sectionMatches[i].PatternHits > sectionMatches[j].PatternHits
			}
			if len(sectionMatches[i].RiskKeywords) != len(sectionMatches[j].RiskKeywords) {
				return len(sectionMatches[i].RiskKeywords) > len(sectionMatches[j].RiskKeywords)
			}
			if sectionMatches[i].FirstMatchOffset != sectionMatches[j].FirstMatchOffset {
				return sectionMatches[i].FirstMatchOffset < sectionMatches[j].FirstMatchOffset
			}
			return sectionMatches[i].ClauseType < sectionMatches[j].ClauseType
		})

		primary := sectionMatches[0].ClauseType
		matchedTypes := make([]model.ClauseType, 0, len(sectionMatches))
		for _, match := range sectionMatches {
			matchedTypes = append(matchedTypes, match.ClauseType)
		}
		for _, match := range sectionMatches {
			match.PrimaryType = primary
			match.MatchedTypes = append([]model.ClauseType(nil), matchedTypes...)
			match.SectionReference = section.Reference
			match.PageNumber = section.PageNumber
			if match.Title == "" {
				match.Title = fmt.Sprintf("%s - %s", section.Reference, humanClauseType(match.ClauseType))
			}
			match.AnalysisSummary = buildClauseAnalysisSummary(match, section.Reference)
			match.Recommendations = e.recs.Recommend(match.ClauseType, match.RiskLevel, match.RiskKeywords, match.Content)
			match.ComplianceFlags = extractClauseComplianceFlags(match)
			clauses = append(clauses, match)
		}
	}
	return clauses, nil
}

func (e *ClauseExtractor) matchSection(text string) []model.ExtractedClause {
	lower := strings.ToLower(text)
	matches := make([]model.ExtractedClause, 0, len(e.patterns))
	for _, clausePattern := range e.patterns {
		patternHits := 0
		firstOffset := -1
		for _, re := range clausePattern.Regexps {
			locs := re.FindAllStringIndex(text, -1)
			patternHits += len(locs)
			if len(locs) > 0 && (firstOffset == -1 || locs[0][0] < firstOffset) {
				firstOffset = locs[0][0]
			}
		}
		if patternHits == 0 {
			continue
		}
		riskKeywords := collectRiskKeywords(lower, clausePattern.RiskKeywords)
		riskLevel := riskLevelFromKeywords(clausePattern.Type, riskKeywords)
		matches = append(matches, model.ExtractedClause{
			ClauseType:           clausePattern.Type,
			Title:                clausePattern.Title,
			Content:              strings.TrimSpace(text),
			RiskLevel:            riskLevel,
			RiskScore:            riskLevel.Score(),
			RiskKeywords:         riskKeywords,
			ExtractionConfidence: extractionConfidence(patternHits, firstOffset, riskKeywords, text),
			PatternHits:          patternHits,
			FirstMatchOffset:     max(firstOffset, 0),
		})
	}
	return matches
}

func collectRiskKeywords(text string, keywords []string) []string {
	out := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			out = append(out, keyword)
		}
	}
	return out
}

func riskLevelFromKeywords(clauseType model.ClauseType, keywords []string) model.RiskLevel {
	joined := strings.ToLower(strings.Join(keywords, " "))
	if clauseType == model.ClauseTypeLimitationOfLiability && strings.Contains(joined, "unlimited") {
		return model.RiskLevelCritical
	}
	if clauseType == model.ClauseTypeAuditRights && strings.Contains(joined, "no audit right") {
		return model.RiskLevelHigh
	}
	switch count := len(keywords); {
	case count >= 5:
		return model.RiskLevelHigh
	case count >= 3:
		return model.RiskLevelMedium
	case count >= 1:
		return model.RiskLevelLow
	default:
		return model.RiskLevelNone
	}
}

func extractionConfidence(patternHits, firstOffset int, keywords []string, content string) float64 {
	if patternHits > 1 {
		return 0.95
	}
	if patternHits == 1 && len(keywords) > 0 {
		return 0.85
	}
	if patternHits == 1 {
		firstWord := strings.Fields(strings.TrimSpace(content))
		if len(firstWord) == 1 || (firstOffset >= 0 && len(firstWord) > 0 && len(firstWord[0]) <= 8) {
			return 0.50
		}
		return 0.70
	}
	return 0.50
}

func buildClauseAnalysisSummary(clause model.ExtractedClause, reference string) string {
	if clause.RiskLevel == model.RiskLevelNone {
		return fmt.Sprintf("%s references %s and does not contain flagged risk keywords.", reference, humanClauseType(clause.ClauseType))
	}
	return fmt.Sprintf(
		"%s references %s and includes %d flagged keyword(s): %s.",
		reference,
		humanClauseType(clause.ClauseType),
		len(clause.RiskKeywords),
		strings.Join(clause.RiskKeywords, ", "),
	)
}

func extractClauseComplianceFlags(clause model.ExtractedClause) []string {
	flags := []string{}
	content := strings.ToLower(clause.Content)
	if clause.ClauseType == model.ClauseTypeDataProtection && strings.Contains(content, "cross-border transfer unrestricted") {
		flags = append(flags, "cross_border_transfer_unrestricted")
	}
	if clause.ClauseType == model.ClauseTypeGoverningLaw && containsAny(content, []string{"foreign law", "vendor's jurisdiction"}) {
		flags = append(flags, "foreign_governing_law")
	}
	if clause.ClauseType == model.ClauseTypeAutoRenewal && containsAny(content, []string{"without notice", "price increase on renewal"}) {
		flags = append(flags, "auto_renewal_notice")
	}
	return flags
}

func humanClauseType(clauseType model.ClauseType) string {
	return strings.ReplaceAll(string(clauseType), "_", " ")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
