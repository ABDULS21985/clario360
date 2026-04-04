package security_test

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	security "github.com/clario360/platform/internal/security"
)

// newTestSSRFValidator creates an SSRFValidator with blockPrivate enabled and
// an optional allowedHosts list.
func newTestSSRFValidator(allowedHosts []string) *security.SSRFValidator {
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	logger := zerolog.Nop()
	return security.NewSSRFValidator(allowedHosts, true, metrics, logger)
}

func TestSSRF_PrivateIPBlocked_10(t *testing.T) {
	v := newTestSSRFValidator(nil)
	err := v.ValidateURL("http://10.0.0.1/admin")
	if err == nil {
		t.Fatal("expected 10.0.0.1 to be blocked, got nil")
	}
	if !errors.Is(err, security.ErrSSRFBlocked) {
		t.Fatalf("expected ErrSSRFBlocked, got: %v", err)
	}
}

func TestSSRF_PrivateIPBlocked_172(t *testing.T) {
	v := newTestSSRFValidator(nil)
	err := v.ValidateURL("http://172.16.0.1/internal")
	if err == nil {
		t.Fatal("expected 172.16.0.1 to be blocked, got nil")
	}
	if !errors.Is(err, security.ErrSSRFBlocked) {
		t.Fatalf("expected ErrSSRFBlocked, got: %v", err)
	}
}

func TestSSRF_PrivateIPBlocked_192(t *testing.T) {
	v := newTestSSRFValidator(nil)
	err := v.ValidateURL("http://192.168.1.1/router")
	if err == nil {
		t.Fatal("expected 192.168.1.1 to be blocked, got nil")
	}
	if !errors.Is(err, security.ErrSSRFBlocked) {
		t.Fatalf("expected ErrSSRFBlocked, got: %v", err)
	}
}

func TestSSRF_PrivateIPBlocked_Loopback(t *testing.T) {
	v := newTestSSRFValidator(nil)
	err := v.ValidateURL("http://127.0.0.1:8080/secret")
	if err == nil {
		t.Fatal("expected 127.0.0.1 to be blocked, got nil")
	}
	if !errors.Is(err, security.ErrSSRFBlocked) {
		t.Fatalf("expected ErrSSRFBlocked, got: %v", err)
	}
}

func TestSSRF_PrivateIPBlocked_IPv6Loopback(t *testing.T) {
	v := newTestSSRFValidator(nil)
	err := v.ValidateURL("http://[::1]/admin")
	if err == nil {
		t.Fatal("expected ::1 to be blocked, got nil")
	}
	if !errors.Is(err, security.ErrSSRFBlocked) {
		t.Fatalf("expected ErrSSRFBlocked, got: %v", err)
	}
}

func TestSSRF_CloudMetadataBlocked(t *testing.T) {
	v := newTestSSRFValidator(nil)
	err := v.ValidateURL("http://169.254.169.254/latest/meta-data/")
	if err == nil {
		t.Fatal("expected cloud metadata endpoint to be blocked, got nil")
	}
	if !errors.Is(err, security.ErrSSRFBlocked) {
		t.Fatalf("expected ErrSSRFBlocked, got: %v", err)
	}
}

func TestSSRF_AllowedHostPasses(t *testing.T) {
	// Use blockPrivate=false to avoid DNS resolution failures in test environments.
	// The allowlist check itself is what we are testing here.
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	logger := zerolog.Nop()
	v := security.NewSSRFValidator([]string{"api.example.com", "cdn.example.com"}, false, metrics, logger)

	err := v.ValidateURL("https://api.example.com/v1/data")
	if err != nil {
		t.Fatalf("expected allowed host to pass, got: %v", err)
	}
}

func TestSSRF_AllowedHostWithWildcard(t *testing.T) {
	// Use blockPrivate=false since we are testing the allowlist wildcard logic,
	// not DNS resolution.
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	logger := zerolog.Nop()
	v := security.NewSSRFValidator([]string{"*.example.com"}, false, metrics, logger)

	err := v.ValidateURL("https://sub.example.com/api")
	if err != nil {
		t.Fatalf("expected wildcard subdomain match to pass, got: %v", err)
	}
}

