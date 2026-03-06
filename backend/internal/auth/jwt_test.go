package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/clario360/platform/internal/config"
)

func newTestJWTManager(t *testing.T) *JWTManager {
	t.Helper()
	mgr, err := NewJWTManager(config.AuthConfig{
		JWTIssuer:       "test-issuer",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewJWTManager failed: %v", err)
	}
	return mgr
}

func TestNewJWTManager_EphemeralKeys(t *testing.T) {
	mgr := newTestJWTManager(t)
	if mgr.privateKey == nil {
		t.Fatal("expected private key to be generated")
	}
	if mgr.publicKey == nil {
		t.Fatal("expected public key to be generated")
	}
}

func TestGenerateTokenPair(t *testing.T) {
	mgr := newTestJWTManager(t)

	pair, err := mgr.GenerateTokenPair("user-1", "tenant-1", "test@example.com", []string{"viewer"})
	if err != nil {
		t.Fatalf("GenerateTokenPair failed: %v", err)
	}
	if pair.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if pair.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if pair.ExpiresAt.IsZero() {
		t.Error("expected non-zero expiry")
	}
}

func TestValidateAccessToken_Success(t *testing.T) {
	mgr := newTestJWTManager(t)

	pair, err := mgr.GenerateTokenPair("user-1", "tenant-1", "test@example.com", []string{"admin", "viewer"})
	if err != nil {
		t.Fatalf("GenerateTokenPair failed: %v", err)
	}

	claims, err := mgr.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}
	if claims.UserID != "user-1" {
		t.Errorf("expected user ID user-1, got %s", claims.UserID)
	}
	if claims.TenantID != "tenant-1" {
		t.Errorf("expected tenant ID tenant-1, got %s", claims.TenantID)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", claims.Email)
	}
	if len(claims.Roles) != 2 {
		t.Errorf("expected 2 roles, got %d", len(claims.Roles))
	}
	if claims.Issuer != "test-issuer" {
		t.Errorf("expected issuer test-issuer, got %s", claims.Issuer)
	}
}

func TestValidateAccessToken_InvalidToken(t *testing.T) {
	mgr := newTestJWTManager(t)

	_, err := mgr.ValidateAccessToken("not-a-valid-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestValidateAccessToken_WrongKey(t *testing.T) {
	mgr1 := newTestJWTManager(t)
	mgr2 := newTestJWTManager(t) // different ephemeral key pair

	pair, _ := mgr1.GenerateTokenPair("user-1", "tenant-1", "test@example.com", nil)
	_, err := mgr2.ValidateAccessToken(pair.AccessToken)
	if err == nil {
		t.Fatal("expected error when validating with different key")
	}
}

func TestValidateRefreshToken_Success(t *testing.T) {
	mgr := newTestJWTManager(t)

	pair, _ := mgr.GenerateTokenPair("user-1", "tenant-1", "test@example.com", nil)

	userID, err := mgr.ValidateRefreshToken(pair.RefreshToken)
	if err != nil {
		t.Fatalf("ValidateRefreshToken failed: %v", err)
	}
	if userID != "user-1" {
		t.Errorf("expected user-1, got %s", userID)
	}
}

func TestValidateRefreshToken_InvalidToken(t *testing.T) {
	mgr := newTestJWTManager(t)

	_, err := mgr.ValidateRefreshToken("garbage")
	if err == nil {
		t.Fatal("expected error for invalid refresh token")
	}
}

func TestNewJWTManager_WithPEMKeys(t *testing.T) {
	// Generate RSA key pair and encode as PEM
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	})

	pubDER, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		t.Fatalf("marshaling public key: %v", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubDER,
	})

	mgr, err := NewJWTManager(config.AuthConfig{
		RSAPrivateKeyPEM: string(privPEM),
		RSAPublicKeyPEM:  string(pubPEM),
		JWTIssuer:        "test-pem",
		AccessTokenTTL:   15 * time.Minute,
		RefreshTokenTTL:  time.Hour,
	})
	if err != nil {
		t.Fatalf("NewJWTManager with PEM keys failed: %v", err)
	}

	// Verify tokens can be generated and validated
	pair, err := mgr.GenerateTokenPair("user-1", "tenant-1", "pem@test.com", []string{"admin"})
	if err != nil {
		t.Fatalf("GenerateTokenPair failed: %v", err)
	}

	claims, err := mgr.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}
	if claims.Issuer != "test-pem" {
		t.Errorf("expected issuer test-pem, got %s", claims.Issuer)
	}
}

func TestNewJWTManager_InvalidPEM(t *testing.T) {
	_, err := NewJWTManager(config.AuthConfig{
		RSAPrivateKeyPEM: "not-valid-pem",
		RSAPublicKeyPEM:  "not-valid-pem",
		JWTIssuer:        "test",
		AccessTokenTTL:   time.Minute,
		RefreshTokenTTL:  time.Hour,
	})
	if err == nil {
		t.Fatal("expected error for invalid PEM keys")
	}
}

func TestValidateAccessToken_ExpiredToken(t *testing.T) {
	// Create a manager with very short TTL
	mgr, err := NewJWTManager(config.AuthConfig{
		JWTIssuer:       "test",
		AccessTokenTTL:  -1 * time.Second, // already expired
		RefreshTokenTTL: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewJWTManager failed: %v", err)
	}

	pair, _ := mgr.GenerateTokenPair("user-1", "tenant-1", "test@example.com", nil)

	_, err = mgr.ValidateAccessToken(pair.AccessToken)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}
