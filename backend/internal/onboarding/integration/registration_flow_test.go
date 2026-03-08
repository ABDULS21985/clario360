//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	onboardingmodel "github.com/clario360/platform/internal/onboarding/model"
)

func TestFullRegistrationFlow(t *testing.T) {
	h := newHarness(t)
	tenant := h.registerAndVerifyTenant(t)

	status := h.waitForProvisioning(t, tenant.TenantID)
	if status.TotalSteps != 11 {
		t.Fatalf("expected 11 provisioning steps, got %d", status.TotalSteps)
	}
	if status.CompletedStep != 11 {
		t.Fatalf("expected all provisioning steps completed, got %d", status.CompletedStep)
	}

	body := h.get(t, fmt.Sprintf("/api/v1/onboarding/status/%s", tenant.TenantID), "", http.StatusOK)
	var apiStatus onboardingmodel.ProvisioningStatus
	mustDecode(t, body, &apiStatus)
	if apiStatus.Status != onboardingmodel.OnboardingProvisioningCompleted {
		t.Fatalf("expected completed provisioning status, got %s", apiStatus.Status)
	}
	if len(apiStatus.Steps) != 11 {
		t.Fatalf("expected 11 status steps, got %d", len(apiStatus.Steps))
	}

	if got := h.countRows(t, h.env.platformPool, `SELECT COUNT(*) FROM roles WHERE tenant_id = $1`, tenant.TenantID); got != 11 {
		t.Fatalf("expected 11 seeded roles, got %d", got)
	}
	if got := h.countRows(t, h.env.platformPool, `SELECT COUNT(*) FROM system_settings WHERE tenant_id = $1`, tenant.TenantID); got != 10 {
		t.Fatalf("expected 10 seeded settings, got %d", got)
	}
	if got := h.countRows(t, h.env.dbPools["cyber_db"], `SELECT COUNT(*) FROM detection_rules WHERE tenant_id = $1`, tenant.TenantID); got != 15 {
		t.Fatalf("expected 15 seeded detection rules, got %d", got)
	}
	if got := h.countRows(t, h.env.dbPools["visus_db"], `SELECT COUNT(*) FROM visus_kpi_definitions WHERE tenant_id = $1`, tenant.TenantID); got != 12 {
		t.Fatalf("expected 12 seeded KPI definitions, got %d", got)
	}
	if got := h.countRows(t, h.env.dbPools["visus_db"], `SELECT COUNT(*) FROM visus_dashboards WHERE tenant_id = $1 AND name = 'Executive Overview'`, tenant.TenantID); got != 1 {
		t.Fatalf("expected 1 executive overview dashboard, got %d", got)
	}
	if got := h.countRows(t, h.env.dbPools["visus_db"], `SELECT COUNT(*) FROM visus_widgets WHERE tenant_id = $1`, tenant.TenantID); got != 8 {
		t.Fatalf("expected 8 seeded widgets, got %d", got)
	}
	if got := h.countRows(t, h.env.dbPools["lex_db"], `SELECT COUNT(*) FROM compliance_rules WHERE tenant_id = $1`, tenant.TenantID); got != 5 {
		t.Fatalf("expected 5 seeded compliance rules, got %d", got)
	}
	if got := h.countRows(t, h.env.platformPool, `SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND action = 'tenant.provisioned'`, tenant.TenantID); got != 1 {
		t.Fatalf("expected 1 tenant.provisioned audit record, got %d", got)
	}

	var tenantStatus string
	if err := h.env.platformPool.QueryRow(h.newContext(), `SELECT status FROM tenants WHERE id = $1`, tenant.TenantID).Scan(&tenantStatus); err != nil {
		t.Fatalf("load tenant status: %v", err)
	}
	if tenantStatus != "active" {
		t.Fatalf("expected tenant status active, got %s", tenantStatus)
	}

	slug := h.tenantSlug(t, tenant.TenantID)
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

func TestRegistrationWrongOTP(t *testing.T) {
	h := newHarness(t)
	tenant := h.registerTenant(t)

	for attempt := 1; attempt <= 4; attempt++ {
		body := h.verifyTenant(t, tenant, "000000", http.StatusUnauthorized)
		var errResp apiErrorResponse
		mustDecode(t, body, &errResp)
		if !strings.Contains(errResp.Error, "attempts remaining") {
			t.Fatalf("expected attempts remaining message on attempt %d, got %q", attempt, errResp.Error)
		}
	}

	body := h.verifyTenant(t, tenant, "000000", http.StatusTooManyRequests)
	var errResp apiErrorResponse
	mustDecode(t, body, &errResp)
	if !strings.Contains(strings.ToLower(errResp.Error), "too many attempts") {
		t.Fatalf("expected lockout message, got %q", errResp.Error)
	}
}

func TestRegistrationOTPExpiry(t *testing.T) {
	h := newHarness(t)
	tenant := h.registerTenant(t)

	if _, err := h.env.platformPool.Exec(h.newContext(), `
		UPDATE email_verifications
		SET expires_at = now() - interval '1 minute'
		WHERE lower(email) = lower($1) AND purpose = 'registration'`,
		tenant.AdminEmail,
	); err != nil {
		t.Fatalf("expire otp: %v", err)
	}

	body := h.verifyTenant(t, tenant, tenant.OTP, http.StatusUnauthorized)
	var errResp apiErrorResponse
	mustDecode(t, body, &errResp)
	if !strings.Contains(strings.ToLower(errResp.Error), "expired") {
		t.Fatalf("expected expired otp error, got %q", errResp.Error)
	}
}
