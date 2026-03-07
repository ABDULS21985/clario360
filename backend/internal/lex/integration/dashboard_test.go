//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/clario360/platform/internal/lex/model"
)

func TestDashboard(t *testing.T) {
	h := newDemoHarness(t)

	first := mustData[model.LexDashboard](t, h.doJSON(t, http.MethodGet, "/api/v1/lex/dashboard", nil), http.StatusOK)
	second := mustData[model.LexDashboard](t, h.doJSON(t, http.MethodGet, "/api/v1/lex/dashboard", nil), http.StatusOK)

	if first.KPIs.ActiveContracts == 0 {
		t.Fatal("expected dashboard active contracts KPI to be populated")
	}
	if first.KPIs.ExpiringIn30Days == 0 || first.KPIs.ExpiringIn7Days == 0 {
		t.Fatalf("expected expiring KPIs to be populated: %+v", first.KPIs)
	}
	if first.KPIs.HighRiskContracts == 0 {
		t.Fatalf("expected high risk KPI to be populated: %+v", first.KPIs)
	}
	if first.KPIs.ComplianceScore <= 0 {
		t.Fatalf("expected compliance score > 0, got %+v", first.KPIs)
	}
	if len(first.ContractsByType) == 0 || len(first.ContractsByStatus) == 0 {
		t.Fatalf("expected dashboard contract distributions, got %+v %+v", first.ContractsByType, first.ContractsByStatus)
	}
	if len(first.ExpiringContracts) == 0 || len(first.HighRiskContracts) == 0 || len(first.RecentContracts) == 0 {
		t.Fatalf("expected populated dashboard lists, got %+v %+v %+v", first.ExpiringContracts, first.HighRiskContracts, first.RecentContracts)
	}
	if len(first.ComplianceAlertsByStatus) == 0 {
		t.Fatal("expected compliance alerts by status to be populated")
	}
	if len(first.TotalContractValue.ByType) == 0 || len(first.TotalContractValue.ByCurrency) == 0 {
		t.Fatalf("expected total contract value breakdowns, got %+v", first.TotalContractValue)
	}
	if len(first.MonthlyActivity) != 12 {
		t.Fatalf("monthly activity size = %d, want 12", len(first.MonthlyActivity))
	}
	if !first.CalculatedAt.Equal(second.CalculatedAt) {
		t.Fatalf("expected cached dashboard timestamp to remain stable, first=%s second=%s", first.CalculatedAt, second.CalculatedAt)
	}

	complianceDashboard := mustData[model.ComplianceDashboard](t, h.doJSON(t, http.MethodGet, "/api/v1/lex/compliance/dashboard", nil), http.StatusOK)
	if complianceDashboard.OpenAlerts == 0 {
		t.Fatalf("expected compliance dashboard open alerts > 0, got %+v", complianceDashboard)
	}
	if complianceDashboard.ComplianceScore <= 0 {
		t.Fatalf("expected compliance dashboard score > 0, got %+v", complianceDashboard)
	}

	score := mustData[model.ComplianceScore](t, h.doJSON(t, http.MethodGet, "/api/v1/lex/compliance/score", nil), http.StatusOK)
	if score.Score <= 0 {
		t.Fatalf("expected compliance score > 0, got %+v", score)
	}
}
