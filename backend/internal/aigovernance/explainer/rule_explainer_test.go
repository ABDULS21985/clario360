package explainer

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
)

func TestRuleExplainerRendersTemplate(t *testing.T) {
	tmpl := `Matched rules: {{join .matched_rules ", "}}`
	version := &aigovmodel.ModelVersion{
		ID:                  uuid.New(),
		ModelSlug:           "cyber-sigma-evaluator",
		VersionNumber:       2,
		ExplainabilityType:  aigovmodel.ExplainabilityRuleTrace,
		ExplanationTemplate: &tmpl,
	}
	output := &aigovernance.ModelOutput{
		Output:     map[string]any{"decision": "match"},
		Confidence: 0.93,
		Metadata: map[string]any{
			"matched_rules": []string{"Impossible Travel", "Threat Intel Match"},
			"rule_weights": map[string]any{
				"Impossible Travel": 0.7,
				"Threat Intel Match": 0.3,
			},
		},
	}

	explanation, err := NewRuleExplainer().Explain(context.Background(), version, nil, output)
	if err != nil {
		t.Fatalf("Explain() error = %v", err)
	}
	if explanation.HumanReadable != "Matched rules: Impossible Travel, Threat Intel Match" {
		t.Fatalf("unexpected human explanation: %q", explanation.HumanReadable)
	}
}

func TestRuleExplainerExtractsFactors(t *testing.T) {
	version := &aigovmodel.ModelVersion{ModelSlug: "data-pii-classifier", VersionNumber: 1}
	output := &aigovernance.ModelOutput{
		Output:     map[string]any{"contains_pii": true},
		Confidence: 0.91,
		Metadata: map[string]any{
			"matched_rules": []string{"column_name_heuristics", "sample_value_patterns", "schema_policy"},
			"rule_weights": map[string]any{
				"column_name_heuristics": 0.25,
				"sample_value_patterns":  0.55,
				"schema_policy":          0.2,
			},
		},
	}

	explanation, err := NewRuleExplainer().Explain(context.Background(), version, nil, output)
	if err != nil {
		t.Fatalf("Explain() error = %v", err)
	}
	if len(explanation.Factors) != 3 {
		t.Fatalf("len(Factors) = %d, want 3", len(explanation.Factors))
	}
	if explanation.Factors[0].Name != "sample_value_patterns" {
		t.Fatalf("expected highest-weighted factor first, got %q", explanation.Factors[0].Name)
	}
}

func TestRuleExplainerNoTemplate(t *testing.T) {
	version := &aigovmodel.ModelVersion{ModelSlug: "lex-clause-extractor", VersionNumber: 1}
	output := &aigovernance.ModelOutput{
		Output:     map[string]any{"clause_count": 2},
		Confidence: 0.86,
		Metadata: map[string]any{
			"matched_rules": []string{"termination_clause", "renewal_clause"},
		},
	}

	explanation, err := NewRuleExplainer().Explain(context.Background(), version, nil, output)
	if err != nil {
		t.Fatalf("Explain() error = %v", err)
	}
	want := "Decision made because 2 rules matched: termination_clause, renewal_clause."
	if explanation.HumanReadable != want {
		t.Fatalf("HumanReadable = %q, want %q", explanation.HumanReadable, want)
	}
}
