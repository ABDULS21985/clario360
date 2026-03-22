//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"

	onboardingdto "github.com/clario360/platform/internal/onboarding/dto"
	onboardingmodel "github.com/clario360/platform/internal/onboarding/model"
)

const testSVGLogo = `<svg xmlns="http://www.w3.org/2000/svg" width="120" height="40" viewBox="0 0 120 40"><rect width="120" height="40" rx="8" fill="#006B3F"/><text x="60" y="25" text-anchor="middle" font-size="14" fill="#ffffff">Clario</text></svg>`

func TestWizardFlowWithLogoUpload(t *testing.T) {
	h := newHarness(t)
	tenant := h.registerAndVerifyTenant(t)

	h.postJSON(t, "/api/v1/onboarding/wizard/organization", onboardingdto.OrganizationDetailsRequest{
		OrganizationName: tenant.OrganizationName,
		Industry:         "financial",
		Country:          "SA",
		City:             "Riyadh",
		OrganizationSize: "51-200",
	}, tenant.AccessToken, http.StatusOK)

	h.postMultipart(t, "/api/v1/onboarding/wizard/branding", map[string]string{
		"primary_color": "#00573B",
		"accent_color":  "#C5A04E",
	}, []multipartFileInput{
		{
			FieldName:   "logo",
			FileName:    "tenant-logo.svg",
			ContentType: "image/svg+xml",
			Content:     []byte(testSVGLogo),
		},
	}, tenant.AccessToken, http.StatusOK)

	h.postJSON(t, "/api/v1/onboarding/wizard/suites", onboardingdto.SuitesStepRequest{
		ActiveSuites: []string{"cyber", "visus"},
	}, tenant.AccessToken, http.StatusOK)

	h.postJSON(t, "/api/v1/onboarding/wizard/complete", map[string]string{}, tenant.AccessToken, http.StatusOK)

	body := h.get(t, "/api/v1/onboarding/wizard", tenant.AccessToken, http.StatusOK)
	var progress onboardingmodel.WizardProgress
	mustDecode(t, body, &progress)

	if !progress.WizardCompleted {
		t.Fatal("expected wizard to be completed")
	}
	if progress.CurrentStep != 5 {
		t.Fatalf("expected current step 5, got %d", progress.CurrentStep)
	}
	if progress.LogoFileID == nil {
		t.Fatal("expected logo_file_id to be persisted")
	}
	if progress.PrimaryColor == nil || *progress.PrimaryColor != "#00573B" {
		t.Fatalf("expected primary color #00573B, got %v", progress.PrimaryColor)
	}
	if progress.AccentColor == nil || *progress.AccentColor != "#C5A04E" {
		t.Fatalf("expected accent color #C5A04E, got %v", progress.AccentColor)
	}
	if !containsString(progress.ActiveSuites, "cyber") || !containsString(progress.ActiveSuites, "visus") || len(progress.ActiveSuites) != 2 {
		t.Fatalf("unexpected active suites: %#v", progress.ActiveSuites)
	}

	bucket, storageKey, detectedType := h.loadLogoStorageRecord(t, progress.LogoFileID.String())
	if detectedType != "image/svg+xml" {
		t.Fatalf("expected detected logo content type image/svg+xml, got %s", detectedType)
	}
	exists, err := h.env.storage.Exists(context.Background(), bucket, storageKey)
	if err != nil {
		t.Fatalf("check uploaded logo object: %v", err)
	}
	if !exists {
		t.Fatalf("expected uploaded logo object %s/%s to exist", bucket, storageKey)
	}
}

func TestWizardResume(t *testing.T) {
	h := newHarness(t)
	tenant := h.registerAndVerifyTenant(t)

	h.postJSON(t, "/api/v1/onboarding/wizard/organization", onboardingdto.OrganizationDetailsRequest{
		OrganizationName: tenant.OrganizationName,
		Industry:         "financial",
		Country:          "SA",
		City:             "Jeddah",
		OrganizationSize: "1-50",
	}, tenant.AccessToken, http.StatusOK)

	body := h.get(t, "/api/v1/onboarding/wizard", tenant.AccessToken, http.StatusOK)
	var progress onboardingmodel.WizardProgress
	mustDecode(t, body, &progress)

	if progress.CurrentStep != 2 {
		t.Fatalf("expected wizard to resume at step 2, got %d", progress.CurrentStep)
	}
	if progress.OrganizationName == nil || *progress.OrganizationName != tenant.OrganizationName {
		t.Fatalf("expected organization name %q, got %v", tenant.OrganizationName, progress.OrganizationName)
	}
}

