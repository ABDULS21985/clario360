package config

import "testing"

func TestDefaultRoutes_ContainsAllServices(t *testing.T) {
	routes := DefaultRoutes()

	expectedPrefixes := map[string]bool{
		"/.well-known":     false,
		"/api/v1/auth":      false,
		"/api/v1/users":     false,
		"/api/v1/roles":     false,
		"/api/v1/tenants":   false,
		"/api/v1/api-keys":  false,
		"/api/v1/audit":     false,
		"/api/v1/workflows": false,
		"/api/v1/cyber":     false,
		"/api/v1/data":      false,
		"/api/v1/acta":      false,
		"/api/v1/lex":       false,
		"/api/v1/visus":     false,
	}

	for _, route := range routes {
		if _, ok := expectedPrefixes[route.Prefix]; ok {
			expectedPrefixes[route.Prefix] = true
		}
	}

	for prefix, found := range expectedPrefixes {
		if !found {
			t.Errorf("expected route prefix %s not found", prefix)
		}
	}
}

func TestDefaultRoutes_AuthIsPublic(t *testing.T) {
	routes := DefaultRoutes()
	for _, route := range routes {
		if route.Prefix == "/api/v1/auth" {
			if !route.Public {
				t.Error("expected auth route to be public")
			}
			if route.EndpointGroup != EndpointGroupAuth {
				t.Errorf("expected auth endpoint group, got %s", route.EndpointGroup)
			}
			return
		}
	}
	t.Error("auth route not found")
}

func TestDefaultRoutes_ProtectedRoutesNotPublic(t *testing.T) {
	routes := DefaultRoutes()
	for _, route := range routes {
		if route.Prefix != "/api/v1/auth" && route.Prefix != "/.well-known" && route.Public {
			t.Errorf("expected route %s to not be public", route.Prefix)
		}
	}
}

func TestDefaultServices_ContainsAllBackends(t *testing.T) {
	services := DefaultServices()

	expectedServices := map[string]bool{
		"iam-service":     false,
		"audit-service":   false,
		"workflow-engine": false,
		"cyber-service":   false,
		"data-service":    false,
		"acta-service":    false,
		"lex-service":     false,
		"visus-service":   false,
	}

	for _, svc := range services {
		if _, ok := expectedServices[svc.Name]; ok {
			expectedServices[svc.Name] = true
		}
		if svc.URL == "" {
			t.Errorf("service %s has empty URL", svc.Name)
		}
		if svc.Timeout == 0 {
			t.Errorf("service %s has zero timeout", svc.Name)
		}
	}

	for name, found := range expectedServices {
		if !found {
			t.Errorf("expected service %s not found", name)
		}
	}
}
