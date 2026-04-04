package lineage

import (
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/model"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func newTestSearcher() *EntitySearcher {
	return NewEntitySearcher(zerolog.Nop())
}

func makeSearchNode(entityType model.LineageEntityType, name, status string, meta map[string]any) model.LineageNode {
	id := uuid.New()
	return model.LineageNode{
		ID:       nodeKey(entityType, id),
		EntityID: id,
		Type:     string(entityType),
		Name:     name,
		Status:   status,
		Metadata: meta,
	}
}

func containsSearchField(fields []string, target string) bool {
	for _, f := range fields {
		if f == target {
			return true
		}
	}
	return false
}

// ─── empty graph / nil input ──────────────────────────────────────────────────

func TestSearch_EmptyGraph(t *testing.T) {
	results := newTestSearcher().Search(nil, "anything", "", 10)
	if len(results) != 0 {
		t.Fatalf("expected 0 results on nil graph, got %d", len(results))
	}
}

func TestSearch_EmptySlice(t *testing.T) {
	results := newTestSearcher().Search([]model.LineageNode{}, "query", "", 10)
	if len(results) != 0 {
		t.Fatalf("expected 0 results on empty slice, got %d", len(results))
	}
}

// ─── empty query ──────────────────────────────────────────────────────────────

func TestSearch_EmptyQuery_ReturnsAll(t *testing.T) {
	nodes := []model.LineageNode{
		makeSearchNode(model.LineageEntityDataSource, "Source A", "active", nil),
		makeSearchNode(model.LineageEntityDataModel, "Model B", "active", nil),
		makeSearchNode(model.LineageEntityPipeline, "Pipeline C", "running", nil),
	}
	results := newTestSearcher().Search(nodes, "", "", 100)
	if len(results) != 3 {
		t.Fatalf("empty query should return all nodes, got %d", len(results))
	}
}

func TestSearch_EmptyQuery_WhitespaceOnly_ReturnsAll(t *testing.T) {
	nodes := []model.LineageNode{
		makeSearchNode(model.LineageEntityDataSource, "Source A", "active", nil),
		makeSearchNode(model.LineageEntityDataSource, "Source B", "active", nil),
	}
	results := newTestSearcher().Search(nodes, "   ", "", 100)
	if len(results) != 2 {
		t.Fatalf("whitespace-only query should be treated as empty, got %d", len(results))
	}
}

func TestSearch_EmptyQuery_ScoreIsZero(t *testing.T) {
	nodes := []model.LineageNode{
		makeSearchNode(model.LineageEntityDataSource, "Source A", "active", nil),
	}
	results := newTestSearcher().Search(nodes, "", "", 10)
	if results[0].Score != 0 {
		t.Fatalf("empty query should yield score=0, got %v", results[0].Score)
	}
	if len(results[0].MatchFields) != 0 {
		t.Fatalf("empty query should yield no match fields, got %v", results[0].MatchFields)
	}
}

// ─── result shape ─────────────────────────────────────────────────────────────

func TestSearch_ResultContainsNode(t *testing.T) {
	node := makeSearchNode(model.LineageEntityDataModel, "payroll", "active", nil)
	results := newTestSearcher().Search([]model.LineageNode{node}, "payroll", "", 10)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Node.ID != node.ID {
		t.Fatalf("result node ID mismatch: got %q, want %q", results[0].Node.ID, node.ID)
	}
	if results[0].Score <= 0 {
		t.Fatalf("expected positive score, got %v", results[0].Score)
	}
	if len(results[0].MatchFields) == 0 {
		t.Fatal("expected non-empty MatchFields for a matching query")
	}
}

// ─── type filtering ───────────────────────────────────────────────────────────

func TestSearch_TypeFilter_ReturnsOnlyMatchingType(t *testing.T) {
	nodes := []model.LineageNode{
		makeSearchNode(model.LineageEntityDataSource, "Source A", "active", nil),
		makeSearchNode(model.LineageEntityDataSource, "Source B", "active", nil),
		makeSearchNode(model.LineageEntityDataModel, "Model A", "active", nil),
	}
	results := newTestSearcher().Search(nodes, "", string(model.LineageEntityDataSource), 100)
	if len(results) != 2 {
		t.Fatalf("type filter should return 2 data_source nodes, got %d", len(results))
	}
	for _, r := range results {
		if r.Node.Type != string(model.LineageEntityDataSource) {
			t.Fatalf("unexpected type %q in type-filtered results", r.Node.Type)
		}
	}
}

