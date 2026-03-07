package database

import (
	"fmt"
	"strings"
)

// QueryBuilder constructs parameterized PostgreSQL queries dynamically.
// It tracks parameter indices ($1, $2, …) automatically so callers cannot
// produce index mismatches.
//
// Example:
//
//	qb := NewQueryBuilder("SELECT a.* FROM assets a")
//	qb.Where("a.tenant_id = ?", tenantID)
//	qb.WhereIf(filter != "", "a.name ILIKE ?", "%"+filter+"%")
//	qb.OrderBy("created_at", "desc", allowedSorts)
//	qb.Paginate(1, 25)
//	sql, args := qb.Build()
//	count, cargs := qb.BuildCount()
type QueryBuilder struct {
	baseSelect  string
	joins       []string
	conditions  []string
	args        []any
	groupBy     string
	having      []string
	havingArgs  []any
	orderClause string
	limit       int
	offset      int
	nextArg     int
}

// NewQueryBuilder creates a builder with the given SELECT…FROM clause.
// nextArg starts at 1.
func NewQueryBuilder(baseSelect string) *QueryBuilder {
	return &QueryBuilder{
		baseSelect: baseSelect,
		nextArg:    1,
	}
}

// Join appends a JOIN clause. Joins are static — no parameter handling.
func (qb *QueryBuilder) Join(joinClause string) *QueryBuilder {
	qb.joins = append(qb.joins, joinClause)
	return qb
}

// Where appends a WHERE condition. Every "?" in condition is replaced with
// the next positional placeholder ($N) and the corresponding arg is stored.
func (qb *QueryBuilder) Where(condition string, args ...any) *QueryBuilder {
	cond, addedArgs := qb.replacePlaceholders(condition, args)
	qb.conditions = append(qb.conditions, cond)
	qb.args = append(qb.args, addedArgs...)
	return qb
}

// WhereIf calls Where only when pred is true. This is a no-op when pred is false.
func (qb *QueryBuilder) WhereIf(pred bool, condition string, args ...any) *QueryBuilder {
	if pred {
		qb.Where(condition, args...)
	}
	return qb
}

// WhereIn generates: column IN ($N, $N+1, …) with one placeholder per value.
// If values is empty the call is a no-op (prevents invalid SQL "IN ()").
func (qb *QueryBuilder) WhereIn(column string, values []string) *QueryBuilder {
	if len(values) == 0 {
		return qb
	}
	placeholders := make([]string, len(values))
	for i, v := range values {
		placeholders[i] = fmt.Sprintf("$%d", qb.nextArg)
		qb.args = append(qb.args, v)
		qb.nextArg++
	}
	cond := fmt.Sprintf("%s IN (%s)", column, strings.Join(placeholders, ", "))
	qb.conditions = append(qb.conditions, cond)
	return qb
}

// WhereExists appends: EXISTS (subquery) where "?" in subquery is replaced
// with $N placeholders.
func (qb *QueryBuilder) WhereExists(subquery string, args ...any) *QueryBuilder {
	inner, addedArgs := qb.replacePlaceholders(subquery, args)
	cond := fmt.Sprintf("EXISTS (%s)", inner)
	qb.conditions = append(qb.conditions, cond)
	qb.args = append(qb.args, addedArgs...)
	return qb
}

// WhereArrayContains generates: $N = ANY(column)
func (qb *QueryBuilder) WhereArrayContains(column string, value string) *QueryBuilder {
	cond := fmt.Sprintf("$%d = ANY(%s)", qb.nextArg, column)
	qb.args = append(qb.args, value)
	qb.nextArg++
	qb.conditions = append(qb.conditions, cond)
	return qb
}

// WhereArrayContainsAll generates: column @> ARRAY[$N, $N+1, …]
// Requires that the column contains ALL listed values.
func (qb *QueryBuilder) WhereArrayContainsAll(column string, values []string) *QueryBuilder {
	if len(values) == 0 {
		return qb
	}
	placeholders := make([]string, len(values))
	for i, v := range values {
		placeholders[i] = fmt.Sprintf("$%d", qb.nextArg)
		qb.args = append(qb.args, v)
		qb.nextArg++
	}
	cond := fmt.Sprintf("%s @> ARRAY[%s]", column, strings.Join(placeholders, ", "))
	qb.conditions = append(qb.conditions, cond)
	return qb
}

// WhereFTS generates a full-text search condition plus exact IP/tag fallbacks.
// columns are joined with coalesce(col, ”) || ' ' for the tsvector expression.
func (qb *QueryBuilder) WhereFTS(columns []string, query string) *QueryBuilder {
	if len(columns) == 0 || query == "" {
		return qb
	}

	// Build the tsvector concat expression
	parts := make([]string, len(columns))
	for i, col := range columns {
		parts[i] = fmt.Sprintf("coalesce(%s, '')", col)
	}
	tsvector := fmt.Sprintf("to_tsvector('english', %s)", strings.Join(parts, " || ' ' || "))
	tsquery := fmt.Sprintf("plainto_tsquery('english', $%d)", qb.nextArg)
	qb.args = append(qb.args, query)
	qb.nextArg++

	// Exact IP address match
	ipIdx := qb.nextArg
	qb.args = append(qb.args, query)
	qb.nextArg++

	// Tag match
	tagIdx := qb.nextArg
	qb.args = append(qb.args, query)
	qb.nextArg++

	cond := fmt.Sprintf(
		"(%s @@ %s OR host(ip_address) = $%d OR $%d = ANY(tags))",
		tsvector, tsquery, ipIdx, tagIdx,
	)
	qb.conditions = append(qb.conditions, cond)
	return qb
}

