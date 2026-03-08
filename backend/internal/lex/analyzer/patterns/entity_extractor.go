package patterns

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/clario360/platform/internal/lex/model"
)

type EntityExtractor struct {
	partyBetween  *regexp.Regexp
	partyLabelA   *regexp.Regexp
	partyLabelB   *regexp.Regexp
	datePatterns  map[string][]*regexp.Regexp
	amountPattern *regexp.Regexp
}

func NewEntityExtractor() *EntityExtractor {
	return &EntityExtractor{
		partyBetween: regexp.MustCompile(`(?is)(?:by\s+and\s+between|between)\s+([A-Z][A-Za-z0-9&,\.\- ]+?)\s+(?:and|,)\s+([A-Z][A-Za-z0-9&,\.\- ]+?)(?:\n|,|\.|$)`),
		partyLabelA:  regexp.MustCompile(`(?im)^Party\s*A:\s*(.+)$`),
		partyLabelB:  regexp.MustCompile(`(?im)^Party\s*B:\s*(.+)$`),
		datePatterns: map[string][]*regexp.Regexp{
			"effective_date": {
				regexp.MustCompile(`(?i)effective(?:\s+as\s+of|\s+date is|\s+on)?\s+([A-Za-z]+\s+\d{1,2},\s+\d{4})`),
				regexp.MustCompile(`(?i)effective(?:\s+as\s+of|\s+date is|\s+on)?\s+(\d{4}-\d{2}-\d{2})`),
			},
			"expiry_date": {
				regexp.MustCompile(`(?i)(?:expiry|expiration|end|termination)\s+date(?:\s+is|\s+on)?\s+([A-Za-z]+\s+\d{1,2},\s+\d{4})`),
				regexp.MustCompile(`(?i)(?:expiry|expiration|end|termination)\s+date(?:\s+is|\s+on)?\s+(\d{4}-\d{2}-\d{2})`),
			},
			"renewal_date": {
				regexp.MustCompile(`(?i)renewal\s+date(?:\s+is|\s+on)?\s+([A-Za-z]+\s+\d{1,2},\s+\d{4})`),
				regexp.MustCompile(`(?i)renewal\s+date(?:\s+is|\s+on)?\s+(\d{4}-\d{2}-\d{2})`),
			},
		},
		amountPattern: regexp.MustCompile(`(?i)\b(SAR|USD|EUR|GBP|AED|€|\$)\s*([0-9]{1,3}(?:,[0-9]{3})*(?:\.[0-9]{2})?|[0-9]+(?:\.[0-9]{2})?)`),
	}
}

func (e *EntityExtractor) ExtractParties(text string) []model.PartyExtraction {
	seen := map[string]struct{}{}
	var out []model.PartyExtraction
	if match := e.partyBetween.FindStringSubmatch(text); len(match) == 3 {
		for idx, name := range []string{match[1], match[2]} {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			key := strings.ToLower(name)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			role := "party_a"
			if idx == 1 {
				role = "party_b"
			}
			out = append(out, model.PartyExtraction{Name: name, Role: role, Source: "between_clause"})
		}
	}
	for _, match := range e.partyLabelA.FindAllStringSubmatch(text, -1) {
		name := strings.TrimSpace(match[1])
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, model.PartyExtraction{Name: name, Role: "party_a", Source: "label"})
	}
	for _, match := range e.partyLabelB.FindAllStringSubmatch(text, -1) {
		name := strings.TrimSpace(match[1])
		key := strings.ToLower(name)
		if name == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, model.PartyExtraction{Name: name, Role: "party_b", Source: "label"})
	}
	return out
}

func (e *EntityExtractor) ExtractDates(text string) []model.ExtractedDate {
	// Iterate labels in deterministic order so that the output slice is
	// ordered by the position of the first match in the source text.
	labelOrder := []string{"effective_date", "expiry_date", "renewal_date"}

	type match struct {
		date model.ExtractedDate
		pos  int
	}
	var matches []match
	for _, label := range labelOrder {
		patterns := e.datePatterns[label]
		for _, pattern := range patterns {
			if m := pattern.FindStringSubmatchIndex(text); m != nil {
				raw := text[m[2]:m[3]]
				value := parseDocumentDate(raw)
				matches = append(matches, match{
					date: model.ExtractedDate{Label: label, Value: value, Source: strings.TrimSpace(raw)},
					pos:  m[0],
				})
				break
			}
		}
	}

	sort.Slice(matches, func(i, j int) bool { return matches[i].pos < matches[j].pos })
	out := make([]model.ExtractedDate, len(matches))
	for i, m := range matches {
		out[i] = m.date
	}
	return out
}

func (e *EntityExtractor) ExtractAmounts(text string) []model.ExtractedAmount {
	matches := e.amountPattern.FindAllStringSubmatch(text, -1)
	out := make([]model.ExtractedAmount, 0, len(matches))
	for _, match := range matches {
		value, err := parseAmount(match[2])
		if err != nil {
			continue
		}
		label := "amount"
		context := strings.ToLower(match[0])
		switch {
		case strings.Contains(context, "annual"):
			label = "annual_fee"
		case strings.Contains(context, "total"):
			label = "total_value"
		}
		out = append(out, model.ExtractedAmount{
			Label:    label,
			Currency: normalizeCurrencyCode(match[1]),
			Value:    value,
			Source:   strings.TrimSpace(match[0]),
		})
	}
	return out
}

func parseDocumentDate(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	layouts := []string{
		"January 2, 2006",
		"2 January 2006",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			value := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
			return &value
		}
	}
	return nil
}

func parseAmount(raw string) (float64, error) {
	raw = strings.ReplaceAll(raw, ",", "")
	return strconv.ParseFloat(raw, 64)
}

func normalizeCurrencyCode(raw string) string {
	switch strings.TrimSpace(strings.ToUpper(raw)) {
	case "$":
		return "USD"
	case "€":
		return "EUR"
	default:
		return strings.ToUpper(strings.TrimSpace(raw))
	}
}

func (e *EntityExtractor) Extract(text string) (parties []model.PartyExtraction, dates []model.ExtractedDate, amounts []model.ExtractedAmount) {
	return e.ExtractParties(text), e.ExtractDates(text), e.ExtractAmounts(text)
}

func (e *EntityExtractor) WarnOnMetadataMismatch(contractPartyA, contractPartyB string, parties []model.PartyExtraction) []string {
	var warnings []string
	normalized := make([]string, 0, len(parties))
	for _, party := range parties {
		normalized = append(normalized, strings.ToLower(strings.TrimSpace(party.Name)))
	}
	if contractPartyA != "" && !containsString(normalized, strings.ToLower(strings.TrimSpace(contractPartyA))) {
		warnings = append(warnings, fmt.Sprintf("party A metadata %q not found in contract text", contractPartyA))
	}
	if contractPartyB != "" && !containsString(normalized, strings.ToLower(strings.TrimSpace(contractPartyB))) {
		warnings = append(warnings, fmt.Sprintf("party B metadata %q not found in contract text", contractPartyB))
	}
	return warnings
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
