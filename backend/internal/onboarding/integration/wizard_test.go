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

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
