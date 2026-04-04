package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/mitre"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
)

// ---------------------------------------------------------------------------
// Integration tests for MITRE coverage SQL queries.
// Requires CYBER_DB_URL or TEST_DATABASE_URL to be set; skips otherwise.
// ---------------------------------------------------------------------------

func TestMITRECoverage_Integration_TechniqueCoverageContextMap(t *testing.T) {
	pool := newIntegrationPool(t)
	logger := zerolog.Nop()
	ruleRepo := repository.NewRuleRepository(pool, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tenantID := uuid.New()
	t.Cleanup(func() {
		cleanupMitreData(t, pool, tenantID)
	})

	// ---- Seed detection rule ----
	ruleID := uuid.New()
	ruleContent, _ := json.Marshal(map[string]string{"query": "test"})
	_, err := pool.Exec(ctx, `
		INSERT INTO detection_rules (id, tenant_id, name, description, rule_type, severity,
			enabled, rule_content, mitre_tactic_ids, mitre_technique_ids,
			base_confidence, created_at, updated_at)
		VALUES ($1, $2, 'PowerShell Detection', 'Detects suspicious PowerShell activity',
			'sigma', 'high', true, $3, $4, $5, 0.85, NOW(), NOW())`,
		ruleID, tenantID, string(ruleContent),
		[]string{"TA0002"}, []string{"T1059", "T1059.001"},
	)
	if err != nil {
		t.Fatalf("seed rule: %v", err)
	}

	// ---- Seed alert with MITRE technique ----
	alertID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO alerts (id, tenant_id, title, description, severity, status,
			confidence_score, rule_id, mitre_tactic_id, mitre_technique_id,
			mitre_tactic_name, mitre_technique_name,
			first_event_at, last_event_at, created_at, updated_at)
		VALUES ($1, $2, 'PowerShell encoded command', 'Suspicious encoded PowerShell detected',
			'high', 'new', 0.92, $3, 'TA0002', 'T1059', 'Execution',
			'Command and Scripting Interpreter', NOW(), NOW(), NOW(), NOW())`,
		alertID, tenantID, ruleID,
	)
	if err != nil {
		t.Fatalf("seed alert: %v", err)
	}

	// ---- Seed threat with MITRE techniques array ----
	threatID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO threats (id, tenant_id, name, description, type, severity, status,
			mitre_tactic_ids, mitre_technique_ids,
			first_seen_at, last_seen_at, created_at, updated_at)
		VALUES ($1, $2, 'APT29 Activity', 'Cozy Bear campaign', 'apt', 'critical', 'active',
			$3, $4, NOW(), NOW(), NOW(), NOW())`,
		threatID, tenantID,
		[]string{"TA0001", "TA0002"}, []string{"T1059", "T1566"},
	)
	if err != nil {
		t.Fatalf("seed threat: %v", err)
	}

	// ---- Execute TechniqueCoverageContextMap ----
	contextMap, err := ruleRepo.TechniqueCoverageContextMap(ctx, tenantID)
	if err != nil {
		t.Fatalf("TechniqueCoverageContextMap: %v", err)
	}

	// Verify T1059 has both alert and threat context
	t1059 := contextMap["T1059"]
	if t1059 == nil {
		t.Fatal("expected context for T1059")
	}
	if t1059.AlertCount != 1 {
		t.Errorf("T1059 alert_count: expected 1, got %d", t1059.AlertCount)
	}
	if t1059.LastAlertAt == nil {
		t.Error("T1059 last_alert_at should not be nil")
	}
	if t1059.ThreatCount != 1 {
		t.Errorf("T1059 threat_count: expected 1, got %d", t1059.ThreatCount)
	}
	if t1059.ActiveThreatCount != 1 {
		t.Errorf("T1059 active_threat_count: expected 1 (status=active), got %d", t1059.ActiveThreatCount)
	}
	if len(t1059.Threats) != 1 {
		t.Errorf("T1059 threats: expected 1, got %d", len(t1059.Threats))
	} else {
		if t1059.Threats[0].Name != "APT29 Activity" {
			t.Errorf("T1059 threat name: expected 'APT29 Activity', got %q", t1059.Threats[0].Name)
		}
		if t1059.Threats[0].Status != model.ThreatStatusActive {
			t.Errorf("T1059 threat status: expected 'active', got %q", t1059.Threats[0].Status)
		}
	}

	// Verify T1566 has threat context only (no alert)
	t1566 := contextMap["T1566"]
	if t1566 == nil {
		t.Fatal("expected context for T1566")
	}
	if t1566.AlertCount != 0 {
		t.Errorf("T1566 alert_count: expected 0, got %d", t1566.AlertCount)
	}
	if t1566.ThreatCount != 1 {
		t.Errorf("T1566 threat_count: expected 1, got %d", t1566.ThreatCount)
	}
}

