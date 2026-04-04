package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	iammodel "github.com/clario360/platform/internal/iam/model"
	onboardingmodel "github.com/clario360/platform/internal/onboarding/model"
)

func newProvisionerForTest() (*TenantProvisioner, *fakeProvisioningRepo, uuid.UUID) {
	tenantID := uuid.New()
	onboardingRepo := newFakeOnboardingRepo()
	onboardingRepo.onboardingByTenant[tenantID] = &onboardingmodel.OnboardingStatus{
		ID:            uuid.New(),
		TenantID:      tenantID,
		AdminUserID:   uuid.New(),
		AdminEmail:    "admin@acme.com",
		ActiveSuites:  []string{"cyber", "data", "visus"},
		CreatedAt:     time.Now().Add(-5 * time.Minute),
		UpdatedAt:     time.Now(),
		CurrentStep:   1,
		EmailVerified: true,
	}
	onboardingRepo.tenantIdentities[tenantID] = fakeTenantIdentity{
		name:   "Acme Corp",
		slug:   "acme-corp-a1b2",
		status: iammodel.TenantStatusOnboarding,
	}

	provisioningRepo := newFakeProvisioningRepo()
	provisioner := &TenantProvisioner{
		onboardingRepo:   onboardingRepo,
		provisioningRepo: provisioningRepo,
		logger:           newDiscardLogger(),
	}

	return provisioner, provisioningRepo, tenantID
}

func TestProvisionAllStepsCompleted(t *testing.T) {
	provisioner, provisioningRepo, tenantID := newProvisionerForTest()
	runCounts := map[int]int{}
	provisioner.pipeline = []provisioningPipelineStep{
		{Name: "Step 1", Run: func(ctx context.Context) error { runCounts[1]++; return nil }},
		{Name: "Step 2", Run: func(ctx context.Context) error { runCounts[2]++; return nil }},
		{Name: "Step 3", Run: func(ctx context.Context) error { runCounts[3]++; return nil }},
	}

	if err := provisioner.Provision(context.Background(), tenantID); err != nil {
		t.Fatalf("Provision returned error: %v", err)
	}

	steps, err := provisioningRepo.ListSteps(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("ListSteps returned error: %v", err)
	}
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(steps))
	}
	for _, step := range steps {
		if step.Status != onboardingmodel.ProvisioningStepCompleted {
			t.Fatalf("expected step %d to be completed, got %s", step.StepNumber, step.Status)
		}
	}
	if provisioningRepo.provisioningState[tenantID] != onboardingmodel.OnboardingProvisioningCompleted {
		t.Fatalf("expected provisioning status completed, got %s", provisioningRepo.provisioningState[tenantID])
	}
	if provisioningRepo.tenantStatus[tenantID] != "active" {
		t.Fatalf("expected tenant status active, got %q", provisioningRepo.tenantStatus[tenantID])
	}
	if runCounts[1] != 1 || runCounts[2] != 1 || runCounts[3] != 1 {
		t.Fatalf("expected each step to run once, got %+v", runCounts)
	}
}

func TestProvisionStepFailure(t *testing.T) {
	provisioner, provisioningRepo, tenantID := newProvisionerForTest()
	provisioner.pipeline = []provisioningPipelineStep{
		{Name: "Step 1", Run: func(ctx context.Context) error { return nil }},
		{Name: "Step 2", Run: func(ctx context.Context) error { return errors.New("boom") }},
		{Name: "Step 3", Run: func(ctx context.Context) error { return nil }},
	}

	err := provisioner.Provision(context.Background(), tenantID)
	if err == nil {
		t.Fatal("expected Provision to fail")
	}

	steps, err := provisioningRepo.ListSteps(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("ListSteps returned error: %v", err)
	}
	if steps[0].Status != onboardingmodel.ProvisioningStepCompleted {
		t.Fatalf("expected first step completed, got %s", steps[0].Status)
	}
	if steps[1].Status != onboardingmodel.ProvisioningStepFailed {
		t.Fatalf("expected second step failed, got %s", steps[1].Status)
	}
	if steps[2].Status != onboardingmodel.ProvisioningStepPending {
		t.Fatalf("expected third step pending, got %s", steps[2].Status)
	}
	if provisioningRepo.provisioningState[tenantID] != onboardingmodel.OnboardingProvisioningFailed {
		t.Fatalf("expected provisioning status failed, got %s", provisioningRepo.provisioningState[tenantID])
	}
}

func TestProvisionIdempotent(t *testing.T) {
	provisioner, provisioningRepo, tenantID := newProvisionerForTest()
	runCounts := map[int]int{}
	provisioner.pipeline = []provisioningPipelineStep{
		{Name: "Step 1", Run: func(ctx context.Context) error { runCounts[1]++; return nil }},
		{Name: "Step 2", Run: func(ctx context.Context) error { runCounts[2]++; return nil }},
		{Name: "Step 3", Run: func(ctx context.Context) error { runCounts[3]++; return nil }},
	}

	if err := provisioner.Provision(context.Background(), tenantID); err != nil {
		t.Fatalf("first Provision returned error: %v", err)
	}
	if err := provisioner.Provision(context.Background(), tenantID); err != nil {
		t.Fatalf("second Provision returned error: %v", err)
	}

	if runCounts[1] != 1 || runCounts[2] != 1 || runCounts[3] != 1 {
		t.Fatalf("expected completed steps to be skipped on rerun, got %+v", runCounts)
	}
	if provisioningRepo.startCounts[tenantID][1] != 1 || provisioningRepo.startCounts[tenantID][2] != 1 || provisioningRepo.startCounts[tenantID][3] != 1 {
		t.Fatalf("expected each step to start once, got %+v", provisioningRepo.startCounts[tenantID])
	}
}

func TestProvisionResume(t *testing.T) {
	provisioner, provisioningRepo, tenantID := newProvisionerForTest()
	runCounts := map[int]int{}
	failStepTwo := true
	provisioner.pipeline = []provisioningPipelineStep{
		{Name: "Step 1", Run: func(ctx context.Context) error { runCounts[1]++; return nil }},
		{Name: "Step 2", Run: func(ctx context.Context) error {
			runCounts[2]++
			if failStepTwo {
				return fmt.Errorf("step two failed")
			}
			return nil
		}},
		{Name: "Step 3", Run: func(ctx context.Context) error { runCounts[3]++; return nil }},
	}

	if err := provisioner.Provision(context.Background(), tenantID); err == nil {
		t.Fatal("expected first Provision to fail")
	}

	failStepTwo = false
	if err := provisioner.Provision(context.Background(), tenantID); err != nil {
		t.Fatalf("second Provision returned error: %v", err)
	}

	if runCounts[1] != 1 {
		t.Fatalf("expected first step to be skipped on resume, got %d runs", runCounts[1])
	}
	if runCounts[2] != 2 {
		t.Fatalf("expected failed step to rerun once, got %d runs", runCounts[2])
	}
	if runCounts[3] != 1 {
		t.Fatalf("expected pending step to run after resume, got %d runs", runCounts[3])
	}
	if provisioningRepo.startCounts[tenantID][1] != 1 {
		t.Fatalf("expected first step start count 1, got %d", provisioningRepo.startCounts[tenantID][1])
	}
}