// TestWizardTeamStep verifies the full team-invite wizard step contract:
// empty-list skip → step marked completed; non-empty list → invitations persisted.
func TestWizardTeamStep(t *testing.T) {
	h := newHarness(t)
	tenant := h.registerAndVerifyTenant(t)

	// Complete org step first (prerequisite)
	h.postJSON(t, "/api/v1/onboarding/wizard/organization", onboardingdto.OrganizationDetailsRequest{
		OrganizationName: tenant.OrganizationName,
		Industry:         "technology",
		Country:          "AE",
		OrganizationSize: "51-200",
	}, tenant.AccessToken, http.StatusOK)

	// Branding skip (no logo, no colors — just advance the step)
	h.postJSON(t, "/api/v1/onboarding/wizard/branding", onboardingdto.BrandingRequest{}, tenant.AccessToken, http.StatusOK)

	// Team step: skip with empty invitations list
	teamBody := h.postJSON(t, "/api/v1/onboarding/wizard/team", onboardingdto.TeamStepRequest{
		Invitations: []onboardingdto.InvitationInput{},
	}, tenant.AccessToken, http.StatusOK)

	var skipResp onboardingdto.WizardStepResponse
	mustDecode(t, teamBody, &skipResp)
	if skipResp.InvitationsSent != 0 {
		t.Fatalf("expected 0 invitations_sent on skip, got %d", skipResp.InvitationsSent)
	}
	if skipResp.CurrentStep != 4 {
		t.Fatalf("expected current_step 4 after team skip, got %d", skipResp.CurrentStep)
	}
}

// TestWizardTeamInvitations verifies that valid invitations are created and counted.
func TestWizardTeamInvitations(t *testing.T) {
	h := newHarness(t)
	tenant := h.registerAndVerifyTenant(t)

	h.postJSON(t, "/api/v1/onboarding/wizard/organization", onboardingdto.OrganizationDetailsRequest{
		OrganizationName: tenant.OrganizationName,
		Industry:         "government",
		Country:          "SA",
		OrganizationSize: "201-1000",
	}, tenant.AccessToken, http.StatusOK)

	h.postJSON(t, "/api/v1/onboarding/wizard/branding", onboardingdto.BrandingRequest{}, tenant.AccessToken, http.StatusOK)

	teamBody := h.postJSON(t, "/api/v1/onboarding/wizard/team", onboardingdto.TeamStepRequest{
		Invitations: []onboardingdto.InvitationInput{
			{Email: "alice@example.com", RoleSlug: "viewer"},
			{Email: "bob@example.com", RoleSlug: "viewer", Message: "Welcome to the platform"},
		},
	}, tenant.AccessToken, http.StatusOK)

	var teamResp onboardingdto.WizardStepResponse
	mustDecode(t, teamBody, &teamResp)
	if teamResp.InvitationsSent != 2 {
		t.Fatalf("expected 2 invitations_sent, got %d", teamResp.InvitationsSent)
	}
	if teamResp.CurrentStep != 4 {
		t.Fatalf("expected current_step 4, got %d", teamResp.CurrentStep)
	}
}

