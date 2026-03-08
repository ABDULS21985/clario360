package config

import (
	"os"
	"time"
)

// EndpointGroup classifies routes for rate limiting purposes.
type EndpointGroup string

const (
	EndpointGroupAuth   EndpointGroup = "auth"
	EndpointGroupRead   EndpointGroup = "read"
	EndpointGroupWrite  EndpointGroup = "write"
	EndpointGroupAdmin  EndpointGroup = "admin"
	EndpointGroupUpload EndpointGroup = "upload"
	EndpointGroupWS     EndpointGroup = "ws"
)

// RouteConfig maps a URL prefix to a backend service.
type RouteConfig struct {
	Prefix        string        `yaml:"prefix"`
	Service       string        `yaml:"service"`
	StripPrefix   bool          `yaml:"strip_prefix"`
	Public        bool          `yaml:"public"`
	EndpointGroup EndpointGroup `yaml:"endpoint_group"`
	MaxBodyMB     int           `yaml:"max_body_mb"` // 0 = use global default
	TimeoutSec    int           `yaml:"timeout_sec"` // 0 = use global default
}

// ServiceConfig holds backend service address configuration.
type ServiceConfig struct {
	Name    string        `yaml:"name"`
	URL     string        `yaml:"url"`
	Timeout time.Duration `yaml:"timeout"`
}

// DefaultRoutes returns the standard route configuration. Routes with longer prefixes
// should appear before shorter ones — the router sorts by prefix length descending at startup.
func DefaultRoutes() []RouteConfig {
	return []RouteConfig{
		// IAM - OIDC discovery
		{Prefix: "/.well-known", Service: "iam-service", Public: true, EndpointGroup: EndpointGroupAuth},

		// IAM — Auth (public)
		{Prefix: "/api/v1/auth", Service: "iam-service", Public: true, EndpointGroup: EndpointGroupAuth},

		// IAM — User/Role/Tenant management
		{Prefix: "/api/v1/users", Service: "iam-service", Public: false, EndpointGroup: EndpointGroupWrite},
		{Prefix: "/api/v1/roles", Service: "iam-service", Public: false, EndpointGroup: EndpointGroupAdmin},
		{Prefix: "/api/v1/tenants", Service: "iam-service", Public: false, EndpointGroup: EndpointGroupAdmin},
		{Prefix: "/api/v1/api-keys", Service: "iam-service", Public: false, EndpointGroup: EndpointGroupWrite},
		{Prefix: "/api/v1/notebooks", Service: "iam-service", Public: false, EndpointGroup: EndpointGroupWrite},

		// Audit
		{Prefix: "/api/v1/audit", Service: "audit-service", Public: false, EndpointGroup: EndpointGroupRead},

		// Workflow
		{Prefix: "/api/v1/workflows", Service: "workflow-engine", Public: false, EndpointGroup: EndpointGroupWrite},

		// Notifications (REST)
		{Prefix: "/api/v1/notifications", Service: "notification-service", Public: false, EndpointGroup: EndpointGroupWrite},

		// Files — upload route MUST come before the generic files route (longer prefix wins).
		{Prefix: "/api/v1/files/upload", Service: "file-service", Public: false, EndpointGroup: EndpointGroupUpload, MaxBodyMB: 100, TimeoutSec: 120},
		{Prefix: "/api/v1/files", Service: "file-service", Public: false, EndpointGroup: EndpointGroupRead},

		// Cybersecurity Suite
		{Prefix: "/api/v1/cyber", Service: "cyber-service", Public: false, EndpointGroup: EndpointGroupWrite},

		// Data Suite
		{Prefix: "/api/v1/data", Service: "data-service", Public: false, EndpointGroup: EndpointGroupWrite},

		// Governance Suite (Acta)
		{Prefix: "/api/v1/acta", Service: "acta-service", Public: false, EndpointGroup: EndpointGroupWrite},

		// Legal Suite (Lex)
		{Prefix: "/api/v1/lex", Service: "lex-service", Public: false, EndpointGroup: EndpointGroupWrite},

		// Executive Intelligence (Visus360)
		{Prefix: "/api/v1/visus", Service: "visus-service", Public: false, EndpointGroup: EndpointGroupRead},
	}
}

// DefaultWSRoutes returns authenticated WebSocket proxy routes.
func DefaultWSRoutes() []RouteConfig {
	return []RouteConfig{
		{Prefix: "/ws/v1/notifications", Service: "notification-service", Public: false, EndpointGroup: EndpointGroupWS},
		{Prefix: "/ws/v1/cyber", Service: "cyber-service", Public: false, EndpointGroup: EndpointGroupWS},
		{Prefix: "/ws/v1/visus", Service: "visus-service", Public: false, EndpointGroup: EndpointGroupWS},
	}
}

// DefaultServices returns the backend service configs, preferring GW_SVC_URL_* env vars.
func DefaultServices() []ServiceConfig {
	return []ServiceConfig{
		{Name: "iam-service", URL: envOrDefault("GW_SVC_URL_IAM", "http://localhost:8081"), Timeout: 30 * time.Second},
		{Name: "audit-service", URL: envOrDefault("GW_SVC_URL_AUDIT", "http://localhost:8082"), Timeout: 30 * time.Second},
		{Name: "workflow-engine", URL: envOrDefault("GW_SVC_URL_WORKFLOW", "http://localhost:8083"), Timeout: 60 * time.Second},
		{Name: "notification-service", URL: envOrDefault("GW_SVC_URL_NOTIFICATION", "http://localhost:8089"), Timeout: 30 * time.Second},
		{Name: "file-service", URL: envOrDefault("GW_SVC_URL_FILE", "http://localhost:8091"), Timeout: 120 * time.Second},
		{Name: "cyber-service", URL: envOrDefault("GW_SVC_URL_CYBER", "http://localhost:8084"), Timeout: 30 * time.Second},
		{Name: "data-service", URL: envOrDefault("GW_SVC_URL_DATA", "http://localhost:8085"), Timeout: 60 * time.Second},
		{Name: "acta-service", URL: envOrDefault("GW_SVC_URL_ACTA", "http://localhost:8086"), Timeout: 30 * time.Second},
		{Name: "lex-service", URL: envOrDefault("GW_SVC_URL_LEX", "http://localhost:8087"), Timeout: 30 * time.Second},
		{Name: "visus-service", URL: envOrDefault("GW_SVC_URL_VISUS", "http://localhost:8088"), Timeout: 30 * time.Second},
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
