package classifier

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// CustomClassifier evaluates tenant-defined custom classification rules against
// column names and sample values.
type CustomClassifier struct {
	logger zerolog.Logger
}

// NewCustomClassifier creates a CustomClassifier.
func NewCustomClassifier(logger zerolog.Logger) *CustomClassifier {
	return &CustomClassifier{
		logger: logger.With().Str("component", "custom_classifier").Logger(),
	}
}

// Evaluate matches column names against rule.ColumnPatterns (supports glob
// patterns like "project_*", "*_project_id") and optionally matches sample
// values against rule.ValuePattern.
func (cc *CustomClassifier) Evaluate(
	rules []model.CustomClassificationRule,
	columnNames []string,
	sampleValues map[string][]string,
) []model.PatternMatch {
	if len(rules) == 0 || len(columnNames) == 0 {
		return nil
	}

	var matches []model.PatternMatch

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		cc.logger.Debug().
			Str("rule", rule.Name).
			Int("patterns", len(rule.ColumnPatterns)).
			Msg("evaluating custom rule")

		for _, colName := range columnNames {
			matched := cc.matchColumnPatterns(colName, rule.ColumnPatterns)
			if !matched {
				continue
			}

			// Column name matched. Now check value pattern if set.
			valueMatchCount := 0
			sampleSize := 0

			if rule.ValuePattern != "" && sampleValues != nil {
				values, ok := sampleValues[colName]
				if ok && len(values) > 0 {
					sampleSize = len(values)
					re, err := regexp.Compile(rule.ValuePattern)
					if err != nil {
						cc.logger.Warn().
							Err(err).
							Str("rule", rule.Name).
							Str("value_pattern", rule.ValuePattern).
							Msg("invalid value pattern regex in custom rule")
						continue
					}
					for _, v := range values {
						if re.MatchString(v) {
							valueMatchCount++
						}
					}
					// If value pattern is set but no values matched, skip this column.
					if valueMatchCount == 0 {
						continue
					}
				}
			}

			piiType := rule.PIIType
			if piiType == "" {
				piiType = rule.Classification
			}

			match := model.PatternMatch{
				PatternName: rule.Name,
				ColumnName:  colName,
				Regex:       strings.Join(rule.ColumnPatterns, "|"),
				Weight:      classificationToWeight(rule.Classification),
				MatchCount:  valueMatchCount,
				SampleSize:  sampleSize,
			}

			matches = append(matches, match)

			cc.logger.Debug().
				Str("rule", rule.Name).
				Str("column", colName).
				Int("value_matches", valueMatchCount).
				Msg("custom rule matched")
		}
	}

	return matches
}

// matchColumnPatterns checks if a column name matches any of the given glob patterns.
func (cc *CustomClassifier) matchColumnPatterns(colName string, patterns []string) bool {
	lowerCol := strings.ToLower(colName)
	for _, pattern := range patterns {
		lowerPattern := strings.ToLower(pattern)

		// Use filepath.Match for glob matching (supports *, ?, []).
		matched, err := filepath.Match(lowerPattern, lowerCol)
		if err != nil {
			cc.logger.Warn().
				Err(err).
				Str("pattern", pattern).
				Msg("invalid glob pattern in custom rule")
			continue
		}
		if matched {
			return true
		}

		// Also try as a contains match for simple substring patterns
		// (patterns without glob characters).
		if !containsGlobChars(lowerPattern) && strings.Contains(lowerCol, lowerPattern) {
			return true
		}
	}
	return false
}

// containsGlobChars returns true if the pattern contains glob meta characters.
func containsGlobChars(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

// classificationToWeight maps classification levels to numeric weights.
func classificationToWeight(classification string) float64 {
	switch strings.ToLower(classification) {
	case "top_secret":
		return 1.0
	case "restricted":
		return 0.95
	case "confidential":
		return 0.80
	case "internal":
		return 0.50
	case "public":
		return 0.20
	default:
		return 0.50
	}
}
