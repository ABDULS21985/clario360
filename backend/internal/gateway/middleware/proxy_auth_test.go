package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/config"
)

func newTestRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	return key
}

func encodePrivateKeyPEM(key *rsa.PrivateKey) string {
	b := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: b}
	return string(pem.EncodeToMemory(block))
}

func encodePublicKeyPEM(key *rsa.PublicKey) string {
	b, _ := x509.MarshalPKIXPublicKey(key)
	block := &pem.Block{Type: "PUBLIC KEY", Bytes: b}
	return string(pem.EncodeToMemory(block))
}

func newJWTManager(t *testing.T, key *rsa.PrivateKey, ttl time.Duration) *auth.JWTManager {
	t.Helper()
	mgr, err := auth.NewJWTManager(config.AuthConfig{
		RSAPrivateKeyPEM: encodePrivateKeyPEM(key),
		RSAPublicKeyPEM:  encodePublicKeyPEM(&key.PublicKey),
		JWTIssuer:        "clario360-iam",
		AccessTokenTTL:   ttl,
		RefreshTokenTTL:  7 * 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewJWTManager: %v", err)
	}
	return mgr
}

func makeValidToken(t *testing.T, mgr *auth.JWTManager) string {
	t.Helper()
	pair, err := mgr.GenerateTokenPair("user-1", "tenant-1", "user@example.com", []string{"viewer"})
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}
	return pair.AccessToken
}

// TestProxyAuth_ValidToken — valid RS256 token populates context.
func TestProxyAuth_ValidToken(t *testing.T) {
	key := newTestRSAKey(t)
	mgr := newJWTManager(t, key, 15*time.Minute)
	token := makeValidToken(t, mgr)

	called := false
	handler := ProxyAuth(mgr, nil, zerolog.Nop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		user := auth.UserFromContext(r.Context())
		if user == nil {
			t.Error("expected user in context after valid JWT")
			return
		}
		if user.TenantID != "tenant-1" {
			t.Errorf("expected tenant-1, got %s", user.TenantID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if !called {
		t.Error("expected handler to be called")
	}
}

// TestProxyAuth_MissingToken — no Authorization header returns 401 UNAUTHORIZED.
func TestProxyAuth_MissingToken(t *testing.T) {
	key := newTestRSAKey(t)
	mgr := newJWTManager(t, key, 15*time.Minute)

	handler := ProxyAuth(mgr, nil, zerolog.Nop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called with missing token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
	assertErrorCode(t, rr.Body.Bytes(), "UNAUTHORIZED")
}

// TestProxyAuth_ExpiredToken — expired JWT returns TOKEN_EXPIRED so clients know to refresh.
func TestProxyAuth_ExpiredToken(t *testing.T) {
	key := newTestRSAKey(t)
	// Use a negative TTL so the token is already expired.
	mgr := newJWTManager(t, key, -1*time.Second)
	token := makeValidToken(t, mgr)

	handler := ProxyAuth(mgr, nil, zerolog.Nop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called with expired token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
	assertErrorCode(t, rr.Body.Bytes(), "TOKEN_EXPIRED")
}

// TestProxyAuth_WrongAlgorithm — HS256 token is rejected (algorithm confusion attack blocked).
func TestProxyAuth_WrongAlgorithm(t *testing.T) {
	key := newTestRSAKey(t)
	mgr := newJWTManager(t, key, 15*time.Minute)

	// Forge an HS256 token — this is an algorithm confusion attack.
	hs256Token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": "attacker",
		"tid": "evil-tenant",
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte("secret"))

	handler := ProxyAuth(mgr, nil, zerolog.Nop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("HS256 token must be rejected")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+hs256Token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for HS256 token, got %d", rr.Code)
	}
}

// TestProxyAuth_TamperedToken — modified payload causes signature failure → 401.
func TestProxyAuth_TamperedToken(t *testing.T) {
	key := newTestRSAKey(t)
	mgr := newJWTManager(t, key, 15*time.Minute)
	token := makeValidToken(t, mgr)

	// Flip a character in the payload segment.
	parts := splitOnDot(token)
	if len(parts) == 3 {
		payload := []byte(parts[1])
		if len(payload) > 4 {
			if payload[4] == 'A' {
				payload[4] = 'B'
			} else {
				payload[4] = 'A'
			}
		}
		token = parts[0] + "." + string(payload) + "." + parts[2]
	}

	handler := ProxyAuth(mgr, nil, zerolog.Nop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("tampered token must be rejected")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for tampered token, got %d", rr.Code)
	}
}

// TestProxyAuth_APIKeyForwarding — X-API-Key passes through without JWT validation.
func TestProxyAuth_APIKeyForwarding(t *testing.T) {
	key := newTestRSAKey(t)
	mgr := newJWTManager(t, key, 15*time.Minute)

	called := false
	handler := ProxyAuth(mgr, nil, zerolog.Nop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Header.Get("X-API-Key") == "" {
			t.Error("X-API-Key must be forwarded to backend")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("X-API-Key", "test-api-key-12345")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for API key request, got %d", rr.Code)
	}
	if !called {
		t.Error("expected handler to be called for API key request")
	}
}

// TestProxyAuth_InvalidBearerFormat — malformed Authorization header returns 401.
func TestProxyAuth_InvalidBearerFormat(t *testing.T) {
	key := newTestRSAKey(t)
	mgr := newJWTManager(t, key, 15*time.Minute)

	handler := ProxyAuth(mgr, nil, zerolog.Nop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be called with invalid format")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "NotBearer sometoken")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid format, got %d", rr.Code)
	}
}

func assertErrorCode(t *testing.T, body []byte, wantCode string) {
	t.Helper()
	var resp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode error response: %v (body: %s)", err, body)
	}
	if resp.Error.Code != wantCode {
		t.Errorf("expected error code %q, got %q", wantCode, resp.Error.Code)
	}
}

func splitOnDot(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}
