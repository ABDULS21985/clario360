//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/clario360/platform/internal/lex/dto"
	"github.com/clario360/platform/internal/lex/model"
)

func TestClauseExtraction(t *testing.T) {
	t.Parallel()

	h := newLexHarness(t)
	contract := h.createContractWithText(t, "Eight Clause Extraction Contract", model.ContractTypeServiceAgreement, 950000, eightClauseText())
	h.analyzeContract(t, contract.ID)

	clauses := mustData[[]model.Clause](t, h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/lex/contracts/%s/clauses", contract.ID), nil), http.StatusOK)
	if len(clauses) != 8 {
		t.Fatalf("clause count = %d, want 8", len(clauses))
	}

	expected := map[model.ClauseType]struct{}{
		model.ClauseTypeTermination:           {},
		model.ClauseTypeIndemnification:       {},
		model.ClauseTypeLimitationOfLiability: {},
		model.ClauseTypeConfidentiality:       {},
		model.ClauseTypeForceMajeure:          {},
		model.ClauseTypeDisputeResolution:     {},
		model.ClauseTypeDataProtection:        {},
		model.ClauseTypeAuditRights:           {},
	}
	for _, clause := range clauses {
		delete(expected, clause.ClauseType)
		if clause.SectionReference == nil || *clause.SectionReference == "" {
			t.Fatalf("expected section reference for clause %+v", clause)
		}
	}
	if len(expected) != 0 {
		t.Fatalf("missing expected clause types: %+v", expected)
	}
}

func TestClauseReview(t *testing.T) {
	t.Parallel()

	h := newLexHarness(t)
	contract := h.createContractWithText(t, "Clause Review Contract", model.ContractTypeServiceAgreement, 850000, eightClauseText())
	h.analyzeContract(t, contract.ID)

	clauses := mustData[[]model.Clause](t, h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/lex/contracts/%s/clauses", contract.ID), nil), http.StatusOK)
	if len(clauses) == 0 {
		t.Fatal("expected clauses for review")
	}

	reviewed := mustData[model.Clause](t, h.doJSON(t, http.MethodPut, fmt.Sprintf("/api/v1/lex/contracts/%s/clauses/%s/review", contract.ID, clauses[0].ID), dto.UpdateClauseReviewRequest{
		Status: model.ClauseReviewFlagged,
		Notes:  "Escalate this clause for manual legal review.",
	}), http.StatusOK)
	if reviewed.ReviewStatus != model.ClauseReviewFlagged {
		t.Fatalf("review status = %s, want %s", reviewed.ReviewStatus, model.ClauseReviewFlagged)
	}
	if reviewed.ReviewNotes == nil || *reviewed.ReviewNotes != "Escalate this clause for manual legal review." {
		t.Fatalf("review notes = %v, want persisted notes", reviewed.ReviewNotes)
	}

	detail := mustData[model.Clause](t, h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/lex/contracts/%s/clauses/%s", contract.ID, clauses[0].ID), nil), http.StatusOK)
	if detail.ReviewStatus != model.ClauseReviewFlagged {
		t.Fatalf("clause detail review status = %s, want %s", detail.ReviewStatus, model.ClauseReviewFlagged)
	}
}

func TestHighRiskSummary(t *testing.T) {
	t.Parallel()

	h := newLexHarness(t)
	contract := h.createContractWithText(t, "High Risk Clause Summary Contract", model.ContractTypeServiceAgreement, 1_900_000, highRiskClauseText())
	h.analyzeContract(t, contract.ID)

	summary := mustData[[]model.Clause](t, h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/lex/contracts/%s/clauses/risks", contract.ID), nil), http.StatusOK)
	if len(summary) != 3 {
		t.Fatalf("high risk summary size = %d, want 3", len(summary))
	}

	foundCritical := false
	for _, clause := range summary {
		if clause.RiskLevel != model.RiskLevelHigh && clause.RiskLevel != model.RiskLevelCritical {
			t.Fatalf("unexpected risk level in summary: %+v", clause)
		}
		if clause.ClauseType == model.ClauseTypeLimitationOfLiability && clause.RiskLevel == model.RiskLevelCritical {
			foundCritical = true
		}
	}
	if !foundCritical {
		t.Fatalf("expected critical limitation_of_liability clause in %+v", summary)
	}
}
