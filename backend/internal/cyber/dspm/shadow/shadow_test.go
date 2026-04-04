package shadow

import (
	"encoding/json"
	"testing"
)

// --- Fingerprint ---

func TestComputeFingerprint_Deterministic(t *testing.T) {
	cols := []ColumnDef{
		{Name: "id", Type: "int"},
		{Name: "name", Type: "varchar"},
		{Name: "email", Type: "varchar"},
	}

	h1 := ComputeFingerprint(cols)
	h2 := ComputeFingerprint(cols)

	if h1 != h2 {
		t.Fatal("fingerprint not deterministic")
	}
	if h1 == "" {
		t.Fatal("fingerprint should not be empty")
	}
}

func TestComputeFingerprint_OrderIndependent(t *testing.T) {
	cols1 := []ColumnDef{
		{Name: "id", Type: "int"},
		{Name: "name", Type: "varchar"},
		{Name: "email", Type: "varchar"},
	}
	cols2 := []ColumnDef{
		{Name: "email", Type: "varchar"},
		{Name: "id", Type: "int"},
		{Name: "name", Type: "varchar"},
	}

	if ComputeFingerprint(cols1) != ComputeFingerprint(cols2) {
		t.Fatal("fingerprint should be order-independent")
	}
}

func TestComputeFingerprint_CaseInsensitive(t *testing.T) {
	cols1 := []ColumnDef{{Name: "Email", Type: "VARCHAR"}}
	cols2 := []ColumnDef{{Name: "email", Type: "varchar"}}

	if ComputeFingerprint(cols1) != ComputeFingerprint(cols2) {
		t.Fatal("fingerprint should be case-insensitive")
	}
}

func TestComputeFingerprint_Empty_ReturnsEmpty(t *testing.T) {
	if h := ComputeFingerprint(nil); h != "" {
		t.Fatalf("expected empty fingerprint, got %q", h)
	}
}

func TestComputeFingerprint_DifferentTypes_DifferentHash(t *testing.T) {
	cols1 := []ColumnDef{{Name: "id", Type: "int"}}
	cols2 := []ColumnDef{{Name: "id", Type: "bigint"}}

	if ComputeFingerprint(cols1) == ComputeFingerprint(cols2) {
		t.Fatal("different column types should produce different hashes")
	}
}

// --- ExtractColumnsFromSchema ---

func TestExtractColumnsFromSchema_ValidSchema(t *testing.T) {
	schema := map[string]interface{}{
		"tables": []interface{}{
			map[string]interface{}{
				"name": "users",
				"columns": []interface{}{
					map[string]interface{}{"name": "id", "data_type": "integer"},
					map[string]interface{}{"name": "email", "data_type": "varchar"},
				},
			},
		},
	}

	results := ExtractColumnsFromSchema(schema)
	if len(results) != 1 {
		t.Fatalf("expected 1 table fingerprint, got %d", len(results))
	}
	if results[0].TableName != "users" {
		t.Errorf("table name = %q, want 'users'", results[0].TableName)
	}
	if len(results[0].Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(results[0].Columns))
	}
	if results[0].Hash == "" {
		t.Error("hash should not be empty")
	}
}

func TestExtractColumnsFromSchema_TypeFallback(t *testing.T) {
	schema := map[string]interface{}{
		"tables": []interface{}{
			map[string]interface{}{
				"name": "orders",
				"columns": []interface{}{
					map[string]interface{}{"name": "id", "type": "int"},
				},
			},
		},
	}

	results := ExtractColumnsFromSchema(schema)
	if len(results) != 1 {
		t.Fatalf("expected 1 table, got %d", len(results))
	}
	if results[0].Columns[0].Type != "int" {
		t.Errorf("column type = %q, want 'int' (fallback to 'type' key)", results[0].Columns[0].Type)
	}
}

func TestExtractColumnsFromSchema_StringColumns(t *testing.T) {
	schema := map[string]interface{}{
		"tables": []interface{}{
			map[string]interface{}{
				"name":    "products",
				"columns": []interface{}{"id", "name", "price"},
			},
		},
	}

	results := ExtractColumnsFromSchema(schema)
	if len(results) != 1 {
		t.Fatalf("expected 1 table, got %d", len(results))
	}
	if len(results[0].Columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(results[0].Columns))
	}
}

func TestExtractColumnsFromSchema_EmptySchema(t *testing.T) {
	results := ExtractColumnsFromSchema(map[string]interface{}{})
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty schema, got %d", len(results))
	}
}

