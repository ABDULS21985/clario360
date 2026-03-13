package engine

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	llmmodel "github.com/clario360/platform/internal/cyber/vciso/llm/model"
)

var (
	numberPattern        = regexp.MustCompile(`\b\d+(?:\.\d+)?%?\b`)
	statusPattern        = regexp.MustCompile(`(?i)\b(failing|healthy|critical|resolved|open|closed|passed|degraded|high|medium|low|inactive|active)\b`)
	identifierPattern    = regexp.MustCompile(`\b[a-zA-Z][a-zA-Z0-9._:/-]{2,}\b`)
	sentenceSplitPattern = regexp.MustCompile(`(?m)([^.!?\n]+[.!?]?)`)
	multiSpacePattern    = regexp.MustCompile(`\s+`)
	punctuationStripper  = regexp.MustCompile(`[^\pL\pN\s%._:/-]`)
	stopwordSet          = map[string]struct{}{
		"the": {}, "a": {}, "an": {}, "and": {}, "or": {}, "but": {}, "if": {}, "then": {},
		"is": {}, "are": {}, "was": {}, "were": {}, "be": {}, "been": {}, "being": {},
		"to": {}, "of": {}, "in": {}, "on": {}, "for": {}, "by": {}, "with": {}, "as": {},
		"this": {}, "that": {}, "these": {}, "those": {}, "it": {}, "its": {},
		"we": {}, "you": {}, "they": {}, "he": {}, "she": {}, "them": {}, "their": {},
		"from": {}, "at": {}, "into": {}, "about": {}, "over": {}, "under": {},
	}
)

type HallucinationGuard struct {
	minTokenOverlapRatio float64
	minEvidenceScore     float64
}

func NewHallucinationGuard() *HallucinationGuard {
	return &HallucinationGuard{
		minTokenOverlapRatio: 0.45,
		minEvidenceScore:     0.55,
	}
}

func (g *HallucinationGuard) Check(response string, toolResults []*llmmodel.ToolCallResult) *llmmodel.GroundingResult {
	response = strings.TrimSpace(response)
	if response == "" {
		return &llmmodel.GroundingResult{
			Status:         "passed",
			TotalClaims:    0,
			GroundedClaims: 0,
		}
	}

	claims := extractClaims(response)
	if len(claims) == 0 {
		return &llmmodel.GroundingResult{
			Status:         "passed",
			TotalClaims:    0,
			GroundedClaims: 0,
		}
	}

	evidenceSet := buildEvidenceSet(toolResults)

	result := &llmmodel.GroundingResult{
		Status:         "passed",
		TotalClaims:    len(claims),
		GroundedClaims: 0,
	}

	correctedSentences := make([]string, 0, len(claims))
	hasCriticalUngrounded := false

	for _, claim := range claims {
		if claimIsSafeRecommendation(claim) {
			result.GroundedClaims++
			correctedSentences = append(correctedSentences, claim.Raw)
			continue
		}

		match := g.findBestEvidence(claim, evidenceSet)
		if match.Grounded {
			result.GroundedClaims++
			correctedSentences = append(correctedSentences, claim.Raw)
			continue
		}

		critical := isCriticalClaim(claim)

		result.UngroundedClaims = append(result.UngroundedClaims, llmmodel.UngroundedClaim{
			Claim:      claim.Raw,
			Type:       string(claim.Type),
			Critical:   critical,
			Suggestion: suggestedRepair(claim),
		})

		if critical {
			hasCriticalUngrounded = true
		}

		correctedSentences = append(correctedSentences, softenClaim(claim))
	}

	if len(result.UngroundedClaims) == 0 {
		return result
	}

	if hasCriticalUngrounded {
		result.Status = "blocked"
		result.CorrectedResponse = buildBlockedFallback(toolResults)
		return result
	}

	result.Status = "corrected"
	result.CorrectedResponse = strings.TrimSpace(strings.Join(correctedSentences, " "))
	return result
}

type claimType string

const (
	claimTypeNumeric        claimType = "numeric"
	claimTypeStatus         claimType = "status"
	claimTypeSecurity       claimType = "security"
	claimTypeRecommendation claimType = "recommendation"
	claimTypeGeneral        claimType = "general"
)

type extractedClaim struct {
	Raw               string
	Normalized        string
	Type              claimType
	Tokens            []string
	Numbers           []string
	Statuses          []string
	Identifiers       []string
	HasRecommendation bool
}

type evidenceDocument struct {
	Raw        string
	Normalized string
	Tokens     []string
	Numbers    []string
	Statuses   []string
}

type evidenceMatch struct {
	Grounded bool
	Score    float64
	Evidence string
}

