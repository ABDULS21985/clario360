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

func TestExpiryMonitor(t *testing.T) {
	t.Parallel()

	h := newLexHarness(t)
	contract := createActiveContractForMonitor(t, h, "Expiry Monitor Coverage", time.Now().UTC().Add(25*24*time.Hour), false)

	runExpiryMonitor(t, h)

	alerts := mustPaginated[model.ComplianceAlert](t, h.doJSON(t, http.MethodGet, "/api/v1/lex/compliance/alerts?severity=high&page=1&per_page=20", nil), http.StatusOK)
	found := false
	for _, alert := range alerts.Data {
		if alert.ContractID != nil && *alert.ContractID == contract.ID && alert.Severity == model.ComplianceSeverityHigh {
			found = true
			if !strings.Contains(alert.Title, "expiring in 30 days") {
				t.Fatalf("expiry alert title = %q, want 30-day horizon wording", alert.Title)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected expiry alert for contract %s in %+v", contract.ID, alerts.Data)
	}
}

func TestExpiryNoDuplicate(t *testing.T) {
	t.Parallel()

	h := newLexHarness(t)
	contract := createActiveContractForMonitor(t, h, "Expiry Dedup Contract", time.Now().UTC().Add(25*24*time.Hour), false)

	runExpiryMonitor(t, h)
	runExpiryMonitor(t, h)

	notificationCount := h.scalarInt(t, `SELECT COUNT(*) FROM expiry_notifications WHERE contract_id = $1 AND horizon_days = 30`, contract.ID)
	if notificationCount != 1 {
		t.Fatalf("expiry notification count = %d, want 1", notificationCount)
	}

	alertCount := h.scalarInt(t, `SELECT COUNT(*) FROM compliance_alerts WHERE contract_id = $1 AND dedup_key = $2`, contract.ID, fmt.Sprintf("expiry:%s:%d", contract.ID, 30))
	if alertCount != 1 {
		t.Fatalf("expiry alert count = %d, want 1", alertCount)
	}
}

func TestAutoExpiry(t *testing.T) {
	t.Parallel()

	h := newLexHarness(t)
	contract := createActiveContractForMonitor(t, h, "Auto Expiry Contract", time.Now().UTC().Add(-48*time.Hour), false)

	runExpiryMonitor(t, h)

	detail := mustData[model.ContractDetail](t, h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/lex/contracts/%s", contract.ID), nil), http.StatusOK)
	if detail.Contract.Status != model.ContractStatusExpired {
		t.Fatalf("contract status after expiry monitor = %s, want %s", detail.Contract.Status, model.ContractStatusExpired)
	}
}

func TestComplianceRules(t *testing.T) {
	t.Parallel()

	h := newLexHarness(t)
	contract := h.createContractWithText(t, "Risk Threshold Compliance Contract", model.ContractTypeServiceAgreement, 2_600_000, highRiskClauseText())
	analysis := h.analyzeContract(t, contract.ID)
	if analysis.Analysis == nil || analysis.Analysis.RiskScore <= 70 {
		t.Fatalf("risk score = %v, want > 70", analysis.Analysis)
	}

	rule := mustData[model.ComplianceRule](t, h.doJSON(t, http.MethodPost, "/api/v1/lex/compliance/rules", dto.CreateComplianceRuleRequest{
		Name:        "High risk requires legal review",
		Description: "Contracts above 70 risk must be in legal_review status.",
		RuleType:    model.ComplianceRuleRiskThreshold,
		Severity:    model.ComplianceSeverityCritical,
		Config: map[string]any{
			"min_score":       70,
			"required_status": string(model.ContractStatusLegalReview),
		},
		Enabled: true,
	}), http.StatusCreated)
	if rule.RuleType != model.ComplianceRuleRiskThreshold {
		t.Fatalf("created rule type = %s, want %s", rule.RuleType, model.ComplianceRuleRiskThreshold)
	}

	result := mustData[model.ComplianceRunResult](t, h.doJSON(t, http.MethodPost, "/api/v1/lex/compliance/run", dto.RunComplianceRequest{
		ContractIDs: []string{contract.ID.String()},
	}), http.StatusOK)
	if result.AlertsCreated == 0 {
		t.Fatal("expected compliance run to create at least one alert")
	}

	alerts := mustPaginated[model.ComplianceAlert](t, h.doJSON(t, http.MethodGet, "/api/v1/lex/compliance/alerts?severity=critical&page=1&per_page=20", nil), http.StatusOK)
	found := false
	for _, alert := range alerts.Data {
		if alert.ContractID != nil && *alert.ContractID == contract.ID && alert.RuleID != nil && *alert.RuleID == rule.ID {
			found = true
			if !strings.Contains(strings.ToLower(alert.Description), "risk score") {
				t.Fatalf("unexpected alert description: %q", alert.Description)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected risk threshold alert for contract %s in %+v", contract.ID, alerts.Data)
	}
}
