package cti

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/auth"
	apperrors "github.com/clario360/platform/internal/errors"
)

func TestParseAggregationRefreshScope(t *testing.T) {
	scope, err := ParseAggregationRefreshScope("all")
	if err != nil {
		t.Fatalf("ParseAggregationRefreshScope(all): %v", err)
	}
	if scope != AggregationScopeAllTenants {
		t.Fatalf("expected all_tenants scope, got %q", scope)
	}

	scope, err = ParseAggregationRefreshScope("")
	if err != nil {
		t.Fatalf("ParseAggregationRefreshScope(default): %v", err)
	}
	if scope != AggregationScopeTenant {
		t.Fatalf("expected tenant scope, got %q", scope)
	}
}

func TestServiceRefreshAggregationsAllTenantsRequiresSuperAdmin(t *testing.T) {
	svc := &Service{aggEngine: &aggEngineStub{}}
	ctx := auth.WithTenantID(context.Background(), uuid.NewString())
	ctx = auth.WithUser(ctx, &auth.ContextUser{
		ID:       uuid.NewString(),
		TenantID: auth.TenantFromContext(ctx),
		Email:    "tenant-admin@example.com",
		Roles:    []string{"tenant-admin"},
	})

	err := svc.RefreshAggregations(ctx, AggregationScopeAllTenants)
	if err == nil {
		t.Fatal("expected forbidden error")
	}
	if apperrors.HTTPStatus(err) != 403 {
		t.Fatalf("expected 403, got %d (%v)", apperrors.HTTPStatus(err), err)
	}
}

func TestServiceRefreshAggregationsAllTenantsDelegatesToEngine(t *testing.T) {
	engine := &aggEngineStub{}
	svc := &Service{aggEngine: engine}
	ctx := auth.WithTenantID(context.Background(), uuid.NewString())
	ctx = auth.WithUser(ctx, &auth.ContextUser{
		ID:       uuid.NewString(),
		TenantID: auth.TenantFromContext(ctx),
		Email:    "super-admin@example.com",
		Roles:    []string{"super-admin"},
	})

	if err := svc.RefreshAggregations(ctx, AggregationScopeAllTenants); err != nil {
		t.Fatalf("RefreshAggregations(all): %v", err)
	}
	if !engine.runAllCalled {
		t.Fatal("expected RunAllTenants to be called")
	}
}

func TestServiceRefreshAggregationsTenantScopeUsesContextTenant(t *testing.T) {
	engine := &aggEngineStub{}
	svc := &Service{aggEngine: engine}
	tenantID := uuid.NewString()
	ctx := auth.WithTenantID(context.Background(), tenantID)
	ctx = auth.WithUser(ctx, &auth.ContextUser{
		ID:       uuid.NewString(),
		TenantID: tenantID,
		Email:    "analyst@example.com",
		Roles:    []string{"analyst"},
	})

	if err := svc.RefreshAggregations(ctx, AggregationScopeTenant); err != nil {
		t.Fatalf("RefreshAggregations(tenant): %v", err)
	}
	if len(engine.runFullArgs) != 1 || engine.runFullArgs[0] != tenantID {
		t.Fatalf("expected RunFullAggregation to be called with tenant %s, got %+v", tenantID, engine.runFullArgs)
	}
}

type aggEngineStub struct {
	runAllCalled bool
	runFullArgs  []string
}

func (s *aggEngineStub) RunFullAggregation(_ context.Context, tenantID string) error {
	s.runFullArgs = append(s.runFullArgs, tenantID)
	return nil
}

func (s *aggEngineStub) RunAllTenants(_ context.Context) error {
	s.runAllCalled = true
	return nil
}
