package expression

import (
	"strings"
	"testing"
)

func TestEvaluator_SimpleEquality(t *testing.T) {
	eval := NewEvaluator()

	tests := []struct {
		name       string
		expression string
		data       map[string]interface{}
		want       bool
		wantErr    bool
	}{
		{
			name:       "integer equality - true",
			expression: "a == 1",
			data:       map[string]interface{}{"a": int64(1)},
			want:       true,
		},
		{
			name:       "integer equality - false",
			expression: "a == 2",
			data:       map[string]interface{}{"a": int64(1)},
			want:       false,
		},
		{
			name:       "string equality - true",
			expression: "a == 'hello'",
			data:       map[string]interface{}{"a": "hello"},
			want:       true,
		},
		{
			name:       "string equality - false",
			expression: "a == 'world'",
			data:       map[string]interface{}{"a": "hello"},
			want:       false,
		},
		{
			name:       "boolean equality - true",
			expression: "a == true",
			data:       map[string]interface{}{"a": true},
			want:       true,
		},
		{
			name:       "boolean equality - false",
			expression: "a == false",
			data:       map[string]interface{}{"a": true},
			want:       false,
		},
		{
			name:       "null equality - true",
			expression: "a == null",
			data:       map[string]interface{}{"a": nil},
			want:       true,
		},
		{
			name:       "null equality - false",
			expression: "a == null",
			data:       map[string]interface{}{"a": "something"},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := eval.Evaluate(tt.expression, tt.data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.expression, got, tt.want)
			}
		})
	}
}

func TestEvaluator_Inequality(t *testing.T) {
	eval := NewEvaluator()

	tests := []struct {
		name       string
		expression string
		data       map[string]interface{}
		want       bool
	}{
		{
			name:       "not equal - true",
			expression: "a != 'x'",
			data:       map[string]interface{}{"a": "y"},
			want:       true,
		},
		{
			name:       "not equal - false",
			expression: "a != 'x'",
			data:       map[string]interface{}{"a": "x"},
			want:       false,
		},
		{
			name:       "not equal numeric",
			expression: "a != 5",
			data:       map[string]interface{}{"a": int64(3)},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := eval.Evaluate(tt.expression, tt.data)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.expression, got, tt.want)
			}
		})
	}
}

func TestEvaluator_Comparison(t *testing.T) {
	eval := NewEvaluator()

	tests := []struct {
		name       string
		expression string
		data       map[string]interface{}
		want       bool
	}{
		{
			name:       "greater than - true",
			expression: "a > 5",
			data:       map[string]interface{}{"a": int64(10)},
			want:       true,
		},
		{
			name:       "greater than - false",
			expression: "a > 5",
			data:       map[string]interface{}{"a": int64(3)},
			want:       false,
		},
		{
			name:       "greater than or equal - true (equal)",
			expression: "a >= 5",
			data:       map[string]interface{}{"a": int64(5)},
			want:       true,
		},
		{
			name:       "greater than or equal - true (greater)",
			expression: "a >= 5",
			data:       map[string]interface{}{"a": int64(7)},
			want:       true,
		},
		{
			name:       "greater than or equal - false",
			expression: "a >= 5",
			data:       map[string]interface{}{"a": int64(4)},
			want:       false,
		},
		{
			name:       "less than - true",
			expression: "a < 10",
			data:       map[string]interface{}{"a": int64(5)},
			want:       true,
		},
		{
			name:       "less than - false",
			expression: "a < 10",
			data:       map[string]interface{}{"a": int64(15)},
			want:       false,
		},
		{
			name:       "less than or equal - true (equal)",
			expression: "a <= 10",
			data:       map[string]interface{}{"a": int64(10)},
			want:       true,
		},
		{
			name:       "less than or equal - true (less)",
			expression: "a <= 10",
			data:       map[string]interface{}{"a": int64(5)},
			want:       true,
		},
		{
			name:       "less than or equal - false",
			expression: "a <= 10",
			data:       map[string]interface{}{"a": int64(15)},
			want:       false,
		},
		{
			name:       "float comparison",
			expression: "a > 3.14",
			data:       map[string]interface{}{"a": float64(3.15)},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := eval.Evaluate(tt.expression, tt.data)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.expression, got, tt.want)
			}
		})
	}
}

