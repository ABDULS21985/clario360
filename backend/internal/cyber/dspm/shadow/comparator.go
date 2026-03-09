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

// CompareFingerprints finds matching tables across two sets of fingerprints.
// Returns matches above the similarity threshold.
func CompareFingerprints(source, target []TableFingerprint, threshold float64) []MatchResult {
	var matches []MatchResult

	for _, s := range source {
		for _, t := range target {
			// Skip if same source
			if s.SourceID == t.SourceID && s.SourceID != "" {
				continue
			}

			// Exact fingerprint match
			if s.Hash != "" && s.Hash == t.Hash {
				matches = append(matches, MatchResult{
					SourceFingerprint: s,
					TargetFingerprint: t,
					MatchType:         "exact",
					Similarity:        1.0,
				})
				continue
			}

			// Structural similarity: same column count + high column overlap
			if len(s.Columns) > 0 && len(t.Columns) > 0 {
				colSimilarity := columnSimilarity(s.Columns, t.Columns)
				if colSimilarity >= threshold {
					matches = append(matches, MatchResult{
						SourceFingerprint: s,
						TargetFingerprint: t,
						MatchType:         "structural",
						Similarity:        colSimilarity,
					})
					continue
				}
			}

			// Name similarity with matching column count
			if nameSimilar(s.TableName, t.TableName) && abs(len(s.Columns)-len(t.Columns)) <= 1 {
				nameSim := nameEditDistanceSimilarity(s.TableName, t.TableName)
				if nameSim >= threshold {
					matches = append(matches, MatchResult{
						SourceFingerprint: s,
						TargetFingerprint: t,
						MatchType:         "name_similar",
						Similarity:        nameSim,
					})
				}
			}
		}
	}

	// Sort by similarity descending
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Similarity > matches[j].Similarity
	})

	return matches
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
