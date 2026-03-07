package mitre

import "github.com/clario360/platform/internal/cyber/model"

// TechniqueCoverage describes how many detection rules map to a technique.
type TechniqueCoverage struct {
	Technique    Technique `json:"technique"`
	HasDetection bool      `json:"has_detection"`
	RuleCount    int       `json:"rule_count"`
	RuleNames    []string  `json:"rule_names"`
}

// BuildCoverage computes ATT&CK coverage from the tenant's enabled rules.
func BuildCoverage(rules []*model.DetectionRule) []TechniqueCoverage {
	byTechnique := make(map[string][]string)
	for _, rule := range rules {
		for _, techniqueID := range rule.MITRETechniqueIDs {
			byTechnique[techniqueID] = append(byTechnique[techniqueID], rule.Name)
		}
	}

	results := make([]TechniqueCoverage, 0, len(techniques))
	for _, technique := range techniques {
		ruleNames := byTechnique[technique.ID]
		results = append(results, TechniqueCoverage{
			Technique:    technique,
			HasDetection: len(ruleNames) > 0,
			RuleCount:    len(ruleNames),
			RuleNames:    append([]string(nil), ruleNames...),
		})
	}
	return results
}
