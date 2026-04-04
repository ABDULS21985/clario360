package analyzer

import (
	"reflect"
	"testing"

	"github.com/clario360/platform/internal/lex/model"
)

func TestRecommend_UnlimitedIndemnification(t *testing.T) {
	engine := NewRecommendationEngine("Saudi Arabia")
	got := engine.Recommend(model.ClauseTypeIndemnification, model.RiskLevelHigh, []string{"unlimited", "uncapped"}, "The supplier has unlimited liability.")
	want := []string{"Negotiate a cap on indemnification liability."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Recommend() = %v, want %v", got, want)
	}
}

func TestRecommend_NoTerminationCure(t *testing.T) {
	engine := NewRecommendationEngine("Saudi Arabia")
	got := engine.Recommend(model.ClauseTypeTermination, model.RiskLevelHigh, []string{"no cure period", "immediate"}, "Termination is immediate and no cure period applies.")
	want := []string{"Request a cure period before termination takes effect."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Recommend() = %v, want %v", got, want)
	}
}

func TestRecommend_ForeignJurisdiction(t *testing.T) {
	engine := NewRecommendationEngine("Saudi Arabia")
	got := engine.Recommend(model.ClauseTypeGoverningLaw, model.RiskLevelMedium, []string{"foreign law"}, "This agreement is governed by the laws of New York.")
	want := []string{"Negotiate governing law to local jurisdiction."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Recommend() = %v, want %v", got, want)
	}
}

func TestRecommend_NoAuditRights(t *testing.T) {
	engine := NewRecommendationEngine("Saudi Arabia")
	got := engine.Recommend(model.ClauseTypeAuditRights, model.RiskLevelHigh, []string{"no audit right"}, "There is no audit right.")
	want := []string{"Include audit rights clause per vendor management policy."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Recommend() = %v, want %v", got, want)
	}
}
