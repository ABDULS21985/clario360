package transforms

import "testing"

func TestApplyDerive(t *testing.T) {
	rows := []map[string]any{
		{"first_name": "John", "last_name": "Doe", "name": "john", "price": 12.5, "quantity": 4.0, "nickname": nil},
	}

	t.Run("concatenation", func(t *testing.T) {
		got, _, err := ApplyDerive(rows, DeriveConfig{Name: "full_name", Expression: "first_name + ' ' + last_name"})
		if err != nil {
			t.Fatalf("ApplyDerive() error = %v", err)
		}
		if got[0]["full_name"] != "John Doe" {
			t.Fatalf("ApplyDerive() full_name = %#v", got[0]["full_name"])
		}
	})

	t.Run("function", func(t *testing.T) {
		got, _, err := ApplyDerive(rows, DeriveConfig{Name: "upper_name", Expression: "UPPER(name)"})
		if err != nil {
			t.Fatalf("ApplyDerive() error = %v", err)
		}
		if got[0]["upper_name"] != "JOHN" {
			t.Fatalf("ApplyDerive() upper_name = %#v", got[0]["upper_name"])
		}
	})

	t.Run("arithmetic", func(t *testing.T) {
		got, _, err := ApplyDerive(rows, DeriveConfig{Name: "total", Expression: "price * quantity"})
		if err != nil {
			t.Fatalf("ApplyDerive() error = %v", err)
		}
		if got[0]["total"] != 50.0 {
			t.Fatalf("ApplyDerive() total = %#v", got[0]["total"])
		}
	})

	t.Run("coalesce", func(t *testing.T) {
		got, _, err := ApplyDerive(rows, DeriveConfig{Name: "preferred_name", Expression: "COALESCE(nickname, first_name)"})
		if err != nil {
			t.Fatalf("ApplyDerive() error = %v", err)
		}
		if got[0]["preferred_name"] != "John" {
			t.Fatalf("ApplyDerive() preferred_name = %#v", got[0]["preferred_name"])
		}
	})
}
