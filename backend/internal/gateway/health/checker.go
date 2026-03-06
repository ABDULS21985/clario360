package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/gateway/proxy"
)

// ServiceHealth represents the health status of a single backend service.
type ServiceHealth struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "healthy", "unhealthy", "degraded"
	Latency string `json:"latency,omitempty"`
	Error   string `json:"error,omitempty"`
	Circuit string `json:"circuit_breaker,omitempty"`
}

// AggregatedHealth is the overall gateway health response.
type AggregatedHealth struct {
	Status   string          `json:"status"` // "healthy", "degraded", "unhealthy"
	Services []ServiceHealth `json:"services"`
}

// Checker pings backend services and reports aggregated health.
type Checker struct {
	registry *proxy.ServiceRegistry
	router   *proxy.Router
	logger   zerolog.Logger
	client   *http.Client
}

// NewChecker creates a health checker.
func NewChecker(registry *proxy.ServiceRegistry, router *proxy.Router, logger zerolog.Logger) *Checker {
	return &Checker{
		registry: registry,
		router:   router,
		logger:   logger,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Check pings all backend services and returns aggregated health.
func (c *Checker) Check(ctx context.Context) AggregatedHealth {
	serviceNames := c.registry.ServiceNames()

	results := make([]ServiceHealth, len(serviceNames))
	var wg sync.WaitGroup

	for i, name := range serviceNames {
		wg.Add(1)
		go func(idx int, svcName string) {
			defer wg.Done()
			results[idx] = c.checkService(ctx, svcName)
		}(i, name)
	}

	wg.Wait()

	// Determine overall status
	overallStatus := "healthy"
	unhealthyCount := 0
	for _, svc := range results {
		if svc.Status == "unhealthy" {
			unhealthyCount++
		}
	}

	if unhealthyCount == len(results) {
		overallStatus = "unhealthy"
	} else if unhealthyCount > 0 {
		overallStatus = "degraded"
	}

	return AggregatedHealth{
		Status:   overallStatus,
		Services: results,
	}
}

// checkService pings a single service's health endpoint.
func (c *Checker) checkService(ctx context.Context, serviceName string) ServiceHealth {
	sh := ServiceHealth{Name: serviceName}

	// Get circuit breaker state
	if rp, ok := c.router.GetProxy(serviceName); ok {
		sh.Circuit = rp.CircuitState().String()
	}

	target, ok := c.registry.Resolve(serviceName)
	if !ok {
		sh.Status = "unhealthy"
		sh.Error = "service not registered"
		return sh
	}

	healthURL := fmt.Sprintf("%s/healthz", target.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		sh.Status = "unhealthy"
		sh.Error = err.Error()
		return sh
	}

	start := time.Now()
	resp, err := c.client.Do(req)
	latency := time.Since(start)

	if err != nil {
		sh.Status = "unhealthy"
		sh.Error = err.Error()
		sh.Latency = latency.String()
		return sh
	}
	defer resp.Body.Close()

	sh.Latency = latency.String()

	if resp.StatusCode == http.StatusOK {
		sh.Status = "healthy"
	} else {
		sh.Status = "unhealthy"
		sh.Error = fmt.Sprintf("health check returned %d", resp.StatusCode)
	}

	return sh
}

// Handler returns an HTTP handler for the aggregated health endpoint.
func (c *Checker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result := c.Check(r.Context())

		w.Header().Set("Content-Type", "application/json")
		switch result.Status {
		case "healthy":
			w.WriteHeader(http.StatusOK)
		case "degraded":
			w.WriteHeader(http.StatusOK) // Still operational
		default:
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		_ = json.NewEncoder(w).Encode(result)
	}
}