// GroupBy sets the GROUP BY clause (replaces any previous value).
func (qb *QueryBuilder) GroupBy(clause string) *QueryBuilder {
	qb.groupBy = clause
	return qb
}

// Having appends a HAVING condition using "?" → $N replacement.
func (qb *QueryBuilder) Having(condition string, args ...any) *QueryBuilder {
	cond, addedArgs := qb.replacePlaceholders(condition, args)
	qb.having = append(qb.having, cond)
	qb.havingArgs = append(qb.havingArgs, addedArgs...)
	return qb
}

// OrderBy sets ORDER BY with allowlist validation.
// column must be in allowlist; direction is normalised to "ASC"/"DESC".
// Special cases: "criticality" → ORDER BY severity_order(criticality);
// "vulnerability_count" → ORDER BY open_vulnerability_count.
func (qb *QueryBuilder) OrderBy(column, direction string, allowlist []string) *QueryBuilder {
	// Validate column
	valid := false
	for _, a := range allowlist {
		if strings.EqualFold(a, column) {
			valid = true
			break
		}
	}
	if !valid {
		return qb // silently ignore unknown columns (caller should validate earlier)
	}

	// Normalise direction
	dir := "DESC"
	if strings.EqualFold(direction, "asc") {
		dir = "ASC"
	}

	// Special column mappings
	switch strings.ToLower(column) {
	case "criticality":
		qb.orderClause = fmt.Sprintf("ORDER BY severity_order(a.criticality) %s", dir)
	case "severity":
		qb.orderClause = fmt.Sprintf("ORDER BY severity_order(a.severity) %s", dir)
	case "vulnerability_count":
		qb.orderClause = fmt.Sprintf("ORDER BY open_vulnerability_count %s", dir)
	default:
		qb.orderClause = fmt.Sprintf("ORDER BY a.%s %s", column, dir)
	}
	return qb
}

// Paginate sets LIMIT and OFFSET. page is 1-indexed; perPage is clamped to [1, 200].
func (qb *QueryBuilder) Paginate(page, perPage int) *QueryBuilder {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 1
	}
	if perPage > 200 {
		perPage = 200
	}
	qb.limit = perPage
	qb.offset = (page - 1) * perPage
	return qb
}

// Build assembles the full SQL query and returns it with its args slice.
func (qb *QueryBuilder) Build() (string, []any) {
	var sb strings.Builder
	sb.WriteString(qb.baseSelect)

	for _, j := range qb.joins {
		sb.WriteString(" ")
		sb.WriteString(j)
	}

	if len(qb.conditions) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(qb.conditions, " AND "))
	}

	if qb.groupBy != "" {
		sb.WriteString(" GROUP BY ")
		sb.WriteString(qb.groupBy)
	}

	allArgs := make([]any, len(qb.args))
	copy(allArgs, qb.args)

	if len(qb.having) > 0 {
		sb.WriteString(" HAVING ")
		sb.WriteString(strings.Join(qb.having, " AND "))
		allArgs = append(allArgs, qb.havingArgs...)
	}

	if qb.orderClause != "" {
		sb.WriteString(" ")
		sb.WriteString(qb.orderClause)
	}

	if qb.limit > 0 {
		sb.WriteString(fmt.Sprintf(" LIMIT %d OFFSET %d", qb.limit, qb.offset))
	}

	return sb.String(), allArgs
}

// BuildCount returns a COUNT(DISTINCT a.id) query using the same WHERE/HAVING
// conditions as Build(), without ORDER BY or LIMIT/OFFSET.
func (qb *QueryBuilder) BuildCount() (string, []any) {
	if qb.groupBy != "" || len(qb.having) > 0 {
		query, args := qb.Build()
		return fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS qb_count", query), args
	}

	var sb strings.Builder
	sb.WriteString("SELECT COUNT(DISTINCT a.id)")

	// Extract the FROM clause from baseSelect (everything after SELECT … FROM)
	fromIdx := strings.Index(strings.ToUpper(qb.baseSelect), " FROM ")
	if fromIdx >= 0 {
		sb.WriteString(qb.baseSelect[fromIdx:])
	}

	for _, j := range qb.joins {
		sb.WriteString(" ")
		sb.WriteString(j)
	}

	if len(qb.conditions) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(qb.conditions, " AND "))
	}

	allArgs := make([]any, len(qb.args))
	copy(allArgs, qb.args)

	return sb.String(), allArgs
}

// replacePlaceholders replaces each "?" in s with the next $N placeholder,
// consuming one arg per placeholder. Returns the rewritten string and the
// consumed args in order.
func (qb *QueryBuilder) replacePlaceholders(s string, args []any) (string, []any) {
	var sb strings.Builder
	argIdx := 0
	consumed := make([]any, 0, len(args))

	for i := 0; i < len(s); i++ {
		if s[i] == '?' && argIdx < len(args) {
			sb.WriteString(fmt.Sprintf("$%d", qb.nextArg))
			consumed = append(consumed, args[argIdx])
			qb.nextArg++
			argIdx++
		} else {
			sb.WriteByte(s[i])
		}
	}
	return sb.String(), consumed
}
