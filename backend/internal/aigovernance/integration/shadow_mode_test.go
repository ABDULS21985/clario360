//go:build integration

package integration

import (
	"testing"
	"time"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
)

func TestShadowExecution(t *testing.T) {
	h := newHarness(t)
	slug := "integration-shadow-exec-" + h.tenantID.String()[:8]
	model := h.registerRuleModel(t, slug, aigovmodel.RiskTierMedium)

	v1 := h.createRuleVersion(t, model.ID, "v1")
	h.promote(t, model.ID, v1.ID, nil, false)
	h.promote(t, model.ID, v1.ID, nil, false)

	v2 := h.createRuleVersion(t, model.ID, "v2")
	h.promote(t, model.ID, v2.ID, nil, false)
	h.promote(t, model.ID, v2.ID, nil, false)

	h.invokeRulePrediction(t, slug, 1, true)
	logs := h.waitForPredictionCount(t, model.ID, 2)

	shadowCount := 0
	productionCount := 0
	for _, item := range logs {
		if item.IsShadow {
			shadowCount++
		} else {
			productionCount++
		}
	}
	if productionCount != 1 || shadowCount != 1 {
		t.Fatalf("production/shadow counts = %d/%d, want 1/1", productionCount, shadowCount)
	}
}

func TestShadowComparison(t *testing.T) {
	h := newHarness(t)
	slug := "integration-shadow-compare-" + h.tenantID.String()[:8]
	model := h.registerRuleModel(t, slug, aigovmodel.RiskTierMedium)

	v1 := h.createRuleVersion(t, model.ID, "v1")
	h.promote(t, model.ID, v1.ID, nil, false)
	h.promote(t, model.ID, v1.ID, nil, false)
	v1 = h.currentProduction(t, model.ID)

	v2 := h.createRuleVersion(t, model.ID, "v2")
	h.promote(t, model.ID, v2.ID, nil, false)
	h.promote(t, model.ID, v2.ID, nil, false)
	v2 = h.currentShadow(t, model.ID)

	for i := 0; i < 100; i++ {
		h.invokeRulePrediction(t, slug, i, true)
	}
	h.waitForPredictionCount(t, model.ID, 200)

	comparison, err := h.env.comparisonSvc.Build(h.env.ctx, h.tenantID, v1, v2, time.Hour)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if comparison.TotalPredictions != 100 {
		t.Fatalf("total_predictions = %d, want 100", comparison.TotalPredictions)
	}
	if comparison.AgreementRate < 0.99 {
		t.Fatalf("agreement_rate = %.2f, want >= 0.99", comparison.AgreementRate)
	}
	if comparison.Recommendation != aigovmodel.ShadowRecommendationPromote {
		t.Fatalf("recommendation = %s, want promote", comparison.Recommendation)
	}
}
