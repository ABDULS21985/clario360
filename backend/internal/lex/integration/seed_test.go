//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/lex/model"
)

func TestLiveMigrationsAndSeed(t *testing.T) {
	h := newDemoHarness(t)
	seedTenantID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	contracts := h.scalarInt(t, `SELECT COUNT(*) FROM contracts WHERE tenant_id = $1 AND deleted_at IS NULL`, seedTenantID)
	if contracts != 20 {
		t.Fatalf("seeded contracts = %d, want 20", contracts)
	}

	clauses := h.scalarInt(t, `SELECT COUNT(*) FROM contract_clauses WHERE tenant_id = $1`, seedTenantID)
	if clauses != 60 {
		t.Fatalf("seeded clauses = %d, want 60", clauses)
	}

	alerts := h.scalarInt(t, `SELECT COUNT(*) FROM compliance_alerts WHERE tenant_id = $1`, seedTenantID)
	if alerts != 10 {
		t.Fatalf("seeded alerts = %d, want 10", alerts)
	}

	rules := h.scalarInt(t, `SELECT COUNT(*) FROM compliance_rules WHERE tenant_id = $1 AND deleted_at IS NULL`, seedTenantID)
	if rules != 5 {
		t.Fatalf("seeded compliance rules = %d, want 5", rules)
	}

	documents := h.scalarInt(t, `SELECT COUNT(*) FROM legal_documents WHERE tenant_id = $1 AND deleted_at IS NULL`, seedTenantID)
	if documents != 8 {
		t.Fatalf("seeded legal documents = %d, want 8", documents)
	}

	documentVersions := h.scalarInt(t, `SELECT COUNT(*) FROM document_versions WHERE tenant_id = $1`, seedTenantID)
	if documentVersions != 10 {
		t.Fatalf("seeded document versions = %d, want 10", documentVersions)
	}

	autoRenewContracts := h.scalarInt(t, `SELECT COUNT(*) FROM contracts WHERE tenant_id = $1 AND auto_renew = true AND deleted_at IS NULL`, seedTenantID)
	if autoRenewContracts != 3 {
		t.Fatalf("seeded auto-renew contracts = %d, want 3", autoRenewContracts)
	}

	expiringActive := h.scalarInt(t, `SELECT COUNT(*) FROM contracts WHERE tenant_id = $1 AND status = 'active' AND expiry_date IS NOT NULL AND expiry_date <= CURRENT_DATE + 30 AND deleted_at IS NULL`, seedTenantID)
	if expiringActive != 5 {
		t.Fatalf("seeded active contracts expiring in 30 days = %d, want 5", expiringActive)
	}

	stats := mustData[model.ContractStats](t, h.doJSON(t, http.MethodGet, "/api/v1/lex/contracts/stats", nil), http.StatusOK)
	if stats.ByType[string(model.ContractTypeServiceAgreement)] != 5 {
		t.Fatalf("service_agreement count = %d, want 5", stats.ByType[string(model.ContractTypeServiceAgreement)])
	}
	if stats.ByType[string(model.ContractTypeNDA)] != 4 {
		t.Fatalf("nda count = %d, want 4", stats.ByType[string(model.ContractTypeNDA)])
	}
	if stats.ByStatus[string(model.ContractStatusActive)] != 14 {
		t.Fatalf("active contract count = %d, want 14", stats.ByStatus[string(model.ContractStatusActive)])
	}
	if stats.ByStatus[string(model.ContractStatusExpired)] != 2 {
		t.Fatalf("expired contract count = %d, want 2", stats.ByStatus[string(model.ContractStatusExpired)])
	}
	if stats.Expiring30Days != 5 {
		t.Fatalf("stats expiring 30 days = %d, want 5", stats.Expiring30Days)
	}
}
