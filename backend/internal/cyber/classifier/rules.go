package classifier

import (
	"encoding/json"
	"strings"

	"github.com/clario360/platform/internal/cyber/model"
)

// DefaultRules returns the built-in classification rules sorted by priority.
// Lower priority number = evaluated first; first match wins.
func DefaultRules() []ClassificationRule {
	return []ClassificationRule{
		{
			Name:     "database-assets-are-critical",
			Priority: 1,
			Condition: func(a *model.Asset) bool {
				return a.Type == model.AssetTypeDatabase ||
					containsAny(a.Tags, "database", "db", "sql", "postgres", "mysql", "oracle", "mssql")
			},
			Result: model.CriticalityCritical,
			Reason: "Database assets hold sensitive data and are high-value targets",
		},
		{
			Name:     "domain-controllers-are-critical",
			Priority: 2,
			Condition: func(a *model.Asset) bool {
				return containsAny(a.Tags, "domain-controller", "active-directory", "ldap", "ad") ||
					hostnameMatches(a.Hostname, "*dc*", "*ad-*", "*-dc-*", "*-ad-*")
			},
			Result: model.CriticalityCritical,
			Reason: "Domain controllers manage authentication for the entire environment",
		},
		{
			Name:     "production-assets-are-critical",
			Priority: 3,
			Condition: func(a *model.Asset) bool {
				return containsAny(a.Tags, "production", "prod")
			},
			Result: model.CriticalityCritical,
			Reason: "Production assets directly impact business operations",
		},
		{
			Name:     "internet-facing-are-high",
			Priority: 4,
			Condition: func(a *model.Asset) bool {
				return containsAny(a.Tags, "internet-facing", "dmz", "public", "external") ||
					(metadataHasAnyPort(a.Metadata, 80, 443, 8443) && !containsAny(a.Tags, "internal"))
			},
			Result: model.CriticalityHigh,
			Reason: "Internet-facing assets have a larger attack surface",
		},
		{
			Name:     "iot-devices-are-high",
			Priority: 5,
			Condition: func(a *model.Asset) bool {
				return a.Type == model.AssetTypeIoTDevice
			},
			Result: model.CriticalityHigh,
			Reason: "IoT devices often lack robust security controls",
		},
		{
			Name:     "cloud-resources-with-public-ip-are-high",
			Priority: 6,
			Condition: func(a *model.Asset) bool {
				return a.Type == model.AssetTypeCloudResource && metadataHasKey(a.Metadata, "public_ip")
			},
			Result: model.CriticalityHigh,
			Reason: "Cloud resources with public IPs are accessible from the internet",
		},
		{
			Name:     "assets-with-critical-cves-are-high",
			Priority: 7,
			Condition: func(a *model.Asset) bool {
				return metadataVulnCountCritical(a.Metadata) > 0
			},
			Result: model.CriticalityHigh,
			Reason: "Asset has one or more critical CVEs",
		},
		{
			Name:     "servers-default-medium",
			Priority: 10,
			Condition: func(a *model.Asset) bool {
				return a.Type == model.AssetTypeServer
			},
			Result: model.CriticalityMedium,
			Reason: "Servers default to medium criticality",
		},
		{
			Name:     "network-devices-default-medium",
			Priority: 11,
			Condition: func(a *model.Asset) bool {
				return a.Type == model.AssetTypeNetworkDevice
			},
			Result: model.CriticalityMedium,
			Reason: "Network devices default to medium criticality",
		},
		{
			Name:     "applications-default-medium",
			Priority: 12,
			Condition: func(a *model.Asset) bool {
				return a.Type == model.AssetTypeApplication
			},
			Result: model.CriticalityMedium,
			Reason: "Application assets default to medium criticality",
		},
		{
			Name:     "endpoints-default-low",
			Priority: 15,
			Condition: func(a *model.Asset) bool {
				return a.Type == model.AssetTypeEndpoint
			},
			Result: model.CriticalityLow,
			Reason: "Endpoints default to low criticality",
		},
		{
			Name:     "containers-default-low",
			Priority: 20,
			Condition: func(a *model.Asset) bool {
				return a.Type == model.AssetTypeContainer
			},
			Result: model.CriticalityLow,
			Reason: "Containers default to low criticality",
		},
		{
			Name:     "catch-all-low",
			Priority: 100,
			Condition: func(a *model.Asset) bool {
				return true
			},
			Result: model.CriticalityLow,
			Reason: "Default classification",
		},
	}
}

// containsAny returns true if any of the given values appears in tags (case-insensitive).
func containsAny(tags []string, values ...string) bool {
	for _, tag := range tags {
		tagLower := strings.ToLower(tag)
		for _, v := range values {
			if tagLower == strings.ToLower(v) {
				return true
			}
		}
	}
	return false
}

// hostnameMatches performs case-insensitive glob-like matching (* wildcard).
func hostnameMatches(hostname *string, patterns ...string) bool {
	if hostname == nil {
		return false
	}
	h := strings.ToLower(*hostname)
	for _, pattern := range patterns {
		p := strings.ToLower(pattern)
		if globMatch(p, h) {
			return true
		}
	}
	return false
}

// globMatch implements simple '*' wildcard matching (no '?' support needed here).
func globMatch(pattern, s string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == s
	}
	if !strings.HasPrefix(s, parts[0]) {
		return false
	}
	s = s[len(parts[0]):]
	for i := 1; i < len(parts)-1; i++ {
		idx := strings.Index(s, parts[i])
		if idx < 0 {
			return false
		}
		s = s[idx+len(parts[i]):]
	}
	return strings.HasSuffix(s, parts[len(parts)-1])
}

// metadataHasAnyPort checks if metadata.open_ports contains any of the given ports.
func metadataHasAnyPort(metadata json.RawMessage, ports ...int) bool {
	if len(metadata) == 0 {
		return false
	}
	var m map[string]any
	if err := json.Unmarshal(metadata, &m); err != nil {
		return false
	}
	openPorts, ok := m["open_ports"]
	if !ok {
		return false
	}
	portSlice, ok := openPorts.([]any)
	if !ok {
		return false
	}
	for _, p := range portSlice {
		pNum, ok := p.(float64) // JSON numbers unmarshal as float64
		if !ok {
			continue
		}
		for _, target := range ports {
			if int(pNum) == target {
				return true
			}
		}
	}
	return false
}

// metadataHasKey checks whether a specific top-level key exists in metadata.
func metadataHasKey(metadata json.RawMessage, key string) bool {
	if len(metadata) == 0 {
		return false
	}
	var m map[string]any
	if err := json.Unmarshal(metadata, &m); err != nil {
		return false
	}
	_, ok := m[key]
	return ok
}

// metadataVulnCountCritical reads metadata.vuln_counts.critical.
func metadataVulnCountCritical(metadata json.RawMessage) int {
	if len(metadata) == 0 {
		return 0
	}
	var m map[string]any
	if err := json.Unmarshal(metadata, &m); err != nil {
		return 0
	}
	vulnCounts, ok := m["vuln_counts"].(map[string]any)
	if !ok {
		return 0
	}
	critical, _ := vulnCounts["critical"].(float64)
	return int(critical)
}
