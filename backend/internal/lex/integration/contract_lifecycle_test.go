//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/clario360/platform/internal/lex/dto"
	"github.com/clario360/platform/internal/lex/model"
)

func TestContractFullLifecycle(t *testing.T) {
	t.Parallel()

	h := newLexHarness(t)
	contract := h.createContractWithText(t, "Managed Services Lifecycle", model.ContractTypeServiceAgreement, 1_250_000, lifecycleContractText())
	if contract.Status != model.ContractStatusDraft {
		t.Fatalf("created contract status = %s, want %s", contract.Status, model.ContractStatusDraft)
	}

	revisedText := lifecycleContractText() + "\n\n" + clauseSection(11, "Assignment", "Neither party may assign this agreement without prior written consent except for internal reorganizations.")
	uploadedVersions := h.uploadContractDocument(t, contract.ID, "managed-services-lifecycle-v2.txt", revisedText, "Updated negotiation draft.")
	if len(uploadedVersions) != 2 {
		t.Fatalf("uploaded versions = %d, want 2", len(uploadedVersions))
	}
	if uploadedVersions[0].Version != 2 {
		t.Fatalf("latest version after upload = %d, want 2", uploadedVersions[0].Version)
	}

	result := h.analyzeContract(t, contract.ID)
	if result.Analysis == nil {
		t.Fatal("expected analysis payload")
	}
	if len(result.Clauses) < 10 {
		t.Fatalf("analysis clauses = %d, want at least 10", len(result.Clauses))
	}

	workflow := h.startReview(t, contract.ID, "Internal legal review for lifecycle coverage.")
	if workflow.ContractStatus != model.ContractStatusInternalReview {
		t.Fatalf("workflow contract status = %s, want %s", workflow.ContractStatus, model.ContractStatusInternalReview)
	}

	workflows := mustPaginated[model.LegalWorkflowSummary](t, h.doJSON(t, http.MethodGet, "/api/v1/lex/workflows?page=1&per_page=10", nil), http.StatusOK)
	if workflows.Pagination.Total == 0 {
		t.Fatal("expected at least one active workflow")
	}
	foundWorkflow := false
	for _, item := range workflows.Data {
		if item.ContractID == contract.ID {
			foundWorkflow = true
			break
		}
	}
	if !foundWorkflow {
		t.Fatalf("workflow for contract %s not found in %+v", contract.ID, workflows.Data)
	}

	contract = h.updateContractStatus(t, contract.ID, model.ContractStatusLegalReview)
	contract = h.updateContractStatus(t, contract.ID, model.ContractStatusNegotiation)
	contract = h.updateContractStatus(t, contract.ID, model.ContractStatusPendingSignature)
	contract = h.updateContractStatus(t, contract.ID, model.ContractStatusActive)
	if contract.Status != model.ContractStatusActive {
		t.Fatalf("final contract status = %s, want %s", contract.Status, model.ContractStatusActive)
	}

	analysis := mustData[model.ContractRiskAnalysis](t, h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/lex/contracts/%s/analysis", contract.ID), nil), http.StatusOK)
	if analysis.ClauseCount != len(result.Clauses) {
		t.Fatalf("analysis clause count = %d, want %d", analysis.ClauseCount, len(result.Clauses))
	}

	detail := mustData[model.ContractDetail](t, h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/lex/contracts/%s", contract.ID), nil), http.StatusOK)
	if detail.Contract.Status != model.ContractStatusActive {
		t.Fatalf("detail status = %s, want %s", detail.Contract.Status, model.ContractStatusActive)
	}
	if detail.VersionCount != 2 {
		t.Fatalf("detail version count = %d, want 2", detail.VersionCount)
	}
	if detail.LatestAnalysis == nil {
		t.Fatal("expected latest analysis in contract detail")
	}
	if len(detail.Clauses) != len(result.Clauses) {
		t.Fatalf("detail clauses = %d, want %d", len(detail.Clauses), len(result.Clauses))
	}

	versions := mustData[[]model.ContractVersion](t, h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/lex/contracts/%s/versions", contract.ID), nil), http.StatusOK)
	if len(versions) != 2 {
		t.Fatalf("listed versions = %d, want 2", len(versions))
	}
}

func TestContractAnalysis(t *testing.T) {
	t.Parallel()

	h := newLexHarness(t)
	contract := h.createContractWithText(t, "Targeted Analysis Contract", model.ContractTypeServiceAgreement, 2_400_000, targetedAnalysisText())
	result := h.analyzeContract(t, contract.ID)

	if result.Analysis == nil {
		t.Fatal("expected analysis result")
	}
	if result.Analysis.RiskScore <= 55 {
		t.Fatalf("risk score = %.2f, want > 55", result.Analysis.RiskScore)
	}
	if len(result.Analysis.Recommendations) == 0 {
		t.Fatal("expected recommendations in analysis result")
	}

	foundPIIFlag := false
	for _, flag := range result.Analysis.ComplianceFlags {
		if flag.Code == "pii_without_data_protection" {
			foundPIIFlag = true
			break
		}
	}
	if !foundPIIFlag {
		t.Fatalf("expected pii_without_data_protection flag in %+v", result.Analysis.ComplianceFlags)
	}

	foundCriticalLimitation := false
	for _, clause := range result.Clauses {
		if clause.ClauseType == model.ClauseTypeLimitationOfLiability && clause.RiskLevel == model.RiskLevelCritical {
			foundCriticalLimitation = true
			if len(clause.Recommendations) == 0 {
				t.Fatal("expected limitation clause recommendations")
			}
			if clause.SectionReference == "" {
				t.Fatal("expected limitation clause section reference")
			}
		}
	}
	if !foundCriticalLimitation {
		t.Fatalf("expected critical limitation_of_liability clause in %+v", result.Clauses)
	}

	clauses := mustData[[]model.Clause](t, h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/lex/contracts/%s/clauses", contract.ID), nil), http.StatusOK)
	if len(clauses) < 4 {
		t.Fatalf("persisted clauses = %d, want at least 4", len(clauses))
	}
}

func TestContractRenewal(t *testing.T) {
	t.Parallel()

	h := newLexHarness(t)
	original := createActiveContractForMonitor(t, h, "Renewal Coverage Contract", time.Now().UTC().Add(20*24*time.Hour), false)
	newValue := 375000.0
	newExpiry := time.Now().UTC().AddDate(1, 0, 0)

	renewed := mustData[model.Contract](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/lex/contracts/%s/renew", original.ID), dto.RenewContractRequest{
		NewExpiryDate: newExpiry,
		NewValue:      &newValue,
		ChangeSummary: "Renewed for an additional annual term.",
	}), http.StatusCreated)

	if renewed.Status != model.ContractStatusDraft {
		t.Fatalf("renewed contract status = %s, want %s", renewed.Status, model.ContractStatusDraft)
	}
	if renewed.ParentContractID == nil || *renewed.ParentContractID != original.ID {
		t.Fatalf("renewed parent contract id = %v, want %s", renewed.ParentContractID, original.ID)
	}
	if !strings.Contains(renewed.Title, "(Renewal)") {
		t.Fatalf("renewed title = %q, want renewal suffix", renewed.Title)
	}
	if renewed.TotalValue == nil || *renewed.TotalValue != newValue {
		t.Fatalf("renewed total value = %v, want %.2f", renewed.TotalValue, newValue)
	}

	versions := mustData[[]model.ContractVersion](t, h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/lex/contracts/%s/versions", renewed.ID), nil), http.StatusOK)
	if len(versions) != 1 {
		t.Fatalf("renewed contract versions = %d, want 1", len(versions))
	}

	updatedOriginal := mustData[model.ContractDetail](t, h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/lex/contracts/%s", original.ID), nil), http.StatusOK)
	if updatedOriginal.Contract.Status != model.ContractStatusRenewed {
		t.Fatalf("original contract status = %s, want %s", updatedOriginal.Contract.Status, model.ContractStatusRenewed)
	}
}

func TestContractSearch(t *testing.T) {
	t.Parallel()

	h := newLexHarness(t)
	created := make(map[string]struct{}, 5)
	for idx := 0; idx < 5; idx++ {
		title := fmt.Sprintf("Vendor Search Contract %d", idx+1)
		contract := h.createContract(t, h.baseContractRequest(title, model.ContractTypeVendor, 120000+float64(idx*1000), ""))
		created[contract.ID.String()] = struct{}{}
	}

	search := mustPaginated[model.ContractSummary](t, h.doJSON(t, http.MethodGet, "/api/v1/lex/contracts/search?q=vendor&page=1&per_page=10", nil), http.StatusOK)
	if search.Pagination.Total < 5 {
		t.Fatalf("search total = %d, want at least 5", search.Pagination.Total)
	}

	found := 0
	for _, item := range search.Data {
		if _, ok := created[item.ID.String()]; ok {
			found++
		}
	}
	if found != 5 {
		t.Fatalf("found created search contracts = %d, want 5 in %+v", found, search.Data)
	}
}

func TestAnalysis_Under3s(t *testing.T) {
	h := newLexHarness(t)
	contract := h.createContractWithText(t, "Performance Analysis Contract", model.ContractTypeServiceAgreement, 4_500_000, largeContractText())

	startedAt := time.Now()
	result := h.analyzeContract(t, contract.ID)
	elapsed := time.Since(startedAt)

	if elapsed >= 3*time.Second {
		t.Fatalf("analysis duration = %s, want < 3s", elapsed)
	}
	if result.Analysis == nil {
		t.Fatal("expected analysis result")
	}
	if len(result.Clauses) == 0 {
		t.Fatal("expected extracted clauses from large contract text")
	}
}