func TestSSRF_NonAllowedHostBlocked(t *testing.T) {
	v := newTestSSRFValidator([]string{"api.example.com"})

	err := v.ValidateURL("https://evil.attacker.com/steal")
	if err == nil {
		t.Fatal("expected non-allowed host to be blocked, got nil")
	}
	if !errors.Is(err, security.ErrSSRFBlocked) {
		t.Fatalf("expected ErrSSRFBlocked, got: %v", err)
	}
}

func TestSSRF_NonHTTPSchemeBlocked_FTP(t *testing.T) {
	v := newTestSSRFValidator(nil)
	err := v.ValidateURL("ftp://internal-server/files")
	if err == nil {
		t.Fatal("expected ftp:// scheme to be blocked, got nil")
	}
	if !errors.Is(err, security.ErrSSRFBlocked) {
		t.Fatalf("expected ErrSSRFBlocked, got: %v", err)
	}
}

func TestSSRF_NonHTTPSchemeBlocked_File(t *testing.T) {
	v := newTestSSRFValidator(nil)
	err := v.ValidateURL("file:///etc/passwd")
	if err == nil {
		t.Fatal("expected file:// scheme to be blocked, got nil")
	}
	if !errors.Is(err, security.ErrSSRFBlocked) {
		t.Fatalf("expected ErrSSRFBlocked, got: %v", err)
	}
}

func TestSSRF_EmptyHostnameRejected(t *testing.T) {
	v := newTestSSRFValidator(nil)
	err := v.ValidateURL("http:///path")
	if err == nil {
		t.Fatal("expected empty hostname to be rejected, got nil")
	}
	if !errors.Is(err, security.ErrSSRFBlocked) {
		t.Fatalf("expected ErrSSRFBlocked, got: %v", err)
	}
}

func TestSSRF_RedirectChainLimit(t *testing.T) {
	v := newTestSSRFValidator(nil)

	// Within redirect limit should pass (for an external URL)
	err := v.ValidateRedirect("https://example.com/page2", 1)
	if err != nil {
		t.Fatalf("expected redirect within limit to pass, got: %v", err)
	}

	// Exceeding redirect limit should fail
	err = v.ValidateRedirect("https://example.com/page99", 10)
	if err == nil {
		t.Fatal("expected excessive redirect chain to be blocked, got nil")
	}
	if !errors.Is(err, security.ErrSSRFRedirectChain) {
		t.Fatalf("expected ErrSSRFRedirectChain, got: %v", err)
	}
}

func TestSSRF_DNSRebinding(t *testing.T) {
	// Use localhost as a hostname that resolves to a private IP.
	// This tests the DNS resolution path of validateNotPrivate.
	v := newTestSSRFValidator(nil)
	err := v.ValidateURL("http://localhost/admin")
	if err == nil {
		t.Fatal("expected localhost (resolves to 127.0.0.1) to be blocked, got nil")
	}
	// Could be either ErrSSRFBlocked or ErrSSRFDNSRebinding depending on resolution
	if !errors.Is(err, security.ErrSSRFBlocked) && !errors.Is(err, security.ErrSSRFDNSRebinding) {
		t.Fatalf("expected ErrSSRFBlocked or ErrSSRFDNSRebinding, got: %v", err)
	}
}

func TestSSRF_ValidExternalURL_NoAllowlist(t *testing.T) {
	// When no allowlist is configured, any external URL should pass
	v := newTestSSRFValidator(nil)

	err := v.ValidateURL("https://www.google.com/search?q=test")
	if err != nil {
		t.Fatalf("expected legitimate external URL to pass, got: %v", err)
	}
}

func TestSSRF_HTTPSAllowed(t *testing.T) {
	v := newTestSSRFValidator(nil)

	err := v.ValidateURL("https://api.github.com/repos")
	if err != nil {
		t.Fatalf("expected HTTPS URL to pass, got: %v", err)
	}
}