func TestSearch_TypeFilter_NoMatch_ReturnsEmpty(t *testing.T) {
	nodes := []model.LineageNode{
		makeSearchNode(model.LineageEntityDataSource, "Source A", "active", nil),
	}
	results := newTestSearcher().Search(nodes, "", string(model.LineageEntityPipeline), 100)
	if len(results) != 0 {
		t.Fatalf("expected 0 results when type does not match any node, got %d", len(results))
	}
}

func TestSearch_TypeFilter_CaseInsensitive(t *testing.T) {
	nodes := []model.LineageNode{
		makeSearchNode(model.LineageEntityDataSource, "Source A", "active", nil),
	}
	// Type stored as lowercase; querying with upper should still match.
	results := newTestSearcher().Search(nodes, "", "DATA_SOURCE", 100)
	if len(results) != 1 {
		t.Fatalf("type filter should be case-insensitive, got %d", len(results))
	}
}

// ─── relevance ranking ────────────────────────────────────────────────────────

func TestSearch_ExactNameRanksFirst(t *testing.T) {
	// "payroll" (exact) should rank above "payroll_data" (prefix) and
	// "department_payroll" (suffix / contains).
	nodes := []model.LineageNode{
		makeSearchNode(model.LineageEntityDataModel, "department_payroll", "active", nil),
		makeSearchNode(model.LineageEntityDataModel, "payroll", "active", nil),
		makeSearchNode(model.LineageEntityDataModel, "payroll_data", "active", nil),
	}
	results := newTestSearcher().Search(nodes, "payroll", "", 10)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Node.Name != "payroll" {
		t.Fatalf("exact match should be ranked first, got %q", results[0].Node.Name)
	}
	// Scores must be strictly descending.
	if results[0].Score <= results[1].Score {
		t.Fatalf("score should decrease: [0]=%v [1]=%v", results[0].Score, results[1].Score)
	}
}

func TestSearch_PrefixRanksAboveContains(t *testing.T) {
	// "employees" starts with "employ" → prefix score (80).
	// "senior_employee" contains "employ" → substring score (60).
	nodes := []model.LineageNode{
		makeSearchNode(model.LineageEntityDataModel, "senior_employee", "active", nil),
		makeSearchNode(model.LineageEntityDataModel, "employees", "active", nil),
	}
	results := newTestSearcher().Search(nodes, "employ", "", 10)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Node.Name != "employees" {
		t.Fatalf("prefix match should rank above substring match, got %q", results[0].Node.Name)
	}
}

func TestSearch_MultipleFieldMatchBoostsScore(t *testing.T) {
	// nodeA: name contains "hr" (60 pts).
	// nodeB: name contains "hr" (60 pts) + metadata value contains "hr" (20 pts) = 80 pts total.
	// nodeB should rank first.
	nodeA := makeSearchNode(model.LineageEntityDataModel, "hr_payroll", "active", nil)
	nodeB := makeSearchNode(model.LineageEntityDataModel, "hr_contracts", "active", map[string]any{
		"description": "manages hr data for all employees",
	})
	results := newTestSearcher().Search([]model.LineageNode{nodeA, nodeB}, "hr", "", 10)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Node.ID != nodeB.ID {
		t.Fatalf("node with more matching fields should rank first, got %q", results[0].Node.Name)
	}
}

// ─── limit ────────────────────────────────────────────────────────────────────

func TestSearch_LimitRespected(t *testing.T) {
	nodes := make([]model.LineageNode, 20)
	for i := range nodes {
		nodes[i] = makeSearchNode(model.LineageEntityDataSource, "source", "active", nil)
	}
	results := newTestSearcher().Search(nodes, "source", "", 5)
	if len(results) != 5 {
		t.Fatalf("expected limit 5, got %d", len(results))
	}
}

func TestSearch_DefaultLimit_ZeroFallsBackTo25(t *testing.T) {
	nodes := make([]model.LineageNode, 50)
	for i := range nodes {
		nodes[i] = makeSearchNode(model.LineageEntityDataSource, "src", "active", nil)
	}
	results := newTestSearcher().Search(nodes, "src", "", 0)
	if len(results) != 25 {
		t.Fatalf("limit=0 should fall back to default 25, got %d", len(results))
	}
}

