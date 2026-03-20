package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	iammodel "github.com/clario360/platform/internal/iam/model"
	onboardingdto "github.com/clario360/platform/internal/onboarding/dto"
)

func TestDeprovisionSuspendsUsers(t *testing.T) {
	tenantID := uuid.New()
	adminID := uuid.New()
	onboardingRepo := newFakeOnboardingRepo()
	onboardingRepo.tenantIdentities[tenantID] = fakeTenantIdentity{
		name:   "Acme Corp",
		slug:   "acme-corp-a1b2",
		status: iammodel.TenantStatusActive,
	}
	store := &fakeLifecycleStore{}
	service := &TenantDeprovisioner{
		store:          store,
		onboardingRepo: onboardingRepo,
		logger:         newDiscardLogger(),
	}

	if err := service.Deprovision(context.Background(), tenantID, adminID, onboardingdto.DeprovisionRequest{
		Reason:     "Subscription cancelled",
		RetainDays: 90,
	}); err != nil {
		t.Fatalf("Deprovision returned error: %v", err)
	}

	if len(store.suspendCalls) != 1 || store.suspendCalls[0] != tenantID {
		t.Fatalf("expected suspend call for tenant %s, got %+v", tenantID, store.suspendCalls)
	}
	if len(store.deleteSessionCalls) != 1 || store.deleteSessionCalls[0] != tenantID {
		t.Fatalf("expected delete sessions call for tenant %s, got %+v", tenantID, store.deleteSessionCalls)
	}
}

func TestDeprovisionSoftDeletes(t *testing.T) {
	tenantID := uuid.New()
	onboardingRepo := newFakeOnboardingRepo()
	onboardingRepo.tenantIdentities[tenantID] = fakeTenantIdentity{
		name:   "Acme Corp",
		slug:   "acme-corp-a1b2",
		status: iammodel.TenantStatusActive,
	}
	store := &fakeLifecycleStore{}
	service := &TenantDeprovisioner{
		store:          store,
		onboardingRepo: onboardingRepo,
		logger:         newDiscardLogger(),
	}

	if err := service.Deprovision(context.Background(), tenantID, uuid.New(), onboardingdto.DeprovisionRequest{
		Reason:     "Retention requested",
		RetainDays: 90,
	}); err != nil {
		t.Fatalf("Deprovision returned error: %v", err)
	}

	if len(store.softDeleteCalls) != 1 || store.softDeleteCalls[0] != tenantID {
		t.Fatalf("expected soft delete call for tenant %s, got %+v", tenantID, store.softDeleteCalls)
	}
}

func TestDeprovisionTenantStatus(t *testing.T) {
	tenantID := uuid.New()
	adminID := uuid.New()
	onboardingRepo := newFakeOnboardingRepo()
	onboardingRepo.tenantIdentities[tenantID] = fakeTenantIdentity{
		name:   "Acme Corp",
		slug:   "acme-corp-a1b2",
		status: iammodel.TenantStatusActive,
	}
	store := &fakeLifecycleStore{}
	service := &TenantDeprovisioner{
		store:          store,
		onboardingRepo: onboardingRepo,
		logger:         newDiscardLogger(),
	}

	if err := service.Deprovision(context.Background(), tenantID, adminID, onboardingdto.DeprovisionRequest{
		Reason:     "Contract ended",
		RetainDays: 30,
	}); err != nil {
		t.Fatalf("Deprovision returned error: %v", err)
	}

	if len(store.markedDeprovisioned) != 1 || store.markedDeprovisioned[0] != tenantID {
		t.Fatalf("expected tenant deprovision mark for %s, got %+v", tenantID, store.markedDeprovisioned)
	}
	if len(store.auditRecords) != 1 || store.auditRecords[0].action != "tenant.deprovisioned" {
		t.Fatalf("expected deprovision audit record, got %+v", store.auditRecords)
	}
	if store.lastDeprovisionAdminID != adminID {
		t.Fatalf("expected deprovision admin %s, got %s", adminID, store.lastDeprovisionAdminID)
	}
	if store.lastRetainUntil.Before(time.Now()) {
		t.Fatal("expected retain until to be set in the future")
	}
}

func TestDeprovisionAlreadyDeprovisioned(t *testing.T) {
	tenantID := uuid.New()
	onboardingRepo := newFakeOnboardingRepo()
	onboardingRepo.tenantIdentities[tenantID] = fakeTenantIdentity{
		name:   "Acme Corp",
		slug:   "acme-corp-a1b2",
		status: iammodel.TenantStatusDeprovisioned,
	}
	service := &TenantDeprovisioner{
		store:          &fakeLifecycleStore{},
		onboardingRepo: onboardingRepo,
		logger:         newDiscardLogger(),
	}

	err := service.Deprovision(context.Background(), tenantID, uuid.New(), onboardingdto.DeprovisionRequest{
		Reason:     "Duplicate request",
		RetainDays: 90,
	})
	if !errors.Is(err, iammodel.ErrConflict) {
		t.Fatalf("expected conflict error, got %v", err)
	}
}
