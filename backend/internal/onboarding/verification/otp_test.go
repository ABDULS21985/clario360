package verification

import (
	"regexp"
	"testing"
)

func TestOTPLength(t *testing.T) {
	otp, err := GenerateOTP(6)
	if err != nil {
		t.Fatalf("GenerateOTP returned error: %v", err)
	}
	if len(otp) != 6 {
		t.Fatalf("expected otp length 6, got %d", len(otp))
	}
}

func TestOTPNumeric(t *testing.T) {
	otp, err := GenerateOTP(6)
	if err != nil {
		t.Fatalf("GenerateOTP returned error: %v", err)
	}
	if !regexp.MustCompile(`^\d{6}$`).MatchString(otp) {
		t.Fatalf("expected numeric otp, got %q", otp)
	}
}

func TestOTPUniqueness(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		otp, err := GenerateOTP(6)
		if err != nil {
			t.Fatalf("GenerateOTP returned error: %v", err)
		}
		if _, exists := seen[otp]; exists {
			t.Fatalf("duplicate otp generated: %q", otp)
		}
		seen[otp] = struct{}{}
	}
}

func TestOTPHashAndVerify(t *testing.T) {
	hash, err := HashOTP("123456")
	if err != nil {
		t.Fatalf("HashOTP returned error: %v", err)
	}
	if !VerifyOTP(hash, "123456") {
		t.Fatal("expected VerifyOTP to succeed")
	}
}

func TestOTPWrongCode(t *testing.T) {
	hash, err := HashOTP("123456")
	if err != nil {
		t.Fatalf("HashOTP returned error: %v", err)
	}
	if VerifyOTP(hash, "654321") {
		t.Fatal("expected VerifyOTP to fail for wrong code")
	}
}
