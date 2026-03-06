package proxy

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	gwconfig "github.com/clario360/platform/internal/gateway/config"
)

// serviceEntry holds a resolved backend service URL and its timeout.
type serviceEntry struct {
	URL     *url.URL
	Timeout time.Duration
}

// ServiceRegistry resolves service names to backend URLs.
// In dev it uses environment variables; in production it uses K8s DNS.
type ServiceRegistry struct {
	mu       sync.RWMutex
	services map[string]serviceEntry
}

// NewServiceRegistry creates a registry from service configs, with env var overrides.
// Env var format: SERVICE_URL_{UPPER_SNAKE_NAME} e.g. SERVICE_URL_IAM_SERVICE=http://iam:8081
func NewServiceRegistry(configs []gwconfig.ServiceConfig) (*ServiceRegistry, error) {
	reg := &ServiceRegistry{
		services: make(map[string]serviceEntry, len(configs)),
	}

	for _, cfg := range configs {
		rawURL := cfg.URL

		// Check for env var override
		envKey := "SERVICE_URL_" + strings.ToUpper(strings.ReplaceAll(cfg.Name, "-", "_"))
		if envVal := os.Getenv(envKey); envVal != "" {
			rawURL = envVal
		}

		parsed, err := url.Parse(rawURL)
		if err != nil {
			return nil, fmt.Errorf("invalid URL for service %s: %w", cfg.Name, err)
		}

		timeout := cfg.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}

		reg.services[cfg.Name] = serviceEntry{URL: parsed, Timeout: timeout}
	}

	return reg, nil
}

// Resolve returns the backend URL and timeout for a service name.
func (r *ServiceRegistry) Resolve(serviceName string) (*url.URL, time.Duration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.services[serviceName]
	if !ok {
		return nil, 0, false
	}
	// Return a copy to prevent mutation
	cp := *entry.URL
	return &cp, entry.Timeout, true
}

// Update changes the URL for a service at runtime (for service discovery integrations).
func (r *ServiceRegistry) Update(serviceName string, rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	r.mu.Lock()
	entry := r.services[serviceName]
	entry.URL = parsed
	r.services[serviceName] = entry
	r.mu.Unlock()
	return nil
}

// ServiceNames returns a list of all registered service names.
func (r *ServiceRegistry) ServiceNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.services))
	for name := range r.services {
		names = append(names, name)
	}
	return names
}
