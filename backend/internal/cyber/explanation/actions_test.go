package explanation

import (
	"strings"
	"testing"

	"github.com/clario360/platform/internal/cyber/model"
)

func TestGenerateRecommendedActionsCriticalRansomware(t *testing.T) {
	alert := &model.Alert{Title: "Ransomware File Activity", Severity: model.SeverityCritical}
	actions := GenerateRecommendedActions(alert, nil)
	joined := strings.Join(actions, " ")
	if !strings.Contains(strings.ToLower(joined), "isolate") {
		t.Fatal("expected critical actions to include isolation guidance")
	}
	if !strings.Contains(strings.ToLower(joined), "backup") {
		t.Fatal("expected ransomware actions to include backup guidance")
	}
}