func TestSearch_DefaultLimit_NegativeFallsBackTo25(t *testing.T) {
	nodes := make([]model.LineageNode, 50)
	for i := range nodes {
		nodes[i] = makeSearchNode(model.LineageEntityDataSource, "src", "active", nil)
	}
	results := newTestSearcher().Search(nodes, "src", "", -1)
	if len(results) != 25 {
		t.Fatalf("limit=-1 should fall back to default 25, got %d", len(results))
	}
}

// ─── metadata search ──────────────────────────────────────────────────────────

func TestSearch_MetadataStringValueMatch(t *testing.T) {
	nodes := []model.LineageNode{
		makeSearchNode(model.LineageEntityDataModel, "SalesModel", "active", map[string]any{
			"data_classification": "restricted",
		}),
		makeSearchNode(model.LineageEntityDataModel, "PublicModel", "active", map[string]any{
			"data_classification": "public",
		}),
	}
	results := newTestSearcher().Search(nodes, "restricted", "", 10)
	if len(results) != 1 {
		t.Fatalf("expected 1 result matching metadata string value, got %d", len(results))
	}
	if results[0].Node.Name != "SalesModel" {
		t.Fatalf("expected SalesModel, got %q", results[0].Node.Name)
	}
	if !containsSearchField(results[0].MatchFields, "metadata.data_classification") {
		t.Fatalf("expected metadata field in MatchFields, got %v", results[0].MatchFields)
	}
}

func TestSearch_MetadataKeyMatch(t *testing.T) {
	nodes := []model.LineageNode{
		makeSearchNode(model.LineageEntityPipeline, "Pipeline A", "active", map[string]any{
			"pipeline_type": "etl",
		}),
		makeSearchNode(model.LineageEntityPipeline, "Pipeline B", "active", nil),
	}
	results := newTestSearcher().Search(nodes, "pipeline_type", "", 10)
	if len(results) != 1 {
		t.Fatalf("expected 1 result matching metadata key name, got %d", len(results))
	}
}

func TestSearch_MetadataBoolTrue(t *testing.T) {
	nodes := []model.LineageNode{
		makeSearchNode(model.LineageEntityDataModel, "PIIModel", "active", map[string]any{
			"contains_pii": true,
		}),
		makeSearchNode(model.LineageEntityDataModel, "SafeModel", "active", map[string]any{
			"contains_pii": false,
		}),
	}
	results := newTestSearcher().Search(nodes, "true", "", 10)
	if len(results) != 1 {
		t.Fatalf("expected 1 result matching bool metadata value 'true', got %d", len(results))
	}
	if results[0].Node.Name != "PIIModel" {
		t.Fatalf("expected PIIModel, got %q", results[0].Node.Name)
	}
}

func TestSearch_MetadataBoolFalse(t *testing.T) {
	nodes := []model.LineageNode{
		makeSearchNode(model.LineageEntityDataModel, "PIIModel", "active", map[string]any{
			"contains_pii": true,
		}),
		makeSearchNode(model.LineageEntityDataModel, "SafeModel", "active", map[string]any{
			"contains_pii": false,
		}),
	}
	results := newTestSearcher().Search(nodes, "false", "", 10)
	if len(results) != 1 {
		t.Fatalf("expected 1 result matching bool metadata value 'false', got %d", len(results))
	}
	if results[0].Node.Name != "SafeModel" {
		t.Fatalf("expected SafeModel, got %q", results[0].Node.Name)
	}
}

// ─── status search ────────────────────────────────────────────────────────────

func TestSearch_StatusMatch(t *testing.T) {
	nodes := []model.LineageNode{
		makeSearchNode(model.LineageEntityDataSource, "Source X", "error", nil),
		makeSearchNode(model.LineageEntityDataSource, "Source Y", "active", nil),
	}
	results := newTestSearcher().Search(nodes, "error", "", 10)
	if len(results) != 1 {
		t.Fatalf("expected 1 result matching status field, got %d", len(results))
	}
	if results[0].Node.Status != "error" {
		t.Fatalf("expected status=error, got %q", results[0].Node.Status)
	}
	if !containsSearchField(results[0].MatchFields, "status") {
		t.Fatalf("expected 'status' in MatchFields, got %v", results[0].MatchFields)
	}
}

func TestSearch_StatusMatch_CaseInsensitive(t *testing.T) {
	nodes := []model.LineageNode{
		makeSearchNode(model.LineageEntityDataSource, "Source X", "ERROR", nil),
	}
	results := newTestSearcher().Search(nodes, "error", "", 10)
	if len(results) != 1 {
		t.Fatalf("status match should be case-insensitive, got %d", len(results))
	}
}

