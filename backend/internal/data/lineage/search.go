package lineage

import (
	"sort"
	"strings"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/model"
)

// EntitySearcher performs relevance-ranked search over lineage graph nodes.
// It is stateless — all search logic operates on the provided node slice with no I/O.
type EntitySearcher struct {
	logger zerolog.Logger
}

// NewEntitySearcher creates a new EntitySearcher.
func NewEntitySearcher(logger zerolog.Logger) *EntitySearcher {
	return &EntitySearcher{logger: logger}
}

// nodeScore pairs a node with its computed relevance score and the fields that matched.
type nodeScore struct {
	node        model.LineageNode
	score       float64
	matchFields []string
}

// Search returns at most limit results from the provided slice that match query
// and entityType, ordered by relevance score (highest first). Tie-breaks are
// resolved by node ID for deterministic output.
//
// Each result carries the matched node, its relevance score, and the list of
// fields that contributed to the match (for client-side highlighting).
//
// Scoring rules (additive — a node can match multiple fields):
//
//	100 — exact name match (case-insensitive)
//	 80 — name starts with query, or query starts with name (prefix match)
//	 60 — name contains query (substring)
//	 40 — node compound ID (e.g. "data_source:<uuid>") contains query
//	 30 — status field contains query
//	 20 — any metadata string value contains query
//	 10 — any metadata key contains query, or bool metadata value equals query
//
// Filters:
//   - entityType: if non-empty only nodes of that type are considered (checked before scoring).
//   - query: if empty, all nodes of the requested type are returned with score 0.
//   - limit: capped to 25 when ≤ 0.
func (s *EntitySearcher) Search(nodes []model.LineageNode, query, entityType string, limit int) []model.LineageSearchResult {
	if limit <= 0 {
		limit = 25
	}
	q := strings.TrimSpace(strings.ToLower(query))
	typ := strings.TrimSpace(strings.ToLower(entityType))

	scored := make([]nodeScore, 0, len(nodes))
	for _, node := range nodes {
		// Type filter: O(1) early exit before any string operations.
		if typ != "" && strings.ToLower(node.Type) != typ {
			continue
		}

		// Empty query: include all matching-type nodes with a neutral score.
		if q == "" {
			scored = append(scored, nodeScore{node: node})
			continue
		}

		sc, fields := s.scoreNode(node, q)
		if sc <= 0 {
			continue // Node does not match the query.
		}
		scored = append(scored, nodeScore{node: node, score: sc, matchFields: fields})
	}

	// Primary sort: score descending.
	// Secondary sort: node ID ascending (lexicographic) for deterministic tie-breaking.
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].node.ID < scored[j].node.ID
	})

	if limit < len(scored) {
		scored = scored[:limit]
	}
	results := make([]model.LineageSearchResult, len(scored))
	for i, ns := range scored {
		results[i] = model.LineageSearchResult{
			Node:        ns.node,
			Score:       ns.score,
			MatchFields: ns.matchFields,
		}
	}
	return results
}

// scoreNode computes the relevance score and the list of matched fields for a single node
// against the given (already lower-cased and trimmed) query string.
// Returns (0, nil) when the node does not match the query at all.
func (s *EntitySearcher) scoreNode(node model.LineageNode, query string) (float64, []string) {
	var total float64
	fields := make([]string, 0, 4)

	// ── Name match ───────────────────────────────────────────────────────────
	nameLower := strings.ToLower(node.Name)
	switch {
	case nameLower == query:
		// Exact match: highest confidence.
		total += 100
		fields = append(fields, "name")
	case strings.HasPrefix(nameLower, query) || strings.HasPrefix(query, nameLower):
		// Prefix match in either direction (e.g. query "employ" matches "employees",
		// or query "employees_2024" matches node named "employees").
		total += 80
		fields = append(fields, "name")
	case strings.Contains(nameLower, query):
		// General substring match.
		total += 60
		fields = append(fields, "name")
	}

	// ── Compound ID match ─────────────────────────────────────────────────────
	// The compound ID has the form "{entity_type}:{uuid}". Useful for UUID-based lookups.
	if strings.Contains(strings.ToLower(node.ID), query) {
		total += 40
		fields = searchAppendUnique(fields, "id")
	}

	// ── Status match ──────────────────────────────────────────────────────────
	if node.Status != "" && strings.Contains(strings.ToLower(node.Status), query) {
		total += 30
		fields = searchAppendUnique(fields, "status")
	}

	// ── Metadata match ────────────────────────────────────────────────────────
	// Only string and bool metadata values are searched. Numeric values are skipped
	// because substring matching against numbers produces misleading results.
	for key, val := range node.Metadata {
		keyLower := strings.ToLower(key)
		fieldName := "metadata." + key

		// Match the metadata key name itself.
		if strings.Contains(keyLower, query) {
			total += 10
			fields = searchAppendUnique(fields, fieldName)
			continue
		}

		// Match the metadata value.
		switch typed := val.(type) {
		case string:
			if strings.Contains(strings.ToLower(typed), query) {
				total += 20
				fields = searchAppendUnique(fields, fieldName)
			}
		case bool:
			// Match "true" or "false" literally.
			boolStr := "false"
			if typed {
				boolStr = "true"
			}
			if boolStr == query {
				total += 10
				fields = searchAppendUnique(fields, fieldName)
			}
		}
	}

	return total, fields
}

// searchAppendUnique appends s to slice only if not already present.
// Using a short name prefix ("search") to avoid collision with any future package-level helpers.
func searchAppendUnique(slice []string, s string) []string {
	for _, existing := range slice {
		if existing == s {
			return slice
		}
	}
	return append(slice, s)
}
