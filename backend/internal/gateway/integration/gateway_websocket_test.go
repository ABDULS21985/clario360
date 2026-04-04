package integration

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/gateway/proxy"
)

// newTestJWTManager creates a fresh JWTManager with a generated RSA key for WebSocket tests.
func newTestJWTManager(t *testing.T) *auth.JWTManager {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	pubBytes, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
	mgr, err := auth.NewJWTManager(config.AuthConfig{
		RSAPrivateKeyPEM: string(privPEM),
		RSAPublicKeyPEM:  string(pubPEM),
		JWTIssuer:        "clario360-iam",
		AccessTokenTTL:   15 * time.Minute,
		RefreshTokenTTL:  7 * 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewJWTManager: %v", err)
	}
	return mgr
}

// makeWSToken generates a valid access token for WebSocket tests.
func makeWSToken(t *testing.T, mgr *auth.JWTManager) string {
	t.Helper()
	pair, err := mgr.GenerateTokenPair("user-ws-1", "tenant-ws", "ws@example.com", []string{"viewer"}, "")
	if err != nil {
		t.Fatalf("GenerateTokenPair: %v", err)
	}
	return pair.AccessToken
}

// wsUpgrader is used by test backend servers to upgrade HTTP → WebSocket.
var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// TestWebSocket_AuthBeforeUpgrade — unauthenticated requests are rejected with 401
// BEFORE the WebSocket upgrade handshake occurs.
func TestWebSocket_AuthBeforeUpgrade(t *testing.T) {
	jwtMgr := newTestJWTManager(t)

	// A backend that should never be reached.
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("backend must NOT be called for unauthenticated WS request")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)
	wsp := proxy.NewWebSocketProxy(backendURL, jwtMgr, nil, nil, nil, zerolog.Nop())

	// Wrap in a test HTTP server so the gorilla dialer can connect.
	srv := httptest.NewServer(wsp)
	defer srv.Close()

	// Attempt WS connection with no token — should get an HTTP 401 back, not a WS upgrade.
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/notifications/ws"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected dial to fail for unauthenticated request")
	}
	if resp == nil {
		t.Fatal("expected HTTP response with status code")
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}
}

// TestWebSocket_ProxiesBidirectional — a valid JWT allows the WS connection to upgrade
// and messages flow bidirectionally through the proxy (echo test).
func TestWebSocket_ProxiesBidirectional(t *testing.T) {
	jwtMgr := newTestJWTManager(t)
	token := makeWSToken(t, jwtMgr)

	// Echo backend: reads each message and writes it back.
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("backend upgrade error: %v", err)
			return
		}
		defer conn.Close()
		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if err := conn.WriteMessage(msgType, msg); err != nil {
				return
			}
		}
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)
	wsp := proxy.NewWebSocketProxy(backendURL, jwtMgr, nil, nil, nil, zerolog.Nop())

	srv := httptest.NewServer(wsp)
	defer srv.Close()

	// Connect as a client with a valid ?token= query param.
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/notifications/ws?token=" + token
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed (status %v): %v", resp, err)
	}
	defer conn.Close()

	// Send a message and expect it echoed.
	const payload = "hello-gateway"
	if err := conn.WriteMessage(websocket.TextMessage, []byte(payload)); err != nil {
		t.Fatalf("write message: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	msgType, got, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read echoed message: %v", err)
	}
	if msgType != websocket.TextMessage {
		t.Errorf("expected TextMessage, got type %d", msgType)
	}
	if string(got) != payload {
		t.Errorf("expected %q, got %q", payload, string(got))
	}
}

// TestWebSocket_StripsTokenFromQuery — the ?token= parameter is removed before
// the request is forwarded to the backend, but other query params are preserved.
func TestWebSocket_StripsTokenFromQuery(t *testing.T) {
	jwtMgr := newTestJWTManager(t)
	token := makeWSToken(t, jwtMgr)

	var capturedQuery string

	// Backend that records the raw query string it received.
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		conn.Close()
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)
	wsp := proxy.NewWebSocketProxy(backendURL, jwtMgr, nil, nil, nil, zerolog.Nop())

	srv := httptest.NewServer(wsp)
	defer srv.Close()

	// Connect with both ?token= and another safe param.
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") +
		"/api/v1/notifications/ws?token=" + token + "&room=general"

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	conn.Close()

	// Give the backend handler a moment to run.
	time.Sleep(50 * time.Millisecond)

	if strings.Contains(capturedQuery, "token=") {
		t.Errorf("backend must not receive ?token= param, got query: %q", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "room=general") {
		t.Errorf("backend must receive non-token params, got query: %q", capturedQuery)
	}
}
