package config

import "time"

// EndpointGroup classifies routes for rate limiting purposes.
type EndpointGroup string

const (
	EndpointGroupAuth  EndpointGroup = "auth"
	EndpointGroupRead  EndpointGroup = "read"
	EndpointGroupWrite EndpointGroup = "write"
	EndpointGroupAdmin EndpointGroup = "admin"
)

// RouteConfig maps a URL prefix to a backend service.
type RouteConfig struct {
	Prefix        string        `yaml:"prefix"`
	Service       string        `yaml:"service"`
	StripPrefix   bool          `yaml:"strip_prefix"`
	Public        bool          `yaml:"public"`
	EndpointGroup EndpointGroup `yaml:"endpoint_group"`
}

// ServiceConfig holds backend service address configuration.
type ServiceConfig struct {
	Name    string        `yaml:"name"`
	URL     string        `yaml:"url"`
	Timeout time.Duration `yaml:"timeout"`
}

// DefaultRoutes returns the standard route configuration.
func DefaultRoutes() []RouteConfig {
	return []RouteConfig{
		{Prefix: "/api/v1/auth", Service: "iam-service", StripPrefix: false, Public: true, EndpointGroup: EndpointGroupAuth},
		{Prefix: "/api/v1/users", Service: "iam-service", StripPrefix: false, Public: false, EndpointGroup: EndpointGroupWrite},
		{Prefix: "/api/v1/roles", Service: "iam-service", StripPrefix: false, Public: false, EndpointGroup: EndpointGroupAdmin},
		{Prefix: "/api/v1/tenants", Service: "iam-service", StripPrefix: false, Public: false, EndpointGroup: EndpointGroupAdmin},
		{Prefix: "/api/v1/api-keys", Service: "iam-service", StripPrefix: false, Public: false, EndpointGroup: EndpointGroupAdmin},
		{Prefix: "/api/v1/audit", Service: "audit-service", StripPrefix: false, Public: false, EndpointGroup: EndpointGroupRead},
		{Prefix: "/api/v1/workflows", Service: "workflow-engine", StripPrefix: false, Public: false, EndpointGroup: EndpointGroupWrite},
		{Prefix: "/api/v1/cyber", Service: "cyber-service", StripPrefix: false, Public: false, EndpointGroup: EndpointGroupWrite},
		{Prefix: "/api/v1/data", Service: "data-service", StripPrefix: false, Public: false, EndpointGroup: EndpointGroupWrite},
		{Prefix: "/api/v1/acta", Service: "acta-service", StripPrefix: false, Public: false, EndpointGroup: EndpointGroupWrite},
		{Prefix: "/api/v1/lex", Service: "lex-service", StripPrefix: false, Public: false, EndpointGroup: EndpointGroupWrite},
		{Prefix: "/api/v1/visus", Service: "visus-service", StripPrefix: false, Public: false, EndpointGroup: EndpointGroupWrite},
	}
}

// DefaultServices returns the default backend service URLs (env-based for development).
func DefaultServices() []ServiceConfig {
	return []ServiceConfig{
		{Name: "iam-service", URL: "http://localhost:8081", Timeout: 30 * time.Second},
		{Name: "audit-service", URL: "http://localhost:8082", Timeout: 30 * time.Second},
		{Name: "workflow-engine", URL: "http://localhost:8083", Timeout: 60 * time.Second},
		{Name: "cyber-service", URL: "http://localhost:8084", Timeout: 30 * time.Second},
		{Name: "data-service", URL: "http://localhost:8085", Timeout: 60 * time.Second},
		{Name: "acta-service", URL: "http://localhost:8086", Timeout: 30 * time.Second},
		{Name: "lex-service", URL: "http://localhost:8087", Timeout: 30 * time.Second},
		{Name: "visus-service", URL: "http://localhost:8088", Timeout: 30 * time.Second},
	}
}
