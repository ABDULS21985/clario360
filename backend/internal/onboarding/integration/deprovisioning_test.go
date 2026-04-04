//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"

	iamdto "github.com/clario360/platform/internal/iam/dto"
	onboardingdto "github.com/clario360/platform/internal/onboarding/dto"
)

func TestDeprovisionSuspendsTenantAndBlocksLogin(t *testing.T) {
	h := newHarness(t)
	tenant := h.registerAndVerifyTenant(t)
	h.waitForProvisioning(t, tenant.TenantID)

	adminToken := h.superAdminToken(t)
	h.postJSON(t, fmt.Sprintf("/api/v1/admin/tenants/%s/deprovision", tenant.TenantID), onboardingdto.DeprovisionRequest{
		Reason:     "Subscription cancelled",
		RetainDays: 90,
	}, adminToken, http.StatusOK)

	var tenantStatus string
	if err := h.env.platformPool.QueryRow(h.newContext(), `SELECT status FROM tenants WHERE id = $1`, tenant.TenantID).Scan(&tenantStatus); err != nil {
		t.Fatalf("load tenant status after deprovision: %v", err)
	}
	if tenantStatus != "deprovisioned" {
		t.Fatalf("expected tenant status deprovisioned, got %s", tenantStatus)
	}

	suspendedUsers := h.countRows(t, h.env.platformPool, `SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND status = 'suspended'`, tenant.TenantID)
	if suspendedUsers == 0 {
		t.Fatal("expected suspended users after deprovision")
	}

	h.postJSON(t, "/api/v1/auth/login", iamdto.LoginRequest{
		TenantID: tenant.TenantID.String(),
		Email:    tenant.AdminEmail,
		Password: tenant.Password,
	}, "", http.StatusUnauthorized)

	slug := h.tenantSlug(t, tenant.TenantID)
	for _, bucket := range []string{
		"clario360-" + slug + "-cyber",
		"clario360-" + slug + "-data",
		"clario360-" + slug + "-acta",
		"clario360-" + slug + "-lex",
		"clario360-" + slug + "-visus",
		"clario360-" + slug + "-platform",
	} {
		tags := h.bucketTags(t, bucket)
		if tags["lifecycle"] != "deprovisioned" {
			t.Fatalf("expected bucket %s lifecycle tag to be deprovisioned, got %q", bucket, tags["lifecycle"])
		}
	}
}
