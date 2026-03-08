package security_test

import (
	"os"
	"testing"

	security "github.com/clario360/platform/internal/security"
)

func TestDefaultConfig_Validate(t *testing.T) {
	cfg := security.DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("DefaultConfig() should validate: %v", err)
	}
}

func TestDevelopmentConfig_Validate(t *testing.T) {
	cfg := security.DevelopmentConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("DevelopmentConfig() should validate: %v", err)
	}
}

func TestConfig_Validate_EmptyEnvironment(t *testing.T) {
	cfg := security.DefaultConfig()
	cfg.Environment = ""
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty environment")
	}
}

func TestConfig_Validate_InvalidEnvironment(t *testing.T) {
	cfg := security.DefaultConfig()
	cfg.Environment = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid environment")
	}
}

func TestConfig_Validate_WildcardOriginProd(t *testing.T) {
	cfg := security.DefaultConfig()
	cfg.AllowedOrigins = []string{"*"}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for wildcard origin in production")
	}
}

func TestConfig_Validate_InsecureCSRFProd(t *testing.T) {
	cfg := security.DefaultConfig()
	cfg.CSRFCookieSecure = false
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for insecure CSRF cookie in production")
	}
}

func TestConfig_Validate_EmptyFrameAncestors(t *testing.T) {
	cfg := security.DefaultConfig()
	cfg.FrameAncestors = ""
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty FrameAncestors")
	}
}

func TestConfig_IsProduction(t *testing.T) {
	cfg := security.DefaultConfig()
	if !cfg.IsProduction() {
		t.Error("DefaultConfig should be production")
	}
	cfg.Environment = "development"
	if cfg.IsProduction() {
		t.Error("development should not be production")
	}
}

func TestConfig_IsDevelopment(t *testing.T) {
	cfg := security.DevelopmentConfig()
	if !cfg.IsDevelopment() {
		t.Error("DevelopmentConfig should be development")
	}
}

func TestConfigFromEnv(t *testing.T) {
	// Set a few env vars and verify they override defaults
	t.Setenv("SECURITY_ENVIRONMENT", "staging")
	t.Setenv("SECURITY_HSTS_MAX_AGE", "600")
	t.Setenv("SECURITY_LOGIN_PER_EMAIL", "10")
	t.Setenv("SECURITY_API_RATE_PER_MINUTE", "200")
	t.Setenv("SECURITY_CLAMAV_ADDR", "clamav:3310")
	t.Setenv("SECURITY_VIRUS_SCAN_ENABLED", "false")
	t.Setenv("SECURITY_TAMPER_PROOF", "true")

	cfg := security.ConfigFromEnv()

	if cfg.Environment != "staging" {
		t.Errorf("expected staging, got %s", cfg.Environment)
	}
	if cfg.HSTSMaxAge != 600 {
		t.Errorf("expected HSTSMaxAge=600, got %d", cfg.HSTSMaxAge)
	}
	if cfg.LoginPerEmail != 10 {
		t.Errorf("expected LoginPerEmail=10, got %d", cfg.LoginPerEmail)
	}
	if cfg.APIDefaultPerMinute != 200 {
		t.Errorf("expected APIDefaultPerMinute=200, got %d", cfg.APIDefaultPerMinute)
	}
	if cfg.ClamAVAddr != "clamav:3310" {
		t.Errorf("expected ClamAVAddr=clamav:3310, got %s", cfg.ClamAVAddr)
	}
	if cfg.VirusScanEnabled {
		t.Error("expected VirusScanEnabled=false")
	}
	if !cfg.EnableTamperProof {
		t.Error("expected EnableTamperProof=true")
	}
}

func TestConfigFromEnv_Defaults(t *testing.T) {
	// Clear any env vars that might be set
	for _, key := range []string{
		"SECURITY_ENVIRONMENT", "SECURITY_HSTS_MAX_AGE", "SECURITY_LOGIN_PER_EMAIL",
	} {
		os.Unsetenv(key)
	}

	cfg := security.ConfigFromEnv()
	def := security.DefaultConfig()

	if cfg.Environment != def.Environment {
		t.Errorf("expected default environment %s, got %s", def.Environment, cfg.Environment)
	}
	if cfg.HSTSMaxAge != def.HSTSMaxAge {
		t.Errorf("expected default HSTSMaxAge %d, got %d", def.HSTSMaxAge, cfg.HSTSMaxAge)
	}
}

func TestConfigFromEnv_InvalidValues(t *testing.T) {
	t.Setenv("SECURITY_HSTS_MAX_AGE", "not_a_number")
	t.Setenv("SECURITY_API_BURST_MULTIPLIER", "invalid")

	cfg := security.ConfigFromEnv()
	def := security.DefaultConfig()

	// Invalid values should fall back to defaults
	if cfg.HSTSMaxAge != def.HSTSMaxAge {
		t.Errorf("expected fallback to default HSTSMaxAge, got %d", cfg.HSTSMaxAge)
	}
	if cfg.APIBurstMultiplier != def.APIBurstMultiplier {
		t.Errorf("expected fallback to default burst multiplier, got %f", cfg.APIBurstMultiplier)
	}
}

func TestClamAVConfigFromConfig(t *testing.T) {
	cfg := security.DefaultConfig()
	cfg.ClamAVAddr = "clamav-host:3310"
	clamCfg := cfg.ClamAVConfigFromConfig()

	if clamCfg.Addr != "clamav-host:3310" {
		t.Errorf("expected clamav-host:3310, got %s", clamCfg.Addr)
	}
}

func TestBuildCSP_Production(t *testing.T) {
	cfg := security.DefaultConfig()
	csp := cfg.BuildCSP()

	if !containsDirective(csp, "default-src 'self'") {
		t.Errorf("expected default-src 'self' in CSP, got: %s", csp)
	}
	if !containsDirective(csp, "frame-ancestors 'none'") {
		t.Errorf("expected frame-ancestors 'none' in CSP, got: %s", csp)
	}
	if !containsDirective(csp, "upgrade-insecure-requests") {
		t.Errorf("expected upgrade-insecure-requests in production CSP, got: %s", csp)
	}
}

func TestBuildCSP_Development(t *testing.T) {
	cfg := security.DevelopmentConfig()
	csp := cfg.BuildCSP()

	if !containsDirective(csp, "'unsafe-eval'") {
		t.Errorf("expected 'unsafe-eval' in dev CSP, got: %s", csp)
	}
}

func TestBuildCSP_CustomDirective(t *testing.T) {
	cfg := security.DefaultConfig()
	cfg.CustomCSPDirectives = map[string]string{
		"media-src": "'self' https://cdn.example.com",
	}
	csp := cfg.BuildCSP()

	if !containsDirective(csp, "media-src 'self' https://cdn.example.com") {
		t.Errorf("expected custom media-src directive in CSP, got: %s", csp)
	}
}

func TestBuildCSP_InjectionPrevention(t *testing.T) {
	cfg := security.DefaultConfig()
	cfg.CustomCSPDirectives = map[string]string{
		"script-src; default-src *": "'self'", // Attempt to inject
	}
	csp := cfg.BuildCSP()

	// The malicious directive should be skipped
	if containsDirective(csp, "default-src *") {
		t.Errorf("CSP should not contain injected directive, got: %s", csp)
	}
}

func containsDirective(csp, directive string) bool {
	return len(csp) > 0 && contains(csp, directive)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexSubstring(s, substr) >= 0
}

func indexSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
