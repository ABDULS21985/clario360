package proxy

import (
	"sort"
	"strings"

	"github.com/rs/zerolog"

	gwconfig "github.com/clario360/platform/internal/gateway/config"
)

// Router matches incoming request paths to backend service proxies.
type Router struct {
	routes   []routeEntry
	proxies  map[string]*ReverseProxy
	registry *ServiceRegistry
	logger   zerolog.Logger
}

type routeEntry struct {
	config gwconfig.RouteConfig
	proxy  *ReverseProxy
}

// RouteMatch holds the result of a route lookup.
type RouteMatch struct {
	Proxy   *ReverseProxy
	Config  gwconfig.RouteConfig
	Matched bool
}

// NewRouter creates a router from route configs, building reverse proxies for each backend service.
func NewRouter(routes []gwconfig.RouteConfig, registry *ServiceRegistry, cbCfg CircuitBreakerConfig, logger zerolog.Logger) (*Router, error) {
	r := &Router{
		proxies:  make(map[string]*ReverseProxy),
		registry: registry,
		logger:   logger,
	}

	// Sort routes by prefix length descending for longest-match-first.
	sorted := make([]gwconfig.RouteConfig, len(routes))
	copy(sorted, routes)
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].Prefix) > len(sorted[j].Prefix)
	})

	for _, rc := range sorted {
		// Get or create proxy for this service
		rp, ok := r.proxies[rc.Service]
		if !ok {
			target, timeout, found := registry.Resolve(rc.Service)
			if !found {
				logger.Warn().Str("service", rc.Service).Msg("service not found in registry, skipping route")
				continue
			}

			breaker := NewCircuitBreaker(cbCfg)
			rp = NewReverseProxy(rc.Service, target, timeout, breaker, logger)
			r.proxies[rc.Service] = rp
		}

		r.routes = append(r.routes, routeEntry{config: rc, proxy: rp})
	}

	return r, nil
}

// Match finds the best route for a given path (longest prefix match).
func (r *Router) Match(path string) RouteMatch {
	for _, entry := range r.routes {
		if strings.HasPrefix(path, entry.config.Prefix) {
			return RouteMatch{
				Proxy:   entry.proxy,
				Config:  entry.config,
				Matched: true,
			}
		}
	}
	return RouteMatch{}
}

// GetProxy returns the proxy for a specific service name.
func (r *Router) GetProxy(serviceName string) (*ReverseProxy, bool) {
	rp, ok := r.proxies[serviceName]
	return rp, ok
}

// Proxies returns all registered proxies.
func (r *Router) Proxies() map[string]*ReverseProxy {
	return r.proxies
}