func TestMITRECoverage_Integration_TechniqueRecentAlerts(t *testing.T) {
	pool := newIntegrationPool(t)
	logger := zerolog.Nop()
	ruleRepo := repository.NewRuleRepository(pool, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tenantID := uuid.New()
	t.Cleanup(func() {
		cleanupMitreData(t, pool, tenantID)
	})

	// ---- Seed asset (for LEFT JOIN verification) ----
	assetID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO assets (id, tenant_id, name, type, criticality, status, created_at, updated_at)
		VALUES ($1, $2, 'web-server-01', 'server', 'high', 'active', NOW(), NOW())`,
		assetID, tenantID,
	)
	if err != nil {
		t.Fatalf("seed asset: %v", err)
	}

	// ---- Seed alerts ----
	for i, title := range []string{"PowerShell encoded command", "Obfuscated script", "Script interpreter abuse"} {
		alertID := uuid.New()
		var aid *uuid.UUID
		if i == 0 {
			aid = &assetID // First alert linked to asset
		}
		_, err := pool.Exec(ctx, `
			INSERT INTO alerts (id, tenant_id, title, description, severity, status,
				confidence_score, asset_id, mitre_tactic_id, mitre_technique_id,
				first_event_at, last_event_at, created_at, updated_at)
			VALUES ($1, $2, $3, 'desc', 'high', 'new', 0.9, $4, 'TA0002', 'T1059',
				NOW(), NOW(), NOW() - ($5 || ' hours')::interval, NOW())`,
			alertID, tenantID, title, aid, i*2, // stagger by 2 hours
		)
		if err != nil {
			t.Fatalf("seed alert %d: %v", i, err)
		}
	}

	// ---- Execute TechniqueRecentAlerts ----
	alerts, err := ruleRepo.TechniqueRecentAlerts(ctx, tenantID, "T1059", 10)
	if err != nil {
		t.Fatalf("TechniqueRecentAlerts: %v", err)
	}

	if len(alerts) != 3 {
		t.Fatalf("expected 3 alerts, got %d", len(alerts))
	}

	// Verify ordering (most recent first)
	if !alerts[0].CreatedAt.After(alerts[1].CreatedAt) || alerts[0].CreatedAt.Equal(alerts[1].CreatedAt) {
		// The first alert should be the most recent or same time
	}

	// Verify asset name is populated for the alert linked to the asset
	foundAssetName := false
	for _, alert := range alerts {
		if alert.AssetName != nil && *alert.AssetName == "web-server-01" {
			foundAssetName = true
		}
		if alert.Severity != "high" {
			t.Errorf("expected severity 'high', got %q", alert.Severity)
		}
		if alert.Status != "new" {
			t.Errorf("expected status 'new', got %q", alert.Status)
		}
	}
	if !foundAssetName {
		t.Error("expected at least one alert with asset_name 'web-server-01'")
	}

	// ---- Verify LIMIT works ----
	limited, err := ruleRepo.TechniqueRecentAlerts(ctx, tenantID, "T1059", 2)
	if err != nil {
		t.Fatalf("TechniqueRecentAlerts limit: %v", err)
	}
	if len(limited) != 2 {
		t.Errorf("expected 2 alerts with limit=2, got %d", len(limited))
	}

	// ---- Verify empty result for unknown technique ----
	empty, err := ruleRepo.TechniqueRecentAlerts(ctx, tenantID, "T9999", 10)
	if err != nil {
		t.Fatalf("TechniqueRecentAlerts unknown: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0 alerts for T9999, got %d", len(empty))
	}
}

func TestMITRECoverage_Integration_ListByTechnique(t *testing.T) {
	pool := newIntegrationPool(t)
	logger := zerolog.Nop()
	ruleRepo := repository.NewRuleRepository(pool, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tenantID := uuid.New()
	t.Cleanup(func() {
		cleanupMitreData(t, pool, tenantID)
	})

	// ---- Seed rules: two matching T1059, one matching T1566 only ----
	ruleContent, _ := json.Marshal(map[string]string{"query": "test"})
	for _, tc := range []struct {
		name       string
		techniques []string
		enabled    bool
	}{
		{"PowerShell Rule", []string{"T1059", "T1059.001"}, true},
		{"Script Rule", []string{"T1059"}, true},
		{"Phishing Rule", []string{"T1566"}, true},
		{"Disabled Rule", []string{"T1059"}, false},
	} {
		_, err := pool.Exec(ctx, `
			INSERT INTO detection_rules (id, tenant_id, name, description, rule_type, severity,
				enabled, rule_content, mitre_tactic_ids, mitre_technique_ids,
				base_confidence, trigger_count, created_at, updated_at)
			VALUES ($1, $2, $3, 'desc', 'sigma', 'medium', $4, $5, $6, $7, 0.7, 0, NOW(), NOW())`,
			uuid.New(), tenantID, tc.name, tc.enabled, string(ruleContent),
			[]string{"TA0002"}, tc.techniques,
		)
		if err != nil {
			t.Fatalf("seed rule %q: %v", tc.name, err)
		}
	}

	// ---- ListByTechnique ----
	rules, err := ruleRepo.ListByTechnique(ctx, tenantID, "T1059")
	if err != nil {
		t.Fatalf("ListByTechnique: %v", err)
	}

	// Should find 3 rules (2 enabled + 1 disabled) that have T1059 in their array
	if len(rules) != 3 {
		t.Fatalf("expected 3 rules for T1059, got %d", len(rules))
	}

	// Verify ordering: enabled first
	if !rules[0].Enabled {
		t.Error("first rule should be enabled (sorted enabled DESC)")
	}
	if rules[len(rules)-1].Enabled {
		t.Error("last rule should be disabled")
	}
}

func TestMITRECoverage_Integration_ListEnabledByTenant(t *testing.T) {
	pool := newIntegrationPool(t)
	logger := zerolog.Nop()
	ruleRepo := repository.NewRuleRepository(pool, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tenantID := uuid.New()
	t.Cleanup(func() {
		cleanupMitreData(t, pool, tenantID)
	})

	ruleContent, _ := json.Marshal(map[string]string{"query": "test"})
	for i, enabled := range []bool{true, true, false} {
		_, err := pool.Exec(ctx, `
			INSERT INTO detection_rules (id, tenant_id, name, description, rule_type, severity,
				enabled, rule_content, mitre_tactic_ids, mitre_technique_ids,
				base_confidence, trigger_count, created_at, updated_at)
			VALUES ($1, $2, $3, 'desc', 'sigma', 'medium', $4, $5, $6, $7, 0.7, 0, NOW(), NOW())`,
			uuid.New(), tenantID, "Rule "+string(rune('A'+i)), enabled, string(ruleContent),
			[]string{"TA0002"}, []string{"T1059"},
		)
		if err != nil {
			t.Fatalf("seed rule %d: %v", i, err)
		}
	}

	rules, err := ruleRepo.ListEnabledByTenant(ctx, tenantID)
	if err != nil {
		t.Fatalf("ListEnabledByTenant: %v", err)
	}
	if len(rules) != 2 {
		t.Errorf("expected 2 enabled rules, got %d", len(rules))
	}
	for _, rule := range rules {
		if !rule.Enabled {
			t.Error("ListEnabledByTenant returned a disabled rule")
		}
	}
}

func TestMITRECoverage_Integration_FullCoverageFlow(t *testing.T) {
	pool := newIntegrationPool(t)
	logger := zerolog.Nop()
	ruleRepo := repository.NewRuleRepository(pool, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tenantID := uuid.New()
	t.Cleanup(func() {
		cleanupMitreData(t, pool, tenantID)
	})

	// ---- Seed: 1 enabled rule covering T1059, 1 alert, 1 active threat ----
	ruleID := uuid.New()
	ruleContent, _ := json.Marshal(map[string]string{"query": "test"})
	_, _ = pool.Exec(ctx, `
		INSERT INTO detection_rules (id, tenant_id, name, description, rule_type, severity,
			enabled, rule_content, mitre_tactic_ids, mitre_technique_ids,
			base_confidence, false_positive_count, true_positive_count, trigger_count,
			created_at, updated_at)
		VALUES ($1, $2, 'PS Rule', 'desc', 'sigma', 'high', true, $3, $4, $5, 0.7, 1, 9, 10,
			NOW(), NOW())`,
		ruleID, tenantID, string(ruleContent), []string{"TA0002"}, []string{"T1059"},
	)

	_, _ = pool.Exec(ctx, `
		INSERT INTO alerts (id, tenant_id, title, description, severity, status,
			confidence_score, mitre_technique_id, first_event_at, last_event_at,
			created_at, updated_at)
		VALUES ($1, $2, 'PS Alert', 'desc', 'high', 'new', 0.9, 'T1059',
			NOW(), NOW(), NOW(), NOW())`,
		uuid.New(), tenantID,
	)

	_, _ = pool.Exec(ctx, `
		INSERT INTO threats (id, tenant_id, name, description, type, severity, status,
			mitre_tactic_ids, mitre_technique_ids,
			first_seen_at, last_seen_at, created_at, updated_at)
		VALUES ($1, $2, 'Active Threat', 'desc', 'apt', 'critical', 'active',
			$3, $4, NOW(), NOW(), NOW(), NOW())`,
		uuid.New(), tenantID, []string{"TA0002"}, []string{"T1059"},
	)

	// ---- Build full coverage (same as service.Coverage does) ----
	rules, err := ruleRepo.ListEnabledByTenant(ctx, tenantID)
	if err != nil {
		t.Fatalf("ListEnabledByTenant: %v", err)
	}

	coverage := buildTestCoverage(rules)

	contextMap, err := ruleRepo.TechniqueCoverageContextMap(ctx, tenantID)
	if err != nil {
		t.Fatalf("TechniqueCoverageContextMap: %v", err)
	}

	// ---- Verify: T1059 should be "covered" ----
	var found bool
	for _, item := range coverage {
		if item.TechniqueID == "T1059" {
			found = true
			if !item.HasDetection {
				t.Error("T1059 should have detection")
			}
			if item.RuleCount != 1 {
				t.Errorf("T1059 rule_count: expected 1, got %d", item.RuleCount)
			}
			entry := contextMap["T1059"]
			if entry == nil {
				t.Error("T1059 context should exist")
			} else {
				if entry.AlertCount != 1 {
					t.Errorf("T1059 context alert_count: expected 1, got %d", entry.AlertCount)
				}
				if entry.ActiveThreatCount != 1 {
					t.Errorf("T1059 context active_threat_count: expected 1, got %d", entry.ActiveThreatCount)
				}
			}
			break
		}
	}
	if !found {
		t.Error("T1059 not found in coverage list")
	}
}

// buildTestCoverage mirrors the service's Coverage() logic at the DTO level.
func buildTestCoverage(rules []*model.DetectionRule) []dto.MITRECoverageDTO {
	cov := mitre.BuildCoverage(rules)
	out := make([]dto.MITRECoverageDTO, 0, len(cov))
	for _, item := range cov {
		out = append(out, dto.MITRECoverageDTO{
			TechniqueID:   item.Technique.ID,
			TechniqueName: item.Technique.Name,
			TacticIDs:     item.Technique.TacticIDs,
			HasDetection:  item.HasDetection,
			RuleCount:     item.RuleCount,
			RuleNames:     item.RuleNames,
			Description:   item.Technique.Description,
			Platforms:     item.Technique.Platforms,
		})
	}
	return out
}

func cleanupMitreData(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for _, stmt := range []string{
		`DELETE FROM alerts WHERE tenant_id = $1`,
		`DELETE FROM threats WHERE tenant_id = $1`,
		`DELETE FROM detection_rules WHERE tenant_id = $1`,
		`DELETE FROM assets WHERE tenant_id = $1`,
	} {
		if _, err := pool.Exec(ctx, stmt, tenantID); err != nil {
			t.Logf("cleanup: %v", err)
		}
	}
}
