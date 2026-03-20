package database

import (
	"strings"
	"testing"
)

func TestQB_SimpleWhere(t *testing.T) {
	qb := NewQueryBuilder("SELECT a.* FROM assets a")
	sql, args := qb.Where("a.type = ?", "server").Build()

	if !strings.Contains(sql, "WHERE a.type = $1") {
		t.Fatalf("expected WHERE clause, got %s", sql)
	}
	if len(args) != 1 || args[0] != "server" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestQB_MultipleWhereAndParameterIndexing(t *testing.T) {
	qb := NewQueryBuilder("SELECT a.* FROM assets a")
	sql, args := qb.
		Where("a.type = ?", "server").
		Where("a.status = ?", "active").
		Where("a.criticality = ?", "high").
		Build()

	expected := "WHERE a.type = $1 AND a.status = $2 AND a.criticality = $3"
	if !strings.Contains(sql, expected) {
		t.Fatalf("expected %q in sql, got %s", expected, sql)
	}
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %#v", args)
	}
}

func TestQB_WhereIf(t *testing.T) {
	qb := NewQueryBuilder("SELECT a.* FROM assets a")
	qb.WhereIf(false, "a.status = ?", "active")
	sql, args := qb.Build()
	if strings.Contains(sql, "status") || len(args) != 0 {
		t.Fatalf("expected clause to be skipped, got sql=%s args=%#v", sql, args)
	}

	qb = NewQueryBuilder("SELECT a.* FROM assets a")
	sql, args = qb.WhereIf(true, "a.status = ?", "active").Build()
	if !strings.Contains(sql, "a.status = $1") || len(args) != 1 {
		t.Fatalf("expected conditional clause, got sql=%s args=%#v", sql, args)
	}
}

