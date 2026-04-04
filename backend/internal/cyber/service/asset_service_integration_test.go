package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/classifier"
	cyberconfig "github.com/clario360/platform/internal/cyber/config"
	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/enrichment"
	"github.com/clario360/platform/internal/cyber/metrics"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/cyber/scanner"
	"github.com/clario360/platform/internal/database"
)

var (
	cyberMigrationsOnce sync.Once
	cyberMigrationsErr  error
)

func TestAssetService_Integration_CRUDRelationshipsAndVulnerabilities(t *testing.T) {
	pool := newIntegrationPool(t)
	assetSvc := newIntegrationAssetService(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tenantID := uuid.New()
	userID := uuid.New()
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		cleanupTenantData(t, cleanupCtx, pool, tenantID)
	})

	metadata := json.RawMessage(`{"environment":"prod","source":"integration-test"}`)
	appIP := "10.42.0.10"
	appHost := "payments-api.internal"
	owner := "security-platform"
	department := "platform"
	tags := []string{"prod", "tier1"}

	appAsset, err := assetSvc.CreateAsset(ctx, tenantID, userID, &dto.CreateAssetRequest{
		Name:        "payments-api",
		Type:        model.AssetTypeApplication,
		IPAddress:   &appIP,
		Hostname:    &appHost,
		Owner:       &owner,
		Department:  &department,
		Criticality: model.CriticalityHigh,
		Metadata:    metadata,
		Tags:        tags,
	})
	if err != nil {
		t.Fatalf("CreateAsset(app) error = %v", err)
	}

	dbHost := "payments-db.internal"
	dbAsset, err := assetSvc.CreateAsset(ctx, tenantID, userID, &dto.CreateAssetRequest{
		Name:        "payments-db",
		Type:        model.AssetTypeDatabase,
		Hostname:    &dbHost,
		Criticality: model.CriticalityCritical,
		Tags:        []string{"prod", "data"},
	})
	if err != nil {
		t.Fatalf("CreateAsset(db) error = %v", err)
	}

	if appAsset.DiscoverySource != "manual" {
		t.Fatalf("expected discovery source manual, got %q", appAsset.DiscoverySource)
	}
	if appAsset.Status != model.AssetStatusActive {
		t.Fatalf("expected active status, got %q", appAsset.Status)
	}

	list, err := assetSvc.ListAssets(ctx, tenantID, &dto.AssetListParams{
		Tags:    []string{"prod"},
		Sort:    "name",
		Order:   "asc",
		Page:    1,
		PerPage: 10,
	})
	if err != nil {
		t.Fatalf("ListAssets(all prod) error = %v", err)
	}
	if list.Meta.Total != 2 || len(list.Data) != 2 {
		t.Fatalf("expected 2 assets in list, got total=%d len=%d", list.Meta.Total, len(list.Data))
	}

	appOnly, err := assetSvc.ListAssets(ctx, tenantID, &dto.AssetListParams{
		Types:   []string{string(model.AssetTypeApplication)},
		Tags:    []string{"tier1"},
		Sort:    "name",
		Order:   "asc",
		Page:    1,
		PerPage: 10,
	})
	if err != nil {
		t.Fatalf("ListAssets(application) error = %v", err)
	}
	if appOnly.Meta.Total != 1 || len(appOnly.Data) != 1 {
		t.Fatalf("expected filtered list to contain one asset, got total=%d len=%d", appOnly.Meta.Total, len(appOnly.Data))
	}
	if appOnly.Data[0].ID != appAsset.ID {
		t.Fatalf("expected filtered asset %s, got %s", appAsset.ID, appOnly.Data[0].ID)
	}

	rel, err := assetSvc.CreateRelationship(ctx, tenantID, appAsset.ID, userID, &dto.CreateRelationshipRequest{
		TargetAssetID:    dbAsset.ID.String(),
		RelationshipType: model.RelationshipDependsOn,
		Metadata:         json.RawMessage(`{"protocol":"postgres"}`),
	})
	if err != nil {
		t.Fatalf("CreateRelationship error = %v", err)
	}
	relationship, ok := rel.(*model.AssetRelationship)
	if !ok {
		t.Fatalf("CreateRelationship returned %T, want *model.AssetRelationship", rel)
	}
	if relationship.SourceAssetID != appAsset.ID || relationship.TargetAssetID != dbAsset.ID {
		t.Fatalf("unexpected relationship endpoints: %+v", relationship)
	}

	relsAny, err := assetSvc.ListRelationships(ctx, tenantID, appAsset.ID)
	if err != nil {
		t.Fatalf("ListRelationships error = %v", err)
	}
	rels, ok := relsAny.(map[string][]map[string]any)
	if !ok {
		t.Fatalf("ListRelationships returned %T, want map[string][]map[string]any", relsAny)
	}
	if len(rels["outgoing"]) != 1 || len(rels["incoming"]) != 0 {
		t.Fatalf("expected 1 outgoing and 0 incoming relationships, got %+v", rels)
	}

	remediation := "Patch OpenSSL package"
	vulnAny, err := assetSvc.CreateVulnerability(ctx, tenantID, appAsset.ID, userID, &dto.CreateVulnerabilityRequest{
		CVEID:       testStringPtr("CVE-2026-4242"),
		Title:       "OpenSSL vulnerable version",
		Description: "TLS library is running a vulnerable build",
		Severity:    "critical",
		CVSSScore:   testFloatPtr(9.8),
		Source:      "manual",
		Remediation: &remediation,
	})
	if err != nil {
		t.Fatalf("CreateVulnerability error = %v", err)
	}
	vuln, ok := vulnAny.(*model.Vulnerability)
	if !ok {
		t.Fatalf("CreateVulnerability returned %T, want *model.Vulnerability", vulnAny)
	}
	if vuln.AssetID != appAsset.ID || vuln.Status != "open" {
		t.Fatalf("unexpected vulnerability payload: %+v", vuln)
	}

	vulnsAny, total, err := assetSvc.ListVulnerabilities(ctx, tenantID, appAsset.ID, &dto.VulnerabilityListParams{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("ListVulnerabilities error = %v", err)
	}
	vulns, ok := vulnsAny.([]*model.Vulnerability)
	if !ok {
		t.Fatalf("ListVulnerabilities returned %T, want []*model.Vulnerability", vulnsAny)
	}
	if total != 1 || len(vulns) != 1 {
		t.Fatalf("expected one vulnerability, got total=%d len=%d", total, len(vulns))
	}
	if vulns[0].Severity != "critical" {
		t.Fatalf("expected vulnerability severity critical, got %q", vulns[0].Severity)
	}

	withVulns, err := assetSvc.ListAssets(ctx, tenantID, &dto.AssetListParams{
		HasVulnerabilities: testBoolPtr(true),
		Sort:               "name",
		Order:              "asc",
		Page:               1,
		PerPage:            10,
	})
	if err != nil {
		t.Fatalf("ListAssets(has_vulnerabilities=true) error = %v", err)
	}
	if withVulns.Meta.Total != 1 || len(withVulns.Data) != 1 {
		t.Fatalf("expected one asset with vulnerabilities, got total=%d len=%d", withVulns.Meta.Total, len(withVulns.Data))
	}
	if withVulns.Data[0].ID != appAsset.ID {
		t.Fatalf("expected vulnerable asset %s, got %s", appAsset.ID, withVulns.Data[0].ID)
	}
	if withVulns.Data[0].OpenVulnerabilityCount != 1 {
		t.Fatalf("expected open vulnerability count 1, got %d", withVulns.Data[0].OpenVulnerabilityCount)
	}

	count, err := assetSvc.CountAssets(ctx, tenantID, &dto.AssetListParams{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("CountAssets error = %v", err)
	}
	if count != 2 {
		t.Fatalf("expected CountAssets=2, got %d", count)
	}
}

func newIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dbURL := os.Getenv("CYBER_DB_URL")
	if dbURL == "" {
		dbURL = os.Getenv("TEST_DATABASE_URL")
	}
	if dbURL == "" {
		t.Skip("CYBER_DB_URL or TEST_DATABASE_URL not set; skipping cyber integration test")
	}

	cyberMigrationsOnce.Do(func() {
		cyberMigrationsErr = database.RunMigrations(dbURL, cyberMigrationsPath())
	})
	if cyberMigrationsErr != nil {
		t.Fatalf("run migrations: %v", cyberMigrationsErr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skipf("database unavailable, skipping cyber integration test: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("database unavailable, skipping cyber integration test: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func newIntegrationAssetService(pool *pgxpool.Pool) *AssetService {
	logger := zerolog.Nop()
	assetRepo := repository.NewAssetRepository(pool, logger)
	vulnRepo := repository.NewVulnerabilityRepository(pool, logger)
	relRepo := repository.NewRelationshipRepository(pool, logger)
	scanRepo := repository.NewScanRepository(pool, logger)
	activityRepo := repository.NewActivityRepository(pool, logger)
	enrichSvc := NewEnrichmentService(enrichment.NewPipeline(logger), assetRepo, metrics.New(), logger)

	return NewAssetService(
		assetRepo,
		vulnRepo,
		relRepo,
		scanRepo,
		activityRepo,
		scanner.NewRegistry(),
		classifier.NewAssetClassifier(logger),
		enrichSvc,
		nil,
		metrics.New(),
		&cyberconfig.Config{ClassifyOnCreate: false, ClassifyOnScan: false},
		pool,
		logger,
	)
}

func cleanupTenantData(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID) {
	t.Helper()

	for _, stmt := range []string{
		`DELETE FROM asset_relationships WHERE tenant_id = $1`,
		`DELETE FROM vulnerabilities WHERE tenant_id = $1`,
		`DELETE FROM scan_history WHERE tenant_id = $1`,
		`DELETE FROM assets WHERE tenant_id = $1`,
	} {
		if _, err := pool.Exec(ctx, stmt, tenantID); err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("cleanup tenant data: %v", err)
		}
	}
}

func cyberMigrationsPath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "migrations", "cyber_db"))
}

func testBoolPtr(value bool) *bool {
	return &value
}

func testFloatPtr(value float64) *float64 {
	return &value
}

func testStringPtr(value string) *string {
	return &value
}