func TestExtractColumnsFromJSON_Valid(t *testing.T) {
	raw := json.RawMessage(`{
		"tables": [
			{
				"name": "accounts",
				"columns": [
					{"name": "id", "data_type": "uuid"},
					{"name": "balance", "data_type": "decimal"}
				]
			}
		]
	}`)

	results := ExtractColumnsFromJSON(raw)
	if len(results) != 1 {
		t.Fatalf("expected 1 table, got %d", len(results))
	}
	if results[0].TableName != "accounts" {
		t.Errorf("table name = %q, want 'accounts'", results[0].TableName)
	}
}

func TestExtractColumnsFromJSON_Empty(t *testing.T) {
	results := ExtractColumnsFromJSON(nil)
	if results != nil {
		t.Errorf("expected nil for empty JSON, got %d", len(results))
	}
}

func TestExtractColumnsFromJSON_InvalidJSON(t *testing.T) {
	results := ExtractColumnsFromJSON(json.RawMessage(`{invalid`))
	if results != nil {
		t.Errorf("expected nil for invalid JSON, got %d", len(results))
	}
}

// --- Comparator ---

func TestCompareFingerprints_ExactMatch(t *testing.T) {
	cols := []ColumnDef{
		{Name: "id", Type: "int"},
		{Name: "email", Type: "varchar"},
	}
	hash := ComputeFingerprint(cols)

	source := []TableFingerprint{
		{SourceID: "src1", TableName: "users", Hash: hash, Columns: cols},
	}
	target := []TableFingerprint{
		{SourceID: "src2", TableName: "users_backup", Hash: hash, Columns: cols},
	}

	matches := CompareFingerprints(source, target, 0.8)
	if len(matches) != 1 {
		t.Fatalf("expected 1 exact match, got %d", len(matches))
	}
	if matches[0].MatchType != "exact" {
		t.Errorf("match type = %q, want 'exact'", matches[0].MatchType)
	}
	if matches[0].Similarity != 1.0 {
		t.Errorf("similarity = %f, want 1.0", matches[0].Similarity)
	}
}

func TestCompareFingerprints_SkipsSameSource(t *testing.T) {
	cols := []ColumnDef{{Name: "id", Type: "int"}}
	hash := ComputeFingerprint(cols)

	source := []TableFingerprint{
		{SourceID: "src1", TableName: "t1", Hash: hash, Columns: cols},
	}
	target := []TableFingerprint{
		{SourceID: "src1", TableName: "t2", Hash: hash, Columns: cols},
	}

	matches := CompareFingerprints(source, target, 0.8)
	if len(matches) != 0 {
		t.Errorf("expected 0 matches for same source, got %d", len(matches))
	}
}

func TestCompareFingerprints_StructuralMatch(t *testing.T) {
	source := []TableFingerprint{
		{SourceID: "src1", TableName: "users", Columns: []ColumnDef{
			{Name: "id", Type: "int"},
			{Name: "email", Type: "varchar"},
			{Name: "name", Type: "varchar"},
		}},
	}
	target := []TableFingerprint{
		{SourceID: "src2", TableName: "customers", Columns: []ColumnDef{
			{Name: "id", Type: "bigint"},
			{Name: "email", Type: "text"},
			{Name: "name", Type: "text"},
		}},
	}

	matches := CompareFingerprints(source, target, 0.5)
	if len(matches) == 0 {
		t.Fatal("expected structural match (same column names)")
	}
	if matches[0].MatchType != "structural" {
		t.Errorf("match type = %q, want 'structural'", matches[0].MatchType)
	}
	if matches[0].Similarity < 0.5 {
		t.Errorf("similarity %f below threshold", matches[0].Similarity)
	}
}

func TestCompareFingerprints_NameSimilar(t *testing.T) {
	cols := []ColumnDef{{Name: "id", Type: "int"}, {Name: "val", Type: "text"}}

	source := []TableFingerprint{
		{SourceID: "src1", TableName: "users", Columns: cols},
	}
	target := []TableFingerprint{
		{SourceID: "src2", TableName: "usersbackup", Columns: cols},
	}

	// Use low threshold since name similarity may be moderate
	matches := CompareFingerprints(source, target, 0.5)
	if len(matches) == 0 {
		t.Fatal("expected at least one match (exact hash + name similarity)")
	}
}

func TestCompareFingerprints_NoMatch(t *testing.T) {
	source := []TableFingerprint{
		{SourceID: "src1", TableName: "users", Columns: []ColumnDef{
			{Name: "id", Type: "int"},
			{Name: "email", Type: "varchar"},
		}},
	}
	target := []TableFingerprint{
		{SourceID: "src2", TableName: "products", Columns: []ColumnDef{
			{Name: "sku", Type: "varchar"},
			{Name: "price", Type: "decimal"},
		}},
	}

	matches := CompareFingerprints(source, target, 0.8)
	if len(matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(matches))
	}
}

