// audit-isolation audits all Clario360 databases for tenant isolation compliance.
//
// Usage:
//
//	audit-isolation [flags]
//	  -output string   Output format: json|text (default "text")
//	  -fix             Print remediation SQL (does NOT execute it)
//
// Environment variables:
//
//	PLATFORM_DB_DSN, CYBER_DB_DSN, DATA_DB_DSN, AUDIT_DB_DSN,
//	ACTA_DB_DSN, LEX_DB_DSN, NOTIFICATION_DB_DSN, VISUS_DB_DSN
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ANSI color codes for text output.
const (
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
)

// expectedPolicies is the set of policy names expected on every fully-isolated table.
var expectedPolicies = []string{
	"tenant_isolation",
	"tenant_insert",
	"tenant_update",
	"tenant_delete",
}

// dbTarget describes a database to audit and the tables it contains.
type dbTarget struct {
	Name      string
	DSNEnvVar string
	Tables    []tableTarget
}

// tableTarget describes a single table to audit.
type tableTarget struct {
	Name             string
	NullableTenantID bool // system_settings, notification_templates
	SkipUpdateDelete bool // immutable tables (audit_logs in audit_db)
}

// tableResult holds the audit result for one table.
type tableResult struct {
	Table           string   `json:"table"`
	RLSEnabled      bool     `json:"rls_enabled"`
	RLSForced       bool     `json:"rls_forced"`
	HasTenantIndex  bool     `json:"has_tenant_index"`
	NullTenantCount int64    `json:"null_tenant_count"`
	Policies        []string `json:"policies"`
	MissingPolicies []string `json:"missing_policies"`
	Pass            bool     `json:"pass"`
}

// dbResult holds the audit result for one database.
type dbResult struct {
	Name      string        `json:"name"`
	Reachable bool          `json:"reachable"`
	Error     string        `json:"error,omitempty"`
	Tables    []tableResult `json:"tables"`
}

// summaryResult holds aggregate counts across all databases.
type summaryResult struct {
	TotalTables   int  `json:"total_tables"`
	RLSEnabled    int  `json:"rls_enabled"`
	RLSForced     int  `json:"rls_forced"`
	MissingIndex  int  `json:"missing_index"`
	NullTenantIDs int  `json:"null_tenant_ids"`
	Pass          bool `json:"pass"`
}

// report is the top-level JSON output structure.
type report struct {
	GeneratedAt string        `json:"generated_at"`
	Summary     summaryResult `json:"summary"`
	Databases   []dbResult    `json:"databases"`
}

