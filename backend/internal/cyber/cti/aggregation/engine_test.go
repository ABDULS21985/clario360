package aggregation

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/database"
)

var (
	aggregationMigrationsOnce sync.Once
	aggregationMigrationsErr  error
)

func TestClassifyTrend(t *testing.T) {
	tests := []struct {
		current  int64
		previous int64
		wantDir  string
	}{
		{100, 50, "increasing"},
		{50, 100, "decreasing"},
		{100, 95, "stable"},
		{0, 0, "stable"},
		{10, 0, "increasing"},
		{0, 10, "decreasing"},
	}
	for _, tt := range tests {
		dir, _ := classifyTrend(tt.current, tt.previous)
		if dir != tt.wantDir {
			t.Errorf("classifyTrend(%d, %d) = %q, want %q", tt.current, tt.previous, dir, tt.wantDir)
		}
	}
}

func TestComputeRiskScore(t *testing.T) {
	score := computeRiskScore(200, 20, 10, 10)
	if score != 100 {
		t.Errorf("max risk score: want 100, got %f", score)
	}

	score = computeRiskScore(0, 0, 0, 0)
	if score != 0 {
		t.Errorf("zero risk score: want 0, got %f", score)
	}

	score = computeRiskScore(50, 5, 2, 2)
	if score < 30 || score > 70 {
		t.Errorf("moderate risk score out of expected range: %f", score)
	}
}

func TestDefaultPeriods(t *testing.T) {
	if len(DefaultPeriods) != 4 {
		t.Fatalf("expected 4 periods, got %d", len(DefaultPeriods))
	}
	expected := []string{"24h", "7d", "30d", "90d"}
	for i, p := range DefaultPeriods {
		if p.Label != expected[i] {
			t.Errorf("period %d: want %q, got %q", i, expected[i], p.Label)
		}
		if p.Duration <= 0 {
			t.Errorf("period %q has non-positive duration", p.Label)
		}
	}
}

func TestDefaultAggregationConfig(t *testing.T) {
	cfg := normalizeConfig(Config{})
	if len(cfg.Periods) != 4 {
		t.Fatalf("expected 4 default periods, got %d", len(cfg.Periods))
	}
	if cfg.MaxConcurrency != 5 {
		t.Fatalf("expected default max concurrency 5, got %d", cfg.MaxConcurrency)
	}
}

func TestDefaultScheduleConfig(t *testing.T) {
	c := DefaultScheduleConfig
	if c.FullInterval != 5*time.Minute {
		t.Errorf("FullInterval: want 5m, got %v", c.FullInterval)
	}
	if c.ExecutiveInterval != 2*time.Minute {
		t.Errorf("ExecutiveInterval: want 2m, got %v", c.ExecutiveInterval)
	}
	if c.CleanupInterval != 1*time.Hour {
		t.Errorf("CleanupInterval: want 1h, got %v", c.CleanupInterval)
	}
	if c.MaxAggregationAge != 7*24*time.Hour {
		t.Errorf("MaxAggregationAge: want 7d, got %v", c.MaxAggregationAge)
	}
}

func TestEngineRunFullAggregationIntegration(t *testing.T) {
	pool := aggregationTestPool(t)

	tenantID := uuid.New()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cleanupAggregationTenant(t, ctx, pool, tenantID)
	t.Cleanup(func() { cleanupAggregationTenant(t, context.Background(), pool, tenantID) })

	if err := seedAggregationTenant(ctx, pool, tenantID); err != nil {
		t.Fatalf("seed aggregation tenant: %v", err)
	}

	engine := NewEngineWithConfig(pool, prometheus.NewRegistry(), zerolog.Nop(), DefaultConfig)
	if err := engine.RunFullAggregation(ctx, tenantID.String()); err != nil {
		t.Fatalf("RunFullAggregation: %v", err)
	}

	var geoCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM cti_geo_threat_summary WHERE tenant_id = $1`, tenantID).Scan(&geoCount); err != nil {
		t.Fatalf("count geo summary: %v", err)
	}
	if geoCount != 4 {
		t.Fatalf("expected 4 geo summary rows, got %d", geoCount)
	}

	var topThreatType string
	if err := pool.QueryRow(ctx, `
		SELECT top_threat_type
		FROM cti_geo_threat_summary
		WHERE tenant_id = $1
		ORDER BY period_end DESC
		LIMIT 1`, tenantID).Scan(&topThreatType); err != nil {
		t.Fatalf("load geo top threat type: %v", err)
	}
	if topThreatType != "Phishing" {
		t.Fatalf("expected top threat type Phishing, got %q", topThreatType)
	}

	var sectorCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM cti_sector_threat_summary WHERE tenant_id = $1`, tenantID).Scan(&sectorCount); err != nil {
		t.Fatalf("count sector summary: %v", err)
	}
	if sectorCount != 4 {
		t.Fatalf("expected 4 sector summary rows, got %d", sectorCount)
	}

	var (
		events24h         int
		events7d          int
		events30d         int
		activeCampaigns   int
		criticalCampaigns int
		totalIOCs         int
		brandCritical     int
		brandTotal        int
		trendDirection    string
	)
	if err := pool.QueryRow(ctx, `
		SELECT total_events_24h, total_events_7d, total_events_30d,
		       active_campaigns_count, critical_campaigns_count, total_iocs,
		       brand_abuse_critical_count, brand_abuse_total_count, trend_direction
		FROM cti_executive_snapshot
		WHERE tenant_id = $1`, tenantID).Scan(
		&events24h, &events7d, &events30d,
		&activeCampaigns, &criticalCampaigns, &totalIOCs,
		&brandCritical, &brandTotal, &trendDirection,
	); err != nil {
		t.Fatalf("load executive snapshot: %v", err)
	}

	if events24h != 1 || events7d != 2 || events30d != 2 {
		t.Fatalf("unexpected event counts: 24h=%d 7d=%d 30d=%d", events24h, events7d, events30d)
	}
	if activeCampaigns != 1 || criticalCampaigns != 1 {
		t.Fatalf("unexpected campaign counts: active=%d critical=%d", activeCampaigns, criticalCampaigns)
	}
	if totalIOCs != 7 {
		t.Fatalf("expected total_iocs 7, got %d", totalIOCs)
	}
	if brandCritical != 1 || brandTotal != 1 {
		t.Fatalf("unexpected brand counts: critical=%d total=%d", brandCritical, brandTotal)
	}
	if trendDirection != "increasing" {
		t.Fatalf("expected trend_direction increasing, got %q", trendDirection)
	}
}

