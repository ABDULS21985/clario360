package lineage

import (
	"regexp"
	"strings"

	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// Pre-compiled regex patterns for SQL lineage extraction.
var (
	// CREATE TABLE target AS SELECT ... FROM source1 [JOIN source2 ...]
	createTableAsRe = regexp.MustCompile(
		`(?i)CREATE\s+(?:OR\s+REPLACE\s+)?TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?` +
			`([a-zA-Z_][a-zA-Z0-9_.]*)\s+AS\s+SELECT\s+.+?\s+FROM\s+(.+?)` +
			`(?:\s+WHERE|\s+GROUP|\s+ORDER|\s+LIMIT|\s+HAVING|\s*;|\s*$)`)

	// INSERT INTO target SELECT ... FROM source
	insertSelectRe = regexp.MustCompile(
		`(?i)INSERT\s+(?:INTO\s+)?([a-zA-Z_][a-zA-Z0-9_.]*)\s+` +
			`(?:\([^)]*\)\s+)?SELECT\s+.+?\s+FROM\s+(.+?)` +
			`(?:\s+WHERE|\s+GROUP|\s+ORDER|\s+LIMIT|\s+HAVING|\s*;|\s*$)`)

	// CREATE [OR REPLACE] VIEW target AS SELECT ... FROM source
	createViewRe = regexp.MustCompile(
		`(?i)CREATE\s+(?:OR\s+REPLACE\s+)?(?:MATERIALIZED\s+)?VIEW\s+` +
			`(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z_][a-zA-Z0-9_.]*)\s+AS\s+SELECT\s+.+?\s+FROM\s+(.+?)` +
			`(?:\s+WHERE|\s+GROUP|\s+ORDER|\s+LIMIT|\s+HAVING|\s*;|\s*$)`)

	// JOIN clause to extract additional source tables.
	joinRe = regexp.MustCompile(
		`(?i)(?:INNER|LEFT|RIGHT|FULL|CROSS|NATURAL)?\s*JOIN\s+([a-zA-Z_][a-zA-Z0-9_.]*)`)

	// Simple FROM clause table extraction.
	fromTableRe = regexp.MustCompile(
		`(?i)([a-zA-Z_][a-zA-Z0-9_.]*)(?:\s+(?:AS\s+)?[a-zA-Z_][a-zA-Z0-9_]*)?`)
)

// SQLParser extracts data lineage relationships from SQL statements using
// simplified regex-based parsing.
type SQLParser struct{}

// NewSQLParser creates a SQLParser.
func NewSQLParser() *SQLParser {
	return &SQLParser{}
}

// ExtractLineage parses the given SQL string and returns lineage extractions
// for each recognized data-flow statement.
func (p *SQLParser) ExtractLineage(sql string) []model.SQLLineageExtraction {
	if strings.TrimSpace(sql) == "" {
		return nil
	}

	// Normalize whitespace.
	normalized := normalizeSQL(sql)

	var results []model.SQLLineageExtraction

	// Try each pattern type.
	results = append(results, p.extractCreateTableAs(normalized)...)
	results = append(results, p.extractInsertSelect(normalized)...)
	results = append(results, p.extractCreateView(normalized)...)

	return results
}

// extractCreateTableAs handles CREATE TABLE ... AS SELECT ... FROM patterns.
func (p *SQLParser) extractCreateTableAs(sql string) []model.SQLLineageExtraction {
	matches := createTableAsRe.FindAllStringSubmatch(sql, -1)
	if len(matches) == 0 {
		return nil
	}

	var results []model.SQLLineageExtraction
	for _, match := range matches {
		target := cleanTableName(match[1])
		fromClause := match[2]
		sources := extractSourceTables(fromClause)

		if len(sources) > 0 && target != "" {
			results = append(results, model.SQLLineageExtraction{
				SourceTables:   sources,
				TargetTable:    target,
				Transformation: "CREATE TABLE AS SELECT",
				StatementType:  "ctas",
			})
		}
	}
	return results
}

// extractInsertSelect handles INSERT INTO ... SELECT ... FROM patterns.
func (p *SQLParser) extractInsertSelect(sql string) []model.SQLLineageExtraction {
	matches := insertSelectRe.FindAllStringSubmatch(sql, -1)
	if len(matches) == 0 {
		return nil
	}

	var results []model.SQLLineageExtraction
	for _, match := range matches {
		target := cleanTableName(match[1])
		fromClause := match[2]
		sources := extractSourceTables(fromClause)

		if len(sources) > 0 && target != "" {
			results = append(results, model.SQLLineageExtraction{
				SourceTables:   sources,
				TargetTable:    target,
				Transformation: "INSERT SELECT",
				StatementType:  "insert_select",
			})
		}
	}
	return results
}

// extractCreateView handles CREATE VIEW ... AS SELECT ... FROM patterns.
func (p *SQLParser) extractCreateView(sql string) []model.SQLLineageExtraction {
	matches := createViewRe.FindAllStringSubmatch(sql, -1)
	if len(matches) == 0 {
		return nil
	}

	var results []model.SQLLineageExtraction
	for _, match := range matches {
		target := cleanTableName(match[1])
		fromClause := match[2]
		sources := extractSourceTables(fromClause)

		if len(sources) > 0 && target != "" {
			results = append(results, model.SQLLineageExtraction{
				SourceTables:   sources,
				TargetTable:    target,
				Transformation: "CREATE VIEW",
				StatementType:  "view",
			})
		}
	}
	return results
}

// extractSourceTables parses a FROM clause (including JOINs) to extract
// all source table names.
func extractSourceTables(fromClause string) []string {
	seen := make(map[string]bool)
	var tables []string

	// Extract the first table from the FROM clause.
	parts := strings.SplitN(strings.TrimSpace(fromClause), " ", 2)
	if len(parts) > 0 {
		first := cleanTableName(parts[0])
		// Skip if it looks like a subquery or keyword.
		if first != "" && !isSQLKeyword(first) && first != "(" {
			seen[first] = true
			tables = append(tables, first)
		}
	}

	// Handle comma-separated tables: FROM t1, t2.
	commaParts := strings.Split(fromClause, ",")
	for _, part := range commaParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// Take the first word (table name).
		subMatches := fromTableRe.FindStringSubmatch(part)
		if len(subMatches) > 1 {
			name := cleanTableName(subMatches[1])
			if name != "" && !isSQLKeyword(name) && !seen[name] {
				seen[name] = true
				tables = append(tables, name)
			}
		}
	}

	// Extract JOIN tables.
	joinMatches := joinRe.FindAllStringSubmatch(fromClause, -1)
	for _, m := range joinMatches {
		name := cleanTableName(m[1])
		if name != "" && !isSQLKeyword(name) && !seen[name] {
			seen[name] = true
			tables = append(tables, name)
		}
	}

	return tables
}

