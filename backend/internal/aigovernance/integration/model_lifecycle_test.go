//go:build integration

package integration

import (
	"testing"
	"time"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
)

func TestFullLifecycle(t *testing.T) {
	h := newHarness(t)
	slug := "integration-lifecycle-" + h.tenantID.String()[:8]
	model := h.registerRuleModel(t, slug, aigovmodel.RiskTierMedium)

	v1 := h.createRuleVersion(t, model.ID, "initial production candidate")
	h.promote(t, model.ID, v1.ID, nil, false)
	h.promote(t, model.ID, v1.ID, nil, false)
	v1 = h.currentProduction(t, model.ID)

	v2 := h.createRuleVersion(t, model.ID, "shadow candidate")
	h.promote(t, model.ID, v2.ID, nil, false)
	h.promote(t, model.ID, v2.ID, nil, false)
	v2 = h.currentShadow(t, model.ID)

	for i := 0; i < 20; i++ {
		h.invokeRulePrediction(t, slug, i, true)
	}
	h.waitForPredictionCount(t, model.ID, 40)

	comparison, err := h.env.comparisonSvc.Build(h.env.ctx, h.tenantID, v1, v2, time.Hour)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if comparison.Recommendation != aigovmodel.ShadowRecommendationPromote {
		t.Fatalf("comparison recommendation = %s, want promote", comparison.Recommendation)
	}

	h.promote(t, model.ID, v2.ID, nil, false)

	currentProd := h.currentProduction(t, model.ID)
	if currentProd.ID != v2.ID {
		t.Fatalf("current production version = %s, want %s", currentProd.ID, v2.ID)
	}
	if currentProd.ReplacedVersionID == nil || *currentProd.ReplacedVersionID != v1.ID {
		t.Fatalf("replaced_version_id = %v, want %s", currentProd.ReplacedVersionID, v1.ID)
	}

	previous := h.getVersion(t, model.ID, v1.ID)
	if previous.Status != aigovmodel.VersionStatusRetired {
		t.Fatalf("previous version status = %s, want retired", previous.Status)
	}
}

func TestRollback(t *testing.T) {
	h := newHarness(t)
	slug := "integration-rollback-" + h.tenantID.String()[:8]
	model := h.registerRuleModel(t, slug, aigovmodel.RiskTierMedium)

	v1 := h.createRuleVersion(t, model.ID, "v1")
	h.promote(t, model.ID, v1.ID, nil, false)
	h.promote(t, model.ID, v1.ID, nil, false)
	v1 = h.currentProduction(t, model.ID)

	v2 := h.createRuleVersion(t, model.ID, "v2")
	h.promote(t, model.ID, v2.ID, nil, false)
	h.promote(t, model.ID, v2.ID, nil, false)
	v2 = h.currentShadow(t, model.ID)

	for i := 0; i < 12; i++ {
		h.invokeRulePrediction(t, slug, i, true)
	}
	h.waitForPredictionCount(t, model.ID, 24)
	if _, err := h.env.comparisonSvc.Build(h.env.ctx, h.tenantID, v1, v2, time.Hour); err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	h.promote(t, model.ID, v2.ID, nil, false)

	rolledBack, err := h.env.lifecycleSvc.Rollback(h.env.ctx, h.tenantID, model.ID, h.userID, "integration rollback")
	if err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
	if rolledBack.ID != v1.ID {
		t.Fatalf("rollback restored version = %s, want %s", rolledBack.ID, v1.ID)
	}

	currentProd := h.currentProduction(t, model.ID)
	if currentProd.ID != v1.ID {
		t.Fatalf("current production version = %s, want %s", currentProd.ID, v1.ID)
	}
	currentV2 := h.getVersion(t, model.ID, v2.ID)
	if currentV2.Status != aigovmodel.VersionStatusRolledBack {
		t.Fatalf("v2 status after rollback = %s, want rolled_back", currentV2.Status)
	}
}