// ─── combined type + query ────────────────────────────────────────────────────

func TestSearch_TypeAndQueryCombined(t *testing.T) {
	nodes := []model.LineageNode{
		makeSearchNode(model.LineageEntityDataSource, "hr_source", "active", nil),
		makeSearchNode(model.LineageEntityDataModel, "hr_model", "active", nil),
		makeSearchNode(model.LineageEntityDataSource, "finance_source", "active", nil),
	}
	results := newTestSearcher().Search(nodes, "hr", string(model.LineageEntityDataSource), 10)
	if len(results) != 1 {
		t.Fatalf("expected 1 result for type=data_source + query=hr, got %d", len(results))
	}
	if results[0].Node.Name != "hr_source" {
		t.Fatalf("expected hr_source, got %q", results[0].Node.Name)
	}
}

// ─── case-insensitive name matching ──────────────────────────────────────────

func TestSearch_CaseInsensitive_Name(t *testing.T) {
	nodes := []model.LineageNode{
		makeSearchNode(model.LineageEntityDataModel, "PAYROLL_DATA", "active", nil),
	}
	for _, q := range []string{"payroll", "PAYROLL", "Payroll", "pAyRoLl"} {
		results := newTestSearcher().Search(nodes, q, "", 10)
		if len(results) != 1 {
			t.Fatalf("case-insensitive search for %q: expected 1 result, got %d", q, len(results))
		}
	}
}

// ─── no match ─────────────────────────────────────────────────────────────────

func TestSearch_NoMatch_ReturnsEmpty(t *testing.T) {
	nodes := []model.LineageNode{
		makeSearchNode(model.LineageEntityDataSource, "alpha", "active", nil),
	}
	results := newTestSearcher().Search(nodes, "zzznomatch", "", 10)
	if len(results) != 0 {
		t.Fatalf("expected 0 results for non-matching query, got %d", len(results))
	}
}

// ─── deterministic ordering ───────────────────────────────────────────────────

func TestSearch_DeterministicTieBreak(t *testing.T) {
	// All three nodes have the same name → same score; results must be
	// in the same order (by node ID) across multiple calls.
	n1 := makeSearchNode(model.LineageEntityDataSource, "identical", "active", nil)
	n2 := makeSearchNode(model.LineageEntityDataSource, "identical", "active", nil)
	n3 := makeSearchNode(model.LineageEntityDataSource, "identical", "active", nil)
	nodes := []model.LineageNode{n1, n2, n3}

	r1 := newTestSearcher().Search(nodes, "identical", "", 10)
	r2 := newTestSearcher().Search(nodes, "identical", "", 10)
	if len(r1) != len(r2) {
		t.Fatalf("result length not stable: %d vs %d", len(r1), len(r2))
	}
	for i := range r1 {
		if r1[i].Node.ID != r2[i].Node.ID {
			t.Fatalf("results not deterministic at index %d: %q vs %q", i, r1[i].Node.ID, r2[i].Node.ID)
		}
	}
}

// ─── scoreNode unit tests ─────────────────────────────────────────────────────

func TestScoreNode_ExactName_Score100(t *testing.T) {
	s := newTestSearcher()
	node := makeSearchNode(model.LineageEntityDataModel, "payroll", "active", nil)
	score, fields := s.scoreNode(node, "payroll")
	if score < 100 {
		t.Fatalf("exact name match should score ≥100, got %v", score)
	}
	if !containsSearchField(fields, "name") {
		t.Fatalf("expected 'name' in matched fields, got %v", fields)
	}
}

func TestScoreNode_PrefixName_Score80(t *testing.T) {
	s := newTestSearcher()
	node := makeSearchNode(model.LineageEntityDataModel, "employees", "active", nil)
	score, fields := s.scoreNode(node, "employ")
	if score < 80 || score >= 100 {
		t.Fatalf("prefix name match should score in [80,100), got %v", score)
	}
	if !containsSearchField(fields, "name") {
		t.Fatalf("expected 'name' in matched fields, got %v", fields)
	}
}

