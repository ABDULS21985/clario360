package engine

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	llmmodel "github.com/clario360/platform/internal/cyber/vciso/llm/model"
)

var (
	numberPattern = regexp.MustCompile(`\b\d+(?:\.\d+)?%?\b`)
	entityPattern = regexp.MustCompile(`\b[a-zA-Z][a-zA-Z0-9._-]{2,}\b`)
	statusPattern = regexp.MustCompile(`(?i)\b(failing|healthy|critical|resolved|open|closed|passed|degraded)\b`)
)

type HallucinationGuard struct{}

func NewHallucinationGuard() *HallucinationGuard {
	return &HallucinationGuard{}
}

func (g *HallucinationGuard) Check(response string, toolResults []*llmmodel.ToolCallResult) *llmmodel.GroundingResult {
	claims := extractClaims(response)
	if len(claims) == 0 {
		return &llmmodel.GroundingResult{Status: "passed", TotalClaims: 0, GroundedClaims: 0}
	}
	corpus := buildCorpus(toolResults)
	result := &llmmodel.GroundingResult{
		Status:         "passed",
		TotalClaims:    len(claims),
		GroundedClaims: 0,
	}
	for _, claim := range claims {
		if isRecommendationClaim(claim) {
			result.GroundedClaims++
			continue
		}
		if strings.Contains(corpus, strings.ToLower(claim)) {
			result.GroundedClaims++
			continue
		}
		critical := strings.Contains(strings.ToLower(response), "risk score") || strings.Contains(strings.ToLower(response), "compliance") || strings.Contains(strings.ToLower(response), "alert")
		result.UngroundedClaims = append(result.UngroundedClaims, llmmodel.UngroundedClaim{
			Claim:      claim,
			Type:       claimType(claim),
			Critical:   critical,
			Suggestion: "Replace with a grounded statement based on tool output.",
		})
	}
	if len(result.UngroundedClaims) == 0 {
		return result
	}
	for _, item := range result.UngroundedClaims {
		if item.Critical {
			result.Status = "blocked"
			result.CorrectedResponse = "I wasn't able to verify all of the data in my response. I'm falling back to the confirmed tool results."
			return result
		}
	}
	result.Status = "corrected"
	result.CorrectedResponse = response
	for _, item := range result.UngroundedClaims {
		result.CorrectedResponse = strings.ReplaceAll(result.CorrectedResponse, item.Claim, "[unverified]")
	}
	return result
}

func extractClaims(response string) []string {
	lines := []string{}
	lines = append(lines, numberPattern.FindAllString(response, -1)...)
	lines = append(lines, statusPattern.FindAllString(response, -1)...)
	for _, entity := range entityPattern.FindAllString(response, -1) {
		if len(entity) > 3 && strings.ContainsAny(entity, "-_.") {
			lines = append(lines, entity)
		}
	}
	return cleanStrings(lines)
}

func buildCorpus(results []*llmmodel.ToolCallResult) string {
	builder := strings.Builder{}
	for _, result := range results {
		if result == nil {
			continue
		}
		builder.WriteString(strings.ToLower(result.Text))
		builder.WriteRune(' ')
		builder.WriteString(strings.ToLower(result.Summary))
		builder.WriteRune(' ')
		payload, _ := json.Marshal(result.Data)
		builder.WriteString(strings.ToLower(string(payload)))
		builder.WriteRune(' ')
	}
	return builder.String()
}

func claimType(claim string) string {
	switch {
	case numberPattern.MatchString(claim):
		return "numeric"
	case statusPattern.MatchString(claim):
		return "status"
	default:
		return "entity"
	}
}

func isRecommendationClaim(claim string) bool {
	lower := strings.ToLower(claim)
	return strings.Contains(lower, "recommend") || strings.Contains(lower, "should")
}

func formatGroundingFailure(result *llmmodel.GroundingResult) string {
	if result == nil {
		return ""
	}
	return fmt.Sprintf("%s (%d/%d grounded)", result.Status, result.GroundedClaims, result.TotalClaims)
}