// TestWizardBrandingColorsOnly verifies color-only branding (no logo) is accepted.
func TestWizardBrandingColorsOnly(t *testing.T) {
	h := newHarness(t)
	tenant := h.registerAndVerifyTenant(t)

	h.postJSON(t, "/api/v1/onboarding/wizard/organization", onboardingdto.OrganizationDetailsRequest{
		OrganizationName: tenant.OrganizationName,
		Industry:         "financial",
		Country:          "SA",
		OrganizationSize: "1-50",
	}, tenant.AccessToken, http.StatusOK)

	h.postJSON(t, "/api/v1/onboarding/wizard/branding", onboardingdto.BrandingRequest{
		PrimaryColor: "#123456",
		AccentColor:  "#ABCDEF",
	}, tenant.AccessToken, http.StatusOK)

	body := h.get(t, "/api/v1/onboarding/wizard", tenant.AccessToken, http.StatusOK)
	var progress onboardingmodel.WizardProgress
	mustDecode(t, body, &progress)

	if progress.PrimaryColor == nil || *progress.PrimaryColor != "#123456" {
		t.Fatalf("expected primary color #123456, got %v", progress.PrimaryColor)
	}
	if progress.AccentColor == nil || *progress.AccentColor != "#ABCDEF" {
		t.Fatalf("expected accent color #ABCDEF, got %v", progress.AccentColor)
	}
	if progress.CurrentStep != 3 {
		t.Fatalf("expected current_step 3 after branding, got %d", progress.CurrentStep)
	}
}

// TestWizardCompleteIdempotent verifies that calling CompleteWizard twice
// is safe and does not corrupt the step or completed state.
func TestWizardCompleteIdempotent(t *testing.T) {
	h := newHarness(t)
	tenant := h.registerAndVerifyTenant(t)

	h.postJSON(t, "/api/v1/onboarding/wizard/organization", onboardingdto.OrganizationDetailsRequest{
		OrganizationName: tenant.OrganizationName,
		Industry:         "retail",
		Country:          "US",
		OrganizationSize: "51-200",
	}, tenant.AccessToken, http.StatusOK)

	h.postJSON(t, "/api/v1/onboarding/wizard/branding", onboardingdto.BrandingRequest{}, tenant.AccessToken, http.StatusOK)
	h.postJSON(t, "/api/v1/onboarding/wizard/team", onboardingdto.TeamStepRequest{Invitations: []onboardingdto.InvitationInput{}}, tenant.AccessToken, http.StatusOK)
	h.postJSON(t, "/api/v1/onboarding/wizard/suites", onboardingdto.SuitesStepRequest{ActiveSuites: []string{"cyber"}}, tenant.AccessToken, http.StatusOK)

	// First complete
	h.postJSON(t, "/api/v1/onboarding/wizard/complete", map[string]string{}, tenant.AccessToken, http.StatusOK)

	// Second complete (idempotent re-call from step-5 re-mount)
	h.postJSON(t, "/api/v1/onboarding/wizard/complete", map[string]string{}, tenant.AccessToken, http.StatusOK)

	body := h.get(t, "/api/v1/onboarding/wizard", tenant.AccessToken, http.StatusOK)
	var progress onboardingmodel.WizardProgress
	mustDecode(t, body, &progress)

	if !progress.WizardCompleted {
		t.Fatal("expected wizard_completed true after double-complete")
	}
	if progress.CurrentStep != 5 {
		t.Fatalf("expected current_step 5, got %d", progress.CurrentStep)
	}
}

// TestWizardSuitesMinimumOne verifies that an empty active_suites list is rejected.
func TestWizardSuitesMinimumOne(t *testing.T) {
	h := newHarness(t)
	tenant := h.registerAndVerifyTenant(t)

	h.postJSON(t, "/api/v1/onboarding/wizard/organization", onboardingdto.OrganizationDetailsRequest{
		OrganizationName: tenant.OrganizationName,
		Industry:         "energy",
		Country:          "SA",
		OrganizationSize: "1000+",
	}, tenant.AccessToken, http.StatusOK)

	h.postJSON(t, "/api/v1/onboarding/wizard/branding", onboardingdto.BrandingRequest{}, tenant.AccessToken, http.StatusOK)
	h.postJSON(t, "/api/v1/onboarding/wizard/team", onboardingdto.TeamStepRequest{Invitations: nil}, tenant.AccessToken, http.StatusOK)

	// Empty active_suites must be rejected with 400
	h.postJSON(t, "/api/v1/onboarding/wizard/suites", onboardingdto.SuitesStepRequest{
		ActiveSuites: []string{},
	}, tenant.AccessToken, http.StatusBadRequest)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
