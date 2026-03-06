package database

import (
	"strings"
	"testing"
)

func TestExtractOperation(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		want string
	}{
		{"select", "SELECT * FROM users", "select"},
		{"insert", "INSERT INTO users (name) VALUES ($1)", "insert"},
		{"update", "UPDATE users SET name = $1 WHERE id = $2", "update"},
		{"delete", "DELETE FROM users WHERE id = $1", "delete"},
		{"empty", "", "other"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractOperation(tc.sql)
			if got != tc.want {
				t.Errorf("extractOperation(%q) = %q, want %q", tc.sql, got, tc.want)
			}
		})
	}
}

func TestExtractOperation_CaseInsensitive(t *testing.T) {
	cases := []struct {
		sql  string
		want string
	}{
		{"select * from users", "select"},
		{"SELECT * FROM users", "select"},
		{"Select * From Users", "select"},
		{"INSERT into users VALUES (1)", "insert"},
		{"update users SET x=1", "update"},
		{"Delete from users", "delete"},
	}

	for _, tc := range cases {
		got := extractOperation(tc.sql)
		if got != tc.want {
			t.Errorf("extractOperation(%q) = %q, want %q", tc.sql, got, tc.want)
		}
	}
}

func TestExtractOperation_WithWhitespace(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		want string
	}{
		{"leading spaces", "   SELECT * FROM users", "select"},
		{"leading tab", "\tSELECT * FROM users", "select"},
		{"leading newline", "\nINSERT INTO users VALUES (1)", "insert"},
		{"leading carriage return", "\r\nUPDATE users SET x=1", "update"},
		{"multiple whitespace", "  \t\n  DELETE FROM users", "delete"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractOperation(tc.sql)
			if got != tc.want {
				t.Errorf("extractOperation(%q) = %q, want %q", tc.sql, got, tc.want)
			}
		})
	}
}

func TestExtractOperation_Unknown(t *testing.T) {
	cases := []struct {
		name string
		sql  string
	}{
		{"explain", "EXPLAIN SELECT * FROM users"},
		{"create table", "CREATE TABLE users (id int)"},
		{"drop table", "DROP TABLE users"},
		{"alter table", "ALTER TABLE users ADD COLUMN email TEXT"},
		{"with cte", "WITH cte AS (SELECT 1) SELECT * FROM cte"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractOperation(tc.sql)
			if got != "other" {
				t.Errorf("extractOperation(%q) = %q, want %q", tc.sql, got, "other")
			}
		})
	}
}

func TestExtractTable(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		want string
	}{
		{"select from", "SELECT * FROM users WHERE id = 1", "users"},
		{"insert into", "INSERT INTO orders (user_id) VALUES (1)", "orders"},
		{"update", "UPDATE products SET price = 10", "products"},
		{"select with schema", "SELECT * FROM public.users", "users"},
		{"insert with schema", "INSERT INTO myschema.events (data) VALUES ($1)", "events"},
		{"select with alias", "SELECT * FROM users u WHERE u.id = 1", "users"},
		{"no table", "SELECT 1", ""},
		{"delete from", "DELETE FROM sessions WHERE expired = true", "sessions"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractTable(tc.sql)
			if got != tc.want {
				t.Errorf("extractTable(%q) = %q, want %q", tc.sql, got, tc.want)
			}
		})
	}
}

func TestTruncateSQL(t *testing.T) {
	// Short query should not be truncated.
	short := "SELECT * FROM users"
	if got := truncateSQL(short); got != short {
		t.Errorf("truncateSQL(%q) = %q, want unchanged", short, got)
	}

	// Exactly maxSQLLogLen characters should not be truncated.
	exact := strings.Repeat("x", maxSQLLogLen)
	if got := truncateSQL(exact); got != exact {
		t.Errorf("truncateSQL(exact 200 chars) was modified, want unchanged")
	}

	// 500-character query should be truncated to 200 chars + "..."
	long := strings.Repeat("a", 500)
	got := truncateSQL(long)
	expected := strings.Repeat("a", maxSQLLogLen) + "..."
	if got != expected {
		t.Errorf("truncateSQL(500 chars) = %d chars, want %d chars", len(got), len(expected))
	}
	if len(got) != maxSQLLogLen+3 {
		t.Errorf("truncated length = %d, want %d", len(got), maxSQLLogLen+3)
	}

	// Verify it ends with "..."
	if !strings.HasSuffix(got, "...") {
		t.Error("truncated SQL does not end with '...'")
	}
}

func TestTruncateSQL_EmptyString(t *testing.T) {
	got := truncateSQL("")
	if got != "" {
		t.Errorf("truncateSQL(\"\") = %q, want empty string", got)
	}
}
