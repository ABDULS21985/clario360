// Package main provides a test-data seeder for the cyber_db database.
// It inserts 500 assets, 200 vulnerabilities, and 50 relationships.
//
// Usage:
//
//	GOWORK=off go run ./cmd/seeder
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/observability"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading config: %v\n", err)
		os.Exit(1)
	}

	logger := observability.NewLogger(
		cfg.Observability.LogLevel,
		cfg.Observability.LogFormat,
		"seeder",
	)

	// Connect to cyber_db
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/cyber_db?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.SSLMode,
	)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to cyber_db")
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Fatal().Err(err).Msg("failed to ping cyber_db")
	}

	logger.Info().Msg("connected to cyber_db — starting seed")

	s := &seeder{pool: pool, rng: rand.New(rand.NewSource(42))}

	// Resolve or create a tenant ID for seed data
	tenantID, err := s.resolveOrCreateTenant(ctx)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to resolve tenant")
	}
	logger.Info().Str("tenant_id", tenantID.String()).Msg("using tenant")

	// Seed assets
	assetIDs, err := s.seedAssets(ctx, tenantID, 500)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to seed assets")
	}
	logger.Info().Int("count", len(assetIDs)).Msg("assets seeded")

	// Seed vulnerabilities
	vulnCount, err := s.seedVulnerabilities(ctx, tenantID, assetIDs, 200)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to seed vulnerabilities")
	}
	logger.Info().Int("count", vulnCount).Msg("vulnerabilities seeded")

	// Seed relationships
	relCount, err := s.seedRelationships(ctx, tenantID, assetIDs, 50)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to seed relationships")
	}
	logger.Info().Int("count", relCount).Msg("relationships seeded")

	logger.Info().Msg("seed complete")
}

type seeder struct {
	pool *pgxpool.Pool
	rng  *rand.Rand
}

// ── helpers ──────────────────────────────────────────────────────────────────

