//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	onboardingdto "github.com/clario360/platform/internal/onboarding/dto"
	onboardingmodel "github.com/clario360/platform/internal/onboarding/model"
)

func TestInvitationFlow(t *testing.T) {
	h := newHarness(t)
	tenant := h.registerAndVerifyTenant(t)
	h.waitForProvisioning(t, tenant.TenantID)

	inviteeEmail := fmt.Sprintf("invitee-%d@example.com", h.sequence)
	body := h.postJSON(t, "/api/v1/invitations", onboardingdto.BatchInviteRequest{
		Invitations: []onboardingdto.InvitationInput{
			{
				Email:    inviteeEmail,
				RoleSlug: "security-analyst",
				Message:  "Welcome aboard",
			},
		},
	}, tenant.AccessToken, http.StatusCreated)

	var createResp struct {
		Count int `json:"count"`
	}
	mustDecode(t, body, &createResp)
	if createResp.Count != 1 {
		t.Fatalf("expected 1 invitation created, got %d", createResp.Count)
	}

	token, ok := h.env.emailSender.invitationToken(inviteeEmail)
	if !ok {
		t.Fatalf("expected invitation token for %s", inviteeEmail)
	}

	validateBody := h.get(t, "/api/v1/invitations/validate?token="+url.QueryEscape(token), "", http.StatusOK)
	var details onboardingmodel.InvitationDetails
	mustDecode(t, validateBody, &details)
	if details.Email != inviteeEmail {
		t.Fatalf("expected invited email %s, got %s", inviteeEmail, details.Email)
	}
	if details.RoleSlug != "security-analyst" {
		t.Fatalf("expected security-analyst role, got %s", details.RoleSlug)
	}

	acceptBody := h.postJSON(t, "/api/v1/invitations/accept", onboardingdto.AcceptInviteRequest{
		Token:     token,
		FirstName: "Alice",
		LastName:  "Invitee",
		Password:  "Clario360!Invitee1",
	}, "", http.StatusCreated)

	var acceptResp onboardingdto.AcceptInviteResponse
	mustDecode(t, acceptBody, &acceptResp)
	if acceptResp.AccessToken == "" {
		t.Fatal("expected access token after invitation acceptance")
	}

	var userCount int
	if err := h.env.platformPool.QueryRow(h.newContext(), `
		SELECT COUNT(*)
		FROM users
		WHERE tenant_id = $1 AND lower(email) = lower($2)`,
		tenant.TenantID,
		inviteeEmail,
	).Scan(&userCount); err != nil {
		t.Fatalf("load invited user count: %v", err)
	}
	if userCount != 1 {
		t.Fatalf("expected invited user to be created once, got %d", userCount)
	}

	var roleCount int
	if err := h.env.platformPool.QueryRow(h.newContext(), `
		SELECT COUNT(*)
		FROM user_roles ur
		INNER JOIN users u ON u.id = ur.user_id
		INNER JOIN roles r ON r.id = ur.role_id
		WHERE u.tenant_id = $1
		  AND lower(u.email) = lower($2)
		  AND r.slug = 'security-analyst'`,
		tenant.TenantID,
		inviteeEmail,
	).Scan(&roleCount); err != nil {
		t.Fatalf("load invited user role assignment: %v", err)
	}
	if roleCount != 1 {
		t.Fatalf("expected invited user role assignment once, got %d", roleCount)
	}
}

func TestInvitationExpiry(t *testing.T) {
	h := newHarness(t)
	tenant := h.registerAndVerifyTenant(t)
	h.waitForProvisioning(t, tenant.TenantID)

	inviteeEmail := fmt.Sprintf("expired-%d@example.com", h.sequence)
	h.postJSON(t, "/api/v1/invitations", onboardingdto.BatchInviteRequest{
		Invitations: []onboardingdto.InvitationInput{
			{
				Email:    inviteeEmail,
				RoleSlug: "viewer",
			},
		},
	}, tenant.AccessToken, http.StatusCreated)

	token, ok := h.env.emailSender.invitationToken(inviteeEmail)
	if !ok {
		t.Fatalf("expected invitation token for %s", inviteeEmail)
	}

	if _, err := h.env.platformPool.Exec(h.newContext(), `
		UPDATE invitations
		SET expires_at = now() - interval '1 minute'
		WHERE tenant_id = $1 AND lower(email) = lower($2)`,
		tenant.TenantID,
		inviteeEmail,
	); err != nil {
		t.Fatalf("expire invitation: %v", err)
	}

	body := h.get(t, "/api/v1/invitations/validate?token="+url.QueryEscape(token), "", http.StatusGone)
	var errResp apiErrorResponse
	mustDecode(t, body, &errResp)
	if !strings.Contains(strings.ToLower(errResp.Error), "expired") {
		t.Fatalf("expected expired invitation error, got %q", errResp.Error)
	}
}
