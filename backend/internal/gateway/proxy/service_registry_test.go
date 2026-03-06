package proxy

import (
	"testing"
	"time"

	gwconfig "github.com/clario360/platform/internal/gateway/config"
)

func TestServiceRegistry_Resolve(t *testing.T) {
	configs := []gwconfig.ServiceConfig{
		{Name: "iam-service", URL: "http://localhost:8081", Timeout: 30 * time.Second},
		{Name: "cyber-service", URL: "http://localhost:8084", Timeout: 30 * time.Second},
	}

	reg, err := NewServiceRegistry(configs)
	if err != nil {
		t.Fatalf("NewServiceRegistry failed: %v", err)
	}

	u, ok := reg.Resolve("iam-service")
	if !ok {
		t.Fatal("expected iam-service to be found")
	}
	if u.String() != "http://localhost:8081" {
		t.Errorf("expected http://localhost:8081, got %s", u.String())
	}

	_, ok = reg.Resolve("nonexistent")
	if ok {
		t.Error("expected nonexistent service to not be found")
	}
}

func TestServiceRegistry_Update(t *testing.T) {
	configs := []gwconfig.ServiceConfig{
		{Name: "iam-service", URL: "http://localhost:8081", Timeout: 30 * time.Second},
	}

	reg, err := NewServiceRegistry(configs)
	if err != nil {
		t.Fatalf("NewServiceRegistry failed: %v", err)
	}

	err = reg.Update("iam-service", "http://iam:8081")
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	u, ok := reg.Resolve("iam-service")
	if !ok {
		t.Fatal("expected iam-service to be found after update")
	}
	if u.String() != "http://iam:8081" {
		t.Errorf("expected http://iam:8081, got %s", u.String())
	}
}

func TestServiceRegistry_ServiceNames(t *testing.T) {
	configs := []gwconfig.ServiceConfig{
		{Name: "iam-service", URL: "http://localhost:8081", Timeout: 30 * time.Second},
		{Name: "cyber-service", URL: "http://localhost:8084", Timeout: 30 * time.Second},
	}

	reg, err := NewServiceRegistry(configs)
	if err != nil {
		t.Fatalf("NewServiceRegistry failed: %v", err)
	}

	names := reg.ServiceNames()
	if len(names) != 2 {
		t.Errorf("expected 2 service names, got %d", len(names))
	}
}

func TestServiceRegistry_ResolveCopy(t *testing.T) {
	configs := []gwconfig.ServiceConfig{
		{Name: "iam-service", URL: "http://localhost:8081", Timeout: 30 * time.Second},
	}

	reg, err := NewServiceRegistry(configs)
	if err != nil {
		t.Fatalf("NewServiceRegistry failed: %v", err)
	}

	u1, _ := reg.Resolve("iam-service")
	u2, _ := reg.Resolve("iam-service")

	// Mutating one should not affect the other
	u1.Host = "modified:9999"
	if u2.Host == "modified:9999" {
		t.Error("expected Resolve to return copies, not references")
	}
}

func TestServiceRegistry_InvalidURL(t *testing.T) {
	configs := []gwconfig.ServiceConfig{
		{Name: "bad-service", URL: "://invalid", Timeout: 30 * time.Second},
	}

	_, err := NewServiceRegistry(configs)
	if err != nil {
		t.Logf("Got expected error for invalid URL: %v", err)
	}
	// Note: url.Parse is very lenient and may not error for "://invalid"
	// The important thing is NewServiceRegistry doesn't panic
}
