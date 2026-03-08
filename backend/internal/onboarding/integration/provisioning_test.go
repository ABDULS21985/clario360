//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	onboardingmodel "github.com/clario360/platform/internal/onboarding/model"
)

func TestProvisioningCreatesAllDefaults(t *testing.T) {
	h := newHarness(t)
	tenant := h.registerAndVerifyTenant(t)

	status := h.waitForProvisioning(t, tenant.TenantID)
	if status.Status != onboardingmodel.OnboardingProvisioningCompleted {
		t.Fatalf("expected completed provisioning status, got %s", status.Status)
	}

	assertProvisionedTenantArtifacts(t, h, tenant.TenantID)
}

func TestProvisioningResume(t *testing.T) {
	h := newHarness(t)
	tenant := h.registerAndVerifyTenant(t)
	h.waitForProvisioning(t, tenant.TenantID)

	var stepOneStartedAt, stepOneCompletedAt time.Time
	var stepOneRetryCount int
	if err := h.env.platformPool.QueryRow(h.newContext(), `
		SELECT started_at, completed_at, retry_count
		FROM provisioning_steps
		WHERE tenant_id = $1 AND step_number = 1`,
		tenant.TenantID,
	).Scan(&stepOneStartedAt, &stepOneCompletedAt, &stepOneRetryCount); err != nil {
		t.Fatalf("load completed step metadata: %v", err)
	}

	if _, err := h.env.platformPool.Exec(h.newContext(), `
		UPDATE provisioning_steps
		SET status = 'failed',
		    started_at = now(),
		    completed_at = NULL,
		    duration_ms = NULL,
		    error_message = 'forced reprovision integration failure'
		WHERE tenant_id = $1 AND step_number = 7`,
		tenant.TenantID,
	); err != nil {
		t.Fatalf("mark step 7 failed: %v", err)
	}
	if _, err := h.env.platformPool.Exec(h.newContext(), `
		UPDATE provisioning_steps
		SET status = 'pending',
		    started_at = NULL,
		    completed_at = NULL,
		    duration_ms = NULL,
		    error_message = NULL
		WHERE tenant_id = $1 AND step_number > 7`,
		tenant.TenantID,
	); err != nil {
		t.Fatalf("reset pending steps: %v", err)
	}
	if _, err := h.env.platformPool.Exec(h.newContext(), `
		UPDATE tenant_onboarding
		SET provisioning_status = 'failed',
		    provisioning_completed_at = NULL,
		    provisioning_error = 'forced reprovision integration failure'
		WHERE tenant_id = $1`,
		tenant.TenantID,
	); err != nil {
		t.Fatalf("mark onboarding provisioning failed: %v", err)
	}
	if _, err := h.env.platformPool.Exec(h.newContext(), `
		UPDATE tenants
		SET status = 'onboarding'
		WHERE id = $1`,
		tenant.TenantID,
	); err != nil {
		t.Fatalf("reset tenant status to onboarding: %v", err)
	}

	adminToken := h.superAdminToken(t)
	h.postJSON(t, fmt.Sprintf("/api/v1/admin/tenants/%s/reprovision", tenant.TenantID), map[string]string{}, adminToken, http.StatusAccepted)

	deadline := time.Now().Add(15 * time.Second)
	reprovisionStarted := false
	for time.Now().Before(deadline) {
		var retryCount int
		if err := h.env.platformPool.QueryRow(h.newContext(), `
			SELECT retry_count
			FROM provisioning_steps
			WHERE tenant_id = $1 AND step_number = 7`,
			tenant.TenantID,
		).Scan(&retryCount); err != nil {
			t.Fatalf("load step 7 retry_count while waiting for reprovision: %v", err)
		}
		if retryCount > 0 {
			reprovisionStarted = true
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if !reprovisionStarted {
		t.Fatal("reprovision did not start before timeout")
	}

	status := h.waitForProvisioning(t, tenant.TenantID)
	if status.Status != onboardingmodel.OnboardingProvisioningCompleted {
		t.Fatalf("expected completed provisioning status after reprovision, got %s", status.Status)
	}

	var currentStepOneStartedAt, currentStepOneCompletedAt time.Time
	var currentStepOneRetryCount int
	if err := h.env.platformPool.QueryRow(h.newContext(), `
		SELECT started_at, completed_at, retry_count
		FROM provisioning_steps
		WHERE tenant_id = $1 AND step_number = 1`,
		tenant.TenantID,
	).Scan(&currentStepOneStartedAt, &currentStepOneCompletedAt, &currentStepOneRetryCount); err != nil {
		t.Fatalf("reload completed step metadata: %v", err)
	}
	if !currentStepOneStartedAt.Equal(stepOneStartedAt) || !currentStepOneCompletedAt.Equal(stepOneCompletedAt) {
		t.Fatal("expected completed step 1 timestamps to remain unchanged after reprovision")
	}
	if currentStepOneRetryCount != stepOneRetryCount {
		t.Fatalf("expected completed step 1 retry_count to remain %d, got %d", stepOneRetryCount, currentStepOneRetryCount)
	}

	var stepSevenStatus string
	var stepSevenRetryCount int
	if err := h.env.platformPool.QueryRow(h.newContext(), `
		SELECT status, retry_count
		FROM provisioning_steps
		WHERE tenant_id = $1 AND step_number = 7`,
		tenant.TenantID,
	).Scan(&stepSevenStatus, &stepSevenRetryCount); err != nil {
		t.Fatalf("load reprovisioned step 7 metadata: %v", err)
	}
	if stepSevenStatus != string(onboardingmodel.ProvisioningStepCompleted) {
		t.Fatalf("expected step 7 to be completed after reprovision, got %s", stepSevenStatus)
	}
	if stepSevenRetryCount < 1 {
		t.Fatalf("expected step 7 retry_count to increment on reprovision, got %d", stepSevenRetryCount)
	}

	assertProvisionedTenantArtifacts(t, h, tenant.TenantID)
}

func assertProvisionedTenantArtifacts(t *testing.T, h *onboardingHarness, tenantID uuid.UUID) {
	t.Helper()

	if got := h.countRows(t, h.env.platformPool, `SELECT COUNT(*) FROM roles WHERE tenant_id = $1`, tenantID); got != 11 {
		t.Fatalf("expected 11 seeded roles, got %d", got)
	}
	if got := h.countRows(t, h.env.platformPool, `SELECT COUNT(*) FROM system_settings WHERE tenant_id = $1`, tenantID); got != 10 {
		t.Fatalf("expected 10 seeded settings, got %d", got)
	}
	if got := h.countRows(t, h.env.dbPools["cyber_db"], `SELECT COUNT(*) FROM detection_rules WHERE tenant_id = $1`, tenantID); got != 15 {
		t.Fatalf("expected 15 seeded detection rules, got %d", got)
	}
	if got := h.countRows(t, h.env.dbPools["visus_db"], `SELECT COUNT(*) FROM visus_kpi_definitions WHERE tenant_id = $1`, tenantID); got != 12 {
		t.Fatalf("expected 12 seeded KPI definitions, got %d", got)
	}
	if got := h.countRows(t, h.env.dbPools["visus_db"], `SELECT COUNT(*) FROM visus_dashboards WHERE tenant_id = $1 AND name = 'Executive Overview'`, tenantID); got != 1 {
		t.Fatalf("expected 1 executive overview dashboard, got %d", got)
	}
	if got := h.countRows(t, h.env.dbPools["visus_db"], `SELECT COUNT(*) FROM visus_widgets WHERE tenant_id = $1`, tenantID); got != 8 {
		t.Fatalf("expected 8 seeded widgets, got %d", got)
	}
	if got := h.countRows(t, h.env.dbPools["lex_db"], `SELECT COUNT(*) FROM compliance_rules WHERE tenant_id = $1`, tenantID); got != 5 {
		t.Fatalf("expected 5 seeded compliance rules, got %d", got)
	}
	if got := h.countRows(t, h.env.platformPool, `SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND action = 'tenant.provisioned'`, tenantID); got != 1 {
		t.Fatalf("expected 1 tenant.provisioned audit record, got %d", got)
	}

	var tenantStatus string
	if err := h.env.platformPool.QueryRow(h.newContext(), `SELECT status FROM tenants WHERE id = $1`, tenantID).Scan(&tenantStatus); err != nil {
		t.Fatalf("load tenant status: %v", err)
	}
	if tenantStatus != "active" {
		t.Fatalf("expected tenant status active, got %s", tenantStatus)
	}

	slug := h.tenantSlug(t, tenantID)
	for _, bucket := range []string{
		"clario360-" + slug + "-cyber",
		"clario360-" + slug + "-data",
		"clario360-" + slug + "-acta",
		"clario360-" + slug + "-lex",
		"clario360-" + slug + "-visus",
		"clario360-" + slug + "-platform",
	} {
		if !h.bucketExists(t, bucket) {
			t.Fatalf("expected bucket %s to exist", bucket)
		}
	}
}
