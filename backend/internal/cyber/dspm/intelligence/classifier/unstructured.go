package classifier

import (
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

type matchInfo struct {
	patternName string
	category    string
	weight      float64
	matchCount  int
	samples     []string
}

// UnstructuredClassifier classifies unstructured data (documents, files, free text)
// by scanning text content against all PII patterns.
type UnstructuredClassifier struct {
	patterns []model.PIIPattern
	logger   zerolog.Logger
}

// NewUnstructuredClassifier creates an UnstructuredClassifier with the full pattern library.
func NewUnstructuredClassifier(logger zerolog.Logger) *UnstructuredClassifier {
	return &UnstructuredClassifier{
		patterns: AllPatterns(),
		logger:   logger.With().Str("component", "unstructured_classifier").Logger(),
	}
}

// ClassifyText scans the provided text content against all PII patterns and
// returns an EnhancedClassification based on the highest-sensitivity PII found.
func (u *UnstructuredClassifier) ClassifyText(text string) *model.EnhancedClassification {
	if text == "" {
		return &model.EnhancedClassification{
			Classification: "public",
			Confidence:     1.0,
			PIITypes:       []string{},
			DetectedBy:     model.ClassMethodContent,
			Evidence: model.ClassificationEvidence{
				Explanation: "No text content to classify.",
			},
		}
	}

	compiled := getCompiledPatterns()

	matchesByType := make(map[string]*matchInfo)

	for _, cp := range compiled {
		allMatches := cp.compiled.FindAllString(text, -1)
		if len(allMatches) == 0 {
			continue
		}

		info, exists := matchesByType[cp.Name]
		if !exists {
			info = &matchInfo{
				patternName: cp.Name,
				category:    cp.Category,
				weight:      cp.Weight,
			}
			matchesByType[cp.Name] = info
		}

		info.matchCount += len(allMatches)
		// Keep up to 3 sample matches.
		for _, m := range allMatches {
			if len(info.samples) < 3 {
				// Redact most of the match for safety.
				redacted := redactSample(m)
				info.samples = append(info.samples, redacted)
			}
		}
	}

	if len(matchesByType) == 0 {
		u.logger.Debug().Msg("no PII patterns matched in text")
		return &model.EnhancedClassification{
			Classification: "internal",
			Confidence:     0.70,
			PIITypes:       []string{},
			DetectedBy:     model.ClassMethodContent,
			Evidence: model.ClassificationEvidence{
				Explanation: "No PII patterns detected in text content. Defaulting to internal classification.",
			},
		}
	}

	// Determine classification from the highest-sensitivity match.
	highestRank := 0
	var piiTypes []string
	var patternMatches []model.PatternMatch
	var totalWeight float64

	for _, info := range matchesByType {
		piiTypes = append(piiTypes, info.patternName)

		rank := piiTypeToRank(info.patternName, info.category)
		if rank > highestRank {
			highestRank = rank
		}

		totalWeight += info.weight

		patternMatches = append(patternMatches, model.PatternMatch{
			PatternName: info.patternName,
			ColumnName:  "text_content",
			Weight:      info.weight,
			MatchCount:  info.matchCount,
		})
	}

	sort.Strings(piiTypes)

	// Confidence increases with the number of distinct PII types and match count.
	confidence := calculateTextConfidence(matchesByType)

	classification := rankToClassification[highestRank]

	result := &model.EnhancedClassification{
		AssetID:           uuid.Nil,
		Classification:    classification,
		PIITypes:          piiTypes,
		Confidence:        confidence,
		ContentConfidence: confidence,
		NeedsHumanReview:  confidence < 0.5,
		DetectedBy:        model.ClassMethodContent,
		Evidence: model.ClassificationEvidence{
			PatternMatches: patternMatches,
		},
	}

	// Build explanation.
	var parts []string
	parts = append(parts, "Unstructured text analysis complete.")
	parts = append(parts, strings.Join([]string{
		"Classification: " + strings.ToUpper(classification),
		"PII types found: " + strings.Join(piiTypes, ", "),
	}, ". "))
	if result.NeedsHumanReview {
		parts = append(parts, "LOW CONFIDENCE: Human review recommended.")
	}
	result.Evidence.Explanation = strings.Join(parts, " ")

	u.logger.Info().
		Str("classification", classification).
		Float64("confidence", confidence).
		Int("pii_types", len(piiTypes)).
		Msg("unstructured text classified")

	return result
}

// calculateTextConfidence computes confidence based on match diversity and frequency.
func calculateTextConfidence(matches map[string]*matchInfo) float64 {
	if len(matches) == 0 {
		return 0
	}

	// More distinct PII types increases confidence.
	typeCount := float64(len(matches))
	typeFactor := typeCount / 5.0
	if typeFactor > 1.0 {
		typeFactor = 1.0
	}

	// Higher total match counts increase confidence.
	var totalMatches int
	var maxWeight float64
	for _, m := range matches {
		totalMatches += m.matchCount
		if m.weight > maxWeight {
			maxWeight = m.weight
		}
	}

	countFactor := float64(totalMatches) / 10.0
	if countFactor > 1.0 {
		countFactor = 1.0
	}

	// Weighted combination.
	confidence := (typeFactor*0.3 + countFactor*0.3 + maxWeight*0.4)
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// redactSample partially redacts a matched value, showing only the first and
// last two characters for privacy.
func redactSample(value string) string {
	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}
