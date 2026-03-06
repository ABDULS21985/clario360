package proxy

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// ReverseProxy wraps httputil.ReverseProxy with circuit breaker support.
type ReverseProxy struct {
	serviceName string
	target      *url.URL
	proxy       *httputil.ReverseProxy
	breaker     *CircuitBreaker
	logger      zerolog.Logger
}

// NewReverseProxy creates a reverse proxy for a backend service.
// The timeout parameter controls the ResponseHeaderTimeout for upstream requests.
func NewReverseProxy(serviceName string, target *url.URL, timeout time.Duration, breaker *CircuitBreaker, logger zerolog.Logger) *ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)

	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Custom transport with per-service timeout
	proxy.Transport = &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: timeout,
	}

	rp := &ReverseProxy{
		serviceName: serviceName,
		target:      target,
		proxy:       proxy,
		breaker:     breaker,
		logger:      logger,
	}

	// Custom director to rewrite request URL
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
	}

	// Error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		rp.breaker.RecordFailure()
		rp.logger.Error().
			Err(err).
			Str("service", serviceName).
			Str("path", r.URL.Path).
			Msg("proxy error")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  502,
			"code":    "BAD_GATEWAY",
			"message": fmt.Sprintf("upstream service %s is unavailable", serviceName),
		})
	}

	// Modify response to record success/failure
	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.StatusCode >= 500 {
			rp.breaker.RecordFailure()
		} else {
			rp.breaker.RecordSuccess()
		}
		return nil
	}

	return rp
}

// ServeHTTP handles the proxied request with circuit breaker check.
func (rp *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !rp.breaker.Allow() {
		rp.logger.Warn().
			Str("service", rp.serviceName).
			Str("state", rp.breaker.State().String()).
			Msg("circuit breaker open")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  503,
			"code":    "SERVICE_UNAVAILABLE",
			"message": fmt.Sprintf("service %s is temporarily unavailable", rp.serviceName),
			"service": rp.serviceName,
		})
		return
	}

	rp.proxy.ServeHTTP(w, r)
}

// ServiceName returns the name of the backend service.
func (rp *ReverseProxy) ServiceName() string {
	return rp.serviceName
}

// CircuitState returns the circuit breaker state.
func (rp *ReverseProxy) CircuitState() CircuitState {
	return rp.breaker.State()
}

// NewWebSocketProxy creates a reverse proxy suitable for WebSocket connections.
func NewWebSocketProxy(target *url.URL, logger zerolog.Logger) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host

		// Ensure WebSocket upgrade headers are forwarded
		if strings.EqualFold(req.Header.Get("Upgrade"), "websocket") {
			req.Header.Set("Connection", "Upgrade")
		}
	}

	// Use flush-friendly transport for WebSocket
	proxy.Transport = &http.Transport{
		DialContext:         (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		MaxIdleConns:        50,
		IdleConnTimeout:     120 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Error().Err(err).Str("path", r.URL.Path).Msg("websocket proxy error")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  502,
			"code":    "BAD_GATEWAY",
			"message": "websocket upstream unavailable",
		})
	}

	return proxy
}
