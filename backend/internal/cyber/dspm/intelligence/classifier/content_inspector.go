package classifier

import (
	"sort"

	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// ContentInspector performs content-level data inspection by sampling rows
// and running PII pattern matching on actual cell values.
type ContentInspector struct {
	sampleSize int
}

// NewContentInspector creates a ContentInspector with the given sample size.
func NewContentInspector(sampleSize int) *ContentInspector {
	if sampleSize < 1 {
		sampleSize = 100
	}
	return &ContentInspector{sampleSize: sampleSize}
}

// Inspect examines sampleRows for PII content by running value-level pattern
// matching on each column. It returns a ContentInspectionResult per column
// where at least one match was found, sorted by confidence descending.
func (ci *ContentInspector) Inspect(columns []string, sampleRows [][]string) []model.ContentInspectionResult {
	if len(columns) == 0 || len(sampleRows) == 0 {
		return nil
	}

	patterns := getCompiledPatterns()
	actualSampleSize := len(sampleRows)
	if actualSampleSize > ci.sampleSize {
		actualSampleSize = ci.sampleSize
	}

	var results []model.ContentInspectionResult

	for colIdx, colName := range columns {
		bestMatch := struct {
			patternName string
			matchCount  int
			samples     []string
		}{}

		// Track which pattern produces the most matches for this column.
		type patternHit struct {
			name    string
			count   int
			samples []string
		}
		hitsByPattern := make(map[string]*patternHit)

		for rowIdx := 0; rowIdx < actualSampleSize && rowIdx < len(sampleRows); rowIdx++ {
			row := sampleRows[rowIdx]
			if colIdx >= len(row) {
				continue
			}
			value := row[colIdx]
			if value == "" {
				continue
			}

			for _, p := range patterns {
				if p.compiled.MatchString(value) {
					hit, ok := hitsByPattern[p.Name]
					if !ok {
						hit = &patternHit{name: p.Name}
						hitsByPattern[p.Name] = hit
					}
					hit.count++
					if len(hit.samples) < 3 {
						hit.samples = append(hit.samples, value)
					}
				}
			}
		}

		// Find the pattern with the highest match count for this column.
		for _, hit := range hitsByPattern {
			if hit.count > bestMatch.matchCount {
				bestMatch.patternName = hit.name
				bestMatch.matchCount = hit.count
				bestMatch.samples = hit.samples
			}
		}

		if bestMatch.matchCount == 0 {
			continue
		}

		matchRate := float64(bestMatch.matchCount) / float64(actualSampleSize)
		confidence := matchRateToConfidence(matchRate)

		results = append(results, model.ContentInspectionResult{
			ColumnName:    colName,
			SampleSize:    actualSampleSize,
			MatchCount:    bestMatch.matchCount,
			MatchRate:     matchRate,
			DetectedType:  bestMatch.patternName,
			SampleMatches: bestMatch.samples,
			Confidence:    confidence,
		})
	}

	// Sort by confidence descending.
	sort.Slice(results, func(i, j int) bool {
		return results[i].Confidence > results[j].Confidence
	})

	return results
}

// matchRateToConfidence maps a match rate to a confidence score.
//
//	90%+ match rate  -> 0.95 confidence
//	50-90% match rate -> 0.70 confidence
//	<50% match rate  -> 0.40 confidence
func matchRateToConfidence(rate float64) float64 {
	switch {
	case rate >= 0.90:
		return 0.95
	case rate >= 0.50:
		return 0.70
	default:
		return 0.40
	}
}
