package analytics

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/sqlutil"
)

type BuiltQuery struct {
	SQL       string
	CountSQL  string
	Args      []any
	CountArgs []any
}

func BuildSQL(query *model.AnalyticsQuery, dataModel *model.DataModel, source *model.DataSource) (*BuiltQuery, error) {
	if dataModel == nil || source == nil {
		return nil, fmt.Errorf("model and source are required")
	}
	dialect, tableExpr, err := analyticsTableExpression(dataModel, source)
	if err != nil {
		return nil, err
	}
	builder := newSQLBuilder(dialect)
	selectClause, err := buildSelectClause(query, dataModel, builder)
	if err != nil {
		return nil, err
	}
	whereClause, args, err := buildWhereClause(query.Filters, builder)
	if err != nil {
		return nil, err
	}
	groupByClause, err := buildGroupByClause(query, builder)
	if err != nil {
		return nil, err
	}
	orderByClause, err := buildOrderByClause(query, builder)
	if err != nil {
		return nil, err
	}
	limitOffset := fmt.Sprintf(" LIMIT %d OFFSET %d", query.Limit, query.Offset)
	sql := "SELECT " + selectClause + " FROM " + tableExpr + whereClause + groupByClause + orderByClause + limitOffset
	if err := sqlutil.ValidateReadOnlySQL(sql); err != nil {
		return nil, err
	}
	countSQL := buildCountSQL(query, tableExpr, whereClause, groupByClause)
	if err := sqlutil.ValidateReadOnlySQL(countSQL); err != nil {
		return nil, err
	}
	return &BuiltQuery{
		SQL:       sql,
		CountSQL:  countSQL,
		Args:      args,
		CountArgs: append([]any(nil), args...),
	}, nil
}

type sqlBuilder struct {
	dialect      string
	placeholderN int
}

func newSQLBuilder(dialect string) *sqlBuilder {
	return &sqlBuilder{dialect: dialect, placeholderN: 1}
}

func (b *sqlBuilder) QuoteIdentifier(value string) string {
	parts := strings.Split(strings.TrimSpace(value), ".")
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		switch b.dialect {
		case "mysql":
			quoted = append(quoted, "`"+strings.ReplaceAll(part, "`", "``")+"`")
		default:
			quoted = append(quoted, `"`+strings.ReplaceAll(part, `"`, `""`)+`"`)
		}
	}
	return strings.Join(quoted, ".")
}

func (b *sqlBuilder) Placeholder() string {
	switch b.dialect {
	case "mysql":
		return "?"
	default:
		value := "$" + strconv.Itoa(b.placeholderN)
		b.placeholderN++
		return value
	}
}

func analyticsTableExpression(dataModel *model.DataModel, source *model.DataSource) (string, string, error) {
	if dataModel.SourceTable == nil || strings.TrimSpace(*dataModel.SourceTable) == "" {
		return "", "", fmt.Errorf("model %q does not define a source table", dataModel.Name)
	}
	defaultSchema := "public"
	if source.ConnectionConfig != nil {
		switch source.Type {
		case model.DataSourceTypePostgreSQL:
			var cfg model.PostgresConnectionConfig
			if err := json.Unmarshal(source.ConnectionConfig, &cfg); err == nil && strings.TrimSpace(cfg.Schema) != "" {
				defaultSchema = strings.TrimSpace(cfg.Schema)
			}
		}
	}
	builder := newSQLBuilder(dialectForSource(source.Type))
	parts := strings.Split(*dataModel.SourceTable, ".")
	switch len(parts) {
	case 1:
		if source.Type == model.DataSourceTypePostgreSQL {
			return builder.dialect, builder.QuoteIdentifier(defaultSchema) + "." + builder.QuoteIdentifier(parts[0]), nil
		}
		return builder.dialect, builder.QuoteIdentifier(parts[0]), nil
	case 2:
		return builder.dialect, builder.QuoteIdentifier(parts[0]) + "." + builder.QuoteIdentifier(parts[1]), nil
	default:
		return "", "", fmt.Errorf("invalid source table %q", *dataModel.SourceTable)
	}
}

func dialectForSource(sourceType model.DataSourceType) string {
	if sourceType == model.DataSourceTypeMySQL {
		return "mysql"
	}
	return "postgres"
}

func buildSelectClause(query *model.AnalyticsQuery, dataModel *model.DataModel, builder *sqlBuilder) (string, error) {
	fields := make([]string, 0)
	if len(query.Aggregations) > 0 {
		for _, column := range query.GroupBy {
			fields = append(fields, builder.QuoteIdentifier(column))
		}
		for _, agg := range query.Aggregations {
			function := strings.ToUpper(strings.TrimSpace(agg.Function))
			target := "*"
			if agg.Column != "" {
				target = builder.QuoteIdentifier(agg.Column)
			}
			if agg.Distinct && agg.Column != "" {
				target = "DISTINCT " + target
			}
			expr := fmt.Sprintf("%s(%s)", function, target)
			if alias := strings.TrimSpace(agg.Alias); alias != "" {
				expr += " AS " + builder.QuoteIdentifier(alias)
			}
			fields = append(fields, expr)
		}
	} else {
		if len(query.Columns) == 0 {
			for _, field := range dataModel.SchemaDefinition {
				fields = append(fields, builder.QuoteIdentifier(field.Name))
			}
		} else {
			for _, column := range query.Columns {
				fields = append(fields, builder.QuoteIdentifier(column))
			}
		}
	}
	if len(fields) == 0 {
		return "", fmt.Errorf("query must select at least one column")
	}
	return strings.Join(fields, ", "), nil
}