func TestCompareFingerprints_SortedBySimilarity(t *testing.T) {
	cols := []ColumnDef{
		{Name: "id", Type: "int"},
		{Name: "email", Type: "varchar"},
		{Name: "name", Type: "varchar"},
	}
	hash := ComputeFingerprint(cols)

	source := []TableFingerprint{
		{SourceID: "src1", TableName: "users", Hash: hash, Columns: cols},
	}
	target := []TableFingerprint{
		{SourceID: "src2", TableName: "users_bak", Hash: hash, Columns: cols},
		{SourceID: "src3", TableName: "people", Columns: []ColumnDef{
			{Name: "id", Type: "int"},
			{Name: "email", Type: "text"},
			{Name: "name", Type: "text"},
		}},
	}

	matches := CompareFingerprints(source, target, 0.5)
	if len(matches) < 2 {
		t.Fatalf("expected at least 2 matches, got %d", len(matches))
	}
	for i := 1; i < len(matches); i++ {
		if matches[i].Similarity > matches[i-1].Similarity {
			t.Errorf("matches not sorted: [%d].Similarity=%f > [%d].Similarity=%f",
				i, matches[i].Similarity, i-1, matches[i-1].Similarity)
		}
	}
}

// --- columnSimilarity ---

func TestColumnSimilarity_Identical(t *testing.T) {
	cols := []ColumnDef{{Name: "id"}, {Name: "email"}, {Name: "name"}}
	if s := columnSimilarity(cols, cols); s != 1.0 {
		t.Errorf("identical columns should have similarity 1.0, got %f", s)
	}
}

func TestColumnSimilarity_Disjoint(t *testing.T) {
	a := []ColumnDef{{Name: "id"}, {Name: "email"}}
	b := []ColumnDef{{Name: "sku"}, {Name: "price"}}
	if s := columnSimilarity(a, b); s != 0.0 {
		t.Errorf("disjoint columns should have similarity 0.0, got %f", s)
	}
}

func TestColumnSimilarity_Partial(t *testing.T) {
	a := []ColumnDef{{Name: "id"}, {Name: "email"}, {Name: "name"}}
	b := []ColumnDef{{Name: "id"}, {Name: "email"}, {Name: "phone"}}
	// Intersection: {id, email} = 2. Union: {id, email, name, phone} = 4. Jaccard = 0.5
	s := columnSimilarity(a, b)
	if s != 0.5 {
		t.Errorf("expected Jaccard similarity 0.5, got %f", s)
	}
}

// --- editDistance ---

func TestEditDistance_Identical(t *testing.T) {
	if d := editDistance("hello", "hello"); d != 0 {
		t.Errorf("expected 0, got %d", d)
	}
}

func TestEditDistance_Empty(t *testing.T) {
	if d := editDistance("", "hello"); d != 5 {
		t.Errorf("expected 5, got %d", d)
	}
	if d := editDistance("hello", ""); d != 5 {
		t.Errorf("expected 5, got %d", d)
	}
}

func TestEditDistance_SingleChar(t *testing.T) {
	if d := editDistance("cat", "car"); d != 1 {
		t.Errorf("expected 1, got %d", d)
	}
}

func TestEditDistance_Kitten_Sitting(t *testing.T) {
	if d := editDistance("kitten", "sitting"); d != 3 {
		t.Errorf("expected 3, got %d", d)
	}
}

// --- nameEditDistanceSimilarity ---

func TestNameEditDistanceSimilarity_Identical(t *testing.T) {
	if s := nameEditDistanceSimilarity("users", "users"); s != 1.0 {
		t.Errorf("expected 1.0, got %f", s)
	}
}

func TestNameEditDistanceSimilarity_CaseInsensitive(t *testing.T) {
	if s := nameEditDistanceSimilarity("Users", "users"); s != 1.0 {
		t.Errorf("expected 1.0 for case-insensitive match, got %f", s)
	}
}

func TestNameEditDistanceSimilarity_Similar(t *testing.T) {
	s := nameEditDistanceSimilarity("users", "user")
	if s < 0.7 {
		t.Errorf("'users' vs 'user' should be >0.7 similar, got %f", s)
	}
}

// --- nameSimilar ---

func TestNameSimilar_Identical(t *testing.T) {
	if !nameSimilar("users", "users") {
		t.Error("identical names should be similar")
	}
}

func TestNameSimilar_CloseEdit(t *testing.T) {
	if !nameSimilar("users", "user") {
		t.Error("'users' and 'user' should be similar (edit distance 1)")
	}
}

func TestNameSimilar_Different(t *testing.T) {
	if nameSimilar("users", "products") {
		t.Error("'users' and 'products' should not be similar")
	}
}
