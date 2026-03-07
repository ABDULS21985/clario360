package classifier

import (
	"sort"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// ClassificationRule defines a single auto-classification rule.
type ClassificationRule struct {
	Name      string
	Priority  int // lower = evaluated first
	Condition func(asset *model.Asset) bool
	Result    model.Criticality
	Reason    string
}

// ClassificationResult carries the output of classifying one asset.
type ClassificationResult struct {
	AssetID     uuid.UUID
	Criticality model.Criticality
	RuleName    string
	Reason      string
	Changed     bool // true if new criticality differs from the asset's current criticality
}

// AssetClassifier evaluates rules against assets and determines their criticality.
type AssetClassifier struct {
	rules  []ClassificationRule
	logger zerolog.Logger
}

// NewAssetClassifier creates a classifier seeded with DefaultRules plus any custom rules.
// Custom rules take precedence when they have a lower (or equal-with-same-name) priority.
func NewAssetClassifier(logger zerolog.Logger, customRules ...ClassificationRule) *AssetClassifier {
	defaults := DefaultRules()
	// Build a map from name → rule so custom rules can override defaults.
	byName := make(map[string]ClassificationRule, len(defaults)+len(customRules))
	for _, r := range defaults {
		byName[r.Name] = r
	}
	for _, r := range customRules {
		byName[r.Name] = r
	}
	rules := make([]ClassificationRule, 0, len(byName))
	for _, r := range byName {
		rules = append(rules, r)
	}
	// Sort ascending by priority (lower priority number = evaluated first)
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})

	return &AssetClassifier{rules: rules, logger: logger}
}

// Classify evaluates all rules against asset and returns the first matching criticality.
// Returns (CriticalityLow, "default") if no rule matches.
func (c *AssetClassifier) Classify(asset *model.Asset) (model.Criticality, string, string) {
	for _, rule := range c.rules {
		if rule.Condition(asset) {
			c.logger.Trace().
				Str("asset_id", asset.ID.String()).
				Str("rule", rule.Name).
				Str("result", string(rule.Result)).
				Msg("classification rule matched")
			return rule.Result, rule.Name, rule.Reason
		}
	}
	return model.CriticalityLow, "default", "no rule matched"
}

// ClassifyBatch classifies a slice of assets, returning one result per asset.
func (c *AssetClassifier) ClassifyBatch(assets []*model.Asset) []ClassificationResult {
	results := make([]ClassificationResult, len(assets))
	for i, asset := range assets {
		crit, ruleName, reason := c.Classify(asset)
		results[i] = ClassificationResult{
			AssetID:     asset.ID,
			Criticality: crit,
			RuleName:    ruleName,
			Reason:      reason,
			Changed:     crit != asset.Criticality,
		}
	}
	return results
}
