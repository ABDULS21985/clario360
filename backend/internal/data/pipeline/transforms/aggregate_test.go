package transforms

import "testing"

func TestApplyAggregate(t *testing.T) {
	rows := []map[string]any{
		{"department": "engineering", "salary": 100.0, "category": "staff"},
		{"department": "engineering", "salary": 150.0, "category": "staff"},
		{"department": "finance", "salary": 80.0, "category": "staff"},
	}

	got, stats, err := ApplyAggregate(rows, AggregateConfig{
		GroupBy: []string{"department"},
		Aggregations: []AggregateDefinition{
			{Column: "*", Function: "count", Alias: "employee_count"},
			{Column: "salary", Function: "sum", Alias: "total_salary"},
			{Column: "salary", Function: "avg", Alias: "avg_salary"},
			{Column: "category", Function: "count_distinct", Alias: "category_count"},
		},
	})
	if err != nil {
		t.Fatalf("ApplyAggregate() error = %v", err)
	}
	if stats.InputRows != 3 || stats.OutputRows != 2 {
		t.Fatalf("ApplyAggregate() stats = %+v", stats)
	}

	byDept := make(map[string]map[string]any, len(got))
	for _, row := range got {
		byDept[row["department"].(string)] = row
	}
	if byDept["engineering"]["employee_count"] != 2.0 {
		t.Fatalf("engineering count = %#v, want 2", byDept["engineering"]["employee_count"])
	}
	if byDept["engineering"]["total_salary"] != 250.0 {
		t.Fatalf("engineering total_salary = %#v, want 250", byDept["engineering"]["total_salary"])
	}
	if byDept["engineering"]["avg_salary"] != 125.0 {
		t.Fatalf("engineering avg_salary = %#v, want 125", byDept["engineering"]["avg_salary"])
	}
	if byDept["engineering"]["category_count"] != 1.0 {
		t.Fatalf("engineering category_count = %#v, want 1", byDept["engineering"]["category_count"])
	}
}