// databases defines all databases and their tables to audit.
var databases = []dbTarget{
	{
		Name:      "platform_core",
		DSNEnvVar: "PLATFORM_DB_DSN",
		Tables: []tableTarget{
			{Name: "users"},
			{Name: "roles"},
			{Name: "user_roles"},
			{Name: "sessions"},
			{Name: "api_keys"},
			{Name: "notifications"},
			{Name: "system_settings", NullableTenantID: true},
			{Name: "audit_logs"},
		},
	},
	{
		Name:      "cyber_db",
		DSNEnvVar: "CYBER_DB_DSN",
		Tables: []tableTarget{
			{Name: "assets"},
			{Name: "asset_relationships"},
			{Name: "vulnerabilities"},
			{Name: "threats"},
			{Name: "threat_indicators"},
			{Name: "detection_rules"},
			{Name: "alerts"},
			{Name: "remediation_actions"},
			{Name: "remediation_audit_trail"},
			{Name: "ctem_assessments"},
			{Name: "dspm_data_assets"},
			{Name: "dspm_scans"},
			{Name: "scan_history"},
			{Name: "vciso_briefings"},
		},
	},
	{
		Name:      "data_db",
		DSNEnvVar: "DATA_DB_DSN",
		Tables: []tableTarget{
			{Name: "data_sources"},
			{Name: "data_models"},
			{Name: "quality_rules"},
			{Name: "quality_results"},
			{Name: "contradictions"},
			{Name: "pipelines"},
			{Name: "pipeline_runs"},
			{Name: "pipeline_run_logs"},
			{Name: "data_lineage_edges"},
			{Name: "dark_data_assets"},
			{Name: "dark_data_scans"},
			{Name: "data_catalogs"},
			{Name: "saved_queries"},
			{Name: "analytics_audit_log"},
			{Name: "contradiction_scans"},
		},
	},
	{
		Name:      "audit_db",
		DSNEnvVar: "AUDIT_DB_DSN",
		Tables: []tableTarget{
			{Name: "audit_logs", SkipUpdateDelete: true},
			{Name: "audit_chain_state"},
		},
	},
	{
		Name:      "acta_db",
		DSNEnvVar: "ACTA_DB_DSN",
		Tables: []tableTarget{
			{Name: "committees"},
			{Name: "committee_members"},
			{Name: "meetings"},
			{Name: "meeting_attendance"},
			{Name: "agenda_items"},
			{Name: "meeting_minutes"},
			{Name: "action_items"},
			{Name: "compliance_checks"},
		},
	},
	{
		Name:      "lex_db",
		DSNEnvVar: "LEX_DB_DSN",
		Tables: []tableTarget{
			{Name: "contracts"},
			{Name: "contract_versions"},
			{Name: "contract_clauses"},
			{Name: "contract_analyses"},
			{Name: "legal_documents"},
			{Name: "document_versions"},
			{Name: "compliance_rules"},
			{Name: "compliance_alerts"},
			{Name: "expiry_notifications"},
		},
	},
	{
		Name:      "notification_db",
		DSNEnvVar: "NOTIFICATION_DB_DSN",
		Tables: []tableTarget{
			{Name: "notifications"},
			{Name: "notification_preferences"},
			{Name: "notification_webhooks"},
			{Name: "notification_templates", NullableTenantID: true},
		},
	},
	{
		Name:      "visus_db",
		DSNEnvVar: "VISUS_DB_DSN",
		Tables: []tableTarget{
			{Name: "visus_dashboards"},
			{Name: "visus_widgets"},
			{Name: "visus_kpi_definitions"},
			{Name: "visus_kpi_snapshots"},
			{Name: "visus_executive_alerts"},
			{Name: "visus_report_definitions"},
			{Name: "visus_report_snapshots"},
			{Name: "visus_suite_cache"},
		},
	},
}

func main() {
	outputFmt := flag.String("output", "text", "Output format: json|text")
	fix := flag.Bool("fix", false, "Print remediation SQL (does NOT execute it)")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	r := runAudit(ctx, databases)

	switch *outputFmt {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(r); err != nil {
			fmt.Fprintf(os.Stderr, "error encoding JSON: %v\n", err)
			os.Exit(1)
		}
	default:
		printTextReport(r)
	}

	if *fix {
		fmt.Println()
		printRemediationSQL(r)
	}

	if !r.Summary.Pass {
		os.Exit(1)
	}
}

// runAudit connects to each database and audits all tables.
func runAudit(ctx context.Context, dbs []dbTarget) report {
	r := report{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}

	for _, db := range dbs {
		result := auditDatabase(ctx, db)
		r.Databases = append(r.Databases, result)

		for _, t := range result.Tables {
			r.Summary.TotalTables++
			if t.RLSEnabled {
				r.Summary.RLSEnabled++
			}
			if t.RLSForced {
				r.Summary.RLSForced++
			}
			if !t.HasTenantIndex {
				r.Summary.MissingIndex++
			}
			if t.NullTenantCount > 0 {
				r.Summary.NullTenantIDs++
			}
		}
	}

	r.Summary.Pass = r.Summary.TotalTables > 0 &&
		r.Summary.RLSEnabled == r.Summary.TotalTables &&
		r.Summary.RLSForced == r.Summary.TotalTables &&
		r.Summary.MissingIndex == 0 &&
		r.Summary.NullTenantIDs == 0

	return r
}

// auditDatabase connects to one database and audits all its tables.
func auditDatabase(ctx context.Context, db dbTarget) dbResult {
	result := dbResult{
		Name:      db.Name,
		Reachable: false,
	}

	dsn := os.Getenv(db.DSNEnvVar)
	if dsn == "" {
		result.Error = fmt.Sprintf("env var %s not set", db.DSNEnvVar)
		return result
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		result.Error = fmt.Sprintf("connect: %v", err)
		return result
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		result.Error = fmt.Sprintf("ping: %v", err)
		return result
	}
	result.Reachable = true

	for _, tbl := range db.Tables {
		tr := auditTable(ctx, pool, tbl)
		result.Tables = append(result.Tables, tr)
	}

	return result
}