func TestEvaluator_BooleanOperators(t *testing.T) {
	eval := NewEvaluator()

	tests := []struct {
		name       string
		expression string
		data       map[string]interface{}
		want       bool
	}{
		{
			name:       "and - both true",
			expression: "a == 1 && b == 2",
			data:       map[string]interface{}{"a": int64(1), "b": int64(2)},
			want:       true,
		},
		{
			name:       "and - left false",
			expression: "a == 1 && b == 2",
			data:       map[string]interface{}{"a": int64(0), "b": int64(2)},
			want:       false,
		},
		{
			name:       "and - right false",
			expression: "a == 1 && b == 2",
			data:       map[string]interface{}{"a": int64(1), "b": int64(0)},
			want:       false,
		},
		{
			name:       "or - both true",
			expression: "a == 1 || b == 2",
			data:       map[string]interface{}{"a": int64(1), "b": int64(2)},
			want:       true,
		},
		{
			name:       "or - left true",
			expression: "a == 1 || b == 2",
			data:       map[string]interface{}{"a": int64(1), "b": int64(0)},
			want:       true,
		},
		{
			name:       "or - right true",
			expression: "a == 1 || b == 2",
			data:       map[string]interface{}{"a": int64(0), "b": int64(2)},
			want:       true,
		},
		{
			name:       "or - both false",
			expression: "a == 1 || b == 2",
			data:       map[string]interface{}{"a": int64(0), "b": int64(0)},
			want:       false,
		},
		{
			name:       "complex and/or",
			expression: "a == 1 && b == 2 || c == 3",
			data:       map[string]interface{}{"a": int64(1), "b": int64(2), "c": int64(0)},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := eval.Evaluate(tt.expression, tt.data)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.expression, got, tt.want)
			}
		})
	}
}

func TestEvaluator_Negation(t *testing.T) {
	eval := NewEvaluator()

	tests := []struct {
		name       string
		expression string
		data       map[string]interface{}
		want       bool
	}{
		{
			name:       "negate true",
			expression: "!a",
			data:       map[string]interface{}{"a": true},
			want:       false,
		},
		{
			name:       "negate false",
			expression: "!a",
			data:       map[string]interface{}{"a": false},
			want:       true,
		},
		{
			name:       "negate truthy string",
			expression: "!a",
			data:       map[string]interface{}{"a": "hello"},
			want:       false,
		},
		{
			name:       "negate empty string",
			expression: "!a",
			data:       map[string]interface{}{"a": ""},
			want:       true,
		},
		{
			name:       "negate null",
			expression: "!a",
			data:       map[string]interface{}{"a": nil},
			want:       true,
		},
		{
			name:       "double negation",
			expression: "!!a",
			data:       map[string]interface{}{"a": true},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := eval.Evaluate(tt.expression, tt.data)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.expression, got, tt.want)
			}
		})
	}
}

func TestEvaluator_InOperator(t *testing.T) {
	eval := NewEvaluator()

	tests := []struct {
		name       string
		expression string
		data       map[string]interface{}
		want       bool
	}{
		{
			name:       "in array - found",
			expression: "a in ['x', 'y', 'z']",
			data:       map[string]interface{}{"a": "y"},
			want:       true,
		},
		{
			name:       "in array - not found",
			expression: "a in ['x', 'y', 'z']",
			data:       map[string]interface{}{"a": "w"},
			want:       false,
		},
		{
			name:       "in array - numeric",
			expression: "a in [1, 2, 3]",
			data:       map[string]interface{}{"a": int64(2)},
			want:       true,
		},
		{
			name:       "in array - numeric not found",
			expression: "a in [1, 2, 3]",
			data:       map[string]interface{}{"a": int64(5)},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := eval.Evaluate(tt.expression, tt.data)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.expression, got, tt.want)
			}
		})
	}
}

