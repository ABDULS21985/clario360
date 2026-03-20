package analytics

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/clario360/platform/internal/data/model"
)

func TestBuildSQLSimpleSelect(t *testing.T) {
	source := &model.DataSource{Type: model.DataSourceTypePostgreSQL, ConnectionConfig: json.RawMessage(`{"schema":"public"}`)}
	table := "users"
	dataModel := &model.DataModel{
		Name:        "users",
		SourceTable: &table,
		SchemaDefinition: []model.ModelField{
			{Name: "id"},
			{Name: "user"},
			{Name: "email"},
		},
	}
	query := &model.AnalyticsQuery{
		Columns: []string{"id", "user", "email"},
		OrderBy: []model.AnalyticsOrder{{Column: "id", Direction: "desc"}},
		Limit:   50,
	}
	built, err := BuildSQL(query, dataModel, source)
	if err != nil {
		t.Fatalf("BuildSQL() unexpected error = %v", err)
	}
	if !strings.Contains(built.SQL, `SELECT "id", "user", "email" FROM "public"."users"`) {
		t.Fatalf("SQL = %s", built.SQL)
	}
	if !strings.Contains(built.SQL, `ORDER BY "id" DESC`) {
		t.Fatalf("SQL missing order by: %s", built.SQL)
	}
}

func TestBuildSQLWithFiltersAndCount(t *testing.T) {
	source := &model.DataSource{Type: model.DataSourceTypePostgreSQL, ConnectionConfig: json.RawMessage(`{"schema":"public"}`)}
	table := "payments"
	dataModel := &model.DataModel{
		Name:        "payments",
		SourceTable: &table,
		SchemaDefinition: []model.ModelField{
			{Name: "amount"},
			{Name: "status"},
		},
	}
	query := &model.AnalyticsQuery{
		Columns: []string{"amount", "status"},
		Filters: []model.AnalyticsFilter{
			{Column: "status", Operator: "eq", Value: "posted"},
			{Column: "amount", Operator: "gt", Value: 100},
		},
		Limit: 25,
	}
	built, err := BuildSQL(query, dataModel, source)
	if err != nil {
		t.Fatalf("BuildSQL() unexpected error = %v", err)
	}
	if !strings.Contains(built.SQL, `"status" = $1 AND "amount" > $2`) {
		t.Fatalf("SQL = %s", built.SQL)
	}
	if !strings.Contains(built.CountSQL, `SELECT COUNT(*) FROM "public"."payments" WHERE "status" = $1 AND "amount" > $2`) {
		t.Fatalf("CountSQL = %s", built.CountSQL)
	}
}

func TestBuildSQLAggregation(t *testing.T) {
	source := &model.DataSource{Type: model.DataSourceTypeMySQL, ConnectionConfig: json.RawMessage(`{"database":"warehouse"}`)}
	table := "salaries"
	dataModel := &model.DataModel{
		Name:        "salaries",
		SourceTable: &table,
		SchemaDefinition: []model.ModelField{
			{Name: "dept"},
			{Name: "salary"},
		},
	}
	query := &model.AnalyticsQuery{
		GroupBy: []string{"dept"},
		Aggregations: []model.AnalyticsAggregation{
			{Function: "avg", Column: "salary", Alias: "avg_salary"},
		},
		Limit: 100,
	}
	built, err := BuildSQL(query, dataModel, source)
	if err != nil {
		t.Fatalf("BuildSQL() unexpected error = %v", err)
	}
	if !strings.Contains(built.SQL, "SELECT `dept`, AVG(`salary`) AS `avg_salary` FROM `salaries` GROUP BY `dept`") {
		t.Fatalf("SQL = %s", built.SQL)
	}
	if !strings.Contains(built.CountSQL, "SELECT COUNT(*) FROM (SELECT 1 FROM `salaries` GROUP BY `dept`) AS grouped_rows") {
		t.Fatalf("CountSQL = %s", built.CountSQL)
	}
}
