package security

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/rs/zerolog"
)

// SSRFValidator prevents Server-Side Request Forgery attacks.
type SSRFValidator struct {
	allowedHosts []string
	blockPrivate bool
	maxRedirects int
	metrics      *Metrics
	logger       zerolog.Logger
}

// NewSSRFValidator creates a new SSRF validator.
func NewSSRFValidator(allowedHosts []string, blockPrivate bool, metrics *Metrics, logger zerolog.Logger) *SSRFValidator {
	return &SSRFValidator{
		allowedHosts: allowedHosts,
		blockPrivate: blockPrivate,
		maxRedirects: 3,
		metrics:      metrics,
		logger:       logger.With().Str("component", "ssrf").Logger(),
	}
}

// ValidateURL checks if a URL is safe to request (no SSRF).
func (v *SSRFValidator) ValidateURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: invalid URL", ErrSSRFBlocked)
	}

	// Only allow HTTP and HTTPS
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%w: scheme %q not allowed", ErrSSRFBlocked, parsed.Scheme)
	}

	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("%w: empty hostname", ErrSSRFBlocked)
	}

	// Check against allowlist if configured
	if len(v.allowedHosts) > 0 {
		if !v.isAllowedHost(host) {
			if v.metrics != nil {
				v.metrics.SSRFBlocked.Inc()
			}
			return fmt.Errorf("%w: host %q not in allowlist", ErrSSRFBlocked, host)
		}
	}

	// Block private/internal IPs
	if v.blockPrivate {
		if err := v.validateNotPrivate(host); err != nil {
			if v.metrics != nil {
				v.metrics.SSRFBlocked.Inc()
			}
			return err
		}
	}

	// Block metadata endpoints
	if isCloudMetadataEndpoint(host) {
		if v.metrics != nil {
			v.metrics.SSRFBlocked.Inc()
		}
		return fmt.Errorf("%w: cloud metadata endpoint blocked", ErrSSRFBlocked)
	}

	return nil
}

// ValidateIP checks if an IP address is safe to connect to.
func (v *SSRFValidator) ValidateIP(ip net.IP) error {
	if ip == nil {
		return fmt.Errorf("%w: nil IP address", ErrSSRFBlocked)
	}

	if v.blockPrivate && isPrivateIP(ip) {
		if v.metrics != nil {
			v.metrics.SSRFBlocked.Inc()
		}
		return fmt.Errorf("%w: private IP blocked", ErrSSRFBlocked)
	}

	return nil
}

// ValidateRedirect checks if a redirect target is safe (prevents redirect-based SSRF).
func (v *SSRFValidator) ValidateRedirect(targetURL string, redirectCount int) error {
	if redirectCount > v.maxRedirects {
		return ErrSSRFRedirectChain
	}
	return v.ValidateURL(targetURL)
}

// isAllowedHost checks if the host is in the allowlist.
func (v *SSRFValidator) isAllowedHost(host string) bool {
	for _, allowed := range v.allowedHosts {
		if host == allowed {
			return true
		}
		// Support wildcard subdomains: *.example.com
		if strings.HasPrefix(allowed, "*.") {
			domain := allowed[2:]
			if strings.HasSuffix(host, "."+domain) || host == domain {
				return true
			}
		}
	}
	return false
}

// validateNotPrivate checks if the host resolves to a private IP.
func (v *SSRFValidator) validateNotPrivate(host string) error {
	// Check if it's a direct IP
	ip := net.ParseIP(host)
	if ip != nil {
		if isPrivateIP(ip) {
			return fmt.Errorf("%w: private IP address", ErrSSRFBlocked)
		}
		return nil
	}

	// Resolve hostname
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("%w: DNS resolution failed", ErrSSRFBlocked)
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			v.logger.Warn().
				Str("host", host).
				Str("resolved_ip", ip.String()).
				Msg("SSRF: hostname resolved to private IP (possible DNS rebinding)")
			return fmt.Errorf("%w: hostname resolves to private IP", ErrSSRFDNSRebinding)
		}
	}

	return nil
}

// isPrivateIP checks if an IP is private, loopback, or link-local.
func isPrivateIP(ip net.IP) bool {
	privateRanges := []struct {
		network *net.IPNet
	}{
		{mustParseCIDR("10.0.0.0/8")},
		{mustParseCIDR("172.16.0.0/12")},
		{mustParseCIDR("192.168.0.0/16")},
		{mustParseCIDR("127.0.0.0/8")},
		{mustParseCIDR("169.254.0.0/16")},   // Link-local
		{mustParseCIDR("::1/128")},           // IPv6 loopback
		{mustParseCIDR("fc00::/7")},          // IPv6 ULA
		{mustParseCIDR("fe80::/10")},         // IPv6 link-local
		{mustParseCIDR("0.0.0.0/8")},         // Current network
		{mustParseCIDR("100.64.0.0/10")},     // Shared address space
		{mustParseCIDR("192.0.0.0/24")},      // IETF protocol assignments
		{mustParseCIDR("198.18.0.0/15")},     // Benchmarking
		{mustParseCIDR("198.51.100.0/24")},   // Documentation
		{mustParseCIDR("203.0.113.0/24")},    // Documentation
		{mustParseCIDR("240.0.0.0/4")},       // Reserved
	}

	for _, r := range privateRanges {
		if r.network.Contains(ip) {
			return true
		}
	}

	return false
}

// isCloudMetadataEndpoint checks for known cloud provider metadata endpoints.
func isCloudMetadataEndpoint(host string) bool {
	metadataHosts := []string{
		"169.254.169.254",        // AWS, GCP, Azure
		"metadata.google.internal", // GCP
		"metadata.goog",          // GCP
		"100.100.100.200",        // Alibaba Cloud
		"169.254.170.2",          // AWS ECS
	}
	for _, mh := range metadataHosts {
		if host == mh {
			return true
		}
	}
	return false
}

// mustParseCIDR parses a CIDR and panics on failure (used for static ranges).
func mustParseCIDR(s string) *net.IPNet {
	_, network, err := net.ParseCIDR(s)
	if err != nil {
		panic("invalid CIDR: " + s)
	}
	return network
}