// normalizeSQL collapses whitespace and trims the SQL string.
func normalizeSQL(sql string) string {
	// Replace newlines and tabs with spaces.
	sql = strings.ReplaceAll(sql, "\n", " ")
	sql = strings.ReplaceAll(sql, "\r", " ")
	sql = strings.ReplaceAll(sql, "\t", " ")

	// Collapse multiple spaces.
	spaceRe := regexp.MustCompile(`\s+`)
	sql = spaceRe.ReplaceAllString(sql, " ")

	return strings.TrimSpace(sql)
}

// cleanTableName removes backticks, quotes, and trims whitespace from a table name.
func cleanTableName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Trim(name, "`\"'[]")
	name = strings.TrimRight(name, ",;")
	return name
}

// isSQLKeyword checks if a string is a SQL keyword that should not be treated
// as a table name.
func isSQLKeyword(s string) bool {
	keywords := map[string]bool{
		"select": true, "from": true, "where": true, "join": true,
		"inner": true, "left": true, "right": true, "full": true,
		"outer": true, "cross": true, "natural": true, "on": true,
		"and": true, "or": true, "not": true, "in": true,
		"group": true, "order": true, "by": true, "having": true,
		"limit": true, "offset": true, "union": true, "all": true,
		"as": true, "set": true, "values": true, "into": true,
		"insert": true, "update": true, "delete": true, "create": true,
		"drop": true, "alter": true, "table": true, "view": true,
		"index": true, "exists": true, "if": true, "replace": true,
		"case": true, "when": true, "then": true, "else": true, "end": true,
		"null": true, "is": true, "like": true, "between": true,
		"distinct": true, "count": true, "sum": true, "avg": true,
		"min": true, "max": true, "true": true, "false": true,
	}
	return keywords[strings.ToLower(s)]
}
