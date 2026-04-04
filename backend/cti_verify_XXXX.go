package main

import (
    "context"
    "fmt"
    "os"
    "strings"

    "github.com/jackc/pgx/v5/pgxpool"
)

func mustExec(ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) {
    if _, err := pool.Exec(ctx, sql, args...); err != nil {
        fmt.Fprintf(os.Stderr, "exec failed: %v\nSQL: %s\n", err, sql)
        os.Exit(1)
    }
}

func main() {
    ctx := context.Background()
    dsn := os.Getenv("CYBER_DB_URL")
    if strings.TrimSpace(dsn) == "" {
        fmt.Fprintln(os.Stderr, "CYBER_DB_URL is required")
        os.Exit(1)
    }
    tenantID := "aaaaaaaa-0000-0000-0000-000000000001"

    pool, err := pgxpool.New(ctx, dsn)
    if err != nil {
        fmt.Fprintf(os.Stderr, "pool create failed: %v\n", err)
        os.Exit(1)
    }
    defer pool.Close()

    if err := pool.Ping(ctx); err != nil {
        fmt.Fprintf(os.Stderr, "ping failed: %v\n", err)
        os.Exit(1)
    }

    mustExec(ctx, pool, `SELECT set_config('app.current_tenant_id', $1, false)`, tenantID)

    fmt.Println("== Record Counts ==")
    countQueries := []struct{ label, sql string }{
        {"cti_threat_severity_levels", `SELECT count(*) FROM cti_threat_severity_levels`},
        {"cti_threat_categories", `SELECT count(*) FROM cti_threat_categories`},
        {"cti_geographic_regions", `SELECT count(*) FROM cti_geographic_regions`},
        {"cti_industry_sectors", `SELECT count(*) FROM cti_industry_sectors`},
        {"cti_data_sources", `SELECT count(*) FROM cti_data_sources`},
        {"cti_threat_actors", `SELECT count(*) FROM cti_threat_actors WHERE deleted_at IS NULL`},
        {"cti_campaigns", `SELECT count(*) FROM cti_campaigns WHERE deleted_at IS NULL`},
        {"cti_threat_events", `SELECT count(*) FROM cti_threat_events WHERE deleted_at IS NULL`},
        {"cti_campaign_iocs", `SELECT count(*) FROM cti_campaign_iocs`},
        {"cti_campaign_events", `SELECT count(*) FROM cti_campaign_events`},
        {"cti_monitored_brands", `SELECT count(*) FROM cti_monitored_brands`},
        {"cti_brand_abuse_incidents", `SELECT count(*) FROM cti_brand_abuse_incidents WHERE deleted_at IS NULL`},
        {"cti_geo_threat_summary", `SELECT count(*) FROM cti_geo_threat_summary`},
        {"cti_sector_threat_summary", `SELECT count(*) FROM cti_sector_threat_summary`},
        {"cti_executive_snapshot", `SELECT count(*) FROM cti_executive_snapshot`},
    }
    for _, q := range countQueries {
        var n int
        if err := pool.QueryRow(ctx, q.sql).Scan(&n); err != nil {
            fmt.Fprintf(os.Stderr, "count query failed for %s: %v\n", q.label, err)
            os.Exit(1)
        }
        fmt.Printf("%s: %d\n", q.label, n)
    }

    fmt.Println("\n== Severity Distribution ==")
    rows, err := pool.Query(ctx, `
        SELECT sl.code, count(*)
        FROM cti_threat_events e
        JOIN cti_threat_severity_levels sl ON e.severity_id = sl.id
        WHERE e.deleted_at IS NULL
        GROUP BY sl.code
        ORDER BY count(*) DESC, sl.code ASC`)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    defer rows.Close()
    for rows.Next() {
        var code string
        var n int
        if err := rows.Scan(&code, &n); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
        fmt.Printf("%s: %d\n", code, n)
    }
    if err := rows.Err(); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }

    fmt.Println("\n== Top Origin Countries ==")
    rows, err = pool.Query(ctx, `
        SELECT origin_country_code, count(*)
        FROM cti_threat_events
        WHERE deleted_at IS NULL
        GROUP BY origin_country_code
        ORDER BY count(*) DESC, origin_country_code ASC
        LIMIT 10`)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    defer rows.Close()
    for rows.Next() {
        var code *string
        var n int
        if err := rows.Scan(&code, &n); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
        val := "NULL"
        if code != nil { val = *code }
        fmt.Printf("%s: %d\n", val, n)
    }
    if err := rows.Err(); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }

    fmt.Println("\n== Campaign Event Linkage ==")
    rows, err = pool.Query(ctx, `
        SELECT c.name, count(ce.event_id)
        FROM cti_campaigns c
        LEFT JOIN cti_campaign_events ce ON c.id = ce.campaign_id
        WHERE c.deleted_at IS NULL
        GROUP BY c.name
        ORDER BY count(ce.event_id) DESC, c.name ASC`)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    defer rows.Close()
    for rows.Next() {
        var name string
        var n int
        if err := rows.Scan(&name, &n); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
        fmt.Printf("%s: %d\n", name, n)
    }
    if err := rows.Err(); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }

    fmt.Println("\n== Executive Snapshot ==")
    var (
        total24h int
        total7d int
        total30d int
        activeCampaigns int
        criticalCampaigns int
        totalIOCs int
        brandCritical int
        brandTotal int
        topCountry *string
        riskScore float64
        trend string
        trendPct float64
    )
    err = pool.QueryRow(ctx, `
        SELECT total_events_24h, total_events_7d, total_events_30d,
               active_campaigns_count, critical_campaigns_count, total_iocs,
               brand_abuse_critical_count, brand_abuse_total_count,
               top_threat_origin_country, risk_score_overall, trend_direction, trend_percentage
        FROM cti_executive_snapshot
        WHERE tenant_id = $1`, tenantID).Scan(
        &total24h, &total7d, &total30d,
        &activeCampaigns, &criticalCampaigns, &totalIOCs,
        &brandCritical, &brandTotal,
        &topCountry, &riskScore, &trend, &trendPct,
    )
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    topCountryVal := "NULL"
    if topCountry != nil { topCountryVal = *topCountry }
    fmt.Printf("total_events_24h: %d\n", total24h)
    fmt.Printf("total_events_7d: %d\n", total7d)
    fmt.Printf("total_events_30d: %d\n", total30d)
    fmt.Printf("active_campaigns_count: %d\n", activeCampaigns)
    fmt.Printf("critical_campaigns_count: %d\n", criticalCampaigns)
    fmt.Printf("total_iocs: %d\n", totalIOCs)
    fmt.Printf("brand_abuse_critical_count: %d\n", brandCritical)
    fmt.Printf("brand_abuse_total_count: %d\n", brandTotal)
    fmt.Printf("top_threat_origin_country: %s\n", topCountryVal)
    fmt.Printf("risk_score_overall: %.2f\n", riskScore)
    fmt.Printf("trend_direction: %s\n", trend)
    fmt.Printf("trend_percentage: %.2f\n", trendPct)

    fmt.Println("\n== RLS Check ==")
    mustExec(ctx, pool, `SELECT set_config('app.current_tenant_id', $1, false)`, "00000000-0000-0000-0000-000000000000")
    var wrongTenantCount int
    if err := pool.QueryRow(ctx, `SELECT count(*) FROM cti_threat_events`).Scan(&wrongTenantCount); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
    fmt.Printf("cti_threat_events with wrong tenant: %d\n", wrongTenantCount)

    fmt.Println("\n== Performance Baseline ==")
    mustExec(ctx, pool, `SELECT set_config('app.current_tenant_id', $1, false)`, tenantID)
    rows, err = pool.Query(ctx, `
        EXPLAIN ANALYZE
        SELECT *
        FROM cti_threat_events
        WHERE tenant_id = $1
          AND first_seen_at > NOW() - INTERVAL '24 hours'
        ORDER BY first_seen_at DESC
        LIMIT 50`, tenantID)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    defer rows.Close()
    for rows.Next() {
        var line string
        if err := rows.Scan(&line); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
        fmt.Println(line)
    }
    if err := rows.Err(); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}