// auditTable checks RLS status, tenant_id index, null counts, and policy existence.
func auditTable(ctx context.Context, pool *pgxpool.Pool, tbl tableTarget) tableResult {
	result := tableResult{
		Table: tbl.Name,
	}

	// 1. Check RLS enabled and forced.
	var rlsEnabled, rlsForced bool
	err := pool.QueryRow(ctx,
		"SELECT relrowsecurity, relforcerowsecurity FROM pg_class WHERE relname = $1 AND relkind = 'r'",
		tbl.Name,
	).Scan(&rlsEnabled, &rlsForced)
	if err != nil {
		// Table may not exist (e.g., partitioned parent) — attempt with partitioned tables.
		_ = pool.QueryRow(ctx,
			"SELECT relrowsecurity, relforcerowsecurity FROM pg_class WHERE relname = $1",
			tbl.Name,
		).Scan(&rlsEnabled, &rlsForced)
	}
	result.RLSEnabled = rlsEnabled
	result.RLSForced = rlsForced

	// 2. Check tenant_id index existence.
	var hasIndex bool
	_ = pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM pg_indexes
			WHERE tablename = $1
			  AND indexdef LIKE '%tenant_id%'
		)`, tbl.Name,
	).Scan(&hasIndex)
	result.HasTenantIndex = hasIndex

	// 3. Count NULL tenant_ids (only for tables where tenant_id should never be NULL).
	if !tbl.NullableTenantID {
		// Use a dynamic query safely constructed — table names come from our static list only.
		// This is safe: tbl.Name is never user-supplied.
		var nullCount int64
		nullErr := pool.QueryRow(ctx,
			fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE tenant_id IS NULL", pgQuoteIdentifier(tbl.Name)),
		).Scan(&nullCount)
		if nullErr == nil {
			result.NullTenantCount = nullCount
		}
	}

	// 4. List existing policies.
	rows, err := pool.Query(ctx,
		"SELECT policyname FROM pg_policies WHERE tablename = $1 ORDER BY policyname",
		tbl.Name,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var pname string
			if err := rows.Scan(&pname); err == nil {
				result.Policies = append(result.Policies, pname)
			}
		}
	}

	// 5. Determine expected policies and find missing ones.
	expected := expectedPoliciesFor(tbl)
	policySet := make(map[string]bool, len(result.Policies))
	for _, p := range result.Policies {
		policySet[p] = true
	}
	for _, ep := range expected {
		if !policySet[ep] {
			result.MissingPolicies = append(result.MissingPolicies, ep)
		}
	}

	// 6. Determine pass/fail.
	result.Pass = result.RLSEnabled &&
		result.RLSForced &&
		result.HasTenantIndex &&
		result.NullTenantCount == 0 &&
		len(result.MissingPolicies) == 0

	return result
}

// expectedPoliciesFor returns the list of policies expected on a table.
func expectedPoliciesFor(tbl tableTarget) []string {
	if tbl.SkipUpdateDelete {
		return []string{"tenant_isolation", "tenant_insert"}
	}
	return expectedPolicies
}

// pgQuoteIdentifier quotes a PostgreSQL identifier safely.
// Only used with our static table name list — never with user input.
func pgQuoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// ---------------------------------------------------------------------------
// Text output
// ---------------------------------------------------------------------------

func printTextReport(r report) {
	fmt.Printf("%s%s=== Clario360 Tenant Isolation Audit ===%s\n", colorBold, colorBold, colorReset)
	fmt.Printf("Generated: %s\n\n", r.GeneratedAt)

	for _, db := range r.Databases {
		fmt.Printf("%s%s[%s]%s\n", colorBold, colorBold, db.Name, colorReset)

		if !db.Reachable {
			fmt.Printf("  %sUNREACHABLE%s: %s\n\n", colorRed, colorReset, db.Error)
			continue
		}

		for _, tbl := range db.Tables {
			status := colorGreen + "PASS" + colorReset
			if !tbl.Pass {
				status = colorRed + "FAIL" + colorReset
			}
			fmt.Printf("  [%s] %s\n", status, tbl.Table)

			if !tbl.RLSEnabled {
				fmt.Printf("        %s! RLS not enabled%s\n", colorRed, colorReset)
			}
			if !tbl.RLSForced {
				fmt.Printf("        %s! RLS not forced (table owner can bypass)%s\n", colorYellow, colorReset)
			}
			if !tbl.HasTenantIndex {
				fmt.Printf("        %s! No tenant_id index (performance risk)%s\n", colorYellow, colorReset)
			}
			if tbl.NullTenantCount > 0 {
				fmt.Printf("        %s! %d rows with NULL tenant_id%s\n", colorRed, tbl.NullTenantCount, colorReset)
			}
			for _, mp := range tbl.MissingPolicies {
				fmt.Printf("        %s! Missing policy: %s%s\n", colorRed, mp, colorReset)
			}
		}
		fmt.Println()
	}

	// Summary.
	summaryColor := colorGreen
	summaryStatus := "PASS"
	if !r.Summary.Pass {
		summaryColor = colorRed
		summaryStatus = "FAIL"
	}

	fmt.Printf("%s=== Summary ===%s\n", colorBold, colorReset)
	fmt.Printf("  Total tables audited : %d\n", r.Summary.TotalTables)
	fmt.Printf("  RLS enabled          : %d / %d\n", r.Summary.RLSEnabled, r.Summary.TotalTables)
	fmt.Printf("  RLS forced           : %d / %d\n", r.Summary.RLSForced, r.Summary.TotalTables)
	fmt.Printf("  Missing tenant index : %d\n", r.Summary.MissingIndex)
	fmt.Printf("  Tables with NULL IDs : %d\n", r.Summary.NullTenantIDs)
	fmt.Printf("  Overall status       : %s%s%s\n", summaryColor, summaryStatus, colorReset)
}

// ---------------------------------------------------------------------------
// Remediation SQL output (-fix flag)
// ---------------------------------------------------------------------------

func printRemediationSQL(r report) {
	fmt.Println("-- =================================================================")
	fmt.Println("-- Remediation SQL — review and apply manually after testing")
	fmt.Println("-- IMPORTANT: Run as a role with BYPASSRLS privilege")
	fmt.Println("-- =================================================================")
	fmt.Println()

	for _, db := range r.Databases {
		if !db.Reachable {
			continue
		}

		needsWork := false
		for _, tbl := range db.Tables {
			if !tbl.Pass {
				needsWork = true
				break
			}
		}
		if !needsWork {
			continue
		}

		fmt.Printf("-- Database: %s\n", db.Name)
		fmt.Printf("-- Connect: psql \"$%s_DB_DSN\"\n\n", strings.ToUpper(strings.ReplaceAll(db.Name, "_", "")))

		for _, tbl := range db.Tables {
			if tbl.Pass {
				continue
			}

			fmt.Printf("-- Table: %s\n", tbl.Table)

			if !tbl.RLSEnabled {
				fmt.Printf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY;\n", pgQuoteIdentifier(tbl.Table))
			}
			if !tbl.RLSForced {
				fmt.Printf("ALTER TABLE %s FORCE ROW LEVEL SECURITY;\n", pgQuoteIdentifier(tbl.Table))
			}

			for _, mp := range tbl.MissingPolicies {
				switch mp {
				case "tenant_isolation":
					fmt.Printf(
						"CREATE POLICY tenant_isolation ON %s\n    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);\n",
						pgQuoteIdentifier(tbl.Table),
					)
				case "tenant_insert":
					fmt.Printf(
						"CREATE POLICY tenant_insert ON %s\n    FOR INSERT\n    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);\n",
						pgQuoteIdentifier(tbl.Table),
					)
				case "tenant_update":
					fmt.Printf(
						"CREATE POLICY tenant_update ON %s\n    FOR UPDATE\n    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)\n    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);\n",
						pgQuoteIdentifier(tbl.Table),
					)
				case "tenant_delete":
					fmt.Printf(
						"CREATE POLICY tenant_delete ON %s\n    FOR DELETE\n    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);\n",
						pgQuoteIdentifier(tbl.Table),
					)
				}
			}

			if !tbl.HasTenantIndex {
				fmt.Printf(
					"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_%s_tenant_id ON %s (tenant_id);\n",
					strings.ReplaceAll(tbl.Table, ".", "_"),
					pgQuoteIdentifier(tbl.Table),
				)
			}

			fmt.Println()
		}
	}
}
