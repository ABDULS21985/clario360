package detection

import "testing"

func TestParseConditionPrecedenceAndEvaluate(t *testing.T) {
	expr, err := ParseCondition("A or B and C")
	if err != nil {
		t.Fatalf("ParseCondition returned error: %v", err)
	}

	if !expr.Evaluate(map[string]bool{"A": false, "B": true, "C": true}) {
		t.Fatal("expected expression to evaluate to true")
	}
	if expr.Evaluate(map[string]bool{"A": false, "B": true, "C": false}) {
		t.Fatal("expected expression to evaluate to false")
	}
}

func TestParseConditionComplex(t *testing.T) {
	expr, err := ParseCondition("(selection_main or selection_alt) and not filter_internal")
	if err != nil {
		t.Fatalf("ParseCondition returned error: %v", err)
	}
	if !expr.Evaluate(map[string]bool{
		"selection_main":  true,
		"selection_alt":   false,
		"filter_internal": false,
	}) {
		t.Fatal("expected complex expression to evaluate to true")
	}
}

func TestParseConditionInvalid(t *testing.T) {
	if _, err := ParseCondition("A and and B"); err == nil {
		t.Fatal("expected parse error for invalid syntax")
	}
	if _, err := ParseCondition(""); err == nil {
		t.Fatal("expected parse error for empty condition")
	}
}