func (s *seeder) resolveOrCreateTenant(ctx context.Context) (uuid.UUID, error) {
	// Try to find an existing tenant in platform_core.tenants
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT id FROM tenants LIMIT 1`).Scan(&id)
	if err == nil {
		return id, nil
	}
	// Create a seed tenant directly in this DB if no platform_core linkage exists
	return uuid.New(), nil
}

func (s *seeder) pick(slice []string) string {
	return slice[s.rng.Intn(len(slice))]
}

func (s *seeder) pickN(slice []string, n int) []string {
	cp := make([]string, len(slice))
	copy(cp, slice)
	s.rng.Shuffle(len(cp), func(i, j int) { cp[i], cp[j] = cp[j], cp[i] })
	if n > len(cp) {
		n = len(cp)
	}
	return cp[:n]
}

func (s *seeder) randIP() string {
	return fmt.Sprintf("10.%d.%d.%d", s.rng.Intn(255), s.rng.Intn(255), s.rng.Intn(254)+1)
}

func (s *seeder) randMAC() string {
	b := make([]byte, 6)
	s.rng.Read(b)
	b[0] = (b[0] | 0x02) & 0xfe // local, unicast
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", b[0], b[1], b[2], b[3], b[4], b[5])
}

func (s *seeder) randPast(maxDaysAgo int) time.Time {
	days := s.rng.Intn(maxDaysAgo) + 1
	hours := s.rng.Intn(24)
	mins := s.rng.Intn(60)
	return time.Now().UTC().Add(-time.Duration(days)*24*time.Hour - time.Duration(hours)*time.Hour - time.Duration(mins)*time.Minute)
}

// ── assets ───────────────────────────────────────────────────────────────────

var assetTypes = []string{"server", "endpoint", "network_device", "cloud_resource", "iot_device", "application", "database", "container"}
var criticalities = []string{"critical", "high", "medium", "low"}
var assetStatuses = []string{"active", "inactive", "decommissioned"}
var discoverySources = []string{"manual", "network_scan", "cloud_scan", "agent", "import"}

var osList = []string{"Ubuntu 22.04", "Ubuntu 20.04", "CentOS 8", "RHEL 9", "Debian 12", "Windows Server 2022", "Windows Server 2019", "Alpine 3.18", "macOS 14", "FreeBSD 14"}
var departments = []string{"Engineering", "IT Operations", "Finance", "HR", "Security", "DevOps", "Data Science", "Product"}
var locations = []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1", "on-prem-dc1", "on-prem-dc2", "azure-east-us", "gcp-us-central1"}

var hostnamePrefixes = []string{"web", "db", "api", "cache", "lb", "app", "worker", "monitor", "backup", "proxy"}
var hostnameSuffixes = []string{"prod", "staging", "dev", "qa", "001", "002", "003"}

var tagPool = []string{
	"production", "staging", "development", "critical", "pci-scope", "hipaa-scope",
	"internet-facing", "internal", "dmz", "legacy", "containerized", "high-availability",
	"backup-enabled", "monitored", "encrypted", "patched",
}

func (s *seeder) seedAssets(ctx context.Context, tenantID uuid.UUID, count int) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, count)
	now := time.Now().UTC()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for i := 0; i < count; i++ {
		id := uuid.New()
		assetType := s.pick(assetTypes)
		criticality := s.pick(criticalities)
		status := s.pick(assetStatuses)
		discoverySource := s.pick(discoverySources)

		hostname := fmt.Sprintf("%s-%s-%03d", s.pick(hostnamePrefixes), s.pick(hostnameSuffixes), i+1)
		ip := s.randIP()
		mac := s.randMAC()
		os_ := s.pick(osList)
		dept := s.pick(departments)
		location := s.pick(locations)
		discoveredAt := s.randPast(365)
		lastSeenAt := s.randPast(30)

		// Random subset of 0–4 tags
		numTags := s.rng.Intn(5)
		tags := s.pickN(tagPool, numTags)

		meta, _ := json.Marshal(map[string]interface{}{
			"open_ports":  randomPorts(s.rng),
			"environment": s.pick([]string{"prod", "staging", "dev", "qa"}),
		})

		_, err = tx.Exec(ctx, `
			INSERT INTO assets (
				id, tenant_id, name, type, ip_address, hostname, mac_address,
				os, criticality, status, discovered_at, last_seen_at,
				discovery_source, department, location, metadata, tags,
				created_at, updated_at
			) VALUES (
				$1, $2, $3, $4::asset_type, $5, $6, $7,
				$8, $9::asset_criticality, $10::asset_status, $11, $12,
				$13, $14, $15, $16, $17,
				$18, $18
			) ON CONFLICT DO NOTHING`,
			id, tenantID,
			fmt.Sprintf("Asset-%s-%04d", assetType, i+1),
			assetType, ip, hostname, mac,
			os_, criticality, status, discoveredAt, lastSeenAt,
			discoverySource, dept, location, meta, tags,
			now,
		)
		if err != nil {
			return nil, fmt.Errorf("insert asset %d: %w", i, err)
		}
		ids = append(ids, id)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit assets tx: %w", err)
	}
	return ids, nil
}

func randomPorts(rng *rand.Rand) []int {
	allPorts := []int{22, 80, 443, 3306, 5432, 6379, 8080, 8443, 9200, 27017}
	n := rng.Intn(5)
	rng.Shuffle(len(allPorts), func(i, j int) { allPorts[i], allPorts[j] = allPorts[j], allPorts[i] })
	return allPorts[:n]
}

// ── vulnerabilities ───────────────────────────────────────────────────────────

var severities = []string{"critical", "high", "medium", "low"}
var vulnStatuses = []string{"open", "remediated", "accepted", "false_positive"}

var cvePool = []string{
	"CVE-2024-1234", "CVE-2024-2345", "CVE-2024-3456", "CVE-2024-4567", "CVE-2024-5678",
	"CVE-2023-12345", "CVE-2023-23456", "CVE-2023-34567", "CVE-2023-45678", "CVE-2023-56789",
	"CVE-2022-22965", "CVE-2022-0847", "CVE-2022-30190", "CVE-2021-44228", "CVE-2021-34527",
	"CVE-2020-14882", "CVE-2020-1472", "CVE-2019-0708", "CVE-2019-11510", "CVE-2018-13379",
}

var vulnTitles = []string{
	"Remote Code Execution via Deserialization",
	"SQL Injection in Login Endpoint",
	"Cross-Site Scripting (Reflected)",
	"Privilege Escalation via Misconfigured SUID Binary",
	"Unencrypted Sensitive Data in Transit",
	"Missing Authentication on Admin Endpoint",
	"Path Traversal in File Upload Handler",
	"XML External Entity (XXE) Injection",
	"Server-Side Request Forgery (SSRF)",
	"Insecure Direct Object Reference",
	"Log4Shell Remote Code Execution",
	"PrintNightmare Privilege Escalation",
	"ProxyShell Exchange Vulnerability",
	"BlueKeep Remote Desktop Vulnerability",
	"Zerologon Netlogon Elevation of Privilege",
}

func (s *seeder) seedVulnerabilities(ctx context.Context, tenantID uuid.UUID, assetIDs []uuid.UUID, count int) (int, error) {
	now := time.Now().UTC()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	inserted := 0
	for i := 0; i < count; i++ {
		id := uuid.New()
		assetID := assetIDs[s.rng.Intn(len(assetIDs))]
		cve := s.pick(cvePool)
		title := s.pick(vulnTitles)
		severity := s.pick(severities)
		status := s.pick(vulnStatuses)
		discoveredAt := s.randPast(180)

		var cvssScore *float64
		switch severity {
		case "critical":
			v := 9.0 + s.rng.Float64()*1.0
			cvssScore = &v
		case "high":
			v := 7.0 + s.rng.Float64()*2.0
			cvssScore = &v
		case "medium":
			v := 4.0 + s.rng.Float64()*3.0
			cvssScore = &v
		case "low":
			v := 1.0 + s.rng.Float64()*3.0
			cvssScore = &v
		}

		description := fmt.Sprintf("%s affecting %s component. CVE score: %.1f.", title, assetType(s.rng), *cvssScore)
		remediation := "Apply vendor-provided patch or upgrade to the latest version."

		_, err = tx.Exec(ctx, `
			INSERT INTO vulnerabilities (
				id, tenant_id, asset_id, cve_id, title, description,
				severity, cvss_score, status, discovered_at, remediation,
				created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				$7::severity_level, $8, $9::vulnerability_status, $10, $11,
				$12, $12
			) ON CONFLICT DO NOTHING`,
			id, tenantID, assetID, cve, title, description,
			severity, cvssScore, status, discoveredAt, remediation,
			now,
		)
		if err != nil {
			return inserted, fmt.Errorf("insert vulnerability %d: %w", i, err)
		}
		inserted++
	}

	if err := tx.Commit(ctx); err != nil {
		return inserted, fmt.Errorf("commit vulns tx: %w", err)
	}
	return inserted, nil
}

func assetType(rng *rand.Rand) string {
	components := []string{"kernel", "libc", "openssl", "nginx", "apache", "log4j", "spring", "struts", "curl", "libpng"}
	return components[rng.Intn(len(components))]
}

// ── relationships ─────────────────────────────────────────────────────────────

var relationshipTypes = []string{"hosts", "runs_on", "connects_to", "depends_on", "managed_by", "backs_up", "load_balances"}

func (s *seeder) seedRelationships(ctx context.Context, tenantID uuid.UUID, assetIDs []uuid.UUID, count int) (int, error) {
	now := time.Now().UTC()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	seen := make(map[[2]uuid.UUID]bool, count)
	inserted := 0
	attempts := 0

	for inserted < count && attempts < count*10 {
		attempts++
		idxA := s.rng.Intn(len(assetIDs))
		idxB := s.rng.Intn(len(assetIDs))
		if idxA == idxB {
			continue
		}
		src := assetIDs[idxA]
		tgt := assetIDs[idxB]
		key := [2]uuid.UUID{src, tgt}
		if seen[key] {
			continue
		}
		seen[key] = true

		relType := s.pick(relationshipTypes)
		meta, _ := json.Marshal(map[string]string{"seeded": "true"})

		_, err := tx.Exec(ctx, `
			INSERT INTO asset_relationships (
				id, tenant_id, source_asset_id, target_asset_id,
				relationship_type, metadata, created_at
			) VALUES ($1, $2, $3, $4, $5::text, $6, $7)
			ON CONFLICT DO NOTHING`,
			uuid.New(), tenantID, src, tgt,
			relType, meta, now,
		)
		if err != nil {
			return inserted, fmt.Errorf("insert relationship: %w", err)
		}
		inserted++
	}

	if err := tx.Commit(ctx); err != nil {
		return inserted, fmt.Errorf("commit relationships tx: %w", err)
	}
	return inserted, nil
}
