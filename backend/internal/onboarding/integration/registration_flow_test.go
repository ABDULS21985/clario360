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

	assertProvisionedTenantArtifacts(t, h, tenant.TenantID)
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