func TestEngineGetActiveTenantsIncludesNonEventTenants(t *testing.T) {
	pool := aggregationTestPool(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	eventTenantID := uuid.New()
	brandOnlyTenantID := uuid.New()
	cleanupAggregationTenant(t, ctx, pool, eventTenantID)
	cleanupAggregationTenant(t, ctx, pool, brandOnlyTenantID)
	t.Cleanup(func() {
		cleanupAggregationTenant(t, context.Background(), pool, eventTenantID)
		cleanupAggregationTenant(t, context.Background(), pool, brandOnlyTenantID)
	})

	if err := seedAggregationTenant(ctx, pool, eventTenantID); err != nil {
		t.Fatalf("seed event tenant: %v", err)
	}
	if err := seedBrandOnlyTenant(ctx, pool, brandOnlyTenantID); err != nil {
		t.Fatalf("seed brand-only tenant: %v", err)
	}

	engine := NewEngineWithConfig(pool, prometheus.NewRegistry(), zerolog.Nop(), DefaultConfig)
	tenantIDs, err := engine.GetActiveTenants(ctx)
	if err != nil {
		t.Fatalf("GetActiveTenants: %v", err)
	}

	seen := make(map[string]struct{}, len(tenantIDs))
	for _, tenantID := range tenantIDs {
		seen[tenantID] = struct{}{}
	}
	if _, ok := seen[eventTenantID.String()]; !ok {
		t.Fatalf("event tenant %s missing from GetActiveTenants", eventTenantID)
	}
	if _, ok := seen[brandOnlyTenantID.String()]; !ok {
		t.Fatalf("brand-only tenant %s missing from GetActiveTenants", brandOnlyTenantID)
	}
}

func aggregationTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dbURL := os.Getenv("CYBER_DB_URL")
	if dbURL == "" {
		dbURL = os.Getenv("TEST_DATABASE_URL")
	}
	if dbURL == "" {
		t.Skip("CYBER_DB_URL or TEST_DATABASE_URL not set; skipping CTI aggregation integration test")
	}

	aggregationMigrationsOnce.Do(func() {
		aggregationMigrationsErr = database.RunMigrations(dbURL, aggregationMigrationsPath())
	})
	if aggregationMigrationsErr != nil {
		t.Fatalf("run migrations: %v", aggregationMigrationsErr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skipf("database unavailable, skipping CTI aggregation integration test: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("database unavailable, skipping CTI aggregation integration test: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func aggregationMigrationsPath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "migrations", "cyber_db"))
}

func cleanupAggregationTenant(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) {
	t.Helper()

	for _, stmt := range []string{
		`DELETE FROM cti_executive_snapshot WHERE tenant_id = $1`,
		`DELETE FROM cti_sector_threat_summary WHERE tenant_id = $1`,
		`DELETE FROM cti_geo_threat_summary WHERE tenant_id = $1`,
		`DELETE FROM cti_brand_abuse_incidents WHERE tenant_id = $1`,
		`DELETE FROM cti_monitored_brands WHERE tenant_id = $1`,
		`DELETE FROM cti_campaign_events WHERE tenant_id = $1`,
		`DELETE FROM cti_campaign_iocs WHERE tenant_id = $1`,
		`DELETE FROM cti_threat_event_tags WHERE tenant_id = $1`,
		`DELETE FROM cti_threat_events WHERE tenant_id = $1`,
		`DELETE FROM cti_campaigns WHERE tenant_id = $1`,
		`DELETE FROM cti_threat_actors WHERE tenant_id = $1`,
		`DELETE FROM cti_data_sources WHERE tenant_id = $1`,
		`DELETE FROM cti_industry_sectors WHERE tenant_id = $1`,
		`DELETE FROM cti_geographic_regions WHERE tenant_id = $1`,
		`DELETE FROM cti_threat_categories WHERE tenant_id = $1`,
		`DELETE FROM cti_threat_severity_levels WHERE tenant_id = $1`,
	} {
		if _, err := pool.Exec(ctx, stmt, tenantID); err != nil {
			t.Fatalf("cleanup tenant data: %v", err)
		}
	}
}

func seedAggregationTenant(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenantContext(ctx, tx, tenantID.String()); err != nil {
		return err
	}

	criticalID := uuid.New()
	highID := uuid.New()
	mediumID := uuid.New()
	lowID := uuid.New()
	categoryID := uuid.New()
	sectorID := uuid.New()
	regionID := uuid.New()
	brandID := uuid.New()

	if _, err := tx.Exec(ctx, `
		INSERT INTO cti_threat_severity_levels (id, tenant_id, code, label, color_hex, sort_order)
		VALUES
			($1, $5, 'critical', 'Critical', '#dc2626', 1),
			($2, $5, 'high', 'High', '#ea580c', 2),
			($3, $5, 'medium', 'Medium', '#ca8a04', 3),
			($4, $5, 'low', 'Low', '#2563eb', 4)`,
		criticalID, highID, mediumID, lowID, tenantID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO cti_threat_categories (id, tenant_id, code, label, description)
		VALUES ($1, $2, 'phishing', 'Phishing', 'Phishing campaigns')`,
		categoryID, tenantID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO cti_geographic_regions (id, tenant_id, code, label, latitude, longitude, iso_country_code)
		VALUES ($1, $2, 'ng', 'Nigeria', 9.0820, 8.6753, 'NGA')`,
		regionID, tenantID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO cti_industry_sectors (id, tenant_id, code, label, description, naics_code)
		VALUES ($1, $2, 'technology', 'Technology', 'Technology sector', '5112')`,
		sectorID, tenantID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO cti_monitored_brands (id, tenant_id, brand_name, domain_pattern, keywords)
		VALUES ($1, $2, 'Clario Test', 'clario.example', ARRAY['clario'])`,
		brandID, tenantID); err != nil {
		return err
	}

	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `
		INSERT INTO cti_threat_events (
			id, tenant_id, event_type, title, severity_id, category_id,
			origin_country_code, origin_city, origin_latitude, origin_longitude, origin_region_id,
			target_sector_id, ioc_type, ioc_value, first_seen_at, last_seen_at, resolved_at
		) VALUES
			($1, $9, 'attack_attempt', 'Credential phishing campaign', $2, $4, 'ng', 'Lagos', 6.5244, 3.3792, $5, $6, 'domain', 'phish.example', $7, $8, NULL),
			($3, $9, 'indicator_sighting', 'Phishing IOC follow-up', $10, $4, 'ng', 'Lagos', 6.5244, 3.3792, $5, $6, 'url', 'https://phish.example/login', $11, $12, $13)`,
		uuid.New(), criticalID, uuid.New(), categoryID, regionID, sectorID,
		now.Add(-2*time.Hour), now.Add(-90*time.Minute), tenantID, highID,
		now.Add(-72*time.Hour), now.Add(-70*time.Hour), now.Add(-24*time.Hour)); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO cti_campaigns (
			id, tenant_id, campaign_code, name, status, severity_id,
			target_sectors, first_seen_at, ioc_count
		) VALUES ($1, $2, 'C-TEST-001', 'Test Campaign', 'active', $3, ARRAY[$4]::uuid[], $5, 7)`,
		uuid.New(), tenantID, criticalID, sectorID, now.Add(-5*24*time.Hour)); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO cti_brand_abuse_incidents (
			id, tenant_id, brand_id, malicious_domain, abuse_type, risk_level,
			takedown_status, first_detected_at, last_detected_at
		) VALUES ($1, $2, $3, 'brand-phish.example', 'credential_phishing', 'critical', 'detected', $4, $5)`,
		uuid.New(), tenantID, brandID, now.Add(-6*time.Hour), now.Add(-5*time.Hour)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func seedBrandOnlyTenant(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenantContext(ctx, tx, tenantID.String()); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO cti_monitored_brands (id, tenant_id, brand_name, domain_pattern, keywords)
		VALUES ($1, $2, 'Brand Only Tenant', 'brand-only.example', ARRAY['brand-only'])`,
		uuid.New(), tenantID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
