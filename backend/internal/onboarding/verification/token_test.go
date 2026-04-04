package verification

import (
	"strings"
	"testing"
)

func TestTokenLength(t *testing.T) {
	token, _, err := GenerateInviteToken()
	if err != nil {
		t.Fatalf("GenerateInviteToken returned error: %v", err)
	}
	if len(token) != 43 {
		t.Fatalf("expected token length 43, got %d", len(token))
	}
}

func TestTokenURLSafe(t *testing.T) {
	token, _, err := GenerateInviteToken()
	if err != nil {
		t.Fatalf("GenerateInviteToken returned error: %v", err)
	}
	if strings.ContainsAny(token, "+/=") {
		t.Fatalf("expected URL-safe token, got %q", token)
	}
}

func TestTokenPrefix(t *testing.T) {
	token, prefix, err := GenerateInviteToken()
	if err != nil {
		t.Fatalf("GenerateInviteToken returned error: %v", err)
	}
	if prefix != token[:8] {
		t.Fatalf("expected prefix %q, got %q", token[:8], prefix)
	}
}

func TestTokenHashAndVerify(t *testing.T) {
	token, _, err := GenerateInviteToken()
	if err != nil {
		t.Fatalf("GenerateInviteToken returned error: %v", err)
	}
	hash, err := HashInviteToken(token)
	if err != nil {
		t.Fatalf("HashInviteToken returned error: %v", err)
	}
	if !VerifyInviteToken(hash, token) {
		t.Fatal("expected VerifyInviteToken to succeed")
	}
}
