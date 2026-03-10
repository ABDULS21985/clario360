package shadow

import (
	"math"
	"sort"
	"strings"
)

// MatchResult represents a potential shadow copy match between two tables.
type MatchResult struct {
	SourceFingerprint TableFingerprint
	TargetFingerprint TableFingerprint
	MatchType         string  // "exact", "structural", "name_similar"
	Similarity        float64 // 0.0-1.0
}

// CompareUniqueFingerprints finds matching tables across a single fingerprint set without
// emitting mirrored duplicates. Each pair is compared once.
func CompareUniqueFingerprints(fingerprints []TableFingerprint, threshold float64) []MatchResult {
	var matches []MatchResult

	for i := 0; i < len(fingerprints); i++ {
		for j := i + 1; j < len(fingerprints); j++ {
			if match, ok := compareFingerprintPair(fingerprints[i], fingerprints[j], threshold); ok {
				matches = append(matches, match)
			}
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Similarity > matches[j].Similarity
	})

	return matches
}

// CompareFingerprints finds matching tables across two sets of fingerprints.
// Returns matches above the similarity threshold.
func CompareFingerprints(source, target []TableFingerprint, threshold float64) []MatchResult {
	var matches []MatchResult

	for _, s := range source {
		for _, t := range target {
			if match, ok := compareFingerprintPair(s, t, threshold); ok {
				matches = append(matches, match)
			}
		}
	}

	// Sort by similarity descending
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Similarity > matches[j].Similarity
	})

	return matches
}

func compareFingerprintPair(source, target TableFingerprint, threshold float64) (MatchResult, bool) {
	// Skip if same source
	if source.SourceID == target.SourceID && source.SourceID != "" {
		return MatchResult{}, false
	}

	// Exact fingerprint match
	if source.Hash != "" && source.Hash == target.Hash {
		return MatchResult{
			SourceFingerprint: source,
			TargetFingerprint: target,
			MatchType:         "exact",
			Similarity:        1.0,
		}, true
	}

	// Structural similarity: same column count + high column overlap
	if len(source.Columns) > 0 && len(target.Columns) > 0 {
		colSimilarity := columnSimilarity(source.Columns, target.Columns)
		if colSimilarity >= threshold {
			return MatchResult{
				SourceFingerprint: source,
				TargetFingerprint: target,
				MatchType:         "structural",
				Similarity:        colSimilarity,
			}, true
		}
	}

	// Name similarity with matching column count
	if nameSimilar(source.TableName, target.TableName) && abs(len(source.Columns)-len(target.Columns)) <= 1 {
		nameSim := nameEditDistanceSimilarity(source.TableName, target.TableName)
		if nameSim >= threshold {
			return MatchResult{
				SourceFingerprint: source,
				TargetFingerprint: target,
				MatchType:         "name_similar",
				Similarity:        nameSim,
			}, true
		}
	}

	return MatchResult{}, false
}

// columnSimilarity computes the Jaccard similarity of column sets.
func columnSimilarity(a, b []ColumnDef) float64 {
	setA := make(map[string]bool, len(a))
	for _, col := range a {
		setA[strings.ToLower(col.Name)] = true
	}

	setB := make(map[string]bool, len(b))
	for _, col := range b {
		setB[strings.ToLower(col.Name)] = true
	}

	intersection := 0
	for name := range setA {
		if setB[name] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// nameSimilar checks if two table names have edit distance < 3.
func nameSimilar(a, b string) bool {
	a = strings.ToLower(a)
	b = strings.ToLower(b)
	if a == b {
		return true
	}
	return editDistance(a, b) < 3
}

// nameEditDistanceSimilarity returns a 0-1 similarity score based on edit distance.
func nameEditDistanceSimilarity(a, b string) float64 {
	a = strings.ToLower(a)
	b = strings.ToLower(b)
	if a == b {
		return 1.0
	}
	maxLen := math.Max(float64(len(a)), float64(len(b)))
	if maxLen == 0 {
		return 1.0
	}
	dist := float64(editDistance(a, b))
	return 1.0 - (dist / maxLen)
}

// editDistance computes the Levenshtein distance between two strings.
func editDistance(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(
				prev[j]+1,
				curr[j-1]+1,
				prev[j-1]+cost,
			)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