func TestQB_WhereIn(t *testing.T) {
	qb := NewQueryBuilder("SELECT a.* FROM assets a")
	sql, args := qb.WhereIn("a.type", []string{"server", "endpoint", "database"}).Build()
	if !strings.Contains(sql, "a.type IN ($1, $2, $3)") {
		t.Fatalf("unexpected sql: %s", sql)
	}
	if len(args) != 3 {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestQB_WhereIn_Empty(t *testing.T) {
	qb := NewQueryBuilder("SELECT a.* FROM assets a")
	sql, args := qb.WhereIn("a.type", nil).Build()
	if strings.Contains(sql, "IN ()") || len(args) != 0 {
		t.Fatalf("unexpected sql or args: %s %#v", sql, args)
	}
}

func TestQB_WhereExistsAndArrayConditions(t *testing.T) {
	qb := NewQueryBuilder("SELECT a.* FROM assets a")
	sql, args := qb.
		WhereExists("SELECT 1 FROM vulnerabilities v WHERE v.asset_id = a.id AND v.severity = ?", "critical").
		WhereArrayContains("a.tags", "production").
		WhereArrayContainsAll("a.tags", []string{"production", "web"}).
		Build()

	if !strings.Contains(sql, "EXISTS (SELECT 1 FROM vulnerabilities v WHERE v.asset_id = a.id AND v.severity = $1)") {
		t.Fatalf("unexpected EXISTS sql: %s", sql)
	}
	if !strings.Contains(sql, "$2 = ANY(a.tags)") {
		t.Fatalf("unexpected ANY sql: %s", sql)
	}
	if !strings.Contains(sql, "a.tags @> ARRAY[$3, $4]") {
		t.Fatalf("unexpected array contains sql: %s", sql)
	}
	if len(args) != 4 {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestQB_WhereFTS(t *testing.T) {
	qb := NewQueryBuilder("SELECT a.* FROM assets a")
	sql, args := qb.WhereFTS([]string{"a.name", "a.hostname", "a.owner"}, "web-prod").Build()
	if !strings.Contains(sql, "plainto_tsquery('english', $1)") {
		t.Fatalf("unexpected fts sql: %s", sql)
	}
	if !strings.Contains(sql, "host(ip_address) = $2") {
		t.Fatalf("expected exact ip fallback: %s", sql)
	}
	if !strings.Contains(sql, "$3 = ANY(tags)") {
		t.Fatalf("expected tag fallback: %s", sql)
	}
	if len(args) != 3 {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestQB_OrderByAndPaginate(t *testing.T) {
	qb := NewQueryBuilder("SELECT a.* FROM assets a")
	sql, _ := qb.OrderBy("criticality", "desc", []string{"criticality", "created_at"}).Paginate(3, 25).Build()
	if !strings.Contains(sql, "ORDER BY severity_order(a.criticality::text) DESC") {
		t.Fatalf("unexpected order sql: %s", sql)
	}
	if !strings.Contains(sql, "LIMIT 25 OFFSET 50") {
		t.Fatalf("unexpected pagination sql: %s", sql)
	}
}

func TestQB_OrderBy_SeverityCastsEnumLikeColumns(t *testing.T) {
	qb := NewQueryBuilder("SELECT a.* FROM vulnerabilities a")
	sql, _ := qb.OrderBy("severity", "desc", []string{"severity", "created_at"}).Build()
	if !strings.Contains(sql, "ORDER BY severity_order(a.severity::text) DESC") {
		t.Fatalf("unexpected severity order sql: %s", sql)
	}
}

func TestQB_OrderBy_InvalidColumnIgnored(t *testing.T) {
	qb := NewQueryBuilder("SELECT a.* FROM assets a")
	sql, _ := qb.OrderBy("drop table assets", "desc", []string{"criticality"}).Build()
	if strings.Contains(strings.ToLower(sql), "order by") {
		t.Fatalf("unexpected order by clause for invalid column: %s", sql)
	}
}

func TestQB_BuildCount(t *testing.T) {
	qb := NewQueryBuilder("SELECT a.id FROM assets a")
	sql, args := qb.Where("a.tenant_id = ?", "tenant-1").BuildCount()
	if !strings.HasPrefix(sql, "SELECT COUNT(DISTINCT a.id)") {
		t.Fatalf("unexpected count sql: %s", sql)
	}
	if len(args) != 1 {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestQB_BuildCount_WithHavingWrapsQuery(t *testing.T) {
	qb := NewQueryBuilder("SELECT a.id FROM assets a")
	sql, _ := qb.Where("a.tenant_id = ?", "tenant-1").Having("COUNT(*) > ?", 1).BuildCount()
	if !strings.HasPrefix(sql, "SELECT COUNT(*) FROM (") {
		t.Fatalf("expected wrapped count query, got %s", sql)
	}
}

func TestQB_BuildCount_WithNestedFromInSelectUsesTopLevelFrom(t *testing.T) {
	qb := NewQueryBuilder(`
		SELECT
			a.id,
			(SELECT v.severity FROM vulnerabilities v WHERE v.asset_id = a.id ORDER BY v.created_at DESC LIMIT 1) AS latest_severity
		FROM assets a
		LEFT JOIN asset_relationships r ON r.source_asset_id = a.id`)
	sql, args := qb.Where("a.tenant_id = ?", "tenant-1").BuildCount()

	if !strings.HasPrefix(sql, "SELECT COUNT(DISTINCT a.id) FROM assets a") {
		t.Fatalf("expected top-level FROM in count sql, got %s", sql)
	}
	if strings.Contains(sql, "FROM vulnerabilities v WHERE v.asset_id = a.id") {
		t.Fatalf("expected nested FROM to be excluded from count prefix, got %s", sql)
	}
	if len(args) != 1 || args[0] != "tenant-1" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestQB_ParameterIndexing_ThroughTenConditions(t *testing.T) {
	qb := NewQueryBuilder("SELECT a.* FROM assets a")
	for idx := 1; idx <= 10; idx++ {
		qb.Where("a.metadata->>? = ?", "field", idx)
	}
	sql, args := qb.Build()
	if !strings.Contains(sql, "$10") {
		t.Fatalf("expected tenth placeholder in sql, got %s", sql)
	}
	if len(args) != 20 {
		t.Fatalf("expected 20 args, got %#v", args)
	}
}

func TestQB_ComplexQuery(t *testing.T) {
	qb := NewQueryBuilder("SELECT a.* FROM assets a")
	sql, args := qb.
		Where("a.tenant_id = ?", "tenant-1").
		WhereFTS([]string{"a.name", "a.hostname"}, "web-prod").
		WhereIn("a.type", []string{"server", "cloud_resource"}).
		WhereArrayContainsAll("a.tags", []string{"production", "web"}).
		WhereExists("SELECT 1 FROM vulnerabilities v WHERE v.asset_id = a.id AND v.severity = ?", "critical").
		OrderBy("created_at", "desc", []string{"created_at"}).
		Paginate(2, 25).
		Build()

	checks := []string{
		"a.tenant_id = $1",
		"plainto_tsquery('english', $2)",
		"a.type IN ($5, $6)",
		"a.tags @> ARRAY[$7, $8]",
		"v.severity = $9",
		"LIMIT 25 OFFSET 25",
	}
	for _, check := range checks {
		if !strings.Contains(sql, check) {
			t.Fatalf("expected %q in sql, got %s", check, sql)
		}
	}
	if len(args) != 9 {
		t.Fatalf("expected 9 args, got %#v", args)
	}
}