func TestEvaluator_NestedPaths(t *testing.T) {
	eval := NewEvaluator()

	data := map[string]interface{}{
		"steps": map[string]interface{}{
			"triage": map[string]interface{}{
				"output": map[string]interface{}{
					"is_valid": true,
					"severity": "critical",
					"score":    int64(95),
				},
			},
			"investigate": map[string]interface{}{
				"output": map[string]interface{}{
					"needs_approval": true,
					"status":         "pending",
				},
			},
		},
		"variables": map[string]interface{}{
			"severity": "high",
			"amount":   int64(15000),
			"currency": "USD",
		},
		"trigger": map[string]interface{}{
			"data": map[string]interface{}{
				"id":   "alert-123",
				"type": "security",
			},
		},
	}

	tests := []struct {
		name       string
		expression string
		want       bool
	}{
		{
			name:       "step output boolean",
			expression: "steps.triage.output.is_valid == true",
			want:       true,
		},
		{
			name:       "step output string",
			expression: "steps.triage.output.severity == 'critical'",
			want:       true,
		},
		{
			name:       "step output number comparison",
			expression: "steps.triage.output.score > 90",
			want:       true,
		},
		{
			name:       "variable equality",
			expression: "variables.severity == 'high'",
			want:       true,
		},
		{
			name:       "variable number comparison",
			expression: "variables.amount > 10000",
			want:       true,
		},
		{
			name:       "trigger data",
			expression: "trigger.data.type == 'security'",
			want:       true,
		},
		{
			name:       "step output status not success",
			expression: "steps.investigate.output.status != 'success'",
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := eval.Evaluate(tt.expression, data)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.expression, got, tt.want)
			}
		})
	}
}

func TestEvaluator_Parentheses(t *testing.T) {
	eval := NewEvaluator()

	tests := []struct {
		name       string
		expression string
		data       map[string]interface{}
		want       bool
	}{
		{
			name:       "parentheses change precedence",
			expression: "(a == 1 || b == 2) && c == 3",
			data:       map[string]interface{}{"a": int64(1), "b": int64(0), "c": int64(3)},
			want:       true,
		},
		{
			name:       "parentheses - inner false",
			expression: "(a == 1 || b == 2) && c == 3",
			data:       map[string]interface{}{"a": int64(0), "b": int64(0), "c": int64(3)},
			want:       false,
		},
		{
			name:       "nested parentheses",
			expression: "((a == 1))",
			data:       map[string]interface{}{"a": int64(1)},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := eval.Evaluate(tt.expression, tt.data)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.expression, got, tt.want)
			}
		})
	}
}

func TestEvaluator_ComplexExpressions(t *testing.T) {
	eval := NewEvaluator()

	data := map[string]interface{}{
		"steps": map[string]interface{}{
			"investigate": map[string]interface{}{
				"output": map[string]interface{}{
					"needs_approval": true,
				},
			},
			"dry_run": map[string]interface{}{
				"output": map[string]interface{}{
					"status": "failed",
				},
			},
		},
		"variables": map[string]interface{}{
			"severity": "critical",
			"amount":   int64(15000),
			"currency": "USD",
		},
	}

	tests := []struct {
		name       string
		expression string
		want       bool
	}{
		{
			name:       "complex: step output and variable in array",
			expression: "steps.investigate.output.needs_approval == true && variables.severity in ['critical', 'high']",
			want:       true,
		},
		{
			name:       "complex: amount and currency check",
			expression: "variables.amount > 10000 && variables.currency == 'USD'",
			want:       true,
		},
		{
			name:       "complex: dry run status not success",
			expression: "steps.dry_run.output.status != 'success'",
			want:       true,
		},
		{
			name:       "complex: combined conditions",
			expression: "steps.investigate.output.needs_approval == true && (variables.severity == 'critical' || variables.amount > 20000)",
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := eval.Evaluate(tt.expression, data)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.expression, got, tt.want)
			}
		})
	}
}