func extractClaims(response string) []extractedClaim {
	matches := sentenceSplitPattern.FindAllString(response, -1)
	out := make([]extractedClaim, 0, len(matches))

	for _, m := range matches {
		sentence := strings.TrimSpace(m)
		if sentence == "" {
			continue
		}
		if !looksLikeClaim(sentence) {
			continue
		}

		normalized := normalizeText(sentence)
		tokens := meaningfulTokens(normalized)
		numbers := numberPattern.FindAllString(strings.ToLower(sentence), -1)
		statuses := lowerAll(statusPattern.FindAllString(sentence, -1))
		identifiers := extractIdentifiers(sentence)
		hasRec := containsRecommendationLanguage(sentence)

		ct := classifyClaim(sentence, numbers, statuses, hasRec)

		out = append(out, extractedClaim{
			Raw:               sentence,
			Normalized:        normalized,
			Type:              ct,
			Tokens:            tokens,
			Numbers:           dedupeStrings(numbers),
			Statuses:          dedupeStrings(statuses),
			Identifiers:       dedupeStrings(identifiers),
			HasRecommendation: hasRec,
		})
	}

	return out
}

func buildEvidenceSet(results []*llmmodel.ToolCallResult) []evidenceDocument {
	out := make([]evidenceDocument, 0, len(results)*3)

	for _, result := range results {
		if result == nil {
			continue
		}

		appendEvidence := func(s string) {
			s = strings.TrimSpace(s)
			if s == "" {
				return
			}
			norm := normalizeText(s)
			out = append(out, evidenceDocument{
				Raw:        s,
				Normalized: norm,
				Tokens:     meaningfulTokens(norm),
				Numbers:    dedupeStrings(numberPattern.FindAllString(strings.ToLower(s), -1)),
				Statuses:   dedupeStrings(lowerAll(statusPattern.FindAllString(s, -1))),
			})
		}

		appendEvidence(result.Text)
		appendEvidence(result.Summary)

		if payload, err := json.Marshal(result.Data); err == nil {
			appendEvidence(string(payload))
		}
	}

	return out
}

func (g *HallucinationGuard) findBestEvidence(claim extractedClaim, corpus []evidenceDocument) evidenceMatch {
	best := evidenceMatch{}

	for _, doc := range corpus {
		score := scoreClaimAgainstEvidence(claim, doc, g.minTokenOverlapRatio)
		if score > best.Score {
			best = evidenceMatch{
				Grounded: score >= g.minEvidenceScore,
				Score:    score,
				Evidence: doc.Raw,
			}
		}
	}

	return best
}

func scoreClaimAgainstEvidence(claim extractedClaim, doc evidenceDocument, minOverlap float64) float64 {
	if claim.Normalized == "" || doc.Normalized == "" {
		return 0
	}

	score := 0.0

	if strings.Contains(doc.Normalized, claim.Normalized) {
		score += 0.85
	}

	tokenOverlap := overlapRatio(claim.Tokens, doc.Tokens)
	if tokenOverlap >= minOverlap {
		score += 0.45 * tokenOverlap
	}

	if len(claim.Numbers) > 0 {
		numMatches := intersectionCount(claim.Numbers, doc.Numbers)
		score += 0.20 * fraction(numMatches, len(claim.Numbers))
	} else {
		score += 0.05
	}

	if len(claim.Statuses) > 0 {
		statusMatches := intersectionCount(claim.Statuses, doc.Statuses)
		score += 0.15 * fraction(statusMatches, len(claim.Statuses))
	}

	if len(claim.Identifiers) > 0 {
		idMatches := countIdentifierMatches(claim.Identifiers, doc.Normalized)
		score += 0.20 * fraction(idMatches, len(claim.Identifiers))
	}

	if claim.Type == claimTypeRecommendation {
		score += 0.10
	}

	if score > 1.0 {
		score = 1.0
	}
	return score
}

func classifyClaim(sentence string, numbers, statuses []string, hasRecommendation bool) claimType {
	lower := strings.ToLower(sentence)

	switch {
	case hasRecommendation:
		return claimTypeRecommendation
	case containsAny(lower, "risk score", "compliance", "control gap", "alert", "incident", "vulnerability", "cve", "ioc", "severity"):
		return claimTypeSecurity
	case len(numbers) > 0:
		return claimTypeNumeric
	case len(statuses) > 0:
		return claimTypeStatus
	default:
		return claimTypeGeneral
	}
}

func looksLikeClaim(sentence string) bool {
	lower := strings.ToLower(strings.TrimSpace(sentence))
	if lower == "" {
		return false
	}
	if len([]rune(lower)) < 8 {
		return false
	}
	if containsAny(lower, "maybe", "might", "could", "possibly") {
		return true
	}
	if numberPattern.MatchString(lower) || statusPattern.MatchString(lower) {
		return true
	}
	if containsAny(lower,
		"is", "are", "was", "were", "has", "have", "had",
		"shows", "indicates", "means", "caused", "affects",
		"risk", "alert", "incident", "compliance", "vulnerability",
		"recommend", "should", "must",
	) {
		return true
	}
	return false
}

func containsRecommendationLanguage(sentence string) bool {
	lower := strings.ToLower(strings.TrimSpace(sentence))
	return containsAny(lower, "recommend", "should", "advise", "prioritize", "focus on", "consider")
}

func claimIsSafeRecommendation(claim extractedClaim) bool {
	return claim.Type == claimTypeRecommendation &&
		len(claim.Numbers) == 0 &&
		!containsAny(strings.ToLower(claim.Raw), "guarantee", "ensure", "will prevent", "eliminate risk")
}

