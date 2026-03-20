package repository

import (
	"testing"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/google/uuid"
)

func TestGroupPredictionLogsByTenant(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()

	logA1 := &aigovmodel.PredictionLog{TenantID: tenantA}
	logB1 := &aigovmodel.PredictionLog{TenantID: tenantB}
	logA2 := &aigovmodel.PredictionLog{TenantID: tenantA}

	order, grouped := groupPredictionLogsByTenant([]*aigovmodel.PredictionLog{nil, logA1, logB1, logA2})

	if len(order) != 2 {
		t.Fatalf("len(order) = %d, want 2", len(order))
	}
	if order[0] != tenantA {
		t.Fatalf("order[0] = %s, want %s", order[0], tenantA)
	}
	if order[1] != tenantB {
		t.Fatalf("order[1] = %s, want %s", order[1], tenantB)
	}
	if len(grouped[tenantA]) != 2 {
		t.Fatalf("len(grouped[tenantA]) = %d, want 2", len(grouped[tenantA]))
	}
	if grouped[tenantA][0] != logA1 || grouped[tenantA][1] != logA2 {
		t.Fatal("tenantA logs were not preserved in input order")
	}
	if len(grouped[tenantB]) != 1 || grouped[tenantB][0] != logB1 {
		t.Fatal("tenantB logs were not grouped correctly")
	}
}