func TestEvaluator_EdgeCases(t *testing.T) {
	eval := NewEvaluator()

	t.Run("empty expression returns error", func(t *testing.T) {
		_, err := eval.Evaluate("", map[string]interface{}{})
		if err == nil {
			t.Fatal("expected error for empty expression")
		}
		if !strings.Contains(err.Error(), "empty expression") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("expression too long returns error", func(t *testing.T) {
		longExpr := strings.Repeat("a", 1001)
		_, err := eval.Evaluate(longExpr, map[string]interface{}{})
		if err == nil {
			t.Fatal("expected error for expression too long")
		}
		if !strings.Contains(err.Error(), "maximum length") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("max depth exceeded", func(t *testing.T) {
		// Create deeply nested parentheses.
		expr := strings.Repeat("(", 12) + "a == 1" + strings.Repeat(")", 12)
		_, err := eval.Evaluate(expr, map[string]interface{}{"a": int64(1)})
		if err == nil {
			t.Fatal("expected error for max depth exceeded")
		}
		if !strings.Contains(err.Error(), "depth") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("missing path segment returns error", func(t *testing.T) {
		_, err := eval.Evaluate("a.b.c == 1", map[string]interface{}{"a": map[string]interface{}{"x": 1}})
		if err == nil {
			t.Fatal("expected error for missing path segment")
		}
	})

	t.Run("comparison with non-numeric values", func(t *testing.T) {
		_, err := eval.Evaluate("a > 'hello'", map[string]interface{}{"a": int64(5)})
		if err == nil {
			t.Fatal("expected error for comparison with non-numeric value")
		}
	})

	t.Run("in operator with non-array RHS", func(t *testing.T) {
		_, err := eval.Evaluate("a in 'hello'", map[string]interface{}{"a": "h"})
		if err == nil {
			t.Fatal("expected error for 'in' with non-array RHS")
		}
	})
}

func TestEvaluator_InvalidExpressions(t *testing.T) {
	eval := NewEvaluator()

	tests := []struct {
		name       string
		expression string
	}{
		{
			name:       "unterminated string",
			expression: "a == 'hello",
		},
		{
			name:       "unexpected token",
			expression: "a == ==",
		},
		{
			name:       "missing right operand",
			expression: "a ==",
		},
		{
			name:       "trailing operator",
			expression: "a == 1 &&",
		},
		{
			name:       "unmatched parenthesis",
			expression: "(a == 1",
		},
		{
			name:       "unexpected character",
			expression: "a @ b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := eval.Evaluate(tt.expression, map[string]interface{}{"a": int64(1), "b": int64(2)})
			if err == nil {
				t.Errorf("expected error for expression %q", tt.expression)
			}
		})
	}
}

func TestEvaluator_BooleanTruthiness(t *testing.T) {
	eval := NewEvaluator()

	tests := []struct {
		name       string
		expression string
		data       map[string]interface{}
		want       bool
	}{
		{
			name:       "truthy path evaluates to true",
			expression: "a",
			data:       map[string]interface{}{"a": true},
			want:       true,
		},
		{
			name:       "falsy path evaluates to false",
			expression: "a",
			data:       map[string]interface{}{"a": false},
			want:       false,
		},
		{
			name:       "truthy non-zero number",
			expression: "a",
			data:       map[string]interface{}{"a": int64(42)},
			want:       true,
		},
		{
			name:       "falsy zero",
			expression: "a",
			data:       map[string]interface{}{"a": int64(0)},
			want:       false,
		},
		{
			name:       "falsy null",
			expression: "a",
			data:       map[string]interface{}{"a": nil},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := eval.Evaluate(tt.expression, tt.data)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.expression, got, tt.want)
			}
		})
	}
}

func TestEvaluator_NumericCoercion(t *testing.T) {
	eval := NewEvaluator()

	tests := []struct {
		name       string
		expression string
		data       map[string]interface{}
		want       bool
	}{
		{
			name:       "int and float equality",
			expression: "a == 5",
			data:       map[string]interface{}{"a": float64(5)},
			want:       true,
		},
		{
			name:       "int and float comparison",
			expression: "a > 4",
			data:       map[string]interface{}{"a": float64(4.5)},
			want:       true,
		},
		{
			name:       "int64 equality",
			expression: "a == 100",
			data:       map[string]interface{}{"a": int64(100)},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := eval.Evaluate(tt.expression, tt.data)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Evaluate(%q) = %v, want %v", tt.expression, got, tt.want)
			}
		})
	}
}

func TestEvaluator_ShortCircuit(t *testing.T) {
	eval := NewEvaluator()

	t.Run("and short circuits on false left", func(t *testing.T) {
		// 'b' doesn't exist, but short-circuit should prevent accessing it.
		got, err := eval.Evaluate("a == false && b == 1", map[string]interface{}{"a": false})
		if err != nil {
			t.Fatalf("Evaluate() error = %v", err)
		}
		if got != false {
			t.Errorf("expected false, got %v", got)
		}
	})

	t.Run("or short circuits on true left", func(t *testing.T) {
		// 'b' doesn't exist, but short-circuit should prevent accessing it.
		got, err := eval.Evaluate("a == true || b == 1", map[string]interface{}{"a": true})
		if err != nil {
			t.Fatalf("Evaluate() error = %v", err)
		}
		if got != true {
			t.Errorf("expected true, got %v", got)
		}
	})
}
