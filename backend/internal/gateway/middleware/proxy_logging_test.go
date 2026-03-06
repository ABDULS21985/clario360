package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

// TestLogging_RedactsTokenParam — query param "token" value is replaced with [REDACTED].
func TestLogging_RedactsTokenParam(t *testing.T) {
	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf)

	handler := ProxyLogging(logger, "test-service")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/callback?token=supersecretjwt123&foo=bar", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	logOutput := logBuf.String()
	if strings.Contains(logOutput, "supersecretjwt123") {
		t.Error("token value must be redacted from logs")
	}
	if !strings.Contains(logOutput, "[REDACTED]") {
		t.Error("expected [REDACTED] in log output for token param")
	}
}

// TestLogging_RedactsPasswordParam — query param "password" is redacted.
func TestLogging_RedactsPasswordParam(t *testing.T) {
	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf)

	handler := ProxyLogging(logger, "test-service")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth?password=mypassword&user=alice", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	logOutput := logBuf.String()
	if strings.Contains(logOutput, "mypassword") {
		t.Error("password value must be redacted from logs")
	}
	if !strings.Contains(logOutput, "[REDACTED]") {
		t.Error("expected [REDACTED] in log output for password param")
	}
}

// TestLogging_DoesNotLogAuthHeader — Authorization header value never appears in logs.
func TestLogging_DoesNotLogAuthHeader(t *testing.T) {
	var logBuf bytes.Buffer
	logger := zerolog.New(&logBuf)

	handler := ProxyLogging(logger, "test-service")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.SENSITIVE.SIGNATURE")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	logOutput := logBuf.String()
	if strings.Contains(logOutput, "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9") {
		t.Error("Authorization header value (JWT) must never appear in logs")
	}
	if strings.Contains(logOutput, "SENSITIVE") {
		t.Error("JWT payload must never appear in logs")
	}
}

// TestLogging_StatusCodeLevels — 5xx uses error level, 4xx uses info, 2xx uses debug.
func TestLogging_StatusCodeLevels(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantLevel  string
	}{
		{"2xx uses debug", http.StatusOK, `"level":"debug"`},
		{"4xx uses info", http.StatusNotFound, `"level":"info"`},
		{"5xx uses error", http.StatusInternalServerError, `"level":"error"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			logger := zerolog.New(&logBuf)

			handler := ProxyLogging(logger, "test-service")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))

			req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			logOutput := logBuf.String()
			if !strings.Contains(logOutput, tt.wantLevel) {
				t.Errorf("expected log level %q in output, got: %s", tt.wantLevel, logOutput)
			}
		})
	}
}