func buildWhereClause(filters []model.AnalyticsFilter, builder *sqlBuilder) (string, []any, error) {
	if len(filters) == 0 {
		return "", nil, nil
	}
	conditions := make([]string, 0, len(filters))
	args := make([]any, 0)
	for _, filter := range filters {
		column := builder.QuoteIdentifier(filter.Column)
		switch strings.ToLower(strings.TrimSpace(filter.Operator)) {
		case "eq":
			conditions = append(conditions, column+" = "+builder.Placeholder())
			args = append(args, filter.Value)
		case "neq":
			conditions = append(conditions, column+" != "+builder.Placeholder())
			args = append(args, filter.Value)
		case "gt":
			conditions = append(conditions, column+" > "+builder.Placeholder())
			args = append(args, filter.Value)
		case "gte":
			conditions = append(conditions, column+" >= "+builder.Placeholder())
			args = append(args, filter.Value)
		case "lt":
			conditions = append(conditions, column+" < "+builder.Placeholder())
			args = append(args, filter.Value)
		case "lte":
			conditions = append(conditions, column+" <= "+builder.Placeholder())
			args = append(args, filter.Value)
		case "like":
			conditions = append(conditions, column+" LIKE "+builder.Placeholder())
			args = append(args, filter.Value)
		case "ilike":
			if builder.dialect == "mysql" {
				conditions = append(conditions, "LOWER("+column+") LIKE LOWER("+builder.Placeholder()+")")
			} else {
				conditions = append(conditions, column+" ILIKE "+builder.Placeholder())
			}
			args = append(args, filter.Value)
		case "between":
			values := interfaceSlice(filter.Value)
			conditions = append(conditions, column+" BETWEEN "+builder.Placeholder()+" AND "+builder.Placeholder())
			args = append(args, values[0], values[1])
		case "in":
			placeholders := make([]string, 0)
			for _, value := range interfaceSlice(filter.Value) {
				placeholders = append(placeholders, builder.Placeholder())
				args = append(args, value)
			}
			conditions = append(conditions, column+" IN ("+strings.Join(placeholders, ", ")+")")
		case "not_in":
			placeholders := make([]string, 0)
			for _, value := range interfaceSlice(filter.Value) {
				placeholders = append(placeholders, builder.Placeholder())
				args = append(args, value)
			}
			conditions = append(conditions, column+" NOT IN ("+strings.Join(placeholders, ", ")+")")
		case "is_null":
			conditions = append(conditions, column+" IS NULL")
		case "is_not_null":
			conditions = append(conditions, column+" IS NOT NULL")
		default:
			return "", nil, fmt.Errorf("invalid operator %q", filter.Operator)
		}
	}
	return " WHERE " + strings.Join(conditions, " AND "), args, nil
}

func buildGroupByClause(query *model.AnalyticsQuery, builder *sqlBuilder) (string, error) {
	if len(query.GroupBy) == 0 {
		return "", nil
	}
	quoted := make([]string, 0, len(query.GroupBy))
	for _, column := range query.GroupBy {
		quoted = append(quoted, builder.QuoteIdentifier(column))
	}
	return " GROUP BY " + strings.Join(quoted, ", "), nil
}

func buildOrderByClause(query *model.AnalyticsQuery, builder *sqlBuilder) (string, error) {
	if len(query.OrderBy) == 0 {
		if len(query.Columns) > 0 {
			return " ORDER BY " + builder.QuoteIdentifier(query.Columns[0]) + " ASC", nil
		}
		if len(query.GroupBy) > 0 {
			return " ORDER BY " + builder.QuoteIdentifier(query.GroupBy[0]) + " ASC", nil
		}
		return "", nil
	}
	parts := make([]string, 0, len(query.OrderBy))
	for _, order := range query.OrderBy {
		direction := strings.ToUpper(strings.TrimSpace(order.Direction))
		if direction == "" {
			direction = "ASC"
		}
		parts = append(parts, builder.QuoteIdentifier(order.Column)+" "+direction)
	}
	return " ORDER BY " + strings.Join(parts, ", "), nil
}

func buildCountSQL(query *model.AnalyticsQuery, tableExpr, whereClause, groupByClause string) string {
	if len(query.Aggregations) > 0 || groupByClause != "" {
		return "SELECT COUNT(*) FROM (SELECT 1 FROM " + tableExpr + whereClause + groupByClause + ") AS grouped_rows"
	}
	return "SELECT COUNT(*) FROM " + tableExpr + whereClause
}

func interfaceSlice(value any) []any {
	switch values := value.(type) {
	case []any:
		return values
	case []string:
		out := make([]any, 0, len(values))
		for _, item := range values {
			out = append(out, item)
		}
		return out
	default:
		return []any{value}
	}
}