func isCriticalClaim(claim extractedClaim) bool {
	lower := strings.ToLower(claim.Raw)

	if claim.Type == claimTypeSecurity {
		return true
	}

	if containsAny(lower,
		"risk score", "compliance", "non-compliant", "alert", "incident",
		"breach", "compromised", "malware", "critical severity",
		"vulnerability", "exploit", "exposed",
	) {
		return true
	}

	return false
}

func suggestedRepair(claim extractedClaim) string {
	switch claim.Type {
	case claimTypeNumeric:
		return "Restate the figure only if it appears in the tool output, or remove the number."
	case claimTypeStatus:
		return "Tie the status to explicit tool evidence or replace it with a qualified statement."
	case claimTypeSecurity:
		return "Only state this security or compliance conclusion if the tool results explicitly support it."
	case claimTypeRecommendation:
		return "Keep the recommendation, but avoid unsupported certainty or quantified impact."
	default:
		return "Replace with a statement that is directly supported by tool output."
	}
}

func softenClaim(claim extractedClaim) string {
	switch claim.Type {
	case claimTypeNumeric:
		return "I cannot verify that specific figure from the available tool results."
	case claimTypeStatus:
		return "I cannot confirm that status from the available tool results."
	case claimTypeSecurity:
		return "I cannot verify that security or compliance conclusion from the available tool results."
	case claimTypeRecommendation:
		return "A cautious recommendation is to validate this point directly against the tool results."
	default:
		return "I cannot verify that statement from the available tool results."
	}
}

func buildBlockedFallback(results []*llmmodel.ToolCallResult) string {
	parts := make([]string, 0, len(results))

	for _, r := range results {
		if r == nil {
			continue
		}
		if s := strings.TrimSpace(r.Summary); s != "" {
			parts = append(parts, s)
			continue
		}
		if s := strings.TrimSpace(r.Text); s != "" {
			parts = append(parts, truncateText(s, 240))
		}
	}

	if len(parts) == 0 {
		return "I was not able to verify key security-related claims from the available tool results, so I am limiting the response to confirmed evidence only."
	}

	if len(parts) > 3 {
		parts = parts[:3]
	}

	return "I could not verify all key security-related claims. Confirmed tool evidence: " + strings.Join(parts, " | ")
}

func normalizeText(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = punctuationStripper.ReplaceAllString(s, " ")
	s = multiSpacePattern.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func meaningfulTokens(s string) []string {
	raw := strings.Fields(s)
	out := make([]string, 0, len(raw))
	for _, token := range raw {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if _, isStopword := stopwordSet[token]; isStopword {
			continue
		}
		if utfLen(token) < 3 && !numberPattern.MatchString(token) {
			continue
		}
		out = append(out, token)
	}
	return dedupeStrings(out)
}

func extractIdentifiers(s string) []string {
	found := identifierPattern.FindAllString(s, -1)
	out := make([]string, 0, len(found))
	for _, f := range found {
		if len(f) < 3 {
			continue
		}
		if allLetters(f) && len(f) < 5 {
			continue
		}
		if strings.ContainsAny(f, "-_./:") || hasDigit(f) {
			out = append(out, strings.ToLower(f))
		}
	}
	return dedupeStrings(out)
}

func overlapRatio(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	bset := make(map[string]struct{}, len(b))
	for _, item := range b {
		bset[item] = struct{}{}
	}

	matches := 0
	for _, item := range a {
		if _, ok := bset[item]; ok {
			matches++
		}
	}

	return float64(matches) / float64(len(a))
}

func intersectionCount(a, b []string) int {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	setB := make(map[string]struct{}, len(b))
	for _, x := range b {
		setB[x] = struct{}{}
	}
	count := 0
	for _, x := range a {
		if _, ok := setB[x]; ok {
			count++
		}
	}
	return count
}

func countIdentifierMatches(ids []string, normalizedEvidence string) int {
	count := 0
	for _, id := range ids {
		if strings.Contains(normalizedEvidence, strings.ToLower(id)) {
			count++
		}
	}
	return count
}

func fraction(n, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(n) / float64(total)
}

func dedupeStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(strings.ToLower(item))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func lowerAll(in []string) []string {
	out := make([]string, 0, len(in))
	for _, item := range in {
		out = append(out, strings.ToLower(strings.TrimSpace(item)))
	}
	return out
}

func truncateText(s string, max int) string {
	rs := []rune(strings.TrimSpace(s))
	if len(rs) <= max {
		return string(rs)
	}
	return string(rs[:max]) + "..."
}

func utfLen(s string) int {
	return len([]rune(s))
}

func hasDigit(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func allLetters(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return len(s) > 0
}

func containsAny(s string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(s, term) {
			return true
		}
	}
	return false
}

func formatGroundingFailure(result *llmmodel.GroundingResult) string {
	if result == nil {
		return ""
	}
	return fmt.Sprintf("%s (%d/%d grounded, %d ungrounded)",
		result.Status,
		result.GroundedClaims,
		result.TotalClaims,
		len(result.UngroundedClaims),
	)
}
