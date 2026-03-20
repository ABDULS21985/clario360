package expression

import "testing"

func TestCompileAndEvaluate(t *testing.T) {
	t.Run("arithmetic", func(t *testing.T) {
		expr, err := Compile("1 + 2 * 3")
		if err != nil {
			t.Fatalf("Compile() error = %v", err)
		}
		got, err := expr.Evaluate(nil)
		if err != nil {
			t.Fatalf("Evaluate() error = %v", err)
		}
		if got != 7.0 {
			t.Fatalf("Evaluate() = %#v, want 7", got)
		}
	})

	t.Run("comparison", func(t *testing.T) {
		expr, _ := Compile("5 > 3")
		got, err := expr.Evaluate(nil)
		if err != nil || got != true {
			t.Fatalf("Evaluate() = %#v, err = %v, want true", got, err)
		}
	})

	t.Run("string_concat", func(t *testing.T) {
		expr, _ := Compile("'hello' + ' ' + 'world'")
		got, err := expr.Evaluate(nil)
		if err != nil || got != "hello world" {
			t.Fatalf("Evaluate() = %#v, err = %v", got, err)
		}
	})

	t.Run("variable_access", func(t *testing.T) {
		expr, _ := Compile("amount * 1.1")
		got, err := expr.Evaluate(map[string]any{"amount": 100.0})
		if err != nil || got != 110.00000000000001 {
			t.Fatalf("Evaluate() = %#v, err = %v", got, err)
		}
	})

	t.Run("function_call", func(t *testing.T) {
		expr, _ := Compile("UPPER('hello')")
		got, err := expr.Evaluate(nil)
		if err != nil || got != "HELLO" {
			t.Fatalf("Evaluate() = %#v, err = %v", got, err)
		}
	})

	t.Run("nested_function", func(t *testing.T) {
		expr, _ := Compile("TRIM(UPPER(' hello '))")
		got, err := expr.Evaluate(nil)
		if err != nil || got != "HELLO" {
			t.Fatalf("Evaluate() = %#v, err = %v", got, err)
		}
	})

	t.Run("null_propagation", func(t *testing.T) {
		expr, _ := Compile("null + 5")
		got, err := expr.Evaluate(nil)
		if err != nil {
			t.Fatalf("Evaluate() error = %v", err)
		}
		if got != nil {
			t.Fatalf("Evaluate() = %#v, want nil", got)
		}
	})

	t.Run("boolean_logic", func(t *testing.T) {
		expr, _ := Compile("true AND false")
		got, err := expr.Evaluate(nil)
		if err != nil || got != false {
			t.Fatalf("Evaluate() = %#v, err = %v", got, err)
		}

		expr, _ = Compile("true OR false")
		got, err = expr.Evaluate(nil)
		if err != nil || got != true {
			t.Fatalf("Evaluate() = %#v, err = %v", got, err)
		}
	})

	t.Run("invalid_syntax", func(t *testing.T) {
		if _, err := Compile("1 + + 2"); err == nil {
			t.Fatal("Compile() error = nil, want non-nil")
		}
	})

	t.Run("security_no_sql_injection", func(t *testing.T) {
		if _, err := Compile("status == 'active'; DROP TABLE users"); err == nil {
			t.Fatal("Compile() error = nil, want non-nil")
		}
	})
}
