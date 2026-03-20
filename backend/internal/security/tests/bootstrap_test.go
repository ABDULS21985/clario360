package security_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	security "github.com/clario360/platform/internal/security"
)

func TestBootstrap_Success(t *testing.T) {
	cfg := security.DevelopmentConfig()
	cfg.VirusScanEnabled = false // No ClamAV in tests
	reg := prometheus.NewRegistry()
	logger := zerolog.Nop()

	stack, err := security.Bootstrap(cfg, nil, reg, logger)
	if err != nil {
		t.Fatalf("Bootstrap() failed: %v", err)
	}

	if stack.Config == nil {
		t.Error("expected Config to be set")
	}
	if stack.Metrics == nil {
		t.Error("expected Metrics to be set")
	}
	if stack.Logger == nil {
		t.Error("expected Logger to be set")
	}
	if stack.Sanitizer == nil {
		t.Error("expected Sanitizer to be set")
	}
	if stack.SessionMgr == nil {
		t.Error("expected SessionMgr to be set")
	}
	if stack.AuthLimiter == nil {
		t.Error("expected AuthLimiter to be set")
	}
	if stack.APILimiter == nil {
		t.Error("expected APILimiter to be set")
	}
	if stack.SSRFValidator == nil {
		t.Error("expected SSRFValidator to be set")
	}
	if stack.ClamAV != nil {
		t.Error("expected ClamAV to be nil when virus scan disabled")
	}
}

func TestBootstrap_WithClamAV(t *testing.T) {
	cfg := security.DevelopmentConfig()
	cfg.VirusScanEnabled = true
	cfg.ClamAVAddr = "127.0.0.1:1" // Won't connect, but should initialize
	reg := prometheus.NewRegistry()
	logger := zerolog.Nop()

	stack, err := security.Bootstrap(cfg, nil, reg, logger)
	if err != nil {
		t.Fatalf("Bootstrap() failed: %v", err)
	}

	if stack.ClamAV == nil {
		t.Error("expected ClamAV to be set when virus scan enabled")
	}
}

func TestBootstrap_InvalidConfig(t *testing.T) {
	cfg := &security.Config{} // Empty — will fail validation
	reg := prometheus.NewRegistry()
	logger := zerolog.Nop()

	_, err := security.Bootstrap(cfg, nil, reg, logger)
	if err == nil {
		t.Error("expected error for invalid config")
	}
}

func TestBootstrap_Middleware(t *testing.T) {
	cfg := security.DevelopmentConfig()
	cfg.VirusScanEnabled = false
	reg := prometheus.NewRegistry()
	logger := zerolog.Nop()

	stack, err := security.Bootstrap(cfg, nil, reg, logger)
	if err != nil {
		t.Fatalf("Bootstrap() failed: %v", err)
	}

	middlewares := stack.Middleware(logger)
	if len(middlewares) != 6 {
		t.Errorf("expected 6 middleware functions, got %d", len(middlewares))
	}
}

func TestBootstrap_FileUploadValidator(t *testing.T) {
	cfg := security.DevelopmentConfig()
	cfg.VirusScanEnabled = false
	reg := prometheus.NewRegistry()
	logger := zerolog.Nop()

	stack, err := security.Bootstrap(cfg, nil, reg, logger)
	if err != nil {
		t.Fatalf("Bootstrap() failed: %v", err)
	}

	fuv := stack.FileUploadValidator(logger)
	if fuv == nil {
		t.Error("expected FileUploadValidator to be created")
	}
}
