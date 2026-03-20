//go:build integration

package integration

import (
	"testing"

	aigovdto "github.com/clario360/platform/internal/aigovernance/dto"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
)

func TestPredictionLogged(t *testing.T) {
	h := newHarness(t)
	slug := "integration-prediction-" + h.tenantID.String()[:8]
	model := h.registerRuleModel(t, slug, aigovmodel.RiskTierMedium)

	version := h.createRuleVersion(t, model.ID, "production")
	h.promote(t, model.ID, version.ID, nil, false)
	h.promote(t, model.ID, version.ID, nil, false)

	h.invokeRulePrediction(t, slug, 1, false)
	logs := h.waitForPredictionCount(t, model.ID, 1)
	entry := logs[0]
	if entry.InputHash == "" {
		t.Fatal("prediction input_hash is empty")
	}
	if len(entry.Prediction) == 0 {
		t.Fatal("prediction payload is empty")
	}
	if entry.Confidence == nil {
		t.Fatal("prediction confidence is nil")
	}
	if entry.ExplanationText == "" {
		t.Fatal("prediction explanation text is empty")
	}
}

func TestPredictionFeedback(t *testing.T) {
	h := newHarness(t)
	slug := "integration-feedback-" + h.tenantID.String()[:8]
	model := h.registerRuleModel(t, slug, aigovmodel.RiskTierMedium)

	version := h.createRuleVersion(t, model.ID, "production")
	h.promote(t, model.ID, version.ID, nil, false)
	h.promote(t, model.ID, version.ID, nil, false)

	h.invokeRulePrediction(t, slug, 1, false)
	logs := h.waitForPredictionCount(t, model.ID, 1)
	entry := logs[0]

	err := h.env.predictionSvc.SubmitFeedback(h.env.ctx, h.tenantID, h.userID, entry.ID, aigovdto.PredictionFeedbackRequest{
		Correct:         false,
		CorrectedOutput: []byte(`{"matched":false}`),
		Notes:           "integration correction",
	})
	if err != nil {
		t.Fatalf("SubmitFeedback() error = %v", err)
	}

	updated, err := h.env.predictionSvc.Get(h.env.ctx, h.tenantID, entry.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if updated.FeedbackCorrect == nil || *updated.FeedbackCorrect {
		t.Fatal("feedback_correct was not stored as false")
	}
	if updated.FeedbackNotes == nil || *updated.FeedbackNotes != "integration correction" {
		t.Fatalf("feedback_notes = %v, want integration correction", updated.FeedbackNotes)
	}
}