func TestScoreNode_ContainsName_Score60(t *testing.T) {
	s := newTestSearcher()
	node := makeSearchNode(model.LineageEntityDataModel, "department_payroll_2024", "active", nil)
	score, fields := s.scoreNode(node, "payroll")
	// "payroll" is not an exact or prefix match — substring only.
	if score < 60 || score >= 80 {
		t.Fatalf("substring name match should score in [60,80), got %v", score)
	}
	if !containsSearchField(fields, "name") {
		t.Fatalf("expected 'name' in matched fields, got %v", fields)
	}
}

func TestScoreNode_IDContains_Score40(t *testing.T) {
	s := newTestSearcher()
	id := uuid.MustParse("00000000-0000-0000-0000-000000001234")
	node := model.LineageNode{
		ID:       nodeKey(model.LineageEntityDataSource, id),
		EntityID: id,
		Type:     string(model.LineageEntityDataSource),
		Name:     "nomatchname",
	}
	score, fields := s.scoreNode(node, "000000001234")
	if score < 40 {
		t.Fatalf("ID match should contribute ≥40 to score, got %v", score)
	}
	if !containsSearchField(fields, "id") {
		t.Fatalf("expected 'id' in matched fields, got %v", fields)
	}
}

func TestScoreNode_MetadataStringValue_Score20(t *testing.T) {
	s := newTestSearcher()
	node := makeSearchNode(model.LineageEntityDataModel, "nomatch", "active", map[string]any{
		"data_classification": "restricted",
	})
	score, fields := s.scoreNode(node, "restricted")
	if score < 20 {
		t.Fatalf("metadata value match should contribute ≥20 to score, got %v", score)
	}
	if !containsSearchField(fields, "metadata.data_classification") {
		t.Fatalf("expected 'metadata.data_classification' in matched fields, got %v", fields)
	}
}

func TestScoreNode_MetadataKey_Score10(t *testing.T) {
	s := newTestSearcher()
	node := makeSearchNode(model.LineageEntityPipeline, "nomatch", "active", map[string]any{
		"pipeline_type": "unknown_value",
	})
	score, fields := s.scoreNode(node, "pipeline_type")
	if score < 10 {
		t.Fatalf("metadata key match should contribute ≥10 to score, got %v", score)
	}
	if !containsSearchField(fields, "metadata.pipeline_type") {
		t.Fatalf("expected 'metadata.pipeline_type' in matched fields, got %v", fields)
	}
}

func TestScoreNode_BooleanTrueMatch_Score10(t *testing.T) {
	s := newTestSearcher()
	node := makeSearchNode(model.LineageEntityDataModel, "nomatch", "active", map[string]any{
		"contains_pii": true,
	})
	score, _ := s.scoreNode(node, "true")
	if score < 10 {
		t.Fatalf("boolean metadata value match should contribute ≥10 to score, got %v", score)
	}
}

func TestScoreNode_ZeroScore_OnNoMatch(t *testing.T) {
	s := newTestSearcher()
	node := makeSearchNode(model.LineageEntityDataSource, "alpha", "active", nil)
	score, fields := s.scoreNode(node, "zzznomatch")
	if score != 0 {
		t.Fatalf("expected score 0 for no match, got %v", score)
	}
	if len(fields) != 0 {
		t.Fatalf("expected no matched fields, got %v", fields)
	}
}

func TestScoreNode_NilMetadata_DoesNotPanic(t *testing.T) {
	s := newTestSearcher()
	node := model.LineageNode{
		ID:       nodeKey(model.LineageEntityDataSource, uuid.New()),
		Type:     string(model.LineageEntityDataSource),
		Name:     "source",
		Metadata: nil,
	}
	score, _ := s.scoreNode(node, "source")
	if score <= 0 {
		t.Fatalf("nil metadata node with matching name should still score > 0, got %v", score)
	}
}

// ─── searchAppendUnique unit tests ────────────────────────────────────────────

func TestSearchAppendUnique_NoDuplicates(t *testing.T) {
	result := searchAppendUnique([]string{"name", "id"}, "name")
	if len(result) != 2 {
		t.Fatalf("duplicate entry should not be appended, got %v", result)
	}
}

func TestSearchAppendUnique_AppendsNew(t *testing.T) {
	result := searchAppendUnique([]string{"name"}, "status")
	if len(result) != 2 || result[1] != "status" {
		t.Fatalf("new entry should be appended, got %v", result)
	}
}

func TestSearchAppendUnique_EmptySlice(t *testing.T) {
	result := searchAppendUnique(nil, "name")
	if len(result) != 1 || result[0] != "name" {
		t.Fatalf("appending to nil slice should work, got %v", result)
	}
}
