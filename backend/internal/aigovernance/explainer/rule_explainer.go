package explainer

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
)

type RuleExplainer struct{}

func NewRuleExplainer() *RuleExplainer {
	return &RuleExplainer{}
}

func (e *RuleExplainer) Explain(_ context.Context, version *aigovmodel.ModelVersion, _ any, output *aigovernance.ModelOutput) (*aigovmodel.Explanation, error) {
	meta := copyMap(output.Metadata)
	matchedRules := stringSlice(meta["matched_rules"])
	matchedConditions := stringSlice(meta["matched_conditions"])
	unmatchedConditions := stringSlice(meta["unmatched_conditions"])
	weights := numericMap(meta["rule_weights"])
	matched, hasMatchedFlag := boolValue(meta["matched"])

	if len(matchedRules) == 0 && len(matchedConditions) == 0 && (!hasMatchedFlag || matched) {
		if ruleName, ok := meta["rule_name"].(string); ok && strings.TrimSpace(ruleName) != "" {
			matchedRules = []string{ruleName}
		}
	}

	factors := make([]aigovmodel.Factor, 0, len(matchedRules)+len(matchedConditions))
	for _, ruleName := range matchedRules {
		impact := weights[ruleName]
		if impact == 0 {
			impact = 0.2
		}
		factors = append(factors, aigovmodel.Factor{
			Name:        ruleName,
			Value:       "matched",
			Impact:      impact,
			Direction:   "positive",
			Description: fmt.Sprintf("Rule %s matched the supplied input.", ruleName),
		})
	}
	for _, condition := range matchedConditions {
		factors = append(factors, aigovmodel.Factor{
			Name:        condition,
			Value:       "true",
			Impact:      0.1,
			Direction:   "positive",
			Description: "A configured rule condition evaluated to true.",
		})
	}
	sort.SliceStable(factors, func(i, j int) bool {
		return factors[i].Impact > factors[j].Impact
	})

	structured := map[string]any{
		"matched_rules":        matchedRules,
		"matched_conditions":   matchedConditions,
		"unmatched_conditions": unmatchedConditions,
		"rule_count":           len(matchedRules),
	}
	if hasMatchedFlag {
		structured["matched"] = matched
	} else {
		structured["matched"] = len(matchedRules) > 0 || len(matchedConditions) > 0
	}
	for key, value := range meta {
		if _, exists := structured[key]; !exists {
			structured[key] = value
		}
	}

	human := ""
	if hasMatchedFlag && !matched && len(matchedRules) == 0 && len(matchedConditions) == 0 {
		human = "No configured rules matched."
	}
	if human == "" {
		rendered, err := renderTemplate(version, map[string]any{
			"matched":              structured["matched"],
			"matched_rules":        matchedRules,
			"matched_conditions":   matchedConditions,
			"unmatched_conditions": unmatchedConditions,
			"metadata":             meta,
			"confidence":           output.Confidence,
		})
		if err != nil {
			return nil, err
		}
		human = rendered
	}
	if human == "" {
		names := matchedRules
		if len(names) == 0 {
			names = matchedConditions
		}
		if len(names) == 0 {
			human = "No configured rules matched."
		} else {
			human = fmt.Sprintf("Decision made because %d rules matched: %s.", len(names), strings.Join(names, ", "))
		}
	}

	return &aigovmodel.Explanation{
		Structured:    structured,
		HumanReadable: human,
		Factors:       factors,
		Confidence:    output.Confidence,
		ExplainerType: string(aigovmodel.ExplainabilityRuleTrace),
		ModelSlug:     version.ModelSlug,
		ModelVersion:  version.VersionNumber,
	}, nil
}

func boolValue(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	default:
		return false, false
	}
}

func renderTemplate(version *aigovmodel.ModelVersion, data map[string]any) (string, error) {
	if version == nil || version.ExplanationTemplate == nil || strings.TrimSpace(*version.ExplanationTemplate) == "" {
		return "", nil
	}
	tmpl, err := template.New("explanation").Funcs(templateFuncs()).Parse(*version.ExplanationTemplate)
	if err != nil {
		return "", fmt.Errorf("parse explanation template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute explanation template: %w", err)
	}
	return strings.TrimSpace(buf.String()), nil
}

func copyMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(value))
	for key, item := range value {
		out[key] = item
	}
	return out
}

func stringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func numericMap(value any) map[string]float64 {
	out := map[string]float64{}
	typed, ok := value.(map[string]any)
	if !ok {
		return out
	}
	for key, item := range typed {
		switch number := item.(type) {
		case float64:
			out[key] = number
		case float32:
			out[key] = float64(number)
		case int:
			out[key] = float64(number)
		}
	}
	return out
}
